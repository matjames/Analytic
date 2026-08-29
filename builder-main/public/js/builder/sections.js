// Sections Module
// Section management and rendering

import { elements, reportState, setCurrentEditingComponent, markDirty } from './state.js';

// Callbacks set by main.js to avoid circular dependencies
let onShowComponentModal = () => {};
let onUpdatePreview = () => {};

export function setCallbacks(callbacks) {
    if (callbacks.showComponentModal) onShowComponentModal = callbacks.showComponentModal;
    if (callbacks.updatePreview) onUpdatePreview = callbacks.updatePreview;
}

/**
 * Calculate layout based on component count
 */
export function calculateLayout(componentCount) {
    if (componentCount <= 1) return 'single';
    if (componentCount === 2) return 'two-column';
    if (componentCount === 3) return 'three-column';
    if (componentCount === 4) return 'four-column';
    return 'three-column'; // Cap at 3-column for 5+
}

/**
 * Add a new section
 */
export function addSection() {
    const sectionId = `section-${reportState.sections.length + 1}`;
    const section = {
        id: sectionId,
        title: '',
        description: '',
        layout: 'single',
        components: [],
    };

    reportState.sections.push(section);
    renderSections();
}

/**
 * Remove a section
 */
export function removeSection(index) {
    if (confirm('Remove this section?')) {
        reportState.sections.splice(index, 1);
        renderSections();
    }
}

/**
 * Add component to a section
 */
export function addComponent(sectionIndex) {
    setCurrentEditingComponent({ sectionIndex, compIndex: null, component: {} });
    onShowComponentModal('Add Component');
}

/**
 * Edit a component
 */
export function editComponent(sectionIndex, compIndex) {
    const original = reportState.sections[sectionIndex].components[compIndex];
    const component = JSON.parse(JSON.stringify(original)); // deep copy
    normalizeComponentColors(component);
    setCurrentEditingComponent({ sectionIndex, compIndex, component });
    onShowComponentModal('Edit Component');
}

/**
 * Normalize colors: copy dataset colors to aggregation/calculate colors when missing.
 * Hand-crafted YAMLs often store colors on data.datasets only, but the builder
 * reads colors from query.aggregations/calculate. This bridges the gap.
 */
function normalizeComponentColors(component) {
    const datasets = component.data?.datasets || [];
    if (datasets.length === 0) return;

    if (component.query?.aggregations) {
        component.query.aggregations.forEach(agg => {
            if (!agg.color) {
                const ds = datasets.find(d => d.label === agg.alias);
                if (ds?.color) agg.color = ds.color;
            }
        });
    }

    if (component.query?.calculate) {
        component.query.calculate.forEach(calc => {
            const alias = calc.resultAlias || 'value';
            if (!calc.color) {
                const ds = datasets.find(d => d.label === alias);
                if (ds?.color) calc.color = ds.color;
            }
        });
    }
}

/**
 * Remove a component
 */
export function removeComponent(sectionIndex, compIndex) {
    if (confirm('Remove this component?')) {
        reportState.sections[sectionIndex].components.splice(compIndex, 1);
        renderSections();
    }
}

/**
 * Duplicate a component within the same section
 */
export function duplicateComponent(sectionIndex, compIndex) {
    const original = reportState.sections[sectionIndex].components[compIndex];

    // Deep clone the component
    const copy = JSON.parse(JSON.stringify(original));

    // Append "(copy)" to title if it exists
    if (copy.title) {
        copy.title = copy.title + ' (copy)';
    }

    // Insert copy right after the original
    reportState.sections[sectionIndex].components.splice(compIndex + 1, 0, copy);

    markDirty();
    renderSections();
}

/**
 * Render component card preview
 */
function renderComponentCard(component, sectionIndex, compIndex) {
    const cardEl = document.createElement('div');
    cardEl.className = 'component-card-preview';

    // Make card draggable
    cardEl.draggable = true;
    cardEl.dataset.sectionIndex = sectionIndex;
    cardEl.dataset.compIndex = compIndex;

    // Build query summary
    let querySummary = '';
    if (component.query) {
        const parts = [];
        if (component.query.table) {
            parts.push(`Table: ${component.query.table.split('.')[1] || component.query.table}`);
        }
        if (component.query.aggregations && component.query.aggregations.length > 0) {
            parts.push(`${component.query.aggregations.length} aggregation${component.query.aggregations.length !== 1 ? 's' : ''}`);
        }
        if (component.query.groupBy && component.query.groupBy.length > 0) {
            parts.push(`Grouped by: ${component.query.groupBy[0].field}`);
        }
        querySummary = parts.join(' • ');
    } else if (component.type === 'text') {
        querySummary = 'Static text/KPI';
    } else if (component.type === 'kpi') {
        querySummary = component.tone ? `KPI · ${component.tone}` : 'KPI';
    }

    cardEl.innerHTML = `
        <div class="component-drag-handle" title="Drag to reorder">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                <circle cx="5" cy="5" r="2"></circle>
                <circle cx="12" cy="5" r="2"></circle>
                <circle cx="5" cy="12" r="2"></circle>
                <circle cx="12" cy="12" r="2"></circle>
                <circle cx="5" cy="19" r="2"></circle>
                <circle cx="12" cy="19" r="2"></circle>
            </svg>
        </div>
        <div class="component-preview-header">
            <div class="component-preview-content">
                <div class="component-type-badge">${component.type}</div>
                <div class="component-preview-title">${component.title || 'Untitled'}</div>
                ${querySummary ? `<div class="component-preview-summary">${querySummary}</div>` : ''}
            </div>
        </div>
        <div class="component-card-actions">
            <button class="btn-duplicate-component" title="Duplicate component">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                </svg>
            </button>
            <button class="btn-delete-component" title="Delete component">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"></polyline>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                </svg>
            </button>
        </div>
    `;

    // Drag events
    cardEl.addEventListener('dragstart', handleDragStart);
    cardEl.addEventListener('dragend', handleDragEnd);
    cardEl.addEventListener('dragover', handleDragOver);
    cardEl.addEventListener('dragenter', handleDragEnter);
    cardEl.addEventListener('dragleave', handleDragLeave);
    cardEl.addEventListener('drop', handleDrop);

    // Wire up edit (click on card), duplicate, and delete
    cardEl.addEventListener('click', (e) => {
        if (!e.target.closest('.btn-delete-component') &&
            !e.target.closest('.btn-duplicate-component') &&
            !e.target.closest('.component-drag-handle')) {
            editComponent(sectionIndex, compIndex);
        }
    });

    const duplicateBtn = cardEl.querySelector('.btn-duplicate-component');
    duplicateBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        duplicateComponent(sectionIndex, compIndex);
    });

    const deleteBtn = cardEl.querySelector('.btn-delete-component');
    deleteBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        removeComponent(sectionIndex, compIndex);
    });

    return cardEl;
}

// ========================================
// Drag and Drop Handling
// ========================================

let draggedElement = null;
let draggedSectionIndex = null;
let draggedCompIndex = null;

function handleDragStart(e) {
    draggedElement = this;
    draggedSectionIndex = parseInt(this.dataset.sectionIndex);
    draggedCompIndex = parseInt(this.dataset.compIndex);

    this.classList.add('dragging');

    // Set drag data
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', `${draggedSectionIndex},${draggedCompIndex}`);

    // Slight delay to allow the drag image to be captured before adding opacity
    setTimeout(() => {
        this.style.opacity = '0.4';
    }, 0);
}

function handleDragEnd(e) {
    this.classList.remove('dragging');
    this.style.opacity = '1';

    // Remove all drag-over states
    document.querySelectorAll('.component-card-preview').forEach(card => {
        card.classList.remove('drag-over', 'drag-over-left', 'drag-over-right');
    });

    draggedElement = null;
    draggedSectionIndex = null;
    draggedCompIndex = null;
}

function handleDragOver(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';

    // Determine if dropping left or right of center
    const rect = this.getBoundingClientRect();
    const midpoint = rect.left + rect.width / 2;

    this.classList.remove('drag-over-left', 'drag-over-right');
    if (e.clientX < midpoint) {
        this.classList.add('drag-over-left');
    } else {
        this.classList.add('drag-over-right');
    }
}

function handleDragEnter(e) {
    e.preventDefault();
    if (this !== draggedElement) {
        this.classList.add('drag-over');
    }
}

function handleDragLeave(e) {
    // Only remove if actually leaving the element (not entering a child)
    if (!this.contains(e.relatedTarget)) {
        this.classList.remove('drag-over', 'drag-over-left', 'drag-over-right');
    }
}

function handleDrop(e) {
    e.preventDefault();
    e.stopPropagation();

    if (this === draggedElement) return;

    const targetSectionIndex = parseInt(this.dataset.sectionIndex);
    const targetCompIndex = parseInt(this.dataset.compIndex);

    // Only allow reordering within the same section
    if (targetSectionIndex !== draggedSectionIndex) {
        this.classList.remove('drag-over', 'drag-over-left', 'drag-over-right');
        return;
    }

    // Determine drop position (before or after target)
    const rect = this.getBoundingClientRect();
    const midpoint = rect.left + rect.width / 2;
    const dropAfter = e.clientX >= midpoint;

    // Get the components array
    const components = reportState.sections[targetSectionIndex].components;

    // Remove the dragged component
    const [movedComponent] = components.splice(draggedCompIndex, 1);

    // Calculate new index
    let newIndex = targetCompIndex;
    if (draggedCompIndex < targetCompIndex) {
        // Moving forward: target index shifts down after removal
        newIndex = dropAfter ? targetCompIndex : targetCompIndex - 1;
    } else {
        // Moving backward: target index stays same
        newIndex = dropAfter ? targetCompIndex + 1 : targetCompIndex;
    }

    // Insert at new position
    components.splice(newIndex, 0, movedComponent);

    markDirty();
    renderSections();
}

/**
 * Render the empty state with templates
 */
function renderEmptyState() {
    const emptyEl = document.createElement('div');
    emptyEl.className = 'sections-empty-state';
    emptyEl.innerHTML = `
        <h3 class="empty-state-title">Start Building Your Report</h3>
        <p class="empty-state-description">
            Reports are organized into <strong>sections</strong>, each containing one or more
            <strong>components</strong> (charts, tables, or KPI cards).
        </p>

        <div class="empty-state-actions">
            <button class="btn-primary empty-state-btn" id="btn-empty-add-section">
                <span class="btn-icon">+</span> Add Empty Section
            </button>
        </div>

    `;

    // Wire up event handlers
    setTimeout(() => {
        const addBtn = emptyEl.querySelector('#btn-empty-add-section');
        if (addBtn) {
            addBtn.addEventListener('click', addSection);
        }

    }, 0);

    return emptyEl;
}

/**
 * Render all sections
 */
export function renderSections() {
    elements.sectionsContainer.innerHTML = '';

    if (reportState.sections.length === 0) {
        elements.sectionsContainer.appendChild(renderEmptyState());
        return;
    }

    reportState.sections.forEach((section, sectionIndex) => {
        // Create section preview card
        const sectionEl = document.createElement('div');
        sectionEl.className = 'section-preview';

        // Make section draggable
        sectionEl.draggable = true;
        sectionEl.dataset.sectionIndex = sectionIndex;

        // Section drag events
        sectionEl.addEventListener('dragstart', handleSectionDragStart);
        sectionEl.addEventListener('dragend', handleSectionDragEnd);
        sectionEl.addEventListener('dragover', handleSectionDragOver);
        sectionEl.addEventListener('dragenter', handleSectionDragEnter);
        sectionEl.addEventListener('dragleave', handleSectionDragLeave);
        sectionEl.addEventListener('drop', handleSectionDrop);

        // Section header with title and action buttons
        const headerEl = document.createElement('div');
        headerEl.className = 'section-preview-header';

        // Section drag handle
        const dragHandle = document.createElement('div');
        dragHandle.className = 'section-drag-handle';
        dragHandle.title = 'Drag to reorder section';
        dragHandle.innerHTML = `
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                <circle cx="5" cy="5" r="2"></circle>
                <circle cx="12" cy="5" r="2"></circle>
                <circle cx="5" cy="12" r="2"></circle>
                <circle cx="12" cy="12" r="2"></circle>
                <circle cx="5" cy="19" r="2"></circle>
                <circle cx="12" cy="19" r="2"></circle>
            </svg>
        `;
        headerEl.appendChild(dragHandle);

        const headerLeft = document.createElement('div');
        headerLeft.className = 'section-preview-header-left';

        const titleInput = document.createElement('input');
        titleInput.type = 'text';
        titleInput.className = 'section-title-editable';
        titleInput.value = section.title;
        titleInput.placeholder = 'Section title';
        titleInput.addEventListener('change', () => {
            reportState.sections[sectionIndex].title = titleInput.value;
        });
        // Prevent drag when interacting with inputs
        titleInput.addEventListener('mousedown', (e) => e.stopPropagation());

        headerLeft.appendChild(titleInput);

        const descInput = document.createElement('input');
        descInput.type = 'text';
        descInput.className = 'section-description-editable';
        descInput.value = section.description || '';
        descInput.placeholder = 'Section description (optional, supports HTML)';
        descInput.addEventListener('change', () => {
            reportState.sections[sectionIndex].description = descInput.value;
        });
        // Prevent drag when interacting with inputs
        descInput.addEventListener('mousedown', (e) => e.stopPropagation());

        headerLeft.appendChild(descInput);

        const layoutSelect = document.createElement('select');
        layoutSelect.className = 'section-layout-select';
        layoutSelect.title = 'Section layout';
        layoutSelect.innerHTML = `
            <option value="auto">Auto layout</option>
            <option value="single">Single column</option>
            <option value="two-column">Two columns</option>
            <option value="three-column">Three columns</option>
            <option value="four-column">Four columns</option>
        `;
        const autoLayout = calculateLayout(section.components.length);
        layoutSelect.value = section._userLayout ? (section.layout || 'single') : 'auto';
        layoutSelect.addEventListener('change', () => {
            if (layoutSelect.value === 'auto') {
                delete reportState.sections[sectionIndex]._userLayout;
                reportState.sections[sectionIndex].layout = calculateLayout(reportState.sections[sectionIndex].components.length);
            } else {
                reportState.sections[sectionIndex]._userLayout = true;
                reportState.sections[sectionIndex].layout = layoutSelect.value;
            }
            markDirty();
            renderSections();
            onUpdatePreview();
        });
        layoutSelect.addEventListener('mousedown', (e) => e.stopPropagation());

        headerLeft.appendChild(layoutSelect);

        // Section actions - "+" and trash buttons
        const actionsEl = document.createElement('div');
        actionsEl.className = 'section-header-actions';

        // Add component button
        const addBtn = document.createElement('button');
        addBtn.className = 'btn-add-component';
        addBtn.textContent = '+';
        addBtn.title = 'Add component';
        addBtn.addEventListener('click', () => addComponent(sectionIndex));
        actionsEl.appendChild(addBtn);

        // Delete section button
        const deleteBtn = document.createElement('button');
        deleteBtn.className = 'btn-delete-section';
        deleteBtn.innerHTML = `
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
            </svg>
        `;
        deleteBtn.title = 'Delete section';
        deleteBtn.addEventListener('click', () => removeSection(sectionIndex));
        actionsEl.appendChild(deleteBtn);

        headerEl.appendChild(headerLeft);
        headerEl.appendChild(actionsEl);

        sectionEl.appendChild(headerEl);

        // Auto-calculate layout only if user hasn't explicitly chosen one
        if (!section._userLayout) {
            const autoLayout = calculateLayout(section.components.length);
            reportState.sections[sectionIndex].layout = autoLayout;
        }
        const currentLayout = reportState.sections[sectionIndex].layout || 'single';

        // Components grid
        const gridEl = document.createElement('div');
        gridEl.className = `section-components-grid layout-${currentLayout}`;

        // Render component cards
        section.components.forEach((component, compIndex) => {
            const cardEl = renderComponentCard(component, sectionIndex, compIndex);
            gridEl.appendChild(cardEl);
        });

        sectionEl.appendChild(gridEl);
        elements.sectionsContainer.appendChild(sectionEl);
    });

    // Add the "Add Section" button
    const addSectionBtn = document.createElement('button');
    addSectionBtn.className = 'btn-secondary';
    addSectionBtn.style.marginTop = '15px';
    addSectionBtn.textContent = 'Add Section';
    addSectionBtn.addEventListener('click', addSection);
    elements.sectionsContainer.appendChild(addSectionBtn);
}

// ========================================
// Section Drag and Drop Handling
// ========================================

let draggedSection = null;
let draggedSectionIdx = null;

function handleSectionDragStart(e) {
    // Only drag if starting from the section itself or drag handle, not from component cards
    if (e.target.closest('.component-card-preview')) {
        e.preventDefault();
        return;
    }

    draggedSection = this;
    draggedSectionIdx = parseInt(this.dataset.sectionIndex);

    this.classList.add('section-dragging');

    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', `section:${draggedSectionIdx}`);

    setTimeout(() => {
        this.style.opacity = '0.4';
    }, 0);
}

function handleSectionDragEnd(e) {
    this.classList.remove('section-dragging');
    this.style.opacity = '1';

    // Remove all section drag-over states
    document.querySelectorAll('.section-preview').forEach(section => {
        section.classList.remove('section-drag-over', 'section-drag-over-top', 'section-drag-over-bottom');
    });

    draggedSection = null;
    draggedSectionIdx = null;
}

function handleSectionDragOver(e) {
    // Ignore if this is a component drag
    if (draggedSection === null) return;

    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';

    // Determine if dropping above or below
    const rect = this.getBoundingClientRect();
    const midpoint = rect.top + rect.height / 2;

    this.classList.remove('section-drag-over-top', 'section-drag-over-bottom');
    if (e.clientY < midpoint) {
        this.classList.add('section-drag-over-top');
    } else {
        this.classList.add('section-drag-over-bottom');
    }
}

function handleSectionDragEnter(e) {
    // Ignore if this is a component drag
    if (draggedSection === null) return;

    e.preventDefault();
    if (this !== draggedSection) {
        this.classList.add('section-drag-over');
    }
}

function handleSectionDragLeave(e) {
    if (!this.contains(e.relatedTarget)) {
        this.classList.remove('section-drag-over', 'section-drag-over-top', 'section-drag-over-bottom');
    }
}

function handleSectionDrop(e) {
    // Ignore if this is a component drag
    if (draggedSection === null) return;

    e.preventDefault();
    e.stopPropagation();

    if (this === draggedSection) return;

    const targetSectionIdx = parseInt(this.dataset.sectionIndex);

    // Determine drop position (before or after target)
    const rect = this.getBoundingClientRect();
    const midpoint = rect.top + rect.height / 2;
    const dropAfter = e.clientY >= midpoint;

    // Remove the dragged section
    const [movedSection] = reportState.sections.splice(draggedSectionIdx, 1);

    // Calculate new index
    let newIndex = targetSectionIdx;
    if (draggedSectionIdx < targetSectionIdx) {
        newIndex = dropAfter ? targetSectionIdx : targetSectionIdx - 1;
    } else {
        newIndex = dropAfter ? targetSectionIdx + 1 : targetSectionIdx;
    }

    // Insert at new position
    reportState.sections.splice(newIndex, 0, movedSection);

    markDirty();
    renderSections();
}
