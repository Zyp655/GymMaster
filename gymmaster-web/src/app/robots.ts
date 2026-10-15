import type { MetadataRoute } from 'next';
import { getSiteUrl } from '@/env';

export default function robots(): MetadataRoute.Robots {
  const baseUrl = getSiteUrl();
  return {
    rules: [
      {
        userAgent: '*',
        allow: ['/', '/trainers', '/blog', '/login', '/register'],
        disallow: ['/me/', '/kiosk', '/api/'],
      },
    ],
    sitemap: `${baseUrl}/sitemap.xml`,
  };
}
