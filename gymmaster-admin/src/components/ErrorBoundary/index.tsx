import { Component } from 'react';
import type { ErrorInfo, ReactNode } from 'react';
import { Button, Result } from 'antd';

interface Props {
  children: ReactNode;
  /** Đổi key (ví dụ pathname) để tự reset khi chuyển trang. */
  resetKey?: string;
}

interface State {
  error: Error | null;
}

/** Bắt lỗi render của một trang để không làm trắng toàn bộ ứng dụng. */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Lỗi giao diện:', error, info.componentStack);
  }

  componentDidUpdate(prev: Props) {
    if (prev.resetKey !== this.props.resetKey && this.state.error) this.setState({ error: null });
  }

  render() {
    const { error } = this.state;
    if (!error) return this.props.children;
    const chunkError = /dynamically imported module|Loading chunk/i.test(error.message);
    return (
      <Result
        status="error"
        title={chunkError ? 'Ứng dụng vừa được cập nhật' : 'Đã có lỗi khi hiển thị trang'}
        subTitle={
          chunkError
            ? 'Vui lòng tải lại trang để dùng phiên bản mới nhất.'
            : 'Bạn hãy thử tải lại trang. Nếu lỗi vẫn tiếp diễn, vui lòng báo cho quản trị viên.'
        }
        extra={
          <Button type="primary" onClick={() => window.location.reload()}>
            Tải lại trang
          </Button>
        }
      />
    );
  }
}

export default ErrorBoundary;
