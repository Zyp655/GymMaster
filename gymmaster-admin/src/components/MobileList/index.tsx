import { useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import type { QueryKey } from '@tanstack/react-query';
import { Badge, Button, DatePicker, Drawer, Empty, Input, Select, Spin, Switch, Tag } from 'antd';
import { FilterOutlined, MoreOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { Page } from '@/api/types';
import { usePermission } from '@/hooks/usePermission';
import type { PermInput } from '@/hooks/usePermission';
import { getErrorMessage } from '@/utils/errors';
import './styles.css';

export interface MobileFetchArgs {
  page: number;
  limit: number;
  keyword: string;
  /** Giá trị của `sorts` đang chọn (trang tự quy ước, ví dụ `created_at:DESC`). */
  sort?: string;
  filters: Record<string, unknown>;
}

export interface MobileRowConfig<T> {
  thumb?: (r: T) => string | undefined | null;
  /** Ảnh tròn (avatar), chữ cái đầu khi không có ảnh. */
  avatar?: boolean;
  title: (r: T) => ReactNode;
  subtitle?: (r: T) => ReactNode;
  meta?: (r: T) => (string | number | null | undefined | false)[];
  tags?: (r: T) => { label: ReactNode; color?: string }[];
  toggle?: { value: (r: T) => boolean; onChange?: (r: T, checked: boolean) => void };
}

export interface MobileAction<T> {
  key: string;
  label: ReactNode;
  icon?: ReactNode;
  danger?: boolean;
  disabled?: boolean;
  /** Không đóng action sheet sau khi bấm. */
  keepOpen?: boolean;
  onClick: (r: T) => void;
}

export interface MobileFilterDef {
  name: string;
  label: string;
  type?: 'select' | 'text' | 'daterange';
  options?: { label: string; value: string | number }[];
}

export interface MobileListProps<T> {
  title: ReactNode;
  /** Khóa query gốc (dùng chung với bản desktop để invalidate một lần là đủ). */
  queryKey: QueryKey;
  fetchPage: (args: MobileFetchArgs) => Promise<Page<T>>;
  row: MobileRowConfig<T>;
  rowKey?: (r: T) => string | number;
  onOpen?: (r: T) => void;
  onAdd?: () => void;
  addPerm?: PermInput;
  actions?: (r: T) => MobileAction<T>[];
  /** Cặp [nhãn, giá trị] hiển thị đầu action sheet. */
  details?: (r: T) => [string, ReactNode][];
  sorts?: { label: string; value: string }[];
  filters?: MobileFilterDef[];
  pageSize?: number;
  searchPlaceholder?: string;
  emptyText?: string;
  stats?: [string, ReactNode][];
  showSearch?: boolean;
}

const isEmptyValue = (v: unknown) =>
  v === undefined || v === null || v === '' || (Array.isArray(v) && v.length === 0);

/**
 * Màn hình danh sách dạng thẻ cho điện thoại (theo mẫu strongbody_admin):
 * tìm kiếm, sắp xếp, drawer bộ lọc, "Tải thêm", action sheet cho từng dòng.
 * Trang list: `return isMobile ? <XxxMobile /> : <XxxDesktop />`.
 */
export function MobileList<T>(props: MobileListProps<T>) {
  const {
    title,
    queryKey,
    fetchPage,
    row,
    rowKey = (r: T) => (r as { id?: number }).id ?? JSON.stringify(r),
    onOpen,
    onAdd,
    addPerm,
    actions,
    details,
    sorts,
    filters: filterDefs,
    pageSize = 20,
    searchPlaceholder = 'Tìm kiếm',
    emptyText = 'Chưa có dữ liệu',
    stats,
    showSearch = true,
  } = props;
  const { can } = usePermission();

  const emptyFilters = useMemo(
    () => Object.fromEntries((filterDefs ?? []).map((f) => [f.name, undefined])),
    [filterDefs],
  );

  const [keywordInput, setKeywordInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [sort, setSort] = useState<string | undefined>(sorts?.[0]?.value);
  const [filters, setFilters] = useState<Record<string, unknown>>(emptyFilters);
  const [draftFilters, setDraftFilters] = useState<Record<string, unknown>>(emptyFilters);
  const [filterDrawer, setFilterDrawer] = useState(false);
  const [sheet, setSheet] = useState<T | null>(null);
  const [timer, setTimer] = useState<ReturnType<typeof setTimeout>>();

  const query = useInfiniteQuery({
    queryKey: [...queryKey, 'mobile', { keyword, sort, filters, pageSize }],
    initialPageParam: 1,
    queryFn: ({ pageParam }) =>
      fetchPage({ page: pageParam, limit: pageSize, keyword, sort, filters }),
    getNextPageParam: (last, all) => {
      const loaded = all.reduce((n, p) => n + p.items.length, 0);
      return loaded < last.total ? all.length + 1 : undefined;
    },
  });

  const items = useMemo(() => query.data?.pages.flatMap((p) => p.items) ?? [], [query.data]);
  const total = query.data?.pages[0]?.total ?? 0;

  const onSearchChange = (value: string) => {
    setKeywordInput(value);
    if (timer) clearTimeout(timer);
    setTimer(setTimeout(() => setKeyword(value.trim()), 400));
  };

  const activeFilterCount = Object.values(filters).filter((v) => !isEmptyValue(v)).length;
  const sheetActions = sheet && actions ? actions(sheet) : [];
  const sheetDetails = sheet && details ? details(sheet) : [];

  return (
    <div className="ml">
      <div className="ml__bar">
        <div className="ml__title">
          {title} <span className="ml__count">{total}</span>
        </div>
        <div className="ml__bar-actions">
          <Button
            icon={<ReloadOutlined spin={query.isRefetching} />}
            onClick={() => query.refetch()}
            aria-label="Làm mới"
          />
          {onAdd && can(addPerm) && (
            <Button
              color="orange"
              variant="solid"
              icon={<PlusOutlined />}
              onClick={onAdd}
              aria-label="Thêm mới"
            />
          )}
        </div>
      </div>

      {stats && stats.length > 0 && (
        <div className="ml__stats">
          {stats.map(([label, value]) => (
            <div className="ml__stat" key={label}>
              <div className="ml__stat-value">{value}</div>
              <div className="ml__stat-label">{label}</div>
            </div>
          ))}
        </div>
      )}

      {(showSearch || !!sorts?.length || !!filterDefs?.length) && (
        <div className="ml__filters">
          {showSearch && (
            <Input.Search
              allowClear
              value={keywordInput}
              placeholder={searchPlaceholder}
              onChange={(e) => onSearchChange(e.target.value)}
              onSearch={(v) => setKeyword(v.trim())}
            />
          )}
          {(!!sorts?.length || !!filterDefs?.length) && (
            <div className="ml__filters-row">
              {!!sorts?.length && (
                <Select value={sort} onChange={setSort} options={sorts} className="ml__sort" />
              )}
              {!!filterDefs?.length && (
                <Badge count={activeFilterCount} size="small">
                  <Button
                    icon={<FilterOutlined />}
                    onClick={() => {
                      setDraftFilters(filters);
                      setFilterDrawer(true);
                    }}
                  >
                    Bộ lọc
                  </Button>
                </Badge>
              )}
            </div>
          )}
        </div>
      )}

      {query.isLoading ? (
        <div className="ml__loading">
          <Spin />
        </div>
      ) : query.isError && items.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={getErrorMessage(query.error, 'Không tải được dữ liệu')}
        >
          <Button onClick={() => query.refetch()}>Thử lại</Button>
        </Empty>
      ) : items.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={emptyText} />
      ) : (
        <div className="ml__list">
          {items.map((record) => {
            const thumb = row.thumb?.(record);
            const rowTitle = row.title(record);
            const subtitle = row.subtitle?.(record);
            const meta = (row.meta?.(record) ?? []).filter(
              (m) => m !== null && m !== undefined && m !== false && m !== '',
            );
            const tags = row.tags?.(record) ?? [];
            const toggle = row.toggle;

            return (
              <div className="ml__row" key={rowKey(record)}>
                <button
                  type="button"
                  className="ml__open"
                  onClick={() => (onOpen ? onOpen(record) : setSheet(record))}
                >
                  {row.thumb ? (
                    thumb ? (
                      <img
                        className={`ml__thumb${row.avatar ? ' ml__thumb--round' : ''}`}
                        src={thumb}
                        alt=""
                        loading="lazy"
                      />
                    ) : (
                      <span
                        className={`ml__thumb ml__thumb--empty${row.avatar ? ' ml__thumb--round' : ''}`}
                      >
                        {row.avatar
                          ? String(rowTitle || '?')
                              .charAt(0)
                              .toUpperCase()
                          : null}
                      </span>
                    )
                  ) : null}
                  <span className="ml__text">
                    <span className="ml__name">{rowTitle}</span>
                    {subtitle ? <span className="ml__subtitle">{subtitle}</span> : null}
                    {(meta.length > 0 || tags.length > 0) && (
                      <span className="ml__meta">
                        {tags.map((tag, i) => (
                          <Tag key={i} color={tag.color} className="ml__tag">
                            {tag.label}
                          </Tag>
                        ))}
                        {meta.join(' · ')}
                      </span>
                    )}
                  </span>
                </button>

                <div className="ml__side">
                  {toggle && (
                    <Switch
                      size="small"
                      checked={toggle.value(record)}
                      disabled={!toggle.onChange}
                      onChange={(checked) => toggle.onChange?.(record, checked)}
                    />
                  )}
                  {actions && (
                    <Button
                      type="text"
                      size="small"
                      icon={<MoreOutlined />}
                      onClick={() => setSheet(record)}
                      aria-label="Thao tác"
                    />
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {query.hasNextPage && (
        <div className="ml__more">
          <Button block loading={query.isFetchingNextPage} onClick={() => query.fetchNextPage()}>
            Tải thêm ({items.length}/{total})
          </Button>
        </div>
      )}

      {/* Action sheet */}
      <Drawer
        placement="bottom"
        height="auto"
        open={!!sheet}
        onClose={() => setSheet(null)}
        closable={false}
        styles={{ body: { padding: 0 } }}
      >
        {sheet && (
          <>
            <div className="ml__sheet-head">
              <div className="ml__sheet-title">{row.title(sheet)}</div>
              {sheetDetails.length > 0 && (
                <div className="ml__sheet-grid">
                  {sheetDetails.map(([label, value]) => (
                    <div key={label}>
                      <span>{label}</span>
                      <b>{isEmptyValue(value) ? '—' : value}</b>
                    </div>
                  ))}
                </div>
              )}
            </div>
            <div className="ml__sheet-list">
              {sheetActions.map((action) => (
                <button
                  key={action.key}
                  type="button"
                  className={`ml__sheet-item${action.danger ? ' ml__sheet-item--danger' : ''}`}
                  disabled={action.disabled}
                  onClick={() => {
                    const target = sheet;
                    if (!action.keepOpen) setSheet(null);
                    action.onClick(target);
                  }}
                >
                  {action.icon ? <span className="ml__sheet-icon">{action.icon}</span> : null}
                  {action.label}
                </button>
              ))}
            </div>
          </>
        )}
      </Drawer>

      {/* Bộ lọc */}
      {!!filterDefs?.length && (
        <Drawer
          title="Bộ lọc"
          placement="bottom"
          height="auto"
          open={filterDrawer}
          onClose={() => setFilterDrawer(false)}
          footer={
            <div className="ml__drawer-footer">
              <Button
                onClick={() => {
                  setDraftFilters(emptyFilters);
                  setFilters(emptyFilters);
                  setFilterDrawer(false);
                }}
              >
                Xóa lọc
              </Button>
              <Button
                type="primary"
                onClick={() => {
                  setFilters(draftFilters);
                  setFilterDrawer(false);
                }}
              >
                Áp dụng
              </Button>
            </div>
          }
        >
          <div className="ml__filter-form">
            {filterDefs.map((def) => (
              <label key={def.name}>
                <span>{def.label}</span>
                {def.type === 'text' ? (
                  <Input
                    allowClear
                    value={(draftFilters[def.name] as string) ?? ''}
                    onChange={(e) =>
                      setDraftFilters((prev) => ({ ...prev, [def.name]: e.target.value }))
                    }
                  />
                ) : def.type === 'daterange' ? (
                  <DatePicker.RangePicker
                    format="DD/MM/YYYY"
                    value={(draftFilters[def.name] as never) ?? null}
                    onChange={(dates) =>
                      setDraftFilters((prev) => ({ ...prev, [def.name]: dates }))
                    }
                  />
                ) : (
                  <Select
                    allowClear
                    showSearch
                    optionFilterProp="label"
                    value={draftFilters[def.name] as string | undefined}
                    onChange={(value) =>
                      setDraftFilters((prev) => ({ ...prev, [def.name]: value }))
                    }
                    options={def.options ?? []}
                  />
                )}
              </label>
            ))}
          </div>
        </Drawer>
      )}
    </div>
  );
}

export default MobileList;
