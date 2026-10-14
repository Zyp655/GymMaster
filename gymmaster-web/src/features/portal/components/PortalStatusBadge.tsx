'use client';

import { useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import type {
  BookingStatus,
  FreezeStatus,
  InvoiceStatus,
  Member,
  PTContractStatus,
  PTSessionStatus,
  SubscriptionStatus,
} from '../types';

type BadgeVariant = 'neutral' | 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'outline';

const memberVariant: Record<Member['status'], BadgeVariant> = {
  ACTIVE: 'success',
  INACTIVE: 'neutral',
  BANNED: 'danger',
};

const subscriptionVariant: Record<SubscriptionStatus, BadgeVariant> = {
  ACTIVE: 'success',
  PENDING: 'info',
  FROZEN: 'warning',
  EXPIRED: 'neutral',
  CANCELLED: 'danger',
  TRANSFERRED: 'outline',
};

const freezeVariant: Record<FreezeStatus, BadgeVariant> = {
  PENDING: 'warning',
  APPROVED: 'success',
  REJECTED: 'danger',
  ENDED: 'neutral',
  CANCELLED: 'neutral',
};

const bookingVariant: Record<BookingStatus, BadgeVariant> = {
  BOOKED: 'success',
  WAITLIST: 'warning',
  ATTENDED: 'info',
  CANCELLED: 'neutral',
  NO_SHOW: 'danger',
};

const ptContractVariant: Record<PTContractStatus, BadgeVariant> = {
  ACTIVE: 'success',
  PENDING: 'info',
  COMPLETED: 'brand',
  EXPIRED: 'neutral',
  CANCELLED: 'danger',
};

const ptSessionVariant: Record<PTSessionStatus, BadgeVariant> = {
  SCHEDULED: 'info',
  COMPLETED: 'success',
  CANCELLED: 'neutral',
  NO_SHOW: 'danger',
};

const invoiceVariant: Record<InvoiceStatus, BadgeVariant> = {
  PAID: 'success',
  PARTIAL: 'warning',
  UNPAID: 'danger',
  REFUNDED: 'info',
  VOID: 'neutral',
};

export function MemberStatusBadge({ status }: { status: Member['status'] }) {
  const t = useTranslations('portal.status.member');
  return <Badge variant={memberVariant[status]}>{t(status)}</Badge>;
}

export function SubscriptionStatusBadge({ status }: { status: SubscriptionStatus }) {
  const t = useTranslations('portal.status.subscription');
  return <Badge variant={subscriptionVariant[status]}>{t(status)}</Badge>;
}

export function FreezeStatusBadge({ status }: { status: FreezeStatus }) {
  const t = useTranslations('portal.status.freeze');
  return <Badge variant={freezeVariant[status]}>{t(status)}</Badge>;
}

export function BookingStatusBadge({ status }: { status: BookingStatus }) {
  const t = useTranslations('portal.status.booking');
  return <Badge variant={bookingVariant[status]}>{t(status)}</Badge>;
}

export function PTContractStatusBadge({ status }: { status: PTContractStatus }) {
  const t = useTranslations('portal.status.ptContract');
  return <Badge variant={ptContractVariant[status]}>{t(status)}</Badge>;
}

export function PTSessionStatusBadge({ status }: { status: PTSessionStatus }) {
  const t = useTranslations('portal.status.ptSession');
  return <Badge variant={ptSessionVariant[status]}>{t(status)}</Badge>;
}

export function InvoiceStatusBadge({ status }: { status: InvoiceStatus }) {
  const t = useTranslations('portal.status.invoice');
  return <Badge variant={invoiceVariant[status]}>{t(status)}</Badge>;
}
