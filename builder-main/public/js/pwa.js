// PWA features: offline detection, save for offline, favorites, install prompt
// IIFE pattern matching apiCache.js / filterManager.js
const pwa = (function() {
    let _deferredPrompt = null;
    let _isOffline = !navigator.onLine;
    const MAX_OFFLINE_REPORTS = 20;

    // ============ Offline Detection ============

    function initOfflineDetection() {
        window.addEventListener('online', () => {
            _isOffline = false;
            const banner = document.getElementById('offline-banner');
            if (banner) banner.style.display = 'none';
        });

        window.addEventListener('offline', () => {
            _isOffline = true;
            const banner = document.getElementById('offline-banner');
            if (banner) banner.style.display = 'block';
        });

        // Set initial state
        if (_isOffline) {
            const banner = document.getElementById('offline-banner');
            if (banner) banner.style.display = 'block';
        }
    }

    function isOffline() {
        return _isOffline;
    }

    // ============ Toast Notifications ============

    function showToast(message, duration) {
        duration = duration || 3000;
        // Remove existing toast
        const existing = document.getElementById('pwa-toast');
        if (existing) existing.remove();

        const toast = document.createElement('div');
        toast.id = 'pwa-toast';
        toast.className = 'pwa-toast';
        toast.textContent = message;
        document.body.appendChild(toast);

        // Trigger animation
        requestAnimationFrame(() => toast.classList.add('show'));

        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => toast.remove(), 300);
        }, duration);
    }

    // ============ Save for Offline ============

    async function saveForOffline(reportId, filterParams, reportData) {
        if (!('caches' in window)) {
            showToast('Offline saving not supported in this browser');
            return false;
        }

        try {
            const cache = await caches.open('offline-reports');

            // Build the URL that would be used to fetch this report
            const basePath = window.BASE_PATH || '';
            const params = new URLSearchParams(filterParams || {});
            const queryString = params.toString() ? '?' + params.toString() : '';
            const url = basePath + '/api/report/' + reportId + queryString;

            // Store the JSON response
            const response = new Response(JSON.stringify(reportData), {
                headers: { 'Content-Type': 'application/json' }
            });
            await cache.put(url, response);

            // Track in localStorage
            const saved = JSON.parse(localStorage.getItem('offlineReports') || '{}');
            saved[reportId] = {
                savedAt: Date.now(),
                filters: filterParams || {},
                url: url
            };
            localStorage.setItem('offlineReports', JSON.stringify(saved));

            // Evict oldest entries if over limit
            const ids = Object.keys(saved);
            if (ids.length > MAX_OFFLINE_REPORTS) {
                const sorted = ids.sort((a, b) => (saved[a].savedAt || 0) - (saved[b].savedAt || 0));
                const toRemove = sorted.slice(0, ids.length - MAX_OFFLINE_REPORTS);
                for (const id of toRemove) {
                    try { await cache.delete(saved[id].url); } catch (_) {}
                    delete saved[id];
                }
                localStorage.setItem('offlineReports', JSON.stringify(saved));
            }

            showToast('Report saved for offline viewing');
            return true;
        } catch (err) {
            console.error('Failed to save for offline:', err);
            showToast('Failed to save report offline');
            return false;
        }
    }

    function isReportSavedOffline(reportId) {
        const saved = JSON.parse(localStorage.getItem('offlineReports') || '{}');
        return !!saved[reportId];
    }

    function getOfflineSavedDate(reportId) {
        const saved = JSON.parse(localStorage.getItem('offlineReports') || '{}');
        return saved[reportId] ? saved[reportId].savedAt : null;
    }

    async function removeOfflineReport(reportId) {
        const saved = JSON.parse(localStorage.getItem('offlineReports') || '{}');
        if (saved[reportId] && 'caches' in window) {
            try {
                const cache = await caches.open('offline-reports');
                await cache.delete(saved[reportId].url);
            } catch (err) {
                console.warn('Failed to remove from cache:', err);
            }
        }
        delete saved[reportId];
        localStorage.setItem('offlineReports', JSON.stringify(saved));
    }

    // ============ Favorites ============

    function getFavorites() {
        return JSON.parse(localStorage.getItem('favorites') || '[]');
    }

    function isFavorite(reportId) {
        return getFavorites().includes(reportId);
    }

    function toggleFavorite(reportId) {
        const favorites = getFavorites();
        const index = favorites.indexOf(reportId);
        if (index === -1) {
            favorites.unshift(reportId);
        } else {
            favorites.splice(index, 1);
        }
        localStorage.setItem('favorites', JSON.stringify(favorites));
        renderFavorites();
        return index === -1; // returns true if just added
    }

    function renderFavorites() {
        const section = document.getElementById('favorites-section');
        const list = document.getElementById('favorites-list');
        if (!section || !list) return;

        const favorites = getFavorites();
        // allReports is a global from app.js
        const reports = typeof allReports !== 'undefined' ? allReports : [];

        if (favorites.length === 0) {
            section.style.display = 'none';
            return;
        }

        section.style.display = 'block';
        list.innerHTML = '';

        favorites.forEach(id => {
            const report = reports.find(r => r.id === id);
            if (!report) return;

            const item = document.createElement('div');
            item.className = 'favorites-item';

            const btn = document.createElement('button');
            btn.className = 'favorites-link';
            btn.type = 'button';
            btn.textContent = report.title;
            btn.addEventListener('click', async (e) => {
                e.preventDefault();
                await selectReport(id);
                if (typeof closeMobileSidebar === 'function') closeMobileSidebar();
            });

            const removeBtn = document.createElement('button');
            removeBtn.className = 'favorites-remove';
            removeBtn.type = 'button';
            removeBtn.innerHTML = '&times;';
            removeBtn.title = 'Remove from favorites';
            removeBtn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                toggleFavorite(id);
                // Update star button if viewing this report
                updateStarButton(id);
            });

            item.appendChild(btn);
            item.appendChild(removeBtn);
            list.appendChild(item);
        });
    }

    // Helper: update star button state if it exists
    function updateStarButton(reportId) {
        const starBtn = document.getElementById('btn-favorite');
        if (starBtn && starBtn.dataset.reportId === reportId) {
            const isFav = isFavorite(reportId);
            starBtn.classList.toggle('active', isFav);
            starBtn.title = isFav ? 'Remove from favorites' : 'Add to favorites';
        }
    }

    // ============ Install Prompt ============

    function initInstallPrompt() {
        // Skip if already installed (standalone mode)
        if (window.matchMedia('(display-mode: standalone)').matches) return;
        if (navigator.standalone) return; // iOS

        // Listen for beforeinstallprompt (Chrome/Edge/Android)
        window.addEventListener('beforeinstallprompt', (e) => {
            e.preventDefault();
            _deferredPrompt = e;
            showInstallBanner('chrome');
        });

        // Detect iOS Safari
        const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent) && !window.MSStream;
        const isSafari = /Safari/.test(navigator.userAgent) && !/CriOS|FxiOS/.test(navigator.userAgent);
        if (isIOS && isSafari) {
            // Show iOS install instructions after a short delay
            setTimeout(() => showInstallBanner('ios'), 3000);
        }
    }

    function showInstallBanner(platform) {
        // Check if dismissed recently (30 days)
        const dismissed = localStorage.getItem('installBannerDismissed');
        if (dismissed) {
            const dismissedAt = parseInt(dismissed, 10);
            if (Date.now() - dismissedAt < 30 * 24 * 60 * 60 * 1000) return;
        }

        const banner = document.getElementById('install-banner');
        if (!banner) return;

        if (platform === 'ios') {
            banner.innerHTML =
                '<div class="install-banner-content">' +
                    '<span class="install-banner-text">Install StatGate Report Builder: tap <strong>Share</strong> then <strong>Add to Home Screen</strong></span>' +
                    '<button class="install-banner-dismiss" id="install-dismiss">Dismiss</button>' +
                '</div>';
        } else {
            banner.innerHTML =
                '<div class="install-banner-content">' +
                    '<span class="install-banner-text">Install StatGate Report Builder for quick access</span>' +
                    '<div class="install-banner-actions">' +
                        '<button class="install-banner-btn" id="install-accept">Install</button>' +
                        '<button class="install-banner-dismiss" id="install-dismiss">Not now</button>' +
                    '</div>' +
                '</div>';
        }

        banner.style.display = 'block';
        requestAnimationFrame(() => banner.classList.add('show'));

        // Wire up buttons
        const acceptBtn = document.getElementById('install-accept');
        if (acceptBtn) {
            acceptBtn.addEventListener('click', async () => {
                if (_deferredPrompt) {
                    _deferredPrompt.prompt();
                    const result = await _deferredPrompt.userChoice;
                    console.log('Install prompt result:', result.outcome);
                    _deferredPrompt = null;
                }
                hideInstallBanner();
            });
        }

        const dismissBtn = document.getElementById('install-dismiss');
        if (dismissBtn) {
            dismissBtn.addEventListener('click', () => {
                localStorage.setItem('installBannerDismissed', String(Date.now()));
                hideInstallBanner();
            });
        }
    }

    function hideInstallBanner() {
        const banner = document.getElementById('install-banner');
        if (banner) {
            banner.classList.remove('show');
            setTimeout(() => { banner.style.display = 'none'; }, 300);
        }
    }

    // ============ Init ============

    function init() {
        initOfflineDetection();
        initInstallPrompt();
        renderFavorites();
    }

    return {
        init: init,
        isOffline: isOffline,
        showToast: showToast,
        saveForOffline: saveForOffline,
        isReportSavedOffline: isReportSavedOffline,
        getOfflineSavedDate: getOfflineSavedDate,
        removeOfflineReport: removeOfflineReport,
        getFavorites: getFavorites,
        isFavorite: isFavorite,
        toggleFavorite: toggleFavorite,
        renderFavorites: renderFavorites
    };
})();
