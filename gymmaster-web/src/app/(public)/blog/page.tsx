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
import { PostCard } from '@/features/landing/components/PostCard';

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations('landing.blog');
  return {
    title: t('metaTitle'),
    description: t('metaDescription'),
    alternates: { canonical: '/blog' },
    openGraph: {
      title: t('metaTitle'),
      description: t('metaDescription'),
      url: '/blog',
      type: 'website',
    },
  };
}

export default async function PublicBlogPage() {
  const [t, session, postsRes] = await Promise.all([
    getTranslations('landing.blog'),
    getCachedSession(),
    landingApi.posts(),
  ]);
  const posts = postsRes.ok ? postsRes.data : [];

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
          <Button variant="outline" asChild>
            <Link href="/">{t('backHome')}</Link>
          </Button>
        </div>

        {!postsRes.ok ? (
          <Card className="p-8 text-center text-body text-secondary">{t('loadError')}</Card>
        ) : posts.length === 0 ? (
          <Card className="p-8 text-center text-body text-secondary">{t('empty')}</Card>
        ) : (
          <div className="grid gap-6 md:grid-cols-2">
            {posts.map(post => (
              <PostCard key={post.id} post={post} />
            ))}
          </div>
        )}
      </main>
      <LandingFooter />
    </div>
  );
}
