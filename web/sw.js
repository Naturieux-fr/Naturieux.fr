// Naturieux service worker: makes the app installable and serves a cached
// shell offline. API and media are always fetched live (never cached).
const CACHE = 'naturieux-shell-v1';

self.addEventListener('install', (e) => {
  e.waitUntil(caches.open(CACHE).then((c) => c.add('/')).then(() => self.skipWaiting()));
});

self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (e) => {
  const url = new URL(e.request.url);
  if (e.request.method !== 'GET') return;
  // Live data: quiz/account/media must never be served stale.
  if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/media/')) return;

  e.respondWith(
    caches.match(e.request).then((hit) =>
      hit ||
      fetch(e.request)
        .then((resp) => {
          if (url.pathname.startsWith('/static/') || url.pathname === '/') {
            const copy = resp.clone();
            caches.open(CACHE).then((c) => c.put(e.request, copy));
          }
          return resp;
        })
        .catch(() => caches.match('/'))
    )
  );
});
