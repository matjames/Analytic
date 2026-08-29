import { initAuth } from './auth.js';
import { authFetchJSON, authPostJSON, authPutJSON } from './authFetch.js';

const DRAFT_KEY = 'requirements_form_draft_v1';

// Set to a numeric ID when the form is opened in edit mode (?edit={id})
let editId = null;

function escapeHtml(value) {
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

function byId(id) {
    return document.getElementById(id);
}

function collectChecked(name) {
    return Array.from(document.querySelectorAll(`input[name="${name}"]:checked`)).map((el) => el.value);
}

function normalizeDataElementRow(row) {
    if (!row) return null;
    if (typeof row === 'string') {
        return { label: row, dataElement: row, dataElementId: '', categoryCombo: '' };
    }
    return {
        label: row.label || row.dataElement || row.dataElementId || '',
        dataElement: row.dataElement || '',
        dataElementId: row.dataElementId || '',
        categoryCombo: row.categoryCombo || '',
    };
}

function selectorState(root) {
    if (!root) {
        return { selected: [], query: '', searchTimer: null, searchToken: 0 };
    }
    if (!root.__selectorState) {
        root.__selectorState = { selected: [], query: '', searchTimer: null, searchToken: 0 };
    }
    return root.__selectorState;
}

function createSelectorMarkup(root, kind, initialSelected = []) {
    root.innerHTML = `
        <div class="selector-head">
            <div class="selector-title">${kind}</div>
            <span class="selector-count"></span>
        </div>
        <div class="selector-search-row">
            <input type="search" class="selector-search" placeholder="Search tables">
            <button type="button" class="btn btn-secondary btn-sm selector-search-btn">Search</button>
        </div>
        <div class="selector-selected"></div>
        <div class="selector-results hidden"></div>
    `;
    const state = selectorState(root);
    state.selected = initialSelected.map(normalizeDataElementRow).filter(Boolean);
    renderSelectorSelected(root);
}

function renderSelectorSelected(root) {
    const state = selectorState(root);
    const selectedEl = root.querySelector('.selector-selected');
    const countEl = root.querySelector('.selector-count');
    if (!selectedEl) return;

    selectedEl.innerHTML = state.selected.length
        ? state.selected.map((item, index) => `
            <span class="selector-chip" data-index="${index}">
                <span class="selector-chip-label">${escapeHtml(item.label || item.dataElement || item.dataElementId || '')}</span>
                <button type="button" class="selector-chip-remove" aria-label="Remove selected data element">×</button>
            </span>
        `).join('')
        : '<span class="selector-empty">No data elements selected yet.</span>';

    if (countEl) {
        countEl.textContent = state.selected.length ? `${state.selected.length} selected` : '';
    }
}

function addSelectorItem(root, item) {
    const state = selectorState(root);
    const normalized = normalizeDataElementRow(item);
    if (!normalized) return;

    const exists = state.selected.some((existing) =>
        (existing.dataElementId && existing.dataElementId === normalized.dataElementId) ||
        (existing.dataElement && existing.dataElement === normalized.dataElement)
    );
    if (exists) return;

    state.selected.push(normalized);
    renderSelectorSelected(root);
}

async function searchSelector(root, query) {
    const state = selectorState(root);
    state.query = query;
    state.searchToken += 1;
    const token = state.searchToken;

    const resultsEl = root.querySelector('.selector-results');
    if (!resultsEl) return;

    if (!query) {
        resultsEl.innerHTML = '';
        resultsEl.classList.add('hidden');
        return;
    }

    const basePath = window.BASE_PATH || '';
    // List tables from the live schema catalog (the data-element
    // search service was removed).
    const url = `${basePath}/api/schema/tables`;
    resultsEl.classList.remove('hidden');
    resultsEl.innerHTML = '<div class="selector-loading">Searching...</div>';

    try {
        const raw = await authFetchJSON(url);
        if (token !== selectorState(root).searchToken) return;
        const tables = Array.isArray(raw) ? raw : [];
        const results = tables
            .filter((t) => !query || String(t.table || t.name || '').toLowerCase().includes(query.toLowerCase()))
            .map((t) => ({ label: `${t.schema || ''}.${t.table || t.name || ''}` }))
            .slice(0, 300);
        if (results.length === 0) {
            resultsEl.innerHTML = '<div class="selector-empty">No matches found.</div>';
            return;
        }

        resultsEl.innerHTML = results.map((item, index) => `
            <button type="button" class="selector-result-item" data-index="${index}">
                <span class="selector-result-title">${escapeHtml(item.label || '')}</span>
            </button>
        `).join('');

        resultsEl.querySelectorAll('.selector-result-item').forEach((button, index) => {
            button.addEventListener('click', () => {
                addSelectorItem(root, results[index]);
                button.classList.add('selector-result-item-selected');
            });
        });
    } catch (error) {
        if (token !== selectorState(root).searchToken) return;
        resultsEl.innerHTML = `<div class="selector-empty">Search failed: ${escapeHtml(error.message || 'unknown error')}</div>`;
    }
}

function setupSelector(root, kind, initialSelected = []) {
    createSelectorMarkup(root, kind, initialSelected);

    root.querySelector('.selector-search-btn').addEventListener('click', () => {
        const query = root.querySelector('.selector-search')?.value?.trim() || '';
        searchSelector(root, query);
    });

    root.querySelector('.selector-search').addEventListener('input', () => {
        const query = root.querySelector('.selector-search')?.value?.trim() || '';
        const state = selectorState(root);
        if (state.searchTimer) {
            clearTimeout(state.searchTimer);
        }
        state.searchTimer = setTimeout(() => {
            searchSelector(root, query);
        }, 250);
    });

    root.querySelector('.selector-search').addEventListener('keydown', (event) => {
        if (event.key === 'Enter') {
            event.preventDefault();
            root.querySelector('.selector-search-btn').click();
        }
    });

    root.querySelector('.selector-selected').addEventListener('click', (event) => {
        const removeButton = event.target.closest('.selector-chip-remove');
        if (!removeButton) return;
        const chip = removeButton.closest('.selector-chip');
        if (!chip) return;
        const index = Number(chip.dataset.index);
        if (Number.isFinite(index)) {
            const state = selectorState(root);
            state.selected.splice(index, 1);
            renderSelectorSelected(root);
        }
    });
}

function collectVisualSpecs() {
    return Array.from(document.querySelectorAll('.visual-spec-item')).map((item) => {
        const visualType = item.querySelector('.visual-type')?.value?.trim() || '';
        const visualTitle = item.querySelector('.visual-title')?.value?.trim() || '';
        const visualDescription = item.querySelector('.visual-description')?.value?.trim() || '';
        const indicators = Array.from(item.querySelectorAll('.visual-indicator-row')).map((row) => {
            const mode = row.querySelector('.visual-indicator-mode:checked')?.value || 'direct';
            const label = row.querySelector('.visual-indicator-label')?.value?.trim() || '';
            const functionName = row.querySelector('.visual-indicator-function')?.value || 'sum';
            const mainSelector = selectorState(row.querySelector('.visual-selector-block')).selected || [];
            const numeratorSelector = selectorState(row.querySelector('.visual-numerator-selector')).selected || [];
            const denominatorSelector = selectorState(row.querySelector('.visual-denominator-selector')).selected || [];

            return {
                indicatorLabel: label,
                mode,
                function: functionName,
                dataElements: mainSelector,
                numerator: numeratorSelector,
                denominator: denominatorSelector,
            };
        }).filter((row) => row.indicatorLabel || row.dataElements.length || row.numerator.length || row.denominator.length);

        return {
            visualType,
            visualTitle,
            visualDescription,
            indicators,
        };
    }).filter((row) => row.visualType || row.visualTitle || row.visualDescription || row.indicators.length > 0);
}

function readFormPayload() {
    const form = byId('requirements-form');
    const fd = new FormData(form);
    const value = (name) => String(fd.get(name) || '').trim();

    return {
        documentControl: {
            reportName: value('reportName'),
            reportId: value('reportId'),
            programArea: value('programArea'),
            requestingDepartment: value('requestingDepartment'),
            businessOwner: value('businessOwner'),
            dataAnalyst: value('dataAnalyst'),
            reportDeveloper: value('reportDeveloper'),
            dateRequested: value('dateRequested'),
            version: value('version'),
            status: value('status'),
        },
        reportOverview: {
            reportTitle: value('overviewTitle'),
            businessObjective: value('businessObjective'),
            targetAudience: value('targetAudience'),
            reportingFrequency: value('reportingFrequency'),
            reportingPeriod: value('reportingPeriod'),
        },
        reportAccessRequirements: {
            userRoles: value('userRoles'),
            permissions: value('permissions'),
            authenticationRequirements: value('authenticationRequirements'),
        },
        dataRequirements: {
            dataSources: value('dataSources'),
            dataQualityRequirements: value('dataQualityRequirements'),
        },
        reportFilters: {
            selectedFilters: collectChecked('filters'),
            otherFilters: value('otherFilters'),
        },
        indicatorRequirements: [],
        reportStructure: collectChecked('reportStructure'),
        visualizationRequirements: {
            specifications: collectVisualSpecs(),
            notes: value('visualizationRequirements'),
        },
        geospatialRequirements: {
            maps: value('maps'),
            geographicLevels: value('geographicLevels'),
            metricsDisplayed: value('metricsDisplayed'),
        },
        exportRequirements: collectChecked('exportFormats'),
        navigationRequirements: {
            sections: value('navigationSections'),
            navigationLabels: value('navigationLabels'),
            bookmarks: value('bookmarks'),
        },
        businessRules: value('businessRules'),
        performanceRequirements: {
            loadTimeSeconds: value('loadTimeSeconds'),
            exportTimeSeconds: value('exportTimeSeconds'),
            expectedUsers: value('expectedUsers'),
        },
        validationRequirements: collectChecked('validationRequirements'),
        acceptanceCriteria: collectChecked('acceptanceCriteria'),
        approvals: {
            preparedBy: value('preparedBy'),
            reviewedBy: value('reviewedBy'),
            approvedBy: value('approvedBy'),
            preparedDate: value('preparedDate'),
            reviewedDate: value('reviewedDate'),
            approvedDate: value('approvedDate'),
        },
    };
}

function renderStatus(kind, text) {
    const status = byId('submit-status');
    status.classList.remove('hidden', 'success', 'error');
    status.classList.add(kind);
    status.textContent = text;
}

function appendVisualIndicatorRow(container, initial = '') {
    const tpl = byId('visual-indicator-template');
    const node = tpl.content.firstElementChild.cloneNode(true);
    const rowId = `mode_${Math.random().toString(36).slice(2, 10)}`;
    node.querySelectorAll('.visual-indicator-mode').forEach((radio) => {
        radio.name = rowId;
    });

    // Accept either a plain string label or a full indicator object
    const isObj = initial && typeof initial === 'object';
    const labelStr = isObj ? (initial.indicatorLabel || initial.label || '') : String(initial || '');
    const initialMode = isObj ? (initial.mode || 'direct') : 'direct';
    const initialFunction = isObj ? (initial.function || 'sum') : 'sum';
    const initialDataElements = isObj && Array.isArray(initial.dataElements) ? initial.dataElements : [];
    const initialNumerator = isObj && Array.isArray(initial.numerator) ? initial.numerator : [];
    const initialDenominator = isObj && Array.isArray(initial.denominator) ? initial.denominator : [];

    const labelInput = node.querySelector('.visual-indicator-label');
    if (labelInput) labelInput.value = labelStr;

    // Restore mode radio
    node.querySelectorAll('.visual-indicator-mode').forEach((r) => { r.checked = r.value === initialMode; });

    // Restore function select
    const funcSelect = node.querySelector('.visual-indicator-function');
    if (funcSelect) funcSelect.value = initialFunction;

    const mainSelector = node.querySelector('.visual-selector-block');
    const numeratorSelector = node.querySelector('.visual-numerator-selector');
    const denominatorSelector = node.querySelector('.visual-denominator-selector');
    setupSelector(mainSelector, 'Indicator data elements', initialDataElements);
    setupSelector(numeratorSelector, 'Numerator data elements', initialNumerator);
    setupSelector(denominatorSelector, 'Denominator data elements', initialDenominator);

    const calcBlock = node.querySelector('.visual-indicator-calculation');
    const resultsBlock = node.querySelector('.visual-indicator-results');
    const modes = Array.from(node.querySelectorAll('.visual-indicator-mode'));
    const toggleMode = () => {
        const selectedMode = node.querySelector('.visual-indicator-mode:checked')?.value || 'direct';
        if (calcBlock) {
            calcBlock.classList.toggle('hidden', selectedMode === 'direct');
        }
        if (resultsBlock) {
            resultsBlock.classList.toggle('hidden', true);
        }
    };
    modes.forEach((mode) => mode.addEventListener('change', toggleMode));
    toggleMode();

    node.querySelector('.remove-visual-indicator').addEventListener('click', () => {
        node.remove();
    });
    container.appendChild(node);
}

function renumberVisualSpecs() {
    byId('visual-spec-list').querySelectorAll('.visual-spec-item').forEach((item, i) => {
        const h3 = item.querySelector('.visual-spec-number');
        if (h3) h3.textContent = `Component ${i + 1}`;
    });
}

function appendVisualSpecRow(initial = {}) {
    const tpl = byId('visual-spec-template');
    const node = tpl.content.firstElementChild.cloneNode(true);
    const indicatorList = node.querySelector('.visual-indicator-list');

    const visualType = node.querySelector('.visual-type');
    if (visualType) visualType.value = initial.visualType || 'Bar';
    const visualTitle = node.querySelector('.visual-title');
    if (visualTitle) visualTitle.value = initial.visualTitle || '';
    const visualDescription = node.querySelector('.visual-description');
    if (visualDescription) visualDescription.value = initial.visualDescription || '';

    node.querySelector('.remove-visual-spec').addEventListener('click', () => {
        node.remove();
        renumberVisualSpecs();
    });

    node.querySelector('.add-visual-indicator').addEventListener('click', () => {
        appendVisualIndicatorRow(indicatorList);
    });

    const indicators = Array.isArray(initial.indicators) ? initial.indicators : [];
    if (indicators.length > 0) {
        indicators.forEach((ind) => appendVisualIndicatorRow(indicatorList, ind));
    } else {
        appendVisualIndicatorRow(indicatorList);
    }

    byId('visual-spec-list').appendChild(node);
    renumberVisualSpecs();
}

// Populates all form fields from a payload object.
// Used by both loadDraft (from localStorage) and loadEditEntry (from server).
function populateFormFromPayload(payload) {
    const form = byId('requirements-form');
    const set = (name, val) => {
        const field = form.elements.namedItem(name);
        if (field && typeof val === 'string') {
            field.value = val;
        }
    };

    const dc = payload.documentControl || {};
    set('reportName', dc.reportName || '');
    set('reportId', dc.reportId || '');
    set('programArea', dc.programArea || '');
    set('requestingDepartment', dc.requestingDepartment || '');
    set('businessOwner', dc.businessOwner || '');
    set('dataAnalyst', dc.dataAnalyst || '');
    set('reportDeveloper', dc.reportDeveloper || '');
    set('dateRequested', dc.dateRequested || '');
    set('version', dc.version || '');
    set('status', dc.status || 'Draft');

    const ro = payload.reportOverview || {};
    set('overviewTitle', ro.reportTitle || '');
    set('businessObjective', ro.businessObjective || '');
    set('targetAudience', ro.targetAudience || '');
    set('reportingFrequency', ro.reportingFrequency || 'Monthly');
    set('reportingPeriod', ro.reportingPeriod || '');

    const ar = payload.reportAccessRequirements || {};
    set('userRoles', ar.userRoles || '');
    set('permissions', ar.permissions || '');
    set('authenticationRequirements', ar.authenticationRequirements || '');

    const dr = payload.dataRequirements || {};
    set('dataSources', dr.dataSources || '');
    set('dataQualityRequirements', dr.dataQualityRequirements || '');

    set('otherFilters', payload.reportFilters?.otherFilters || '');
    const vizReq = payload.visualizationRequirements;
    if (typeof vizReq === 'string') {
        set('visualizationRequirements', vizReq);
    } else {
        set('visualizationRequirements', vizReq?.notes || '');
    }
    set('maps', payload.geospatialRequirements?.maps || '');
    set('geographicLevels', payload.geospatialRequirements?.geographicLevels || '');
    set('metricsDisplayed', payload.geospatialRequirements?.metricsDisplayed || '');
    set('navigationSections', payload.navigationRequirements?.sections || '');
    set('navigationLabels', payload.navigationRequirements?.navigationLabels || '');
    set('bookmarks', payload.navigationRequirements?.bookmarks || '');
    set('businessRules', payload.businessRules || '');
    set('loadTimeSeconds', payload.performanceRequirements?.loadTimeSeconds || '');
    set('exportTimeSeconds', payload.performanceRequirements?.exportTimeSeconds || '');
    set('expectedUsers', payload.performanceRequirements?.expectedUsers || '');
    set('preparedBy', payload.approvals?.preparedBy || '');
    set('reviewedBy', payload.approvals?.reviewedBy || '');
    set('approvedBy', payload.approvals?.approvedBy || '');
    set('preparedDate', payload.approvals?.preparedDate || '');
    set('reviewedDate', payload.approvals?.reviewedDate || '');
    set('approvedDate', payload.approvals?.approvedDate || '');

    const markChecked = (name, values) => {
        if (!Array.isArray(values)) return;
        const selected = new Set(values);
        document.querySelectorAll(`input[name="${name}"]`).forEach((el) => {
            el.checked = selected.has(el.value);
        });
    };

    markChecked('filters', payload.reportFilters?.selectedFilters || []);
    markChecked('reportStructure', payload.reportStructure || []);
    markChecked('exportFormats', payload.exportRequirements || []);
    markChecked('validationRequirements', payload.validationRequirements || []);
    markChecked('acceptanceCriteria', payload.acceptanceCriteria || []);

    byId('visual-spec-list').innerHTML = '';
    const visualSpecs = Array.isArray(vizReq?.specifications) ? vizReq.specifications : [];
    if (visualSpecs.length) {
        visualSpecs.forEach((row) => appendVisualSpecRow(row));
    } else {
        appendVisualSpecRow();
    }
}

function saveDraft() {
    const payload = readFormPayload();
    localStorage.setItem(DRAFT_KEY, JSON.stringify(payload));
    renderStatus('success', 'Draft saved on this browser.');
}

function loadDraft() {
    const raw = localStorage.getItem(DRAFT_KEY);
    if (!raw) {
        appendVisualSpecRow();
        return;
    }

    try {
        const payload = JSON.parse(raw);
        populateFormFromPayload(payload);
        renderStatus('success', 'Draft loaded from this browser.');
    } catch {
        appendVisualSpecRow();
    }
}

async function submitForm(event) {
    event.preventDefault();

    const payload = readFormPayload();
    if (!payload.documentControl.reportName) {
        renderStatus('error', 'Report name is required.');
        return;
    }
    if (!payload.reportOverview.businessObjective) {
        renderStatus('error', 'Business objective is required.');
        return;
    }

    const submitBtn = byId('submit-form');
    submitBtn.disabled = true;
    submitBtn.textContent = editId ? 'Updating...' : 'Submitting...';

    try {
        const basePath = window.BASE_PATH || '';
        if (editId) {
            await authPutJSON(`${basePath}/api/requirements-specs/${editId}`, payload);
            renderStatus('success', 'Updated successfully.');
        } else {
            const response = await authPostJSON(`${basePath}/api/requirements-specs`, payload);
            localStorage.removeItem(DRAFT_KEY);
            renderStatus('success', `Submitted successfully. Reference ID: ${response.id}`);
            byId('requirements-form').reset();
            byId('visual-spec-list').innerHTML = '';
            appendVisualSpecRow();
        }
    } catch (error) {
        renderStatus('error', error.message || (editId ? 'Failed to update requirements.' : 'Failed to submit requirements.'));
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = editId ? 'Update Requirements' : 'Submit Requirements';
    }
}

async function loadEditEntry(id) {
    const basePath = window.BASE_PATH || '';
    try {
        const entry = await authFetchJSON(`${basePath}/api/requirements-specs/${id}`);
        const payload = JSON.parse(entry.payloadJson);
        populateFormFromPayload(payload);

        // Switch UI to edit mode
        const submitBtn = byId('submit-form');
        if (submitBtn) submitBtn.textContent = 'Update Requirements';
        const h1 = document.querySelector('.requirements-hero h1');
        if (h1) h1.textContent = 'Edit Requirements Specification';
        renderStatus('success', `Editing: ${entry.reportName}`);
    } catch (err) {
        renderStatus('error', `Failed to load entry for editing: ${err.message || err}`);
        appendVisualSpecRow();
    }
}

async function initPage() {
    await initAuth();

    byId('add-visual-spec').addEventListener('click', () => appendVisualSpecRow());
    byId('save-draft').addEventListener('click', saveDraft);
    byId('requirements-form').addEventListener('submit', submitForm);

    const params = new URLSearchParams(window.location.search);
    const editParam = params.get('edit');
    if (editParam) {
        const id = parseInt(editParam, 10);
        if (id > 0) {
            editId = id;
            await loadEditEntry(id);
            return;
        }
    }

    loadDraft();
}

initPage().catch((error) => {
    renderStatus('error', `Unable to initialize form: ${error.message || error}`);
});
