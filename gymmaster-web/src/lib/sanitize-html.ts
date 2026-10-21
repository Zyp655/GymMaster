/**
 * Bộ lọc HTML theo danh sách cho phép (allowlist) – chạy trên trình duyệt bằng DOMParser
 * (tài liệu "inert": không chạy script, không tải ảnh). Dùng cho nội dung HTML do API dựng sẵn
 * (VD: hợp đồng `rendered_html`) trước khi đưa vào `dangerouslySetInnerHTML`.
 */

const ALLOWED_TAGS = new Set([
  'a',
  'b',
  'blockquote',
  'br',
  'caption',
  'code',
  'col',
  'colgroup',
  'dd',
  'div',
  'dl',
  'dt',
  'em',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'hr',
  'i',
  'img',
  'li',
  'ol',
  'p',
  'pre',
  's',
  'small',
  'span',
  'strong',
  'sub',
  'sup',
  'table',
  'tbody',
  'td',
  'tfoot',
  'th',
  'thead',
  'tr',
  'u',
  'ul',
]);

/** Thẻ bị bỏ cả nội dung bên trong (không chỉ gỡ thẻ). */
const DROP_WITH_CONTENT = new Set([
  'script',
  'style',
  'iframe',
  'object',
  'embed',
  'noscript',
  'template',
  'svg',
  'math',
  'form',
  'input',
  'button',
  'select',
  'textarea',
  'link',
  'meta',
  'base',
  'title',
  'head',
]);

const GLOBAL_ATTRS = new Set(['title', 'colspan', 'rowspan', 'align']);
const TAG_ATTRS: Record<string, Set<string>> = {
  a: new Set(['href']),
  img: new Set(['src', 'alt', 'width', 'height']),
  th: new Set(['scope']),
};

// Đường dẫn tương đối chỉ chấp nhận "/" đơn – chặn "//host" và "/\\host" (trình duyệt coi "\\" như "/").
const SAFE_URL = /^(https?:|mailto:|tel:|#|\/(?![/\\]))/i;
const SAFE_IMG = /^(https?:|\/(?![/\\])|data:image\/(png|jpe?g|gif|webp);base64,)/i;

/**
 * Chuẩn hóa URL trước khi kiểm tra: trình duyệt bỏ qua tab/xuống dòng trong URL
 * ("/\t/evil.com" → "//evil.com"), nên gỡ chúng trước. URL chứa "\\" bị từ chối hẳn.
 */
export function safeUrl(value: string, kind: 'href' | 'src' = 'href'): string | null {
  const normalized = value.replace(/[\u0000-\u001F\u007F]/g, '').trim();
  if (!normalized || normalized.includes('\\')) return null;
  return (kind === 'href' ? SAFE_URL : SAFE_IMG).test(normalized) ? normalized : null;
}

function cleanNode(node: Node, doc: Document): void {
  for (const child of Array.from(node.childNodes)) {
    if (child.nodeType === Node.TEXT_NODE) continue;
    if (child.nodeType !== Node.ELEMENT_NODE) {
      child.remove();
      continue;
    }
    const el = child as Element;
    const tag = el.tagName.toLowerCase();
    if (DROP_WITH_CONTENT.has(tag)) {
      el.remove();
      continue;
    }
    cleanNode(el, doc);
    if (!ALLOWED_TAGS.has(tag)) {
      // Gỡ thẻ, giữ phần chữ bên trong.
      const frag = doc.createDocumentFragment();
      while (el.firstChild) frag.appendChild(el.firstChild);
      el.replaceWith(frag);
      continue;
    }
    for (const attr of Array.from(el.attributes)) {
      const name = attr.name.toLowerCase();
      const value = attr.value.trim();
      const allowed = GLOBAL_ATTRS.has(name) || TAG_ATTRS[tag]?.has(name);
      if (!allowed) {
        el.removeAttribute(attr.name);
      } else if (name === 'href' || name === 'src') {
        const cleaned = safeUrl(value, name);
        if (cleaned === null) el.removeAttribute(attr.name);
        else el.setAttribute(attr.name, cleaned);
      }
    }
    if (tag === 'a') {
      el.setAttribute('target', '_blank');
      el.setAttribute('rel', 'noopener noreferrer nofollow');
    }
  }
}

/** Trả về HTML đã lọc; trên server (không có DOMParser) trả `null` để caller hiển thị dạng chữ. */
export function sanitizeHtml(html: string): string | null {
  if (typeof window === 'undefined' || typeof DOMParser === 'undefined') return null;
  const doc = new DOMParser().parseFromString(`<body>${html}</body>`, 'text/html');
  cleanNode(doc.body, doc);
  return doc.body.innerHTML;
}

/** Nhận diện chuỗi có chứa thẻ HTML. */
export function looksLikeHtml(text: string): boolean {
  return /<\/?[a-z][\s\S]*?>/i.test(text);
}

/** Bỏ toàn bộ thẻ – phương án dự phòng khi chưa thể lọc (SSR). */
export function stripTags(html: string): string {
  return html
    .replace(/<(script|style)[\s\S]*?<\/\1>/gi, '')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/(p|div|h[1-6]|li|tr)>/gi, '\n')
    .replace(/<[^>]+>/g, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
}
