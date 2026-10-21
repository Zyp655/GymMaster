'use client';

import { useState } from 'react';
import { Bars3Icon } from '@heroicons/react/24/outline';
import { Logo } from '@/components/layout/Logo';
import { Button } from '@/components/ui/button';
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet';

/** Menu điều hướng trang công khai cho màn hình < lg (desktop dùng thanh nav ngang). */
export function LandingMobileNav({
  links,
  labels,
}: {
  links: { href: string; label: string }[];
  labels: { open: string; close: string; title: string };
}) {
  const [open, setOpen] = useState(false);
  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="lg:hidden" aria-label={labels.open}>
          <Bars3Icon />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" closeLabel={labels.close} className="p-5">
        <div className="mb-6">
          <Logo />
        </div>
        <SheetTitle className="sr-only">{labels.title}</SheetTitle>
        <SheetDescription className="sr-only">{labels.title}</SheetDescription>
        <nav aria-label={labels.title} className="grid gap-1">
          {links.map(link => (
            <SheetClose asChild key={link.href}>
              <a
                href={link.href}
                className="rounded-xl px-3 py-2.5 text-body font-medium text-secondary transition-colors hover:bg-subtle hover:text-primary"
              >
                {link.label}
              </a>
            </SheetClose>
          ))}
        </nav>
      </SheetContent>
    </Sheet>
  );
}
