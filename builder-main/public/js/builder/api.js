// API Module
// Handles all API calls to the backend

import { API_BASE, reportState } from './state.js';
import { showError } from './utils.js';
import { authPostJSON } from '../authFetch.js';

// Builds a ?datasource=... (or &datasource=...) fragment for the report's
// selected connection, so schema introspection targets the right database.
function dsParam(sep = '?') {
    return reportState.datasource
        ? `${sep}datasource=${encodeURIComponent(reportState.datasource)}`
        : '';
}

/**
/**
 * Load available database schemas
 */
export async function loadAvailableSchemas() {
    try {
        const schemas = await apiCache.fetch(`${API_BASE}/schema/schemas${dsParam()}`);
        // Handle both array response and error object
        if (Array.isArray(schemas)) {
            reportState.availableSchemas = schemas;
        } else {
            console.warn('Schemas API returned non-array:', schemas);
            reportState.availableSchemas = ['report', 'public'];
        }
        // Default to 'report' if available, otherwise first schema
        if (reportState.availableSchemas.includes('report')) {
            reportState.selectedSchema = 'report';
        } else if (reportState.availableSchemas.length > 0) {
            reportState.selectedSchema = reportState.availableSchemas[0];
        }
    } catch (err) {
        console.error('Failed to load schemas:', err);
        // Fallback to default schemas
        reportState.availableSchemas = ['report', 'public'];
    }
}

/**
 * Load available tables for a schema
 */
export async function loadAvailableTables(schema = '') {
    try {
        const url = schema
            ? `${API_BASE}/schema/tables?schema=${encodeURIComponent(schema)}${dsParam('&')}`
            : `${API_BASE}/schema/tables${dsParam()}`;
        const tables = await apiCache.fetch(url);
        reportState.availableTables = tables.map(t => `${t.schema}.${t.table}`);
        reportState.selectedTable = reportState.availableTables[0] || null;
    } catch (err) {
        console.error('Failed to load tables:', err);
        showError('Failed to load available tables');
    }
}

/**
 * Load columns for a specific table (with caching)
 */
export async function loadTableColumns(table) {
    if (reportState.availableColumns[table]) {
        return reportState.availableColumns[table];
    }

    try {
        const [schema, tableId] = table.split('.');
        const data = await apiCache.fetch(`${API_BASE}/schema/table/${schema}/${tableId}/columns${dsParam()}`);
        reportState.availableColumns[table] = data.columns;
        return data.columns;
    } catch (err) {
        console.error('Failed to load columns:', err);
        return [];
    }
}

/**
 * Load sample data preview for a specific table
 */
export async function loadTablePreview(table) {
    try {
        const [schema, tableId] = table.split('.');
        return await apiCache.fetch(
            `${API_BASE}/schema/table/${schema}/${tableId}/preview?limit=5${dsParam('&')}`
        );
    } catch (err) {
        console.error('Failed to load table preview:', err);
        return null;
    }
}

/**
 * Validate a query structure
 */
export async function validateQuery(request) {
    return authPostJSON(`${API_BASE}/query/validate`, request);
}

/**
 * Preview query data (runs against the report's selected connection)
 */
export async function previewQuery(request) {
    return authPostJSON(`${API_BASE}/query/preview`, { datasource: reportState.datasource || '', ...request });
}

/**
 * Generate YAML from report structure
 */
export async function generateReportYAML(request) {
    return authPostJSON(`${API_BASE}/report/generate`, request);
}

/**
 * Parse YAML into report structure
 */
export async function parseReportYAML(yaml) {
    return authPostJSON(`${API_BASE}/report/parse`, { yaml });
}

/**
 * Preview advanced SQL query (runs against the report's selected connection)
 */
export async function previewAdvancedSQL(request) {
    return authPostJSON(`${API_BASE}/query/preview-sql`, { datasource: reportState.datasource || '', ...request });
}
