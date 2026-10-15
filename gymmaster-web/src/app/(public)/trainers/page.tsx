import type { Metadata } from 'next';
import Link from 'next/link';
import { StarIcon } from '@heroicons/react/24/solid';
import { getCachedSession } from '@/guards/getCachedSession';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { landingApi } from '@/features/landing/api';
import { LandingFooter } from '@/features/landing/components/LandingFooter';
import { LandingHeader } from '@/features/landing/components/LandingHeader';

export const metadata: Metadata = {
  title: 'Đội ngũ Huấn luyện viên cá nhân (PT) | GymMaster',
  description:
    'Khám phá đội ngũ Huấn luyện viên cá nhân giàu kinh nghiệm, chứng chỉ quốc tế tại hệ thống phòng tập GymMaster.',
};

export default async function PublicTrainersPage() {
  const [session, trainersRes] = await Promise.all([getCachedSession(), landingApi.trainers()]);
  const trainers = trainersRes.ok ? trainersRes.data : [];

  return (
    <div className="flex min-h-dvh flex-col bg-canvas">
      <LandingHeader isAuthenticated={!!session && !session.error} />
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-12 sm:px-6">
        <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
          <div>
            <Badge variant="brand" className="mb-2">
              Huấn luyện viên cá nhân
            </Badge>
            <h1 className="text-h1 font-bold text-primary">Đội ngũ HLV Chuyên nghiệp</h1>
            <p className="mt-2 max-w-2xl text-body text-secondary">
              Mỗi huấn luyện viên tại GymMaster đều được tuyển chọn khắt khe về chuyên môn, chứng
              chỉ quốc tế và tận tâm đồng hành cùng mục tiêu hình thể của bạn.
            </p>
          </div>
          <Button asChild>
            <Link href="/#tap-thu">Đăng ký tập thử cùng PT</Link>
          </Button>
        </div>

        {trainers.length === 0 ? (
          <Card className="p-8 text-center text-body text-secondary">
            Danh sách hồ sơ huấn luyện viên đang được cập nhật.
          </Card>
        ) : (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {trainers.map(tr => (
              <Card key={tr.id} className="flex flex-col justify-between p-6">
                <div>
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h2 className="text-h3 font-bold text-primary">{tr.full_name}</h2>
                      <p className="mt-0.5 text-small font-medium text-brand-600">{tr.headline}</p>
                    </div>
                    <span className="inline-flex items-center gap-1 rounded-full bg-amber-50 px-2.5 py-1 text-xs font-semibold text-amber-700">
                      <StarIcon className="size-3.5 text-amber-500" />
                      {(tr.rating_avg || 5).toFixed(1)} ({tr.rating_count})
                    </span>
                  </div>
                  <p className="mt-4 text-body text-secondary">{tr.bio}</p>
                </div>
                <div className="mt-6 space-y-1.5 border-t border-line pt-4 text-small text-secondary">
                  <div>
                    <strong className="text-primary">Chuyên môn:</strong>{' '}
                    {tr.specialties || 'Tăng cơ, Giảm mỡ, Phục hồi'}
                  </div>
                  <div>
                    <strong className="text-primary">Chứng chỉ:</strong>{' '}
                    {tr.certifications || 'NASM / ACE CPT'}
                  </div>
                  <div>
                    <strong className="text-primary">Kinh nghiệm:</strong> {tr.experience_years}{' '}
                    năm huấn luyện
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
