import type { ReactNode } from 'react';
import {
  BellOutlined,
  CalendarOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  DollarOutlined,
  ExclamationCircleOutlined,
  FieldTimeOutlined,
  GiftOutlined,
  InboxOutlined,
  LoginOutlined,
  PauseCircleOutlined,
  RiseOutlined,
  ScheduleOutlined,
  StarOutlined,
  StopOutlined,
  ToolOutlined,
  UserAddOutlined,
  UserDeleteOutlined,
} from '@ant-design/icons';
import './notifications.css';

interface TypeMeta {
  icon: ReactNode;
  /** Màu biểu tượng + nền nhạt. */
  fg: string;
  bg: string;
}

const META: Record<string, TypeMeta> = {
  SUBSCRIPTION_EXPIRING: { icon: <FieldTimeOutlined />, fg: '#B45309', bg: '#FEF3C7' },
  SUBSCRIPTION_EXPIRED: { icon: <ExclamationCircleOutlined />, fg: '#B91C1C', bg: '#FEE2E2' },
  FREEZE_REQUEST: { icon: <PauseCircleOutlined />, fg: '#0A3150', bg: '#E9EFF5' },
  FREEZE_RESULT: { icon: <PauseCircleOutlined />, fg: '#12805C', bg: '#E5F6F0' },
  BOOKING_PROMOTED: { icon: <RiseOutlined />, fg: '#12805C', bg: '#E5F6F0' },
  CLASS_CANCELLED: { icon: <StopOutlined />, fg: '#B91C1C', bg: '#FEE2E2' },
  PT_SCHEDULED: { icon: <ScheduleOutlined />, fg: '#1D4ED8', bg: '#DBEAFE' },
  PT_CANCELLED: { icon: <CloseCircleOutlined />, fg: '#B91C1C', bg: '#FEE2E2' },
  PT_NO_SHOW: { icon: <UserDeleteOutlined />, fg: '#B45309', bg: '#FEF3C7' },
  COMMISSION_PAID: { icon: <DollarOutlined />, fg: '#12805C', bg: '#E5F6F0' },
  LOW_STOCK: { icon: <InboxOutlined />, fg: '#C2410C', bg: '#FFF1E6' },
  PAYMENT_SUCCESS: { icon: <CheckCircleOutlined />, fg: '#12805C', bg: '#E5F6F0' },
  REVIEW_PROMPT: { icon: <StarOutlined />, fg: '#D97706', bg: '#FEF3C7' },
  MAINTENANCE_DUE: { icon: <ToolOutlined />, fg: '#C2410C', bg: '#FFF1E6' },
  BIRTHDAY: { icon: <GiftOutlined />, fg: '#9333EA', bg: '#F3E8FF' },
  LEAD_NEW: { icon: <UserAddOutlined />, fg: '#1D4ED8', bg: '#DBEAFE' },
  CHECKIN: { icon: <LoginOutlined />, fg: '#0F766E', bg: '#CCFBF1' },
  SYSTEM: { icon: <BellOutlined />, fg: '#4B5563', bg: '#F1F3F6' },
};

const FALLBACK: TypeMeta = { icon: <CalendarOutlined />, fg: '#4B5563', bg: '#F1F3F6' };

function notificationMeta(type?: string): TypeMeta {
  return (type && META[type]) || FALLBACK;
}

/** Ô tròn biểu tượng theo loại thông báo. */
export function NotificationIcon({ type, size = 34 }: { type?: string; size?: number }) {
  const m = notificationMeta(type);
  return (
    <span
      className="notif-icon"
      style={{ width: size, height: size, background: m.bg, color: m.fg, fontSize: size * 0.48 }}
    >
      {m.icon}
    </span>
  );
}
