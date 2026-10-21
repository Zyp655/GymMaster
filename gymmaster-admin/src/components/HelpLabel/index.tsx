import type { ReactNode } from 'react';
import { Tooltip } from 'antd';
import { QuestionCircleOutlined } from '@ant-design/icons';

export interface HelpLabelProps {
  children: ReactNode;
  /** Giải thích cách tính (hiện khi rê chuột / chạm vào dấu ?). */
  help: ReactNode;
}

/**
 * Nhãn kèm dấu "?" giải thích cách tính số liệu – dùng cho StatCard/tiêu đề thẻ báo cáo.
 *   <StatCard label={<HelpLabel help={REVENUE_HELP.cash}>Doanh thu tháng</HelpLabel>} ... />
 */
export function HelpLabel({ children, help }: HelpLabelProps) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      {children}
      <Tooltip title={help}>
        <QuestionCircleOutlined
          aria-label="Giải thích cách tính"
          style={{ fontSize: 12, opacity: 0.65, cursor: 'help' }}
          onClick={(e) => e.stopPropagation()}
        />
      </Tooltip>
    </span>
  );
}

export default HelpLabel;
