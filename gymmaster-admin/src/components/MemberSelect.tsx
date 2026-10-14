import { useEffect, useMemo, useState } from 'react';
import { Avatar, Select, Space, Spin, Typography } from 'antd';
import type { SelectProps } from 'antd';
import { UserOutlined } from '@ant-design/icons';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import memberApi from '@/api/member';
import type { MemberOption } from '@/api/member';
import { formatPhone } from '@/utils/format';

export interface MemberSelectProps extends Omit<
  SelectProps<number>,
  'options' | 'onChange' | 'value' | 'showSearch' | 'filterOption' | 'onSearch' | 'mode'
> {
  /** id hội viên đang chọn. */
  value?: number | null;
  /** Đổi lựa chọn: `id` (undefined khi xóa) + đối tượng hội viên rút gọn. */
  onChange?: (id: number | undefined, member: MemberOption | undefined) => void;
  /** Giới hạn theo chi nhánh chính (`branch_id`). Bỏ trống = mọi chi nhánh trong phạm vi user. */
  branchId?: number | null;
  /**
   * Hội viên đang chọn sẵn (khi mở form sửa) – để hiển thị nhãn ngay mà không cần tìm kiếm.
   * Truyền đối tượng tối thiểu `{ id, full_name, member_code, phone, avatar_url, status }`.
   */
  initialMember?: MemberOption | null;
  /** Ẩn hội viên không ở trạng thái ACTIVE. */
  activeOnly?: boolean;
}

const DEBOUNCE_MS = 300;

/** Avatar tròn: ảnh hoặc chữ cái đầu của tên. */
export function MemberAvatar({
  src,
  name,
  size = 28,
}: {
  src?: string | null;
  name?: string;
  size?: number;
}) {
  const initial = name?.trim().split(/\s+/).pop()?.charAt(0).toUpperCase();
  return (
    <Avatar
      src={src || undefined}
      size={size}
      icon={!src && !initial ? <UserOutlined /> : undefined}
      style={src ? undefined : { background: '#0A3150', flexShrink: 0 }}
    >
      {initial}
    </Avatar>
  );
}

/**
 * Ô chọn hội viên có tìm kiếm phía server – dùng chung cho bán gói, lớp, PT, POS, check-in...
 *
 * - Gọi `GET /admin/members/options?keyword=&branch_id=` (tối đa 20 kết quả, debounce 300 ms).
 *   Endpoint cho phép các quyền: member.view | subscription.sell | checkin.scan | booking.manage |
 *   pt.sell | pos.sell.
 * - Mỗi option: avatar + họ tên + mã + SĐT. Hội viên không ACTIVE có nhãn trạng thái.
 * - `value` là **id hội viên**; `onChange(id, member)` trả kèm đối tượng `MemberOption`.
 *
 * ```tsx
 * <Form.Item name="member_id" label="Hội viên" rules={[{ required: true }]}>
 *   <MemberSelect branchId={branchId} />
 * </Form.Item>
 *
 * <MemberSelect value={id} onChange={(id, m) => { setId(id); setMember(m); }} activeOnly />
 *
 * // Form sửa: truyền initialMember để hiện tên ngay
 * <MemberSelect initialMember={{ id: r.member_id, full_name: r.member_name, member_code: r.member_code,
 *   phone: r.member_phone, avatar_url: '', status: 'ACTIVE' }} />
 * ```
 */
export function MemberSelect({
  value,
  onChange,
  branchId,
  initialMember,
  activeOnly,
  placeholder = 'Tìm hội viên theo tên, SĐT hoặc mã',
  ...rest
}: MemberSelectProps) {
  const [input, setInput] = useState('');
  const [keyword, setKeyword] = useState('');
  const [selected, setSelected] = useState<MemberOption | undefined>(initialMember ?? undefined);

  useEffect(() => {
    const t = setTimeout(() => setKeyword(input.trim()), DEBOUNCE_MS);
    return () => clearTimeout(t);
  }, [input]);

  useEffect(() => {
    if (initialMember) setSelected(initialMember);
  }, [initialMember]);

  const query = useQuery({
    queryKey: [memberApi.key, 'options', keyword, branchId ?? null],
    queryFn: () => memberApi.options(keyword, branchId),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });

  const list = useMemo(() => {
    const items = (query.data ?? []).filter((m) => !activeOnly || m.status === 'ACTIVE');
    // Giữ hội viên đang chọn trong danh sách để Select hiển thị đúng nhãn.
    if (selected && selected.id === value && !items.some((m) => m.id === selected.id)) {
      return [selected, ...items];
    }
    return items;
  }, [query.data, activeOnly, selected, value]);

  const options = list.map((m) => ({
    value: m.id,
    label: `${m.full_name} · ${m.member_code}`,
    member: m,
  }));

  return (
    <Select<number>
      showSearch
      allowClear
      placeholder={placeholder}
      value={value ?? undefined}
      filterOption={false}
      searchValue={input}
      onSearch={setInput}
      loading={query.isFetching}
      notFoundContent={
        query.isFetching ? <Spin size="small" /> : input ? 'Không tìm thấy hội viên' : 'Gõ để tìm'
      }
      optionLabelProp="label"
      options={options}
      optionRender={(opt) => {
        const m = (opt.data as { member: MemberOption }).member;
        return (
          <Space size={8} align="center">
            <MemberAvatar src={m.avatar_url} name={m.full_name} />
            <span style={{ lineHeight: 1.25 }}>
              <span>{m.full_name}</span>
              {m.status !== 'ACTIVE' && (
                <Typography.Text type="danger" style={{ fontSize: 12 }}>
                  {' '}
                  ({m.status === 'BANNED' ? 'Cấm' : 'Ngừng'})
                </Typography.Text>
              )}
              <br />
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {m.member_code} · {formatPhone(m.phone)}
              </Typography.Text>
            </span>
          </Space>
        );
      }}
      onChange={(id) => {
        const m = list.find((x) => x.id === id);
        setSelected(m);
        setInput('');
        onChange?.(id ?? undefined, m);
      }}
      style={{ width: '100%' }}
      {...rest}
    />
  );
}

export default MemberSelect;
