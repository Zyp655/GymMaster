/* eslint-disable react-refresh/only-export-components -- file registry (cấu hình), không phải component */
/**
 * Registry tab của trang hồ sơ hội viên `/app/members/:id`.
 *
 * Module khác (bán gói, hóa đơn, lớp/PT...) thêm tab bằng cách:
 *  - thay `render` của tab placeholder tương ứng bên dưới bằng component thật, hoặc
 *  - gọi `registerMemberTab({...})` (ghi đè nếu trùng `key`), ví dụ trong file tab của module:
 *
 *   registerMemberTab({
 *     key: 'subscriptions', label: 'Gói tập', perm: [PERM.SUBSCRIPTION_VIEW], order: 20,
 *     render: (member) => <MemberSubscriptionsTab member={member} />,
 *   });
 *
 * `render` nhận `Member` (hồ sơ đầy đủ). Tab chỉ hiện khi user có MỘT trong các quyền `perm`.
 * Component nặng nên `lazy()` – trang chi tiết đã bọc `Suspense`.
 */
import { lazy } from 'react';
import type { ReactNode } from 'react';
import type { Member } from '@/api/member';
import { PERM } from '@/constants/permissions';

export interface MemberTabDef {
  /** Khóa duy nhất – cũng là giá trị `?tab=` trên URL. */
  key: string;
  label: ReactNode;
  /** anyOf; bỏ trống = ai xem được hồ sơ đều thấy. */
  perm?: string[];
  /** Thứ tự hiển thị (nhỏ trước). */
  order: number;
  render: (member: Member) => ReactNode;
}

const OverviewTab = lazy(() => import('./OverviewTab'));
const MetricsTab = lazy(() => import('./MetricsTab'));
const CheckinHistoryTab = lazy(() => import('./CheckinHistoryTab'));
const MemberBookingsTab = lazy(() => import('@/routes/classes/MemberBookingsTab'));
const MemberSubscriptionsTab = lazy(() => import('@/routes/subscriptions/MemberSubscriptionsTab'));
const MemberInvoicesTab = lazy(() => import('@/routes/invoices/MemberInvoicesTab'));

const tabs: MemberTabDef[] = [
  { key: 'overview', label: 'Tổng quan', order: 0, render: (m) => <OverviewTab member={m} /> },
  {
    key: 'metrics',
    label: 'Chỉ số cơ thể',
    order: 10,
    render: (m) => <MetricsTab member={m} />,
  },
  {
    key: 'subscriptions',
    label: 'Gói tập',
    perm: [PERM.SUBSCRIPTION_VIEW, PERM.SUBSCRIPTION_SELL],
    order: 20,
    render: (m) => <MemberSubscriptionsTab member={m} />,
  },
  {
    key: 'invoices',
    label: 'Hóa đơn',
    perm: [PERM.INVOICE_VIEW],
    order: 30,
    render: (m) => <MemberInvoicesTab member={m} />,
  },
  {
    key: 'classes',
    label: 'Lớp & PT',
    perm: [PERM.CLASS_VIEW, PERM.BOOKING_MANAGE, PERM.PT_VIEW],
    order: 40,
    render: (m) => <MemberBookingsTab memberId={m.id} />,
  },
  {
    key: 'checkins',
    label: 'Lịch sử check-in',
    perm: [PERM.CHECKIN_VIEW],
    order: 50,
    render: (m) => <CheckinHistoryTab member={m} />,
  },
];

/** Thêm hoặc ghi đè (trùng `key`) một tab. */
export function registerMemberTab(tab: MemberTabDef): void {
  const i = tabs.findIndex((t) => t.key === tab.key);
  if (i >= 0) tabs[i] = tab;
  else tabs.push(tab);
}

/** Danh sách tab đã sắp theo `order`. */
export function getMemberTabs(): MemberTabDef[] {
  return [...tabs].sort((a, b) => a.order - b.order);
}

export default tabs;
