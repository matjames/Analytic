// Builder State Management
// Centralized state for the report builder application

// API base URL - uses BASE_PATH from server for subpath deployments
export const API_BASE = (window.BASE_PATH || '') + '/api';
export const DEFAULT_LIMIT = 10;

// Global application state
export const reportState = {
    id: '',
    title: '',
    keywords: '', // Comma-separated keywords for search discoverability
    author: '',   // Author name for audit trail (not displayed in reports)
    datasource: '', // Optional named .env DB connection to read from (blank = primary)
    filters: [],
    customFilters: [], // Array of {column, table, label, type, defaultValue?}
    timeColumns: {},
    locationColumns: {},
    sections: [],
    currentTab: 'metadata',
    availableSchemas: [],
    selectedSchema: 'report',
    availableTables: [],
    availableColumns: {},
    selectedTable: null,
    isDirty: false, // Track unsaved changes
};

// Callback for when state changes (set by main.js)
let onStateChange = null;

export function setOnStateChange(callback) {
    onStateChange = callback;
}

// Mark state as dirty and notify listeners
export function markDirty() {
    reportState.isDirty = true;
    if (onStateChange) onStateChange();
}

// Mark state as clean (after save/load)
export function markClean() {
    reportState.isDirty = false;
}

// Publishing state (shared to avoid circular imports)
let publishingAllowed = true;

export function setPublishingAllowed(allowed) {
    publishingAllowed = allowed;
}

export function isPublishingAllowed() {
    return publishingAllowed;
}

// Current component being edited in the modal
export let currentEditingComponent = null;

export function setCurrentEditingComponent(value) {
    currentEditingComponent = value;
}

// DOM element references - initialized after DOM loads
export let elements = {};

export function initElements() {
    elements = {
        // Form inputs
        reportId: document.getElementById('report-id'),
        reportTitle: document.getElementById('report-title'),
        reportKeywords: document.getElementById('report-keywords'),
        reportAuthor: document.getElementById('report-author'),
        // reportDatasource removed (single-connection mode)
        filterCheckboxes: document.querySelectorAll('input[name="filters"]'),

        // Containers
        sectionsContainer: document.getElementById('sections-container'),
        componentsContainer: document.getElementById('components-container'),
        previewContent: document.getElementById('preview-content'),
        importYaml: document.getElementById('import-yaml'),

        // Buttons
        importMenuBtn: document.getElementById('btn-import-menu'),
        importYamlBtn: document.getElementById('btn-import-yaml'),
        copyPreviewBtn: document.getElementById('btn-copy-preview'),
        generateYamlBtn: document.getElementById('btn-generate-yaml'),

        // Modals
        importYamlModal: document.getElementById('import-yaml-modal'),
        closeImportModalBtn: document.getElementById('btn-close-import-modal'),
        closeImportModalBtn2: document.getElementById('btn-close-import-modal-2'),

        componentModal: document.getElementById('component-modal'),
        closeComponentModalBtn: document.getElementById('btn-close-component-modal'),
        saveComponentBtn: document.getElementById('btn-save-component'),
        componentFormContainer: document.getElementById('component-form-container'),

        yamlModal: document.getElementById('yaml-modal'),
        closeYamlModalBtn: document.getElementById('btn-close-yaml-modal'),
        yamlOutput: document.getElementById('yaml-output'),
        copyYamlBtn: document.getElementById('btn-copy-yaml'),
        downloadYamlBtn: document.getElementById('btn-download-yaml'),

        previewModal: document.getElementById('preview-modal'),
        closePreviewModalBtn: document.getElementById('btn-close-preview-modal'),
        previewResultContainer: document.getElementById('preview-result-container'),

    };
}
