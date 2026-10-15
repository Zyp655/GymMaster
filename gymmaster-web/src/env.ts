import { z } from 'zod';

/**
 * Biến môi trường dùng được ở cả client và server.
 * NEXT_PUBLIC_* phải được tham chiếu tĩnh để Next.js inline khi build.
 */
const clientSchema = z.object({
  NEXT_PUBLIC_API_URL: z.string().url().default('http://localhost:8000/v1'),
});

export const env = clientSchema.parse({
  NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || undefined,
});

const serverSchema = z.object({
  API_URL_INTERNAL: z.string().url().optional(),
  NEXTAUTH_URL: z.string().url().optional(),
  NEXTAUTH_SECRET: z.string().min(16, 'NEXTAUTH_SECRET phải có ít nhất 16 ký tự'),
});

type ServerEnv = z.infer<typeof serverSchema>;

let cachedServerEnv: ServerEnv | null = null;

/** Biến môi trường chỉ dùng phía server. Validate lười để `next build` không cần secret thật. */
export function getServerEnv(): ServerEnv {
  if (typeof window !== 'undefined') {
    throw new Error('getServerEnv() chỉ được gọi phía server');
  }
  if (cachedServerEnv) return cachedServerEnv;
  const isBuild = process.env.NEXT_PHASE === 'phase-production-build';
  cachedServerEnv = serverSchema.parse({
    API_URL_INTERNAL: process.env.API_URL_INTERNAL || undefined,
    NEXTAUTH_URL: process.env.NEXTAUTH_URL || undefined,
    NEXTAUTH_SECRET:
      process.env.NEXTAUTH_SECRET || (isBuild ? 'build-time-placeholder-secret' : undefined),
  });
  return cachedServerEnv;
}

/** URL backend: server ưu tiên URL nội bộ (docker network), client dùng URL public. */
export function getApiBaseUrl(): string {
  if (typeof window === 'undefined') {
    return getServerEnv().API_URL_INTERNAL ?? env.NEXT_PUBLIC_API_URL;
  }
  return env.NEXT_PUBLIC_API_URL;
}

/** URL trang web public dùng cho SEO sitemap / robots. */
export function getSiteUrl(): string {
  return process.env.NEXT_PUBLIC_SITE_URL || process.env.NEXTAUTH_URL || 'http://localhost:3000';
}
