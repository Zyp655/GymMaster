'use client';

import type { ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import { ExclamationTriangleIcon } from '@heroicons/react/24/outline';
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';

/**
 * Hộp thoại xác nhận cho thao tác nguy hiểm (xóa...).
 * Nút xác nhận giữ dialog mở và hiện trạng thái loading cho tới khi onConfirm hoàn tất.
 */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  loading,
  onConfirm,
  variant = 'danger',
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title?: ReactNode;
  description: ReactNode;
  confirmLabel?: ReactNode;
  loading?: boolean;
  onConfirm: () => void | Promise<unknown>;
  variant?: 'danger' | 'primary';
}) {
  const t = useTranslations('common');
  return (
    <AlertDialog open={open} onOpenChange={next => !loading && onOpenChange(next)}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <div className="flex items-start gap-3">
            {variant === 'danger' ? (
              <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-danger-50 text-danger">
                <ExclamationTriangleIcon className="size-5" />
              </div>
            ) : null}
            <div className="grid gap-1">
              <AlertDialogTitle>{title ?? t('confirm.deleteTitle')}</AlertDialogTitle>
              <AlertDialogDescription>{description}</AlertDialogDescription>
            </div>
          </div>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={loading}>{t('actions.cancel')}</AlertDialogCancel>
          <Button
            variant={variant}
            disabled={loading}
            onClick={() => {
              void onConfirm();
            }}
          >
            {loading ? t('confirm.deleting') : (confirmLabel ?? t('actions.delete'))}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
