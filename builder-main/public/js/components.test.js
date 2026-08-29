/**
 * Tests for components.js
 *
 * Run with: npm test
 * Watch mode: npm run test:watch
 *
 * Note: This file tests the pure utility functions and DOM rendering.
 * Chart.js and Leaflet dependent functions are tested separately via E2E tests.
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { JSDOM } from 'jsdom';

// ============== Setup ==============

// Create a fresh DOM for each test
let dom;
let document;
let window;

beforeEach(() => {
    dom = new JSDOM('<!DOCTYPE html><html><body><div id="app"></div></body></html>', {
        url: 'http://localhost',
    });
    document = dom.window.document;
    window = dom.window;

    // Set up globals
    global.document = document;
    global.window = window;
    global.HTMLElement = window.HTMLElement;

    // Mock window.BASE_PATH
    window.BASE_PATH = '';
});

afterEach(() => {
    dom = null;
    document = null;
    window = null;
});

// ============== Import functions by re-implementing pure ones for testing ==============
// Since components.js uses global scope, we'll test the logic directly

/**
 * Normalize layout value to supported CSS classes
 */
function normalizeLayout(layout) {
    const layoutMap = {
        'one-column': 'single',
        'single-column': 'single',
        'full': 'single',
        'single': 'single',
        'two-column': 'two-column',
        'three-column': 'three-column',
        'four-column': 'four-column'
    };
    return layoutMap[layout] || 'single';
}

/**
 * Format a number with thousand separators. Mirror of the implementation in
 * components.js — opts.decimals overrides the default cap of 1.
 */
function formatNumber(value, opts) {
    if (value === null || value === undefined || value === '') {
        return '';
    }
    const trimmed = String(value).trim();
    const num = Number(trimmed);
    if (trimmed === '' || isNaN(num)) {
        return String(value);
    }
    const decimals = opts && Number.isInteger(opts.decimals) && opts.decimals >= 0
        ? opts.decimals
        : 1;
    return num.toLocaleString('en-US', { maximumFractionDigits: decimals, minimumFractionDigits: 0 });
}

/**
 * Format numbers in HTML content. Safety cap of 1 fractional digit — substituted
 * DB values are pre-formatted server-side using the author's roundTo, so this
 * only affects literal numbers typed into the content template.
 */
function formatNumbersInHtml(html) {
    return html.replace(/(?<![\d.])(\d+(?:\.\d+)?)\b(?![^<]*>)/g, (match) => {
        const num = Number(match);
        if (isNaN(num)) return match;
        const hasFraction = match.includes('.');
        if (!hasFraction && num < 1000) return match;
        return num.toLocaleString('en-US', { maximumFractionDigits: 1 });
    });
}

/**
 * Look up decimals hint for a column from component.data.columnFormats.
 */
function getColumnDecimals(componentData, columnName) {
    if (!componentData || !componentData.columnFormats || !columnName) return undefined;
    const fmt = componentData.columnFormats[columnName];
    if (!fmt || !Number.isInteger(fmt.decimals) || fmt.decimals < 0) return undefined;
    return fmt.decimals;
}

/**
 * Get color based on value (for markers)
 */
function getMarkerColor(value) {
    if (value >= 85) return '#27ae60';
    if (value >= 70) return '#f39c12';
    if (value >= 50) return '#e67e22';
    return '#e74c3c';
}

/**
 * Calculate quantile bins from district values
 */
function calculateQuantileBins(districtValues, numBins = 3) {
    const values = Object.values(districtValues)
        .filter(v => v !== null && v !== undefined)
        .map(v => parseFloat(v))
        .sort((a, b) => a - b);

    if (values.length === 0) return [0, 0, 0];

    const q1Index = Math.floor(values.length / 3);
    const q2Index = Math.floor(2 * values.length / 3);

    return [values[q1Index], values[q2Index], values[values.length - 1]];
}

// ============== normalizeLayout Tests ==============

describe('normalizeLayout', () => {
    describe('valid layouts', () => {
        it('returns single for "single"', () => {
            expect(normalizeLayout('single')).toBe('single');
        });

        it('returns two-column for "two-column"', () => {
            expect(normalizeLayout('two-column')).toBe('two-column');
        });

        it('returns three-column for "three-column"', () => {
            expect(normalizeLayout('three-column')).toBe('three-column');
        });

        it('returns four-column for "four-column"', () => {
            expect(normalizeLayout('four-column')).toBe('four-column');
        });
    });

    describe('legacy layouts', () => {
        it('normalizes "one-column" to "single"', () => {
            expect(normalizeLayout('one-column')).toBe('single');
        });

        it('normalizes "single-column" to "single"', () => {
            expect(normalizeLayout('single-column')).toBe('single');
        });

        it('normalizes "full" to "single"', () => {
            expect(normalizeLayout('full')).toBe('single');
        });
    });

    describe('unknown layouts', () => {
        it('defaults to "single" for unknown layout', () => {
            expect(normalizeLayout('unknown')).toBe('single');
        });

        it('defaults to "single" for empty string', () => {
            expect(normalizeLayout('')).toBe('single');
        });

        it('defaults to "single" for null', () => {
            expect(normalizeLayout(null)).toBe('single');
        });

        it('defaults to "single" for undefined', () => {
            expect(normalizeLayout(undefined)).toBe('single');
        });

        it('defaults to "single" for typos', () => {
            expect(normalizeLayout('two-columns')).toBe('single'); // Missing 's'
            expect(normalizeLayout('2-column')).toBe('single');
        });
    });
});

// ============== formatNumber Tests ==============

describe('formatNumber', () => {
    describe('valid numbers', () => {
        it('formats small numbers without separators', () => {
            expect(formatNumber(100)).toBe('100');
            expect(formatNumber(999)).toBe('999');
        });

        it('formats thousands with comma separators', () => {
            expect(formatNumber(1000)).toBe('1,000');
            expect(formatNumber(10000)).toBe('10,000');
            expect(formatNumber(100000)).toBe('100,000');
        });

        it('formats millions', () => {
            expect(formatNumber(1000000)).toBe('1,000,000');
            expect(formatNumber(1234567)).toBe('1,234,567');
        });

        it('formats decimal numbers with one decimal place max', () => {
            expect(formatNumber(1234.5)).toBe('1,234.5');
            expect(formatNumber(1234.56)).toBe('1,234.6'); // Rounds
            expect(formatNumber(1234.04)).toBe('1,234'); // Drops trailing zero
        });

        it('formats string numbers', () => {
            expect(formatNumber('1000')).toBe('1,000');
            expect(formatNumber('1234.5')).toBe('1,234.5');
        });

        it('handles zero', () => {
            expect(formatNumber(0)).toBe('0');
        });

        it('handles negative numbers', () => {
            expect(formatNumber(-1000)).toBe('-1,000');
            expect(formatNumber(-1234.5)).toBe('-1,234.5');
        });
    });

    describe('edge cases', () => {
        it('returns empty string for null', () => {
            expect(formatNumber(null)).toBe('');
        });

        it('returns empty string for undefined', () => {
            expect(formatNumber(undefined)).toBe('');
        });

        it('returns empty string for empty string', () => {
            expect(formatNumber('')).toBe('');
        });

        it('returns original string for non-numeric strings', () => {
            expect(formatNumber('abc')).toBe('abc');
            expect(formatNumber('N/A')).toBe('N/A');
        });

        it('handles very large numbers', () => {
            expect(formatNumber(1234567890)).toBe('1,234,567,890');
        });

        it('handles very small decimals', () => {
            expect(formatNumber(0.1)).toBe('0.1');
            expect(formatNumber(0.01)).toBe('0');
        });
    });

    describe('decimals option', () => {
        it('honors opts.decimals=0', () => {
            expect(formatNumber(83.7, { decimals: 0 })).toBe('84');
            expect(formatNumber(83.2, { decimals: 0 })).toBe('83');
        });

        it('honors opts.decimals=2 and does not pad trailing zeros', () => {
            expect(formatNumber(83.25, { decimals: 2 })).toBe('83.25');
            expect(formatNumber(83.2, { decimals: 2 })).toBe('83.2');
            expect(formatNumber(83, { decimals: 2 })).toBe('83');
        });

        it('honors opts.decimals=3', () => {
            expect(formatNumber(1.2345, { decimals: 3 })).toBe('1.235');
            expect(formatNumber(1234.5678, { decimals: 3 })).toBe('1,234.568');
        });

        it('ignores invalid decimals (falls back to default 1)', () => {
            expect(formatNumber(83.25, { decimals: -1 })).toBe('83.3');
            expect(formatNumber(83.25, { decimals: 1.5 })).toBe('83.3');
            expect(formatNumber(83.25, {})).toBe('83.3');
        });
    });
});

// ============== getColumnDecimals Tests ==============

describe('getColumnDecimals', () => {
    it('returns undefined for missing data', () => {
        expect(getColumnDecimals(null, 'X')).toBeUndefined();
        expect(getColumnDecimals({}, 'X')).toBeUndefined();
        expect(getColumnDecimals({ columnFormats: {} }, 'X')).toBeUndefined();
    });

    it('returns the column decimals when defined', () => {
        const data = { columnFormats: { 'ANC4 Coverage': { decimals: 1 }, 'Cases': { decimals: 0 } } };
        expect(getColumnDecimals(data, 'ANC4 Coverage')).toBe(1);
        expect(getColumnDecimals(data, 'Cases')).toBe(0);
    });

    it('returns undefined for unknown columns', () => {
        const data = { columnFormats: { 'X': { decimals: 2 } } };
        expect(getColumnDecimals(data, 'Y')).toBeUndefined();
    });

    it('rejects non-integer or negative decimals', () => {
        const data = { columnFormats: {
            'A': { decimals: 'two' },
            'B': { decimals: -1 },
            'C': { decimals: 1.5 },
        } };
        expect(getColumnDecimals(data, 'A')).toBeUndefined();
        expect(getColumnDecimals(data, 'B')).toBeUndefined();
        expect(getColumnDecimals(data, 'C')).toBeUndefined();
    });
});

// ============== formatNumbersInHtml Tests ==============

describe('formatNumbersInHtml', () => {
    describe('basic formatting', () => {
        it('formats standalone numbers >= 1000', () => {
            expect(formatNumbersInHtml('Total: 1000')).toBe('Total: 1,000');
            expect(formatNumbersInHtml('Cases: 12345')).toBe('Cases: 12,345');
        });

        it('does not format numbers < 1000', () => {
            expect(formatNumbersInHtml('Count: 999')).toBe('Count: 999');
            expect(formatNumbersInHtml('Value: 100')).toBe('Value: 100');
        });

        it('formats multiple numbers', () => {
            expect(formatNumbersInHtml('A: 1000 B: 2000'))
                .toBe('A: 1,000 B: 2,000');
        });
    });

    describe('HTML preservation', () => {
        it('does not format numbers inside HTML tags', () => {
            // The regex (?![^<]*>) prevents matching inside tags
            const html = '<div data-value="1234">Text</div>';
            expect(formatNumbersInHtml(html)).toBe('<div data-value="1234">Text</div>');
        });

        it('formats numbers in text content', () => {
            const html = '<div>Total cases: 5000</div>';
            expect(formatNumbersInHtml(html)).toBe('<div>Total cases: 5,000</div>');
        });

        it('preserves HTML structure', () => {
            const html = '<span class="value">10000</span>';
            expect(formatNumbersInHtml(html)).toBe('<span class="value">10,000</span>');
        });
    });

    describe('edge cases', () => {
        it('handles empty string', () => {
            expect(formatNumbersInHtml('')).toBe('');
        });

        it('handles string with no numbers', () => {
            expect(formatNumbersInHtml('No numbers here')).toBe('No numbers here');
        });

        it('handles mixed content', () => {
            const html = 'Found 15000 cases in 50 districts';
            expect(formatNumbersInHtml(html)).toBe('Found 15,000 cases in 50 districts');
        });

        it('does not comma-separate fractional digits of floats', () => {
            expect(formatNumbersInHtml('Rate: 99.5 cases'))
                .toBe('Rate: 99.5 cases');
        });

        it('caps fractional digits at 1 as a safety net', () => {
            // DB-substituted values are pre-formatted server-side now — this cap
            // only affects literal numbers in author-supplied content templates.
            expect(formatNumbersInHtml('63.6000000000000000'))
                .toBe('63.6');
            expect(formatNumbersInHtml('KPI: 83.2000000000000000%'))
                .toBe('KPI: 83.2%');
            expect(formatNumbersInHtml('Ratio: 1.2345'))
                .toBe('Ratio: 1.2');
        });

        it('still formats whole-number part of large floats', () => {
            expect(formatNumbersInHtml('12345.67')).toBe('12,345.7');
        });
    });
});

// ============== getMarkerColor Tests ==============

describe('getMarkerColor', () => {
    describe('color thresholds', () => {
        it('returns green (#27ae60) for values >= 85', () => {
            expect(getMarkerColor(85)).toBe('#27ae60');
            expect(getMarkerColor(90)).toBe('#27ae60');
            expect(getMarkerColor(100)).toBe('#27ae60');
        });

        it('returns orange (#f39c12) for values 70-84', () => {
            expect(getMarkerColor(70)).toBe('#f39c12');
            expect(getMarkerColor(75)).toBe('#f39c12');
            expect(getMarkerColor(84)).toBe('#f39c12');
        });

        it('returns dark orange (#e67e22) for values 50-69', () => {
            expect(getMarkerColor(50)).toBe('#e67e22');
            expect(getMarkerColor(60)).toBe('#e67e22');
            expect(getMarkerColor(69)).toBe('#e67e22');
        });

        it('returns red (#e74c3c) for values < 50', () => {
            expect(getMarkerColor(49)).toBe('#e74c3c');
            expect(getMarkerColor(25)).toBe('#e74c3c');
            expect(getMarkerColor(0)).toBe('#e74c3c');
        });
    });

    describe('boundary values', () => {
        it('handles exact boundary at 85', () => {
            expect(getMarkerColor(84.9)).toBe('#f39c12');
            expect(getMarkerColor(85)).toBe('#27ae60');
        });

        it('handles exact boundary at 70', () => {
            expect(getMarkerColor(69.9)).toBe('#e67e22');
            expect(getMarkerColor(70)).toBe('#f39c12');
        });

        it('handles exact boundary at 50', () => {
            expect(getMarkerColor(49.9)).toBe('#e74c3c');
            expect(getMarkerColor(50)).toBe('#e67e22');
        });
    });

    describe('edge cases', () => {
        it('handles negative values', () => {
            expect(getMarkerColor(-10)).toBe('#e74c3c');
        });

        it('handles very large values', () => {
            expect(getMarkerColor(1000)).toBe('#27ae60');
        });
    });
});

// ============== calculateQuantileBins Tests ==============

describe('calculateQuantileBins', () => {
    describe('normal data', () => {
        it('calculates bins for evenly distributed data', () => {
            const values = {
                'A': 10,
                'B': 20,
                'C': 30,
                'D': 40,
                'E': 50,
                'F': 60,
            };
            const bins = calculateQuantileBins(values);

            // With 6 values: q1Index = 2, q2Index = 4
            // Sorted: [10, 20, 30, 40, 50, 60]
            // bins = [30, 50, 60]
            expect(bins).toEqual([30, 50, 60]);
        });

        it('calculates bins for 3 values', () => {
            const values = { 'A': 10, 'B': 50, 'C': 100 };
            const bins = calculateQuantileBins(values);

            // q1Index = 1, q2Index = 2
            // bins = [50, 100, 100]
            expect(bins).toEqual([50, 100, 100]);
        });

        it('calculates bins for large dataset', () => {
            const values = {};
            for (let i = 1; i <= 100; i++) {
                values[`District${i}`] = i;
            }
            const bins = calculateQuantileBins(values);

            // q1Index = 33, q2Index = 66
            // bins = [34, 67, 100]
            expect(bins[0]).toBe(34);
            expect(bins[1]).toBe(67);
            expect(bins[2]).toBe(100);
        });
    });

    describe('edge cases', () => {
        it('returns [0, 0, 0] for empty object', () => {
            expect(calculateQuantileBins({})).toEqual([0, 0, 0]);
        });

        it('filters out null values', () => {
            const values = {
                'A': 10,
                'B': null,
                'C': 30,
                'D': null,
                'E': 50,
            };
            const bins = calculateQuantileBins(values);

            // Only [10, 30, 50] after filtering
            expect(bins.length).toBe(3);
        });

        it('filters out undefined values', () => {
            const values = {
                'A': 10,
                'B': undefined,
                'C': 50,
            };
            const bins = calculateQuantileBins(values);
            expect(bins.length).toBe(3);
        });

        it('handles single value', () => {
            const values = { 'A': 100 };
            const bins = calculateQuantileBins(values);

            // q1Index = 0, q2Index = 0
            expect(bins).toEqual([100, 100, 100]);
        });

        it('handles string numbers', () => {
            const values = { 'A': '10', 'B': '50', 'C': '100' };
            const bins = calculateQuantileBins(values);

            expect(bins[0]).toBe(50);
            expect(bins[2]).toBe(100);
        });

        it('handles duplicate values', () => {
            const values = {
                'A': 50,
                'B': 50,
                'C': 50,
                'D': 50,
            };
            const bins = calculateQuantileBins(values);

            expect(bins).toEqual([50, 50, 50]);
        });
    });
});

// ============== DOM Rendering Tests ==============

describe('DOM rendering helpers', () => {
    /**
     * Render a text component
     */
    function renderText(element, component) {
        const div = document.createElement('div');
        div.className = 'component component-text';
        div.innerHTML = formatNumbersInHtml(component.content);
        element.appendChild(div);
    }

    describe('renderText', () => {
        it('creates a div with correct class', () => {
            const container = document.createElement('div');
            renderText(container, { content: 'Hello World' });

            const rendered = container.querySelector('.component-text');
            expect(rendered).not.toBeNull();
            expect(rendered.className).toBe('component component-text');
        });

        it('renders content as innerHTML', () => {
            const container = document.createElement('div');
            renderText(container, { content: '<strong>Bold</strong> text' });

            const rendered = container.querySelector('.component-text');
            expect(rendered.innerHTML).toBe('<strong>Bold</strong> text');
        });

        it('formats numbers in content', () => {
            const container = document.createElement('div');
            renderText(container, { content: 'Total: 10000 cases' });

            const rendered = container.querySelector('.component-text');
            expect(rendered.innerHTML).toBe('Total: 10,000 cases');
        });

        it('handles empty content', () => {
            const container = document.createElement('div');
            renderText(container, { content: '' });

            const rendered = container.querySelector('.component-text');
            expect(rendered.innerHTML).toBe('');
        });
    });

    describe('section layout classes', () => {
        function createSectionDiv(layout) {
            const componentsDiv = document.createElement('div');
            const normalizedLayout = normalizeLayout(layout);
            componentsDiv.className = `section-components layout-${normalizedLayout}`;
            return componentsDiv;
        }

        it('creates layout-single class', () => {
            const div = createSectionDiv('single');
            expect(div.className).toBe('section-components layout-single');
        });

        it('creates layout-two-column class', () => {
            const div = createSectionDiv('two-column');
            expect(div.className).toBe('section-components layout-two-column');
        });

        it('normalizes legacy layout to layout-single', () => {
            const div = createSectionDiv('one-column');
            expect(div.className).toBe('section-components layout-single');
        });

        it('defaults unknown layout to layout-single', () => {
            const div = createSectionDiv('invalid');
            expect(div.className).toBe('section-components layout-single');
        });
    });
});

// ============== Table Sorting Logic Tests ==============

describe('table sorting logic', () => {
    /**
     * Detect if a column contains mostly numbers
     */
    function isNumericColumn(rows, colIndex) {
        const sampleSize = Math.min(10, rows.length);
        let numericCount = 0;
        for (let i = 0; i < sampleSize; i++) {
            const val = rows[i][colIndex];
            if (!isNaN(parseFloat(val)) && String(val).trim() !== '') {
                numericCount++;
            }
        }
        return numericCount / sampleSize > 0.7;
    }

    /**
     * Sort rows by column
     */
    function sortRows(rows, colIndex, direction, isNumeric) {
        if (colIndex === null) return rows;

        const sorted = [...rows].sort((a, b) => {
            const aVal = a[colIndex];
            const bVal = b[colIndex];

            if (isNumeric) {
                const aNum = parseFloat(aVal) || 0;
                const bNum = parseFloat(bVal) || 0;
                return direction === 'asc' ? aNum - bNum : bNum - aNum;
            } else {
                const aStr = aVal.toString().toLowerCase();
                const bStr = bVal.toString().toLowerCase();
                return direction === 'asc'
                    ? aStr.localeCompare(bStr)
                    : bStr.localeCompare(aStr);
            }
        });
        return sorted;
    }

    describe('isNumericColumn', () => {
        it('detects numeric columns', () => {
            const rows = [
                ['Kampala', 100],
                ['Central', 200],
                ['Jinja', 300],
            ];
            expect(isNumericColumn(rows, 1)).toBe(true);
        });

        it('detects string columns', () => {
            const rows = [
                ['Kampala', 100],
                ['Central', 200],
                ['Jinja', 300],
            ];
            expect(isNumericColumn(rows, 0)).toBe(false);
        });

        it('handles mixed columns (70% threshold)', () => {
            // 2 out of 3 = 66.67% numeric, which is < 70%, so should be false
            const rowsBelow = [
                ['Kampala', '100'],
                ['Central', '200'],
                ['Jinja', 'N/A'],
            ];
            expect(isNumericColumn(rowsBelow, 1)).toBe(false);

            // 3 out of 4 = 75% numeric, which is > 70%, so should be true
            const rowsAbove = [
                ['Kampala', '100'],
                ['Central', '200'],
                ['Jinja', '300'],
                ['Mbarara', 'N/A'],
            ];
            expect(isNumericColumn(rowsAbove, 1)).toBe(true);
        });

        it('handles empty rows', () => {
            expect(isNumericColumn([], 0)).toBe(false);
        });
    });

    describe('sortRows', () => {
        it('sorts numeric column ascending', () => {
            const rows = [
                ['B', 30],
                ['A', 10],
                ['C', 20],
            ];
            const sorted = sortRows(rows, 1, 'asc', true);

            expect(sorted[0][1]).toBe(10);
            expect(sorted[1][1]).toBe(20);
            expect(sorted[2][1]).toBe(30);
        });

        it('sorts numeric column descending', () => {
            const rows = [
                ['B', 30],
                ['A', 10],
                ['C', 20],
            ];
            const sorted = sortRows(rows, 1, 'desc', true);

            expect(sorted[0][1]).toBe(30);
            expect(sorted[1][1]).toBe(20);
            expect(sorted[2][1]).toBe(10);
        });

        it('sorts string column ascending', () => {
            const rows = [
                ['Banana', 1],
                ['Apple', 2],
                ['Cherry', 3],
            ];
            const sorted = sortRows(rows, 0, 'asc', false);

            expect(sorted[0][0]).toBe('Apple');
            expect(sorted[1][0]).toBe('Banana');
            expect(sorted[2][0]).toBe('Cherry');
        });

        it('sorts string column descending', () => {
            const rows = [
                ['Banana', 1],
                ['Apple', 2],
                ['Cherry', 3],
            ];
            const sorted = sortRows(rows, 0, 'desc', false);

            expect(sorted[0][0]).toBe('Cherry');
            expect(sorted[1][0]).toBe('Banana');
            expect(sorted[2][0]).toBe('Apple');
        });

        it('returns original rows when colIndex is null', () => {
            const rows = [['A', 1], ['B', 2]];
            const sorted = sortRows(rows, null, 'asc', false);

            expect(sorted).toEqual(rows);
        });

        it('is case insensitive for strings', () => {
            const rows = [
                ['banana', 1],
                ['Apple', 2],
                ['CHERRY', 3],
            ];
            const sorted = sortRows(rows, 0, 'asc', false);

            expect(sorted[0][0]).toBe('Apple');
            expect(sorted[1][0]).toBe('banana');
            expect(sorted[2][0]).toBe('CHERRY');
        });

        it('handles missing/NaN numeric values', () => {
            const rows = [
                ['A', 'N/A'],
                ['B', 100],
                ['C', ''],
            ];
            const sorted = sortRows(rows, 1, 'asc', true);

            // N/A and '' become 0
            expect(sorted[0][1]).toBe('N/A'); // 0
            expect(sorted[1][1]).toBe(''); // 0
            expect(sorted[2][1]).toBe(100);
        });

        it('does not mutate original array', () => {
            const rows = [['B', 2], ['A', 1]];
            const original = [...rows];
            sortRows(rows, 0, 'asc', false);

            expect(rows).toEqual(original);
        });
    });
});

// ============== Table Filtering Logic Tests ==============

describe('table filtering logic', () => {
    function filterRows(rows, term) {
        if (!term.trim()) {
            return [...rows];
        }
        const lowerTerm = term.toLowerCase();
        return rows.filter(row =>
            row.some(cell => String(cell).toLowerCase().includes(lowerTerm))
        );
    }

    it('returns all rows for empty search term', () => {
        const rows = [['A', 1], ['B', 2]];
        expect(filterRows(rows, '')).toEqual(rows);
        expect(filterRows(rows, '   ')).toEqual(rows);
    });

    it('filters rows by matching text', () => {
        const rows = [
            ['Kampala', 100],
            ['Central', 200],
            ['Jinja', 300],
        ];
        const filtered = filterRows(rows, 'kampala');

        expect(filtered.length).toBe(1);
        expect(filtered[0][0]).toBe('Kampala');
    });

    it('is case insensitive', () => {
        const rows = [
            ['KAMPALA', 100],
            ['Central', 200],
        ];
        const filtered = filterRows(rows, 'kampala');

        expect(filtered.length).toBe(1);
    });

    it('filters by numeric values', () => {
        const rows = [
            ['A', 100],
            ['B', 200],
            ['C', 1000],
        ];
        const filtered = filterRows(rows, '100');

        expect(filtered.length).toBe(2); // '100' and '1000'
    });

    it('matches partial strings', () => {
        const rows = [
            ['Kampala District', 100],
            ['Central District', 200],
            ['Jinja City', 300],
        ];
        const filtered = filterRows(rows, 'district');

        expect(filtered.length).toBe(2);
    });

    it('returns empty array when no matches', () => {
        const rows = [['A', 1], ['B', 2]];
        const filtered = filterRows(rows, 'xyz');

        expect(filtered).toEqual([]);
    });

    it('does not mutate original array', () => {
        const rows = [['A', 1], ['B', 2]];
        const original = [...rows];
        filterRows(rows, 'A');

        expect(rows).toEqual(original);
    });
});

// ============== Pagination Logic Tests ==============

describe('pagination logic', () => {
    function resolveTablePageSize(pageSize) {
        const parsed = Math.floor(Number(pageSize));
        return Number.isFinite(parsed) && parsed > 0 ? parsed : 20;
    }

    function getPageRows(rows, currentPage, pageSize = 20) {
        const PAGE_SIZE = resolveTablePageSize(pageSize);
        const startIndex = (currentPage - 1) * PAGE_SIZE;
        const endIndex = startIndex + PAGE_SIZE;
        return rows.slice(startIndex, endIndex);
    }

    function getTotalPages(rows, pageSize = 20) {
        const PAGE_SIZE = resolveTablePageSize(pageSize);
        return Math.ceil(rows.length / PAGE_SIZE);
    }

    describe('resolveTablePageSize', () => {
        it('defaults to 20 when pageSize is missing', () => {
            expect(resolveTablePageSize(undefined)).toBe(20);
        });

        it('uses a positive custom pageSize', () => {
            expect(resolveTablePageSize(5)).toBe(5);
            expect(resolveTablePageSize('10')).toBe(10);
        });

        it('falls back to 20 for invalid pageSize values', () => {
            expect(resolveTablePageSize(0)).toBe(20);
            expect(resolveTablePageSize(-5)).toBe(20);
            expect(resolveTablePageSize('abc')).toBe(20);
        });
    });

    describe('getTotalPages', () => {
        it('calculates pages for exact multiples', () => {
            const rows = new Array(40).fill(['A', 1]);
            expect(getTotalPages(rows)).toBe(2);
        });

        it('rounds up for partial pages', () => {
            const rows = new Array(25).fill(['A', 1]);
            expect(getTotalPages(rows)).toBe(2);
        });

        it('returns 1 for less than PAGE_SIZE rows', () => {
            const rows = new Array(10).fill(['A', 1]);
            expect(getTotalPages(rows)).toBe(1);
        });

        it('returns 0 for empty array', () => {
            expect(getTotalPages([])).toBe(0);
        });

        it('uses custom pageSize when provided', () => {
            const rows = new Array(11).fill(['A', 1]);
            expect(getTotalPages(rows, 5)).toBe(3);
        });
    });

    describe('getPageRows', () => {
        it('returns first page correctly', () => {
            const rows = new Array(50).fill(0).map((_, i) => [i]);
            const page = getPageRows(rows, 1);

            expect(page.length).toBe(20);
            expect(page[0][0]).toBe(0);
            expect(page[19][0]).toBe(19);
        });

        it('returns second page correctly', () => {
            const rows = new Array(50).fill(0).map((_, i) => [i]);
            const page = getPageRows(rows, 2);

            expect(page.length).toBe(20);
            expect(page[0][0]).toBe(20);
            expect(page[19][0]).toBe(39);
        });

        it('returns partial last page', () => {
            const rows = new Array(25).fill(0).map((_, i) => [i]);
            const page = getPageRows(rows, 2);

            expect(page.length).toBe(5);
            expect(page[0][0]).toBe(20);
            expect(page[4][0]).toBe(24);
        });

        it('returns empty array for page beyond data', () => {
            const rows = new Array(10).fill(['A']);
            const page = getPageRows(rows, 5);

            expect(page).toEqual([]);
        });

        it('returns custom-sized pages', () => {
            const rows = new Array(12).fill(0).map((_, i) => [i]);
            const page = getPageRows(rows, 2, 5);

            expect(page.length).toBe(5);
            expect(page[0][0]).toBe(5);
            expect(page[4][0]).toBe(9);
        });
    });
});

// ============== Pie Label Bounds Tests ==============

describe('pie outer label bounds', () => {
    function getPieOuterLabelBounds(chart, labelBlockHeight, overflow = 26) {
        const halfLabelHeight = labelBlockHeight / 2;
        const edgePadding = 4;
        const chartHeight = Number(chart.height) || 0;
        const chartArea = chart.chartArea || { top: 0, bottom: chartHeight };
        const legend = chart.legend;
        const legendOptions = legend?.options || {};
        const legendDisplayed = !!legend && legendOptions.display !== false;
        const legendPosition = legendOptions.position || legend.position || '';

        const topOverflow = legendDisplayed && legendPosition === 'top' ? 0 : overflow;
        const bottomOverflow = legendDisplayed && legendPosition === 'bottom' ? 0 : overflow;

        const minY = halfLabelHeight + edgePadding;
        const maxY = chartHeight - halfLabelHeight - edgePadding;
        const labelTop = Math.max(minY, chartArea.top - topOverflow);
        const labelBottom = Math.min(maxY, chartArea.bottom + bottomOverflow);

        return {
            labelTop,
            labelBottom: Math.max(labelTop, labelBottom),
        };
    }

    it('keeps labels above a bottom legend', () => {
        const bounds = getPieOuterLabelBounds({
            height: 400,
            chartArea: { top: 40, bottom: 300 },
            legend: { options: { display: true, position: 'bottom' } },
        }, 22);

        expect(bounds.labelBottom).toBe(300);
    });

    it('still allows extra lower room when the legend is not below the pie', () => {
        const bounds = getPieOuterLabelBounds({
            height: 400,
            chartArea: { top: 40, bottom: 300 },
            legend: { options: { display: true, position: 'top' } },
        }, 22);

        expect(bounds.labelBottom).toBe(326);
    });
});

// ============== Heatmap Helper Tests ==============

// Re-implement the pure functions for testing (same as components.js)
function heatmapColor(ratio, scale) {
    if (scale === 'red-green') {
        return `hsl(${ratio * 120}, 55%, 88%)`;
    }
    if (scale === 'blue') {
        return `rgba(49, 130, 189, ${(0.06 + ratio * 0.30).toFixed(2)})`;
    }
    return `hsl(${120 - ratio * 120}, 55%, 88%)`;
}

function heatmapRuleColor(value, rule) {
    const [lo, hi] = rule.thresholds;
    const red = 'hsl(0, 55%, 88%)';
    const yellow = 'hsl(48, 55%, 88%)';
    const green = 'hsl(120, 55%, 88%)';

    if (rule.colors === 'green-yellow-red') {
        if (value < lo) return green;
        if (value < hi) return yellow;
        return red;
    }
    // default: red-yellow-green
    if (value < lo) return red;
    if (value < hi) return yellow;
    return green;
}

function buildHeatmapRuleIndex(rules, headers) {
    const index = {};
    const lowerHeaders = headers.map(h => h.toLowerCase());
    for (const rule of rules) {
        if (!rule.column || !Array.isArray(rule.thresholds) || rule.thresholds.length !== 2) continue;
        const colIndex = lowerHeaders.indexOf(rule.column.toLowerCase());
        if (colIndex === -1) continue;
        index[colIndex] = rule;
    }
    return index;
}

function parseHeatmapConfig(heatmap) {
    if (!heatmap) return null;
    if (Array.isArray(heatmap.rules) && heatmap.rules.length > 0) return { rules: heatmap.rules };
    if (heatmap.scale) return { scale: heatmap.scale };
    return null;
}

function computeHeatmapStats(rows, isNumericFn, colCount) {
    const stats = {};
    for (let c = 0; c < colCount; c++) {
        if (!isNumericFn(c)) continue;
        let min = Infinity, max = -Infinity;
        for (let r = 0; r < rows.length; r++) {
            const val = parseFloat(rows[r][c]);
            if (isNaN(val)) continue;
            if (val < min) min = val;
            if (val > max) max = val;
        }
        if (min < max) {
            stats[c] = { min, max };
        }
    }
    return stats;
}

describe('heatmapColor', () => {
    describe('green-red scale', () => {
        it('returns green at ratio 0', () => {
            const color = heatmapColor(0, 'green-red');
            expect(color).toBe('hsl(120, 55%, 88%)');
        });

        it('returns yellow at ratio 0.5', () => {
            const color = heatmapColor(0.5, 'green-red');
            expect(color).toBe('hsl(60, 55%, 88%)');
        });

        it('returns red at ratio 1', () => {
            const color = heatmapColor(1, 'green-red');
            expect(color).toBe('hsl(0, 55%, 88%)');
        });
    });

    describe('red-green scale', () => {
        it('returns red at ratio 0', () => {
            const color = heatmapColor(0, 'red-green');
            expect(color).toBe('hsl(0, 55%, 88%)');
        });

        it('returns yellow at ratio 0.5', () => {
            const color = heatmapColor(0.5, 'red-green');
            expect(color).toBe('hsl(60, 55%, 88%)');
        });

        it('returns green at ratio 1', () => {
            const color = heatmapColor(1, 'red-green');
            expect(color).toBe('hsl(120, 55%, 88%)');
        });
    });

    describe('blue scale', () => {
        it('returns low opacity at ratio 0', () => {
            const color = heatmapColor(0, 'blue');
            expect(color).toBe('rgba(49, 130, 189, 0.06)');
        });

        it('returns mid opacity at ratio 0.5', () => {
            const color = heatmapColor(0.5, 'blue');
            expect(color).toBe('rgba(49, 130, 189, 0.21)');
        });

        it('returns high opacity at ratio 1', () => {
            const color = heatmapColor(1, 'blue');
            expect(color).toBe('rgba(49, 130, 189, 0.36)');
        });
    });

    it('defaults to green-red for unknown scale', () => {
        const color = heatmapColor(0, 'unknown');
        expect(color).toBe('hsl(120, 55%, 88%)');
    });
});

describe('parseHeatmapConfig', () => {
    it('returns null for null input', () => {
        expect(parseHeatmapConfig(null)).toBeNull();
    });

    it('returns null for undefined input', () => {
        expect(parseHeatmapConfig(undefined)).toBeNull();
    });

    it('returns null for empty object', () => {
        expect(parseHeatmapConfig({})).toBeNull();
    });

    it('returns null for object without scale', () => {
        expect(parseHeatmapConfig({ other: 'value' })).toBeNull();
    });

    it('returns config for valid green-red scale', () => {
        const result = parseHeatmapConfig({ scale: 'green-red' });
        expect(result).toEqual({ scale: 'green-red' });
    });

    it('returns config for valid red-green scale', () => {
        const result = parseHeatmapConfig({ scale: 'red-green' });
        expect(result).toEqual({ scale: 'red-green' });
    });

    it('returns config for valid blue scale', () => {
        const result = parseHeatmapConfig({ scale: 'blue' });
        expect(result).toEqual({ scale: 'blue' });
    });

    it('returns rules when rules array is present', () => {
        const rules = [{ column: 'Coverage', thresholds: [50, 80], colors: 'red-yellow-green' }];
        const result = parseHeatmapConfig({ rules });
        expect(result).toEqual({ rules });
    });

    it('rules take precedence over scale', () => {
        const rules = [{ column: 'Coverage', thresholds: [50, 80], colors: 'red-yellow-green' }];
        const result = parseHeatmapConfig({ scale: 'green-red', rules });
        expect(result).toEqual({ rules });
    });

    it('falls back to scale when rules array is empty', () => {
        const result = parseHeatmapConfig({ scale: 'blue', rules: [] });
        expect(result).toEqual({ scale: 'blue' });
    });

    it('returns null for empty rules and no scale', () => {
        expect(parseHeatmapConfig({ rules: [] })).toBeNull();
    });
});

describe('heatmapRuleColor', () => {
    describe('red-yellow-green (low=bad)', () => {
        const rule = { thresholds: [50, 80], colors: 'red-yellow-green' };

        it('returns red below low threshold', () => {
            expect(heatmapRuleColor(30, rule)).toBe('hsl(0, 55%, 88%)');
        });

        it('returns yellow between thresholds', () => {
            expect(heatmapRuleColor(65, rule)).toBe('hsl(48, 55%, 88%)');
        });

        it('returns green at or above high threshold', () => {
            expect(heatmapRuleColor(80, rule)).toBe('hsl(120, 55%, 88%)');
            expect(heatmapRuleColor(95, rule)).toBe('hsl(120, 55%, 88%)');
        });

        it('returns yellow at exact low threshold', () => {
            expect(heatmapRuleColor(50, rule)).toBe('hsl(48, 55%, 88%)');
        });
    });

    describe('green-yellow-red (high=bad)', () => {
        const rule = { thresholds: [100, 500], colors: 'green-yellow-red' };

        it('returns green below low threshold', () => {
            expect(heatmapRuleColor(50, rule)).toBe('hsl(120, 55%, 88%)');
        });

        it('returns yellow between thresholds', () => {
            expect(heatmapRuleColor(250, rule)).toBe('hsl(48, 55%, 88%)');
        });

        it('returns red at or above high threshold', () => {
            expect(heatmapRuleColor(500, rule)).toBe('hsl(0, 55%, 88%)');
            expect(heatmapRuleColor(1000, rule)).toBe('hsl(0, 55%, 88%)');
        });

        it('returns yellow at exact low threshold', () => {
            expect(heatmapRuleColor(100, rule)).toBe('hsl(48, 55%, 88%)');
        });
    });

    it('defaults to red-yellow-green for unknown colors string', () => {
        const rule = { thresholds: [10, 20], colors: 'unknown' };
        expect(heatmapRuleColor(5, rule)).toBe('hsl(0, 55%, 88%)');
        expect(heatmapRuleColor(15, rule)).toBe('hsl(48, 55%, 88%)');
        expect(heatmapRuleColor(25, rule)).toBe('hsl(120, 55%, 88%)');
    });
});

describe('buildHeatmapRuleIndex', () => {
    const headers = ['District', 'Coverage', 'Cases', 'Population'];

    it('maps rules to correct column indices', () => {
        const rules = [
            { column: 'Coverage', thresholds: [50, 80], colors: 'red-yellow-green' },
            { column: 'Cases', thresholds: [100, 500], colors: 'green-yellow-red' },
        ];
        const index = buildHeatmapRuleIndex(rules, headers);
        expect(index[1]).toBe(rules[0]);
        expect(index[2]).toBe(rules[1]);
        expect(index[0]).toBeUndefined();
        expect(index[3]).toBeUndefined();
    });

    it('skips rules for missing columns', () => {
        const rules = [
            { column: 'NonExistent', thresholds: [10, 20], colors: 'red-yellow-green' },
        ];
        const index = buildHeatmapRuleIndex(rules, headers);
        expect(Object.keys(index)).toHaveLength(0);
    });

    it('skips rules with wrong thresholds length', () => {
        const rules = [
            { column: 'Coverage', thresholds: [50], colors: 'red-yellow-green' },
            { column: 'Cases', thresholds: [10, 20, 30], colors: 'green-yellow-red' },
        ];
        const index = buildHeatmapRuleIndex(rules, headers);
        expect(Object.keys(index)).toHaveLength(0);
    });

    it('skips rules without column name', () => {
        const rules = [
            { column: '', thresholds: [50, 80], colors: 'red-yellow-green' },
            { thresholds: [50, 80], colors: 'red-yellow-green' },
        ];
        const index = buildHeatmapRuleIndex(rules, headers);
        expect(Object.keys(index)).toHaveLength(0);
    });

    it('handles empty rules array', () => {
        const index = buildHeatmapRuleIndex([], headers);
        expect(index).toEqual({});
    });

    it('matches column names case-insensitively', () => {
        const rules = [
            { column: 'coverage', thresholds: [50, 80], colors: 'red-yellow-green' },
            { column: 'CASES', thresholds: [100, 500], colors: 'green-yellow-red' },
        ];
        const index = buildHeatmapRuleIndex(rules, headers);
        expect(index[1]).toBe(rules[0]); // 'coverage' matches 'Coverage'
        expect(index[2]).toBe(rules[1]); // 'CASES' matches 'Cases'
    });

    it('matches mixed-case headers against mixed-case rules', () => {
        const mixedHeaders = ['district_name', 'Total Cases', 'avg_COVERAGE'];
        const rules = [
            { column: 'total cases', thresholds: [10, 50], colors: 'red-yellow-green' },
            { column: 'AVG_coverage', thresholds: [60, 90], colors: 'red-yellow-green' },
        ];
        const index = buildHeatmapRuleIndex(rules, mixedHeaders);
        expect(index[1]).toBe(rules[0]);
        expect(index[2]).toBe(rules[1]);
        expect(index[0]).toBeUndefined();
    });
});

describe('computeHeatmapStats', () => {
    const alwaysNumeric = () => true;
    const neverNumeric = () => false;

    it('computes min/max for numeric columns', () => {
        const rows = [
            ['Kampala', 10, 200],
            ['Central', 50, 100],
            ['Jinja', 30, 300],
        ];
        const isNumeric = (c) => c > 0;
        const stats = computeHeatmapStats(rows, isNumeric, 3);

        expect(stats[1]).toEqual({ min: 10, max: 50 });
        expect(stats[2]).toEqual({ min: 100, max: 300 });
        expect(stats[0]).toBeUndefined(); // non-numeric column skipped
    });

    it('skips non-numeric columns', () => {
        const rows = [['A', 1], ['B', 2]];
        const stats = computeHeatmapStats(rows, neverNumeric, 2);

        expect(stats).toEqual({});
    });

    it('handles empty rows', () => {
        const stats = computeHeatmapStats([], alwaysNumeric, 3);
        expect(stats).toEqual({});
    });

    it('excludes columns where all values are the same (max === min)', () => {
        const rows = [
            [5, 10],
            [5, 20],
            [5, 30],
        ];
        const stats = computeHeatmapStats(rows, alwaysNumeric, 2);

        expect(stats[0]).toBeUndefined(); // all 5s — no coloring
        expect(stats[1]).toEqual({ min: 10, max: 30 });
    });

    it('skips NaN values in rows', () => {
        const rows = [
            [10, 'N/A'],
            [20, 100],
            [30, 200],
        ];
        const stats = computeHeatmapStats(rows, alwaysNumeric, 2);

        expect(stats[0]).toEqual({ min: 10, max: 30 });
        expect(stats[1]).toEqual({ min: 100, max: 200 });
    });

    it('handles single row (no range)', () => {
        const rows = [[42]];
        const stats = computeHeatmapStats(rows, alwaysNumeric, 1);

        expect(stats[0]).toBeUndefined(); // single value, min === max
    });
});
