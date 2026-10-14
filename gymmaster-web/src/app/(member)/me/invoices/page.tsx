import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { InvoicesView } from '@/features/portal/components/InvoicesView';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('portal.invoices');
  return { title: t('title') };
}

export default function MemberInvoicesPage() {
  return <InvoicesView />;
}
