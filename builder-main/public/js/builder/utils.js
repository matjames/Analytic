// Utility Functions
// Toast notifications, HTML escaping, theme management

/**
 * Escape HTML to prevent XSS
 */
export function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

/**
 * Show error toast notification
 */
export function showError(message) {
    const toast = document.createElement('div');
    toast.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: #f44336;
        color: white;
        padding: 16px 24px;
        border-radius: 4px;
        z-index: 2000;
        font-weight: 500;
        box-shadow: 0 4px 12px rgba(244, 67, 54, 0.3);
        animation: slideIn 0.3s ease-out;
    `;
    toast.textContent = message;
    document.body.appendChild(toast);
    setTimeout(() => {
        toast.style.animation = 'slideOut 0.3s ease-out';
        setTimeout(() => toast.remove(), 300);
    }, 4000);
}

/**
 * Show success toast notification
 */
export function showSuccess(message) {
    const toast = document.createElement('div');
    toast.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: #4caf50;
        color: white;
        padding: 16px 24px;
        border-radius: 4px;
        z-index: 2000;
        font-weight: 500;
        box-shadow: 0 4px 12px rgba(76, 175, 80, 0.3);
        animation: slideIn 0.3s ease-out;
    `;
    toast.textContent = message;
    document.body.appendChild(toast);
    setTimeout(() => {
        toast.style.animation = 'slideOut 0.3s ease-out';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

/**
 * Show copy feedback on an element
 */
export function showCopyFeedback(elementId) {
    const el = document.getElementById(elementId);
    if (!el) return;
    el.textContent = '✓ Copied!';
    el.classList.remove('hidden');
    setTimeout(() => {
        el.textContent = '';
        el.classList.add('hidden');
    }, 2000);
}

