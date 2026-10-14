import { useCallback, useState } from 'react';
import { App } from 'antd';
import { DeleteOutlined, EditOutlined, EyeOutlined } from '@ant-design/icons';
import roomApi from '@/api/room';
import type { Room } from '@/api/room';
import type { ActiveStatus, SortDir } from '@/api/types';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import { ACTIVE_STATUS_OPTIONS, STATUS_PRESETS } from '@/constants/status';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { formatDateTime } from '@/utils/format';
import RoomForm from './RoomForm';
import { useDeleteRooms } from './queries';

/** /app/rooms trên điện thoại. */
export default function RoomsMobile() {
  const { can } = usePermission();
  const { modal } = App.useApp();
  const { branchId } = useWorkingBranch();
  const remove = useDeleteRooms();
  const [form, setForm] = useState<{ open: boolean; record: Room | null }>({
    open: false,
    record: null,
  });

  const fetchPage = useCallback(
    ({ page, limit, keyword, sort, filters }: MobileFetchArgs) => {
      const [order_by, order_dir] = (sort ?? 'id:DESC').split(':') as [string, SortDir];
      return roomApi.list({
        page,
        limit,
        keyword: keyword || undefined,
        order_by,
        order_dir,
        branch_id: branchId ?? undefined,
        status: (filters.status as ActiveStatus | undefined) || undefined,
      });
    },
    [branchId],
  );

  const openForm = (record: Room | null) => setForm({ open: true, record });
  const canManage = can(PERM.ROOM_MANAGE);

  return (
    <>
      <MobileList<Room>
        title="Phòng tập"
        queryKey={[roomApi.key, 'list', branchId]}
        fetchPage={fetchPage}
        searchPlaceholder="Tìm theo tên, mô tả"
        emptyText="Chưa có phòng nào"
        sorts={[
          { label: 'Mới nhất', value: 'id:DESC' },
          { label: 'Tên A → Z', value: 'name:ASC' },
          { label: 'Sức chứa lớn nhất', value: 'capacity:DESC' },
        ]}
        filters={[{ name: 'status', label: 'Trạng thái', options: ACTIVE_STATUS_OPTIONS }]}
        row={{
          title: (r) => r.name,
          subtitle: (r) => r.branch_name,
          meta: (r) => [`${r.capacity} người`, r.description],
          tags: (r) => [
            {
              label: STATUS_PRESETS[r.status]?.label ?? r.status,
              color: STATUS_PRESETS[r.status]?.color,
            },
          ],
        }}
        onAdd={() => openForm(null)}
        addPerm={PERM.ROOM_MANAGE}
        details={(r) => [
          ['Chi nhánh', r.branch_name],
          ['Sức chứa', `${r.capacity} người`],
          ['Mô tả', r.description],
          ['Cập nhật', formatDateTime(r.updated_at)],
        ]}
        actions={(r) => [
          {
            key: 'edit',
            icon: canManage ? <EditOutlined /> : <EyeOutlined />,
            label: canManage ? 'Sửa phòng' : 'Xem chi tiết',
            onClick: openForm,
          },
          ...(canManage
            ? [
                {
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  label: 'Xóa phòng',
                  danger: true,
                  onClick: () =>
                    modal.confirm({
                      title: `Xóa phòng ${r.name}?`,
                      content: 'Không thể xóa nếu phòng còn lịch lớp sắp diễn ra.',
                      okText: 'Xóa',
                      okButtonProps: { danger: true },
                      cancelText: 'Hủy',
                      onOk: () => remove.mutateAsync([r]).catch(() => undefined),
                    }),
                },
              ]
            : []),
        ]}
      />
      <RoomForm
        open={form.open}
        record={form.record}
        onClose={() => setForm((f) => ({ ...f, open: false }))}
      />
    </>
  );
}
