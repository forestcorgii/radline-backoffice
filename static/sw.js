const CACHE_NAME = 'rav-v6.5';
const PRECACHE_ASSETS = [
  '/offline.html',
  '/static/index.css?v=6.5',
  '/static/htmx.min.js',
  '/static/js/base-ui.js?v=6.5',
  '/static/js/table-engine.js?v=6.5',
  '/manifest.webmanifest',
  '/static/icons/icon.svg',
  '/static/icons/icon-192.png',
  '/static/icons/icon-512.png'
];

// Install: Pre-cache core shell & offline assets
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(PRECACHE_ASSETS).catch((err) => {
        console.warn('[SW] Some precache assets failed to fetch:', err);
      });
    }).then(() => self.skipWaiting())
  );
});

// Activate: Clean up old cache versions and claim clients
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => {
      return Promise.all(
        keys.map((key) => {
          if (key !== CACHE_NAME) {
            return caches.delete(key);
          }
        })
      );
    }).then(() => self.clients.claim())
  );
});

// Fetch: Strategy depending on request type
self.addEventListener('fetch', (event) => {
  const request = event.request;

  // Only handle GET requests; mutating actions (POST, PUT, DELETE) must bypass cache
  if (request.method !== 'GET') {
    return;
  }

  const url = new URL(request.url);

  // 1. Navigation requests (Full page loads / address bar navigation)
  if (request.mode === 'navigate') {
    event.respondWith(
      fetch(request)
        .catch(async () => {
          const cache = await caches.open(CACHE_NAME);
          const cachedOffline = await cache.match('/offline.html') || await cache.match('/static/offline.html');
          return cachedOffline || new Response('Offline', { status: 503, statusText: 'Service Unavailable' });
        })
    );
    return;
  }

  // 2. HTMX partial page requests (hx-get swaps)
  if (request.headers.get('HX-Request') === 'true') {
    event.respondWith(
      fetch(request)
        .catch(async () => {
          return new Response('Offline — unable to complete request.', {
            status: 503,
            statusText: 'Service Unavailable',
            headers: { 'Content-Type': 'text/plain; charset=utf-8' }
          });
        })
    );
    return;
  }

  // 3. Static assets & fonts: Stale-while-revalidate (serve cache immediately, update in background)
  if (
    url.pathname.startsWith('/static/') ||
    url.hostname === 'fonts.googleapis.com' ||
    url.hostname === 'fonts.gstatic.com'
  ) {
    event.respondWith(
      caches.match(request).then((cached) => {
        const networkFetch = fetch(request).then((response) => {
          if (response && response.status === 200) {
            const clone = response.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(request, clone));
          }
          return response;
        }).catch(() => cached);
        return cached || networkFetch;
      })
    );
    return;
  }

  // 4. Default: Network-first
  event.respondWith(
    fetch(request).catch(() => caches.match(request))
  );
});
