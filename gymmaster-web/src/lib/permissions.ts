import type { AuthUser } from '@/types/auth';

/** Mã quyền – khớp bảng permissions ở backend. */
export const Perm = {
  BRANCH_VIEW: 'branch.view',
  BRANCH_CREATE: 'branch.create',
  BRANCH_UPDATE: 'branch.update',
  BRANCH_DELETE: 'branch.delete',
  ROLE_VIEW: 'role.view',
  ROLE_CREATE: 'role.create',
  ROLE_UPDATE: 'role.update',
  ROLE_DELETE: 'role.delete',
  STAFF_VIEW: 'staff.view',
  STAFF_CREATE: 'staff.create',
  STAFF_UPDATE: 'staff.update',
  STAFF_DELETE: 'staff.delete',
  ROOM_VIEW: 'room.view',
  ROOM_MANAGE: 'room.manage',
  SHIFT_VIEW: 'shift.view',
  SHIFT_MANAGE: 'shift.manage',
  MEMBER_VIEW: 'member.view',
  MEMBER_CREATE: 'member.create',
  MEMBER_UPDATE: 'member.update',
  MEMBER_DELETE: 'member.delete',
  MEMBER_METRIC: 'member.metric',
  MEMBER_IMPORT: 'member.import',
  PLAN_VIEW: 'plan.view',
  PLAN_MANAGE: 'plan.manage',
  PROMOTION_VIEW: 'promotion.view',
  PROMOTION_MANAGE: 'promotion.manage',
  SETTING_MANAGE: 'setting.manage',
  SUBSCRIPTION_VIEW: 'subscription.view',
  SUBSCRIPTION_SELL: 'subscription.sell',
  SUBSCRIPTION_FREEZE: 'subscription.freeze',
  SUBSCRIPTION_APPROVE_FREEZE: 'subscription.approve_freeze',
  SUBSCRIPTION_TRANSFER: 'subscription.transfer',
  SUBSCRIPTION_CANCEL: 'subscription.cancel',
  INVOICE_VIEW: 'invoice.view',
  INVOICE_COLLECT: 'invoice.collect',
  INVOICE_VOID: 'invoice.void',
  INVOICE_REFUND: 'invoice.refund',
  CHECKIN_VIEW: 'checkin.view',
  CHECKIN_SCAN: 'checkin.scan',
  CLASS_VIEW: 'class.view',
  CLASS_MANAGE: 'class.manage',
  BOOKING_MANAGE: 'booking.manage',
  PT_VIEW: 'pt.view',
  PT_SELL: 'pt.sell',
  PT_SCHEDULE: 'pt.schedule',
  PT_COMPLETE: 'pt.complete',
  PT_CANCEL_CONTRACT: 'pt.cancel_contract',
  COMMISSION_VIEW: 'commission.view',
  COMMISSION_PAY: 'commission.pay',
  PRODUCT_VIEW: 'product.view',
  PRODUCT_MANAGE: 'product.manage',
  INVENTORY_VIEW: 'inventory.view',
  INVENTORY_IMPORT: 'inventory.import',
  INVENTORY_TRANSFER: 'inventory.transfer',
  INVENTORY_ADJUST: 'inventory.adjust',
  POS_SELL: 'pos.sell',
  REPORT_VIEW: 'report.view',
  REPORT_EXPORT: 'report.export',
  AUDIT_VIEW: 'audit.view',
  JOB_VIEW: 'job.view',
} as const;

export type PermCode = (typeof Perm)[keyof typeof Perm] | (string & {});

/** OWNER có toàn quyền; các role khác kiểm tra theo danh sách permission. */
export function hasPermission(
  user: Pick<AuthUser, 'roles' | 'permissions'> | null | undefined,
  perm: PermCode,
) {
  if (!user) return false;
  if (user.roles.includes('OWNER')) return true;
  return user.permissions.includes(perm);
}

export function hasAnyPermission(
  user: Pick<AuthUser, 'roles' | 'permissions'> | null | undefined,
  perms: PermCode[],
) {
  return perms.some(p => hasPermission(user, p));
}
