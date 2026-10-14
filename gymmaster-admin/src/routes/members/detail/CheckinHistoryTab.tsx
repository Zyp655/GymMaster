import { useState } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { Table } from 'antd';
import checkinApi from '@/api/checkin';
import type { CheckinItem } from '@/api/checkin';
import type { Member } from '@/api/member';
import { getErrorMessage } from '@/utils/errors';
import { checkinColumns } from '../../checkins/columns';

/** Tab "Lịch sử check-in" của hồ sơ hội viên – `GET /admin/checkins?member_id=`. */
export default function CheckinHistoryTab({ member }: { member: Member }) {
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const query = useQuery({
    queryKey: [checkinApi.key, 'list', { member_id: member.id, page, limit }],
    queryFn: () => checkinApi.list({ member_id: member.id, page, limit }),
    placeholderData: keepPreviousData,
  });

  return (
    <Table<CheckinItem>
      size="small"
      rowKey="id"
      loading={query.isFetching}
      columns={checkinColumns({ showMember: false }).map((c) => ({ ...c, sorter: false }))}
      dataSource={query.data?.items ?? []}
      scroll={{ x: 'max-content' }}
      locale={{
        emptyText: query.isError
          ? getErrorMessage(query.error, 'Không tải được lịch sử')
          : 'Chưa có lượt check-in nào',
      }}
      pagination={{
        current: page,
        pageSize: limit,
        total: query.data?.total ?? 0,
        showSizeChanger: true,
        showTotal: (t) => `${t} lượt`,
        onChange: (p, l) => {
          setPage(l !== limit ? 1 : p);
          setLimit(l);
        },
      }}
    />
  );
}
