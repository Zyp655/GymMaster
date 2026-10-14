import type { MetadataRoute } from 'next';

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: 'GymMaster — Cổng hội viên',
    short_name: 'GymMaster',
    description: 'Hệ thống số hóa quy trình quản lý trung tâm Fitness',
    start_url: '/me',
    display: 'standalone',
    background_color: '#f8fafc',
    theme_color: '#ea580c',
    icons: [
      {
        src: '/icon.svg',
        sizes: 'any',
        type: 'image/svg+xml',
      },
    ],
  };
}
