'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Checkbox } from '@/components/ui/checkbox';
import type { Permission } from '../types';

/** Danh sách quyền nhóm theo module, có "Chọn tất cả" cho từng nhóm. */
export function PermissionChecklist({
  permissions,
  value,
  onChange,
  disabled,
}: {
  permissions: Permission[];
  value: string[];
  onChange: (next: string[]) => void;
  disabled?: boolean;
}) {
  const t = useTranslations('role');
  const selected = useMemo(() => new Set(value), [value]);

  const groups = useMemo(() => {
    const map = new Map<string, Permission[]>();
    for (const p of permissions) {
      const list = map.get(p.module) ?? [];
      list.push(p);
      map.set(p.module, list);
    }
    return Array.from(map.entries());
  }, [permissions]);

  const moduleLabel = (module: string) => {
    const key = `modules.${module}` as const;
    return t.has(key as Parameters<typeof t.has>[0]) ? t(key as Parameters<typeof t>[0]) : module;
  };

  const toggle = (code: string, checked: boolean) => {
    const next = new Set(selected);
    if (checked) next.add(code);
    else next.delete(code);
    onChange(Array.from(next));
  };

  const toggleGroup = (codes: string[], checked: boolean) => {
    const next = new Set(selected);
    for (const code of codes) {
      if (checked) next.add(code);
      else next.delete(code);
    }
    onChange(Array.from(next));
  };

  return (
    <div className="grid gap-3">
      {groups.map(([module, perms]) => {
        const codes = perms.map(p => p.code);
        const count = codes.filter(c => selected.has(c)).length;
        const groupState = count === 0 ? false : count === codes.length ? true : 'indeterminate';
        const groupId = `perm-group-${module}`;
        return (
          <div key={module} className="grid gap-3 rounded-xl bg-canvas p-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <label htmlFor={groupId} className="flex cursor-pointer items-center gap-2">
                <Checkbox
                  id={groupId}
                  checked={groupState}
                  disabled={disabled}
                  onCheckedChange={checked => toggleGroup(codes, checked === true)}
                />
                <span className="text-body font-semibold text-primary">{moduleLabel(module)}</span>
              </label>
              <span className="text-caption text-tertiary">
                {t('selectedCount', { selected: count, total: codes.length })}
              </span>
            </div>
            <div className="grid gap-2 pl-7 sm:grid-cols-2">
              {perms.map(p => {
                const id = `perm-${p.code}`;
                return (
                  <label
                    key={p.code}
                    htmlFor={id}
                    className="flex cursor-pointer items-start gap-2"
                  >
                    <Checkbox
                      id={id}
                      checked={selected.has(p.code)}
                      disabled={disabled}
                      onCheckedChange={checked => toggle(p.code, checked === true)}
                    />
                    <span className="grid gap-0.5">
                      <span className="text-body text-primary">{p.name}</span>
                      <span className="text-caption text-tertiary">{p.code}</span>
                    </span>
                  </label>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}
