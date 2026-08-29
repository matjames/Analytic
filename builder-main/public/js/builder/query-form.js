// Query Form Module
// Handles aggregations, calculations, colors, select columns in the component form

import { currentEditingComponent, reportState } from './state.js';
import { refreshAutocomplete } from './autocomplete.js';

// Color palette for auto-assigning distinct colors to aggregations and calculations
const COLOR_PALETTE = [
    '#0f766e', // Teal
    '#e74c3c', // Red
    '#2ecc71', // Green
    '#f39c12', // Orange
    '#9b59b6', // Purple
    '#1abc9c', // Teal
    '#e91e63', // Pink
    '#00bcd4', // Cyan
    '#ff9800', // Amber
    '#795548', // Brown
];

/**
 * Get a color from the palette based on index
 * @param {number} index - The index of the item
 * @returns {string} A hex color code
 */
function getColorFromPalette(index) {
    return COLOR_PALETTE[index % COLOR_PALETTE.length];
}

// Time-based groupBy formats that benefit from periodLimit auto-expansion
// 'year' excluded — periodLimit expands backward within a year
// 'date' excluded — raw date, not periodic grouping
const TIME_BASED_FORMATS = new Set([
    'month', 'quarter', 'week', 'month_noyear', 'week_noyear', 'quarter_noyear'
]);

/**
 * Add a new aggregation row
 */
export function addAggregation() {
    const aggsList = document.getElementById('aggregations-list');
    if (!aggsList) return;

    const newAgg = document.createElement('div');
    newAgg.style.cssText = 'display: flex; gap: 8px; margin-bottom: 8px; align-items: center;';
    newAgg.innerHTML = `
        <input type="text" class="form-input agg-column" placeholder="Column or expression (e.g., col1 + col2)" list="column-suggestions" style="flex: 1; margin: 0;">
        <select class="form-input agg-function" style="flex: 0.5; margin: 0;">
            <option value="sum">SUM</option>
            <option value="count">COUNT</option>
            <option value="count_distinct">COUNT DISTINCT</option>
            <option value="avg">AVG</option>
            <option value="max">MAX</option>
            <option value="min">MIN</option>
            <option value="variance">VARIANCE</option>
            <option value="stddev">STDDEV</option>
        </select>
        <input type="text" class="form-input agg-alias" placeholder="Alias" style="flex: 0.8; margin: 0;">
        <input type="number" class="form-input agg-roundto" value="1" placeholder="Round" min="0" max="6" style="flex: 0.4; margin: 0;" title="Decimal places (default 1; ignored for COUNT)">
        <button class="btn-small btn-remove-agg" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">×</button>
    `;

    // Wire up remove button
    const removeBtn = newAgg.querySelector('.btn-remove-agg');
    removeBtn.addEventListener('click', () => {
        newAgg.remove();
        refreshSelectColumnsList();
        refreshCalcAliasChips();
    });

    // Wire up alias change to update select columns
    const aliasInput = newAgg.querySelector('.agg-alias');
    if (aliasInput) {
        aliasInput.addEventListener('change', () => {
            refreshSelectColumnsList();
            refreshCalcAliasChips();
        });
    }

    aggsList.appendChild(newAgg);
    refreshSelectColumnsList();
    refreshCalcAliasChips();

    // Initialize autocomplete on the new column input
    refreshAutocomplete();

    // Run inline validation after adding
    if (window.runInlineValidation) window.runInlineValidation();
}

/**
 * Add a new group by row
 */
export function addGroupBy() {
    const groupByList = document.getElementById('groupby-list');
    if (!groupByList) return;

    // Remove the "no groupby" placeholder if present
    const placeholder = groupByList.querySelector('#no-groupby-msg');
    if (placeholder) placeholder.remove();

    const newRow = document.createElement('div');
    newRow.className = 'groupby-row';
    newRow.style.cssText = 'display: flex; gap: 8px; margin-bottom: 8px; align-items: center; flex-wrap: wrap;';
    newRow.innerHTML = `
        <input type="text" class="form-input groupby-field" placeholder="Column name" list="column-suggestions" style="flex: 1; min-width: 120px; margin: 0;">
        <select class="form-input groupby-format" style="flex: 0 0 140px; margin: 0;">
            <optgroup label="Period">
                <option value="month">Month</option>
                <option value="quarter">Quarter</option>
                <option value="week">Week</option>
                <option value="year">Year</option>
                <option value="date">Date</option>
            </optgroup>
            <optgroup label="Period (no year)">
                <option value="month_noyear">Month (no year)</option>
                <option value="week_noyear">Week (no year)</option>
                <option value="quarter_noyear">Quarter (no year)</option>
            </optgroup>
            <optgroup label="Other">
                <option value="">No Format</option>
            </optgroup>
        </select>
        <input type="text" class="form-input groupby-alias" placeholder="Alias" style="flex: 0 0 100px; margin: 0;">
        <button class="btn-small btn-remove-groupby" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">×</button>
    `;

    // Wire up remove button
    const removeBtn = newRow.querySelector('.btn-remove-groupby');
    removeBtn.addEventListener('click', () => {
        newRow.remove();
        refreshSelectColumnsList();
    });

    // Wire up alias change to update select columns
    const aliasInput = newRow.querySelector('.groupby-alias');
    if (aliasInput) {
        aliasInput.addEventListener('change', () => {
            refreshSelectColumnsList();
        });
    }

    // Wire up format change to auto-set periodLimit for time-based formats
    const formatSelect = newRow.querySelector('.groupby-format');
    if (formatSelect) {
        formatSelect.addEventListener('change', () => {
            maybeSetDefaultPeriodLimit(formatSelect.value);
        });
    }

    groupByList.appendChild(newRow);

    // Initialize autocomplete on the new field input
    refreshAutocomplete();

    refreshSelectColumnsList();

    // Run inline validation after adding
    if (window.runInlineValidation) window.runInlineValidation();
}

/**
 * Add a new calculation row
 */
export function addCalculation() {
    const calcList = document.getElementById('calculations-list');
    if (!calcList) return;

    // Remove the "no calculations" placeholder if present
    const placeholder = calcList.querySelector('p');
    if (placeholder) placeholder.remove();

    const newCalcRow = document.createElement('div');
    newCalcRow.innerHTML = renderCalculationRow();
    const calcRow = newCalcRow.firstElementChild;

    // Wire up remove button
    const removeBtn = calcRow.querySelector('.btn-remove-calc');
    if (removeBtn) {
        removeBtn.addEventListener('click', () => {
            calcRow.remove();
            refreshSelectColumnsList();
        });
    }

    calcList.appendChild(calcRow);
    refreshCalcAliasChips();
    wireUpCalcRow(calcRow);
    refreshSelectColumnsList();
}

/**
 * Wire up template buttons, alias chip clicks, and formula validation for a calc row.
 */
function wireUpCalcRow(row) {
    // Template buttons
    row.querySelectorAll('.btn-calc-template').forEach(btn => {
        btn.addEventListener('click', () => {
            const formulaInput = row.querySelector('.calc-formula');
            if (formulaInput) {
                formulaInput.value = btn.dataset.template;
                formulaInput.focus();
                validateCalcFormula(row);
            }
        });
    });

    // Alias chip click — insert at cursor
    row.querySelector('.calc-alias-chips')?.addEventListener('click', (e) => {
        const chip = e.target.closest('.calc-chip');
        if (!chip) return;
        const formulaInput = row.querySelector('.calc-formula');
        if (!formulaInput) return;
        const alias = chip.dataset.alias;
        const start = formulaInput.selectionStart;
        const end = formulaInput.selectionEnd;
        const val = formulaInput.value;
        formulaInput.value = val.substring(0, start) + alias + val.substring(end);
        const newPos = start + alias.length;
        formulaInput.setSelectionRange(newPos, newPos);
        formulaInput.focus();
    });

    // Validate on blur
    const formulaInput = row.querySelector('.calc-formula');
    if (formulaInput) {
        formulaInput.addEventListener('blur', () => validateCalcFormula(row));
    }
}

/**
 * Wire up all existing calculation rows (called on form init).
 */
export function wireUpExistingCalcRows() {
    document.querySelectorAll('#calculations-list .calculation-row').forEach(row => {
        wireUpCalcRow(row);
        // Wire up remove button for pre-rendered rows
        const removeBtn = row.querySelector('.btn-remove-calc');
        if (removeBtn) {
            removeBtn.addEventListener('click', () => {
                row.remove();
                refreshSelectColumnsList();
            });
        }
    });
    refreshCalcAliasChips();
}

/**
 * Render a calculation row HTML
 */
export function renderCalculationRow(calc = {}) {
    return `
        <div class="calculation-row" style="border: 1px solid var(--color-border); border-radius: 4px; padding: 10px; margin-bottom: 10px; background: var(--color-bg-secondary);">
            <div class="calc-templates" style="display: flex; gap: 4px; margin-bottom: 8px; flex-wrap: wrap;">
                <button type="button" class="btn-calc-template" data-template="(A / B * 100)" style="font-size: 11px; padding: 2px 8px; border: 1px solid var(--color-border); border-radius: 3px; background: var(--color-bg); cursor: pointer;">% A/B×100</button>
                <button type="button" class="btn-calc-template" data-template="(A / B * 1000)" style="font-size: 11px; padding: 2px 8px; border: 1px solid var(--color-border); border-radius: 3px; background: var(--color-bg); cursor: pointer;">Rate A/B×1000</button>
                <button type="button" class="btn-calc-template" data-template="((A - B) / A * 100)" style="font-size: 11px; padding: 2px 8px; border: 1px solid var(--color-border); border-radius: 3px; background: var(--color-bg); cursor: pointer;">Dropout (A-B)/A×100</button>
            </div>
            <div class="calc-alias-chips" style="display: flex; gap: 4px; margin-bottom: 6px; flex-wrap: wrap; min-height: 20px;"></div>
            <div style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center;">
                <input type="text" class="form-input calc-formula" value="${calc.formula || ''}" placeholder="Formula: (alias1 / alias2 * 100)" style="flex: 2; margin: 0;">
                <input type="text" class="form-input calc-alias" value="${calc.resultAlias || ''}" placeholder="Result alias" style="flex: 1; margin: 0;">
                <button class="btn-small btn-remove-calc" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">×</button>
            </div>
            <div class="calc-validation-msg" style="display: none; font-size: 11px; margin-bottom: 6px; padding: 4px 8px; border-radius: 3px;"></div>
            <div style="display: flex; gap: 8px; align-items: center;">
                <input type="number" class="form-input calc-roundto" value="${calc.roundTo != null ? calc.roundTo : 1}" placeholder="Round to" style="flex: 1; margin: 0;" title="Decimal places">
                <input type="text" class="form-input calc-whenzero" value="${calc.whenZero || ''}" placeholder="When zero" style="flex: 1; margin: 0;" title="Value when denominator is zero">
            </div>
        </div>
    `;
}

/**
 * Refresh alias chips for all calculation rows.
 * Shows clickable badges for available aggregation/groupBy aliases.
 */
export function refreshCalcAliasChips() {
    const columns = getAvailableColumns();
    document.querySelectorAll('#calculations-list .calc-alias-chips').forEach(container => {
        if (columns.length === 0) {
            container.innerHTML = '<span style="font-size: 11px; color: var(--color-text-secondary);">Add aggregations above to use in formulas</span>';
            return;
        }
        container.innerHTML = columns.map(col =>
            `<span class="calc-chip" style="font-size: 11px; padding: 2px 8px; border-radius: 10px; background: var(--color-primary, #0f766e); color: white; cursor: pointer; user-select: none;" data-alias="${col}">${col}</span>`
        ).join('');
    });
}

/**
 * Validate a calculate formula and show inline feedback.
 * @param {HTMLElement} row - The .calculation-row element
 */
export function validateCalcFormula(row) {
    const formulaInput = row.querySelector('.calc-formula');
    const msgEl = row.querySelector('.calc-validation-msg');
    if (!formulaInput || !msgEl) return;

    const formula = formulaInput.value.trim();
    if (!formula) {
        msgEl.style.display = 'none';
        return;
    }

    const columns = getAvailableColumns();

    // Check unmatched parentheses
    const opens = (formula.match(/\(/g) || []).length;
    const closes = (formula.match(/\)/g) || []).length;
    if (opens !== closes) {
        showCalcMsg(msgEl, 'red', `Unmatched parentheses: ${opens} open, ${closes} close`);
        return;
    }

    // Extract identifiers (words that aren't numbers or operators)
    const identifiers = formula.match(/\b[a-zA-Z_][a-zA-Z0-9_]*\b/g) || [];
    const unknown = identifiers.filter(id => !columns.includes(id));
    if (unknown.length > 0 && columns.length > 0) {
        const suggestions = unknown.map(u => {
            const closest = findClosestAlias(u, columns);
            return closest ? `"${u}" — did you mean "${closest}"?` : `"${u}" not found`;
        });
        showCalcMsg(msgEl, '#b45309', suggestions.join('; '));
        return;
    }

    msgEl.style.display = 'none';
}

function showCalcMsg(el, color, text) {
    el.style.display = 'block';
    el.style.color = color;
    el.style.background = color === 'red' ? '#fef2f2' : '#fffbeb';
    el.textContent = text;
}

/**
 * Simple closest-match finder using character overlap.
 */
function findClosestAlias(input, aliases) {
    if (aliases.length === 0) return null;
    const lower = input.toLowerCase();
    let best = null;
    let bestScore = 0;
    for (const alias of aliases) {
        const al = alias.toLowerCase();
        // Simple: count matching characters at each position
        let score = 0;
        const minLen = Math.min(lower.length, al.length);
        for (let i = 0; i < minLen; i++) {
            if (lower[i] === al[i]) score++;
        }
        // Bonus for substring match
        if (al.includes(lower) || lower.includes(al)) score += 3;
        if (score > bestScore) {
            bestScore = score;
            best = alias;
        }
    }
    return bestScore >= 2 ? best : null;
}

/**
 * Get available column names from the current form state (aggregations, groupBy, calculations).
 * Returns an array of column name strings.
 */
export function getAvailableColumns() {
    const columns = [];

    // GroupBy aliases
    document.querySelectorAll('#groupby-list .groupby-row').forEach(row => {
        const field = row.querySelector('.groupby-field')?.value?.trim();
        const alias = row.querySelector('.groupby-alias')?.value?.trim();
        if (field) columns.push(alias || field);
    });

    // Aggregation aliases
    document.querySelectorAll('#aggregations-list > div').forEach(row => {
        const alias = row.querySelector('.agg-alias')?.value?.trim();
        if (alias) columns.push(alias);
    });

    // Calculation aliases
    document.querySelectorAll('#calculations-list .calculation-row').forEach(row => {
        const alias = row.querySelector('.calc-alias')?.value?.trim();
        if (alias) columns.push(alias);
    });

    return columns;
}

function isChoroplethCustomSQLMode(componentType = '') {
    if (componentType !== 'choropleth') return false;

    const sqlToggle = document.getElementById('choropleth-sql-enabled');
    if (sqlToggle) return sqlToggle.checked;

    return !!currentEditingComponent?.component?.sql;
}

function getStoredChoroplethSQLPreviewHeaders() {
    const section = document.getElementById('section-select-columns');
    if (!section?.dataset.choroplethSqlHeaders) return [];

    try {
        const parsed = JSON.parse(section.dataset.choroplethSqlHeaders);
        return Array.isArray(parsed) ? parsed : [];
    } catch {
        return [];
    }
}

function getChoroplethSQLDistrictColumn() {
    return document.getElementById('choropleth-district-column')?.value?.trim()
        || currentEditingComponent?.component?.columnMapping?.district
        || '';
}

function getChoroplethSQLValueColumn() {
    return document.getElementById('choropleth-value-column')?.value?.trim()
        || currentEditingComponent?.component?.columnMapping?.value
        || '';
}

/**
 * Render select columns checkboxes
 */
export function renderSelectColumnsCheckboxes(query, componentType = 'bar') {
    if (isChoroplethCustomSQLMode(componentType)) {
        const districtColumn = getChoroplethSQLDistrictColumn();
        const currentValueColumn = getChoroplethSQLValueColumn();
        const availableColumns = getStoredChoroplethSQLPreviewHeaders().filter(col => col && col !== districtColumn);

        if (currentValueColumn && currentValueColumn !== districtColumn && !availableColumns.includes(currentValueColumn)) {
            availableColumns.push(currentValueColumn);
        }

        if (availableColumns.length === 0) {
            return '<p style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">Run SQL Preview to load available SQL columns, then choose one value column for the map.</p>';
        }

        const selectedColumns = new Set(currentValueColumn ? [currentValueColumn] : []);
        return availableColumns.map(col => `
            <div class="select-column-row" data-column="${col}" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center; padding: 8px; background: var(--color-bg-secondary); border-radius: 4px;">
                <input type="checkbox" class="select-column-checkbox" value="${col}" ${selectedColumns.has(col) ? 'checked' : ''}>
                <label style="margin: 0; cursor: pointer; flex: 1; font-size: 12px;">${col}</label>
                <span style="font-size: 10px; color: var(--color-text-secondary);">(map value)</span>
            </div>
        `).join('');
    }

    const availableColumns = [];
    const columnMeta = {};

    // Add all groupBy columns
    if (query.groupBy && query.groupBy.length > 0) {
        query.groupBy.forEach((gb, i) => {
            const gbAlias = gb.alias || gb.field || `label${i > 0 ? i + 1 : ''}`;
            availableColumns.push(gbAlias);
            columnMeta[gbAlias] = { isGroupBy: true, groupByIndex: i };
        });
    }

    // Add aggregation aliases
    let colorIndex = 0;
    if (query.aggregations && query.aggregations.length > 0) {
        query.aggregations.forEach((agg, i) => {
            if (agg.alias) {
                availableColumns.push(agg.alias);
                columnMeta[agg.alias] = {
                    color: agg.color || getColorFromPalette(colorIndex),
                    chartType: agg.chartType || (i === 0 ? 'bar' : 'line'),
                    isGroupBy: false
                };
                colorIndex++;
            }
        });
    }

    // Add calculated fields
    if (query.calculate && query.calculate.length > 0) {
        query.calculate.forEach(calc => {
            const alias = calc.resultAlias || 'value';
            availableColumns.push(alias);
            columnMeta[alias] = {
                color: calc.color || getColorFromPalette(colorIndex),
                chartType: calc.chartType || 'line',
                isGroupBy: false
            };
            colorIndex++;
        });
    }

    if (availableColumns.length === 0) {
        return '<p style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">Add aggregations or calculations to select columns.</p>';
    }

    // Default to all columns selected
    const hasExplicitSelection = query.selectColumns && query.selectColumns.length > 0;
    const selectedColumns = hasExplicitSelection ? new Set(query.selectColumns) : new Set(availableColumns);

    const isBarLine = componentType === 'bar_line';

    return availableColumns.map(col => {
        const meta = columnMeta[col] || {};
        const isGroupBy = meta.isGroupBy;

        if (isGroupBy) {
            return `
                <div class="select-column-row" data-column="${col}" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center; padding: 8px; background: var(--color-bg-secondary); border-radius: 4px;">
                    <input type="checkbox" class="select-column-checkbox" value="${col}"
                           ${selectedColumns.has(col) ? 'checked' : ''}>
                    <label style="margin: 0; cursor: pointer; flex: 1; font-size: 12px;">${col}</label>
                    <span style="font-size: 10px; color: var(--color-text-secondary);">(x-axis)</span>
                </div>
            `;
        }

        const chartTypeSelector = isBarLine ? `
            <select class="select-column-charttype" style="width: 70px; padding: 4px; font-size: 11px;">
                <option value="bar" ${meta.chartType === 'bar' ? 'selected' : ''}>Bar</option>
                <option value="line" ${meta.chartType === 'line' ? 'selected' : ''}>Line</option>
            </select>
        ` : '';

        return `
            <div class="select-column-row" data-column="${col}" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center; padding: 8px; background: var(--color-bg-secondary); border-radius: 4px;">
                <input type="checkbox" class="select-column-checkbox" value="${col}"
                       ${selectedColumns.has(col) ? 'checked' : ''}>
                <label style="margin: 0; cursor: pointer; flex: 1; font-size: 12px;">${col}</label>
                ${chartTypeSelector}
                <input type="color" class="select-column-color" value="${meta.color}" style="width: 40px; height: 28px; padding: 2px; margin: 0;" title="Chart color">
            </div>
        `;
    }).join('');
}

/**
 * Refresh the select columns list
 */
export function refreshSelectColumnsList() {
    const selectColumnsList = document.getElementById('select-columns-list');
    if (!selectColumnsList) return;

    const compType = document.getElementById('comp-type')?.value || 'bar';

    // Preserve current selections from DOM
    const currentSelections = {};
    selectColumnsList.querySelectorAll('.select-column-row').forEach(row => {
        const col = row.dataset.column;
        if (col) {
            currentSelections[col] = {
                checked: row.querySelector('.select-column-checkbox')?.checked ?? true,
                color: row.querySelector('.select-column-color')?.value,
                chartType: row.querySelector('.select-column-charttype')?.value
            };
        }
    });

    // Gather current query state from the form
    const query = {
        groupBy: [],
        aggregations: [],
        calculate: [],
        selectColumns: []
    };

    // Get all groupBy fields
    document.querySelectorAll('#groupby-list .groupby-row').forEach(row => {
        const field = row.querySelector('.groupby-field')?.value;
        const format = row.querySelector('.groupby-format')?.value || undefined;
        const alias = row.querySelector('.groupby-alias')?.value;
        if (field) {
            query.groupBy.push({ field: field, format: format, alias: alias || field });
        }
    });

    // Get aggregations
    let colorIdx = 0;
    document.querySelectorAll('#aggregations-list > div').forEach((row, i) => {
        const alias = row.querySelector('.agg-alias')?.value;
        if (alias) {
            const preserved = currentSelections[alias];
            const existingAgg = currentEditingComponent?.component?.query?.aggregations?.find(a => a.alias === alias);
            const existingDs = currentEditingComponent?.component?.data?.datasets?.find(d => d.label === alias);
            query.aggregations.push({
                alias,
                color: preserved?.color || existingAgg?.color || existingDs?.color || getColorFromPalette(colorIdx),
                chartType: preserved?.chartType || existingAgg?.chartType || (i === 0 ? 'bar' : 'line')
            });
            colorIdx++;
        }
    });

    // Get calculations
    document.querySelectorAll('.calculation-row').forEach(row => {
        const alias = row.querySelector('.calc-alias')?.value;
        if (alias) {
            const preserved = currentSelections[alias];
            const existingCalc = currentEditingComponent?.component?.query?.calculate?.find(c => c.resultAlias === alias);
            const existingDs = currentEditingComponent?.component?.data?.datasets?.find(d => d.label === alias);
            query.calculate.push({
                resultAlias: alias,
                color: preserved?.color || existingCalc?.color || existingDs?.color || getColorFromPalette(colorIdx),
                chartType: preserved?.chartType || existingCalc?.chartType || 'line'
            });
            colorIdx++;
        }
    });

    // Preserve checked columns
    const currentlyChecked = Object.entries(currentSelections)
        .filter(([_, sel]) => sel.checked)
        .map(([col, _]) => col);
    if (currentlyChecked.length > 0) {
        query.selectColumns = currentlyChecked;
    }

    selectColumnsList.innerHTML = renderSelectColumnsCheckboxes(query, compType);

    if (window.refreshBuilderHeaderTooltipSuggestions) {
        window.refreshBuilderHeaderTooltipSuggestions();
    }
}

/**
 * Populate column suggestions datalist from available columns
 * Used by WHERE clause and aggregation column inputs
 */
export function populateColumnSuggestions() {
    const datalist = document.getElementById('column-suggestions');
    const tableSelect = document.getElementById('comp-table');
    if (!datalist || !tableSelect) return;

    const table = tableSelect.value;
    const columns = reportState.availableColumns[table] || [];

    datalist.innerHTML = columns.map(col =>
        `<option value="${col.name}">${col.name} (${col.type})</option>`
    ).join('');
}

// Generic toggle for collapsible sections
export function toggleSection(contentId, iconId, onOpen) {
    const content = document.getElementById(contentId);
    const icon = document.getElementById(iconId);
    if (content && icon) {
        const isHidden = content.style.display === 'none';
        content.style.display = isHidden ? 'block' : 'none';
        icon.textContent = isHidden ? '▼' : '▶';
        if (isHidden && onOpen) onOpen();
    }
}

export function toggleSampleDataSection() { toggleSection('sampledata-content', 'sampledata-toggle-icon'); }
export function toggleWhereSection() {
    toggleSection('where-content', 'where-toggle-icon', () => {
        populateColumnSuggestions();
        refreshAutocomplete();
    });
}
export function toggleCalculateSection() { toggleSection('calc-content', 'calc-toggle-icon', refreshCalcAliasChips); }
export function toggleOrderBySection() { toggleSection('orderby-content', 'orderby-toggle-icon'); }
export function toggleSelectColumnsSection() {
    toggleSection('selectcols-content', 'selectcols-toggle-icon', () => {
        refreshSelectColumnsList();
    });
}
export function toggleTransposeSection() { toggleSection('transpose-content', 'transpose-toggle-icon'); }
export function toggleGeojsonSection() { toggleSection('geojson-content', 'geojson-toggle-icon'); }
export function toggleFacetSection() { toggleSection('facet-content', 'facet-toggle-icon'); }
export function toggleBinConfigSection() { toggleSection('bin-config-content', 'bin-toggle-icon'); }
export function togglePeriodLimitSection() { toggleSection('periodlimit-content', 'periodlimit-toggle-icon'); }
export function toggleBarOptionsSection() { toggleSection('baroptions-content', 'baroptions-toggle-icon'); }
export function toggleLegendSection() { toggleSection('legend-content', 'legend-toggle-icon'); }

export function setupFacetListeners() {
    const facetEnabled = document.getElementById('facet-enabled');
    const facetOptions = document.getElementById('facet-options');
    if (facetEnabled && facetOptions) {
        facetEnabled.addEventListener('change', () => {
            facetOptions.style.display = facetEnabled.checked ? 'block' : 'none';
        });
    }

    const periodTypeSelect = document.getElementById('facet-period-type');
    if (periodTypeSelect) {
        periodTypeSelect.addEventListener('change', () => {
            _onFacetPeriodTypeChange(periodTypeSelect.value);
        });
        // Initialise on open — e.g. reopening a saved quarter facet
        _onFacetPeriodTypeChange(periodTypeSelect.value);
    }
}

function _onFacetPeriodTypeChange(periodType) {
    const hint = document.getElementById('facet-quarter-hint');
    if (hint) {
        hint.style.display = periodType === 'quarter' ? 'block' : 'none';
    }

    if (periodType === 'quarter') {
        // Quarter facets require both year and quarter filters to be active
        ['year', 'quarter'].forEach(name => {
            const cb = document.getElementById(`filter-${name}`);
            if (cb && !cb.checked) {
                cb.checked = true;
            }
        });
        // Sync reportState.filters from the checkbox state
        reportState.filters = Array.from(
            document.querySelectorAll('input[name="filters"]:checked')
        ).map(el => el.value);
    }
}

// Auto-expand functions
export function autoExpandCalculateIfNeeded() {
    const hasFormula = Array.from(document.querySelectorAll('.calc-formula')).some(
        el => el.value && el.value.trim()
    );
    if (hasFormula) {
        const content = document.getElementById('calc-content');
        const icon = document.getElementById('calc-toggle-icon');
        if (content && icon) {
            content.style.display = 'block';
            icon.textContent = '▼';
        }
    }
}

export function autoExpandOrderByIfNeeded() {
    const field = document.getElementById('orderby-field');
    if (field && field.value && field.value.trim()) {
        const content = document.getElementById('orderby-content');
        const icon = document.getElementById('orderby-toggle-icon');
        if (content && icon) {
            content.style.display = 'block';
            icon.textContent = '▼';
        }
    }
}

export function autoExpandPeriodLimitIfNeeded() {
    const field = document.getElementById('period-limit');
    if (field && field.value && field.value.trim()) {
        const content = document.getElementById('periodlimit-content');
        const icon = document.getElementById('periodlimit-toggle-icon');
        if (content && icon) {
            content.style.display = 'block';
            icon.textContent = '▼';
        }
    }
}

/**
 * Auto-set periodLimit to 6 when a time-based groupBy format is selected and the field is empty
 */
export function maybeSetDefaultPeriodLimit(format) {
    if (!TIME_BASED_FORMATS.has(format)) return;

    const input = document.getElementById('period-limit');
    if (!input || input.value.trim() !== '') return;

    input.value = '6';

    // Auto-expand the period limit section
    const content = document.getElementById('periodlimit-content');
    const icon = document.getElementById('periodlimit-toggle-icon');
    if (content && icon) {
        content.style.display = 'block';
        icon.textContent = '▼';
    }

    // Show temporary hint
    const existing = document.getElementById('periodlimit-auto-hint');
    if (existing) existing.remove();

    const hint = document.createElement('small');
    hint.id = 'periodlimit-auto-hint';
    hint.className = 'hint';
    hint.style.cssText = 'color: var(--color-success); transition: opacity 0.5s;';
    hint.textContent = 'Auto-set to 6 for time-based grouping';
    input.parentElement.appendChild(hint);

    setTimeout(() => {
        hint.style.opacity = '0';
        setTimeout(() => hint.remove(), 500);
    }, 8000);
}

export function autoExpandBarOptionsIfNeeded() {
    const stacked = document.getElementById('bar-stacked');
    const horizontal = document.getElementById('bar-horizontal');
    const showValues = document.getElementById('bar-show-values');
    const showValuesWithAxis = document.getElementById('bar-show-values-with-axis');
    const showAxisTitles = document.getElementById('bar-show-axis-titles');
    if ((stacked && stacked.checked) || (horizontal && horizontal.checked) || (showValues && showValues.checked) || (showValuesWithAxis && showValuesWithAxis.checked) || (showAxisTitles && showAxisTitles.checked)) {
        const content = document.getElementById('baroptions-content');
        const icon = document.getElementById('baroptions-toggle-icon');
        if (content && icon) {
            content.style.display = 'block';
            icon.textContent = '▼';
        }
    }
}

export function autoExpandLegendIfNeeded() {
    const legendShow = document.getElementById('legend-show');
    const legendPosition = document.getElementById('legend-position');
    // Auto-expand if legend is hidden or position is not default
    if ((legendShow && !legendShow.checked) || (legendPosition && legendPosition.value !== 'top')) {
        const content = document.getElementById('legend-content');
        const icon = document.getElementById('legend-toggle-icon');
        if (content && icon) {
            content.style.display = 'block';
            icon.textContent = '▼';
        }
    }
}

export function autoExpandTransposeIfNeeded() {
    const transposeEnabled = document.getElementById('transpose-enabled');
    if (transposeEnabled && transposeEnabled.checked) {
        const content = document.getElementById('transpose-content');
        const icon = document.getElementById('transpose-toggle-icon');
        if (content && icon) {
            content.style.display = 'block';
            icon.textContent = '▼';
        }
        // Also expand the parent select columns section
        const selectColsContent = document.getElementById('selectcols-content');
        const selectColsIcon = document.getElementById('selectcols-toggle-icon');
        if (selectColsContent && selectColsIcon) {
            selectColsContent.style.display = 'block';
            selectColsIcon.textContent = '▼';
        }
    }
}

/**
 * Load existing transpose configuration into the form
 */
export function loadTransposeConfig(transpose) {
    if (!transpose) return;

    const transposeEnabled = document.getElementById('transpose-enabled');
    const transposeOptions = document.getElementById('transpose-options');
    const rowLabel = document.getElementById('transpose-row-label');

    if (transposeEnabled) {
        transposeEnabled.checked = transpose.enabled || false;
    }

    if (transposeOptions) {
        transposeOptions.style.opacity = transpose.enabled ? '1' : '0.5';
    }

    if (rowLabel) {
        rowLabel.value = transpose.rowLabel || '';
    }
}

/**
 * Setup transpose enable checkbox listener
 */
export function setupTransposeListeners() {
    const transposeEnabled = document.getElementById('transpose-enabled');
    const transposeOptions = document.getElementById('transpose-options');

    if (transposeEnabled && transposeOptions) {
        transposeEnabled.addEventListener('change', () => {
            transposeOptions.style.opacity = transposeEnabled.checked ? '1' : '0.5';
        });
    }
}

/**
 * Update transpose options visibility based on checkbox
 */
export function updateTransposeOptions() {
    const transposeEnabled = document.getElementById('transpose-enabled');
    const transposeOptions = document.getElementById('transpose-options');

    if (transposeEnabled && transposeOptions) {
        transposeOptions.style.opacity = transposeEnabled.checked ? '1' : '0.5';
    }
}

export function toggleHeatmapOptions() {
    const enabled = document.getElementById('heatmap-enabled');
    const options = document.getElementById('heatmap-options');
    if (enabled && options) {
        options.style.display = enabled.checked ? 'block' : 'none';
    }
}

export function toggleHeatmapAdvanced() {
    const advanced = document.getElementById('heatmap-advanced');
    const simpleGroup = document.getElementById('heatmap-simple-group');
    const advancedGroup = document.getElementById('heatmap-advanced-group');
    if (advanced && simpleGroup && advancedGroup) {
        simpleGroup.style.display = advanced.checked ? 'none' : 'block';
        advancedGroup.style.display = advanced.checked ? 'block' : 'none';
    }
}

export function toggleHeatmapColumns() {
    const useColumns = document.getElementById('heatmap-use-columns');
    const columnsGroup = document.getElementById('heatmap-columns-group');
    if (useColumns && columnsGroup) {
        columnsGroup.style.display = useColumns.checked ? 'block' : 'none';
    }
}

// Make toggle functions available globally for onclick handlers in HTML
window.toggleHeatmapOptions = toggleHeatmapOptions;
window.toggleHeatmapAdvanced = toggleHeatmapAdvanced;
window.toggleHeatmapColumns = toggleHeatmapColumns;
window.toggleSampleDataSection = toggleSampleDataSection;
window.toggleWhereSection = toggleWhereSection;
window.toggleCalculateSection = toggleCalculateSection;
window.toggleOrderBySection = toggleOrderBySection;
window.toggleSelectColumnsSection = toggleSelectColumnsSection;
window.toggleGeojsonSection = toggleGeojsonSection;
window.toggleFacetSection = toggleFacetSection;
window.toggleBinConfigSection = toggleBinConfigSection;
window.togglePeriodLimitSection = togglePeriodLimitSection;
window.toggleBarOptionsSection = toggleBarOptionsSection;
window.toggleLegendSection = toggleLegendSection;
window.toggleTransposeSection = toggleTransposeSection;
window.updateTransposeOptions = updateTransposeOptions;
