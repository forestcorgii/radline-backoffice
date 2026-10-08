const CACHE_NAME = 'radline-v1';
const PRECACHE_ASSETS = [
  '/',
  '/offline.html',
  '/static/offline.html',
  '/static/index.css?v=5.2',
  '/static/index.css',
  '/static/htmx.min.js',
  '/manifest.webmanifest',
  '/static/manifest.webmanifest',
  '/static/icons/icon-192.png',
  '/static/icons/icon-512.png',
  '/static/icons/icon-maskable-192.png',
  '/static/icons/icon-maskable-512.png',
  '/static/icons/icon.svg'
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
          return new Response(`
            <div class="card" style="margin: 2rem auto; max-width: 500px; text-align: center; padding: 2rem;">
              <div style="font-size: 2.5rem; margin-bottom: 0.75rem;">📡</div>
              <h3 style="margin-bottom: 0.5rem; font-size: 1.15rem; font-weight: 600;">Connection Lost</h3>
              <p style="color: var(--text-muted); font-size: 0.9rem; margin-bottom: 1.25rem;">Unable to load data while offline. Please check your network connection.</p>
              <button onclick="window.location.reload()" class="btn" style="padding: 0.5rem 1.25rem;">Retry Connection</button>
            </div>
          `, {
            status: 200,
            headers: { 'Content-Type': 'text/html; charset=utf-8' }
          });
        })
    );
    return;
  }

  // 3. Static assets & fonts: Cache-first with network fallback & background update
  if (
    url.pathname.startsWith('/static/') ||
    url.hostname === 'fonts.googleapis.com' ||
    url.hostname === 'fonts.gstatic.com'
  ) {
    event.respondWith(
      caches.match(request).then((cached) => {
        if (cached) {
          return cached;
        }
        return fetch(request).then((response) => {
          if (response && response.status === 200) {
            const clone = response.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(request, clone));
          }
          return response;
        });
      })
    );
    return;
  }

  // 4. Default: Network-first
  event.respondWith(
    fetch(request).catch(() => caches.match(request))
  );
});
