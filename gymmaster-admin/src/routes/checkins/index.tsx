import { useCallback } from 'react';
import { Card, Col, DatePicker, Row } from 'antd';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import checkinApi, {
  CHECKIN_METHOD_LABEL,
  CHECKIN_METHOD_OPTIONS,
  CHECKIN_RESULT_MAP,
  CHECKIN_RESULT_OPTIONS,
} from '@/api/checkin';
import type { CheckinItem, CheckinListParams } from '@/api/checkin';
import ActionBar from '@/components/ActionBar';
import DataTable from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import MobileList from '@/components/MobileList';
import type { MobileFetchArgs } from '@/components/MobileList';
import PageTitle from '@/components/PageTitle';
import { useBranchOptions } from '@/hooks/useBranchOptions';
import { useIsMobile } from '@/hooks/useIsMobile';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useUrlParams } from '@/hooks/useUrlParams';
import { useWorkingBranch } from '@/hooks/useWorkingBranch';
import { DATE_FORMAT, formatDateTime, toApiDate } from '@/utils/format';
import { checkinColumns } from './columns';

const TITLE = 'Lịch sử ra vào';
const SUBTITLE = 'Mọi lượt quét check-in: cho vào, từ chối, checkout';

function CheckinsDesktop() {
  const url = useUrlParams();
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();
  const urlBranch = url.get('branch_id');

  const table = useTableQuery<CheckinItem, CheckinListParams>({
    queryKey: [checkinApi.key, 'list'],
    queryFn: (p) => checkinApi.list(p),
    filterKeys: ['branch_id', 'result', 'method', 'from', 'to'],
    numberKeys: ['branch_id'],
    defaults: { order_by: 'checked_in_at', order_dir: 'DESC' },
    extraParams: { branch_id: urlBranch ? undefined : (branchId ?? undefined) },
  });

  const from = url.get('from');
  const to = url.get('to');

  return (
    <Card variant="borderless" className="page-card">
      <Row justify="space-between" align="middle" gutter={[8, 12]} className="page-head">
        <Col>
          <PageTitle title={TITLE} subtitle={SUBTITLE} />
        </Col>
        <Col>
          <ActionBar onRefresh={() => table.refetch()} refreshing={table.isFetching} />
        </Col>
      </Row>

      <FilterBar
        searchPlaceholder="Tên, mã hoặc SĐT hội viên"
        colTextSearch={5}
        filters={[
          {
            name: 'branch_id',
            placeholder: 'Chi nhánh',
            options: branches.map((b) => ({ label: b.name, value: b.id })),
          },
          { name: 'result', placeholder: 'Kết quả', options: CHECKIN_RESULT_OPTIONS, col: 3 },
          { name: 'method', placeholder: 'Phương thức', options: CHECKIN_METHOD_OPTIONS, col: 3 },
        ]}
      >
        <Col xs={24} sm={12} md={6}>
          <DatePicker.RangePicker
            style={{ width: '100%' }}
            format={DATE_FORMAT}
            placeholder={['Từ ngày', 'Đến ngày']}
            value={from || to ? [from ? dayjs(from) : null, to ? dayjs(to) : null] : null}
            allowEmpty={[true, true]}
            presets={[
              { label: 'Hôm nay', value: [dayjs(), dayjs()] },
              { label: '7 ngày qua', value: [dayjs().subtract(6, 'day'), dayjs()] },
              { label: 'Tháng này', value: [dayjs().startOf('month'), dayjs()] },
            ]}
            onChange={(v) =>
              url.set(
                { from: toApiDate(v?.[0]) ?? undefined, to: toApiDate(v?.[1]) ?? undefined },
                { resetPage: true },
              )
            }
          />
        </Col>
      </FilterBar>

      <DataTable<CheckinItem>
        table={table}
        columns={checkinColumns()}
        emptyText="Chưa có lượt check-in nào"
      />
    </Card>
  );
}

function CheckinsMobile() {
  const { branchId } = useWorkingBranch();
  const { branches } = useBranchOptions();

  const fetchPage = useCallback(
    ({ page, limit, keyword, filters }: MobileFetchArgs) => {
      const f = filters as Record<string, unknown>;
      const range = f.range as [Dayjs | null, Dayjs | null] | undefined;
      return checkinApi.list({
        page,
        limit,
        keyword: keyword || undefined,
        branch_id: (f.branch_id as number | undefined) ?? branchId ?? undefined,
        result: (f.result as CheckinListParams['result']) || undefined,
        method: (f.method as CheckinListParams['method']) || undefined,
        from: toApiDate(range?.[0]) ?? undefined,
        to: toApiDate(range?.[1]) ?? undefined,
      });
    },
    [branchId],
  );

  return (
    <MobileList<CheckinItem>
      title={TITLE}
      queryKey={[checkinApi.key, 'list', branchId]}
      fetchPage={fetchPage}
      searchPlaceholder="Tên, mã, SĐT hội viên"
      emptyText="Chưa có lượt check-in nào"
      filters={[
        {
          name: 'branch_id',
          label: 'Chi nhánh',
          options: branches.map((b) => ({ label: b.name, value: b.id })),
        },
        { name: 'result', label: 'Kết quả', options: CHECKIN_RESULT_OPTIONS },
        { name: 'method', label: 'Phương thức', options: CHECKIN_METHOD_OPTIONS },
        { name: 'range', label: 'Khoảng ngày', type: 'daterange' },
      ]}
      row={{
        thumb: (r) => r.member_avatar_url,
        avatar: true,
        title: (r) => r.member_name,
        subtitle: (r) => (r.result === 'DENIED' ? r.deny_message : (r.plan_name ?? '')),
        meta: (r) => [formatDateTime(r.checked_in_at), r.member_code, r.branch_name],
        tags: (r) => [
          { label: CHECKIN_RESULT_MAP[r.result].label, color: CHECKIN_RESULT_MAP[r.result].color },
          { label: CHECKIN_METHOD_LABEL[r.method] },
        ],
      }}
      details={(r) => [
        ['Thời điểm', formatDateTime(r.checked_in_at)],
        ['Ra về', r.checked_out_at ? formatDateTime(r.checked_out_at) : '—'],
        ['Chi nhánh', r.branch_name],
        ['Nhân viên', r.staff_name ?? '—'],
        ['Ghi chú', r.note || '—'],
      ]}
    />
  );
}

export default function CheckinsPage() {
  return useIsMobile() ? <CheckinsMobile /> : <CheckinsDesktop />;
}
