import { useCallback, useState } from 'react';
import staffApi from '@/api/staff';
import type { Staff, StaffPosition, StaffStatus } from '@/api/staff';
import type { SortDir } from '@/api/types';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import { PERM } from '@/constants/permissions';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { formatDate, formatMoney, formatPhone } from '@/utils/format';
import ResetPasswordModal from './ResetPasswordModal';
import StaffForm from './StaffForm';
import {
  POSITION_META,
  POSITION_OPTIONS,
  STAFF_STATUS_MAP,
  STAFF_STATUS_OPTIONS,
} from './constants';
import { useStaffActions } from './useStaffActions';

/** /app/staff trên điện thoại. */
export default function StaffMobile() {
  const { branchId } = useWorkingBranch();
  const [form, setForm] = useState<{ open: boolean; record: Staff | null }>({
    open: false,
    record: null,
  });
  const [resetFor, setResetFor] = useState<Staff | null>(null);
  const openForm = (record: Staff | null) => setForm({ open: true, record });
  const { itemsFor } = useStaffActions({ onEdit: openForm, onResetPassword: setResetFor });

  const fetchPage = useCallback(
    ({ page, limit, keyword, sort, filters }: MobileFetchArgs) => {
      const [order_by, order_dir] = (sort ?? 'id:DESC').split(':') as [string, SortDir];
      return staffApi.list({
        page,
        limit,
        keyword: keyword || undefined,
        order_by,
        order_dir,
        branch_id: branchId ?? undefined,
        position: (filters.position as StaffPosition | undefined) || undefined,
        status: (filters.status as StaffStatus | undefined) || undefined,
      });
    },
    [branchId],
  );

  return (
    <>
      <MobileList<Staff>
        title="Nhân viên"
        queryKey={[staffApi.key, 'list', branchId]}
        fetchPage={fetchPage}
        searchPlaceholder="Tìm theo tên, email, SĐT, mã NV"
        emptyText="Chưa có nhân viên nào"
        sorts={[
          { label: 'Mới nhất', value: 'id:DESC' },
          { label: 'Tên A → Z', value: 'full_name:ASC' },
          { label: 'Mã NV', value: 'staff_code:ASC' },
        ]}
        filters={[
          { name: 'position', label: 'Chức vụ', options: POSITION_OPTIONS },
          { name: 'status', label: 'Trạng thái', options: STAFF_STATUS_OPTIONS },
        ]}
        row={{
          thumb: (r) => r.avatar_url || null,
          avatar: true,
          title: (r) => r.full_name,
          subtitle: (r) =>
            r.roles.map((x) => `${x.role_name} · ${x.branch_name || 'Toàn hệ thống'}`).join(', '),
          meta: (r) => [r.staff_profile?.staff_code, r.phone && formatPhone(r.phone)],
          tags: (r) => [
            ...(r.staff_profile
              ? [
                  {
                    label: POSITION_META[r.staff_profile.position]?.label,
                    color: POSITION_META[r.staff_profile.position]?.color,
                  },
                ]
              : []),
            ...(r.status === 'LOCKED'
              ? [{ label: STAFF_STATUS_MAP.LOCKED.label, color: STAFF_STATUS_MAP.LOCKED.color }]
              : []),
          ],
        }}
        onAdd={() => openForm(null)}
        addPerm={PERM.STAFF_CREATE}
        details={(r) => [
          ['Mã NV', r.staff_profile?.staff_code],
          ['Email', r.email],
          ['Điện thoại', r.phone ? formatPhone(r.phone) : ''],
          ['Ngày vào làm', r.staff_profile ? formatDate(r.staff_profile.hire_date) : ''],
          ['Lương cơ bản', r.staff_profile ? formatMoney(r.staff_profile.base_salary) : ''],
          ['Trạng thái', STAFF_STATUS_MAP[r.status]?.label],
        ]}
        actions={(r) => itemsFor(r)}
      />
      <StaffForm
        open={form.open}
        record={form.record}
        onClose={() => setForm((f) => ({ ...f, open: false }))}
      />
      <ResetPasswordModal staff={resetFor} onClose={() => setResetFor(null)} />
    </>
  );
}
