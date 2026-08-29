// Service Worker — StatGate Report Builder PWA
// Strategies:
//   Static assets (css/js/images): cache-first, network-fallback
//   CDN libs (Chart.js, Leaflet, html2canvas): cache-first, network-fallback
//   HTML pages: network-first, cache-fallback
//   API calls (/api/*): network-only (offline handled by offline-reports cache)

const CACHE_VERSION = 'v14';
const STATIC_CACHE = 'static-' + CACHE_VERSION;
const CDN_CACHE = 'cdn-' + CACHE_VERSION;
const HTML_CACHE = 'html-' + CACHE_VERSION;
const OFFLINE_CACHE = 'offline-reports';

// Static assets to pre-cache on install (NOT geojson — 9.6MB, cached on first use)
const PRECACHE_ASSETS = [
    './',
    'css/style.css',
    'css/Aimara.css',
    'css/_buttons.css',
    'js/app.js',
    'js/apiCache.js',
    'js/components.js',
    'js/filterManager.js',
    'js/pwa.js',
    'js/auth.js',
    'lib/Aimara.js',
    'favicon.png',
    'assets/logo.png',
    'assets/icon-192.png',
    'assets/icon-512.png'
];

// CDN domains to cache
const CDN_HOSTS = [
    'cdn.jsdelivr.net',
    'cdnjs.cloudflare.com'
];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(STATIC_CACHE).then((cache) => {
            // Resolve paths relative to service worker scope
            const scope = self.registration.scope;
            const urls = PRECACHE_ASSETS.map(path => new URL(path, scope).href);
            return cache.addAll(urls).catch((err) => {
                console.warn('SW: Some assets failed to precache:', err);
            });
        })
    );
    // Activate immediately without waiting for existing tabs to close
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((cacheNames) => {
            return Promise.all(
                cacheNames.map((name) => {
                    // Delete old versioned caches but keep offline-reports
                    if (name === OFFLINE_CACHE) return;
                    if (name !== STATIC_CACHE && name !== CDN_CACHE && name !== HTML_CACHE) {
                        console.log('SW: Deleting old cache:', name);
                        return caches.delete(name);
                    }
                })
            );
        }).then(() => self.clients.claim())
    );
});

self.addEventListener('fetch', (event) => {
    const url = new URL(event.request.url);

    // Skip non-GET requests
    if (event.request.method !== 'GET') return;

    // API calls: network-only, with offline-reports fallback for report data
    if (url.pathname.includes('/api/')) {
        event.respondWith(handleAPIRequest(event.request));
        return;
    }

    // CDN requests: cache-first
    if (CDN_HOSTS.some(host => url.hostname === host)) {
        event.respondWith(cacheFirst(event.request, CDN_CACHE));
        return;
    }

    // HTML pages: network-first
    if (event.request.mode === 'navigate' ||
        url.pathname.endsWith('.html') ||
        url.pathname.endsWith('/')) {
        event.respondWith(networkFirst(event.request, HTML_CACHE));
        return;
    }

    // Static assets: cache-first
    event.respondWith(cacheFirst(event.request, STATIC_CACHE));
});

// Cache-first strategy: serve from cache, fall back to network (and update cache)
async function cacheFirst(request, cacheName) {
    const cached = await caches.match(request);
    if (cached) return cached;

    try {
        const response = await fetch(request);
        if (response.ok) {
            const cache = await caches.open(cacheName);
            cache.put(request, response.clone());
        }
        return response;
    } catch (err) {
        // Offline and not cached — return basic offline response
        return new Response('Offline', { status: 503, statusText: 'Service Unavailable' });
    }
}

// Network-first strategy: try network, fall back to cache
async function networkFirst(request, cacheName) {
    try {
        const response = await fetch(request);
        if (response.ok) {
            const cache = await caches.open(cacheName);
            cache.put(request, response.clone());
        }
        return response;
    } catch (err) {
        const cached = await caches.match(request);
        if (cached) return cached;
        // Return a minimal offline page with a link back to the cached home page
        const scope = self.registration.scope.replace(/^https?:\/\/[^/]+/, '');
        return new Response(
            '<html><body style="font-family:sans-serif;text-align:center;padding:40px;">' +
            '<h2>You are offline</h2><p>This page is not available offline.</p>' +
            '<a href="' + scope + '" style="display:inline-block;margin-top:20px;padding:10px 24px;background:#2563eb;color:#fff;border-radius:6px;text-decoration:none;font-size:1rem;">Back to Home</a>' +
            '</body></html>',
            { status: 503, headers: { 'Content-Type': 'text/html' } }
        );
    }
}

// API handler: network-only, with offline-reports cache fallback for report data
async function handleAPIRequest(request) {
    try {
        return await fetch(request);
    } catch (err) {
        // Offline — check if this report is saved for offline
        const cached = await caches.match(request, { cacheName: OFFLINE_CACHE });
        if (cached) return cached;

        // Also try matching just the pathname (saved reports may have been stored with filters)
        const url = new URL(request.url);
        if (url.pathname.includes('/api/report/')) {
            // Try matching without query string for saved reports
            const baseUrl = url.origin + url.pathname;
            const cache = await caches.open(OFFLINE_CACHE);
            const keys = await cache.keys();
            for (const key of keys) {
                const keyUrl = new URL(key.url);
                if (keyUrl.pathname === url.pathname) {
                    const match = await cache.match(key);
                    if (match) return match;
                }
            }
        }

        return new Response(
            JSON.stringify({ error: 'offline', message: 'This report is not available offline. Save reports for offline viewing while connected.' }),
            { status: 503, headers: { 'Content-Type': 'application/json' } }
        );
    }
}

// Listen for messages from the main app
self.addEventListener('message', (event) => {
    if (event.data && event.data.type === 'CACHE_VERSION') {
        // Could use this to update cache version in the future
        console.log('SW: Received version info:', event.data.version);
    }
});
