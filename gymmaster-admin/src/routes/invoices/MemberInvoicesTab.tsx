import { useMemo, useState } from 'react';
import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { Table } from 'antd';
import invoiceApi from '@/api/invoice';
import type { Invoice } from '@/api/invoice';
import type { Member } from '@/api/member';
import { getErrorMessage } from '@/utils/errors';
import { invoiceColumns } from './columns';

/** Tab "Hóa đơn" của hồ sơ hội viên – `GET /admin/invoices?member_id=`. */
export default function MemberInvoicesTab({ member }: { member: Member }) {
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(10);
  const query = useQuery({
    queryKey: [invoiceApi.key, 'list', { member_id: member.id, page, limit }],
    queryFn: () =>
      invoiceApi.list({ member_id: member.id, page, limit, order_by: 'id', order_dir: 'DESC' }),
    placeholderData: keepPreviousData,
  });
  const columns = useMemo(
    () => invoiceColumns({ showMember: false }).map((c) => ({ ...c, sorter: false })),
    [],
  );

  return (
    <Table<Invoice>
      size="small"
      rowKey="id"
      loading={query.isFetching}
      columns={columns}
      dataSource={query.data?.items ?? []}
      scroll={{ x: 'max-content' }}
      locale={{
        emptyText: query.isError
          ? getErrorMessage(query.error, 'Không tải được hóa đơn')
          : 'Hội viên chưa có hóa đơn nào',
      }}
      pagination={{
        current: page,
        pageSize: limit,
        total: query.data?.total ?? 0,
        showSizeChanger: true,
        showTotal: (t) => `${t} hóa đơn`,
        onChange: (p, l) => {
          setPage(l !== limit ? 1 : p);
          setLimit(l);
        },
      }}
    />
  );
}
