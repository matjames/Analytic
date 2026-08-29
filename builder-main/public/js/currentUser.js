let currentUserProfile = null;
let currentUserPromise = null;

function basePath() {
    return window.BASE_PATH || '';
}

async function resolveToken() {
    if (typeof window.getAuthToken !== 'function') {
        return null;
    }
    try {
        return await window.getAuthToken();
    } catch (error) {
        console.warn('[CurrentUser] Failed to resolve auth token:', error);
        return null;
    }
}

function unwrapUser(body) {
    return body?.data?.user || body?.user || null;
}

export async function fetchCurrentUser(options = {}) {
    const force = options.force === true;
    if (!force && currentUserProfile) {
        return currentUserProfile;
    }
    if (!force && currentUserPromise) {
        return currentUserPromise;
    }

    currentUserPromise = (async () => {
        const headers = { Accept: 'application/json' };
        const token = await resolveToken();
        if (token) {
            headers.Authorization = 'Bearer ' + token;
        }

        const response = await fetch(basePath() + '/api/me', { headers, credentials: 'include' });
        if (!response.ok) {
            throw new Error('Current user fetch failed: HTTP ' + response.status);
        }

        const body = await response.json();
        const user = unwrapUser(body);
        if (!user) {
            throw new Error('Current user response did not include a user');
        }

        currentUserProfile = normalizeUser(user);
        window.currentUser = currentUserProfile;
        window.dispatchEvent(new CustomEvent('current-user:loaded', { detail: currentUserProfile }));
        return currentUserProfile;
    })();

    try {
        return await currentUserPromise;
    } finally {
        currentUserPromise = null;
    }
}

export function getCurrentUserProfile() {
    return currentUserProfile || window.currentUser || null;
}

export function getAccessibleSystems() {
    const user = getCurrentUserProfile();
    return Array.isArray(user?.accessibleSystems) ? user.accessibleSystems : [];
}

export function hasPermission(permission) {
    const user = getCurrentUserProfile();
    return Array.isArray(user?.permissions) && user.permissions.includes(permission);
}

function normalizeUser(user) {
    return {
        ...user,
        realmRoles: Array.isArray(user.realmRoles) ? user.realmRoles : [],
        clientRoles: user.clientRoles && typeof user.clientRoles === 'object' ? user.clientRoles : {},
        permissions: Array.isArray(user.permissions) ? user.permissions : [],
        systems: Array.isArray(user.systems) ? user.systems : [],
        accessibleSystems: Array.isArray(user.accessibleSystems) ? user.accessibleSystems : [],
    };
}

window.fetchCurrentUser = fetchCurrentUser;
window.getCurrentUserProfile = getCurrentUserProfile;
window.getAccessibleSystems = getAccessibleSystems;
window.hasPermission = hasPermission;
