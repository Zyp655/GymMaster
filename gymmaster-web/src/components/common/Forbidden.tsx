import { getTranslations } from 'next-intl/server';
import { NoSymbolIcon } from '@heroicons/react/24/outline';
import { EmptyState } from './EmptyState';

/** Hiển thị khi người dùng mở trực tiếp URL của trang mà họ không có quyền xem. */
export async function Forbidden() {
  const t = await getTranslations('common.forbiddenPage');
  return (
    <div className="rounded-2xl border border-line bg-surface">
      <EmptyState icon={NoSymbolIcon} title={t('title')} description={t('description')} />
    </div>
  );
}
