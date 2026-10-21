import type { Metadata } from 'next';
import { Be_Vietnam_Pro } from 'next/font/google';
import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import { AppProviders } from '@/components/providers/AppProviders';
import { getSiteUrl } from '@/env';
import { getCachedSession } from '@/guards/getCachedSession';
import { LOCALE, TIME_ZONE } from '@/i18n/request';
import '@/styles/globals.css';

const beVietnam = Be_Vietnam_Pro({
  subsets: ['latin', 'vietnamese'],
  weight: ['400', '500', '600', '700'],
  variable: '--font-be-vietnam',
  display: 'swap',
});

export const metadata: Metadata = {
  // Base cho canonical/og:url tương đối của các trang con.
  metadataBase: new URL(getSiteUrl()),
  // Trang con chỉ đặt tiêu đề ngắn (KHÔNG tự thêm "| GymMaster") – template tự nối hậu tố.
  title: { default: 'GymMaster', template: '%s · GymMaster' },
  openGraph: { siteName: 'GymMaster', locale: 'vi_VN', type: 'website' },
  description: 'Hệ thống số hóa quy trình quản lý trung tâm Fitness',
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const [session, messages] = await Promise.all([getCachedSession(), getMessages()]);
  return (
    <html lang={LOCALE} className={beVietnam.variable} suppressHydrationWarning>
      <body suppressHydrationWarning>
        <NextIntlClientProvider locale={LOCALE} timeZone={TIME_ZONE} messages={messages}>
          <AppProviders session={session}>{children}</AppProviders>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
