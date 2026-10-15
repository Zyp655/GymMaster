import { createContext, useContext, useEffect, useMemo, useState } from 'react';
import { App as AntApp, ConfigProvider, theme as antdTheme } from 'antd';
import viVN from 'antd/locale/vi_VN';
import { QueryClientProvider } from '@tanstack/react-query';
import { Provider } from 'react-redux';
import { BrowserRouter } from 'react-router-dom';
import MainApp from '@/container/MainApp';
import { queryClient } from '@/queryClient';
import { store } from '@/store';
import { theme } from '@/theme';
import { bindFeedback } from '@/utils/feedback';

interface ThemeModeCtx {
  darkMode: boolean;
  toggleDarkMode: () => void;
}

export const ThemeModeContext = createContext<ThemeModeCtx>({
  darkMode: false,
  toggleDarkMode: () => {},
});

export const useThemeMode = () => useContext(ThemeModeContext);

/** Gắn message/modal/notification theo theme hiện tại cho code ngoài component. */
function FeedbackBridge() {
  const { message, modal, notification } = AntApp.useApp();
  useEffect(() => bindFeedback({ message, modal, notification }), [message, modal, notification]);
  return null;
}

export default function App() {
  const [darkMode, setDarkMode] = useState<boolean>(() => {
    try {
      return localStorage.getItem('gymmaster_admin_dark') === '1';
    } catch {
      return false;
    }
  });

  const toggleDarkMode = () => {
    setDarkMode((prev) => {
      const next = !prev;
      try {
        localStorage.setItem('gymmaster_admin_dark', next ? '1' : '0');
      } catch {
        // ignore
      }
      return next;
    });
  };

  const mergedTheme = useMemo(
    () => ({
      ...theme,
      algorithm: darkMode ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    }),
    [darkMode],
  );

  return (
    <Provider store={store}>
      <QueryClientProvider client={queryClient}>
        <ThemeModeContext.Provider value={{ darkMode, toggleDarkMode }}>
          <ConfigProvider locale={viVN} theme={mergedTheme}>
            <AntApp>
              <FeedbackBridge />
              <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
                <MainApp />
              </BrowserRouter>
            </AntApp>
          </ConfigProvider>
        </ThemeModeContext.Provider>
      </QueryClientProvider>
    </Provider>
  );
}
