import { useMemo } from 'react';
import { Button, Card, Checkbox, Col, Empty, Input, Row, Space, Typography } from 'antd';
import { useState } from 'react';
import type { Permission } from '@/api/types';
import { PERMISSION_MODULE_LABELS } from '@/constants/permissions';
import { normalizeText } from '@/utils/format';

export interface PermissionPickerProps {
  permissions: Permission[];
  value?: string[];
  onChange?: (codes: string[]) => void;
  disabled?: boolean;
}

interface Group {
  module: string;
  label: string;
  items: Permission[];
}

/**
 * Cây checkbox 2 cấp: nhóm theo `module` (checkbox nhóm có trạng thái "một phần") → từng quyền.
 * Dùng như ô nhập của antd Form (value = mảng mã quyền).
 */
export default function PermissionPicker({
  permissions,
  value = [],
  onChange,
  disabled,
}: PermissionPickerProps) {
  const [keyword, setKeyword] = useState('');
  const selected = useMemo(() => new Set(value), [value]);

  const groups = useMemo<Group[]>(() => {
    const map = new Map<string, Permission[]>();
    permissions.forEach((p) => map.set(p.module, [...(map.get(p.module) ?? []), p]));
    return [...map.entries()].map(([module, items]) => ({
      module,
      label: PERMISSION_MODULE_LABELS[module] ?? module,
      items,
    }));
  }, [permissions]);

  const kw = normalizeText(keyword.trim());
  const visible = kw
    ? groups
        .map((g) => ({
          ...g,
          items: normalizeText(g.label).includes(kw)
            ? g.items
            : g.items.filter((p) => normalizeText(`${p.name} ${p.code}`).includes(kw)),
        }))
        .filter((g) => g.items.length > 0)
    : groups;

  const emit = (next: Set<string>) =>
    onChange?.(permissions.map((p) => p.code).filter((c) => next.has(c)));

  const toggleGroup = (g: Group, checked: boolean) => {
    const next = new Set(selected);
    g.items.forEach((p) => (checked ? next.add(p.code) : next.delete(p.code)));
    emit(next);
  };

  const toggleOne = (code: string, checked: boolean) => {
    const next = new Set(selected);
    if (checked) next.add(code);
    else next.delete(code);
    emit(next);
  };

  return (
    <div>
      <Space wrap style={{ marginBottom: 12, width: '100%', justifyContent: 'space-between' }}>
        <Input.Search
          allowClear
          placeholder="Tìm quyền..."
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          style={{ width: 260 }}
        />
        <Space size={4}>
          <Typography.Text type="secondary">
            Đã chọn {selected.size}/{permissions.length}
          </Typography.Text>
          <Button
            size="small"
            type="link"
            disabled={disabled}
            onClick={() => emit(new Set(permissions.map((p) => p.code)))}
          >
            Chọn tất cả
          </Button>
          <Button size="small" type="link" disabled={disabled} onClick={() => emit(new Set())}>
            Bỏ chọn
          </Button>
        </Space>
      </Space>

      {visible.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="Không có quyền phù hợp" />
      ) : (
        <Row gutter={[12, 12]}>
          {visible.map((g) => {
            const count = g.items.filter((p) => selected.has(p.code)).length;
            return (
              <Col xs={24} md={12} key={g.module}>
                <Card
                  size="small"
                  style={{ height: '100%', borderRadius: 10 }}
                  title={
                    <Checkbox
                      disabled={disabled}
                      checked={count === g.items.length}
                      indeterminate={count > 0 && count < g.items.length}
                      onChange={(e) => toggleGroup(g, e.target.checked)}
                    >
                      <b>{g.label}</b>{' '}
                      <Typography.Text type="secondary" style={{ fontWeight: 400 }}>
                        ({count}/{g.items.length})
                      </Typography.Text>
                    </Checkbox>
                  }
                >
                  <Space direction="vertical" size={4}>
                    {g.items.map((p) => (
                      <Checkbox
                        key={p.code}
                        disabled={disabled}
                        checked={selected.has(p.code)}
                        onChange={(e) => toggleOne(p.code, e.target.checked)}
                      >
                        {p.name}{' '}
                        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                          {p.code}
                        </Typography.Text>
                      </Checkbox>
                    ))}
                  </Space>
                </Card>
              </Col>
            );
          })}
        </Row>
      )}
    </div>
  );
}
