import { useCallback, useState } from 'react';
import { App } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import promotionApi from '@/api/promotion';
import type { Promotion, PromotionAppliesTo, PromotionListParams } from '@/api/promotion';
import type { SortDir } from '@/api/types';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { formatDateTime, formatMoney } from '@/utils/format';
import PromotionCheckModal from './PromotionCheckModal';
import PromotionForm from './PromotionForm';
import {
  APPLIES_TO_LABELS,
  APPLIES_TO_OPTIONS,
  VALIDITY_MAP,
  VALIDITY_OPTIONS,
  discountText,
} from './constants';
import { useDeletePromotion } from './queries';

export default function PromotionsMobile() {
  const { can } = usePermission();
  const { modal } = App.useApp();
  const remove = useDeletePromotion();
  const canManage = can(PERM.PROMOTION_MANAGE);
  const [form, setForm] = useState<{ open: boolean; record: Promotion | null }>({
    open: false,
    record: null,
  });
  const [checkCode, setCheckCode] = useState<string | null>(null);
  const openForm = (record: Promotion | null) => setForm({ open: true, record });

  const fetchPage = useCallback(({ page, limit, keyword, sort, filters }: MobileFetchArgs) => {
    const [order_by, order_dir] = (sort ?? 'id:DESC').split(':') as [string, SortDir];
    return promotionApi.list({
      page,
      limit,
      keyword: keyword || undefined,
      order_by,
      order_dir,
      validity: (filters.validity as PromotionListParams['validity']) || undefined,
      applies_to: (filters.applies_to as PromotionAppliesTo | undefined) || undefined,
    });
  }, []);

  return (
    <>
      <MobileList<Promotion>
        title="Khuyến mãi"
        queryKey={[promotionApi.key, 'list']}
        fetchPage={fetchPage}
        searchPlaceholder="Tìm theo mã, tên"
        emptyText="Chưa có khuyến mãi nào"
        sorts={[
          { label: 'Mới nhất', value: 'id:DESC' },
          { label: 'Sắp hết hạn', value: 'end_at:ASC' },
          { label: 'Dùng nhiều nhất', value: 'used_count:DESC' },
        ]}
        filters={[
          { name: 'validity', label: 'Hiệu lực', options: VALIDITY_OPTIONS },
          { name: 'applies_to', label: 'Áp dụng cho', options: APPLIES_TO_OPTIONS },
        ]}
        row={{
          title: (r) => r.name,
          subtitle: (r) => discountText(r),
          meta: (r) => [r.code, `${r.used_count}${r.usage_limit ? `/${r.usage_limit}` : ''} lượt`],
          tags: (r) => [
            { label: VALIDITY_MAP[r.validity]?.label, color: VALIDITY_MAP[r.validity]?.color },
          ],
        }}
        onAdd={() => openForm(null)}
        addPerm={PERM.PROMOTION_MANAGE}
        details={(r) => [
          ['Mã', r.code],
          ['Áp dụng cho', APPLIES_TO_LABELS[r.applies_to]],
          ['Đơn tối thiểu', formatMoney(r.min_amount)],
          ['Bắt đầu', formatDateTime(r.start_at)],
          ['Kết thúc', formatDateTime(r.end_at)],
          ['Đã dùng', `${r.used_count}${r.usage_limit ? ` / ${r.usage_limit}` : ''}`],
        ]}
        actions={(r) => [
          {
            key: 'edit',
            icon: canManage ? <EditOutlined /> : <EyeOutlined />,
            label: canManage ? 'Sửa khuyến mãi' : 'Xem chi tiết',
            onClick: openForm,
          },
          {
            key: 'check',
            icon: <SafetyCertificateOutlined />,
            label: 'Kiểm tra mã',
            onClick: () => setCheckCode(r.code),
          },
          ...(canManage
            ? [
                {
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  label: 'Xóa khuyến mãi',
                  danger: true,
                  onClick: () =>
                    modal.confirm({
                      title: `Xóa mã ${r.code}?`,
                      content: 'Mã đã được sử dụng không thể xóa.',
                      okText: 'Xóa',
                      okButtonProps: { danger: true },
                      cancelText: 'Hủy',
                      onOk: () => remove.mutateAsync(r).catch(() => undefined),
                    }),
                },
              ]
            : []),
        ]}
      />
      <PromotionForm
        open={form.open}
        record={form.record}
        onClose={() => setForm((f) => ({ ...f, open: false }))}
      />
      <PromotionCheckModal
        open={checkCode !== null}
        initialCode={checkCode || undefined}
        onClose={() => setCheckCode(null)}
      />
    </>
  );
}
