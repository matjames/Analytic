/**
 * Tests for builder component form behavior.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';

vi.mock('./api.js', () => ({
    loadTableColumns: vi.fn(),
    loadTablePreview: vi.fn(),
    loadAvailableTables: vi.fn(),
    previewAdvancedSQL: vi.fn(),
}));

vi.mock('./query-form.js', () => ({
    addAggregation: vi.fn(),
    addCalculation: vi.fn(),
    addGroupBy: vi.fn(),
    renderCalculationRow: vi.fn(() => ''),
    renderSelectColumnsCheckboxes: vi.fn(() => ''),
    refreshSelectColumnsList: vi.fn(),
    autoExpandCalculateIfNeeded: vi.fn(),
    autoExpandOrderByIfNeeded: vi.fn(),
    autoExpandPeriodLimitIfNeeded: vi.fn(),
    autoExpandBarOptionsIfNeeded: vi.fn(),
    autoExpandLegendIfNeeded: vi.fn(),
    autoExpandTransposeIfNeeded: vi.fn(),
    loadTransposeConfig: vi.fn(),
    setupTransposeListeners: vi.fn(),
    setupFacetListeners: vi.fn(),
    populateColumnSuggestions: vi.fn(),
    maybeSetDefaultPeriodLimit: vi.fn(),
    getAvailableColumns: vi.fn(() => []),
    refreshCalcAliasChips: vi.fn(),
    wireUpExistingCalcRows: vi.fn(),
}));

vi.mock('./autocomplete.js', () => ({
    initAllAutocomplete: vi.fn(),
    refreshAutocomplete: vi.fn(),
}));

import { initElements, reportState, setCurrentEditingComponent } from './state.js';
import { renderComponentForm, saveComponent, setCallbacks } from './components.js';

function renderMinimalAdvancedTableForm(pageSizeValue) {
    const pageSizeInput = pageSizeValue === undefined
        ? ''
        : `<input id="table-page-size" value="${pageSizeValue}">`;

    document.body.innerHTML = `
        <div id="component-modal"></div>
        <div id="component-form-container">
            <input id="comp-type" value="table_advanced">
            <input id="comp-title" value="Advanced Table">
            <input id="comp-description" value="">
            <textarea id="advanced-sql">SELECT * FROM report.example</textarea>
            ${pageSizeInput}
        </div>
    `;
    initElements();
}

describe('builder table page size', () => {
    beforeEach(() => {
        reportState.sections = [{ id: 'section-1', components: [] }];
        setCurrentEditingComponent(null);
        setCallbacks({
            renderSections: vi.fn(),
            updatePreview: vi.fn(),
            previewQueryData: vi.fn(),
        });
        document.body.innerHTML = '';
    });

    it('saves custom rows per page for advanced tables', () => {
        renderMinimalAdvancedTableForm('5');
        setCurrentEditingComponent({ sectionIndex: 0, compIndex: null, component: {} });

        saveComponent();

        expect(reportState.sections[0].components[0].pageSize).toBe(5);
    });

    it('uses implicit default when rows per page is saved as 20', () => {
        reportState.sections[0].components = [{ type: 'table_advanced', pageSize: 5 }];
        renderMinimalAdvancedTableForm('20');
        setCurrentEditingComponent({
            sectionIndex: 0,
            compIndex: 0,
            component: { type: 'table_advanced', pageSize: 5 },
        });

        saveComponent();

        expect(reportState.sections[0].components[0]).not.toHaveProperty('pageSize');
    });

    it('preserves an existing custom page size if the form control is missing', () => {
        reportState.sections[0].components = [{ type: 'table_advanced', pageSize: 5 }];
        renderMinimalAdvancedTableForm(undefined);
        setCurrentEditingComponent({
            sectionIndex: 0,
            compIndex: 0,
            component: { type: 'table_advanced', pageSize: 5 },
        });

        saveComponent();

        expect(reportState.sections[0].components[0].pageSize).toBe(5);
    });

    it('renders saved custom rows per page instead of falling back to 20', () => {
        document.body.innerHTML = renderComponentForm({
            type: 'table_advanced',
            title: 'Advanced Table',
            sql: 'SELECT * FROM report.example',
            pageSize: 5,
        });

        expect(document.getElementById('table-page-size').value).toBe('5');
    });
});
