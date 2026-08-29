/**
 * User Management – Keycloak Admin API integration.
 * Uses authFetch for authenticated requests. Requires admin_manage (or KEYCLOAK_ADMIN_ROLE) role.
 * Roles and user/group role assignments use client roles (app client e.g. dwhlanding).
 */

import { authFetch } from './authFetch.js';
import { getAuthConfig } from './config.js';

const base = () => (window.BASE_PATH || '') + '/api/keycloak';

async function apiGet(path) {
    const res = await authFetch(base() + path);
    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        throw new Error(err.error || `HTTP ${res.status}`);
    }
    return res.json();
}

async function apiPost(path, body) {
    const res = await authFetch(base() + path, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
    if (res.status === 204 || res.status === 201) return res.headers.get('Location') || null;
    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        throw new Error(err.error || `HTTP ${res.status}`);
    }
    return res.json ? await res.json() : null;
}

async function apiPut(path, body) {
    const res = await authFetch(base() + path, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });
    if (res.status === 204) return null;
    if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        throw new Error(err.error || `HTTP ${res.status}`);
    }
    return res.json ? await res.json() : null;
}

async function apiDelete(path) {
    const res = await authFetch(base() + path, { method: 'DELETE' });
    if (res.status === 204 || res.status === 200) return;
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || `HTTP ${res.status}`);
}

async function parseKeycloakError(res, fallback) {
    const text = await res.text();
    if (text) {
        try {
            const err = JSON.parse(text);
            const msg = err.error_description || err.errorMessage || err.message || err.error;
            if (msg) return msg;
        } catch (_) {}
    }
    return fallback + (res.status ? ` (${res.status})` : '');
}

// --- Users ---
export async function listUsers(opts = {}) {
    const q = new URLSearchParams();
    if (opts.first != null) q.set('first', opts.first);
    if (opts.max != null) q.set('max', opts.max);
    if (opts.search) q.set('search', opts.search);
    const query = q.toString();
    return apiGet('/users' + (query ? '?' + query : ''));
}

export async function createUser(user) {
    return apiPost('/users', user);
}

export async function getUser(id) {
    return apiGet('/users/' + encodeURIComponent(id));
}

export async function updateUser(id, user) {
    return apiPut('/users/' + encodeURIComponent(id), user);
}

export async function deleteUser(id) {
    return apiDelete('/users/' + encodeURIComponent(id));
}

/** Get groups the user belongs to. Returns array of { id, name, path, ... }. */
export async function getUserGroups(userId) {
    const raw = await apiGet('/users/' + encodeURIComponent(userId) + '/groups?max=500');
    return Array.isArray(raw) ? raw : (raw && raw.groups) || [];
}

/**
 * Reset user password. Body: { value: string, temporary?: boolean }.
 * Keycloak expects { type: "password", value, temporary }.
 */
export async function resetUserPassword(userId, body) {
    const payload = { type: 'password', value: body.value || '', temporary: body.temporary === true };
    return authFetch(base() + '/users/' + encodeURIComponent(userId) + '/reset-password', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
    }).then(res => {
        if (!res.ok) {
            return res.json().then(err => { throw new Error(err.error || 'Failed to set password'); });
        }
    });
}

export async function getUserRealmRoles(userId) {
    return apiGet('/users/' + encodeURIComponent(userId) + '/role-mappings/realm');
}

export async function addUserRealmRoles(userId, roles) {
    return authFetch(base() + '/users/' + encodeURIComponent(userId) + '/role-mappings/realm', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(roles),
    }).then(res => { if (!res.ok) throw new Error('Failed to add roles'); });
}

export async function removeUserRealmRoles(userId, roles) {
    return authFetch(base() + '/users/' + encodeURIComponent(userId) + '/role-mappings/realm', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(roles),
    }).then(res => { if (!res.ok) throw new Error('Failed to remove roles'); });
}

// --- Client roles (for app client e.g. dwhlanding) ---
export async function listClientRoles(clientId) {
    return apiGet('/clients/' + encodeURIComponent(clientId) + '/roles');
}

export async function createClientRole(clientId, role) {
    return authFetch(base() + '/clients/' + encodeURIComponent(clientId) + '/roles', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(role),
    }).then(res => { if (!res.ok) throw new Error('Failed to create role'); });
}

export async function getClientRoleByName(clientId, name) {
    return apiGet('/clients/' + encodeURIComponent(clientId) + '/roles/' + encodeURIComponent(name));
}

export async function updateClientRole(clientId, roleName, body) {
    return authFetch(base() + '/clients/' + encodeURIComponent(clientId) + '/roles/' + encodeURIComponent(roleName), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    }).then(res => {
        if (!res.ok) {
            return res.json().then(err => { throw new Error(err.error || 'Failed to update role'); });
        }
    });
}

export async function deleteClientRole(clientId, name) {
    return apiDelete('/clients/' + encodeURIComponent(clientId) + '/roles/' + encodeURIComponent(name));
}

export async function getUserClientRoles(userId, clientId) {
    return apiGet('/users/' + encodeURIComponent(userId) + '/role-mappings/clients/' + encodeURIComponent(clientId));
}

export async function addUserClientRoles(userId, clientId, roles) {
    return authFetch(base() + '/users/' + encodeURIComponent(userId) + '/role-mappings/clients/' + encodeURIComponent(clientId), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(roles),
    }).then(res => { if (!res.ok) throw new Error('Failed to add roles'); });
}

export async function removeUserClientRoles(userId, clientId, roles) {
    return authFetch(base() + '/users/' + encodeURIComponent(userId) + '/role-mappings/clients/' + encodeURIComponent(clientId), {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(roles),
    }).then(res => { if (!res.ok) throw new Error('Failed to remove roles'); });
}

// --- Realm roles (kept for Role Permissions view) ---
export async function listRealmRoles() {
    return apiGet('/roles');
}

export async function createRealmRole(role) {
    return apiPost('/roles', role);
}

export async function getRealmRoleByName(name) {
    return apiGet('/roles/' + encodeURIComponent(name));
}

export async function deleteRealmRoleByName(name) {
    return apiDelete('/roles/' + encodeURIComponent(name));
}

// --- Configs tree & role permissions (report access per role) ---
export async function getConfigsTree() {
    return apiGet('/configs-tree');
}

export async function getRolePermissions(role) {
    return apiGet('/role-permissions?role=' + encodeURIComponent(role));
}

export async function saveRolePermissions(role, paths) {
    return authFetch(base() + '/role-permissions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role, paths }),
    }).then(res => { if (!res.ok) throw new Error('Failed to save'); });
}

// --- Groups ---
export async function listGroups(opts = {}) {
    const q = new URLSearchParams();
    if (opts.first != null) q.set('first', opts.first);
    if (opts.max != null) q.set('max', opts.max);
    if (opts.search) q.set('search', opts.search);
    const query = q.toString();
    return apiGet('/groups' + (query ? '?' + query : ''));
}

export async function createGroup(group) {
    return apiPost('/groups', group);
}

export async function getGroup(id) {
    return apiGet('/groups/' + encodeURIComponent(id));
}

export async function updateGroup(id, group) {
    return apiPut('/groups/' + encodeURIComponent(id), group);
}

export async function deleteGroup(id) {
    return apiDelete('/groups/' + encodeURIComponent(id));
}

export async function getGroupClientRoles(groupId, clientId) {
    return apiGet('/groups/' + encodeURIComponent(groupId) + '/role-mappings/clients/' + encodeURIComponent(clientId));
}

export async function addGroupClientRoles(groupId, clientId, roles) {
    const res = await authFetch(base() + '/groups/' + encodeURIComponent(groupId) + '/role-mappings/clients/' + encodeURIComponent(clientId), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(roles),
    });
    if (!res.ok) {
        const msg = await parseKeycloakError(res, 'Failed to add roles');
        throw new Error(msg);
    }
}

export async function removeGroupClientRoles(groupId, clientId, roles) {
    const res = await authFetch(base() + '/groups/' + encodeURIComponent(groupId) + '/role-mappings/clients/' + encodeURIComponent(clientId), {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(roles),
    });
    if (!res.ok) {
        const msg = await parseKeycloakError(res, 'Failed to remove roles');
        throw new Error(msg);
    }
}

// --- Permissions (realm + client roles) ---
export async function listPermissions() {
    return apiGet('/permissions');
}

// --- UI (run after auth) ---
function el(id) { return document.getElementById(id); }
function escapeHtml(s) {
    if (s == null) return '';
    const div = document.createElement('div');
    div.textContent = s;
    return div.innerHTML;
}

let umToastTimeout = null;
/** Show a small popup toast for success or error. type: 'success' | 'error'. Auto-hides after 4s. */
function showUmToast(message, type = 'success') {
    const toast = el('um-toast');
    const msgEl = el('um-toast-message');
    if (!toast || !msgEl) return;
    if (umToastTimeout) clearTimeout(umToastTimeout);
    toast.classList.remove('um-toast-success', 'um-toast-error', 'hidden');
    toast.classList.add(type === 'error' ? 'um-toast-error' : 'um-toast-success');
    msgEl.textContent = message || (type === 'error' ? 'Something went wrong.' : 'Done.');
    toast.classList.remove('hidden');
    umToastTimeout = setTimeout(() => {
        toast.classList.add('hidden');
        umToastTimeout = null;
    }, 4000);
}
function hideUmToast() {
    const toast = el('um-toast');
    if (toast) toast.classList.add('hidden');
    if (umToastTimeout) clearTimeout(umToastTimeout);
    umToastTimeout = null;
}

export async function initUserManagementUI() {
    const authConfig = await getAuthConfig();
    const clientId = (authConfig.keycloak && (authConfig.keycloak.rolesClientId || authConfig.keycloak.clientId))
        ? (authConfig.keycloak.rolesClientId || authConfig.keycloak.clientId)
        : 'dwhlanding';

    const usersContainer = el('um-users-container');
    const rolesContainer = el('um-roles-container');
    const permissionsContainer = el('um-permissions-container');
    const groupsContainer = el('um-groups-container');
    if (!usersContainer || !rolesContainer || !permissionsContainer) return;

    // Users tab: pagination state
    let usersPage = 1;
    let usersPageSize = 25;
    const USERS_PAGE_SIZES = [10, 25, 50, 100];

    usersContainer.innerHTML = `
        <div class="um-toolbar">
            <input type="search" id="um-user-search" class="form-input um-search" placeholder="Search users...">
            <button type="button" class="btn-primary" id="um-user-add">Add user</button>
        </div>
        <div id="um-users-list" class="um-table-wrap"></div>
        <div id="um-users-pagination" class="um-pagination"></div>
        <div id="um-user-message" class="um-message"></div>
    `;
    const usersList = el('um-users-list');
    const usersPagination = el('um-users-pagination');
    const userMessage = el('um-user-message');
    let userSearchTimeout;
    el('um-user-search').addEventListener('input', () => {
        clearTimeout(userSearchTimeout);
        userSearchTimeout = setTimeout(() => { usersPage = 1; loadUsers(); }, 300);
    });
    el('um-user-add').addEventListener('click', () => showUserForm());

    function renderUsersPagination(hasMore, totalReturned) {
        if (!usersPagination) return;
        const first = (usersPage - 1) * usersPageSize + 1;
        const last = (usersPage - 1) * usersPageSize + totalReturned;
        const showPrev = usersPage > 1;
        const showNext = hasMore;
        usersPagination.innerHTML = `
            <div class="um-pagination-bar">
                <label class="um-pagination-size">
                    Per page
                    <select id="um-users-page-size" class="form-input um-page-size-select" aria-label="Users per page">
                        ${USERS_PAGE_SIZES.map(n => `<option value="${n}" ${n === usersPageSize ? 'selected' : ''}>${n}</option>`).join('')}
                    </select>
                </label>
                <span class="um-pagination-info">${totalReturned === 0 ? 'No users' : `Showing ${first}–${last}`}</span>
                <div class="um-pagination-buttons">
                    <button type="button" class="btn-small um-pagination-prev" ${showPrev ? '' : 'disabled'}>Previous</button>
                    <span class="um-pagination-page">Page ${usersPage}</span>
                    <button type="button" class="btn-small um-pagination-next" ${showNext ? '' : 'disabled'}>Next</button>
                </div>
            </div>
        `;
        const sizeSelect = el('um-users-page-size');
        if (sizeSelect) sizeSelect.addEventListener('change', () => { usersPageSize = parseInt(sizeSelect.value, 10); usersPage = 1; loadUsers(); });
        usersPagination.querySelector('.um-pagination-prev')?.addEventListener('click', () => { if (showPrev) { usersPage--; loadUsers(); } });
        usersPagination.querySelector('.um-pagination-next')?.addEventListener('click', () => { if (showNext) { usersPage++; loadUsers(); } });
    }

    async function loadUsers() {
        const search = el('um-user-search').value.trim();
        const first = (usersPage - 1) * usersPageSize;
        try {
            const list = await listUsers({ first, max: usersPageSize, search: search || undefined });
            const withGroups = await Promise.all((list || []).map(async (u) => {
                let groupNames = '';
                try {
                    const groups = await getUserGroups(u.id);
                    groupNames = (groups || []).map(g => g.name || g.path || g.id || '').filter(Boolean).join(', ') || '—';
                } catch (_) {
                    groupNames = '—';
                }
                return { ...u, groupNames };
            }));
            renderUsersTable(withGroups);
            const hasMore = (list || []).length >= usersPageSize;
            renderUsersPagination(hasMore, (list || []).length);
        } catch (e) {
            usersList.innerHTML = '<p class="um-error">' + escapeHtml(e.message) + '</p>';
            if (usersPagination) usersPagination.innerHTML = '';
        }
    }

    function renderUsersTable(list) {
        if (!list || list.length === 0) {
            usersList.innerHTML = '<p class="um-empty">No users found.</p>';
            return;
        }
        usersList.innerHTML = `
            <table class="um-table">
                <thead><tr>
                    <th>Username</th><th>Email</th><th>First</th><th>Last</th><th>Groups</th><th>Enabled</th><th></th>
                </tr></thead>
                <tbody>
                    ${list.map(u => `
                        <tr>
                            <td>${escapeHtml(u.username || '')}</td>
                            <td>${escapeHtml(u.email || '')}</td>
                            <td>${escapeHtml(u.firstName || '')}</td>
                            <td>${escapeHtml(u.lastName || '')}</td>
                            <td class="um-user-groups">${escapeHtml(u.groupNames != null ? u.groupNames : '—')}</td>
                            <td>${u.enabled !== false ? 'Yes' : 'No'}</td>
                            <td>
                                <button type="button" class="btn-small um-btn-edit" data-id="${escapeHtml(u.id)}">Edit</button>
                                <button type="button" class="btn-small um-btn-roles" data-id="${escapeHtml(u.id)}" data-username="${escapeHtml(u.username || '')}">Roles</button>
                                <button type="button" class="btn-small um-btn-reset-pw" data-id="${escapeHtml(u.id)}" data-username="${escapeHtml(u.username || '')}">Reset password</button>
                                <button type="button" class="btn-small um-btn-delete" data-id="${escapeHtml(u.id)}">Delete</button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
        usersList.querySelectorAll('.um-btn-edit').forEach(b => b.addEventListener('click', () => showUserForm(b.dataset.id)));
        usersList.querySelectorAll('.um-btn-roles').forEach(b => b.addEventListener('click', () => showRoleMappingModal(b.dataset.id, b.dataset.username)));
        usersList.querySelectorAll('.um-btn-reset-pw').forEach(b => b.addEventListener('click', () => showResetPasswordModal(b.dataset.id, b.dataset.username)));
        usersList.querySelectorAll('.um-btn-delete').forEach(b => b.addEventListener('click', () => confirmDeleteUser(b.dataset.id)));
    }

    function parseUserIdFromLocation(location) {
        if (!location || typeof location !== 'string') return null;
        const trimmed = location.replace(/\/$/, '');
        const segments = trimmed.split('/');
        return segments[segments.length - 1] || null;
    }

    async function showResetPasswordModal(userId, username) {
        const modal = el('um-modal-reset-password');
        if (!modal) return;
        modal.classList.remove('hidden');
        modal.dataset.userId = userId;
        modal.querySelector('.um-modal-reset-username').textContent = username || userId;
        modal.querySelector('#um-reset-password-value').value = '';
        modal.querySelector('#um-reset-password-temporary').checked = true;
        modal.querySelector('#um-reset-password-message').textContent = '';
    }

    async function saveResetPasswordForm() {
        const modal = el('um-modal-reset-password');
        if (!modal) return;
        const userId = modal.dataset.userId;
        const value = modal.querySelector('#um-reset-password-value').value;
        const temporary = modal.querySelector('#um-reset-password-temporary').checked;
        const messageEl = modal.querySelector('#um-reset-password-message');
        if (!value.trim()) {
            messageEl.textContent = 'Please enter a password.';
            showUmToast('Please enter a password.', 'error');
            return;
        }
        messageEl.textContent = '';
        try {
            await resetUserPassword(userId, { value: value.trim(), temporary });
            messageEl.textContent = 'Password set.';
            messageEl.className = 'um-message';
            showUmToast('Password set successfully.', 'success');
            setTimeout(() => { modal.classList.add('hidden'); }, 800);
        } catch (e) {
            messageEl.textContent = e.message || 'Failed to set password.';
            messageEl.className = 'um-message um-error';
            showUmToast(e.message || 'Failed to set password.', 'error');
        }
    }

    function showUserForm(userId) {
        const modal = el('um-modal-user');
        if (!modal) return;
        modal.classList.remove('hidden');
        modal.querySelector('#um-user-form-id').value = userId || '';
        const passwordSection = modal.querySelector('#um-user-password-section');
        if (userId) {
            if (passwordSection) passwordSection.style.display = 'none';
            getUser(userId).then(u => {
                modal.querySelector('#um-user-username').value = u.username || '';
                modal.querySelector('#um-user-email').value = u.email || '';
                modal.querySelector('#um-user-firstName').value = u.firstName || '';
                modal.querySelector('#um-user-lastName').value = u.lastName || '';
                modal.querySelector('#um-user-enabled').checked = u.enabled !== false;
                modal.querySelector('#um-user-username').readOnly = true;
            }).catch(e => { userMessage.textContent = e.message; });
        } else {
            if (passwordSection) passwordSection.style.display = '';
            modal.querySelector('#um-user-username').value = '';
            modal.querySelector('#um-user-email').value = '';
            modal.querySelector('#um-user-firstName').value = '';
            modal.querySelector('#um-user-lastName').value = '';
            modal.querySelector('#um-user-enabled').checked = true;
            modal.querySelector('#um-user-username').readOnly = false;
            modal.querySelector('#um-user-password').value = '';
            modal.querySelector('#um-user-password-temporary').checked = true;
        }
    }

    async function saveUserForm() {
        const id = el('um-user-form-id').value;
        const payload = {
            username: el('um-user-username').value.trim(),
            email: el('um-user-email').value.trim(),
            firstName: el('um-user-firstName').value.trim(),
            lastName: el('um-user-lastName').value.trim(),
            enabled: el('um-user-enabled').checked,
        };
        const passwordInput = el('um-user-password');
        const passwordTemporary = el('um-user-password-temporary');
        const passwordValue = passwordInput ? passwordInput.value : '';
        if (!payload.username && !id) {
            userMessage.textContent = 'Username is required.';
            showUmToast('Username is required.', 'error');
            return;
        }
        try {
            if (id) {
                await updateUser(id, payload);
                userMessage.textContent = 'User updated.';
                showUmToast('User updated successfully.', 'success');
            } else {
                const location = await createUser(payload);
                const newId = parseUserIdFromLocation(location);
                if (newId && passwordValue.trim()) {
                    await resetUserPassword(newId, { value: passwordValue.trim(), temporary: passwordTemporary ? passwordTemporary.checked : true });
                    userMessage.textContent = 'User created and password set.';
                    showUmToast('User created and password set successfully.', 'success');
                } else {
                    userMessage.textContent = 'User created.' + (passwordValue.trim() ? '' : ' Set password via Reset password.');
                    showUmToast('User created successfully.', 'success');
                }
            }
            el('um-modal-user').classList.add('hidden');
            loadUsers();
        } catch (e) {
            userMessage.textContent = e.message;
            showUmToast(e.message || 'Operation failed.', 'error');
        }
    }

    async function confirmDeleteUser(id) {
        if (!confirm('Delete this user? This cannot be undone.')) return;
        try {
            await deleteUser(id);
            userMessage.textContent = 'User deleted.';
            showUmToast('User deleted successfully.', 'success');
            loadUsers();
        } catch (e) {
            userMessage.textContent = e.message;
            showUmToast(e.message || 'Failed to delete user.', 'error');
        }
    }

    async function showRoleMappingModal(userId, username) {
        const modal = el('um-modal-roles');
        if (!modal) return;
        modal.classList.remove('hidden');
        modal.dataset.userId = userId;
        modal.querySelector('.um-modal-roles-username').textContent = username || userId;
        try {
            const [allRoles, userRoles] = await Promise.all([listClientRoles(clientId), getUserClientRoles(userId, clientId)]);
            const userRoleIds = new Set((userRoles || []).map(r => r.id));
            modal.querySelector('.um-modal-roles-list').innerHTML = (allRoles || []).map(r => `
                <label class="um-role-row">
                    <input type="checkbox" ${userRoleIds.has(r.id) ? 'checked' : ''} data-id="${escapeHtml(r.id)}" data-name="${escapeHtml(r.name || '')}">
                    <span>${escapeHtml(r.name || '')}</span> ${r.description ? '<small>' + escapeHtml(r.description) + '</small>' : ''}
                </label>
            `).join('');
        } catch (e) {
            modal.querySelector('.um-modal-roles-list').innerHTML = '<p class="um-error">' + escapeHtml(e.message) + '</p>';
        }
    }

    async function saveRoleMapping() {
        const modal = el('um-modal-roles');
        const userId = modal.dataset.userId;
        const checked = modal.querySelectorAll('.um-modal-roles-list input:checked');
        const unchecked = modal.querySelectorAll('.um-modal-roles-list input:not(:checked)');
        const toAdd = [];
        const toRemove = [];
        const allRoles = await listClientRoles(clientId);
        const userRoles = await getUserClientRoles(userId, clientId);
        const userRoleIds = new Set((userRoles || []).map(r => r.id));
        checked.forEach(cb => { if (!userRoleIds.has(cb.dataset.id)) toAdd.push({ id: cb.dataset.id, name: cb.dataset.name }); });
        unchecked.forEach(cb => { if (userRoleIds.has(cb.dataset.id)) toRemove.push({ id: cb.dataset.id }); });
        try {
            if (toAdd.length) await addUserClientRoles(userId, clientId, toAdd);
            if (toRemove.length) await removeUserClientRoles(userId, clientId, toRemove);
            modal.classList.add('hidden');
            userMessage.textContent = 'Roles updated.';
            showUmToast('User roles updated successfully.', 'success');
        } catch (e) {
            userMessage.textContent = e.message;
            showUmToast(e.message || 'Failed to update user roles.', 'error');
        }
    }

    // Roles tab
    // Roles tab: pagination state (client-side; API returns all roles)
    let rolesPage = 1;
    let rolesPageSize = 25;
    let rolesAll = [];
    const ROLES_PAGE_SIZES = [10, 25, 50, 100];

    rolesContainer.innerHTML = `
        <div class="um-toolbar">
            <button type="button" class="btn-primary" id="um-role-add">Add role</button>
        </div>
        <div id="um-roles-list" class="um-table-wrap"></div>
        <div id="um-roles-pagination" class="um-pagination"></div>
        <div id="um-role-message" class="um-message"></div>
    `;
    const rolesList = el('um-roles-list');
    const rolesPagination = el('um-roles-pagination');
    const roleMessage = el('um-role-message');
    el('um-role-add').addEventListener('click', () => showRoleForm());

    function renderRolesPagination(hasMore, totalReturned) {
        if (!rolesPagination) return;
        const first = totalReturned === 0 ? 0 : (rolesPage - 1) * rolesPageSize + 1;
        const last = (rolesPage - 1) * rolesPageSize + totalReturned;
        const showPrev = rolesPage > 1;
        const showNext = hasMore;
        rolesPagination.innerHTML = `
            <div class="um-pagination-bar">
                <label class="um-pagination-size">
                    Per page
                    <select id="um-roles-page-size" class="form-input um-page-size-select" aria-label="Roles per page">
                        ${ROLES_PAGE_SIZES.map(n => `<option value="${n}" ${n === rolesPageSize ? 'selected' : ''}>${n}</option>`).join('')}
                    </select>
                </label>
                <span class="um-pagination-info">${totalReturned === 0 ? 'No roles' : `Showing ${first}–${last}`}</span>
                <div class="um-pagination-buttons">
                    <button type="button" class="btn-small um-pagination-prev" ${showPrev ? '' : 'disabled'}>Previous</button>
                    <span class="um-pagination-page">Page ${rolesPage}</span>
                    <button type="button" class="btn-small um-pagination-next" ${showNext ? '' : 'disabled'}>Next</button>
                </div>
            </div>
        `;
        const sizeSelect = el('um-roles-page-size');
        if (sizeSelect) sizeSelect.addEventListener('change', () => { rolesPageSize = parseInt(sizeSelect.value, 10); rolesPage = 1; renderRolesPage(); });
        rolesPagination.querySelector('.um-pagination-prev')?.addEventListener('click', () => { if (showPrev) { rolesPage--; renderRolesPage(); } });
        rolesPagination.querySelector('.um-pagination-next')?.addEventListener('click', () => { if (showNext) { rolesPage++; renderRolesPage(); } });
    }

    function renderRolesPage() {
        const start = (rolesPage - 1) * rolesPageSize;
        const slice = rolesAll.slice(start, start + rolesPageSize);
        if (rolesAll.length === 0) {
            rolesList.innerHTML = '<p class="um-empty">No client roles. Create one with Add role.</p>';
            renderRolesPagination(false, 0);
            return;
        }
        rolesList.innerHTML = `
            <table class="um-table">
                <thead><tr><th>Name</th><th>Description</th><th></th></tr></thead>
                <tbody>
                    ${slice.map(r => `
                        <tr>
                            <td>${escapeHtml(r.name || '')}</td>
                            <td>${escapeHtml(r.description || '')}</td>
                            <td>
                                <button type="button" class="btn-small um-btn-edit-role" data-name="${escapeHtml(r.name || '')}">Edit</button>
                                <button type="button" class="btn-small um-btn-delete-role" data-name="${escapeHtml(r.name || '')}">Delete</button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;
        rolesList.querySelectorAll('.um-btn-edit-role').forEach(b => b.addEventListener('click', () => showRoleForm(b.dataset.name)));
        rolesList.querySelectorAll('.um-btn-delete-role').forEach(b => b.addEventListener('click', () => confirmDeleteRole(b.dataset.name)));
        const hasMore = rolesAll.length > rolesPage * rolesPageSize;
        renderRolesPagination(hasMore, slice.length, rolesAll.length);
    }

    async function loadRoles() {
        try {
            rolesAll = await listClientRoles(clientId) || [];
            rolesPage = 1;
            renderRolesPage();
        } catch (e) {
            rolesList.innerHTML = '<p class="um-error">' + escapeHtml(e.message) + '</p>';
            if (rolesPagination) rolesPagination.innerHTML = '';
        }
    }

    function showRoleForm(roleName) {
        const modal = el('um-modal-role');
        if (!modal) return;
        const titleEl = modal.querySelector('#um-modal-role-title');
        const submitBtn = modal.querySelector('#um-modal-role-submit');
        const nameInput = modal.querySelector('#um-role-name');
        const descInput = modal.querySelector('#um-role-description');
        modal.classList.remove('hidden');
        if (roleName) {
            modal.dataset.editingRole = roleName;
            if (titleEl) titleEl.textContent = 'Edit client role';
            if (submitBtn) submitBtn.textContent = 'Save';
            if (nameInput) nameInput.readOnly = true;
            getClientRoleByName(clientId, roleName).then(r => {
                if (nameInput) nameInput.value = r.name || roleName;
                if (descInput) descInput.value = r.description || '';
            }).catch(e => { roleMessage.textContent = e.message; showUmToast(e.message || 'Failed to load role.', 'error'); });
        } else {
            delete modal.dataset.editingRole;
            if (titleEl) titleEl.textContent = 'New client role';
            if (submitBtn) submitBtn.textContent = 'Create';
            if (nameInput) { nameInput.value = ''; nameInput.readOnly = false; }
            if (descInput) descInput.value = '';
        }
    }

    async function saveRoleForm() {
        const modal = el('um-modal-role');
        const name = el('um-role-name').value.trim();
        const description = el('um-role-description').value.trim();
        const editingRole = modal && modal.dataset.editingRole;
        if (!name) {
            roleMessage.textContent = 'Role name is required.';
            showUmToast('Role name is required.', 'error');
            return;
        }
        try {
            if (editingRole) {
                await updateClientRole(clientId, editingRole, { name: editingRole, description });
                el('um-modal-role').classList.add('hidden');
                roleMessage.textContent = 'Role updated.';
                showUmToast('Role updated successfully.', 'success');
            } else {
                await createClientRole(clientId, { name, description });
                el('um-modal-role').classList.add('hidden');
                roleMessage.textContent = 'Role created.';
                showUmToast('Role created successfully.', 'success');
            }
            loadRoles();
        } catch (e) {
            roleMessage.textContent = e.message;
            showUmToast(e.message || 'Failed to save role.', 'error');
        }
    }

    async function confirmDeleteRole(name) {
        if (!confirm('Delete role "' + name + '"?')) return;
        try {
            await deleteClientRole(clientId, name);
            roleMessage.textContent = 'Role deleted.';
            showUmToast('Role deleted successfully.', 'success');
            loadRoles();
        } catch (e) {
            roleMessage.textContent = e.message;
            showUmToast(e.message || 'Failed to delete role.', 'error');
        }
    }

    // Role Permissions tab: client roles only; dropdown + configs tree with checkboxes (folder/report access)
    let selectedRoleForPermissions = '';
    let configsTreeData = [];
    let rolePermissionsPaths = [];
    let rolePermissionsBound = false;

    async function loadPermissions() {
        let roles = [];
        let treeError = '';
        let rolesError = '';
        try {
            const treeRes = await getConfigsTree();
            configsTreeData = (treeRes && treeRes.tree) ? treeRes.tree : [];
        } catch (e) {
            treeError = e.message || 'Failed to load configs tree';
            configsTreeData = [];
        }
        try {
            const rolesRes = await listClientRoles(clientId);
            roles = Array.isArray(rolesRes) ? rolesRes : [];
        } catch (e) {
            rolesError = e.message || 'Failed to load client roles';
        }
        selectedRoleForPermissions = roles.length ? roles[0].name : '';

        const roleSelect = el('um-role-permissions-select');
        const roleNameEl = el('um-permissions-role-name');
        const treeContainer = el('um-configs-tree');
        const messageEl = el('um-role-permissions-message');

        if (roleSelect) {
            roleSelect.innerHTML = roles.length
                ? roles.map(r => `<option value="${escapeHtml(r.name || '')}">${escapeHtml(r.name || '')}</option>`).join('')
                : '<option value="">' + (rolesError ? escapeHtml(rolesError) : 'No client roles') + '</option>';
            roleSelect.value = selectedRoleForPermissions;
        }
        if (roleNameEl) roleNameEl.textContent = selectedRoleForPermissions;
        if (messageEl) {
            messageEl.textContent = '';
            if (rolesError) messageEl.textContent = rolesError;
        }
        if (treeContainer) {
            treeContainer.innerHTML = treeError ? '<p class="um-error">' + escapeHtml(treeError) + '</p>' : '';
        }

        if (!rolePermissionsBound && roleSelect) {
            rolePermissionsBound = true;
            roleSelect.addEventListener('change', async () => {
                selectedRoleForPermissions = roleSelect.value;
                if (roleNameEl) roleNameEl.textContent = selectedRoleForPermissions;
                await loadRolePermissionsForRole(selectedRoleForPermissions);
                renderConfigsTree(configsTreeData, rolePermissionsPaths);
            });
            const saveBtn = el('um-role-permissions-save');
            if (saveBtn) saveBtn.addEventListener('click', saveRolePermissionsClick);
        }

        await loadRolePermissionsForRole(selectedRoleForPermissions);
        if (!treeError && treeContainer) {
            renderConfigsTree(configsTreeData, rolePermissionsPaths);
        }
    }

    async function loadRolePermissionsForRole(role) {
        if (!role) {
            rolePermissionsPaths = [];
            const messageEl = el('um-role-permissions-message');
            if (messageEl && !messageEl.textContent) {
                messageEl.textContent = 'No client role selected.';
            }
            return;
        }
        try {
            const data = await getRolePermissions(role);
            rolePermissionsPaths = (data && data.paths) ? data.paths : [];
        } catch (_) {
            rolePermissionsPaths = [];
        }
    }

    function renderConfigsTree(tree, allowedPaths) {
        const container = el('um-configs-tree');
        if (!container) return;
        const allowedSet = new Set(allowedPaths || []);

        function renderNode(node, depth) {
            const path = node.id;
            const isFolder = node.type === 'folder';
            const checked = allowedSet.has(path) || (isFolder && allowedSet.has('*'));
            const indent = depth * 20;
            const hasChildren = isFolder && node.children && node.children.length > 0;
            // Default: collapsed (▶); click to expand (▼)
            const toggle = hasChildren ? `<span class="um-tree-toggle" data-expanded="false">▶</span>` : '<span class="um-tree-toggle um-tree-toggle-empty"></span>';
            const icon = isFolder ? '📁' : '📄';
            const childId = 'um-tree-' + path.replace(/\//g, '-').replace(/[^a-zA-Z0-9-_]/g, '_');
            let html = `
                <div class="um-tree-row" data-depth="${depth}" data-path="${escapeHtml(path)}" data-type="${node.type}">
                    <span style="padding-left: ${indent}px;" class="um-tree-cell">
                        ${toggle}
                        <label class="um-tree-label">
                            <input type="checkbox" class="um-tree-cb" data-path="${escapeHtml(path)}" ${checked ? 'checked' : ''}>
                            <span class="um-tree-icon">${icon}</span>
                            <span class="um-tree-name">${escapeHtml(node.name)}</span>
                        </label>
                    </span>
                </div>
            `;
            if (hasChildren) {
                html += `<div class="um-tree-children" id="${childId}" style="display: none;">`;
                node.children.forEach(child => { html += renderNode(child, depth + 1); });
                html += '</div>';
            }
            return html;
        }

        container.innerHTML = configsTreeData.map(node => renderNode(node, 0)).join('');

        container.querySelectorAll('.um-tree-toggle:not(.um-tree-toggle-empty)').forEach(toggle => {
            toggle.addEventListener('click', (e) => {
                e.stopPropagation();
                const row = toggle.closest('.um-tree-row');
                const path = (row && row.dataset.path) ? row.dataset.path.replace(/\//g, '-').replace(/[^a-zA-Z0-9-_]/g, '_') : '';
                const children = path ? container.querySelector('#um-tree-' + path) : null;
                if (!children) return;
                const expanded = toggle.getAttribute('data-expanded') === 'true';
                children.style.display = expanded ? 'none' : 'block';
                toggle.setAttribute('data-expanded', !expanded);
                toggle.textContent = expanded ? '▶' : '▼';
            });
        });

        container.querySelectorAll('.um-tree-cb').forEach(cb => {
            cb.addEventListener('change', () => {
                const path = cb.dataset.path;
                const row = cb.closest('.um-tree-row');
                if (cb.checked) {
                    allowedSet.add(path);
                } else {
                    allowedSet.delete(path);
                }
                if (row && row.dataset.type === 'folder') {
                    const childId = 'um-tree-' + path.replace(/\//g, '-').replace(/[^a-zA-Z0-9-_]/g, '_');
                    const children = container.querySelector('#' + childId);
                    if (children) {
                        children.querySelectorAll('.um-tree-cb').forEach(childCb => {
                            childCb.checked = cb.checked;
                            if (cb.checked) allowedSet.add(childCb.dataset.path);
                            else allowedSet.delete(childCb.dataset.path);
                        });
                    }
                }
            });
        });

        window._umRolePermissionsSet = allowedSet;

        const builderCb = el('um-perm-builder');
        const adminCb = el('um-perm-admin');
        if (builderCb) {
            builderCb.checked = allowedSet.has('__builder');
            builderCb.onchange = () => {
                if (builderCb.checked) allowedSet.add('__builder');
                else allowedSet.delete('__builder');
            };
        }
        if (adminCb) {
            adminCb.checked = allowedSet.has('__admin');
            adminCb.onchange = () => {
                if (adminCb.checked) allowedSet.add('__admin');
                else allowedSet.delete('__admin');
            };
        }

        const selectAllBtn = el('um-tree-select-all');
        const deselectAllBtn = el('um-tree-deselect-all');
        if (selectAllBtn) {
            selectAllBtn.onclick = () => {
                container.querySelectorAll('.um-tree-cb').forEach(cb => {
                    cb.checked = true;
                    allowedSet.add(cb.dataset.path);
                });
                if (builderCb) { builderCb.checked = true; allowedSet.add('__builder'); }
                if (adminCb) { adminCb.checked = true; allowedSet.add('__admin'); }
            };
        }
        if (deselectAllBtn) {
            deselectAllBtn.onclick = () => {
                container.querySelectorAll('.um-tree-cb').forEach(cb => { cb.checked = false; });
                allowedSet.clear();
                if (builderCb) builderCb.checked = false;
                if (adminCb) adminCb.checked = false;
            };
        }
    }

    async function saveRolePermissionsClick() {
        const set = window._umRolePermissionsSet;
        const paths = set ? Array.from(set) : [];
        const messageEl = el('um-role-permissions-message');
        try {
            await saveRolePermissions(selectedRoleForPermissions, paths);
            if (messageEl) messageEl.textContent = 'Permissions saved.';
            showUmToast('Role permissions saved successfully.', 'success');
        } catch (e) {
            if (messageEl) messageEl.textContent = e.message || 'Failed to save.';
            showUmToast(e.message || 'Failed to save permissions.', 'error');
        }
    }

    async function showGroupRolesModal(groupId, groupName) {
        const modal = el('um-modal-group-roles');
        if (!modal) return;
        modal.classList.remove('hidden');
        modal.dataset.groupId = groupId;
        modal.querySelector('.um-modal-group-roles-name').textContent = groupName || groupId;
        try {
            const [allRoles, groupRoles] = await Promise.all([listClientRoles(clientId), getGroupClientRoles(groupId, clientId)]);
            const groupRoleIds = new Set((groupRoles || []).map(r => r.id));
            modal.querySelector('.um-modal-group-roles-list').innerHTML = (allRoles || []).map(r => `
                <label class="um-role-row">
                    <input type="checkbox" ${groupRoleIds.has(r.id) ? 'checked' : ''} data-id="${escapeHtml(r.id)}" data-name="${escapeHtml(r.name || '')}">
                    <span>${escapeHtml(r.name || '')}</span>
                </label>
            `).join('');
        } catch (e) {
            modal.querySelector('.um-modal-group-roles-list').innerHTML = '<p class="um-error">' + escapeHtml(e.message) + '</p>';
        }
    }

    async function saveGroupRoleMapping() {
        const modal = el('um-modal-group-roles');
        const groupId = modal.dataset.groupId;
        const checked = modal.querySelectorAll('.um-modal-group-roles-list input:checked');
        const unchecked = modal.querySelectorAll('.um-modal-group-roles-list input:not(:checked)');
        const toAdd = [];
        const toRemove = [];
        const allRoles = await listClientRoles(clientId);
        const groupRoles = await getGroupClientRoles(groupId, clientId);
        const groupRoleIds = new Set((groupRoles || []).map(r => r.id));
        checked.forEach(cb => { if (!groupRoleIds.has(cb.dataset.id)) toAdd.push({ id: cb.dataset.id, name: cb.dataset.name }); });
        unchecked.forEach(cb => { if (groupRoleIds.has(cb.dataset.id)) toRemove.push({ id: cb.dataset.id }); });
        try {
            if (toAdd.length) await addGroupClientRoles(groupId, clientId, toAdd);
            if (toRemove.length) await removeGroupClientRoles(groupId, clientId, toRemove);
            modal.classList.add('hidden');
            const groupMessage = el('um-group-message');
            if (groupMessage) groupMessage.textContent = 'Group roles updated.';
            showUmToast('Group roles updated successfully.', 'success');
        } catch (e) {
            const groupMessage = el('um-group-message');
            if (groupMessage) groupMessage.textContent = e.message;
            showUmToast(e.message || 'Failed to update group roles.', 'error');
        }
    }

    // Groups tab: pagination state (server-side)
    let groupsPage = 1;
    let groupsPageSize = 25;
    const GROUPS_PAGE_SIZES = [10, 25, 50, 100];

    if (groupsContainer) {
        groupsContainer.innerHTML = `
            <div class="um-toolbar">
                <input type="search" id="um-group-search" class="form-input um-search" placeholder="Search groups...">
                <button type="button" class="btn-primary" id="um-group-add">Add group</button>
            </div>
            <div id="um-groups-list" class="um-table-wrap"></div>
            <div id="um-groups-pagination" class="um-pagination"></div>
            <div id="um-group-message" class="um-message"></div>
        `;
        const groupsList = el('um-groups-list');
        const groupsPagination = el('um-groups-pagination');
        const groupMessage = el('um-group-message');
        let groupSearchTimeout;
        el('um-group-search').addEventListener('input', () => {
            clearTimeout(groupSearchTimeout);
            groupSearchTimeout = setTimeout(() => { groupsPage = 1; loadGroups(); }, 300);
        });
        el('um-group-add').addEventListener('click', () => showGroupForm());

        function renderGroupsPagination(hasMore, totalReturned) {
            if (!groupsPagination) return;
            const first = totalReturned === 0 ? 0 : (groupsPage - 1) * groupsPageSize + 1;
            const last = (groupsPage - 1) * groupsPageSize + totalReturned;
            const showPrev = groupsPage > 1;
            const showNext = hasMore;
            groupsPagination.innerHTML = `
                <div class="um-pagination-bar">
                    <label class="um-pagination-size">
                        Per page
                        <select id="um-groups-page-size" class="form-input um-page-size-select" aria-label="Groups per page">
                            ${GROUPS_PAGE_SIZES.map(n => `<option value="${n}" ${n === groupsPageSize ? 'selected' : ''}>${n}</option>`).join('')}
                        </select>
                    </label>
                    <span class="um-pagination-info">${totalReturned === 0 ? 'No groups' : `Showing ${first}–${last}`}</span>
                    <div class="um-pagination-buttons">
                        <button type="button" class="btn-small um-pagination-prev" ${showPrev ? '' : 'disabled'}>Previous</button>
                        <span class="um-pagination-page">Page ${groupsPage}</span>
                        <button type="button" class="btn-small um-pagination-next" ${showNext ? '' : 'disabled'}>Next</button>
                    </div>
                </div>
            `;
            const sizeSelect = el('um-groups-page-size');
            if (sizeSelect) sizeSelect.addEventListener('change', () => { groupsPageSize = parseInt(sizeSelect.value, 10); groupsPage = 1; loadGroups(); });
            groupsPagination.querySelector('.um-pagination-prev')?.addEventListener('click', () => { if (showPrev) { groupsPage--; loadGroups(); } });
            groupsPagination.querySelector('.um-pagination-next')?.addEventListener('click', () => { if (showNext) { groupsPage++; loadGroups(); } });
        }

        async function loadGroups() {
            const search = el('um-group-search').value.trim();
            const first = (groupsPage - 1) * groupsPageSize;
            try {
                const list = await listGroups({ first, max: groupsPageSize, search: search || undefined }) || [];
                if (!list || list.length === 0) {
                    groupsList.innerHTML = '<p class="um-empty">No groups found.</p>';
                    renderGroupsPagination(false, 0);
                    return;
                }
                groupsList.innerHTML = `
                    <table class="um-table">
                        <thead><tr><th>Name</th><th>Path</th><th></th></tr></thead>
                        <tbody>
                            ${list.map(g => `
                                <tr>
                                    <td>${escapeHtml(g.name || '')}</td>
                                    <td>${escapeHtml(g.path || '')}</td>
                                    <td>
                                        <button type="button" class="btn-small um-btn-edit-group" data-id="${escapeHtml(g.id)}" data-name="${escapeHtml(g.name || '')}" data-path="${escapeHtml(g.path || '')}">Edit</button>
                                        <button type="button" class="btn-small um-btn-group-roles" data-id="${escapeHtml(g.id)}" data-name="${escapeHtml(g.name || '')}">Assign roles</button>
                                        <button type="button" class="btn-small um-btn-delete-group" data-id="${escapeHtml(g.id)}">Delete</button>
                                    </td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                `;
                groupsList.querySelectorAll('.um-btn-edit-group').forEach(b => b.addEventListener('click', () => showGroupForm(b.dataset.id, b.dataset.name, b.dataset.path)));
                groupsList.querySelectorAll('.um-btn-group-roles').forEach(b => b.addEventListener('click', () => showGroupRolesModal(b.dataset.id, b.dataset.name)));
                groupsList.querySelectorAll('.um-btn-delete-group').forEach(b => b.addEventListener('click', () => confirmDeleteGroup(b.dataset.id)));
                const hasMore = list.length >= groupsPageSize;
                renderGroupsPagination(hasMore, list.length);
            } catch (e) {
                groupsList.innerHTML = '<p class="um-error">' + escapeHtml(e.message) + '</p>';
                if (groupsPagination) groupsPagination.innerHTML = '';
            }
        }

        function showGroupForm(id, name, path) {
            const modal = el('um-modal-group');
            if (!modal) return;
            modal.classList.remove('hidden');
            el('um-group-form-id').value = id || '';
            el('um-group-name').value = name || '';
            el('um-group-path').value = path || (name ? '/' + (name || '').replace(/\s+/g, '-') : '');
        }

        async function saveGroupForm() {
            const id = el('um-group-form-id').value;
            const name = el('um-group-name').value.trim();
            const path = el('um-group-path').value.trim() || '/' + name.replace(/\s+/g, '-');
            if (!name) {
                groupMessage.textContent = 'Group name is required.';
                showUmToast('Group name is required.', 'error');
                return;
            }
            try {
                if (id) {
                    await updateGroup(id, { name, path });
                    groupMessage.textContent = 'Group updated.';
                    showUmToast('Group updated successfully.', 'success');
                } else {
                    await createGroup({ name, path });
                    groupMessage.textContent = 'Group created.';
                    showUmToast('Group created successfully.', 'success');
                }
                el('um-modal-group').classList.add('hidden');
                loadGroups();
            } catch (e) {
                groupMessage.textContent = e.message;
                showUmToast(e.message || 'Failed to save group.', 'error');
            }
        }

        async function confirmDeleteGroup(id) {
            if (!confirm('Delete this group? This cannot be undone.')) return;
            try {
                await deleteGroup(id);
                groupMessage.textContent = 'Group deleted.';
                showUmToast('Group deleted successfully.', 'success');
                loadGroups();
            } catch (e) {
                groupMessage.textContent = e.message;
                showUmToast(e.message || 'Failed to delete group.', 'error');
            }
        }

        const modalGroup = el('um-modal-group');
        if (modalGroup) {
            modalGroup.querySelector('.um-modal-save')?.addEventListener('click', saveGroupForm);
            modalGroup.querySelector('.um-modal-close')?.addEventListener('click', () => modalGroup.classList.add('hidden'));
            modalGroup.querySelector('.um-modal-backdrop')?.addEventListener('click', () => modalGroup.classList.add('hidden'));
        }
        const modalGroupRoles = el('um-modal-group-roles');
        if (modalGroupRoles) {
            modalGroupRoles.querySelector('.um-modal-save')?.addEventListener('click', saveGroupRoleMapping);
            modalGroupRoles.querySelector('.um-modal-close')?.addEventListener('click', () => modalGroupRoles.classList.add('hidden'));
            modalGroupRoles.querySelector('.um-modal-backdrop')?.addEventListener('click', () => modalGroupRoles.classList.add('hidden'));
        }

        loadGroups();
    }

    // Bind modals
    const modalUser = el('um-modal-user');
    if (modalUser) {
        modalUser.querySelector('.um-modal-save')?.addEventListener('click', saveUserForm);
        modalUser.querySelector('.um-modal-close')?.addEventListener('click', () => modalUser.classList.add('hidden'));
        modalUser.querySelector('.um-modal-backdrop')?.addEventListener('click', () => modalUser.classList.add('hidden'));
    }
    const modalResetPw = el('um-modal-reset-password');
    if (modalResetPw) {
        modalResetPw.querySelector('.um-modal-save')?.addEventListener('click', saveResetPasswordForm);
        modalResetPw.querySelector('.um-modal-close')?.addEventListener('click', () => modalResetPw.classList.add('hidden'));
        modalResetPw.querySelector('.um-modal-backdrop')?.addEventListener('click', () => modalResetPw.classList.add('hidden'));
    }
    const toastClose = document.querySelector('#um-toast .um-toast-close');
    if (toastClose) toastClose.addEventListener('click', hideUmToast);
    const modalRoles = el('um-modal-roles');
    if (modalRoles) {
        modalRoles.querySelector('.um-modal-save')?.addEventListener('click', saveRoleMapping);
        modalRoles.querySelector('.um-modal-close')?.addEventListener('click', () => modalRoles.classList.add('hidden'));
        modalRoles.querySelector('.um-modal-backdrop')?.addEventListener('click', () => modalRoles.classList.add('hidden'));
    }
    const modalRole = el('um-modal-role');
    if (modalRole) {
        modalRole.querySelector('.um-modal-save')?.addEventListener('click', saveRoleForm);
        modalRole.querySelector('.um-modal-close')?.addEventListener('click', () => modalRole.classList.add('hidden'));
        modalRole.querySelector('.um-modal-backdrop')?.addEventListener('click', () => modalRole.classList.add('hidden'));
    }

    loadUsers();
    loadRoles();
    loadPermissions();
}
