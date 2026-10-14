import { useCallback, useMemo, useState } from 'react';
import type { Key } from 'react';
import type { TableRowSelection } from 'antd/es/table/interface';

/**
 * Chọn nhiều dòng cho bảng:
 *   const selection = useRowSelection<Branch>();
 *   <DataTable rowSelection={selection.rowSelection} ... />
 *   <ActionBar selectedCount={selection.keys.length} onDelete={() => remove(selection.keys)} />
 */
export function useRowSelection<T>(getCheckboxProps?: TableRowSelection<T>['getCheckboxProps']) {
  const [keys, setKeys] = useState<Key[]>([]);
  const [rows, setRows] = useState<T[]>([]);

  const clear = useCallback(() => {
    setKeys([]);
    setRows([]);
  }, []);

  const rowSelection = useMemo<TableRowSelection<T>>(
    () => ({
      selectedRowKeys: keys,
      preserveSelectedRowKeys: true,
      onChange: (nextKeys, nextRows) => {
        setKeys(nextKeys);
        setRows(nextRows);
      },
      getCheckboxProps,
    }),
    [keys, getCheckboxProps],
  );

  return { keys, rows, setKeys, clear, rowSelection };
}

export default useRowSelection;
