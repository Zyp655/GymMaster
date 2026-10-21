import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { ArrowRightIcon } from '@heroicons/react/20/solid';
import { StarIcon } from '@heroicons/react/24/solid';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { postHref } from '../post-slug';
import {
  postCategoryLabel,
  type PublicPost,
  type PublicTestimonial,
  type PublicTrainer,
} from '../types';
import { TrainerRating } from './TrainerRating';

export async function CommunitySection({
  trainers,
  testimonials,
  posts,
}: {
  trainers: PublicTrainer[];
  testimonials: PublicTestimonial[];
  posts: PublicPost[];
}) {
  const t = await getTranslations('landing.community');
  return (
    <section id="cong-dong" className="border-t border-line bg-surface py-16">
      <div className="mx-auto max-w-6xl space-y-16 px-4 sm:px-6">
        {/* Đội ngũ HLV */}
        <div>
          <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
            <div className="grid gap-2">
              <Badge variant="brand" className="w-fit">
                {t('trainersEyebrow')}
              </Badge>
              <h2 className="text-h2 font-semibold text-primary">{t('trainersTitle')}</h2>
              <p className="text-body text-secondary">{t('trainersSubtitle')}</p>
            </div>
            <Button variant="outline" asChild>
              <Link href="/trainers">{t('allTrainers')}</Link>
            </Button>
          </div>

          {trainers.length === 0 ? (
            <Card className="p-6 text-body text-secondary">{t('trainersEmpty')}</Card>
          ) : (
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {trainers.slice(0, 3).map(tr => (
                <Card key={tr.id} className="flex flex-col justify-between gap-4 p-6">
                  <div className="grid gap-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <h3 className="text-h4 font-semibold break-words text-primary">
                          {tr.full_name}
                        </h3>
                        <p className="text-body font-medium text-brand-600">{tr.headline}</p>
                      </div>
                      <TrainerRating trainer={tr} />
                    </div>
                    <p className="line-clamp-3 text-body text-secondary">{tr.bio}</p>
                  </div>
                  <div className="grid gap-1 border-t border-line pt-3 text-caption text-tertiary">
                    {tr.specialties ? (
                      <div>{t('specialties', { value: tr.specialties })}</div>
                    ) : null}
                    <div>
                      {t('experience', { years: tr.experience_years })}
                      {tr.certifications ? ` · ${tr.certifications}` : ''}
                    </div>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </div>

        {/* Cảm nhận hội viên */}
        {testimonials.length > 0 && (
          <div>
            <div className="mb-6 grid gap-2">
              <Badge variant="brand" className="w-fit">
                {t('testimonialsEyebrow')}
              </Badge>
              <h2 className="text-h2 font-semibold text-primary">{t('testimonialsTitle')}</h2>
            </div>
            <div className="grid gap-6 md:grid-cols-2">
              {testimonials.slice(0, 4).map(item => (
                <Card key={item.id} className="gap-4 p-6">
                  <div className="flex items-center gap-1 text-warning">
                    {Array.from({ length: Math.min(Math.max(item.rating || 5, 1), 5) }).map(
                      (_, idx) => (
                        <StarIcon key={idx} className="size-4" />
                      ),
                    )}
                  </div>
                  <p className="text-body text-primary italic">&ldquo;{item.content}&rdquo;</p>
                  <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line pt-3">
                    <div>
                      <div className="text-body font-semibold text-primary">
                        {item.author_name}
                      </div>
                      <div className="text-caption text-secondary">{item.author_role}</div>
                    </div>
                    {item.before_after_note ? (
                      <Badge variant="success">{item.before_after_note}</Badge>
                    ) : null}
                  </div>
                </Card>
              ))}
            </div>
          </div>
        )}

        {/* Cẩm nang tập luyện & Blog */}
        <div>
          <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
            <div className="grid gap-2">
              <Badge variant="brand" className="w-fit">
                {t('postsEyebrow')}
              </Badge>
              <h2 className="text-h2 font-semibold text-primary">{t('postsTitle')}</h2>
            </div>
            <Button variant="outline" asChild>
              <Link href="/blog">{t('morePosts')}</Link>
            </Button>
          </div>
          {posts.length === 0 ? (
            <Card className="p-6 text-body text-secondary">{t('postsEmpty')}</Card>
          ) : (
            <div className="grid gap-6 md:grid-cols-2">
              {posts.slice(0, 2).map(post => (
                <Card key={post.id} className="flex flex-col justify-between gap-4 p-6">
                  <div className="grid gap-2">
                    <Badge variant="neutral" className="w-fit">
                      {postCategoryLabel(post.category)}
                    </Badge>
                    <h3 className="text-h4 font-semibold text-primary">{post.title}</h3>
                    <p className="line-clamp-3 text-body text-secondary">{post.excerpt}</p>
                  </div>
                  <Link
                    href={postHref(post.slug)}
                    className="inline-flex items-center gap-1 text-body font-semibold text-brand-600 hover:underline"
                  >
                    {t('readMore')}
                    <ArrowRightIcon className="size-4" />
                  </Link>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
