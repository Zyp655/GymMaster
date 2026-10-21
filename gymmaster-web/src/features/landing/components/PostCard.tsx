import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { ArrowRightIcon } from '@heroicons/react/20/solid';
import { Badge } from '@/components/ui/badge';
import { Card } from '@/components/ui/card';
import { formatDate } from '@/lib/format';
import { postHref } from '../post-slug';
import { postCategoryLabel, type PublicPost } from '../types';

/** Thẻ bài viết (danh sách blog): tóm tắt + liên kết tới /blog/[slug]. */
export async function PostCard({ post }: { post: PublicPost }) {
  const t = await getTranslations('landing.blog');
  return (
    <Card className="flex flex-col justify-between gap-4 p-6">
      <div className="grid gap-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Badge variant="brand">{postCategoryLabel(post.category)}</Badge>
          <span className="text-caption text-tertiary">
            {t('publishedAt', { date: formatDate(post.published_at || post.created_at) })}
          </span>
        </div>
        <h2 className="text-h3 font-semibold text-primary">
          <Link href={postHref(post.slug)} className="hover:text-brand-600">
            {post.title}
          </Link>
        </h2>
        {post.excerpt ? (
          <p className="line-clamp-3 text-body text-secondary">{post.excerpt}</p>
        ) : null}
      </div>
      <Link
        href={postHref(post.slug)}
        className="inline-flex items-center gap-1 text-body font-semibold text-brand-600 hover:underline"
      >
        {t('readMore')}
        <ArrowRightIcon className="size-4" />
      </Link>
    </Card>
  );
}
