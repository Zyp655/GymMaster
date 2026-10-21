/** Mô tả ngắn thiết bị từ User-Agent: "Chrome · macOS". Không nhận ra → null (caller dùng UA thô). */
export function describeUserAgent(ua: string | null | undefined): string | null {
  const s = (ua ?? '').trim();
  if (!s) return null;
  const browser = /Edg\//.test(s)
    ? 'Edge'
    : /OPR\/|Opera/.test(s)
      ? 'Opera'
      : /Firefox\//.test(s)
        ? 'Firefox'
        : /Chrome\//.test(s) || /CriOS\//.test(s)
          ? 'Chrome'
          : /Safari\//.test(s) && /Version\//.test(s)
            ? 'Safari'
            : null;
  const os = /iPhone|iPad|iPod/.test(s)
    ? 'iOS'
    : /Android/.test(s)
      ? 'Android'
      : /Windows/.test(s)
        ? 'Windows'
        : /Mac OS X|Macintosh/.test(s)
          ? 'macOS'
          : /Linux/.test(s)
            ? 'Linux'
            : null;
  if (!browser && !os) return null;
  return [browser, os].filter(Boolean).join(' · ');
}
