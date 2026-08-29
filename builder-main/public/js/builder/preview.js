// Preview Module
// Backend-driven YAML preview

import { API_BASE, elements, reportState } from './state.js';
import { validateQuery, previewQuery, generateReportYAML } from './api.js';
import { showError, showSuccess, showCopyFeedback, escapeHtml } from './utils.js';
import { buildReportObject } from './yaml-handler.js';

/**
 * Update the preview panel by calling backend
 * Called after Save Component or Generate button click
 */
export async function updatePreview() {
    elements.previewContent.textContent = '// Generating YAML...';

    try {
        const request = buildReportObject();
        const result = await generateReportYAML(request);

        if (result.error) {
            const errorMsg = result.details ? `${result.error}: ${result.details}` : result.error;
            elements.previewContent.textContent = '// Validation error: ' + errorMsg;
            showError(errorMsg);
            return;
        }

        if (result.yaml) {
            elements.previewContent.textContent = result.yaml;
            elements.yamlOutput.value = result.yaml; // Keep modal in sync
        }
    } catch (err) {
        elements.previewContent.textContent = '// Error generating YAML: ' + err.message;
        showError('Failed to generate YAML: ' + err.message);
    }
}

/**
 * Handle Generate YAML button click - generates and shows in preview
 */
export async function handleGenerateYAML() {
    await updatePreview();
    showSuccess('YAML generated');
}

/**
 * Preview query data from the component form
 */
export async function previewQueryData() {
    const table = document.getElementById('comp-table')?.value;
    if (!table) {
        showError('Please select a table first');
        return;
    }

    // Collect aggregations
    const aggregations = [];
    const aggRows = document.querySelectorAll('#aggregations-list > div');
    aggRows.forEach((row) => {
        const colInput = row.querySelector('.agg-column');
        const funcSelect = row.querySelector('.agg-function');
        const aliasInput = row.querySelector('.agg-alias');
        const roundInput = row.querySelector('.agg-roundto');

        if (colInput && funcSelect && aliasInput) {
            const col = colInput.value?.trim() || '';
            const func = funcSelect.value || 'sum';
            const alias = aliasInput.value?.trim() || '';
            const roundToRaw = roundInput?.value;

            if (col && alias) {
                const agg = { column: col, function: func, alias: alias };
                if (roundToRaw !== '' && roundToRaw != null && func !== 'count' && func !== 'count_distinct') {
                    const n = parseInt(roundToRaw, 10);
                    if (!isNaN(n) && n >= 0) agg.roundTo = n;
                }
                aggregations.push(agg);
            }
        }
    });

    if (aggregations.length === 0) {
        showError('Please add at least one aggregation');
        return;
    }

    // Collect groupBy from dynamic list
    const groupBy = [];
    document.querySelectorAll('#groupby-list .groupby-row').forEach((row) => {
        const field = row.querySelector('.groupby-field')?.value?.trim();
        const format = row.querySelector('.groupby-format')?.value || undefined;
        const alias = row.querySelector('.groupby-alias')?.value?.trim() || undefined;
        if (field) {
            groupBy.push({ field, format, alias });
        }
    });

    // Collect calculate
    const formula = document.getElementById('calc-formula')?.value?.trim();
    let calculate = null;
    if (formula) {
        calculate = { formula: formula };
        const roundTo = document.getElementById('calc-roundto')?.value;
        if (roundTo !== '' && roundTo != null) calculate.roundTo = parseInt(roundTo);
        const whenZero = document.getElementById('calc-whenzero')?.value?.trim();
        if (whenZero) calculate.whenZero = whenZero;
        const resultAlias = document.getElementById('calc-alias')?.value?.trim();
        if (resultAlias) calculate.resultAlias = resultAlias;
    }

    // Collect WHERE clause
    const whereClause = document.getElementById('comp-where')?.value?.trim() || '';

    // Show loading
    const btn = document.getElementById('btn-preview-query');
    const originalText = btn.textContent;
    btn.textContent = 'Loading...';
    btn.disabled = true;

    try {
        const result = await previewQuery({
            table: table,
            aggregations: aggregations,
            groupBy: groupBy,
            calculate: calculate,
            where: whereClause || undefined,
            limit: 10
        });

        if (result.error) {
            showError('Query error: ' + result.error);
            return;
        }

        showQueryPreviewResults(result);

    } catch (err) {
        showError('Failed to preview data: ' + err.message);
    } finally {
        btn.textContent = originalText;
        btn.disabled = false;
    }
}

/**
 * Show query preview results in modal
 */
export function showQueryPreviewResults(result) {
    const container = elements.previewResultContainer;

    if (!result.headers || result.headers.length === 0) {
        container.innerHTML = '<p style="color: var(--color-text-secondary);">No data returned from query.</p>';
        elements.previewModal.classList.remove('hidden');
        return;
    }

    let html = '<div style="overflow-x: auto;">';
    html += '<table style="width: 100%; border-collapse: collapse; font-size: 12px;">';

    // Headers
    html += '<thead><tr>';
    result.headers.forEach(h => {
        html += `<th style="padding: 8px 12px; border: 1px solid var(--color-border); background: var(--color-hover); text-align: left; font-weight: 600;">${escapeHtml(h)}</th>`;
    });
    html += '</tr></thead>';

    // Rows
    html += '<tbody>';
    if (result.rows && result.rows.length > 0) {
        result.rows.forEach(row => {
            html += '<tr>';
            row.forEach(cell => {
                const cellValue = cell !== null && cell !== undefined ? cell : '';
                html += `<td style="padding: 8px 12px; border: 1px solid var(--color-border);">${escapeHtml(String(cellValue))}</td>`;
            });
            html += '</tr>';
        });
    } else {
        html += `<tr><td colspan="${result.headers.length}" style="padding: 16px; text-align: center; color: var(--color-text-secondary);">No rows returned</td></tr>`;
    }
    html += '</tbody></table></div>';

    // Applied filters info
    if (result.appliedFilters) {
        html += `<p style="margin-top: 10px; font-size: 12px; color: var(--color-primary);"><strong>Filters applied:</strong> ${escapeHtml(result.appliedFilters)}</p>`;
    }

    // Row count and execution time
    const rowCount = result.rows ? result.rows.length : 0;
    let infoText = `Showing ${rowCount} row${rowCount !== 1 ? 's' : ''}`;
    if (result.executionTime) {
        infoText += ` (${result.executionTime})`;
    }
    html += `<p style="margin-top: 5px; font-size: 12px; color: var(--color-text-secondary);">${infoText}</p>`;

    container.innerHTML = html;
    elements.previewModal.classList.remove('hidden');
}

/**
 * Generate SQL preview for all components
 */
export async function generateSQLPreview() {
    let sqlPreview = '';
    let hasQueries = false;

    for (const section of reportState.sections) {
        for (const component of section.components) {
            if (component.query && component.query.table) {
                hasQueries = true;
                try {
                    const result = await validateQuery({
                        table: component.query.table,
                        aggregations: component.query.aggregations || [],
                        groupBy: component.query.groupBy || [],
                        orderBy: component.query.orderBy || [],
                        calculate: component.query.calculate || [],
                        where: component.query.where || undefined
                    });

                    sqlPreview += `-- ${component.title || 'Untitled Component'}\n`;
                    sqlPreview += `-- Type: ${component.type}\n`;
                    sqlPreview += (result.generatedSQL || result.sql || '-- Error generating SQL') + '\n\n';
                } catch (err) {
                    sqlPreview += `-- ${component.title || 'Untitled Component'}\n`;
                    sqlPreview += `-- Error: ${err.message}\n\n`;
                }
            }
        }
    }

    if (!hasQueries) {
        sqlPreview = '-- No queries configured yet.\n-- Add a component with a table and aggregations to see SQL.';
    }

    elements.previewContent.textContent = sqlPreview;
}

/**
 * Copy preview content to clipboard
 */
export function copyPreview() {
    const text = elements.previewContent.textContent;
    navigator.clipboard.writeText(text).then(() => {
        showCopyFeedback('copy-feedback');
    }).catch(err => {
        showError('Failed to copy: ' + err);
    });
}

/**
 * Close preview modal
 */
export function closePreviewModal() {
    elements.previewModal.classList.add('hidden');
}
