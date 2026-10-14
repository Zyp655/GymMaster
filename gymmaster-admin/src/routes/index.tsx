/**
 * Khai báo route của khu vực `/app/*` (theo mẫu strongbody_admin `src/routes/index.js`).
 * `path` tương đối với `/app`. `perm` (anyOf) phải khớp với `perm` của lá menu tương ứng.
 * Thiếu quyền → trang 403 (`Forbidden`), không phải 404.
 */
import { lazy } from 'react';
import type { ComponentType, LazyExoticComponent, ReactElement } from 'react';
import { PERM } from '@/constants/permissions';

export interface AppRoute {
  /** Đường dẫn dưới `/app`, ví dụ `branches` hoặc `members/:id`. */
  path: string;
  element: ReactElement;
  /** anyOf. Bỏ trống = mọi nhân viên đã đăng nhập. */
  perm?: string[];
}

const page = (C: LazyExoticComponent<ComponentType>) => <C />;

const Dashboard = lazy(() => import('./home/Dashboard'));
const Branches = lazy(() => import('./branches'));
const Roles = lazy(() => import('./roles'));
const Profile = lazy(() => import('./profile'));
const Notifications = lazy(() => import('./notifications'));
const Members = lazy(() => import('./members'));
const MemberDetail = lazy(() => import('./members/detail'));
const Checkin = lazy(() => import('./checkin'));
const Checkins = lazy(() => import('./checkins'));
const ClassSchedules = lazy(() => import('./classes'));
const ClassTypes = lazy(() => import('./class-types'));
const Staff = lazy(() => import('./staff'));
const Rooms = lazy(() => import('./rooms'));
const Shifts = lazy(() => import('./shifts'));
const Plans = lazy(() => import('./plans'));
const Promotions = lazy(() => import('./promotions'));
const Settings = lazy(() => import('./settings'));
const Subscriptions = lazy(() => import('./subscriptions'));
const SubscriptionFreezes = lazy(() => import('./subscription-freezes'));
const Invoices = lazy(() => import('./invoices'));
const InvoiceDetail = lazy(() => import('./invoices/detail'));
const Payments = lazy(() => import('./payments'));
const PtContracts = lazy(() => import('./pt/ContractsPage'));
const PtSessions = lazy(() => import('./pt/SessionsPage'));
const Commissions = lazy(() => import('./pt/CommissionsPage'));
const Pos = lazy(() => import('./pos'));
const PosOrders = lazy(() => import('./pos-orders'));
const Products = lazy(() => import('./products'));
const ProductCategories = lazy(() => import('./product-categories'));
const Inventory = lazy(() => import('./inventory'));
const Suppliers = lazy(() => import('./suppliers'));
const Reports = lazy(() => import('./reports'));
const AuditLogs = lazy(() => import('./audit-logs'));
const Jobs = lazy(() => import('./jobs'));

export const routes: AppRoute[] = [
  { path: 'dashboard', element: page(Dashboard) },
  { path: 'profile', element: page(Profile) },
  { path: 'notifications', element: page(Notifications) },

  // Hội viên
  { path: 'members', element: page(Members), perm: [PERM.MEMBER_VIEW] },
  { path: 'members/:id', element: page(MemberDetail), perm: [PERM.MEMBER_VIEW] },
  {
    path: 'subscriptions',
    element: page(Subscriptions),
    perm: [PERM.SUBSCRIPTION_VIEW, PERM.SUBSCRIPTION_SELL],
  },
  {
    path: 'subscription-freezes',
    element: page(SubscriptionFreezes),
    perm: [PERM.SUBSCRIPTION_VIEW, PERM.SUBSCRIPTION_APPROVE_FREEZE],
  },
  { path: 'checkin', element: page(Checkin), perm: [PERM.CHECKIN_SCAN] },
  { path: 'checkins', element: page(Checkins), perm: [PERM.CHECKIN_VIEW] },

  // Lớp & PT
  {
    path: 'classes',
    element: page(ClassSchedules),
    perm: [PERM.CLASS_VIEW, PERM.BOOKING_MANAGE],
  },
  { path: 'class-types', element: page(ClassTypes), perm: [PERM.CLASS_VIEW] },
  { path: 'pt-contracts', element: page(PtContracts), perm: [PERM.PT_VIEW, PERM.PT_SELL] },
  { path: 'pt-sessions', element: page(PtSessions), perm: [PERM.PT_VIEW, PERM.PT_SCHEDULE] },
  { path: 'commissions', element: page(Commissions), perm: [PERM.COMMISSION_VIEW] },

  // Tài chính
  { path: 'invoices', element: page(Invoices), perm: [PERM.INVOICE_VIEW] },
  { path: 'invoices/:id', element: page(InvoiceDetail), perm: [PERM.INVOICE_VIEW] },
  {
    path: 'payments',
    element: page(Payments),
    perm: [PERM.INVOICE_VIEW, PERM.INVOICE_COLLECT],
  },
  { path: 'promotions', element: page(Promotions), perm: [PERM.PROMOTION_VIEW] },

  // Bán hàng
  { path: 'pos', element: page(Pos), perm: [PERM.POS_SELL] },
  {
    path: 'pos-orders',
    element: page(PosOrders),
    perm: [PERM.POS_SELL, PERM.INVOICE_VIEW],
  },
  { path: 'products', element: page(Products), perm: [PERM.PRODUCT_VIEW] },
  {
    path: 'product-categories',
    element: page(ProductCategories),
    perm: [PERM.PRODUCT_VIEW],
  },
  { path: 'inventory', element: page(Inventory), perm: [PERM.INVENTORY_VIEW] },
  {
    path: 'suppliers',
    element: page(Suppliers),
    perm: [PERM.PRODUCT_VIEW, PERM.INVENTORY_VIEW, PERM.INVENTORY_IMPORT],
  },

  // Danh mục
  { path: 'plans', element: page(Plans), perm: [PERM.PLAN_VIEW] },

  // Nhân sự
  { path: 'staff', element: page(Staff), perm: [PERM.STAFF_VIEW] },
  { path: 'shifts', element: page(Shifts), perm: [PERM.SHIFT_VIEW, PERM.SHIFT_MANAGE] },

  // Báo cáo
  { path: 'reports', element: page(Reports), perm: [PERM.REPORT_VIEW] },

  // Hệ thống
  { path: 'branches', element: page(Branches), perm: [PERM.BRANCH_VIEW] },
  { path: 'rooms', element: page(Rooms), perm: [PERM.ROOM_VIEW] },
  { path: 'roles', element: page(Roles), perm: [PERM.ROLE_VIEW] },
  { path: 'settings', element: page(Settings), perm: [PERM.SETTING_MANAGE] },
  { path: 'audit-logs', element: page(AuditLogs), perm: [PERM.AUDIT_VIEW] },
  { path: 'jobs', element: page(Jobs), perm: [PERM.JOB_VIEW] },
];

export default routes;
