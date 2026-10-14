'use client';

import { useState } from 'react';
import { signOut } from 'next-auth/react';
import { useTranslations } from 'next-intl';
import {
  ArrowRightStartOnRectangleIcon,
  ChevronDownIcon,
  KeyIcon,
} from '@heroicons/react/24/outline';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ChangePasswordDialog } from '@/features/auth/components/ChangePasswordDialog';
import { useAuth } from '@/hooks/use-auth';
import { initials } from '@/lib/format';
import { setClientAccessToken } from '@/lib/session-token';

export function UserMenu() {
  const t = useTranslations('layout');
  const { user } = useAuth();
  const [changePasswordOpen, setChangePasswordOpen] = useState(false);

  if (!user) return null;

  const roleLabel = user.roles
    .map(code =>
      t.has(`roleNames.${code}` as Parameters<typeof t.has>[0])
        ? t(`roleNames.${code}` as Parameters<typeof t>[0])
        : code,
    )
    .join(', ');

  const logout = async () => {
    setClientAccessToken(null);
    // events.signOut ở authOptions sẽ gọi POST /auth/logout để thu hồi refresh token.
    await signOut({ callbackUrl: '/login' });
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger className="flex cursor-pointer items-center gap-2.5 rounded-xl p-1 pr-2 outline-none hover:bg-subtle focus-visible:ring-2 focus-visible:ring-ring">
          <Avatar>
            {user.avatar_url ? <AvatarImage src={user.avatar_url} alt={user.full_name} /> : null}
            <AvatarFallback>{initials(user.full_name)}</AvatarFallback>
          </Avatar>
          <span className="hidden min-w-0 text-left md:grid">
            <span className="truncate text-body font-medium text-primary">{user.full_name}</span>
            <span className="truncate text-caption text-tertiary">{roleLabel}</span>
          </span>
          <ChevronDownIcon className="hidden size-4 text-tertiary md:block" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-64">
          <DropdownMenuLabel className="grid gap-0.5">
            <span className="truncate">{user.full_name}</span>
            <span className="truncate text-caption font-normal text-tertiary">{user.email}</span>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={() => setChangePasswordOpen(true)}>
            <KeyIcon />
            {t('user.changePassword')}
          </DropdownMenuItem>
          <DropdownMenuItem variant="danger" onSelect={() => void logout()}>
            <ArrowRightStartOnRectangleIcon />
            {t('user.logout')}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <ChangePasswordDialog open={changePasswordOpen} onOpenChange={setChangePasswordOpen} />
    </>
  );
}
