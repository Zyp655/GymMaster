const CACHE_NAME = 'gymmaster-shell-v1';
const SHELL_URLS = ['/', '/me', '/me/qr'];

self.addEventListener('install', event => {
  self.skipWaiting();
  event.waitUntil(
    caches
      .open(CACHE_NAME)
      .then(cache => cache.addAll(SHELL_URLS))
      .catch(() => undefined),
  );
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches
      .keys()
      .then(keys =>
        Promise.all(
          keys
            .filter(key => key !== CACHE_NAME)
            .map(key => caches.delete(key)),
        ),
      )
      .then(() => self.clients.claim()),
  );
});

self.addEventListener('fetch', event => {
  const { request } = event;
  if (request.method !== 'GET') return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  if (url.pathname.startsWith('/api/')) return;

  event.respondWith(
    fetch(request)
      .then(response => {
        if (
          response.ok &&
          (request.mode === 'navigate' || url.pathname.startsWith('/_next/static/'))
        ) {
          const clone = response.clone();
          void caches.open(CACHE_NAME).then(cache => cache.put(request, clone));
        }
        return response;
      })
      .catch(async () => {
        const cached = await caches.match(request);
        if (cached) return cached;
        if (request.mode === 'navigate') {
          const fallback = await caches.match('/me/qr');
          if (fallback) return fallback;
        }
        throw new Error('Network error');
      }),
  );
});
