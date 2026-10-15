import type { Metadata } from 'next';
import { getTranslations } from 'next-intl/server';
import { getCachedSession } from '@/guards/getCachedSession';
import { landingApi } from '@/features/landing/api';
import { BranchesSection } from '@/features/landing/components/BranchesSection';
import { CommunitySection } from '@/features/landing/components/CommunitySection';
import { Features } from '@/features/landing/components/Features';
import { Hero } from '@/features/landing/components/Hero';
import { LandingFooter } from '@/features/landing/components/LandingFooter';
import { LandingHeader } from '@/features/landing/components/LandingHeader';
import { PricingSection } from '@/features/landing/components/PricingSection';
import { PTSection } from '@/features/landing/components/PTSection';
import { PublicScheduleSection } from '@/features/landing/components/PublicScheduleSection';
import { TrialLeadSection } from '@/features/landing/components/TrialLeadSection';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('common.landing');
  return { title: t('metaTitle'), description: t('metaDescription') };
}

/** Trang công khai: bảng giá, gói PT, lịch lớp, HLV, cảm nhận, blog, chi nhánh, đăng ký tập thử. */
export default async function LandingPage() {
  const [session, plans, branches, schedules, trainers, testimonials, posts] = await Promise.all([
    getCachedSession(),
    landingApi.plans(),
    landingApi.branches(),
    landingApi.classSchedules(),
    landingApi.trainers(),
    landingApi.testimonials(),
    landingApi.posts(),
  ]);

  const planList = plans.ok ? plans.data.membership_plans : [];
  const ptList = plans.ok ? plans.data.pt_packages : [];
  const branchList = branches.ok ? branches.data : [];
  const scheduleList = schedules.ok ? schedules.data : [];
  const trainerList = trainers.ok ? trainers.data : [];
  const testimonialList = testimonials.ok ? testimonials.data : [];
  const postList = posts.ok ? posts.data : [];
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
        <PublicScheduleSection branches={branchList} initialSchedules={scheduleList} />
        <CommunitySection
          trainers={trainerList}
          testimonials={testimonialList}
          posts={postList}
        />
        <BranchesSection result={branches} />
        <TrialLeadSection branches={branchList} plans={planList} />
      </main>
      <LandingFooter />
    </div>
  );
}
