// Validation Module
// Form validation functions

import { elements, reportState } from './state.js';
import { showError } from './utils.js';

/**
 * Validate the report ID field
 */
export function validateReportIdField(input) {
    const value = input.value.trim();
    const existingMessage = input.parentNode.querySelector('.validation-message');
    if (existingMessage) existingMessage.remove();

    if (!value) {
        input.classList.remove('valid', 'invalid');
        return true; // Empty is not invalid yet, just not valid
    }

    // Check for invalid characters
    if (!/^[a-zA-Z0-9_\-/]+$/.test(value)) {
        input.classList.remove('valid');
        input.classList.add('invalid');
        addValidationMessage(input, 'Only letters, numbers, hyphens, underscores, and one slash allowed');
        return false;
    }

    // Check for max one slash
    if (value.split('/').length > 2) {
        input.classList.remove('valid');
        input.classList.add('invalid');
        addValidationMessage(input, 'Maximum one slash allowed (category/report)');
        return false;
    }

    // Check for leading/trailing slashes
    if (value.startsWith('/') || value.endsWith('/')) {
        input.classList.remove('valid');
        input.classList.add('invalid');
        addValidationMessage(input, 'Cannot start or end with a slash');
        return false;
    }

    // Check for empty segments
    if (value.includes('//')) {
        input.classList.remove('valid');
        input.classList.add('invalid');
        addValidationMessage(input, 'Cannot have empty segments (double slashes)');
        return false;
    }

    // Valid!
    input.classList.remove('invalid');
    input.classList.add('valid');
    return true;
}

/**
 * Validate a required field
 */
export function validateRequiredField(input, fieldName) {
    const value = input.value.trim();
    const existingMessage = input.parentNode.querySelector('.validation-message');
    if (existingMessage) existingMessage.remove();

    if (!value) {
        input.classList.remove('valid');
        input.classList.add('invalid');
        addValidationMessage(input, `${fieldName} is required`);
        return false;
    }

    input.classList.remove('invalid');
    input.classList.add('valid');
    return true;
}

/**
 * Add validation message below input
 */
export function addValidationMessage(input, message) {
    const msgEl = document.createElement('div');
    msgEl.className = 'validation-message';
    msgEl.textContent = message;
    input.parentNode.appendChild(msgEl);
}

/**
 * Validate report info (ID, title, description)
 */
export function validateReportInfo() {
    const id = elements.reportId.value.trim();
    const title = elements.reportTitle.value.trim();
    const description = elements.reportDescription.value.trim();

    // Run all validations
    const idValid = validateReportIdField(elements.reportId);
    const titleValid = validateRequiredField(elements.reportTitle, 'Title');
    const descValid = validateRequiredField(elements.reportDescription, 'Description');

    if (!id) {
        showError('Report ID is required');
        elements.reportId.focus();
        return false;
    }

    if (!idValid) {
        showError('Report ID is invalid');
        elements.reportId.focus();
        return false;
    }

    if (!title || !titleValid) {
        showError('Title is required');
        elements.reportTitle.focus();
        return false;
    }

    // Description is optional - no validation required

    reportState.id = id;
    reportState.title = title;
    reportState.description = description || '';

    // Capture timeColumns
    const yearCol = document.getElementById('time-column-year')?.value?.trim() || '';
    const monthCol = document.getElementById('time-column-month')?.value?.trim() || '';
    const weekCol = document.getElementById('time-column-week')?.value?.trim() || '';
    const quarterCol = document.getElementById('time-column-quarter')?.value?.trim() || '';

    reportState.timeColumns = {
        year: yearCol || undefined,
        month: monthCol || undefined,
        week: weekCol || undefined,
        quarter: quarterCol || undefined,
    };

    return true;
}

/**
 * Validate that sections exist
 */
export function validateSections() {
    if (reportState.sections.length === 0) {
        showError('Please add at least one section');
        return false;
    }
    return true;
}

/**
 * Validate that components have required data
 */
export function validateComponents() {
    for (const section of reportState.sections) {
        if (section.components.length === 0) {
            showError(`Section "${section.id}" has no components`);
            return false;
        }
        for (const component of section.components) {
            // text and infobox components don't need query/sql
            // kpi uses sql directly (same as table_advanced)
            // table_advanced uses sql instead of query
            const hasQuery = component.query
                || (component.type === 'kpi' && component.sql)
                || (component.type === 'table_advanced' && component.sql)
                || (['bar', 'line', 'bar_line', 'pie', 'pyramid', 'choropleth'].includes(component.type) && component.sql);
            if (!hasQuery && component.type !== 'text' && component.type !== 'infobox') {
                showError(`Component "${component.title || 'Untitled'}" is missing a query`);
                return false;
            }
        }
    }
    return true;
}
