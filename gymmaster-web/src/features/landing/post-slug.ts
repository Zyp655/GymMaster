/** Slug bài viết hợp lệ (khớp quy tắc API): chữ thường a-z, số, '-' ở giữa; 3–180 ký tự. */
export function isValidPostSlug(slug: string | null | undefined): slug is string {
  return (
    typeof slug === 'string' &&
    slug.length >= 3 &&
    slug.length <= 180 &&
    /^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug)
  );
}

/** Đường dẫn trang chi tiết bài viết. */
export function postHref(slug: string): string {
  return `/blog/${encodeURIComponent(slug)}`;
}
