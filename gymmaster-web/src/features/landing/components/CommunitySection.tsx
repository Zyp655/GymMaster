import Link from 'next/link';
import { StarIcon } from '@heroicons/react/24/solid';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import type { PublicPost, PublicTestimonial, PublicTrainer } from '../types';

export function CommunitySection({
  trainers,
  testimonials,
  posts,
}: {
  trainers: PublicTrainer[];
  testimonials: PublicTestimonial[];
  posts: PublicPost[];
}) {
  return (
    <section id="cong-dong" className="border-t border-line bg-surface py-16">
      <div className="mx-auto max-w-6xl space-y-16 px-4 sm:px-6">
        {/* Đội ngũ HLV */}
        <div>
          <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
            <div>
              <Badge variant="brand" className="mb-2">
                Đội ngũ chuyên gia
              </Badge>
              <h2 className="text-h2 font-bold text-primary">
                Huấn luyện viên cá nhân đạt chuẩn quốc tế
              </h2>
              <p className="mt-1 text-body text-secondary">
                Đồng hành 1-1 xây dựng giáo án tập luyện và thực đơn dinh dưỡng cá nhân hóa.
              </p>
            </div>
            <Button variant="outline" asChild>
              <Link href="/trainers">Xem tất cả HLV</Link>
            </Button>
          </div>

          {trainers.length === 0 ? (
            <Card className="p-6 text-body text-secondary">
              Danh sách huấn luyện viên đang được cập nhật.
            </Card>
          ) : (
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {trainers.slice(0, 3).map(tr => (
                <Card key={tr.id} className="flex flex-col justify-between p-6">
                  <div>
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <h3 className="text-h4 font-semibold text-primary">{tr.full_name}</h3>
                        <p className="text-small font-medium text-brand-600">{tr.headline}</p>
                      </div>
                      <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1 text-xs font-semibold text-amber-700">
                        <StarIcon className="size-3.5 text-amber-500" />
                        {(tr.rating_avg || 5).toFixed(1)} ({tr.rating_count})
                      </span>
                    </div>
                    <p className="mt-3 line-clamp-3 text-small text-secondary">{tr.bio}</p>
                  </div>
                  <div className="mt-4 border-t border-line pt-3 text-xs text-muted">
                    <div>Chuyên môn: {tr.specialties || 'Tăng cơ, Giảm mỡ, Phục hồi'}</div>
                    <div className="mt-1">
                      Kinh nghiệm: {tr.experience_years} năm · {tr.certifications}
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
            <div className="mb-6">
              <Badge variant="brand" className="mb-2">
                Câu chuyện thành công
              </Badge>
              <h2 className="text-h2 font-bold text-primary">Cảm nhận từ Hội viên GymMaster</h2>
            </div>
            <div className="grid gap-6 md:grid-cols-2">
              {testimonials.slice(0, 4).map(item => (
                <Card key={item.id} className="p-6">
                  <div className="mb-2 flex items-center gap-1 text-amber-500">
                    {Array.from({ length: item.rating || 5 }).map((_, idx) => (
                      <StarIcon key={idx} className="size-4" />
                    ))}
                  </div>
                  <p className="text-body italic text-primary">&ldquo;{item.content}&rdquo;</p>
                  <div className="mt-4 flex items-center justify-between border-t border-line pt-3">
                    <div>
                      <div className="text-small font-semibold text-primary">
                        {item.author_name}
                      </div>
                      <div className="text-xs text-secondary">{item.author_role}</div>
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
            <div>
              <Badge variant="brand" className="mb-2">
                Kiến thức & Tin tức
              </Badge>
              <h2 className="text-h2 font-bold text-primary">Cẩm nang Tập luyện & Dinh dưỡng</h2>
            </div>
            <Button variant="outline" asChild>
              <Link href="/blog">Đọc thêm bài viết</Link>
            </Button>
          </div>
          {posts.length === 0 ? (
            <Card className="p-6 text-body text-secondary">
              Chưa có bài viết công khai nào.
            </Card>
          ) : (
            <div className="grid gap-6 md:grid-cols-2">
              {posts.slice(0, 2).map(post => (
                <Card key={post.id} className="flex flex-col justify-between p-6">
                  <div>
                    <Badge variant="neutral" className="mb-2">
                      {post.category}
                    </Badge>
                    <h3 className="text-h4 font-semibold text-primary">{post.title}</h3>
                    <p className="mt-2 line-clamp-3 text-small text-secondary">{post.excerpt}</p>
                  </div>
                  <div className="mt-4 pt-2">
                    <Link
                      href="/blog"
                      className="text-small font-semibold text-brand-600 hover:underline"
                    >
                      Xem chi tiết →
                    </Link>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
