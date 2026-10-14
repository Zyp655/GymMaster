import { useCallback, useState } from 'react';
import { App } from 'antd';
import { DeleteOutlined, EditOutlined, EyeOutlined, PhoneOutlined } from '@ant-design/icons';
import branchApi from '@/api/branch';
import type { ActiveStatus, Branch, SortDir } from '@/api/types';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import { ACTIVE_STATUS_OPTIONS, STATUS_PRESETS } from '@/constants/status';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { formatDateTime, formatPhone } from '@/utils/format';
import BranchForm from './BranchForm';
import { useDeleteBranches } from './queries';

/** Màn hình /app/branches trên điện thoại – cấu hình mỏng trên MobileList. */
export default function BranchesMobile() {
  const { can } = usePermission();
  const { modal } = App.useApp();
  const remove = useDeleteBranches();
  const [form, setForm] = useState<{ open: boolean; record: Branch | null }>({
    open: false,
    record: null,
  });

  const fetchPage = useCallback(({ page, limit, keyword, sort, filters }: MobileFetchArgs) => {
    const [order_by, order_dir] = (sort ?? 'id:DESC').split(':') as [string, SortDir];
    return branchApi.list({
      page,
      limit,
      keyword: keyword || undefined,
      order_by,
      order_dir,
      status: (filters.status as ActiveStatus | undefined) || undefined,
    });
  }, []);

  const openForm = (record: Branch | null) => setForm({ open: true, record });

  const confirmDelete = (record: Branch) =>
    modal.confirm({
      title: `Xóa chi nhánh ${record.code}?`,
      content: `"${record.name}" sẽ bị xóa. Không thể xóa nếu còn nhân viên thuộc chi nhánh.`,
      okText: 'Xóa',
      okButtonProps: { danger: true },
      cancelText: 'Hủy',
      onOk: () => remove.mutateAsync([record]).catch(() => undefined),
    });

  return (
    <>
      <MobileList<Branch>
        title="Chi nhánh"
        queryKey={[branchApi.key, 'list']}
        fetchPage={fetchPage}
        searchPlaceholder="Tìm theo mã, tên, địa chỉ"
        emptyText="Chưa có chi nhánh nào"
        sorts={[
          { label: 'Mới nhất', value: 'id:DESC' },
          { label: 'Tên A → Z', value: 'name:ASC' },
          { label: 'Mã A → Z', value: 'code:ASC' },
        ]}
        filters={[{ name: 'status', label: 'Trạng thái', options: ACTIVE_STATUS_OPTIONS }]}
        row={{
          title: (r) => r.name,
          subtitle: (r) => r.address || 'Chưa có địa chỉ',
          meta: (r) => [r.code, `${r.open_time}–${r.close_time}`, r.phone && formatPhone(r.phone)],
          tags: (r) => [
            {
              label: STATUS_PRESETS[r.status]?.label ?? r.status,
              color: STATUS_PRESETS[r.status]?.color,
            },
          ],
        }}
        onAdd={() => openForm(null)}
        addPerm={PERM.BRANCH_CREATE}
        details={(r) => [
          ['Mã', r.code],
          ['Giờ mở cửa', `${r.open_time}–${r.close_time}`],
          ['Điện thoại', r.phone ? formatPhone(r.phone) : ''],
          ['Cập nhật', formatDateTime(r.updated_at)],
        ]}
        actions={(r) => [
          {
            key: 'edit',
            icon: can(PERM.BRANCH_UPDATE) ? <EditOutlined /> : <EyeOutlined />,
            label: can(PERM.BRANCH_UPDATE) ? 'Sửa chi nhánh' : 'Xem chi tiết',
            onClick: openForm,
          },
          ...(r.phone
            ? [
                {
                  key: 'call',
                  icon: <PhoneOutlined />,
                  label: `Gọi ${formatPhone(r.phone)}`,
                  onClick: () => window.open(`tel:${r.phone}`),
                },
              ]
            : []),
          ...(can(PERM.BRANCH_DELETE)
            ? [
                {
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  label: 'Xóa chi nhánh',
                  danger: true,
                  onClick: confirmDelete,
                },
              ]
            : []),
        ]}
      />
      <BranchForm
        open={form.open}
        record={form.record}
        onClose={() => setForm((f) => ({ ...f, open: false }))}
      />
    </>
  );
}
