'use client';

import { useEffect, useState, type ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import {
  Bars3Icon,
  ChevronDoubleLeftIcon,
  ChevronDoubleRightIcon,
} from '@heroicons/react/24/outline';
import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '@/components/ui/sheet';
import { cn } from '@/lib/utils';
import { useUiStore } from '@/store/ui-store';
import { Logo } from './Logo';
import { SidebarNav } from './SidebarNav';
import { UserMenu } from './UserMenu';

/**
 * Khung trang quản trị: sidebar tối (thu gọn được trên desktop, dạng Sheet trên mobile)
 * + topbar + vùng nội dung nền xám.
 */
export function AdminShell({ children }: { children: ReactNode }) {
  const t = useTranslations('layout.sidebar');
  const tc = useTranslations('common.actions');
  const collapsed = useUiStore(s => s.sidebarCollapsed);
  const toggleSidebar = useUiStore(s => s.toggleSidebar);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    void useUiStore.persist.rehydrate();
  }, []);

  return (
    <div className="flex min-h-dvh">
      {/* Sidebar desktop */}
      <aside
        className={cn(
          'sticky top-0 hidden h-dvh shrink-0 flex-col bg-ink-900 transition-[width] duration-200 lg:flex',
          collapsed ? 'w-20' : 'w-64',
        )}
      >
        <div className={cn('flex h-16 items-center px-5', collapsed && 'justify-center px-0')}>
          <Logo tone="light" showText={!collapsed} />
        </div>
        <div className="flex-1 overflow-y-auto px-3 py-4">
          <SidebarNav collapsed={collapsed} />
        </div>
        <div className="border-t border-ink-800 p-3">
          <button
            type="button"
            onClick={toggleSidebar}
            className={cn(
              'flex h-10 w-full cursor-pointer items-center gap-3 rounded-xl px-3 text-body text-disabled hover:bg-ink-800 hover:text-white',
              collapsed && 'justify-center px-0',
            )}
            aria-label={collapsed ? t('expand') : t('collapse')}
          >
            {collapsed ? (
              <ChevronDoubleRightIcon className="size-5" />
            ) : (
              <>
                <ChevronDoubleLeftIcon className="size-5" />
                <span>{t('collapse')}</span>
              </>
            )}
          </button>
        </div>
      </aside>

      {/* Sidebar mobile */}
      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent side="left" className="bg-ink-900" closeLabel={tc('close')}>
          <SheetTitle className="sr-only">{t('menuTitle')}</SheetTitle>
          <SheetDescription className="sr-only">{t('menuTitle')}</SheetDescription>
          <div className="flex h-16 items-center px-5">
            <Logo tone="light" />
          </div>
          <div className="flex-1 overflow-y-auto px-3 py-4">
            <SidebarNav onNavigate={() => setMobileOpen(false)} />
          </div>
        </SheetContent>
      </Sheet>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 flex h-16 items-center gap-3 border-b border-line bg-surface px-4 lg:px-8">
          <Button
            variant="ghost"
            size="icon"
            className="lg:hidden"
            onClick={() => setMobileOpen(true)}
            aria-label={t('openMenu')}
          >
            <Bars3Icon />
          </Button>
          <Logo className="lg:hidden" showText={false} />
          <div className="flex-1" />
          <UserMenu />
        </header>
        <main className="flex-1 px-4 py-6 lg:px-8 lg:py-8">
          <div className="mx-auto w-full max-w-7xl">{children}</div>
        </main>
      </div>
    </div>
  );
}
