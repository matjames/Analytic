/**
 * Tests for validation.js
 *
 * Run with: npm test
 * Watch mode: npm run test:watch
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';

// Mock the state module
vi.mock('./state.js', () => ({
    elements: {
        reportId: null,
        reportTitle: null,
        reportDescription: null,
    },
    reportState: {
        id: '',
        title: '',
        description: '',
        sections: [],
        timeColumns: {},
    },
}));

// Mock the utils module
vi.mock('./utils.js', () => ({
    showError: vi.fn(),
}));

// Import after mocking
import {
    validateReportIdField,
    validateRequiredField,
    addValidationMessage,
    validateSections,
    validateComponents,
} from './validation.js';
import { elements, reportState } from './state.js';
import { showError } from './utils.js';

// ============== Helper Functions ==============

/**
 * Create a mock input element for testing
 */
function createMockInput(value = '') {
    const parentNode = document.createElement('div');
    const input = document.createElement('input');
    input.value = value;
    parentNode.appendChild(input);
    return input;
}

// ============== validateReportIdField Tests ==============

describe('validateReportIdField', () => {
    describe('valid inputs', () => {
        it('accepts simple report name', () => {
            const input = createMockInput('my-report');
            expect(validateReportIdField(input)).toBe(true);
            expect(input.classList.contains('valid')).toBe(true);
        });

        it('accepts report with category (one slash)', () => {
            const input = createMockInput('category/report');
            expect(validateReportIdField(input)).toBe(true);
            expect(input.classList.contains('valid')).toBe(true);
        });

        it('accepts underscores', () => {
            const input = createMockInput('my_report_name');
            expect(validateReportIdField(input)).toBe(true);
        });

        it('accepts numbers', () => {
            const input = createMockInput('report-2024');
            expect(validateReportIdField(input)).toBe(true);
        });

        it('accepts hyphens', () => {
            const input = createMockInput('disease-surveillance');
            expect(validateReportIdField(input)).toBe(true);
        });

        it('accepts mixed valid characters', () => {
            const input = createMockInput('health_metrics/fever-report-2024');
            expect(validateReportIdField(input)).toBe(true);
        });

        it('returns true for empty input (not invalid, just not valid)', () => {
            const input = createMockInput('');
            expect(validateReportIdField(input)).toBe(true);
            expect(input.classList.contains('valid')).toBe(false);
            expect(input.classList.contains('invalid')).toBe(false);
        });

        it('trims whitespace', () => {
            const input = createMockInput('  my-report  ');
            expect(validateReportIdField(input)).toBe(true);
        });
    });

    describe('invalid characters', () => {
        it('rejects spaces', () => {
            const input = createMockInput('my report');
            expect(validateReportIdField(input)).toBe(false);
            expect(input.classList.contains('invalid')).toBe(true);
        });

        it('rejects special characters @', () => {
            const input = createMockInput('report@2024');
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects special characters #', () => {
            const input = createMockInput('report#1');
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects special characters $', () => {
            const input = createMockInput('report$money');
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects special characters %', () => {
            const input = createMockInput('report%20name');
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects dots', () => {
            const input = createMockInput('report.yaml');
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects parentheses', () => {
            const input = createMockInput('report(1)');
            expect(validateReportIdField(input)).toBe(false);
        });
    });

    describe('slash rules', () => {
        it('rejects multiple slashes (more than one category level)', () => {
            const input = createMockInput('a/b/c');
            expect(validateReportIdField(input)).toBe(false);
            expect(input.classList.contains('invalid')).toBe(true);
        });

        it('rejects leading slash', () => {
            const input = createMockInput('/report');
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects trailing slash', () => {
            const input = createMockInput('report/');
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects double slashes (empty segment)', () => {
            const input = createMockInput('category//report');
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects triple slashes', () => {
            const input = createMockInput('a///b');
            expect(validateReportIdField(input)).toBe(false);
        });
    });

    describe('security-related inputs', () => {
        it('rejects path traversal attempt', () => {
            const input = createMockInput('../etc/passwd');
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects SQL injection attempt', () => {
            const input = createMockInput("'; DROP TABLE--");
            expect(validateReportIdField(input)).toBe(false);
        });

        it('rejects XSS attempt', () => {
            const input = createMockInput('<script>alert(1)</script>');
            expect(validateReportIdField(input)).toBe(false);
        });
    });

    describe('validation message handling', () => {
        it('removes existing validation message before adding new one', () => {
            const input = createMockInput('invalid@id');

            // First validation adds a message
            validateReportIdField(input);
            expect(input.parentNode.querySelector('.validation-message')).not.toBeNull();

            // Second validation should replace, not add another
            validateReportIdField(input);
            const messages = input.parentNode.querySelectorAll('.validation-message');
            expect(messages.length).toBe(1);
        });

        it('removes validation message when input becomes valid', () => {
            const input = createMockInput('invalid@id');
            validateReportIdField(input);

            // Now make it valid
            input.value = 'valid-id';
            validateReportIdField(input);

            expect(input.classList.contains('valid')).toBe(true);
            expect(input.classList.contains('invalid')).toBe(false);
        });
    });
});

// ============== validateRequiredField Tests ==============

describe('validateRequiredField', () => {
    it('returns false for empty string', () => {
        const input = createMockInput('');
        expect(validateRequiredField(input, 'Title')).toBe(false);
        expect(input.classList.contains('invalid')).toBe(true);
    });

    it('returns false for whitespace only', () => {
        const input = createMockInput('   ');
        expect(validateRequiredField(input, 'Title')).toBe(false);
    });

    it('returns true for non-empty value', () => {
        const input = createMockInput('My Report Title');
        expect(validateRequiredField(input, 'Title')).toBe(true);
        expect(input.classList.contains('valid')).toBe(true);
    });

    it('adds validation message with field name', () => {
        const input = createMockInput('');
        validateRequiredField(input, 'Description');

        const message = input.parentNode.querySelector('.validation-message');
        expect(message).not.toBeNull();
        expect(message.textContent).toContain('Description');
        expect(message.textContent).toContain('required');
    });

    it('removes previous validation message', () => {
        const input = createMockInput('');
        validateRequiredField(input, 'Title');
        validateRequiredField(input, 'Title');

        const messages = input.parentNode.querySelectorAll('.validation-message');
        expect(messages.length).toBe(1);
    });
});

// ============== addValidationMessage Tests ==============

describe('addValidationMessage', () => {
    it('adds message element to parent', () => {
        const input = createMockInput('');
        addValidationMessage(input, 'This is an error');

        const message = input.parentNode.querySelector('.validation-message');
        expect(message).not.toBeNull();
        expect(message.textContent).toBe('This is an error');
    });

    it('adds correct CSS class', () => {
        const input = createMockInput('');
        addValidationMessage(input, 'Error message');

        const message = input.parentNode.querySelector('.validation-message');
        expect(message.className).toBe('validation-message');
    });

    it('can add multiple messages (if not cleared)', () => {
        const input = createMockInput('');
        addValidationMessage(input, 'Error 1');
        addValidationMessage(input, 'Error 2');

        const messages = input.parentNode.querySelectorAll('.validation-message');
        expect(messages.length).toBe(2);
    });
});

// ============== validateSections Tests ==============

describe('validateSections', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        reportState.sections = [];
    });

    it('returns false when no sections exist', () => {
        reportState.sections = [];
        expect(validateSections()).toBe(false);
        expect(showError).toHaveBeenCalledWith('Please add at least one section');
    });

    it('returns true when sections exist', () => {
        reportState.sections = [{ id: 'section-1', components: [] }];
        expect(validateSections()).toBe(true);
        expect(showError).not.toHaveBeenCalled();
    });

    it('returns true for multiple sections', () => {
        reportState.sections = [
            { id: 'section-1', components: [] },
            { id: 'section-2', components: [] },
        ];
        expect(validateSections()).toBe(true);
    });
});

// ============== validateComponents Tests ==============

describe('validateComponents', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        reportState.sections = [];
    });

    it('returns false when section has no components', () => {
        reportState.sections = [{ id: 'section-1', components: [] }];
        expect(validateComponents()).toBe(false);
        expect(showError).toHaveBeenCalled();
    });

    it('returns true when section has components with queries', () => {
        reportState.sections = [{
            id: 'section-1',
            components: [{
                type: 'bar',
                title: 'Bar Chart',
                query: { table: 'report.data', aggregations: [] },
            }],
        }];
        expect(validateComponents()).toBe(true);
    });

    it('returns false when non-text component is missing query', () => {
        reportState.sections = [{
            id: 'section-1',
            components: [{
                type: 'bar',
                title: 'Missing Query Chart',
            }],
        }];
        expect(validateComponents()).toBe(false);
        expect(showError).toHaveBeenCalled();
    });

    it('allows text components without query', () => {
        reportState.sections = [{
            id: 'section-1',
            components: [{
                type: 'text',
                title: 'Info Text',
                content: 'Some text content',
            }],
        }];
        expect(validateComponents()).toBe(true);
    });

    it('allows table_advanced with sql instead of query', () => {
        reportState.sections = [{
            id: 'section-1',
            components: [{
                type: 'table_advanced',
                title: 'Advanced Table',
                sql: 'SELECT * FROM data',
            }],
        }];
        expect(validateComponents()).toBe(true);
    });

    it('returns false when table_advanced has neither sql nor query', () => {
        reportState.sections = [{
            id: 'section-1',
            components: [{
                type: 'table_advanced',
                title: 'Broken Table',
            }],
        }];
        expect(validateComponents()).toBe(false);
    });

    it('validates all sections', () => {
        reportState.sections = [
            {
                id: 'section-1',
                components: [{ type: 'bar', query: { table: 't' } }],
            },
            {
                id: 'section-2',
                components: [], // Empty - should fail
            },
        ];
        expect(validateComponents()).toBe(false);
    });

    it('handles untitled components in error message', () => {
        reportState.sections = [{
            id: 'section-1',
            components: [{
                type: 'bar',
                // No title
            }],
        }];
        validateComponents();
        expect(showError).toHaveBeenCalledWith(expect.stringContaining('Untitled'));
    });
});

// ============== Edge Cases & Integration ==============

describe('edge cases', () => {
    it('handles unicode in report ID (should reject)', () => {
        const input = createMockInput('レポート');
        expect(validateReportIdField(input)).toBe(false);
    });

    it('handles very long report ID', () => {
        const longId = 'a'.repeat(1000);
        const input = createMockInput(longId);
        // Should be valid (no length limit in current implementation)
        expect(validateReportIdField(input)).toBe(true);
    });

    it('handles null-ish values gracefully', () => {
        const input = createMockInput('');
        input.value = null;
        // Should not throw
        expect(() => validateReportIdField(input)).not.toThrow();
    });
});
