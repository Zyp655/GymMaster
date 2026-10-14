import { useEffect } from 'react';
import { App as AntApp, ConfigProvider } from 'antd';
import viVN from 'antd/locale/vi_VN';
import { QueryClientProvider } from '@tanstack/react-query';
import { Provider } from 'react-redux';
import { BrowserRouter } from 'react-router-dom';
import MainApp from '@/container/MainApp';
import { queryClient } from '@/queryClient';
import { store } from '@/store';
import { theme } from '@/theme';
import { bindFeedback } from '@/utils/feedback';

/** Gắn message/modal/notification theo theme hiện tại cho code ngoài component. */
function FeedbackBridge() {
  const { message, modal, notification } = AntApp.useApp();
  useEffect(() => bindFeedback({ message, modal, notification }), [message, modal, notification]);
  return null;
}

export default function App() {
  return (
    <Provider store={store}>
      <QueryClientProvider client={queryClient}>
        <ConfigProvider locale={viVN} theme={theme}>
          <AntApp>
            <FeedbackBridge />
            <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
              <MainApp />
            </BrowserRouter>
          </AntApp>
        </ConfigProvider>
      </QueryClientProvider>
    </Provider>
  );
}
