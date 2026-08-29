import Keycloak from './keycloak.js';
import { configs, getAuthConfig } from './config.js';
import { fetchCurrentUser } from './currentUser.js';

// Keycloak instance - initialized lazily only when auth is required
export let keycloak = null;

// Track whether Keycloak is active (mode="on" and successfully initialized)
let keycloakActive = false;

// Track whether publishing is allowed (only in dev mode)
let canPublish = true;

/**
 * Get current valid token, refreshing if needed
 * Used by apiCache for automatic token injection
 * @returns {Promise<string|null>} Token or null if not authenticated
 */
async function getAuthToken() {
    if (!keycloak || !keycloak.authenticated) {
        return null;
    }
    try {
        // Refresh token if it expires in less than 30 seconds
        await keycloak.updateToken(30);
        return keycloak.token;
    } catch (error) {
        console.error('[Auth] Token refresh failed:', error);
        return null;
    }
}

/**
 * Check if publishing is allowed (only in dev mode)
 * @returns {boolean} true if publishing is allowed
 */
export function isPublishingAllowed() {
    return canPublish;
}

// Function to handle URL changes
function handleUrlChange() {
    const url = window.location.href;
    const title = document.title; 

    if (keycloak && keycloak.authenticated) {
        const userId = keycloak.tokenParsed?.sub;

        if (userId && typeof gtag === "function") {
            // 1. Set the user_id for this session
            gtag('set', { 'user_id': userId });

            // 2. "Send" the data using a page_view event
            gtag('event', 'page_view', {
                'page_location': url,   
                'page_title': title,   
                'user_id': userId,      
                'report_url': url      
            });

            console.log("GA Page View sent for: " + url);
        }
    }
}

window.addEventListener('popstate', handleUrlChange);

const originalPushState = history.pushState;
history.pushState = function() {
    originalPushState.apply(this, arguments);
    handleUrlChange();
};

const originalReplaceState = history.replaceState;
history.replaceState = function() {
    originalReplaceState.apply(this, arguments);
    handleUrlChange();
};

/**
 * Initialize authentication based on backend configuration.
 * - mode="on": Initialize Keycloak with login-required
 * - mode="off": Skip Keycloak entirely, resolve immediately
 * @returns {Promise<boolean>} true if authenticated (or auth not required)
 */
export async function initAuth() {
    try {
        // Fetch auth config from backend
        const authConfig = await getAuthConfig();
        console.log('[Auth] Backend config:', authConfig);

        // Set publishing permission from backend
        canPublish = authConfig.canPublish !== false;

        // If auth mode is not "on", skip Keycloak entirely
        if (!authConfig.requireLogin) {
            console.log('[Auth] Auth not required (mode=' + authConfig.mode + '), skipping Keycloak');
            hideLogoutButton();
            // Expose a no-op token getter for consistency
            window.getAuthToken = () => Promise.resolve(null);
            await fetchCurrentUser().catch((error) => console.warn('[Auth] Current user unavailable:', error));
            await fetchAndApplyMenuPermissions();
            return true; // Considered "authenticated" for app flow
        }

        // Auth mode is "on" - initialize Keycloak
        const keycloakConfig = authConfig.keycloak || {
            url: configs.url,
            realm: configs.realm,
            clientId: configs.clientId
        };

        keycloak = new Keycloak({
            url: keycloakConfig.url,
            realm: keycloakConfig.realm,
            clientId: keycloakConfig.clientId
        });

        const authenticated = await keycloak.init({
            onLoad: 'login-required',
            pkceMethod: 'S256',
            checkLoginIframe: false
        });

        if (authenticated) {
            keycloakActive = true;

            // Connect apiCache to use our tokens (if apiCache is loaded)
            if (typeof apiCache !== 'undefined' && apiCache.setAuthProvider) {
                apiCache.setAuthProvider(getAuthToken);
            }

            // Expose token getter globally for non-module scripts (e.g., app.js)
            window.getAuthToken = getAuthToken;

            await fetchCurrentUser().catch((error) => console.warn('[Auth] Current user unavailable:', error));
            handleAuthenticatedUser();
        }
        return authenticated;
    } catch (error) {
        console.error('[Auth] Initialization failed:', error);
        throw error;
    }
}

function handleAuthenticatedUser() {
    fetchAndApplyMenuPermissions();

    const logoutBtn = document.getElementById('logoutBtn');
    if (logoutBtn && keycloakActive) {
        logoutBtn.style.display = 'block';
        logoutBtn.addEventListener('click', () => {
            keycloak.logout({
                redirectUri: configs.environment === 'dev'
                    ? `http://localhost:${location.port || '8110'}`
                    : `${location.origin}${location.pathname}`
            });
        });
    }

    // Show sign out in menu
    const menuSignOut = document.getElementById('menu-sign-out');
    if (menuSignOut && keycloakActive) {
        menuSignOut.style.display = '';
        menuSignOut.addEventListener('click', () => {
            keycloak.logout({
                redirectUri: configs.environment === 'dev'
                    ? `http://localhost:${location.port || '8110'}`
                    : `${location.origin}${location.pathname}`
            });
        });
    }
}

/**
 * Silent variant of initAuth for pages (like /metrics) that should remain
 * viewable to anonymous users but light up extra features when a Keycloak
 * session already exists. Uses onLoad: 'check-sso' so there is no forced
 * redirect; callers can trigger an explicit login via window.signIn().
 * @returns {Promise<boolean>} true when a Keycloak session is established.
 */
export async function initAuthSilent() {
    try {
        const authConfig = await getAuthConfig();
        canPublish = authConfig.canPublish !== false;

        if (!authConfig.requireLogin) {
            window.getAuthToken = () => Promise.resolve(null);
            window.signIn = () => {};
            return true;
        }

        const keycloakConfig = authConfig.keycloak || {
            url: configs.url,
            realm: configs.realm,
            clientId: configs.clientId
        };

        keycloak = new Keycloak({
            url: keycloakConfig.url,
            realm: keycloakConfig.realm,
            clientId: keycloakConfig.clientId
        });

        // Expose login trigger immediately so the UI can offer a sign-in button
        // even if check-sso reports the user is not authenticated.
        window.signIn = () => keycloak.login();

        const authenticated = await keycloak.init({
            onLoad: 'check-sso',
            pkceMethod: 'S256',
            checkLoginIframe: false
        });

        if (authenticated) {
            keycloakActive = true;
            window.getAuthToken = getAuthToken;
            handleAuthenticatedUser();
        } else {
            window.getAuthToken = () => Promise.resolve(null);
        }
        return authenticated;
    } catch (error) {
        console.error('[Auth] Silent init failed:', error);
        window.getAuthToken = () => Promise.resolve(null);
        window.signIn = () => {};
        return false;
    }
}

function hideLogoutButton() {
    const logoutBtn = document.getElementById('logoutBtn');
    if (logoutBtn) {
        logoutBtn.style.display = 'none';
    }
}

function hideBuilderLink() {
    const builderLink = document.getElementById('yaml-builder');
    if (builderLink) {
        builderLink.style.display = 'none';
    }
}

function showBuilderLink() {
    const builderLink = document.getElementById('yaml-builder');
    if (builderLink) {
        builderLink.style.display = 'block';
    }
}

function hideUserManagementLink() {
    const userManagementLink = document.getElementById('user-management');
    if (userManagementLink) {
        userManagementLink.style.display = 'none';
    }
}

function showUserManagementLink() {
    const userManagementLink = document.getElementById('user-management');
    if (userManagementLink) {
        userManagementLink.style.display = '';
    }
}

function applyMenuPermissions(perms) {
    if (perms && perms.builder === true) {
        showBuilderLink();
    } else {
        hideBuilderLink();
    }

    if (perms && perms.admin === true) {
        showUserManagementLink();
    } else {
        hideUserManagementLink();
    }
}

async function fetchAndApplyMenuPermissions() {
    hideBuilderLink();
    hideUserManagementLink();

    try {
        const basePath = window.BASE_PATH || '';
        const headers = {};
        if (typeof window.getAuthToken === 'function') {
            const token = await window.getAuthToken();
            if (token) {
                headers.Authorization = 'Bearer ' + token;
            }
        }

        const res = await fetch(basePath + '/api/me/permissions', { headers });
        if (!res.ok) {
            applyMenuPermissions({ builder: false, admin: false });
            return;
        }

        const perms = await res.json();
        applyMenuPermissions(perms);
    } catch (error) {
        console.warn('[Auth] Failed to fetch menu permissions:', error);
        applyMenuPermissions({ builder: false, admin: false });
    }
}
