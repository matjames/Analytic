export const configs = {
    environment: 'prod',
    url: 'https://auth.statgate.ug',
    realm: 'StatGate',
    clientId: 'statgate-report-browser'
};

// Cached auth config from backend
let authConfigCache = null;

/**
 * Fetch auth configuration from backend.
 * Returns cached result on subsequent calls.
 * @returns {Promise<{mode: string, requireLogin: boolean, keycloak?: {url: string, realm: string, clientId: string}}>}
 */
export async function getAuthConfig() {
    if (authConfigCache !== null) {
        return authConfigCache;
    }

    try {
        const basePath = window.BASE_PATH || '';
        const response = await fetch(`${basePath}/api/auth/config`);
        if (!response.ok) {
            throw new Error(`Auth config fetch failed: ${response.status}`);
        }
        authConfigCache = await response.json();
        return authConfigCache;
    } catch (error) {
        console.error('[Config] Failed to fetch auth config:', error);
        // Default to requiring login on fetch failure (fail secure)
        authConfigCache = { mode: 'on', requireLogin: true };
        return authConfigCache;
    }
}