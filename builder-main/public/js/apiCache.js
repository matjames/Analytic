/**
 * API Cache Module
 * Centralized caching layer for fetch calls with configurable TTL per endpoint
 * Supports Keycloak auth token injection via setAuthProvider()
 */

const apiCache = (function() {
    // In-memory cache storage
    const cache = new Map();

    // Auth token provider function (set via setAuthProvider)
    let authTokenProvider = null;

    // TTL configuration in milliseconds
    const TTL = {
        HOUR: 60 * 60 * 1000,
        DAY: 24 * 60 * 60 * 1000,
        MINUTES_5: 5 * 60 * 1000,
        MINUTES_6_HOURS: 6 * 60 * 60 * 1000,
    };

    // Default TTLs by URL pattern (order matters - first match wins)
    const DEFAULT_TTLS = [
        // Schema endpoints - very stable
        { pattern: /\/api\/schema\/schemas$/, ttl: TTL.DAY },
        { pattern: /\/api\/schema\/tables/, ttl: TTL.DAY },
        { pattern: /\/api\/schema\/table\/[^/]+\/[^/]+\/columns/, ttl: TTL.DAY },

        // Builder categories - stable
        { pattern: /\/api\/builder\/categories$/, ttl: TTL.DAY },

        // Filter endpoints
        { pattern: /\/api\/filters\/districts$/, ttl: TTL.DAY },
        { pattern: /\/api\/filters\/regions$/, ttl: TTL.DAY },
        { pattern: /\/api\/filters\/years$/, ttl: TTL.DAY },
        { pattern: /\/api\/filters\/facilities$/, ttl: TTL.MINUTES_6_HOURS },
        { pattern: /\/api\/filters\/months$/, ttl: TTL.DAY },
        { pattern: /\/api\/filters\/quarters$/, ttl: TTL.DAY },
        { pattern: /\/api\/filters\/weeks$/, ttl: TTL.DAY },

        // Report list - relatively stable
        { pattern: /\/api\/reports$/, ttl: TTL.HOUR },

        // Report source (YAML) - may be edited
        { pattern: /\/api\/report\/source\//, ttl: TTL.HOUR },

        // Report data queries - cache briefly
        { pattern: /\/api\/report\/[^/]/, ttl: TTL.MINUTES_5 },
    ];

    /**
     * Generate cache key from URL and options
     */
    function getCacheKey(url, options = {}) {
        const method = (options.method || 'GET').toUpperCase();
        // Only cache GET requests
        if (method !== 'GET') {
            return null;
        }
        return `${method}:${url}`;
    }

    /**
     * Map HTTP status (and optional API error string) to a concise user-facing message.
     */
    function messageForHttpError(status, serverMessage) {
        const s = (serverMessage || '').trim();
        if (status === 401) {
            return 'Your session may have expired. Refresh the page or sign in again.';
        }
        if (status === 403) {
            return 'You do not have permission for this action.';
        }
        if (status === 429) {
            return s || 'Too many requests. Please wait a moment and try again.';
        }
        if (status === 404) {
            return s || 'The requested resource was not found.';
        }
        if (status >= 500) {
            const generic = /^internal server error$/i.test(s);
            if (!s || generic) {
                return 'Something went wrong on the server. Please try again.';
            }
            return s.length > 300 ? 'Something went wrong on the server. Please try again.' : s;
        }
        if (s) {
            return s;
        }
        return 'Request failed (HTTP ' + status + ').';
    }

    /**
     * Get TTL for a URL based on configured patterns
     */
    function getTTL(url) {
        for (const config of DEFAULT_TTLS) {
            if (config.pattern.test(url)) {
                return config.ttl;
            }
        }
        // Default: 5 minutes for unknown GET endpoints
        return TTL.MINUTES_5;
    }

    /**
     * Check if a cache entry is still valid
     */
    function isValid(entry) {
        if (!entry) return false;
        return Date.now() < entry.expires;
    }

    /**
     * Set the auth token provider function
     * Called after Keycloak initializes to enable token injection
     * @param {Function} tokenFn - Async function that returns a valid token or null
     */
    function setAuthProvider(tokenFn) {
        authTokenProvider = tokenFn;
        console.debug('[apiCache] Auth provider configured');
    }

    /**
     * Get auth headers if provider is configured
     * @returns {Promise<Object>} Headers object with Authorization if authenticated
     */
    async function getAuthHeaders() {
        if (!authTokenProvider) {
            return {};
        }

        try {
            const token = await authTokenProvider();
            if (token) {
                return { 'Authorization': `Bearer ${token}` };
            }
        } catch (error) {
            console.warn('[apiCache] Failed to get auth token:', error);
        }

        return {};
    }

    /**
     * Cached fetch wrapper
     * @param {string} url - URL to fetch
     * @param {Object} options - Fetch options plus cache options
     * @param {number} options.ttl - Override TTL in milliseconds
     * @param {boolean} options.bypassCache - Skip cache and fetch fresh
     * @returns {Promise<any>} Parsed JSON response
     */
    async function cachedFetch(url, options = {}) {
        const { ttl: customTTL, bypassCache, ...fetchOptions } = options;

        // Inject auth headers if provider is configured
        const authHeaders = await getAuthHeaders();
        fetchOptions.headers = {
            ...fetchOptions.headers,
            ...authHeaders
        };

        const cacheKey = getCacheKey(url, fetchOptions);

        // Non-GET requests or bypass: fetch directly
        if (!cacheKey || bypassCache) {
            const response = await fetch(url, fetchOptions);

            if (response.status === 401) {
                console.warn('[apiCache] Unauthorized response - token may be expired');
            }

            if (!response.ok) {
                let serverMsg = '';
                try {
                    const body = await response.json();
                    if (body && body.error) serverMsg = body.error;
                } catch (_) {}
                throw new Error(messageForHttpError(response.status, serverMsg));
            }

            return response.json();
        }

        // Check cache
        const cached = cache.get(cacheKey);
        if (isValid(cached)) {
            return cached.data;
        }

        // Fetch fresh data
        const response = await fetch(url, fetchOptions);

        if (response.status === 401) {
            console.warn('[apiCache] Unauthorized response - token may be expired');
        }

        if (!response.ok) {
            let serverMsg = '';
            try {
                const body = await response.json();
                if (body && body.error) serverMsg = body.error;
            } catch (_) {}
            throw new Error(messageForHttpError(response.status, serverMsg));
        }

        const data = await response.json();

        // Store in cache with TTL (only cache successful responses)
        const ttl = customTTL || getTTL(url);
        cache.set(cacheKey, {
            data,
            expires: Date.now() + ttl,
            url,
        });

        return data;
    }

    /**
     * Invalidate a specific URL from cache
     * @param {string} url - URL to invalidate
     */
    function invalidate(url) {
        const key = `GET:${url}`;
        cache.delete(key);
    }

    /**
     * Invalidate all URLs matching a pattern
     * @param {RegExp} pattern - Pattern to match against cached URLs
     */
    function invalidatePattern(pattern) {
        for (const [key, entry] of cache.entries()) {
            if (pattern.test(entry.url)) {
                cache.delete(key);
            }
        }
    }

    /**
     * Clear entire cache
     */
    function clear() {
        cache.clear();
    }

    /**
     * Get cache stats (for debugging)
     */
    function getStats() {
        const entries = [];
        const now = Date.now();
        for (const [key, entry] of cache.entries()) {
            entries.push({
                url: entry.url,
                ttlRemaining: Math.max(0, entry.expires - now),
                expired: now >= entry.expires,
            });
        }
        return {
            size: cache.size,
            entries,
        };
    }

    // Public API
    return {
        fetch: cachedFetch,
        setAuthProvider,
        invalidate,
        invalidatePattern,
        clear,
        getStats,
        TTL,
    };
})();

// Export for both module and global use
if (typeof module !== 'undefined' && module.exports) {
    module.exports = apiCache;
}
