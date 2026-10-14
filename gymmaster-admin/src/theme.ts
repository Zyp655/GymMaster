import type { ThemeConfig } from 'antd';

/** Màu thương hiệu GymMaster. */
export const BRAND = {
  /** Navy – màu chính (giữ từ strongbody_admin). */
  primary: '#0A3150',
  /** Cam GymMaster – nút "Thêm mới" / CTA nổi bật: `<Button color="orange" variant="solid">`. */
  accent: '#F97316',
  bg: '#F3F5F8',
} as const;

export const theme: ThemeConfig = {
  token: {
    colorPrimary: BRAND.primary,
    colorLink: BRAND.primary,
    colorInfo: BRAND.primary,
    // Seed màu preset "orange" = cam thương hiệu → Button color="orange" đúng màu accent.
    orange: BRAND.accent,
    borderRadius: 8,
    fontFamily:
      "'Be Vietnam Pro', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    colorBgLayout: BRAND.bg,
  },
  components: {
    Layout: {
      headerBg: '#ffffff',
      siderBg: '#ffffff',
      headerPadding: '0 16px',
      headerHeight: 60,
      footerBg: 'transparent',
      footerPadding: '12px 16px 20px',
    },
    Menu: {
      itemSelectedBg: BRAND.primary,
      itemSelectedColor: '#ffffff',
      itemBorderRadius: 8,
      itemMarginInline: 8,
      subMenuItemBg: '#ffffff',
    },
    Switch: { colorPrimary: '#12805C', colorPrimaryHover: '#0f6b4d' },
    Table: { headerBg: '#F7F8FA', headerColor: '#4B5563' },
    Card: { borderRadiusLG: 12 },
  },
};
