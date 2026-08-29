/**
 * Telemetry Module
 * Thin client for UI interaction events. Call sites invoke telemetry.track()
 * at the point of emission; this module handles queuing, batching, and flush.
 *
 * No DOM listeners are attached here — instrumentation is explicit at each
 * call site (matches the project's "explicit over magic" principle).
 *
 * Follows the apiCache.js IIFE pattern. Exposes window.telemetry.
 *
 * API:
 *   telemetry.track(name, props)     Queue an event. `name` must match the
 *                                    server whitelist; unknown names are
 *                                    accepted here (server drops them) to
 *                                    avoid coupling client to server list.
 *   telemetry.flush()                Force-flush the queue (fetch).
 *   telemetry.setEnabled(bool)       Kill switch — disables tracking entirely.
 */

const telemetry = (function() {
    // ---- Config ----
    const FLUSH_INTERVAL_MS = 10_000;  // Periodic flush cadence
    const QUEUE_FLUSH_SIZE  = 20;      // Flush early when queue hits this
    const MAX_QUEUE_SIZE    = 100;     // Drop oldest beyond this
    const ENDPOINT_PATH     = '/api/metrics/ui-event';

    // ---- State ----
    let _queue = [];
    let _enabled = true;
    let _sessionId = null;
    let _flushTimer = null;
    let _initialized = false;

    // ---- Session ID ----
    // One per tab, survives navigation within the SPA. Anonymous.
    function _getSessionId() {
        if (_sessionId) return _sessionId;
        try {
            _sessionId = sessionStorage.getItem('telemetry_session_id');
            if (!_sessionId) {
                _sessionId = (crypto.randomUUID && crypto.randomUUID()) ||
                             String(Date.now()) + '-' + Math.random().toString(36).slice(2, 10);
                sessionStorage.setItem('telemetry_session_id', _sessionId);
            }
        } catch (_) {
            // Private mode / storage disabled — fall back to in-memory ID
            _sessionId = String(Date.now()) + '-' + Math.random().toString(36).slice(2, 10);
        }
        return _sessionId;
    }

    // ---- Flush mechanics ----
    function _endpoint() {
        return (window.BASE_PATH || '') + ENDPOINT_PATH;
    }

    function _buildBody(events) {
        return JSON.stringify({
            sessionId: _getSessionId(),
            events: events
        });
    }

    // Normal flush — fetch, keepalive so in-flight requests survive navigation.
    function _flushFetch(events) {
        fetch(_endpoint(), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: _buildBody(events),
            keepalive: true
        }).catch(() => { /* drop; telemetry must never throw in call sites */ });
    }

    // Unload-path flush — sendBeacon is the only reliable channel on pagehide.
    function _flushBeacon(events) {
        try {
            const blob = new Blob([_buildBody(events)], { type: 'application/json' });
            navigator.sendBeacon(_endpoint(), blob);
        } catch (_) {
            // Nothing useful to do on unload failure.
        }
    }

    function _drainAndFlush(useBeacon) {
        if (_queue.length === 0) return;
        // Cap per-batch to respect server limit (50).
        const batch = _queue.splice(0, 50);
        if (useBeacon) {
            _flushBeacon(batch);
        } else {
            _flushFetch(batch);
        }
    }

    // ---- Public API ----
    function track(name, props) {
        if (!_enabled || typeof name !== 'string' || !name) return;

        _initOnce();

        const event = { name: name };

        if (props && typeof props === 'object') {
            // Extract reportId as a top-level field (server stores it separately).
            if (props.reportId) {
                event.reportId = String(props.reportId);
            }
            // Remaining keys become props (bounded; server enforces hard limits).
            const rest = {};
            let keys = 0;
            for (const k in props) {
                if (k === 'reportId') continue;
                if (keys >= 8) break;
                const v = props[k];
                if (v === null || v === undefined) continue;
                rest[k] = String(v);
                keys++;
            }
            if (keys > 0) event.props = rest;
        }

        _queue.push(event);

        if (_queue.length >= MAX_QUEUE_SIZE) {
            _queue = _queue.slice(-MAX_QUEUE_SIZE);  // drop oldest
        }
        if (_queue.length >= QUEUE_FLUSH_SIZE) {
            _drainAndFlush(false);
        }
    }

    function flush() {
        _drainAndFlush(false);
    }

    function setEnabled(enabled) {
        _enabled = !!enabled;
        if (!_enabled) _queue = [];
    }

    // ---- Lazy init (no work until first track() call) ----
    function _initOnce() {
        if (_initialized) return;
        _initialized = true;

        _flushTimer = setInterval(() => _drainAndFlush(false), FLUSH_INTERVAL_MS);

        // Unload path — sendBeacon ensures last events ship.
        window.addEventListener('pagehide', () => _drainAndFlush(true));
        // visibilitychange covers tab-switch-then-close on some browsers.
        document.addEventListener('visibilitychange', () => {
            if (document.visibilityState === 'hidden') _drainAndFlush(true);
        });
    }

    return {
        track: track,
        flush: flush,
        setEnabled: setEnabled
    };
})();

if (typeof window !== 'undefined') {
    window.telemetry = telemetry;
}
