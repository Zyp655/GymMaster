'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { DocumentTextIcon, EyeIcon } from '@heroicons/react/24/outline';
import { EmptyState } from '@/components/common/EmptyState';
import { LoadError } from '@/components/common/LoadError';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { formatCurrency, formatDateTime } from '@/lib/format';
import { looseT } from '@/lib/utils';
import { useInvoiceDetailQuery, useInvoicesQuery } from '../queries';
import { PaginationControls } from './PaginationControls';
import { PortalPageHeader } from './PortalPageHeader';
import { InvoiceStatusBadge } from './PortalStatusBadge';

const PAGE_LIMIT = 10;

export function InvoicesView() {
  const t = useTranslations('portal.invoices');
  const [page, setPage] = useState(1);
  const [selectedInvoiceId, setSelectedInvoiceId] = useState<number | null>(null);

  const query = useInvoicesQuery(page, PAGE_LIMIT);

  return (
    <div className="grid gap-6">
      <PortalPageHeader title={t('title')} subtitle={t('subtitle')} />

      <Card>
        {query.isLoading ? (
          <div className="grid gap-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : query.isError || !query.data ? (
          <LoadError error={query.error} onRetry={() => void query.refetch()} />
        ) : query.data.items.length === 0 ? (
          <EmptyState
            icon={DocumentTextIcon}
            title={t('emptyTitle')}
            description={t('emptyDesc')}
          />
        ) : (
          <div className="grid gap-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('colInvoiceNo')}</TableHead>
                  <TableHead>{t('colIssuedAt')}</TableHead>
                  <TableHead>{t('colBranch')}</TableHead>
                  <TableHead>{t('colTotal')}</TableHead>
                  <TableHead>{t('colPaid')}</TableHead>
                  <TableHead>{t('colDue')}</TableHead>
                  <TableHead>{t('colStatus')}</TableHead>
                  <TableHead className="text-right">{t('colAction')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {query.data.items.map(invoice => (
                  <TableRow key={invoice.id}>
                    <TableCell className="font-semibold text-primary">
                      {invoice.invoice_no}
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-secondary">
                      {formatDateTime(invoice.issued_at)}
                    </TableCell>
                    <TableCell>{invoice.branch_name}</TableCell>
                    <TableCell className="font-semibold">
                      {formatCurrency(invoice.total)}
                    </TableCell>
                    <TableCell className="text-secondary">
                      {formatCurrency(invoice.paid_amount)}
                    </TableCell>
                    <TableCell
                      className={
                        invoice.due > 0 ? 'font-semibold text-danger' : 'text-secondary'
                      }
                    >
                      {formatCurrency(invoice.due)}
                    </TableCell>
                    <TableCell>
                      <InvoiceStatusBadge status={invoice.status} />
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedInvoiceId(invoice.id)}
                      >
                        <EyeIcon />
                        {t('viewDetail')}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>

            <PaginationControls
              page={query.data.current_page}
              totalPage={query.data.total_page}
              total={query.data.total}
              onPageChange={setPage}
            />
          </div>
        )}
      </Card>

      <InvoiceDetailDialog
        invoiceId={selectedInvoiceId}
        onOpenChange={open => !open && setSelectedInvoiceId(null)}
      />
    </div>
  );
}

function InvoiceDetailDialog({
  invoiceId,
  onOpenChange,
}: {
  invoiceId: number | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations('portal.invoices.detail');
  const tc = useTranslations('common');
  const ti = looseT(useTranslations('portal.invoices.detail.itemTypes'));
  const tm = looseT(useTranslations('portal.invoices.detail.paymentMethods'));

  const detailQuery = useInvoiceDetailQuery(invoiceId);
  const detail = detailQuery.data;

  const translateItemType = (type: string) => {
    try {
      return ti(type);
    } catch {
      return type;
    }
  };

  const translateMethod = (method: string) => {
    try {
      return tm(method);
    } catch {
      return method;
    }
  };

  return (
    <Dialog open={invoiceId !== null} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl" closeLabel={tc('actions.close')}>
        <DialogHeader>
          <div className="flex flex-wrap items-center gap-2">
            <DialogTitle>{t('title', { no: detail?.invoice_no ?? invoiceId ?? '' })}</DialogTitle>
            {detail ? <InvoiceStatusBadge status={detail.status} /> : null}
          </div>
          {detail ? (
            <DialogDescription>
              {t('issuedAt', {
                date: formatDateTime(detail.issued_at),
                branch: detail.branch_name,
              })}
            </DialogDescription>
          ) : null}
        </DialogHeader>

        {detailQuery.isLoading ? (
          <div className="grid gap-3">
            <Skeleton className="h-24 w-full rounded-xl" />
            <Skeleton className="h-32 w-full rounded-xl" />
          </div>
        ) : detailQuery.isError || !detail ? (
          <LoadError error={detailQuery.error} onRetry={() => void detailQuery.refetch()} />
        ) : (
          <div className="grid gap-5">
            <div className="grid gap-2.5 rounded-xl bg-canvas p-4 sm:grid-cols-3">
              <div>
                <p className="text-caption text-secondary">{t('subtotal')}</p>
                <p className="text-body font-medium text-primary">
                  {formatCurrency(detail.subtotal)}
                </p>
              </div>
              <div>
                <p className="text-caption text-secondary">{t('discount')}</p>
                <p className="text-body font-medium text-primary">
                  {formatCurrency(detail.discount)}
                  {detail.promotion_code ? ` (${detail.promotion_code})` : ''}
                </p>
              </div>
              <div>
                <p className="text-caption text-secondary">{t('total')}</p>
                <p className="text-body font-semibold text-primary">
                  {formatCurrency(detail.total)}
                </p>
              </div>
              <div>
                <p className="text-caption text-secondary">{t('paid')}</p>
                <p className="text-body font-medium text-success">
                  {formatCurrency(detail.paid_amount)}
                </p>
              </div>
              <div>
                <p className="text-caption text-secondary">{t('refunded')}</p>
                <p className="text-body font-medium text-secondary">
                  {formatCurrency(detail.refunded_amount)}
                </p>
              </div>
              <div>
                <p className="text-caption text-secondary">{t('due')}</p>
                <p
                  className={
                    detail.due > 0
                      ? 'text-body font-semibold text-danger'
                      : 'text-body font-medium text-primary'
                  }
                >
                  {formatCurrency(detail.due)}
                </p>
              </div>
            </div>

            <div className="grid gap-2">
              <h4 className="text-body font-semibold text-primary">{t('itemsTitle')}</h4>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('colItemDesc')}</TableHead>
                    <TableHead>{t('colItemType')}</TableHead>
                    <TableHead>{t('colQty')}</TableHead>
                    <TableHead>{t('colUnitPrice')}</TableHead>
                    <TableHead>{t('colItemDiscount')}</TableHead>
                    <TableHead className="text-right">{t('colAmount')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {detail.items.map(item => (
                    <TableRow key={item.id}>
                      <TableCell className="font-medium">{item.description}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{translateItemType(item.item_type)}</Badge>
                      </TableCell>
                      <TableCell>{item.quantity}</TableCell>
                      <TableCell>{formatCurrency(item.unit_price)}</TableCell>
                      <TableCell>{formatCurrency(item.discount)}</TableCell>
                      <TableCell className="text-right font-semibold">
                        {formatCurrency(item.amount)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-2">
                <h4 className="text-body font-semibold text-primary">{t('paymentsTitle')}</h4>
                {detail.payments.length === 0 ? (
                  <p className="text-body text-secondary">{t('noPayments')}</p>
                ) : (
                  <div className="grid gap-2">
                    {detail.payments.map(payment => (
                      <div
                        key={payment.id}
                        className="flex items-center justify-between gap-2 rounded-xl bg-canvas p-3"
                      >
                        <div>
                          <p className="text-body font-semibold text-success">
                            +{formatCurrency(payment.amount)}
                          </p>
                          <p className="text-caption text-secondary">
                            {t('paymentMethod', { method: translateMethod(payment.method) })}
                          </p>
                        </div>
                        <span className="text-caption text-tertiary">
                          {formatDateTime(payment.paid_at)}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="grid gap-2">
                <h4 className="text-body font-semibold text-primary">{t('refundsTitle')}</h4>
                {detail.refunds.length === 0 ? (
                  <p className="text-body text-secondary">{t('noRefunds')}</p>
                ) : (
                  <div className="grid gap-2">
                    {detail.refunds.map(refund => (
                      <div
                        key={refund.id}
                        className="flex items-start justify-between gap-2 rounded-xl bg-canvas p-3"
                      >
                        <div>
                          <p className="text-body font-semibold text-warning">
                            -{formatCurrency(refund.amount)}
                          </p>
                          <p className="text-caption text-secondary">
                            {t('paymentMethod', { method: translateMethod(refund.method) })}
                          </p>
                          {refund.reason ? (
                            <p className="text-caption text-secondary">
                              {t('refundReason', { reason: refund.reason })}
                            </p>
                          ) : null}
                        </div>
                        <span className="text-caption text-tertiary">
                          {formatDateTime(refund.created_at)}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {tc('actions.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
