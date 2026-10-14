import type { ComponentType, SVGProps } from 'react';
import {
  ArchiveBoxIcon,
  BanknotesIcon,
  BuildingOffice2Icon,
  BuildingStorefrontIcon,
  CalendarDaysIcon,
  ChartBarIcon,
  ClipboardDocumentListIcon,
  ClockIcon,
  Cog6ToothIcon,
  CreditCardIcon,
  CubeIcon,
  HomeIcon,
  IdentificationIcon,
  QrCodeIcon,
  ReceiptPercentIcon,
  ShieldCheckIcon,
  ShoppingCartIcon,
  TicketIcon,
  UserGroupIcon,
  UsersIcon,
} from '@heroicons/react/24/outline';
import { Perm, type PermCode } from '@/lib/permissions';

export type NavKey =
  | 'dashboard'
  | 'members'
  | 'subscriptions'
  | 'checkin'
  | 'classes'
  | 'pt'
  | 'finance'
  | 'pos'
  | 'inventory'
  | 'reports'
  | 'plans'
  | 'promotions'
  | 'products'
  | 'staff'
  | 'shifts'
  | 'branches'
  | 'rooms'
  | 'roles'
  | 'settings'
  | 'audit';

export interface NavItem {
  key: NavKey;
  href: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  /** Cần một trong các quyền này để thấy menu. Bỏ trống = ai đăng nhập cũng thấy. */
  anyOf?: PermCode[];
  /** false = module chưa làm → ẩn khỏi menu. */
  enabled: boolean;
}

export interface NavGroup {
  key: 'groupOperation' | 'groupCatalog' | 'groupSystem';
  items: NavItem[];
}

export const NAV_GROUPS: NavGroup[] = [
  {
    key: 'groupOperation',
    items: [
      { key: 'dashboard', href: '/admin/dashboard', icon: HomeIcon, enabled: true },
      {
        key: 'members',
        href: '/admin/members',
        icon: UserGroupIcon,
        anyOf: [Perm.MEMBER_VIEW],
        enabled: true,
      },
      {
        key: 'subscriptions',
        href: '/admin/subscriptions',
        icon: CreditCardIcon,
        anyOf: [Perm.SUBSCRIPTION_VIEW],
        enabled: true,
      },
      {
        key: 'checkin',
        href: '/admin/checkin',
        icon: QrCodeIcon,
        anyOf: [Perm.CHECKIN_VIEW, Perm.CHECKIN_SCAN],
        enabled: true,
      },
      {
        key: 'classes',
        href: '/admin/classes',
        icon: CalendarDaysIcon,
        anyOf: [Perm.CLASS_VIEW],
        enabled: true,
      },
      {
        key: 'pt',
        href: '/admin/pt',
        icon: IdentificationIcon,
        anyOf: [Perm.PT_VIEW],
        enabled: true,
      },
      {
        key: 'finance',
        href: '/admin/finance',
        icon: BanknotesIcon,
        anyOf: [Perm.INVOICE_VIEW],
        enabled: true,
      },
      {
        key: 'pos',
        href: '/admin/pos',
        icon: ShoppingCartIcon,
        anyOf: [Perm.POS_SELL],
        enabled: true,
      },
      {
        key: 'inventory',
        href: '/admin/inventory',
        icon: ArchiveBoxIcon,
        anyOf: [Perm.INVENTORY_VIEW],
        enabled: true,
      },
      {
        key: 'reports',
        href: '/admin/reports',
        icon: ChartBarIcon,
        anyOf: [Perm.REPORT_VIEW],
        enabled: true,
      },
    ],
  },
  {
    key: 'groupCatalog',
    items: [
      {
        key: 'plans',
        href: '/admin/plans',
        icon: TicketIcon,
        anyOf: [Perm.PLAN_VIEW],
        enabled: true,
      },
      {
        key: 'promotions',
        href: '/admin/promotions',
        icon: ReceiptPercentIcon,
        anyOf: [Perm.PROMOTION_VIEW],
        enabled: true,
      },
      {
        key: 'products',
        href: '/admin/products',
        icon: CubeIcon,
        anyOf: [Perm.PRODUCT_VIEW],
        enabled: true,
      },
    ],
  },
  {
    key: 'groupSystem',
    items: [
      {
        key: 'staff',
        href: '/admin/staff',
        icon: UsersIcon,
        anyOf: [Perm.STAFF_VIEW],
        enabled: true,
      },
      {
        key: 'shifts',
        href: '/admin/shifts',
        icon: ClockIcon,
        anyOf: [Perm.SHIFT_VIEW],
        enabled: true,
      },
      {
        key: 'branches',
        href: '/admin/branches',
        icon: BuildingStorefrontIcon,
        anyOf: [Perm.BRANCH_VIEW],
        enabled: true,
      },
      {
        key: 'rooms',
        href: '/admin/rooms',
        icon: BuildingOffice2Icon,
        anyOf: [Perm.ROOM_VIEW],
        enabled: true,
      },
      {
        key: 'roles',
        href: '/admin/roles',
        icon: ShieldCheckIcon,
        anyOf: [Perm.ROLE_VIEW],
        enabled: true,
      },
      {
        key: 'settings',
        href: '/admin/settings',
        icon: Cog6ToothIcon,
        anyOf: [Perm.SETTING_MANAGE],
        enabled: true,
      },
      {
        key: 'audit',
        href: '/admin/audit',
        icon: ClipboardDocumentListIcon,
        anyOf: [Perm.AUDIT_VIEW],
        enabled: true,
      },
    ],
  },
];
