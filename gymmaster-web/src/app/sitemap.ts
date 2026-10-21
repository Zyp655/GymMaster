import type { MetadataRoute } from 'next';
import { getSiteUrl } from '@/env';
import { landingApi } from '@/features/landing/api';
import { postHref } from '@/features/landing/post-slug';

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const baseUrl = getSiteUrl();
  const now = new Date();
  // Bài viết đã xuất bản (API công khai chỉ trả PUBLISHED); lỗi API → bỏ qua phần bài viết.
  const postsRes = await landingApi.posts();
  const postEntries: MetadataRoute.Sitemap = postsRes.ok
    ? postsRes.data.map(post => ({
        url: `${baseUrl}${postHref(post.slug)}`,
        lastModified: new Date(post.published_at || post.created_at),
        changeFrequency: 'monthly' as const,
        priority: 0.7,
      }))
    : [];

  return [
    {
      url: `${baseUrl}/`,
      lastModified: now,
      changeFrequency: 'daily',
      priority: 1,
    },
    {
      url: `${baseUrl}/trainers`,
      lastModified: now,
      changeFrequency: 'weekly',
      priority: 0.85,
    },
    {
      url: `${baseUrl}/blog`,
      lastModified: now,
      changeFrequency: 'daily',
      priority: 0.8,
    },
    {
      url: `${baseUrl}/login`,
      lastModified: now,
      changeFrequency: 'monthly',
      priority: 0.5,
    },
    {
      url: `${baseUrl}/register`,
      lastModified: now,
      changeFrequency: 'monthly',
      priority: 0.6,
    },
    ...postEntries,
  ];
}
