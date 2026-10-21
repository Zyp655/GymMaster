import type { SearchHit } from '@/api/cms-system';
import { http } from '@/utils/api';

/** Độ trễ gõ phím trước khi gọi API tìm kiếm nhanh (Ctrl+K). */
export const GLOBAL_SEARCH_DEBOUNCE_MS = 250;

/** Từ khóa đủ điều kiện tìm (đã trim) hoặc `null` nếu rỗng. */
export function searchTerm(text: string): string | null {
  const t = text.trim();
  return t.length >= 1 ? t : null;
}

/**
 * Tìm kiếm nhanh toàn hệ thống – nhận `signal` của TanStack Query để hủy request cũ khi từ khóa
 * đổi / modal đóng (`cmsSystemApi.globalSearch` không hỗ trợ signal).
 * `silent`: request bị hủy bị interceptor coi là lỗi mạng → không bật toast; lỗi thật hiện ngay
 * trong modal (có nút "Thử lại").
 */
export function globalSearch(q: string, signal?: AbortSignal) {
  return http.get<{ items: SearchHit[] }>('/admin/search', { params: { q }, signal, silent: true });
}
