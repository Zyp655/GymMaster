import { createContext, useContext } from 'react';
import type { ThemeConfig } from 'antd';

export interface ThemeModeCtx {
  darkMode: boolean;
  toggleDarkMode: () => void;
}

export const ThemeModeContext = createContext<ThemeModeCtx>({
  darkMode: false,
  toggleDarkMode: () => {},
});

export const useThemeMode = () => useContext(ThemeModeContext);

/** Màu thương hiệu GymMaster. */
export const BRAND = {
  /** Navy – màu chính (giữ từ strongbody_admin). */
  primary: '#0A3150',
  /** Cam GymMaster – nút "Thêm mới" / CTA nổi bật: `<Button color="orange" variant="solid">`. */
  accent: '#F97316',
  /** Biến thể sáng hơn của navy cho chế độ tối. */
  primaryDark: '#2F7BC0',
  bg: '#F3F5F8',
} as const;

/** Theme antd theo chế độ sáng/tối – màu nền lấy từ token của thuật toán (không cố định trắng). */
export function getTheme(darkMode = false): ThemeConfig {
  return {
    token: {
      // Nền tối: navy thương hiệu quá tối → dùng sắc xanh sáng hơn để đủ tương phản.
      colorPrimary: darkMode ? BRAND.primaryDark : BRAND.primary,
      colorLink: darkMode ? '#6AA8E0' : BRAND.primary,
      // Không dùng navy cho info: palette sinh từ navy quá tối → Alert info nền xám đậm (C11).
      colorInfo: darkMode ? BRAND.primaryDark : '#1D6FB8',
      // Seed màu preset "orange" = cam thương hiệu → Button color="orange" đúng màu accent.
      orange: BRAND.accent,
      borderRadius: 8,
      fontFamily:
        "'Be Vietnam Pro', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
      ...(darkMode ? {} : { colorBgLayout: BRAND.bg }),
    },
    components: {
      Layout: {
        headerBg: darkMode ? '#141414' : '#ffffff',
        siderBg: darkMode ? '#141414' : '#ffffff',
        headerPadding: '0 16px',
        headerHeight: 60,
        footerBg: 'transparent',
        footerPadding: '12px 16px 20px',
      },
      Menu: {
        itemSelectedBg: darkMode ? BRAND.primaryDark : BRAND.primary,
        itemSelectedColor: '#ffffff',
        itemBorderRadius: 8,
        itemMarginInline: 8,
        ...(darkMode ? { subMenuItemSelectedColor: '#7CB3E6' } : { subMenuItemBg: '#ffffff' }),
      },
      Switch: { colorPrimary: '#12805C', colorPrimaryHover: '#0f6b4d' },
      Table: darkMode ? {} : { headerBg: '#F7F8FA', headerColor: '#4B5563' },
      Card: { borderRadiusLG: 12 },
    },
  };
}

/** Theme sáng mặc định (giữ cho mã cũ). */
export const theme: ThemeConfig = getTheme(false);
