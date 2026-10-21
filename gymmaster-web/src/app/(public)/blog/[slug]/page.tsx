import { cache } from 'react';
import type { Metadata } from 'next';
import Link from 'next/link';
import { notFound } from 'next/navigation';
import { getTranslations } from 'next-intl/server';
import { ArrowLeftIcon } from '@heroicons/react/20/solid';
import { getCachedSession } from '@/guards/getCachedSession';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { RichHtml } from '@/components/ui/rich-html';
import { landingApi } from '@/features/landing/api';
import { LandingFooter } from '@/features/landing/components/LandingFooter';
import { LandingHeader } from '@/features/landing/components/LandingHeader';
import { postHref } from '@/features/landing/post-slug';
import { postCategoryLabel } from '@/features/landing/types';
import { formatDate } from '@/lib/format';
import { stripTags } from '@/lib/sanitize-html';

type Params = { params: Promise<{ slug: string }> };

/** Dùng chung một lần gọi API cho generateMetadata và trang trong cùng request. */
const loadPost = cache((slug: string) => landingApi.postDetail(slug));

function safeDecode(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

function describe(excerpt: string, content: string): string {
  const text = (excerpt || stripTags(content)).replace(/\s+/g, ' ').trim();
  return text.length > 160 ? `${text.slice(0, 157)}...` : text;
}

export async function generateMetadata({ params }: Params): Promise<Metadata> {
  const { slug } = await params;
  const res = await loadPost(safeDecode(slug));
  if (!res.ok) {
    const t = await getTranslations('landing.blog');
    return { title: res.notFound ? '404' : t('loadError'), robots: { index: false } };
  }
  const post = res.data;
  const description = describe(post.excerpt, post.content);
  const url = postHref(post.slug);
  return {
    title: post.title,
    description,
    alternates: { canonical: url },
    openGraph: {
      type: 'article',
      title: post.title,
      description,
      url,
      publishedTime: post.published_at || post.created_at,
      images: post.cover_url ? [{ url: post.cover_url }] : undefined,
    },
  };
}

export default async function BlogPostPage({ params }: Params) {
  const { slug } = await params;
  const [t, session, res] = await Promise.all([
    getTranslations('landing.blog'),
    getCachedSession(),
    loadPost(safeDecode(slug)),
  ]);
  if (!res.ok) {
    if (res.notFound) notFound();
    throw new Error(t('loadError'));
  }
  const post = res.data;

  return (
    <div className="flex min-h-dvh flex-col bg-canvas">
      <LandingHeader isAuthenticated={!!session && !session.error} />
      <main className="mx-auto w-full max-w-3xl flex-1 px-4 py-10 sm:px-6">
        <Button variant="ghost" size="sm" asChild className="mb-4 -ml-3">
          <Link href="/blog">
            <ArrowLeftIcon />
            {t('backToBlog')}
          </Link>
        </Button>
        <article className="grid gap-6">
          <header className="grid gap-3">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="brand">{postCategoryLabel(post.category)}</Badge>
              <span className="text-caption text-tertiary">
                {t('publishedAt', { date: formatDate(post.published_at || post.created_at) })}
              </span>
            </div>
            <h1 className="text-h1 font-semibold break-words text-primary">{post.title}</h1>
            {post.excerpt ? (
              <p className="text-body-lg text-secondary">{post.excerpt}</p>
            ) : null}
          </header>
          <Card className="p-6 sm:p-8">
            <RichHtml html={post.content} className="text-body-lg text-secondary" />
          </Card>
          <div className="flex flex-col gap-4 rounded-2xl bg-brand-50 p-6 sm:flex-row sm:items-center sm:justify-between">
            <p className="text-body text-secondary">{t('relatedCtaDesc')}</p>
            <Button asChild className="shrink-0">
              <Link href="/#tap-thu">{t('relatedCta')}</Link>
            </Button>
          </div>
        </article>
      </main>
      <LandingFooter />
    </div>
  );
}
