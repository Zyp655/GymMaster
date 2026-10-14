import { useCallback, useState } from 'react';
import { App } from 'antd';
import { DeleteOutlined, EyeOutlined, PhoneOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import memberApi, {
  GENDER_OPTIONS,
  MEMBER_STATUS_MAP,
  MEMBER_STATUS_OPTIONS,
  SOURCE_LABEL,
  SOURCE_OPTIONS,
  SUBSCRIPTION_FILTER_OPTIONS,
} from '@/api/member';
import type { MemberItem, MemberListParams } from '@/api/member';
import type { SortDir } from '@/api/types';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import { PERM } from '@/constants/permissions';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { usePermission } from '@/hooks/usePermission';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { formatDate, formatPhone } from '@/utils/format';
import MemberForm from './MemberForm';
import { daysLeftText, expiryColor } from './utils';
import { useDeleteMember } from './queries';

/** /app/members trên điện thoại. */
export default function MembersMobile() {
  const { can } = usePermission();
  const { modal } = App.useApp();
  const navigate = useNavigate();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const remove = useDeleteMember();
  const [formOpen, setFormOpen] = useState(false);

  const fetchPage = useCallback(
    ({ page, limit, keyword, sort, filters }: MobileFetchArgs) => {
      const [order_by, order_dir] = (sort ?? 'id:DESC').split(':') as [string, SortDir];
      const f = filters as Partial<Record<string, string | number>>;
      return memberApi.list({
        page,
        limit,
        keyword: keyword || undefined,
        order_by,
        order_dir,
        branch_id: (f.branch_id as number | undefined) ?? branchId ?? undefined,
        status: (f.status as MemberListParams['status']) || undefined,
        gender: (f.gender as MemberListParams['gender']) || undefined,
        source: (f.source as MemberListParams['source']) || undefined,
        subscription_status:
          (f.subscription_status as MemberListParams['subscription_status']) || undefined,
      });
    },
    [branchId],
  );

  const confirmDelete = (r: MemberItem) =>
    modal.confirm({
      title: `Xóa hội viên ${r.member_code}?`,
      content: `"${r.full_name}" sẽ bị xóa. Không thể xóa nếu còn gói đang hiệu lực hoặc bảo lưu.`,
      okText: 'Xóa',
      okButtonProps: { danger: true },
      cancelText: 'Hủy',
      onOk: () => remove.mutateAsync(r.id).catch(() => undefined),
    });

  return (
    <>
      <MobileList<MemberItem>
        title="Hội viên"
        queryKey={[memberApi.key, 'list', branchId]}
        fetchPage={fetchPage}
        searchPlaceholder="Tên, SĐT, mã GM…"
        emptyText="Chưa có hội viên nào"
        sorts={[
          { label: 'Mới nhất', value: 'id:DESC' },
          { label: 'Tên A → Z', value: 'full_name:ASC' },
          { label: 'Mã A → Z', value: 'member_code:ASC' },
        ]}
        filters={[
          {
            name: 'branch_id',
            label: 'Chi nhánh',
            options: branches.map((b) => ({ label: b.name, value: b.id })),
          },
          {
            name: 'subscription_status',
            label: 'Tình trạng gói',
            options: SUBSCRIPTION_FILTER_OPTIONS,
          },
          { name: 'status', label: 'Trạng thái', options: MEMBER_STATUS_OPTIONS },
          { name: 'gender', label: 'Giới tính', options: GENDER_OPTIONS },
          { name: 'source', label: 'Nguồn', options: SOURCE_OPTIONS },
        ]}
        row={{
          thumb: (r) => r.avatar_url,
          avatar: true,
          title: (r) => r.full_name,
          subtitle: (r) =>
            r.active_subscription
              ? `${r.active_subscription.plan_name} · HSD ${formatDate(r.active_subscription.end_date)}`
              : 'Chưa có gói',
          meta: (r) => [r.member_code, formatPhone(r.phone), r.home_branch_name],
          tags: (r) => [
            {
              label: MEMBER_STATUS_MAP[r.status]?.label ?? r.status,
              color: MEMBER_STATUS_MAP[r.status]?.color,
            },
            ...(r.active_subscription && expiryColor(r.active_subscription.days_left)
              ? [
                  {
                    label: daysLeftText(r.active_subscription.days_left),
                    color: expiryColor(r.active_subscription.days_left),
                  },
                ]
              : []),
          ],
        }}
        onOpen={(r) => navigate(`/app/members/${r.id}`)}
        onAdd={() => setFormOpen(true)}
        addPerm={PERM.MEMBER_CREATE}
        details={(r) => [
          ['Mã', r.member_code],
          ['Điện thoại', formatPhone(r.phone)],
          ['Chi nhánh', r.home_branch_name],
          ['Nguồn', SOURCE_LABEL[r.source]],
        ]}
        actions={(r) => [
          {
            key: 'view',
            icon: <EyeOutlined />,
            label: 'Xem hồ sơ',
            onClick: () => navigate(`/app/members/${r.id}`),
          },
          {
            key: 'call',
            icon: <PhoneOutlined />,
            label: `Gọi ${formatPhone(r.phone)}`,
            onClick: () => window.open(`tel:${r.phone}`),
          },
          ...(can(PERM.MEMBER_DELETE)
            ? [
                {
                  key: 'delete',
                  icon: <DeleteOutlined />,
                  label: 'Xóa hội viên',
                  danger: true,
                  onClick: confirmDelete,
                },
              ]
            : []),
        ]}
      />
      <MemberForm
        open={formOpen}
        record={null}
        onClose={() => setFormOpen(false)}
        onSaved={(m) => navigate(`/app/members/${m.id}`)}
      />
    </>
  );
}
