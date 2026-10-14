import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { Button, Col, Input, Row, Select } from 'antd';
import { ClearOutlined } from '@ant-design/icons';
import { useUrlParams } from '@/hooks/useUrlParams';
import { normalizeText } from '@/utils/format';

export interface FilterOption {
  label: string;
  value: string | number;
}

export interface FilterDef {
  /** Tên tham số trên URL/API, ví dụ `status`. */
  name: string;
  placeholder?: string;
  options: FilterOption[];
  mode?: 'multiple';
  /** Độ rộng cột trên desktop (thang 24). Mặc định 4. */
  col?: number;
  loading?: boolean;
}

export interface FilterBarProps {
  filters?: FilterDef[];
  /** Hiện ô tìm kiếm từ khóa (`keyword`). */
  textSearch?: boolean;
  searchPlaceholder?: string;
  /** Độ rộng ô tìm kiếm (thang 24). Mặc định 6. */
  colTextSearch?: number;
  /** Phần tử lọc tùy biến thêm (DatePicker...). Tự gọi `useUrlParams().set(...)`. */
  children?: ReactNode;
}

const SEARCH_DEBOUNCE = 400;

/**
 * Thanh lọc: ô tìm kiếm + các Select cấu hình bằng mảng. Giá trị ghi thẳng lên URL query
 * (về trang 1), nên `useTableQuery` tự đọc lại và tải dữ liệu.
 *
 *   <FilterBar
 *     searchPlaceholder="Tìm theo mã, tên"
 *     filters={[{ name: 'status', placeholder: 'Trạng thái', options: STATUS_OPTIONS }]}
 *   />
 */
export function FilterBar({
  filters = [],
  textSearch = true,
  searchPlaceholder = 'Tìm kiếm...',
  colTextSearch = 6,
  children,
}: FilterBarProps) {
  const url = useUrlParams();
  const urlKeyword = url.get('keyword') ?? '';
  const [keyword, setKeyword] = useState(urlKeyword);
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);

  // URL đổi từ bên ngoài (nút Xóa lọc, quay lại trang) → đồng bộ ô nhập.
  useEffect(() => setKeyword(urlKeyword), [urlKeyword]);
  useEffect(() => () => clearTimeout(timer.current), []);

  const commitKeyword = (value: string) => {
    clearTimeout(timer.current);
    url.set({ keyword: value.trim() }, { resetPage: true });
  };

  const onKeywordChange = (value: string) => {
    setKeyword(value);
    clearTimeout(timer.current);
    timer.current = setTimeout(() => commitKeyword(value), SEARCH_DEBOUNCE);
  };

  const valueOf = (def: FilterDef) => {
    const raw = url.get(def.name);
    if (raw === undefined) return undefined;
    const pick = (s: string) => def.options.find((o) => String(o.value) === s)?.value ?? s;
    return def.mode === 'multiple' ? raw.split(',').map(pick) : pick(raw);
  };

  const hasActive = !!urlKeyword || filters.some((f) => url.get(f.name) !== undefined);

  const clearAll = () => {
    const patch: Record<string, undefined> = { keyword: undefined };
    filters.forEach((f) => (patch[f.name] = undefined));
    url.set(patch, { resetPage: true });
  };

  return (
    <Row gutter={[8, 8]} className="filter-bar">
      {textSearch && (
        <Col xs={24} sm={12} md={colTextSearch}>
          <Input.Search
            allowClear
            value={keyword}
            placeholder={searchPlaceholder}
            onChange={(e) => onKeywordChange(e.target.value)}
            onSearch={commitKeyword}
          />
        </Col>
      )}
      {filters.map((def) => (
        <Col xs={12} sm={8} md={def.col ?? 4} key={def.name}>
          <Select
            allowClear
            showSearch
            mode={def.mode}
            maxTagCount="responsive"
            loading={def.loading}
            placeholder={def.placeholder}
            value={valueOf(def)}
            options={def.options}
            style={{ width: '100%' }}
            filterOption={(input, option) =>
              normalizeText(String(option?.label ?? '')).includes(normalizeText(input))
            }
            onChange={(value) => url.set({ [def.name]: value as never }, { resetPage: true })}
          />
        </Col>
      ))}
      {children}
      {hasActive && (
        <Col>
          <Button type="text" icon={<ClearOutlined />} onClick={clearAll}>
            Xóa lọc
          </Button>
        </Col>
      )}
    </Row>
  );
}

export default FilterBar;
