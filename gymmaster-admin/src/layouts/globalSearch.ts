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

/**
 * Chỉ số kết quả đang chọn sau khi bấm phím điều hướng (↑/↓/Home/End) – vòng tròn ở hai đầu.
 * Trả `-1` khi danh sách rỗng; phím khác giữ nguyên.
 */
export function nextActiveIndex(current: number, key: string, length: number): number {
  if (length <= 0) return -1;
  switch (key) {
    case 'ArrowDown':
      return current < 0 || current >= length - 1 ? 0 : current + 1;
    case 'ArrowUp':
      return current <= 0 ? length - 1 : current - 1;
    case 'Home':
      return 0;
    case 'End':
      return length - 1;
    default:
      return Math.min(current, length - 1);
  }
}
