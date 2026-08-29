/**
 * Tests for api.js
 *
 * Run with: npm test
 * Watch mode: npm run test:watch
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';

// Use vi.hoisted to create mocks that can be referenced in vi.mock
const { mockReportState, mockShowError } = vi.hoisted(() => ({
    mockReportState: {
        availableSchemas: [],
        selectedSchema: '',
        availableTables: [],
        selectedTable: null,
        availableColumns: {},
    },
    mockShowError: vi.fn(),
}));

// Mock the state module
vi.mock('./state.js', () => ({
    API_BASE: '/api',
    reportState: mockReportState,
}));

// Mock the utils module
vi.mock('./utils.js', () => ({
    showError: mockShowError,
}));

// Mock apiCache global (loaded as script in browser)
global.apiCache = {
    fetch: vi.fn(),
    invalidate: vi.fn(),
    invalidatePattern: vi.fn(),
    clear: vi.fn(),
};

// Import after mocking
import {
    loadAvailableSchemas,
    loadAvailableTables,
    loadTableColumns,
    validateQuery,
    previewQuery,
    generateReportYAML,
    parseReportYAML,
    previewAdvancedSQL,
} from './api.js';

// ============== Test Helpers ==============

/**
 * Create a mock fetch response
 */
function mockFetchResponse(data, ok = true, status = 200) {
    return Promise.resolve({
        ok,
        status,
        json: () => Promise.resolve(data),
    });
}

/**
 * Create a mock fetch that rejects
 */
function mockFetchError(error) {
    return Promise.reject(error);
}

/**
 * Reset state before each test
 */
function resetState() {
    mockReportState.availableSchemas = [];
    mockReportState.selectedSchema = '';
    mockReportState.availableTables = [];
    mockReportState.selectedTable = null;
    mockReportState.availableColumns = {};
}

// ============== loadAvailableSchemas Tests ==============

describe('loadAvailableSchemas', () => {
    beforeEach(() => {
        resetState();
        vi.clearAllMocks();
        global.fetch = vi.fn();
        global.apiCache.fetch = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('fetches from correct endpoint', async () => {
        global.apiCache.fetch.mockResolvedValue(['report', 'public']);

        await loadAvailableSchemas();

        expect(global.apiCache.fetch).toHaveBeenCalledWith('/api/schema/schemas');
    });

    it('updates state with array response', async () => {
        global.apiCache.fetch.mockResolvedValue(['report', 'public', 'analytics']);

        await loadAvailableSchemas();

        expect(mockReportState.availableSchemas).toEqual(['report', 'public', 'analytics']);
    });

    it('selects "report" schema by default if available', async () => {
        global.apiCache.fetch.mockResolvedValue(['public', 'report', 'analytics']);

        await loadAvailableSchemas();

        expect(mockReportState.selectedSchema).toBe('report');
    });

    it('selects first schema if "report" not available', async () => {
        global.apiCache.fetch.mockResolvedValue(['analytics', 'public']);

        await loadAvailableSchemas();

        expect(mockReportState.selectedSchema).toBe('analytics');
    });

    it('falls back to defaults when API returns non-array', async () => {
        global.apiCache.fetch.mockResolvedValue({ error: 'something went wrong' });

        await loadAvailableSchemas();

        expect(mockReportState.availableSchemas).toEqual(['report', 'public']);
    });

    it('falls back to defaults on fetch error', async () => {
        global.apiCache.fetch.mockRejectedValue(new Error('Network error'));

        await loadAvailableSchemas();

        expect(mockReportState.availableSchemas).toEqual(['report', 'public']);
    });

    it('handles empty array response', async () => {
        global.apiCache.fetch.mockResolvedValue([]);

        await loadAvailableSchemas();

        expect(mockReportState.availableSchemas).toEqual([]);
        expect(mockReportState.selectedSchema).toBe(''); // No schema to select
    });
});

// ============== loadAvailableTables Tests ==============

describe('loadAvailableTables', () => {
    beforeEach(() => {
        resetState();
        vi.clearAllMocks();
        global.fetch = vi.fn();
        global.apiCache.fetch = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('fetches all tables when no schema specified', async () => {
        global.apiCache.fetch.mockResolvedValue([
            { schema: 'report', table: 'data' },
        ]);

        await loadAvailableTables();

        expect(global.apiCache.fetch).toHaveBeenCalledWith('/api/schema/tables');
    });

    it('fetches tables for specific schema', async () => {
        global.apiCache.fetch.mockResolvedValue([
            { schema: 'report', table: 'data' },
        ]);

        await loadAvailableTables('report');

        expect(global.apiCache.fetch).toHaveBeenCalledWith('/api/schema/tables?schema=report');
    });

    it('encodes schema parameter', async () => {
        global.apiCache.fetch.mockResolvedValue([]);

        await loadAvailableTables('schema with spaces');

        expect(global.apiCache.fetch).toHaveBeenCalledWith('/api/schema/tables?schema=schema%20with%20spaces');
    });

    it('maps response to schema.table format', async () => {
        global.apiCache.fetch.mockResolvedValue([
            { schema: 'report', table: 'mv_data' },
            { schema: 'public', table: 'users' },
        ]);

        await loadAvailableTables();

        expect(mockReportState.availableTables).toEqual(['report.mv_data', 'public.users']);
    });

    it('selects first table by default', async () => {
        global.apiCache.fetch.mockResolvedValue([
            { schema: 'report', table: 'first_table' },
            { schema: 'report', table: 'second_table' },
        ]);

        await loadAvailableTables();

        expect(mockReportState.selectedTable).toBe('report.first_table');
    });

    it('sets selectedTable to null when no tables', async () => {
        global.apiCache.fetch.mockResolvedValue([]);

        await loadAvailableTables();

        expect(mockReportState.selectedTable).toBeNull();
    });

    it('shows error on fetch failure', async () => {
        global.apiCache.fetch.mockRejectedValue(new Error('Network error'));

        await loadAvailableTables();

        expect(mockShowError).toHaveBeenCalledWith('Failed to load available tables');
    });
});

// ============== loadTableColumns Tests ==============

describe('loadTableColumns', () => {
    beforeEach(() => {
        resetState();
        vi.clearAllMocks();
        global.fetch = vi.fn();
        global.apiCache.fetch = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('fetches columns from correct endpoint', async () => {
        global.apiCache.fetch.mockResolvedValue({
            columns: ['id', 'name', 'value'],
        });

        await loadTableColumns('report.mv_data');

        expect(global.apiCache.fetch).toHaveBeenCalledWith('/api/schema/table/report/mv_data/columns');
    });

    it('returns columns from API', async () => {
        const columns = ['id', 'district', 'cases', 'year'];
        global.apiCache.fetch.mockResolvedValue({ columns });

        const result = await loadTableColumns('report.data');

        expect(result).toEqual(columns);
    });

    it('caches columns in state', async () => {
        const columns = ['col1', 'col2'];
        global.apiCache.fetch.mockResolvedValue({ columns });

        await loadTableColumns('report.test');

        expect(mockReportState.availableColumns['report.test']).toEqual(columns);
    });

    it('returns cached columns without fetching', async () => {
        mockReportState.availableColumns['report.cached'] = ['cached_col'];

        const result = await loadTableColumns('report.cached');

        expect(global.apiCache.fetch).not.toHaveBeenCalled();
        expect(result).toEqual(['cached_col']);
    });

    it('returns empty array on fetch error', async () => {
        global.apiCache.fetch.mockRejectedValue(new Error('Network error'));

        const result = await loadTableColumns('report.broken');

        expect(result).toEqual([]);
    });

    it('handles table names with special characters', async () => {
        global.apiCache.fetch.mockResolvedValue({ columns: [] });

        await loadTableColumns('schema_name.table_name');

        expect(global.apiCache.fetch).toHaveBeenCalledWith('/api/schema/table/schema_name/table_name/columns');
    });
});

// ============== validateQuery Tests ==============

describe('validateQuery', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        global.fetch = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('sends POST request to correct endpoint', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ valid: true }));

        await validateQuery({ table: 'report.data' });

        expect(global.fetch).toHaveBeenCalledWith(
            '/api/query/validate',
            expect.objectContaining({ method: 'POST' })
        );
    });

    it('sends correct headers', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ valid: true }));

        await validateQuery({ table: 'test' });

        expect(global.fetch).toHaveBeenCalledWith(
            expect.any(String),
            expect.objectContaining({
                headers: { 'Content-Type': 'application/json' },
            })
        );
    });

    it('serializes request body as JSON', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ valid: true }));

        const request = {
            table: 'report.data',
            aggregations: [{ column: 'value', function: 'sum', alias: 'total' }],
        };

        await validateQuery(request);

        expect(global.fetch).toHaveBeenCalledWith(
            expect.any(String),
            expect.objectContaining({
                body: JSON.stringify(request),
            })
        );
    });

    it('returns validation response', async () => {
        const validationResult = { valid: true, sql: 'SELECT ...' };
        global.fetch.mockImplementation(() => mockFetchResponse(validationResult));

        const result = await validateQuery({ table: 'test' });

        expect(result).toEqual(validationResult);
    });

    it('returns error response from server', async () => {
        const errorResult = { valid: false, error: 'Invalid column name' };
        global.fetch.mockImplementation(() => mockFetchResponse(errorResult));

        const result = await validateQuery({ table: 'test' });

        expect(result).toEqual(errorResult);
    });
});

// ============== previewQuery Tests ==============

describe('previewQuery', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        global.fetch = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('sends POST request to preview endpoint', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ data: [] }));

        await previewQuery({ table: 'report.data' });

        expect(global.fetch).toHaveBeenCalledWith(
            '/api/query/preview',
            expect.objectContaining({ method: 'POST' })
        );
    });

    it('returns preview data', async () => {
        const previewData = {
            headers: ['district', 'cases'],
            rows: [['Kampala', 100], ['Central', 200]],
        };
        global.fetch.mockImplementation(() => mockFetchResponse(previewData));

        const result = await previewQuery({ table: 'test' });

        expect(result).toEqual(previewData);
    });

    it('sends complex query structure', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ data: [] }));

        const request = {
            table: 'report.mv_data',
            aggregations: [
                { column: 'cases', function: 'sum', alias: 'TotalCases' },
            ],
            groupBy: [{ field: 'district' }],
            filters: { year: 2024 },
        };

        await previewQuery(request);

        expect(global.fetch).toHaveBeenCalledWith(
            expect.any(String),
            expect.objectContaining({
                body: JSON.stringify({ datasource: '', ...request }),
            })
        );
    });
});

// ============== generateReportYAML Tests ==============

describe('generateReportYAML', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        global.fetch = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('sends POST request to generate endpoint', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ yaml: '' }));

        await generateReportYAML({ id: 'test', title: 'Test' });

        expect(global.fetch).toHaveBeenCalledWith(
            '/api/report/generate',
            expect.objectContaining({ method: 'POST' })
        );
    });

    it('returns generated YAML', async () => {
        const yamlResponse = {
            yaml: 'id: test\ntitle: Test Report\n',
            success: true,
        };
        global.fetch.mockImplementation(() => mockFetchResponse(yamlResponse));

        const result = await generateReportYAML({ id: 'test' });

        expect(result).toEqual(yamlResponse);
    });

    it('sends full report structure', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ yaml: '' }));

        const report = {
            id: 'disease/malaria',
            title: 'Malaria Report',
            description: 'Monthly malaria statistics',
            filters: ['district', 'year', 'month'],
            sections: [
                {
                    id: 'overview',
                    layout: 'two-column',
                    components: [],
                },
            ],
        };

        await generateReportYAML(report);

        expect(global.fetch).toHaveBeenCalledWith(
            expect.any(String),
            expect.objectContaining({
                body: JSON.stringify(report),
            })
        );
    });
});

// ============== parseReportYAML Tests ==============

describe('parseReportYAML', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        global.fetch = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('sends POST request to parse endpoint', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ report: {} }));

        await parseReportYAML('id: test');

        expect(global.fetch).toHaveBeenCalledWith(
            '/api/report/parse',
            expect.objectContaining({ method: 'POST' })
        );
    });

    it('wraps yaml in object', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ report: {} }));

        const yaml = 'id: test\ntitle: Test Report';

        await parseReportYAML(yaml);

        expect(global.fetch).toHaveBeenCalledWith(
            expect.any(String),
            expect.objectContaining({
                body: JSON.stringify({ yaml }),
            })
        );
    });

    it('returns parsed report structure', async () => {
        const parsedReport = {
            success: true,
            report: {
                id: 'test',
                title: 'Test Report',
                sections: [],
            },
        };
        global.fetch.mockImplementation(() => mockFetchResponse(parsedReport));

        const result = await parseReportYAML('id: test');

        expect(result).toEqual(parsedReport);
    });

    it('returns parse errors', async () => {
        const errorResponse = {
            success: false,
            error: 'Invalid YAML syntax at line 5',
        };
        global.fetch.mockImplementation(() => mockFetchResponse(errorResponse));

        const result = await parseReportYAML('invalid: yaml: syntax');

        expect(result).toEqual(errorResponse);
    });
});

// ============== previewAdvancedSQL Tests ==============

describe('previewAdvancedSQL', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        global.fetch = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('sends POST request to preview-sql endpoint', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ data: [] }));

        await previewAdvancedSQL({ sql: 'SELECT * FROM data' });

        expect(global.fetch).toHaveBeenCalledWith(
            '/api/query/preview-sql',
            expect.objectContaining({ method: 'POST' })
        );
    });

    it('sends SQL query in request body', async () => {
        global.fetch.mockImplementation(() => mockFetchResponse({ data: [] }));

        const request = {
            sql: 'SELECT district, SUM(cases) FROM report.data GROUP BY district',
            filters: { year: 2024 },
        };

        await previewAdvancedSQL(request);

        expect(global.fetch).toHaveBeenCalledWith(
            expect.any(String),
            expect.objectContaining({
                body: JSON.stringify({ datasource: '', ...request }),
            })
        );
    });

    it('returns preview results', async () => {
        const previewResult = {
            headers: ['district', 'total'],
            rows: [['Kampala', 500]],
        };
        global.fetch.mockImplementation(() => mockFetchResponse(previewResult));

        const result = await previewAdvancedSQL({ sql: 'SELECT ...' });

        expect(result).toEqual(previewResult);
    });
});

// ============== Error Handling Integration ==============

describe('error handling', () => {
    beforeEach(() => {
        resetState();
        vi.clearAllMocks();
        global.fetch = vi.fn();
        global.apiCache.fetch = vi.fn();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('loadAvailableSchemas handles fetch error', async () => {
        global.apiCache.fetch.mockRejectedValue(new Error('Network error'));

        // Should not throw, falls back to defaults
        await expect(loadAvailableSchemas()).resolves.not.toThrow();
        expect(mockReportState.availableSchemas).toEqual(['report', 'public']);
    });

    it('loadAvailableTables handles fetch error', async () => {
        global.apiCache.fetch.mockRejectedValue(new Error('Network error'));

        await loadAvailableTables();

        expect(mockShowError).toHaveBeenCalled();
    });

    it('POST endpoints propagate errors to caller', async () => {
        global.fetch.mockImplementation(() => mockFetchError(new Error('Network error')));

        await expect(validateQuery({})).rejects.toThrow('Network error');
    });
});

// ============== URL Construction ==============

describe('URL construction', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        global.fetch = vi.fn(() => mockFetchResponse([]));
        global.apiCache.fetch = vi.fn().mockResolvedValue([]);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('loadAvailableTables encodes special characters in schema', async () => {
        await loadAvailableTables('test&schema');
        expect(global.apiCache.fetch).toHaveBeenCalledWith('/api/schema/tables?schema=test%26schema');
    });

    it('loadTableColumns splits table name correctly', async () => {
        global.apiCache.fetch.mockResolvedValue({ columns: [] });

        await loadTableColumns('my_schema.my_table');

        expect(global.apiCache.fetch).toHaveBeenCalledWith('/api/schema/table/my_schema/my_table/columns');
    });
});
