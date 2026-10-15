/**
 * Cây menu tĩnh của sidebar (theo mẫu strongbody_admin `src/menu/index.js`).
 * Mỗi lá: `{ key: '/app/...', label, icon, perm?: string[] }` – `perm` là anyOf (có MỘT quyền là thấy).
 * Nhóm không còn lá nào sau khi lọc quyền sẽ bị ẩn. OWNER thấy tất cả.
 *
 * Thêm module mới: thêm lá vào nhóm phù hợp + khai báo route cùng `perm` trong `src/routes/index.tsx`.
 */
import type { ReactNode } from 'react';
import type { MenuProps } from 'antd';
import {
  AccountBookOutlined,
  AimOutlined,
  ApiOutlined,
  AppstoreOutlined,
  AuditOutlined,
  BankOutlined,
  BarChartOutlined,
  BookOutlined,
  CalendarOutlined,
  ClockCircleOutlined,
  CreditCardOutlined,
  CrownOutlined,
  DashboardOutlined,
  DollarOutlined,
  FieldTimeOutlined,
  FileDoneOutlined,
  FileProtectOutlined,
  FileTextOutlined,
  FilterOutlined,
  GiftOutlined,
  HistoryOutlined,
  HomeOutlined,
  IdcardOutlined,
  InboxOutlined,
  KeyOutlined,
  LoginOutlined,
  NotificationOutlined,
  PauseCircleOutlined,
  ProfileOutlined,
  SafetyCertificateOutlined,
  ScheduleOutlined,
  SettingOutlined,
  ShopOutlined,
  ShoppingCartOutlined,
  SolutionOutlined,
  StarOutlined,
  TagsOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  ToolOutlined,
  TrophyOutlined,
  TruckOutlined,
  UserOutlined,
  WalletOutlined,
} from '@ant-design/icons';
import { PERM } from '@/constants/permissions';

export interface MenuLeaf {
  key: string;
  label: string;
  icon?: ReactNode;
  /** anyOf; bỏ trống = mọi nhân viên đã đăng nhập đều thấy. */
  perm?: string[];
}

export interface MenuGroup {
  key: string;
  label: string;
  icon?: ReactNode;
  children: MenuLeaf[];
}

export type MenuNode = MenuLeaf | MenuGroup;

const isGroup = (node: MenuNode): node is MenuGroup => 'children' in node;

const menu: MenuNode[] = [
  { key: '/app/dashboard', label: 'Tổng quan', icon: <DashboardOutlined /> },
  {
    key: 'members',
    label: 'Hội viên',
    icon: <TeamOutlined />,
    children: [
      {
        key: '/app/members',
        label: 'Danh sách hội viên',
        icon: <UserOutlined />,
        perm: [PERM.MEMBER_VIEW],
      },
      {
        key: '/app/subscriptions',
        label: 'Đăng ký gói tập',
        icon: <IdcardOutlined />,
        perm: [PERM.SUBSCRIPTION_VIEW, PERM.SUBSCRIPTION_SELL],
      },
      {
        key: '/app/subscription-freezes',
        label: 'Duyệt bảo lưu',
        icon: <PauseCircleOutlined />,
        perm: [PERM.SUBSCRIPTION_VIEW, PERM.SUBSCRIPTION_APPROVE_FREEZE],
      },
      {
        key: '/app/contracts-passes',
        label: 'Hợp đồng & Vé ngày',
        icon: <FileDoneOutlined />,
        perm: [PERM.CONTRACT_MANAGE],
      },
      {
        key: '/app/checkin',
        label: 'Check-in tại quầy',
        icon: <LoginOutlined />,
        perm: [PERM.CHECKIN_SCAN],
      },
      {
        key: '/app/checkins',
        label: 'Lịch sử ra vào',
        icon: <HistoryOutlined />,
        perm: [PERM.CHECKIN_VIEW],
      },
      {
        key: '/app/lockers',
        label: 'Tủ đồ (Locker)',
        icon: <KeyOutlined />,
        perm: [PERM.LOCKER_VIEW, PERM.LOCKER_MANAGE],
      },
      {
        key: '/app/leads',
        label: 'Khách tiềm năng & CSKH',
        icon: <FilterOutlined />,
        perm: [PERM.LEAD_VIEW, PERM.LEAD_MANAGE, PERM.MEMBER_VIEW],
      },
      {
        key: '/app/loyalty',
        label: 'Tích điểm & Giới thiệu',
        icon: <CrownOutlined />,
        perm: [PERM.LOYALTY_VIEW, PERM.LOYALTY_MANAGE],
      },
      {
        key: '/app/engagement',
        label: 'Bản tin & Khảo sát NPS',
        icon: <NotificationOutlined />,
        perm: [PERM.ENGAGEMENT_VIEW, PERM.ENGAGEMENT_MANAGE],
      },
    ],
  },
  {
    key: 'classes',
    label: 'Lớp & PT',
    icon: <ThunderboltOutlined />,
    children: [
      {
        key: '/app/classes',
        label: 'Lịch lớp',
        icon: <CalendarOutlined />,
        perm: [PERM.CLASS_VIEW, PERM.BOOKING_MANAGE],
      },
      {
        key: '/app/class-types',
        label: 'Loại lớp',
        icon: <AppstoreOutlined />,
        perm: [PERM.CLASS_VIEW],
      },
      {
        key: '/app/pt-contracts',
        label: 'Hợp đồng PT',
        icon: <SolutionOutlined />,
        perm: [PERM.PT_VIEW, PERM.PT_SELL],
      },
      {
        key: '/app/pt-sessions',
        label: 'Lịch PT',
        icon: <ScheduleOutlined />,
        perm: [PERM.PT_VIEW, PERM.PT_SCHEDULE],
      },
      {
        key: '/app/workout-plans',
        label: 'Giáo án tập luyện',
        icon: <ProfileOutlined />,
        perm: [PERM.WORKOUT_VIEW, PERM.WORKOUT_MANAGE],
      },
      {
        key: '/app/exercises',
        label: 'Thư viện bài tập',
        icon: <BookOutlined />,
        perm: [PERM.WORKOUT_VIEW, PERM.WORKOUT_MANAGE],
      },
      {
        key: '/app/reviews',
        label: 'Đánh giá HLV & lớp',
        icon: <StarOutlined />,
        perm: [PERM.REVIEW_VIEW],
      },
      {
        key: '/app/commissions',
        label: 'Hoa hồng HLV',
        icon: <TrophyOutlined />,
        perm: [PERM.COMMISSION_VIEW],
      },
    ],
  },
  {
    key: 'finance',
    label: 'Tài chính',
    icon: <WalletOutlined />,
    children: [
      {
        key: '/app/invoices',
        label: 'Hóa đơn',
        icon: <FileTextOutlined />,
        perm: [PERM.INVOICE_VIEW],
      },
      {
        key: '/app/payments',
        label: 'Sổ thu tiền',
        icon: <CreditCardOutlined />,
        perm: [PERM.INVOICE_VIEW, PERM.INVOICE_COLLECT],
      },
      {
        key: '/app/finance',
        label: 'Chi phí, Quỹ ca & P&L',
        icon: <AccountBookOutlined />,
        perm: [PERM.EXPENSE_VIEW, PERM.EXPENSE_MANAGE, PERM.CASH_SESSION_MANAGE],
      },
      {
        key: '/app/promotions',
        label: 'Khuyến mãi',
        icon: <GiftOutlined />,
        perm: [PERM.PROMOTION_VIEW],
      },
    ],
  },
  {
    key: 'sales',
    label: 'Bán hàng',
    icon: <ShopOutlined />,
    children: [
      { key: '/app/pos', label: 'POS', icon: <ShoppingCartOutlined />, perm: [PERM.POS_SELL] },
      {
        key: '/app/pos-orders',
        label: 'Đơn hàng POS',
        icon: <FileTextOutlined />,
        perm: [PERM.POS_SELL, PERM.INVOICE_VIEW],
      },
      {
        key: '/app/products',
        label: 'Sản phẩm',
        icon: <TagsOutlined />,
        perm: [PERM.PRODUCT_VIEW],
      },
      {
        key: '/app/product-categories',
        label: 'Danh mục sản phẩm',
        icon: <AppstoreOutlined />,
        perm: [PERM.PRODUCT_VIEW],
      },
      {
        key: '/app/inventory',
        label: 'Kho hàng',
        icon: <InboxOutlined />,
        perm: [PERM.INVENTORY_VIEW],
      },
      {
        key: '/app/suppliers',
        label: 'Nhà cung cấp',
        icon: <TruckOutlined />,
        perm: [PERM.PRODUCT_VIEW, PERM.INVENTORY_VIEW, PERM.INVENTORY_IMPORT],
      },
    ],
  },
  {
    key: 'catalog',
    label: 'Danh mục',
    icon: <ProfileOutlined />,
    children: [
      {
        key: '/app/plans',
        label: 'Gói tập & gói PT',
        icon: <DollarOutlined />,
        perm: [PERM.PLAN_VIEW],
      },
    ],
  },
  {
    key: 'hr',
    label: 'Nhân sự',
    icon: <IdcardOutlined />,
    children: [
      {
        key: '/app/staff',
        label: 'Nhân viên & HLV',
        icon: <TeamOutlined />,
        perm: [PERM.STAFF_VIEW],
      },
      {
        key: '/app/shifts',
        label: 'Ca làm việc',
        icon: <ClockCircleOutlined />,
        perm: [PERM.SHIFT_VIEW, PERM.SHIFT_MANAGE],
      },
      {
        key: '/app/payrolls',
        label: 'Bảng lương & Chốt công',
        icon: <DollarOutlined />,
        perm: [PERM.PAYROLL_VIEW, PERM.PAYROLL_MANAGE],
      },
      {
        key: '/app/trainer-schedules',
        label: 'Lịch rảnh HLV & Nghỉ phép',
        icon: <ScheduleOutlined />,
        perm: [PERM.LEAVE_MANAGE, PERM.HOLIDAY_MANAGE],
      },
    ],
  },
  { key: '/app/reports', label: 'Báo cáo', icon: <BarChartOutlined />, perm: [PERM.REPORT_VIEW] },
  {
    key: '/app/kpi-analytics',
    label: 'KPI, Cohort & Sức chứa',
    icon: <AimOutlined />,
    perm: [PERM.KPI_MANAGE, PERM.REPORT_VIEW],
  },
  {
    key: 'system',
    label: 'Hệ thống',
    icon: <SettingOutlined />,
    children: [
      {
        key: '/app/branches',
        label: 'Chi nhánh',
        icon: <BankOutlined />,
        perm: [PERM.BRANCH_VIEW],
      },
      { key: '/app/rooms', label: 'Phòng tập', icon: <HomeOutlined />, perm: [PERM.ROOM_VIEW] },
      {
        key: '/app/equipments',
        label: 'Thiết bị & bảo trì',
        icon: <ToolOutlined />,
        perm: [PERM.EQUIPMENT_VIEW, PERM.EQUIPMENT_MANAGE],
      },
      {
        key: '/app/access-devices',
        label: 'Cổng từ & Khuôn mặt',
        icon: <ApiOutlined />,
        perm: [PERM.DEVICE_VIEW, PERM.DEVICE_MANAGE],
      },
      {
        key: '/app/roles',
        label: 'Vai trò & quyền',
        icon: <SafetyCertificateOutlined />,
        perm: [PERM.ROLE_VIEW],
      },
      {
        key: '/app/settings',
        label: 'Cấu hình',
        icon: <SettingOutlined />,
        perm: [PERM.SETTING_MANAGE],
      },
      {
        key: '/app/security-privacy',
        label: 'Bảo mật 2FA & Dữ liệu',
        icon: <FileProtectOutlined />,
        perm: [PERM.PRIVACY_MANAGE, PERM.MEMBER_VIEW_SENSITIVE, PERM.SETTING_MANAGE],
      },
      {
        key: '/app/cms-marketing',
        label: 'Nội dung Landing & Tin nhắn PT',
        icon: <NotificationOutlined />,
        perm: [PERM.CMS_MANAGE, PERM.PT_VIEW, PERM.LEAD_VIEW],
      },
      {
        key: '/app/system-tools',
        label: 'Thiết lập & Chuyển đổi dữ liệu',
        icon: <ToolOutlined />,
        perm: [PERM.IMPORT_LEGACY, PERM.SETUP_WIZARD, PERM.SETTING_MANAGE, PERM.MEMBER_IMPORT],
      },
      {
        key: '/app/audit-logs',
        label: 'Nhật ký thao tác',
        icon: <AuditOutlined />,
        perm: [PERM.AUDIT_VIEW],
      },
      {
        key: '/app/jobs',
        label: 'Job định kỳ',
        icon: <FieldTimeOutlined />,
        perm: [PERM.JOB_VIEW],
      },
    ],
  },
];

type MenuItems = NonNullable<MenuProps['items']>;

/** Lọc menu theo quyền và chuyển sang `items` của antd Menu. */
export function buildMenuItems(
  can: (perm?: string[]) => boolean,
  nodes: MenuNode[] = menu,
): MenuItems {
  return nodes.flatMap<MenuItems[number]>((node) => {
    if (isGroup(node)) {
      const children = node.children.filter((leaf) => can(leaf.perm));
      if (children.length === 0) return [];
      return [
        {
          key: node.key,
          label: node.label,
          icon: node.icon,
          children: children.map(({ key, label, icon }) => ({ key, label, icon })),
        },
      ];
    }
    return can(node.perm) ? [{ key: node.key, label: node.label, icon: node.icon }] : [];
  });
}

/** Tìm lá menu khớp đường dẫn (khớp tiền tố dài nhất) và nhóm cha của nó. */
export function findMenuTrail(pathname: string): { leaf?: MenuLeaf; group?: MenuGroup } {
  let best: { leaf?: MenuLeaf; group?: MenuGroup } = {};
  let bestLen = 0;
  const check = (leaf: MenuLeaf, group?: MenuGroup) => {
    if (
      (pathname === leaf.key || pathname.startsWith(`${leaf.key}/`)) &&
      leaf.key.length > bestLen
    ) {
      best = { leaf, group };
      bestLen = leaf.key.length;
    }
  };
  menu.forEach((node) => {
    if (isGroup(node)) node.children.forEach((leaf) => check(leaf, node));
    else check(node);
  });
  return best;
}

export default menu;
