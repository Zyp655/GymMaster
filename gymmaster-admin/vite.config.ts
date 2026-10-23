import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Tiện ích dùng chung của antd / rc-* / icons – gom một chunk nền để các chunk antd không vòng lẫn nhau.
const ANTD_BASE = new Set([
  'classnames',
  '@babel/runtime',
  '@ant-design/colors',
  '@ant-design/fast-color',
  '@ant-design/cssinjs',
  '@ant-design/cssinjs-utils',
  '@ant-design/icons-svg',
  '@ctrl/tinycolor',
  'stylis',
  'react-is',
  'resize-observer-polyfill',
  'throttle-debounce',
  'copy-to-clipboard',
  'toggle-selection',
  'compute-scroll-into-view',
  'scroll-into-view-if-needed',
  'json2mq',
  'string-convert',
]);

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: { port: 3001, strictPort: true },
  preview: { port: 3001 },
  build: {
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
      output: {
        // Tách vendor để không chunk nào > 1500 kB: antd tách riêng icons + rc-*; @ant-design/charts chỉ
        // nằm trong chunk tải lười (components/Charts) và lõi đồ họa @antv/* (trừ G2) được tách khỏi phần G2/plots.
        manualChunks(id) {
          // Helper nội bộ của Vite/Rollup dùng ở mọi chunk → để trong chunk luôn tải (không lọt vào chunk biểu đồ).
          if (id.includes('vite/preload-helper') || id.includes('commonjsHelpers')) return 'react';
          if (!id.includes('node_modules')) return undefined;
          const m = /node_modules\/((?:@[^/]+\/)?[^/]+)/.exec(
            id.slice(id.lastIndexOf('node_modules')),
          );
          const pkg = m?.[1] ?? '';
          if (['react', 'react-dom', 'react-router', 'react-router-dom', 'scheduler'].includes(pkg))
            return 'react';
          // Thứ tự phụ thuộc một chiều (tránh "Circular chunk"): antd → antd-rc/antd-icons → antd-base.
          if (ANTD_BASE.has(pkg) || pkg.startsWith('@emotion/') || pkg === 'rc-util')
            return 'antd-base';
          if (pkg === '@ant-design/icons') return 'antd-icons';
          if (pkg.startsWith('rc-') || pkg.startsWith('@rc-component/')) return 'antd-rc';
          if (pkg === 'antd' || pkg === '@ant-design/react-slick') return 'antd';
          // Lõi đồ họa AntV (renderer/scale/util) – chỉ là phụ thuộc của chunk biểu đồ tải lười.
          if (pkg.startsWith('@antv/') && !pkg.startsWith('@antv/g2')) return 'charts-vendor';
          if (
            ['@tanstack/react-query', '@reduxjs/toolkit', 'react-redux', 'axios', 'dayjs'].includes(
              pkg,
            )
          )
            return 'data';
          return undefined;
        },
      },
    },
  },
});
