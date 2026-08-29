// Recent Tables Module
// Stores and retrieves recently used tables from localStorage

const STORAGE_KEY = 'builder_recent_tables';
const MAX_RECENT = 5;

/**
 * Get the list of recent tables
 * @returns {string[]} Array of "schema.table" strings
 */
export function getRecentTables() {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (!stored) return [];
        const tables = JSON.parse(stored);
        return Array.isArray(tables) ? tables : [];
    } catch (e) {
        console.warn('Failed to load recent tables:', e);
        return [];
    }
}

/**
 * Add a table to the recent list
 * @param {string} table - The "schema.table" string
 */
export function addRecentTable(table) {
    if (!table || typeof table !== 'string') return;

    try {
        let tables = getRecentTables();

        // Remove if already exists (we'll add it to the front)
        tables = tables.filter(t => t.toLowerCase() !== table.toLowerCase());

        // Add to front
        tables.unshift(table);

        // Keep only the most recent
        tables = tables.slice(0, MAX_RECENT);

        localStorage.setItem(STORAGE_KEY, JSON.stringify(tables));
    } catch (e) {
        console.warn('Failed to save recent table:', e);
    }
}

/**
 * Render table options HTML with recent tables section
 * @param {string[]} allTables - All available tables
 * @param {string} selectedTable - Currently selected table (if any)
 * @returns {string} HTML options string
 */
export function renderTableOptionsWithRecent(allTables, selectedTable = '') {
    const recentTables = getRecentTables();
    const selectedLower = selectedTable?.toLowerCase() || '';

    let html = '<option value="">-- Select Table --</option>';

    // Filter recent tables to only those that exist in available tables
    const availableRecent = recentTables.filter(recent =>
        allTables.some(t => t.toLowerCase() === recent.toLowerCase())
    );

    // Add recent section if there are any
    if (availableRecent.length > 0) {
        html += '<optgroup label="Recent">';
        availableRecent.forEach(table => {
            const isSelected = table.toLowerCase() === selectedLower;
            html += `<option value="${table}" ${isSelected ? 'selected' : ''}>${table}</option>`;
        });
        html += '</optgroup>';

        // Add all tables section (excluding recent ones to avoid duplicates)
        const otherTables = allTables.filter(t =>
            !availableRecent.some(r => r.toLowerCase() === t.toLowerCase())
        );

        if (otherTables.length > 0) {
            html += '<optgroup label="All Tables">';
            otherTables.forEach(table => {
                const isSelected = table.toLowerCase() === selectedLower;
                html += `<option value="${table}" ${isSelected ? 'selected' : ''}>${table}</option>`;
            });
            html += '</optgroup>';
        }
    } else {
        // No recent tables, just show all
        allTables.forEach(table => {
            const isSelected = table.toLowerCase() === selectedLower;
            html += `<option value="${table}" ${isSelected ? 'selected' : ''}>${table}</option>`;
        });
    }

    return html;
}
