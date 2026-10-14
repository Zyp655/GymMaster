import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { getCachedSession } from '@/guards/getCachedSession';
import { landingApi } from '@/features/landing/api';
import { BranchesSection } from '@/features/landing/components/BranchesSection';
import { Features } from '@/features/landing/components/Features';
import { Hero } from '@/features/landing/components/Hero';
import { LandingFooter } from '@/features/landing/components/LandingFooter';
import { LandingHeader } from '@/features/landing/components/LandingHeader';
import { PricingSection } from '@/features/landing/components/PricingSection';
import { PTSection } from '@/features/landing/components/PTSection';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('common.landing');
  return { title: t('metaTitle'), description: t('metaDescription') };
}

/** Trang công khai: bảng giá, gói PT, chi nhánh. Dữ liệu lấy phía server mỗi request (no-store). */
export default async function LandingPage() {
  const [session, plans, branches] = await Promise.all([
    getCachedSession(),
    landingApi.plans(),
    landingApi.branches(),
  ]);

  const planList = plans.ok ? plans.data.membership_plans : [];
  const ptList = plans.ok ? plans.data.pt_packages : [];
  const branchList = branches.ok ? branches.data : [];
  const hours =
    branchList.length > 0
      ? {
          open: branchList.map(b => b.open_time).sort()[0],
          close: branchList
            .map(b => b.close_time)
            .sort()
            .reverse()[0],
        }
      : null;

  return (
    <div className="flex min-h-dvh flex-col bg-canvas">
      <LandingHeader isAuthenticated={!!session && !session.error} />
      <main className="flex-1">
        <Hero
          branchCount={branchList.length}
          planCount={planList.length}
          ptCount={ptList.length}
          hours={hours}
        />
        <Features />
        <PricingSection result={plans.ok ? { ok: true, data: planList } : { ok: false }} />
        <PTSection result={plans.ok ? { ok: true, data: ptList } : { ok: false }} />
        <BranchesSection result={branches} />
      </main>
      <LandingFooter />
    </div>
  );
}
