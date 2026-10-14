import { getTranslations } from 'next-intl/server';
import {
  CalendarDaysIcon,
  ChartBarIcon,
  QrCodeIcon,
  UserGroupIcon,
} from '@heroicons/react/24/outline';

export async function Features() {
  const t = await getTranslations('common.landing.features');
  const items = [
    { icon: QrCodeIcon, title: t('qrTitle'), desc: t('qrDesc') },
    { icon: CalendarDaysIcon, title: t('classTitle'), desc: t('classDesc') },
    { icon: UserGroupIcon, title: t('ptTitle'), desc: t('ptDesc') },
    { icon: ChartBarIcon, title: t('progressTitle'), desc: t('progressDesc') },
  ];
  return (
    <section className="mx-auto grid max-w-6xl gap-8 px-4 py-16 sm:px-6">
      <h2 className="text-h2 font-semibold text-primary">{t('title')}</h2>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {items.map(({ icon: Icon, title, desc }) => (
          <div
            key={title}
            className="grid content-start gap-3 rounded-2xl border border-line bg-surface p-5"
          >
            <span className="flex size-11 items-center justify-center rounded-xl bg-brand-50 text-brand-600">
              <Icon className="size-6" />
            </span>
            <h3 className="text-h4 font-semibold text-primary">{title}</h3>
            <p className="text-body text-secondary">{desc}</p>
          </div>
        ))}
      </div>
    </section>
  );
}
