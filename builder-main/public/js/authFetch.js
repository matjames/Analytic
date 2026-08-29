/**
 * Auth-aware fetch wrapper
 * Adds Authorization header with Keycloak token to all requests
 */

import { keycloak } from './auth.js';

/**
 * Get current valid token, refreshing if needed
 * @returns {Promise<string|null>} Bearer token or null if not authenticated
 */
export async function getAuthToken() {
    if (!keycloak || !keycloak.authenticated) {
        return null;
    }

    try {
        // Refresh token if it expires in less than 30 seconds
        const refreshed = await keycloak.updateToken(30);
        if (refreshed) {
            console.debug('[Auth] Token refreshed');
        }
        return keycloak.token;
    } catch (error) {
        console.error('[Auth] Failed to refresh token:', error);
        // Token refresh failed - user needs to re-login
        keycloak.login();
        return null;
    }
}

/**
 * Fetch with automatic auth header injection
 * @param {string} url - URL to fetch
 * @param {RequestInit} options - Fetch options
 * @returns {Promise<Response>}
 */
export async function authFetch(url, options = {}) {
    const token = await getAuthToken();

    const headers = {};
    const incoming = options.headers;
    if (incoming) {
        if (typeof incoming.forEach === 'function') {
            incoming.forEach((value, key) => {
                headers[key] = value;
            });
        } else {
            Object.assign(headers, incoming);
        }
    }
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    return fetch(url, {
        ...options,
        headers,
    });
}

/**
 * Fetch JSON with automatic auth header injection
 * @param {string} url - URL to fetch
 * @param {RequestInit} options - Fetch options
 * @returns {Promise<any>} Parsed JSON response
 */
export async function authFetchJSON(url, options = {}) {
    const response = await authFetch(url, options);

    if (!response.ok) {
        if (response.status === 401) {
            console.warn('[Auth] Unauthorized - redirecting to login');
            keycloak.login();
            throw new Error('Authentication required');
        }
        if (response.status === 403) {
            throw new Error('Insufficient permissions');
        }
        let message = `HTTP ${response.status}: ${response.statusText}`;
        try {
            const body = await response.json();
            if (body.error) message = body.error;
        } catch {}
        throw new Error(message);
    }

    return response.json();
}

/**
 * POST JSON with automatic auth header injection
 * @param {string} url - URL to post to
 * @param {any} body - Body to serialize as JSON
 * @returns {Promise<any>} Parsed JSON response
 */
export async function authPostJSON(url, body) {
    return authFetchJSON(url, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(body)
    });
}

export async function authPutJSON(url, body) {
    return authFetchJSON(url, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(body)
    });
}

/**
 * Check if current user has a specific role
 * @param {string} role - Role to check (e.g., 'admin')
 * @param {string} client - Client ID (default: 'dwhlanding')
 * @returns {boolean}
 */
export function hasRole(role, client = 'dwhlanding') {
    if (!keycloak || !keycloak.authenticated || !keycloak.tokenParsed) {
        return false;
    }

    const clientRoles = keycloak.tokenParsed?.resource_access?.[client]?.roles || [];
    return clientRoles.includes(role);
}

/**
 * Check if current user has any of the specified roles
 * @param {string[]} roles - Roles to check
 * @param {string} client - Client ID (default: 'dwhlanding')
 * @returns {boolean}
 */
export function hasAnyRole(roles, client = 'dwhlanding') {
    return roles.some(role => hasRole(role, client));
}

/**
 * Get current user info from token
 * @returns {Object|null} User info or null if not authenticated
 */
export function getCurrentUser() {
    if (!keycloak || !keycloak.authenticated || !keycloak.tokenParsed) {
        return null;
    }

    const token = keycloak.tokenParsed;
    return {
        subject: token.sub,
        username: token.preferred_username,
        email: token.email,
        name: token.name,
        roles: token.resource_access?.dwhlanding?.roles || [],
        realmRoles: token.realm_access?.roles || []
    };
}
