// Column Autocomplete Module
// Provides enhanced autocomplete for column inputs in the builder

import { reportState } from './state.js';

// Default column names when not mapped in report metadata
const DEFAULT_PERIOD = ['year', 'month', 'quarter', 'week'];
const DEFAULT_LOCATION = ['district', 'region', 'facility'];

/**
 * Categorize columns into Period, Location, and Other groups
 * Uses mapped column names from reportState metadata, falling back to defaults.
 * Exact matches only — no substring matching.
 */
function categorizeColumns(columns) {
    const tc = reportState.timeColumns || {};
    const lc = reportState.locationColumns || {};

    // Use mapped names if set, otherwise use defaults
    const periodNames = new Set(
        [tc.year || 'year', tc.month || 'month', tc.quarter || 'quarter', tc.week || 'week']
            .map(n => n.toLowerCase())
    );
    const locationNames = new Set(
        [lc.district || 'district', lc.region || 'region', lc.facility || 'facility']
            .map(n => n.toLowerCase())
    );

    const period = [];
    const location = [];
    const other = [];

    columns.forEach(col => {
        const nameLower = col.name.toLowerCase();
        if (periodNames.has(nameLower)) {
            period.push(col);
        } else if (locationNames.has(nameLower)) {
            location.push(col);
        } else {
            other.push(col);
        }
    });

    // Sort each group alphabetically
    period.sort((a, b) => a.name.localeCompare(b.name));
    location.sort((a, b) => a.name.localeCompare(b.name));
    other.sort((a, b) => a.name.localeCompare(b.name));

    return { period, location, other };
}

/**
 * Create an autocomplete dropdown for a given input element
 * @param {HTMLInputElement} input - The input element to attach autocomplete to
 * @param {Function} getColumns - Function that returns array of {name, type} column objects
 * @param {Object} options - Optional config: { categorized: boolean }
 */
export function createColumnAutocomplete(input, getColumns, options = {}) {
    if (!input || input.dataset.autocompleteAttached) return;
    input.dataset.autocompleteAttached = 'true';

    const categorized = options.categorized || false;

    // Remove datalist attribute if present (we're using custom dropdown)
    input.removeAttribute('list');

    // Create dropdown container - append to body for proper z-index handling
    const dropdown = document.createElement('div');
    dropdown.className = 'column-autocomplete-dropdown';
    dropdown.style.display = 'none';
    dropdown.style.position = 'fixed';
    document.body.appendChild(dropdown);

    let selectedIndex = -1;
    let filteredColumns = [];
    let allColumns = [];

    // Position dropdown below input
    function positionDropdown() {
        const rect = input.getBoundingClientRect();
        dropdown.style.top = `${rect.bottom + 2}px`;
        dropdown.style.left = `${rect.left}px`;
        dropdown.style.width = `${Math.max(rect.width, 250)}px`;
    }

    // Render categorized dropdown with group headers
    function renderCategorized() {
        const { period, location, other } = categorizeColumns(allColumns);
        let html = '';
        let idx = 0;

        filteredColumns = [];

        if (period.length > 0) {
            html += '<div class="autocomplete-group-header">Period</div>';
            period.forEach(col => {
                filteredColumns.push(col);
                html += renderItem(col, idx++, '');
            });
        }

        if (location.length > 0) {
            html += '<div class="autocomplete-group-header">Location</div>';
            location.forEach(col => {
                filteredColumns.push(col);
                html += renderItem(col, idx++, '');
            });
        }

        if (other.length > 0) {
            html += '<div class="autocomplete-group-header">Other</div>';
            // Show limited other columns to keep dropdown manageable
            const otherSlice = other.slice(0, Math.max(10, 20 - period.length - location.length));
            otherSlice.forEach(col => {
                filteredColumns.push(col);
                html += renderItem(col, idx++, '');
            });
            if (other.length > otherSlice.length) {
                html += `<div class="autocomplete-hint">Type to search ${other.length - otherSlice.length} more...</div>`;
            }
        }

        return html;
    }

    // Render a single autocomplete item
    function renderItem(col, index, searchToken) {
        return `<div class="autocomplete-item${index === selectedIndex ? ' selected' : ''}" data-index="${index}" data-value="${col.name}">
                <span class="autocomplete-column-name">${highlightMatch(col.name, searchToken)}</span>
                <span class="autocomplete-column-type">${col.type}</span>
            </div>`;
    }

    // Filter and show dropdown
    function updateDropdown() {
        allColumns = getColumns();

        // If no columns available, don't show dropdown
        if (!allColumns || allColumns.length === 0) {
            dropdown.style.display = 'none';
            return;
        }

        // Get the last token being typed (for expressions like "col1 + col")
        const tokens = input.value.split(/[\s\+\-\*\/\(\),]+/);
        const lastToken = tokens[tokens.length - 1].toLowerCase().trim();

        if (lastToken === '') {
            if (categorized) {
                // Show categorized view: Period, Location, then Other
                dropdown.innerHTML = renderCategorized();
            } else {
                // Show all columns when input is empty (sorted alphabetically)
                filteredColumns = [...allColumns].sort((a, b) => a.name.localeCompare(b.name));
                filteredColumns = filteredColumns.slice(0, 20);
                dropdown.innerHTML = filteredColumns.map((col, index) => renderItem(col, index, '')).join('');
            }
        } else {
            // Filter and sort by relevance (starts with > contains)
            const startsWithMatch = [];
            const containsMatch = [];

            allColumns.forEach(col => {
                const nameLower = col.name.toLowerCase();
                if (nameLower.startsWith(lastToken)) {
                    startsWithMatch.push(col);
                } else if (nameLower.includes(lastToken)) {
                    containsMatch.push(col);
                }
            });

            filteredColumns = [...startsWithMatch, ...containsMatch];
            filteredColumns = filteredColumns.slice(0, 20);
            dropdown.innerHTML = filteredColumns.map((col, index) => renderItem(col, index, lastToken)).join('');
        }

        if (filteredColumns.length === 0) {
            dropdown.innerHTML = '<div class="autocomplete-no-results">No matching columns</div>';
            positionDropdown();
            dropdown.style.display = 'block';
            return;
        }

        positionDropdown();
        dropdown.style.display = 'block';
        selectedIndex = -1;
    }

    // Highlight matching text
    function highlightMatch(text, match) {
        if (!match) return text;
        const regex = new RegExp(`(${escapeRegex(match)})`, 'gi');
        return text.replace(regex, '<strong>$1</strong>');
    }

    function escapeRegex(str) {
        return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    }

    // Select an item
    function selectItem(index) {
        if (index < 0 || index >= filteredColumns.length) return;

        const column = filteredColumns[index];
        const currentValue = input.value;

        // Find the position of the last token and replace it
        const tokens = currentValue.split(/(\s*[\+\-\*\/\(\),]\s*)/);
        tokens[tokens.length - 1] = column.name;
        input.value = tokens.join('');

        dropdown.style.display = 'none';
        selectedIndex = -1;
        input.focus();

        // Trigger change event
        input.dispatchEvent(new Event('change', { bubbles: true }));
    }

    // Hide dropdown
    function hideDropdown() {
        setTimeout(() => {
            dropdown.style.display = 'none';
            selectedIndex = -1;
        }, 150);
    }

    // Handle input changes
    function onInput() {
        updateDropdown();
    }

    // Event listeners
    input.addEventListener('input', onInput);
    input.addEventListener('focus', updateDropdown);
    input.addEventListener('blur', hideDropdown);

    input.addEventListener('keydown', (e) => {
        if (dropdown.style.display === 'none') return;

        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                selectedIndex = Math.min(selectedIndex + 1, filteredColumns.length - 1);
                updateSelection();
                break;
            case 'ArrowUp':
                e.preventDefault();
                selectedIndex = Math.max(selectedIndex - 1, -1);
                updateSelection();
                break;
            case 'Enter':
                if (selectedIndex >= 0) {
                    e.preventDefault();
                    selectItem(selectedIndex);
                }
                break;
            case 'Escape':
                dropdown.style.display = 'none';
                selectedIndex = -1;
                break;
            case 'Tab':
                if (selectedIndex >= 0) {
                    e.preventDefault();
                    selectItem(selectedIndex);
                }
                break;
        }
    });

    // Update selection highlight
    function updateSelection() {
        const items = dropdown.querySelectorAll('.autocomplete-item');
        items.forEach((item, i) => {
            item.classList.toggle('selected', i === selectedIndex);
        });

        // Scroll into view if needed
        if (selectedIndex >= 0 && items[selectedIndex]) {
            items[selectedIndex].scrollIntoView({ block: 'nearest' });
        }
    }

    // Click handler for dropdown items
    dropdown.addEventListener('mousedown', (e) => {
        e.preventDefault(); // Prevent blur from firing before click
        const item = e.target.closest('.autocomplete-item');
        if (item) {
            const index = parseInt(item.dataset.index, 10);
            selectItem(index);
        }
    });

    // Reposition on scroll
    const modalContent = input.closest('.modal-content');
    if (modalContent) {
        modalContent.addEventListener('scroll', () => {
            if (dropdown.style.display !== 'none') {
                positionDropdown();
            }
        });
    }

    return dropdown;
}

/**
 * Get columns for the currently selected table
 */
export function getTableColumns() {
    const tableSelect = document.getElementById('comp-table');
    if (!tableSelect || !tableSelect.value) return [];

    const table = tableSelect.value;
    const columns = reportState.availableColumns[table] || [];
    return columns;
}

/**
 * Initialize autocomplete on all column inputs in the aggregations list
 */
export function initAggregationAutocomplete() {
    const aggsList = document.getElementById('aggregations-list');
    if (!aggsList) return;

    // Find all column inputs without autocomplete
    const columnInputs = aggsList.querySelectorAll('.agg-column:not([data-autocomplete-attached])');
    columnInputs.forEach(input => {
        createColumnAutocomplete(input, getTableColumns);
    });
}

/**
 * Initialize autocomplete on all column inputs in the group by list
 */
export function initGroupByAutocomplete() {
    const groupByList = document.getElementById('groupby-list');
    if (!groupByList) return;

    const fieldInputs = groupByList.querySelectorAll('.groupby-field:not([data-autocomplete-attached])');
    fieldInputs.forEach(input => {
        createColumnAutocomplete(input, getTableColumns, { categorized: true });
    });
}

/**
 * Initialize autocomplete on the WHERE clause input
 */
export function initWhereAutocomplete() {
    const whereInput = document.getElementById('comp-where');
    if (whereInput && !whereInput.dataset.autocompleteAttached) {
        createColumnAutocomplete(whereInput, getTableColumns);
    }
}

/**
 * Initialize all autocomplete inputs
 * Call this when the component modal is opened or when new aggregation rows are added
 */
export function initAllAutocomplete() {
    initAggregationAutocomplete();
    initGroupByAutocomplete();
    initWhereAutocomplete();
}

/**
 * Reinitialize autocomplete after DOM changes (e.g., new aggregation added)
 */
export function refreshAutocomplete() {
    // Use setTimeout to ensure DOM is updated
    setTimeout(() => {
        initAllAutocomplete();
    }, 50);
}
