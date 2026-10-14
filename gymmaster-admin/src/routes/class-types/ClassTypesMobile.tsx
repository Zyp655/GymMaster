import { useCallback, useState } from 'react';
import { App } from 'antd';
import { DeleteOutlined, EditOutlined, EyeOutlined } from '@ant-design/icons';
import classApi from '@/api/class';
import type { ClassLevel, ClassType } from '@/api/class';
import type { ActiveStatus, SortDir } from '@/api/types';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import { ACTIVE_STATUS_OPTIONS, STATUS_PRESETS } from '@/constants/status';
import { PERM } from '@/constants/permissions';
import { usePermission } from '@/hooks/usePermission';
import { formatDateTime } from '@/utils/format';
import { LEVEL_META, LEVEL_OPTIONS } from '../classes/constants';
import ClassTypeForm from './ClassTypeForm';
import { useDeleteClassTypes } from './queries';

/** Màn hình /app/class-types trên điện thoại. */
export default function ClassTypesMobile() {
  const { can } = usePermission();
  const { modal } = App.useApp();
  const remove = useDeleteClassTypes();
  const [form, setForm] = useState<{ open: boolean; record: ClassType | null }>({
    open: false,
    record: null,
  });

  const fetchPage = useCallback(({ page, limit, keyword, sort, filters }: MobileFetchArgs) => {
    const [order_by, order_dir] = (sort ?? 'id:DESC').split(':') as [string, SortDir];
    return classApi.listTypes({
      page,
      limit,
      keyword: keyword || undefined,
      order_by,
      order_dir,
      status: (filters.status as ActiveStatus | undefined) || undefined,
      level: (filters.level as ClassLevel | undefined) || undefined,
    });
  }, []);

  const openForm = (record: ClassType | null) => setForm({ open: true, record });

  const confirmDelete = (record: ClassType) =>
    modal.confirm({
      title: `Xóa loại lớp ${record.code}?`,
      content: `"${record.name}" sẽ bị xóa. Không thể xóa nếu còn lịch lớp sắp diễn ra.`,
      okText: 'Xóa',
      okButtonProps: { danger: true },
      cancelText: 'Hủy',
      onOk: () => remove.mutateAsync([record]).catch(() => undefined),
    });

  const canManage = can(PERM.CLASS_MANAGE);

  return (
    <>
      <MobileList<ClassType>
        title="Loại lớp"
        queryKey={[classApi.typeKey, 'list']}
        fetchPage={fetchPage}
        searchPlaceholder="Tìm theo mã, tên"
        emptyText="Chưa có loại lớp nào"
        sorts={[
          { label: 'Mới nhất', value: 'id:DESC' },
          { label: 'Tên A → Z', value: 'name:ASC' },
          { label: 'Mã A → Z', value: 'code:ASC' },
        ]}
        filters={[
          { name: 'level', label: 'Cấp độ', options: LEVEL_OPTIONS },
          { name: 'status', label: 'Trạng thái', options: ACTIVE_STATUS_OPTIONS },
        ]}
        row={{
          title: (r) => (
            <span>
              <span
                aria-hidden
                style={{
                  display: 'inline-block',
                  width: 10,
                  height: 10,
                  borderRadius: 3,
                  background: r.color,
                  marginRight: 6,
                }}
              />
              {r.name}
            </span>
          ),
          subtitle: (r) => r.description || LEVEL_META[r.level]?.label,
          meta: (r) => [r.code, `${r.default_duration_min} phút`, `${r.default_capacity} người`],
          tags: (r) => [
            {
              label: STATUS_PRESETS[r.status]?.label ?? r.status,
              color: STATUS_PRESETS[r.status]?.color,
            },
          ],
        }}
        onAdd={() => openForm(null)}
        addPerm={PERM.CLASS_MANAGE}
        details={(r) => [
          ['Mã', r.code],
          ['Cấp độ', LEVEL_META[r.level]?.label ?? r.level],
          ['Thời lượng', `${r.default_duration_min} phút`],
          ['Sức chứa', `${r.default_capacity} người`],
          ['Cập nhật', formatDateTime(r.updated_at)],
        ]}
        actions={() => [
          {
            key: 'edit',
            icon: canManage ? <EditOutlined /> : <EyeOutlined />,
            label: canManage ? 'Sửa loại lớp' : 'Xem chi tiết',
            onClick: openForm,
          },
          ...(canManage
            ? [
                {
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  label: 'Xóa loại lớp',
                  danger: true,
                  onClick: confirmDelete,
                },
              ]
            : []),
        ]}
      />
      <ClassTypeForm
        open={form.open}
        record={form.record}
        onClose={() => setForm((f) => ({ ...f, open: false }))}
      />
    </>
  );
}
