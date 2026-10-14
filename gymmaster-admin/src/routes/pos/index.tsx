import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { KeyboardEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  App,
  Badge,
  Button,
  Card,
  Col,
  Empty,
  Input,
  InputNumber,
  Result,
  Row,
  Segmented,
  Skeleton,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { InputRef } from 'antd';
import {
  BarcodeOutlined,
  CheckCircleOutlined,
  CloseOutlined,
  DeleteOutlined,
  MinusOutlined,
  PlusOutlined,
  ProfileOutlined,
  SearchOutlined,
  ShoppingOutlined,
  TagOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import invoiceApi from '@/api/invoice';
import inventoryApi from '@/api/inventory';
import posApi from '@/api/pos';
import type { PosOrderInput, PosOrderResult, PosProduct } from '@/api/pos';
import productApi, { productCategoryApi } from '@/api/product';
import promotionApi from '@/api/promotion';
import BranchSelect from '@/components/BranchSelect';
import ConfirmButton from '@/components/ConfirmButton';
import MemberSelect from '@/components/MemberSelect';
import PageTitle from '@/components/PageTitle';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { playError, playSuccess } from '@/routes/checkin/sound';
import { isApiError } from '@/utils/api';
import { getErrorMessage, showError } from '@/utils/errors';
import { formatMoney, formatNumber, normalizeText } from '@/utils/format';
import { paymentTotals, unitPrice, useCart } from './cart';
import PaymentPanel from './PaymentPanel';
import SuccessModal from './SuccessModal';
import './pos.css';

const ALL = 'all';

/** Ảnh sản phẩm hoặc ô giữ chỗ. */
function Thumb({ src, size }: { src?: string; size: number }) {
  return src ? (
    <img src={src} alt="" className="pos-thumb" style={{ width: size, height: size }} />
  ) : (
    <div className="pos-thumb pos-thumb--empty" style={{ width: size, height: size }}>
      <ShoppingOutlined style={{ fontSize: size * 0.4 }} />
    </div>
  );
}

/** Giá hiển thị: giá áp dụng + giá gốc gạch ngang khi đang dùng giá hội viên. */
function PriceText({ price, original }: { price: number; original: number }) {
  return (
    <span className="pos-price">
      <b>{formatMoney(price)}</b>
      {original !== price && <del>{formatMoney(original)}</del>}
    </span>
  );
}

function PosScreen({ branchId, branchName }: { branchId: number; branchName: string }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message, modal } = App.useApp();
  const { can } = usePermission();
  const { cart, patch, add, setQty, remove, sync, reset } = useCart(branchId);

  const [search, setSearch] = useState('');
  const [category, setCategory] = useState<string>(ALL);
  const [code, setCode] = useState('');
  const [promoInput, setPromoInput] = useState(cart.promoCode);
  const [shortage, setShortage] = useState<string | null>(null);
  const [result, setResult] = useState<PosOrderResult | null>(null);
  const scanRef = useRef<InputRef>(null);

  /* ------------------------------------------------------------ dữ liệu */

  const productsQuery = useQuery({
    queryKey: [posApi.key, 'products', branchId],
    queryFn: () => posApi.products({ branch_id: branchId }),
    staleTime: 30_000,
  });
  const products = useMemo(() => productsQuery.data ?? [], [productsQuery.data]);

  const categoriesQuery = useQuery({
    queryKey: [productCategoryApi.key, 'list'],
    queryFn: () => productCategoryApi.list(),
    staleTime: 5 * 60_000,
  });

  // Tồn/giá trong giỏ luôn theo danh sách mới nhất.
  useEffect(() => {
    if (productsQuery.data) sync(productsQuery.data);
  }, [productsQuery.data, sync]);

  const filtered = useMemo(() => {
    const kw = normalizeText(search.trim());
    return products.filter((p) => {
      if (category !== ALL && String(p.category_id ?? '') !== category) return false;
      if (!kw) return true;
      return (
        normalizeText(p.name).includes(kw) ||
        normalizeText(p.sku).includes(kw) ||
        (p.barcode ?? '').includes(search.trim())
      );
    });
  }, [products, search, category]);

  /* ------------------------------------------------------------ tiền */

  const member = cart.member;
  const subtotal = cart.lines.reduce((s, l) => s + unitPrice(l.product, member) * l.quantity, 0);
  const memberSaving = cart.lines.reduce(
    (s, l) => s + (l.product.sell_price - unitPrice(l.product, member)) * l.quantity,
    0,
  );

  const promoQuery = useQuery({
    queryKey: [promotionApi.key, 'check', 'PRODUCT', cart.promoCode, subtotal],
    queryFn: () =>
      promotionApi.check({ code: cart.promoCode, applies_to: 'PRODUCT', amount: subtotal }),
    enabled: !!cart.promoCode && subtotal > 0,
    retry: false,
    staleTime: 60_000,
  });
  const discount = promoQuery.isSuccess ? promoQuery.data.discount : 0;
  const promoError = promoQuery.isError ? getErrorMessage(promoQuery.error) : null;
  const total = Math.max(0, subtotal - discount);
  const pay = paymentTotals(cart.payments, total);

  /* ------------------------------------------------------------ thao tác giỏ */

  const addProduct = useCallback(
    (p: PosProduct) => {
      if (p.stock <= 0) {
        message.warning(`"${p.name}" đã hết hàng tại chi nhánh`);
        return false;
      }
      const ok = add(p);
      if (!ok) message.warning(`"${p.name}" chỉ còn ${formatNumber(p.stock)} ${p.unit}`);
      setShortage(null);
      return ok;
    },
    [add, message],
  );

  const barcode = useMutation({
    mutationFn: (value: string) => posApi.byBarcode(value, branchId),
    onSuccess: (p) => {
      if (addProduct(p)) playSuccess();
      else playError();
    },
    onError: (e) => {
      playError();
      showError(e, 'Không tìm thấy sản phẩm');
    },
  });

  const onScanKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter') return;
    e.preventDefault();
    const value = code.trim();
    setCode('');
    if (value) barcode.mutate(value);
  };

  const focusScan = useCallback(() => scanRef.current?.focus({ preventScroll: true }), []);

  // Ô quét luôn giữ focus (máy quét cầm tay gõ như bàn phím) trừ khi đang nhập ô khác / mở hộp thoại.
  useEffect(() => {
    focusScan();
    const t = setInterval(() => {
      const el = document.activeElement;
      const busy =
        el &&
        el !== document.body &&
        (el.matches('input, textarea, select, button, [contenteditable="true"]') ||
          !!el.closest('.ant-select-dropdown, .ant-modal, .ant-dropdown, .ant-popover'));
      if (!busy && !result) focusScan();
    }, 800);
    return () => clearInterval(t);
  }, [focusScan, result]);

  const applyPromo = () => {
    const v = promoInput.trim().toUpperCase();
    setPromoInput(v);
    patch({ promoCode: v });
  };

  /* ------------------------------------------------------------ thanh toán */

  const order = useMutation({
    mutationFn: (data: PosOrderInput) => posApi.createOrder(data),
    onSuccess: (res) => {
      playSuccess();
      setResult(res);
      reset();
      setPromoInput('');
      setShortage(null);
      void queryClient.invalidateQueries({ queryKey: [posApi.key] });
      void queryClient.invalidateQueries({ queryKey: [invoiceApi.key] });
      void queryClient.invalidateQueries({ queryKey: [inventoryApi.key] });
      void queryClient.invalidateQueries({ queryKey: [productApi.key] });
    },
    onError: (e) => {
      playError();
      if (isApiError(e) && e.code === 40900) {
        const name = /Sản phẩm "(.+?)"/.exec(e.message)?.[1] ?? null;
        setShortage(name);
        void productsQuery.refetch();
        modal.error({ title: 'Không đủ hàng', content: e.message, okText: 'Đã hiểu' });
        return;
      }
      showError(e, 'Không thể thanh toán');
    },
  });

  const blockReason =
    cart.lines.length === 0
      ? 'Giỏ hàng trống'
      : promoQuery.isFetching
        ? 'Đang kiểm tra mã khuyến mãi'
        : promoError
          ? 'Mã khuyến mãi không hợp lệ – bỏ mã hoặc sửa lại'
          : pay.error
            ? pay.error
            : pay.missing > 0
              ? `Còn thiếu ${formatMoney(pay.missing)}`
              : null;

  const checkout = useCallback(() => {
    if (blockReason || order.isPending) {
      if (blockReason) message.warning(blockReason);
      return;
    }
    order.mutate({
      branch_id: branchId,
      member_id: cart.member?.id ?? null,
      items: cart.lines.map((l) => ({ product_id: l.product.id, quantity: l.quantity })),
      promotion_code: cart.promoCode || '',
      payments:
        total === 0
          ? []
          : cart.payments
              .filter((p) => (p.amount ?? 0) > 0)
              .map((p) => ({
                amount: p.amount!,
                method: p.method,
                reference: p.method === 'CASH' ? '' : p.reference.trim(),
              })),
      note: cart.note.trim(),
    });
  }, [blockReason, order, message, branchId, cart, total]);

  // Phím tắt F9 = Thanh toán.
  useEffect(() => {
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'F9') {
        e.preventDefault();
        if (!result) checkout();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [checkout, result]);

  const newOrder = useCallback(() => {
    setResult(null);
    setTimeout(focusScan, 50);
  }, [focusScan]);

  const qtyInCart = (id: number) => cart.lines.find((l) => l.product.id === id)?.quantity ?? 0;

  /* ------------------------------------------------------------ giao diện */

  const categoryOptions = [
    { label: 'Tất cả', value: ALL },
    ...(categoriesQuery.data ?? []).map((c) => ({ label: c.name, value: String(c.id) })),
  ];

  return (
    <div className="pos-page">
      <Row gutter={[12, 12]} wrap>
        {/* ----------------------------------------------- trái: sản phẩm */}
        <Col xs={24} md={14} xl={15}>
          <Card variant="borderless" className="page-card pos-left">
            <Row justify="space-between" align="middle" gutter={[8, 8]} className="page-head">
              <Col>
                <PageTitle title="Bán hàng (POS)" subtitle={branchName} />
              </Col>
              <Col>
                {can([PERM.POS_SELL, PERM.INVOICE_VIEW]) && (
                  <Button icon={<ProfileOutlined />} onClick={() => navigate('/app/pos-orders')}>
                    Đơn POS
                  </Button>
                )}
              </Col>
            </Row>

            <Row gutter={[8, 8]} style={{ marginBottom: 10 }}>
              <Col xs={24} lg={12}>
                <Input
                  ref={scanRef}
                  size="large"
                  className="pos-scan"
                  prefix={<BarcodeOutlined />}
                  placeholder="Quét mã vạch / nhập SKU rồi Enter"
                  value={code}
                  autoComplete="off"
                  spellCheck={false}
                  onChange={(e) => setCode(e.target.value)}
                  onKeyDown={onScanKey}
                  suffix={barcode.isPending ? <Typography.Text type="secondary">…</Typography.Text> : null}
                />
              </Col>
              <Col xs={24} lg={12}>
                <Input
                  size="large"
                  allowClear
                  prefix={<SearchOutlined />}
                  placeholder="Tìm theo tên, SKU"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
              </Col>
            </Row>

            <div className="pos-cats">
              <Segmented
                options={categoryOptions}
                value={category}
                onChange={(v) => setCategory(String(v))}
              />
            </div>

            {productsQuery.isLoading ? (
              <Skeleton active paragraph={{ rows: 8 }} />
            ) : productsQuery.isError ? (
              <Result
                status="warning"
                title={getErrorMessage(productsQuery.error, 'Không tải được sản phẩm')}
                extra={<Button onClick={() => productsQuery.refetch()}>Thử lại</Button>}
              />
            ) : filtered.length === 0 ? (
              <Empty
                description={products.length === 0 ? 'Chưa có sản phẩm đang bán' : 'Không có sản phẩm phù hợp'}
              />
            ) : (
              <div className="pos-grid">
                {filtered.map((p) => {
                  const price = unitPrice(p, member);
                  const out = p.stock <= 0;
                  const full = !out && qtyInCart(p.id) >= p.stock;
                  const card = (
                    <button
                      type="button"
                      key={p.id}
                      className={`pos-card${out || full ? ' pos-card--disabled' : ''}`}
                      disabled={out || full}
                      onClick={() => addProduct(p)}
                      title={p.name}
                    >
                      <Thumb src={p.image_url} size={64} />
                      <div className="pos-card__name">{p.name}</div>
                      <PriceText price={price} original={p.sell_price} />
                      <div className={`pos-card__stock${p.low_stock ? ' is-low' : ''}`}>
                        {out
                          ? 'Hết hàng'
                          : `Còn ${formatNumber(p.stock)} ${p.unit}${full ? ' (đã đủ trong giỏ)' : ''}`}
                      </div>
                      {qtyInCart(p.id) > 0 && (
                        <span className="pos-card__qty">{qtyInCart(p.id)}</span>
                      )}
                    </button>
                  );
                  return p.low_stock && !out ? (
                    <Badge.Ribbon key={p.id} text="Tồn thấp" color="orange">
                      {card}
                    </Badge.Ribbon>
                  ) : (
                    card
                  );
                })}
              </div>
            )}
          </Card>
        </Col>

        {/* ----------------------------------------------- phải: giỏ hàng */}
        <Col xs={24} md={10} xl={9}>
          <Card
            variant="borderless"
            className="page-card pos-cart"
            title={
              <Space>
                <span>Giỏ hàng</span>
                {cart.lines.length > 0 && <Tag color="blue">{cart.lines.length} sản phẩm</Tag>}
              </Space>
            }
            extra={
              cart.lines.length > 0 && (
                <ConfirmButton
                  type="text"
                  size="small"
                  danger
                  icon={<CloseOutlined />}
                  confirmTitle="Xóa toàn bộ giỏ hàng?"
                  onConfirm={() => {
                    reset();
                    setPromoInput('');
                    setShortage(null);
                  }}
                >
                  Xóa giỏ
                </ConfirmButton>
              )
            }
          >
            <MemberSelect
              value={member?.id}
              initialMember={member}
              onChange={(_, m) => patch({ member: m ?? null })}
              placeholder="Khách lẻ – tìm hội viên để áp giá hội viên"
            />
            {member && member.status !== 'ACTIVE' && (
              <Typography.Text type="warning" style={{ fontSize: 12 }}>
                Hội viên không ở trạng thái hoạt động – tính theo giá bán thường.
              </Typography.Text>
            )}

            <div className="pos-lines">
              {cart.lines.length === 0 ? (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description="Chưa có sản phẩm – chạm vào sản phẩm hoặc quét mã vạch"
                />
              ) : (
                cart.lines.map((l) => {
                  const price = unitPrice(l.product, member);
                  const short = shortage === l.product.name;
                  return (
                    <div key={l.product.id} className={`pos-line${short ? ' is-short' : ''}`}>
                      <div className="pos-line__info">
                        <div className="pos-line__name">{l.product.name}</div>
                        <PriceText price={price} original={l.product.sell_price} />
                        {short && (
                          <div className="pos-line__err">
                            Không đủ hàng – tồn hiện tại {formatNumber(l.product.stock)}{' '}
                            {l.product.unit}
                          </div>
                        )}
                      </div>
                      <div className="pos-line__qty">
                        <Button
                          size="small"
                          icon={<MinusOutlined />}
                          onClick={() => setQty(l.product.id, l.quantity - 1)}
                          aria-label="Giảm"
                        />
                        <InputNumber
                          size="small"
                          min={1}
                          max={Math.max(1, l.product.stock)}
                          precision={0}
                          value={l.quantity}
                          onChange={(v) => v && setQty(l.product.id, v)}
                          controls={false}
                          style={{ width: 48 }}
                        />
                        <Tooltip
                          title={
                            l.quantity >= l.product.stock
                              ? `Chỉ còn ${l.product.stock} ${l.product.unit}`
                              : undefined
                          }
                        >
                          <Button
                            size="small"
                            icon={<PlusOutlined />}
                            disabled={l.quantity >= l.product.stock}
                            onClick={() => setQty(l.product.id, l.quantity + 1)}
                            aria-label="Tăng"
                          />
                        </Tooltip>
                      </div>
                      <div className="pos-line__amount">{formatMoney(price * l.quantity)}</div>
                      <Button
                        type="text"
                        size="small"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => remove(l.product.id)}
                        aria-label="Xóa dòng"
                      />
                    </div>
                  );
                })
              )}
            </div>

            <Space.Compact style={{ width: '100%', marginBottom: 4 }}>
              <Input
                prefix={<TagOutlined />}
                placeholder="Mã khuyến mãi"
                value={promoInput}
                maxLength={30}
                onChange={(e) => setPromoInput(e.target.value.toUpperCase())}
                onPressEnter={applyPromo}
                allowClear
              />
              {cart.promoCode && cart.promoCode === promoInput.trim() ? (
                <Button
                  onClick={() => {
                    patch({ promoCode: '' });
                    setPromoInput('');
                  }}
                >
                  Bỏ mã
                </Button>
              ) : (
                <Button onClick={applyPromo} disabled={!promoInput.trim()}>
                  Áp dụng
                </Button>
              )}
            </Space.Compact>
            {cart.promoCode && (
              <div style={{ fontSize: 12, marginBottom: 6 }}>
                {promoQuery.isFetching ? (
                  <Typography.Text type="secondary">Đang kiểm tra mã…</Typography.Text>
                ) : promoError ? (
                  <Typography.Text type="danger">{promoError}</Typography.Text>
                ) : promoQuery.data ? (
                  <Typography.Text type="success">
                    <CheckCircleOutlined /> {promoQuery.data.promotion.name} – giảm{' '}
                    {formatMoney(discount)}
                  </Typography.Text>
                ) : null}
              </div>
            )}

            <div className="pos-sum">
              <span>Tạm tính</span>
              <b>{formatMoney(subtotal)}</b>
              {memberSaving > 0 && (
                <>
                  <span>Tiết kiệm giá hội viên</span>
                  <span style={{ color: '#12805C' }}>{formatMoney(memberSaving)}</span>
                </>
              )}
              <span>Giảm giá</span>
              <span>{discount > 0 ? `−${formatMoney(discount)}` : formatMoney(0)}</span>
              <span className="pos-total-label">Tổng cộng</span>
              <b className="pos-total">{formatMoney(total)}</b>
            </div>

            {total > 0 && (
              <PaymentPanel
                total={total}
                payments={cart.payments}
                onChange={(payments) => patch({ payments })}
                disabled={order.isPending}
              />
            )}

            <Input.TextArea
              placeholder="Ghi chú đơn (không bắt buộc)"
              autoSize={{ minRows: 1, maxRows: 3 }}
              maxLength={500}
              value={cart.note}
              onChange={(e) => patch({ note: e.target.value })}
              style={{ margin: '8px 0' }}
            />

            <Tooltip title={blockReason ?? undefined}>
              <Button
                block
                size="large"
                color="orange"
                variant="solid"
                className="pos-pay-btn"
                loading={order.isPending}
                disabled={!!blockReason}
                onClick={checkout}
              >
                Thanh toán {total > 0 ? formatMoney(total) : ''} <kbd>F9</kbd>
              </Button>
            </Tooltip>
            {cart.lines.length > 0 && blockReason && (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {blockReason}
              </Typography.Text>
            )}
          </Card>
        </Col>
      </Row>

      <SuccessModal result={result} onNewOrder={newOrder} />
    </div>
  );
}

/** `/app/pos` – màn hình bán hàng tại quầy; chi nhánh = chi nhánh làm việc ở header. */
export default function PosPage() {
  const { branchId, setBranchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const branch = branches.find((b) => b.id === branchId);

  if (!branchId) {
    return (
      <Card variant="borderless" className="page-card">
        <PageTitle title="Bán hàng (POS)" />
        <Result
          status="info"
          title="Chọn chi nhánh bán hàng"
          subTitle="POS bán và trừ kho theo chi nhánh làm việc. Vui lòng chọn chi nhánh (hoặc đổi ở thanh trên cùng)."
          extra={
            <div style={{ maxWidth: 320, margin: '0 auto' }}>
              <BranchSelect
                activeOnly
                size="large"
                onChange={(v) => typeof v === 'number' && setBranchId(v)}
              />
            </div>
          }
        />
      </Card>
    );
  }

  if (branch && branch.status === 'INACTIVE') {
    return (
      <Alert
        type="warning"
        showIcon
        message={`Chi nhánh ${branch.name} đang ngừng hoạt động – không thể bán hàng.`}
      />
    );
  }

  return <PosScreen key={branchId} branchId={branchId} branchName={branch?.name ?? ''} />;
}
