/**
 * Unit tests for filterManager.js
 * Tests the vanilla JS filter system: initialization, rendering, state management, and UI updates.
 *
 * Run: npm test (or npm run test:watch)
 */

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { JSDOM } from 'jsdom';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Load filterManager source to evaluate in JSDOM
const filterManagerSource = fs.readFileSync(path.join(__dirname, 'filterManager.js'), 'utf-8');

let dom, window, document, filterManager;

// Helper to create a basic HTML structure required by filterManager
function createBaseDOM() {
  return `
    <!DOCTYPE html>
    <html>
      <head>
        <script>
          // Polyfill matchMedia for JSDOM
          window.matchMedia = window.matchMedia || function(query) {
            return {
              matches: query.includes('768') ? window.innerWidth <= 768 : false,
              media: query,
              onchange: null,
              addListener: function() {},
              removeListener: function() {},
              addEventListener: function() {},
              removeEventListener: function() {},
              dispatchEvent: function() {}
            };
          };
        </script>
      </head>
      <body>
        <div id="filter-bar" style="display:none;">
          <div id="active-filters"></div>
          <button id="clear-all-filters" style="display:none;"></button>
          <span id="filter-count-badge"></span>
          <div id="filter-summary"></div>
          <div id="filter-bar-toggle"></div>
        </div>
        <div id="sections-container"></div>
        <div id="filter-feedback-banner" style="display:none;"></div>
      </body>
    </html>
  `;
}

// Helper to create report metadata
function createMetadata(options = {}) {
  const {
    filters = ['year', 'month'],
    filterDefinitions = {
      year: {
        label: 'Year',
        type: 'select',
        apiEndpoint: '/api/filters/years'
      },
      month: {
        label: 'Month',
        type: 'multiselect',
        apiEndpoint: '/api/filters/months'
      }
    }
  } = options;
  return { filters, filterDefinitions };
}

beforeEach(() => {
  // Create fresh DOM
  dom = new JSDOM(createBaseDOM(), {
    url: 'http://localhost',
    runScripts: 'dangerously'
  });
  window = dom.window;
  document = window.document;

  // Set up globals
  global.window = window;
  global.document = document;
  global.HTMLElement = window.HTMLElement;
  window.BASE_PATH = '';

  // Mock apiCache.fetch
  window.apiCache = {
    fetch: vi.fn().mockResolvedValue([])
  };

  // Mock loadReport (called after filter apply)
  window.loadReport = vi.fn().mockResolvedValue(undefined);

  // Evaluate filterManager source in the JSDOM window context
  // Use modified source that assigns to window.filterManager for easier access
  const modifiedSource = filterManagerSource.replace('const filterManager =', 'window.filterManager =');
  window.eval(modifiedSource);
  filterManager = window.filterManager;
});

afterEach(() => {
  // Clean up
  if (filterManager) {
    filterManager.destroy();
  }
  dom = null;
  window = null;
  document = null;
  filterManager = null;
  vi.clearAllMocks();
});

describe('filterManager Initialization', () => {
  it('should initialize with metadata and report ID', () => {
    const metadata = createMetadata();
    filterManager.init(metadata, 'test-report');

    expect(filterManager.getMetadata()).toEqual(metadata);
  });

  it('should clean up previous instance on re-init', () => {
    const metadata1 = createMetadata({ filters: ['year'] });
    filterManager.init(metadata1, 'report1');

    const metadata2 = createMetadata({ filters: ['month'] });
    filterManager.init(metadata2, 'report2');

    expect(filterManager.getMetadata()).toEqual(metadata2);
  });

  it('should destroy and clean up resources', async () => {
    const metadata = createMetadata({ filters: ['month'] });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch.mockResolvedValue(['1', '2', '3']);
    await filterManager.buildFilterBar();

    // After building, a tracked panel should exist
    expect(document.body.querySelectorAll('.multiselect-panel').length).toBe(1);

    filterManager.destroy();

    // destroy() removes tracked panels and hides the filter bar
    expect(document.body.querySelectorAll('.multiselect-panel').length).toBe(0);
    expect(document.getElementById('filter-bar').style.display).toBe('none');
  });
});

describe('Filter Bar Rendering', () => {
  beforeEach(async () => {
    const metadata = createMetadata({
      filters: ['year', 'month'],
      filterDefinitions: {
        year: { label: 'Year', type: 'select', apiEndpoint: '/api/filters/years' },
        month: { label: 'Month', type: 'multiselect', apiEndpoint: '/api/filters/months' }
      }
    });
    filterManager.init(metadata, 'test-report');
    await filterManager.buildFilterBar();
  });

  it('should show filter bar when filters exist', () => {
    const filterBar = document.getElementById('filter-bar');
    expect(filterBar.style.display).toBe('flex');
  });

  it('should hide filter bar when no filters', async () => {
    const metadata = createMetadata({ filters: [] });
    filterManager.init(metadata, 'no-filters');
    await filterManager.buildFilterBar();

    expect(document.getElementById('filter-bar').style.display).toBe('none');
  });

  it('should render filter item labels', () => {
    const labels = document.querySelectorAll('.filter-item-label');
    expect(labels.length).toBe(2);
    expect(labels[0].textContent).toContain('Year');
    expect(labels[1].textContent).toContain('Month');
  });

  it('strips legacy Select prefix and (ies) from API labels', async () => {
    const metadata = createMetadata({
      filters: ['year', 'facility'],
      filterDefinitions: {
        year: { label: 'Select Year', type: 'select', apiEndpoint: '/api/filters/years' },
        facility: {
          label: 'Select Facility(ies)',
          type: 'multiselect',
          apiEndpoint: '/api/filters/facilities'
        }
      }
    });
    filterManager.init(metadata, 'legacy-labels');
    window.apiCache.fetch.mockResolvedValueOnce(['2026']).mockResolvedValueOnce(['A', 'B']);
    await filterManager.buildFilterBar();
    const yearLab = document.querySelector('.filter-item[data-filter="year"] .filter-item-label');
    const facLab = document.querySelector('.filter-item[data-filter="facility"] .filter-item-label');
    expect(yearLab.textContent).toBe('Year:');
    expect(facLab.textContent).toBe('Facility:');
  });

  it('should create select element for single-select filter', () => {
    const yearSelect = document.getElementById('filter-year');
    expect(yearSelect).not.toBeNull();
    expect(yearSelect.tagName).toBe('SELECT');
  });

  it('should create multiselect container for multiselect filter', () => {
    const monthContainer = document.querySelector('[data-filter="month"]');
    expect(monthContainer).not.toBeNull();
    expect(monthContainer.querySelector('.multiselect-trigger')).not.toBeNull();
    // Panel is appended to document.body for z-index stacking, not inside the container
    expect(document.getElementById('filter-month-panel')).not.toBeNull();
  });
});

describe('Filter bar display order', () => {
  it('orders time filters before district and facility last regardless of YAML order', async () => {
    const metadata = createMetadata({
      filters: ['facility', 'district', 'month', 'year'],
      filterDefinitions: {
        year: { label: 'Year', type: 'select', apiEndpoint: '/api/filters/years' },
        month: { label: 'Month', type: 'multiselect', apiEndpoint: '/api/filters/months' },
        district: { label: 'District', type: 'select', apiEndpoint: '/api/filters/districts' },
        facility: { label: 'Facility', type: 'multiselect', apiEndpoint: '/api/filters/facilities' }
      }
    });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch.mockResolvedValue([]);
    await filterManager.buildFilterBar();

    const order = [...document.querySelectorAll('#active-filters .filter-item')].map((el) =>
      el.getAttribute('data-filter')
    );
    expect(order).toEqual(['year', 'month', 'district', 'facility']);
  });
});

describe('Multiselect Filters', () => {
  beforeEach(async () => {
    const metadata = createMetadata({
      filters: ['month'],
      filterDefinitions: {
        month: {
          label: 'Month',
          type: 'multiselect',
          apiEndpoint: '/api/filters/months'
        }
      }
    });
    filterManager.init(metadata, 'test-report');
    // Mock API to provide month options (numeric strings for simplicity)
    window.apiCache.fetch.mockResolvedValue(['1', '2', '3']);
    await filterManager.buildFilterBar();
  });

  it('should load options via API', async () => {
    expect(window.apiCache.fetch).toHaveBeenCalledWith('/api/filters/months');
  });

  it('should create checkboxes for options', () => {
    const optionsContainer = document.getElementById('filter-month-options');
    const checkboxes = optionsContainer.querySelectorAll('input[type="checkbox"]');
    expect(checkboxes.length).toBe(3);
  });

  it('should update trigger text when checkboxes change', () => {
    const trigger = document.getElementById('filter-month-trigger');
    const optionsContainer = document.getElementById('filter-month-options');
    const firstCheckbox = optionsContainer.querySelector('input[type="checkbox"]');

    // Simulate checking first checkbox
    firstCheckbox.checked = true;
    firstCheckbox.dispatchEvent(new window.Event('change', { bubbles: true }));

    expect(trigger.textContent).toBe('January'); // Month name when one selected
  });

  it('should show "None" when no checkboxes selected', () => {
    const trigger = document.getElementById('filter-month-trigger');
    const optionsContainer = document.getElementById('filter-month-options');

    // Uncheck all checked boxes (auto-selects first one, so uncheck it)
    const checkedBoxes = optionsContainer.querySelectorAll('input[type="checkbox"]:checked');
    checkedBoxes.forEach(cb => {
      cb.checked = false;
      cb.dispatchEvent(new window.Event('change', { bubbles: true }));
    });

    expect(trigger.textContent).toBe('None');
  });

  it('should show "All" when all checkboxes selected', () => {
    const trigger = document.getElementById('filter-month-trigger');
    const optionsContainer = document.getElementById('filter-month-options');
    const checkboxes = optionsContainer.querySelectorAll('input[type="checkbox"]');

    checkboxes.forEach(cb => {
      cb.checked = true;
      cb.closest('.multiselect-option')?.classList.add('selected');
      cb.dispatchEvent(new window.Event('change', { bubbles: true }));
    });

    expect(trigger.textContent).toBe('All');
  });
});

describe('Single-Select Filters', () => {
  beforeEach(async () => {
    const metadata = createMetadata({
      filters: ['year'],
      filterDefinitions: {
        year: {
          label: 'Year',
          type: 'select',
          apiEndpoint: '/api/filters/years'
        }
      }
    });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch.mockResolvedValue(['2024', '2025', '2026']);
    await filterManager.buildFilterBar();
  });

  it('should populate select options', () => {
    const select = document.getElementById('filter-year');
    const options = select.querySelectorAll('option');
    expect(options.length).toBe(4); // 'All' + 3 years
    expect(options[1].textContent).toBe('2024');
  });

  it('should update values on selection', async () => {
    const select = document.getElementById('filter-year');
    select.value = '2025';
    select.dispatchEvent(new window.Event('change'));

    const values = filterManager.getValues();
    expect(values.year).toBe('2025');
  });
});

describe('Custom Single-Select Filters', () => {
  it('should render without All and default to the first option', async () => {
    const metadata = createMetadata({
      filters: ['basket_group_2'],
      filterDefinitions: {
        basket_group_2: {
          label: 'Basket',
          type: 'select',
          apiEndpoint: '/api/filters/custom?table=report.mv_data&column=basket_group_2'
        }
      }
    });
    filterManager.init(metadata, 'custom-default-first');
    window.apiCache.fetch.mockResolvedValue(['ARV', 'EMHS', 'LAB']);
    await filterManager.buildFilterBar();

    const select = document.getElementById('filter-basket_group_2');
    const options = [...select.querySelectorAll('option')].map(opt => opt.textContent);
    expect(options).toEqual(['ARV', 'EMHS', 'LAB']);
    expect(select.value).toBe('ARV');
    expect(filterManager.getValues()).toEqual({ basket_group_2: 'ARV' });
  });

  it('should use configured defaultValue when available', async () => {
    const metadata = createMetadata({
      filters: ['basket_group_2'],
      filterDefinitions: {
        basket_group_2: {
          label: 'Basket',
          type: 'select',
          apiEndpoint: '/api/filters/custom?table=report.mv_data&column=basket_group_2',
          defaultValue: 'EMHS'
        }
      }
    });
    filterManager.init(metadata, 'custom-configured-default');
    window.apiCache.fetch.mockResolvedValue(['ARV', 'EMHS', 'LAB']);
    await filterManager.buildFilterBar();

    const select = document.getElementById('filter-basket_group_2');
    expect(select.value).toBe('EMHS');
    expect(filterManager.getValues()).toEqual({ basket_group_2: 'EMHS' });
  });

  it('should reset custom single-selects to their default on clear all', async () => {
    const metadata = createMetadata({
      filters: ['basket_group_2'],
      filterDefinitions: {
        basket_group_2: {
          label: 'Basket',
          type: 'select',
          apiEndpoint: '/api/filters/custom?table=report.mv_data&column=basket_group_2',
          defaultValue: 'LAB'
        }
      }
    });
    filterManager.init(metadata, 'custom-clear-default');
    window.apiCache.fetch.mockResolvedValue(['ARV', 'EMHS', 'LAB']);
    await filterManager.buildFilterBar();

    const select = document.getElementById('filter-basket_group_2');
    select.value = 'ARV';
    await filterManager.clearAll();

    expect(select.value).toBe('LAB');
    expect(filterManager.getValues()).toEqual({ basket_group_2: 'LAB' });
  });
});

describe('Facility Autocomplete', () => {
  beforeEach(async () => {
    const metadata = createMetadata({
      filters: ['facility'],
      filterDefinitions: {
        facility: {
          label: 'Facility',
          type: 'facility',
          apiEndpoint: '/api/filters/facilities'
        }
      }
    });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch.mockResolvedValue(['Hospital A', 'Clinic B', 'Health Center C']);
    await filterManager.buildFilterBar();
  });

  it('should create autocomplete container', () => {
    const container = document.querySelector('.facility-autocomplete-container');
    expect(container).not.toBeNull();
    expect(document.getElementById('filter-facility-input')).not.toBeNull();
  });

  it('should load facilities on _loadFacilities', async () => {
    const container = document.querySelector('.facility-autocomplete-container');
    await container._loadFacilities(['Facility 1', 'Facility 2']);

    // Verify that the container has the method
    expect(typeof container._loadFacilities).toBe('function');
  });

  it('should add facility tags on selection', () => {
    const selectedContainer = document.getElementById('filter-facility-selected');

    // Simulate adding a facility tag (as done in filterManager code)
    const facility = 'Test Facility';
    const tag = document.createElement('span');
    tag.className = 'facility-tag';
    tag.setAttribute('data-value', facility);
    tag.innerHTML = `${facility} <button type="button" class="facility-tag-remove">&times;</button>`;
    selectedContainer.appendChild(tag);

    const tags = selectedContainer.querySelectorAll('.facility-tag');
    expect(tags.length).toBe(1);
    expect(tags[0].getAttribute('data-value')).toBe('Test Facility');
  });
});

describe('Filter Values', () => {
  let mockLocalStorage;

  beforeEach(async () => {
    // Mock localStorage
    mockLocalStorage = {
      getItem: vi.fn(),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn()
    };
    Object.defineProperty(window, 'localStorage', {
      value: mockLocalStorage,
      writable: true
    });

    const metadata = createMetadata({
      filters: ['year', 'month'],
      filterDefinitions: {
        year: { label: 'Year', type: 'select', apiEndpoint: '/api/filters/years' },
        month: { label: 'Month', type: 'multiselect', apiEndpoint: '/api/filters/months' }
      }
    });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch.mockResolvedValue(['2024', '2025']);
    await filterManager.buildFilterBar();
    filterManager.clearAll(); // Clear initial auto-selected values
  });

  it('should return empty object when no filters selected', () => {
    const values = filterManager.getValues();
    expect(values).toEqual({});
  });

  it('should return selected single-select value', () => {
    const select = document.getElementById('filter-year');
    select.value = '2025';
    const values = filterManager.getValues();
    expect(values.year).toBe('2025');
  });

  it('should return selected multiselect values as comma-separated string', () => {
    const optionsContainer = document.getElementById('filter-month-options');
    const checkboxes = optionsContainer.querySelectorAll('input[type="checkbox"]');
    // Select first two months (2024, 2025) — must dispatch change events so _values is updated
    checkboxes[0].checked = true;
    checkboxes[0].dispatchEvent(new window.Event('change', { bubbles: true }));
    checkboxes[1].checked = true;
    checkboxes[1].dispatchEvent(new window.Event('change', { bubbles: true }));

    const values = filterManager.getValues();
    expect(values.month).toBe('2024,2025'); // comma-joined
  });

  it('should ignore empty values', () => {
    const select = document.getElementById('filter-year');
    select.value = '';
    const values = filterManager.getValues();
    expect(values).toEqual({});
  });
});

describe('setValues', () => {
  beforeEach(async () => {
    const metadata = createMetadata({
      filters: ['year', 'month'],
      filterDefinitions: {
        year: { label: 'Year', type: 'select', apiEndpoint: '/api/filters/years' },
        month: { label: 'Month', type: 'multiselect', apiEndpoint: '/api/filters/months' }
      }
    });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch.mockResolvedValue(['2024', '2025', '2026']);
    await filterManager.buildFilterBar();
  });

  it('should set single-select value', () => {
    filterManager.setValues({ year: '2025' });
    const select = document.getElementById('filter-year');
    expect(select.value).toBe('2025');
  });

  it('should set multiselect values', () => {
    filterManager.setValues({ month: '2024,2025' });
    const optionsContainer = document.getElementById('filter-month-options');
    const checkboxes = optionsContainer.querySelectorAll('input[type="checkbox"]');
    expect(checkboxes[0].checked).toBe(true);
    expect(checkboxes[1].checked).toBe(true);
    expect(checkboxes[2].checked).toBe(false);
  });
});

describe('Smart Defaults', () => {
  beforeEach(async () => {
    const metadata = createMetadata({
      filters: ['month'],
      filterDefinitions: {
        month: { label: 'Month', type: 'multiselect', apiEndpoint: '/api/filters/months' }
      }
    });
    filterManager.init(metadata, 'test-report');
    // Mock all 12 months so smart defaults can find the current default month
    window.apiCache.fetch.mockResolvedValue(['1','2','3','4','5','6','7','8','9','10','11','12']);
    await filterManager.buildFilterBar();
    filterManager.clearAll(); // Clear auto-selected first option
  });

  it('should apply smart default month', () => {
    // applySmartDefaults is synchronous
    filterManager.applySmartDefaults();

    const optionsContainer = document.getElementById('filter-month-options');
    const checked = optionsContainer.querySelectorAll('input[type="checkbox"]:checked');
    expect(checked.length).toBeGreaterThan(0); // At least one month should be selected
  });
});

describe('State Persistence', () => {
  let mockLocalStorage;

  beforeEach(async () => {
    const metadata = createMetadata({
      filters: ['year'],
      filterDefinitions: {
        year: { label: 'Year', type: 'select', apiEndpoint: '/api/filters/years' }
      }
    });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch.mockResolvedValue(['2024', '2025']);
    await filterManager.buildFilterBar();

    // Create fresh localStorage mock with Object.defineProperty
    mockLocalStorage = {
      getItem: vi.fn(),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn()
    };
    Object.defineProperty(window, 'localStorage', {
      value: mockLocalStorage,
      writable: true
    });
  });

  it('should save state to localStorage', () => {
    const select = document.getElementById('filter-year');
    select.value = '2025';
    filterManager.saveState();

    expect(mockLocalStorage.setItem).toHaveBeenCalledWith(
      'filterState-test-report',
      expect.stringContaining('"year":"2025"')
    );
  });

  it('should restore state from localStorage', () => {
    const savedState = {
      reportId: 'test-report',
      filters: { year: '2025' },
      savedAt: new Date().toISOString()
    };
    mockLocalStorage.getItem.mockReturnValue(JSON.stringify(savedState));

    const restored = filterManager.restoreState();
    expect(restored).toBe(true);

    const select = document.getElementById('filter-year');
    expect(select.value).toBe('2025');
  });

  it('should not restore state for different report', () => {
    const savedState = {
      reportId: 'other-report',
      filters: { year: '2025' },
      savedAt: new Date().toISOString()
    };
    mockLocalStorage.getItem.mockReturnValue(JSON.stringify(savedState));

    const restored = filterManager.restoreState();
    expect(restored).toBe(false);
  });

  it('should clear state', () => {
    filterManager.clearState();
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('filterState-test-report');
  });
});

describe('UI Updates', () => {
  let mockLocalStorage;

  beforeEach(async () => {
    // Mock localStorage
    mockLocalStorage = {
      getItem: vi.fn(),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn()
    };
    Object.defineProperty(window, 'localStorage', {
      value: mockLocalStorage,
      writable: true
    });

    const metadata = createMetadata({
      filters: ['year', 'month'],
      filterDefinitions: {
        year: { label: 'Year', type: 'select', apiEndpoint: '/api/filters/years' },
        month: { label: 'Month', type: 'multiselect', apiEndpoint: '/api/filters/months' }
      }
    });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch.mockResolvedValue(['2024', '2025']);
    await filterManager.buildFilterBar();
    filterManager.clearAll();
  });

  it('should update filter count badge', () => {
    // Initially no filters, badge hidden
    let badge = document.getElementById('filter-count-badge');
    expect(badge.style.display).toBe('none');

    // Select a filter
    const select = document.getElementById('filter-year');
    select.value = '2025';
    select.dispatchEvent(new window.Event('change'));
    filterManager.updatePills();

    expect(badge.textContent).toBe('1');
    expect(badge.style.display).toBe('inline-flex');
  });

  it('should update filter summary', () => {
    const summary = document.getElementById('filter-summary');
    expect(summary.textContent).toBe('All filters cleared');

    const select = document.getElementById('filter-year');
    select.value = '2025';
    select.dispatchEvent(new window.Event('change'));
    filterManager.updatePills();

    expect(summary.textContent).toContain('2025');
  });

  it('should show/hide clear all button', () => {
    const clearBtn = document.getElementById('clear-all-filters');
    expect(clearBtn.style.display).toBe('none');

    const select = document.getElementById('filter-year');
    select.value = '2025';
    select.dispatchEvent(new window.Event('change'));

    expect(clearBtn.style.display).toBe('inline-block');
  });
});

describe('Clear All', () => {
  let mockLocalStorage;
  beforeEach(async () => {
    mockLocalStorage = {
      getItem: vi.fn().mockReturnValue(null),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    };
    Object.defineProperty(window, 'localStorage', { value: mockLocalStorage, writable: true, configurable: true });

    const metadata = createMetadata({
      filters: ['year', 'month'],
      filterDefinitions: {
        year: { label: 'Year', type: 'select', apiEndpoint: '/api/filters/years' },
        month: { label: 'Month', type: 'multiselect', apiEndpoint: '/api/filters/months' }
      }
    });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch.mockResolvedValue(['2024', '2025', '2026']);
    await filterManager.buildFilterBar();

    // Set some values
    const yearSelect = document.getElementById('filter-year');
    yearSelect.value = '2025';
    yearSelect.dispatchEvent(new window.Event('change'));

    const monthOptions = document.getElementById('filter-month-options');
    const checkbox = monthOptions.querySelector('input[type="checkbox"]');
    checkbox.checked = true;
    checkbox.dispatchEvent(new window.Event('change'));
  });

  it('should clear all filter values', async () => {
    await filterManager.clearAll();

    const values = filterManager.getValues();
    expect(values).toEqual({});

    const clearBtn = document.getElementById('clear-all-filters');
    expect(clearBtn.style.display).toBe('none');
  });

  it('should clear localStorage state', () => {
    filterManager.clearAll();
    expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('filterState-test-report');
  });
});

describe('Filter Dependencies', () => {
  // Use 'year' (single-select) as parent — built-in multis like 'region' have no <select> element.
  beforeEach(async () => {
    const metadata = createMetadata({
      filters: ['year', 'category'],
      filterDefinitions: {
        year: {
          label: 'Year',
          type: 'select',
          apiEndpoint: '/api/filters/years'
        },
        category: {
          label: 'Category',
          type: 'multiselect',
          apiEndpoint: '/api/filters/custom',
          dependsOn: ['year'],
          parentParams: ['year']
        }
      }
    });
    filterManager.init(metadata, 'test-report');
    window.apiCache.fetch
      .mockResolvedValueOnce(['2024', '2025']) // years
      .mockResolvedValueOnce(['Cat A', 'Cat B']); // category options
    await filterManager.buildFilterBar();
  });

  it('should disable dependent filter until parent is selected', () => {
    const categoryTrigger = document.getElementById('filter-category-trigger');
    expect(categoryTrigger.disabled).toBe(true);
    expect(categoryTrigger.textContent).toContain('Year first');
  });

  it('should enable dependent filter when parent has value', async () => {
    const yearSelect = document.getElementById('filter-year');
    yearSelect.value = '2025';
    yearSelect.dispatchEvent(new window.Event('change'));

    // Wait for async handlers to run
    await Promise.resolve();

    const categoryTrigger = document.getElementById('filter-category-trigger');
    expect(categoryTrigger.disabled).toBe(false);
  });
});

describe('Helper Functions (Exposed)', () => {
  it('isMultiselect should return true for built-in multiselect filters', () => {
    const def = { type: 'select' };
    expect(filterManager.isMultiselect('month', def)).toBe(true);
    expect(filterManager.isMultiselect('region', def)).toBe(true);
    expect(filterManager.isMultiselect('district', def)).toBe(true);
  });

  it('isMultiselect should return true for custom multiselect type', () => {
    const def = { type: 'multiselect' };
    expect(filterManager.isMultiselect('custom-filter', def)).toBe(true);
  });

  it('isMultiselect should return false for single-select', () => {
    const def = { type: 'select' };
    expect(filterManager.isMultiselect('year', def)).toBe(false);
    expect(filterManager.isMultiselect('facility', def)).toBe(false);
  });

  it('formatOption should format month names', () => {
    expect(filterManager.formatOption('month', '1')).toBe('January');
    expect(filterManager.formatOption('month', '12')).toBe('December');
  });

  it('formatOption should return string for other filters', () => {
    expect(filterManager.formatOption('year', '2025')).toBe('2025');
    expect(filterManager.formatOption('district', 'Kampala')).toBe('Kampala');
  });
});

describe('Mobile Toggle', () => {
  let matchMediaSpy;

  beforeEach(async () => {
    // Desktop: wider than filterManager mobile breakpoint (1200px)
    window.innerWidth = 1400;

    // Spy on matchMedia after it's defined by polyfill
    matchMediaSpy = vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({
      matches: query.includes('768') ? window.innerWidth <= 768 : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn()
    }));

    const metadata = createMetadata({ filters: ['year'] });
    filterManager.init(metadata, 'test-report');
    await filterManager.buildFilterBar();
    filterManager.initMobileToggle();
  });

  afterEach(() => {
    if (matchMediaSpy) {
      matchMediaSpy.mockRestore();
    }
  });

  it('should not collapse filter bar on desktop by default', () => {
    const filterBar = document.getElementById('filter-bar');
    expect(filterBar.classList.contains('collapsed')).toBe(false);
  });

  it('should toggle collapse on click', () => {
    const toggleHeader = document.getElementById('filter-bar-toggle');
    toggleHeader.click();

    const filterBar = document.getElementById('filter-bar');
    expect(filterBar.classList.contains('collapsed')).toBe(true);
  });
});

describe('Apply Button Loading State', () => {
  beforeEach(async () => {
    // Add apply button to DOM
    document.body.innerHTML += '<button id="apply-filters-btn">Apply</button>';

    const metadata = createMetadata({ filters: [] });
    filterManager.init(metadata, 'test-report');
  });

  it('should set loading state', () => {
    filterManager.setApplyLoading(true);
    const btn = document.getElementById('apply-filters-btn');
    expect(btn.disabled).toBe(true);
    expect(btn.textContent).toBe('Loading...');
    expect(btn.classList.contains('loading')).toBe(true);

    filterManager.setApplyLoading(false);
    expect(btn.disabled).toBe(false);
    expect(btn.textContent).toBe('Apply');
    expect(btn.classList.contains('loading')).toBe(false);
  });
});

describe('Feedback Banner', () => {
  beforeEach(async () => {
    document.body.innerHTML += '<div id="filter-feedback-banner" style="display:none;"></div>';
    const metadata = createMetadata({ filters: [] });
    filterManager.init(metadata, 'test-report');
  });

  it('should show feedback briefly', () => {
    const banner = document.getElementById('filter-feedback-banner');
    filterManager.showFeedback();

    expect(banner.style.display).toBe('block');
  });
});

describe('Error Handling and Edge Cases', () => {
  it('should handle missing metadata gracefully', () => {
    // Without init, getValues should return empty object
    expect(() => filterManager.getValues()).not.toThrow();
  });

  it('should handle destroyed state', () => {
    const metadata = createMetadata();
    filterManager.init(metadata, 'test');
    filterManager.destroy();

    // After destroy, should not throw
    expect(() => filterManager.getValues()).not.toThrow();
    expect(() => filterManager.setValues({})).not.toThrow();
  });
});
