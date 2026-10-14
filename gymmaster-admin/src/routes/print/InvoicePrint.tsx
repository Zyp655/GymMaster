import { useEffect, useRef, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Button, Result, Segmented, Space, Spin } from 'antd';
import { ArrowLeftOutlined, PrinterOutlined } from '@ant-design/icons';
import { Navigate, useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import invoiceApi, { PAYMENT_METHOD_LABEL } from '@/api/invoice';
import type { InvoicePrintData } from '@/api/invoice';
import { INVOICE_STATUS_MAP } from '@/constants/status';
import { useAppSelector } from '@/store';
import { getErrorMessage } from '@/utils/errors';
import { formatDateTime, formatMoney, formatNumber, formatPhone } from '@/utils/format';
import './print.css';

type PaperSize = 'a5' | 'r80';
const SIZE_KEY = 'gm.invoicePrintSize';

const PAGE_CSS: Record<PaperSize, string> = {
  a5: '@page { size: A5 portrait; margin: 0; }',
  r80: '@page { size: 80mm auto; margin: 0; }',
};

function readSize(): PaperSize {
  try {
    return localStorage.getItem(SIZE_KEY) === 'r80' ? 'r80' : 'a5';
  } catch {
    return 'a5';
  }
}

/** Nội dung hóa đơn (dùng cho cả A5 và khổ 80mm). */
function Paper({ data, size }: { data: InvoicePrintData; size: PaperSize }) {
  const { invoice: inv, branch, member } = data;
  const a5 = size === 'a5';
  return (
    <div className={`inv-paper ${size}`}>
      <div className="inv-brand">
        <b>GYMMASTER</b>
        <div>{branch.name}</div>
        {branch.address && <div className="inv-muted">{branch.address}</div>}
        {(branch.phone || branch.email) && (
          <div className="inv-muted">
            {branch.phone && `ĐT: ${formatPhone(branch.phone)}`}
            {branch.phone && branch.email && ' · '}
            {branch.email}
          </div>
        )}
      </div>
      <h1>HÓA ĐƠN BÁN HÀNG</h1>
      <div className="inv-center inv-muted">
        Số: <b>{inv.invoice_no}</b> · Ngày {formatDateTime(inv.issued_at)}
      </div>
      {inv.status === 'VOID' && <div className="inv-void">ĐÃ HỦY</div>}

      <div className="inv-meta">
        <span>Khách hàng:</span>
        <b>{member ? `${member.full_name} (${member.member_code})` : 'Khách lẻ'}</b>
        {member?.phone && (
          <>
            <span>Điện thoại:</span>
            <span>{formatPhone(member.phone)}</span>
          </>
        )}
        <span>Thu ngân:</span>
        <span>{inv.issued_by_name || '—'}</span>
      </div>

      <table>
        <thead>
          <tr>
            {a5 && <th>#</th>}
            <th style={{ textAlign: 'left' }}>Nội dung</th>
            <th className="num">SL</th>
            {a5 && <th className="num">Đơn giá</th>}
            {a5 && <th className="num">Giảm</th>}
            <th className="num">Thành tiền</th>
          </tr>
        </thead>
        <tbody>
          {inv.items.map((it, i) => (
            <tr key={it.id}>
              {a5 && <td>{i + 1}</td>}
              <td>
                {it.description}
                {!a5 && (
                  <div className="inv-muted">
                    {formatNumber(it.unit_price)}
                    {it.discount ? ` − ${formatNumber(it.discount)}` : ''}
                  </div>
                )}
              </td>
              <td className="num">{it.quantity}</td>
              {a5 && <td className="num">{formatNumber(it.unit_price)}</td>}
              {a5 && <td className="num">{it.discount ? formatNumber(it.discount) : ''}</td>}
              <td className="num">{formatNumber(it.amount)}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <table className="inv-totals">
        <tbody>
          <tr>
            <td>Tạm tính</td>
            <td className="num">{formatMoney(inv.subtotal)}</td>
          </tr>
          {inv.discount > 0 && (
            <tr>
              <td>Giảm giá{inv.promotion_code ? ` (${inv.promotion_code})` : ''}</td>
              <td className="num">−{formatMoney(inv.discount)}</td>
            </tr>
          )}
          <tr className="strong">
            <td>Tổng cộng</td>
            <td className="num">{formatMoney(inv.total)}</td>
          </tr>
          <tr>
            <td>Đã thanh toán</td>
            <td className="num">{formatMoney(inv.paid_amount)}</td>
          </tr>
          {inv.refunded_amount > 0 && (
            <tr>
              <td>Đã hoàn</td>
              <td className="num">−{formatMoney(inv.refunded_amount)}</td>
            </tr>
          )}
          <tr className="strong">
            <td>Còn nợ</td>
            <td className="num">{formatMoney(inv.due)}</td>
          </tr>
        </tbody>
      </table>
      <div>
        <i>Bằng chữ: {data.total_in_words}</i>
      </div>
      {inv.due > 0 && (
        <div>
          <i>Còn nợ bằng chữ: {data.due_in_words}</i>
        </div>
      )}

      {inv.payments.length > 0 && (
        <>
          <hr />
          <div className="inv-muted">
            {inv.payments.map((p) => (
              <div key={p.id}>
                {formatDateTime(p.paid_at)} · {PAYMENT_METHOD_LABEL[p.method]}
                {p.reference ? ` (${p.reference})` : ''}: {formatMoney(p.amount)}
              </div>
            ))}
          </div>
        </>
      )}
      <div className="inv-muted" style={{ marginTop: 4 }}>
        Trạng thái: {INVOICE_STATUS_MAP[inv.status]?.label ?? inv.status}
      </div>

      {a5 ? (
        <div className="inv-sign">
          <div>
            <b>Khách hàng</b>
            <div className="inv-muted">(Ký, ghi rõ họ tên)</div>
          </div>
          <div>
            <b>Người lập phiếu</b>
            <div className="inv-muted">(Ký, ghi rõ họ tên)</div>
          </div>
        </div>
      ) : (
        <hr />
      )}
      <div className="inv-center inv-muted" style={{ marginTop: 8 }}>
        Cảm ơn quý khách! · In lúc {formatDateTime(data.printed_at)} bởi {data.printed_by_name}
      </div>
    </div>
  );
}

/** `/print/invoices/:id` – trang in hóa đơn, không sidebar. `?auto=1` tự mở hộp thoại in. */
export default function InvoicePrintPage() {
  const user = useAppSelector((s) => s.auth.user);
  const location = useLocation();
  const navigate = useNavigate();
  const { id: idParam } = useParams();
  const [search] = useSearchParams();
  const id = Number(idParam);
  const [size, setSize] = useState<PaperSize>(readSize);
  const autoDone = useRef(false);

  const query = useQuery({
    queryKey: [invoiceApi.key, 'print', id],
    queryFn: () => invoiceApi.print(id),
    enabled: !!user && Number.isInteger(id) && id > 0,
  });

  useEffect(() => {
    document.title = query.data ? `Hóa đơn ${query.data.invoice.invoice_no}` : 'In hóa đơn';
  }, [query.data]);

  useEffect(() => {
    if (query.data && search.get('auto') === '1' && !autoDone.current) {
      autoDone.current = true;
      setTimeout(() => window.print(), 300);
    }
  }, [query.data, search]);

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />;
  }

  const changeSize = (v: PaperSize) => {
    setSize(v);
    try {
      localStorage.setItem(SIZE_KEY, v);
    } catch {
      /* bỏ qua */
    }
  };

  return (
    <div className="inv-print-page">
      <style>{PAGE_CSS[size]}</style>
      <div className="inv-print-toolbar no-print">
        <Space wrap>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => (window.history.length > 1 ? navigate(-1) : window.close())}
          >
            Quay lại
          </Button>
          <Segmented<PaperSize>
            value={size}
            onChange={changeSize}
            options={[
              { label: 'Khổ A5', value: 'a5' },
              { label: 'Khổ 80mm', value: 'r80' },
            ]}
          />
          <Button
            type="primary"
            icon={<PrinterOutlined />}
            disabled={!query.data}
            onClick={() => window.print()}
          >
            In hóa đơn
          </Button>
        </Space>
      </div>
      {query.isLoading ? (
        <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />
      ) : query.isError || !query.data ? (
        <Result
          status="warning"
          title={getErrorMessage(query.error, 'Không tải được hóa đơn')}
          extra={<Button onClick={() => query.refetch()}>Thử lại</Button>}
        />
      ) : (
        <Paper data={query.data} size={size} />
      )}
    </div>
  );
}
