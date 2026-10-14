'use client';

import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import {
  CheckCircleIcon,
  CreditCardIcon,
  DocumentTextIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import { LoadError } from '@/components/common/LoadError';
import { Logo } from '@/components/layout/Logo';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/lib/format';
import { useVerifyVNPayReturnQuery } from '../queries';

export function PaymentResultView() {
  const t = useTranslations('portal.paymentResult');
  const searchParams = useSearchParams();
  const queryString = searchParams.toString();

  const query = useVerifyVNPayReturnQuery(queryString, Boolean(queryString));

  const fallbackTxnRef = searchParams.get('vnp_TxnRef') ?? '';
  const fallbackAmount = Number(searchParams.get('vnp_Amount') ?? 0) / 100;
  const fallbackResponseCode = searchParams.get('vnp_ResponseCode') ?? '';
  const fallbackSuccess = fallbackResponseCode === '00';

  const data = query.data;
  const isSuccess = data ? data.success : fallbackSuccess;
  const txnRef = data?.txn_ref || fallbackTxnRef;
  const invoiceNo = data?.invoice_no || (data?.invoice_id ? `#${data.invoice_id}` : '—');
  const amount = data?.amount ?? fallbackAmount;
  const responseCode = data?.response_code || fallbackResponseCode || '—';

  return (
    <div className="flex min-h-dvh flex-col items-center justify-center bg-canvas px-4 py-12">
      <div className="mb-6">
        <Link href="/">
          <Logo />
        </Link>
      </div>

      <Card className="w-full max-w-lg gap-6 p-6 sm:p-8">
        {query.isLoading ? (
          <div className="grid justify-items-center gap-4 py-6 text-center">
            <Skeleton className="size-16 rounded-full" />
            <Skeleton className="h-7 w-64" />
            <Skeleton className="h-5 w-80 max-w-full" />
            <p className="text-body text-secondary">{t('verifying')}</p>
          </div>
        ) : query.isError && !fallbackTxnRef ? (
          <LoadError error={query.error} onRetry={() => void query.refetch()} />
        ) : (
          <>
            <div className="grid justify-items-center gap-3 text-center">
              <div
                className={
                  isSuccess
                    ? 'flex size-16 items-center justify-center rounded-full bg-success-50 text-success'
                    : 'flex size-16 items-center justify-center rounded-full bg-danger-50 text-danger'
                }
              >
                {isSuccess ? (
                  <CheckCircleIcon className="size-10" />
                ) : (
                  <XCircleIcon className="size-10" />
                )}
              </div>
              <h1 className="text-h2 font-semibold text-primary">
                {isSuccess ? t('successTitle') : t('failedTitle')}
              </h1>
              <p className="text-body text-secondary">
                {data?.message || (isSuccess ? t('successDesc') : t('failedDesc'))}
              </p>
            </div>

            <div className="grid gap-3 rounded-2xl bg-canvas p-4 text-body">
              <div className="flex items-center justify-between gap-2">
                <span className="text-secondary">{t('txnRef')}</span>
                <span className="font-mono font-semibold text-primary">{txnRef || '—'}</span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-secondary">{t('invoiceNo')}</span>
                <span className="font-semibold text-primary">{invoiceNo}</span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-secondary">{t('amount')}</span>
                <span className="text-h4 font-semibold text-brand-600">
                  {formatCurrency(amount)}
                </span>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="text-secondary">{t('responseCode')}</span>
                <span className="font-mono text-secondary">{responseCode}</span>
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <Button asChild size="lg">
                <Link href="/me/subscriptions">
                  <CreditCardIcon />
                  {t('goToSubscriptions')}
                </Link>
              </Button>
              <Button variant="outline" asChild size="lg">
                <Link href="/me/invoices">
                  <DocumentTextIcon />
                  {t('goToInvoices')}
                </Link>
              </Button>
            </div>
          </>
        )}
      </Card>
    </div>
  );
}
