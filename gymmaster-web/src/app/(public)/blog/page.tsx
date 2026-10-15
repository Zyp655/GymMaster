import type { Metadata } from 'next';
import Link from 'next/link';
import { getCachedSession } from '@/guards/getCachedSession';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { landingApi } from '@/features/landing/api';
import { LandingFooter } from '@/features/landing/components/LandingFooter';
import { LandingHeader } from '@/features/landing/components/LandingHeader';
import { formatDate } from '@/lib/format';

export const metadata: Metadata = {
  title: 'Blog Kiến thức Tập luyện & Dinh dưỡng | GymMaster',
  description:
    'Cẩm nang tập luyện khoa học, thực đơn dinh dưỡng chuẩn BMR/TDEE và tin tức sự kiện từ hệ thống GymMaster.',
};

export default async function PublicBlogPage() {
  const [session, postsRes] = await Promise.all([getCachedSession(), landingApi.posts()]);
  const posts = postsRes.ok ? postsRes.data : [];

  return (
    <div className="flex min-h-dvh flex-col bg-canvas">
      <LandingHeader isAuthenticated={!!session && !session.error} />
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-12 sm:px-6">
        <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
          <div>
            <Badge variant="brand" className="mb-2">
              Blog & Kiến thức
            </Badge>
            <h1 className="text-h1 font-bold text-primary">
              Cẩm nang Tập luyện, Dinh dưỡng & Sự kiện
            </h1>
            <p className="mt-2 max-w-2xl text-body text-secondary">
              Tổng hợp kiến thức thể hình, thực đơn lành mạnh và kinh nghiệm tập luyện từ đội ngũ
              chuyên gia của GymMaster.
            </p>
          </div>
          <Button variant="outline" asChild>
            <Link href="/">← Về trang chủ</Link>
          </Button>
        </div>

        {posts.length === 0 ? (
          <Card className="p-8 text-center text-body text-secondary">
            Chưa có bài viết nào được xuất bản.
          </Card>
        ) : (
          <div className="grid gap-6 md:grid-cols-2">
            {posts.map(post => (
              <Card key={post.id} className="flex flex-col justify-between p-6">
                <div>
                  <div className="mb-3 flex items-center justify-between gap-2">
                    <Badge variant="brand">{post.category}</Badge>
                    <span className="text-xs text-muted">
                      {formatDate(post.published_at || post.created_at)}
                    </span>
                  </div>
                  <h2 className="text-h3 font-bold text-primary">{post.title}</h2>
                  <p className="mt-2 text-small font-medium text-secondary">{post.excerpt}</p>
                  <div className="mt-4 whitespace-pre-line border-t border-line pt-4 text-body text-secondary">
                    {post.content}
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}
      </main>
      <LandingFooter />
    </div>
  );
}
