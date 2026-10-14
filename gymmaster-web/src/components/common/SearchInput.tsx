'use client';

import { useEffect, useState } from 'react';
import { MagnifyingGlassIcon } from '@heroicons/react/24/outline';
import { Input } from '@/components/ui/input';
import { useDebounce } from '@/hooks/use-debounce';
import { cn } from '@/lib/utils';

/** Ô tìm kiếm có debounce – chỉ gọi onSearch khi người dùng ngừng gõ. */
export function SearchInput({
  value,
  onSearch,
  placeholder,
  className,
}: {
  value: string;
  onSearch: (keyword: string) => void;
  placeholder?: string;
  className?: string;
}) {
  const [text, setText] = useState(value);
  const [prevValue, setPrevValue] = useState(value);
  // Đồng bộ khi giá trị bị đổi từ bên ngoài (VD: nút "Xóa bộ lọc").
  if (value !== prevValue) {
    setPrevValue(value);
    setText(value);
  }
  const debounced = useDebounce(text, 400);

  useEffect(() => {
    if (debounced.trim() !== value) onSearch(debounced.trim());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debounced]);

  return (
    <div className={cn('relative w-full sm:w-72', className)}>
      <MagnifyingGlassIcon className="pointer-events-none absolute top-1/2 left-3 size-5 -translate-y-1/2 text-tertiary" />
      <Input
        type="search"
        value={text}
        onChange={e => setText(e.target.value)}
        placeholder={placeholder}
        aria-label={placeholder}
        className="pl-10"
      />
    </div>
  );
}
