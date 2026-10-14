import { useMemo } from 'react';
import type { ReactNode } from 'react';
import { Button, Empty, Result, Table } from 'antd';
import type { TableProps } from 'antd';
import type { ColumnsType, SorterResult } from 'antd/es/table/interface';
import type { ListParams } from '@/api/types';
import { appConfig } from '@/config';
import type { TableQuery } from '@/hooks/useTableQuery';
import { getErrorMessage } from '@/utils/errors';

export interface DataTableProps<T> extends Omit<
  TableProps<T>,
  'dataSource' | 'pagination' | 'onChange' | 'loading' | 'columns'
> {
  /** Kết quả của `useTableQuery`. */
  table: TableQuery<T, ListParams>;
  /**
   * Cột antd. Cột có `sorter: true` sẽ sắp xếp phía server theo `key` (hoặc `dataIndex`),
   * và `sortOrder` được điều khiển từ URL.
   */
  columns: ColumnsType<T>;
  emptyText?: ReactNode;
}

function columnField<T>(col: ColumnsType<T>[number]): string | undefined {
  if ('dataIndex' in col && typeof col.dataIndex === 'string') return col.dataIndex;
  return typeof col.key === 'string' ? col.key : undefined;
}

/**
 * Bảng danh sách chuẩn: phân trang + sort server-side đồng bộ URL, size small, cuộn ngang,
 * trạng thái loading / rỗng / lỗi (có nút thử lại).
 *
 *   const table = useTableQuery({ queryKey: [branchApi.key, 'list'], queryFn: branchApi.list });
 *   <DataTable table={table} columns={columns} rowSelection={selection.rowSelection} />
 */
export function DataTable<T extends object>({
  table,
  columns,
  emptyText = 'Chưa có dữ liệu',
  rowKey = 'id',
  ...rest
}: DataTableProps<T>) {
  const controlledColumns = useMemo(
    () =>
      columns.map((col) => {
        const field = columnField(col);
        if (col.sorter === true && field) {
          return {
            ...col,
            sortOrder: table.sortOrderOf(field),
            sortDirections: ['ascend', 'descend'],
          };
        }
        return col;
      }) as ColumnsType<T>,
    [columns, table],
  );

  if (table.isError && table.items.length === 0) {
    return (
      <Result
        status="warning"
        title="Không tải được dữ liệu"
        subTitle={getErrorMessage(table.error)}
        extra={
          <Button type="primary" onClick={() => table.refetch()}>
            Thử lại
          </Button>
        }
      />
    );
  }

  const onChange: TableProps<T>['onChange'] = (pagination, _filters, sorter, extra) => {
    if (extra.action === 'paginate') {
      table.setPage(pagination.current ?? 1, pagination.pageSize);
      return;
    }
    if (extra.action === 'sort') {
      const s = (Array.isArray(sorter) ? sorter[0] : sorter) as SorterResult<T>;
      const field =
        (typeof s.columnKey === 'string' ? s.columnKey : undefined) ??
        (typeof s.field === 'string' ? s.field : undefined);
      table.setSort(
        field,
        s.order === 'ascend' ? 'ASC' : s.order === 'descend' ? 'DESC' : undefined,
      );
    }
  };

  return (
    <Table<T>
      size="small"
      tableLayout="auto"
      scroll={{ x: 'max-content' }}
      rowKey={rowKey}
      columns={controlledColumns}
      dataSource={table.items}
      loading={table.isFetching}
      onChange={onChange}
      locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={emptyText} /> }}
      pagination={{
        current: table.params.page,
        pageSize: table.params.limit,
        total: table.total,
        showSizeChanger: true,
        pageSizeOptions: appConfig.pageSizeOptions.map(String),
        showTotal: (total, range) => `${range[0]}–${range[1]} / ${total} bản ghi`,
        hideOnSinglePage: false,
      }}
      {...rest}
    />
  );
}

export default DataTable;
