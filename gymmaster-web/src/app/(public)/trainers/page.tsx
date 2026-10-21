import type { Metadata } from 'next';
import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { getCachedSession } from '@/guards/getCachedSession';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { landingApi } from '@/features/landing/api';
import { LandingFooter } from '@/features/landing/components/LandingFooter';
import { LandingHeader } from '@/features/landing/components/LandingHeader';
import { TrainerRating } from '@/features/landing/components/TrainerRating';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('landing.trainers');
  return {
    title: t('metaTitle'),
    description: t('metaDescription'),
    alternates: { canonical: '/trainers' },
    openGraph: {
      title: t('metaTitle'),
      description: t('metaDescription'),
      url: '/trainers',
      type: 'website',
    },
  };
}

export default async function PublicTrainersPage() {
  const [t, session, trainersRes] = await Promise.all([
    getTranslations('landing.trainers'),
    getCachedSession(),
    landingApi.trainers(),
  ]);
  const trainers = trainersRes.ok ? trainersRes.data : [];

  return (
    <div className="flex min-h-dvh flex-col bg-canvas">
      <LandingHeader isAuthenticated={!!session && !session.error} />
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-12 sm:px-6">
        <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
          <div className="grid gap-2">
            <Badge variant="brand" className="w-fit">
              {t('eyebrow')}
            </Badge>
            <h1 className="text-h1 font-semibold text-primary">{t('title')}</h1>
            <p className="max-w-2xl text-body text-secondary">{t('subtitle')}</p>
          </div>
          <Button asChild>
            <Link href="/#tap-thu">{t('trialCta')}</Link>
          </Button>
        </div>

        {!trainersRes.ok ? (
          <Card className="p-8 text-center text-body text-secondary">{t('loadError')}</Card>
        ) : trainers.length === 0 ? (
          <Card className="p-8 text-center text-body text-secondary">{t('empty')}</Card>
        ) : (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {trainers.map(tr => (
              <Card key={tr.id} className="flex flex-col justify-between gap-6 p-6">
                <div className="grid gap-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <h2 className="text-h3 font-semibold break-words text-primary">
                        {tr.full_name}
                      </h2>
                      <p className="mt-0.5 text-body font-medium text-brand-600">{tr.headline}</p>
                    </div>
                    <TrainerRating trainer={tr} />
                  </div>
                  <p className="text-body text-secondary">{tr.bio}</p>
                </div>
                <dl className="grid gap-1.5 border-t border-line pt-4 text-body text-secondary">
                  <div>
                    <dt className="inline font-semibold text-primary">{t('specialtiesLabel')}</dt>{' '}
                    <dd className="inline">{tr.specialties || t('notUpdated')}</dd>
                  </div>
                  <div>
                    <dt className="inline font-semibold text-primary">
                      {t('certificationsLabel')}
                    </dt>{' '}
                    <dd className="inline">{tr.certifications || t('notUpdated')}</dd>
                  </div>
                  <div>
                    <dt className="inline font-semibold text-primary">{t('experienceLabel')}</dt>{' '}
                    <dd className="inline">
                      {t('experienceValue', { years: tr.experience_years })}
                    </dd>
                  </div>
                </dl>
              </Card>
            ))}
          </div>
        )}
      </main>
      <LandingFooter />
    </div>
  );
}
