// Preview Tab Module
// Handles rendering the report preview with real data in the builder

import { reportState, API_BASE } from "./state.js";
import { authPostJSON } from "../authFetch.js";
import { escapeHtml } from "./utils.js";

// Track current filter values for preview
let previewFilters = {};

// Callback to switch tabs (set by main.js)
let showTabCallback = null;
let showComponentModalCallback = null;

/**
 * Set callbacks from main.js
 */
export function setPreviewCallbacks(callbacks) {
  showTabCallback = callbacks.showTab;
  showComponentModalCallback = callbacks.showComponentModal;
}

/**
 * Initialize the preview tab
 * Called when switching to the preview tab
 */
export async function initPreviewTab() {
  const emptyState = document.getElementById("preview-empty-state");
  const renderArea = document.getElementById("preview-render-area");
  const loadingState = document.getElementById("preview-loading");
  const filtersBar = document.getElementById("preview-filters-bar");

  // Hide all states first
  emptyState?.classList.add("hidden");
  renderArea?.classList.add("hidden");
  loadingState?.classList.add("hidden");

  // Check if there are components to preview
  const hasComponents =
    reportState.sections &&
    reportState.sections.length > 0 &&
    reportState.sections.some((s) => s.components && s.components.length > 0);

  console.log(
    "Preview tab init - sections:",
    reportState.sections?.length,
    "hasComponents:",
    hasComponents,
  );

  if (!hasComponents) {
    // Show empty state
    emptyState?.classList.remove("hidden");
    if (filtersBar) filtersBar.innerHTML = "";
    return;
  }

  // Show loading state while filters load
  loadingState?.classList.remove("hidden");
  resetLoadingSteps();
  setLoadingStatus("Loading filter options...");

  // Render filter controls and wait for options to load
  await renderFilterControls();
  setLoadingStatus("Filters ready");

  // Set default filter values (now that options are loaded)
  setDefaultFilters();

  // Now refresh preview with filters properly set
  refreshPreview();
}

/**
 * Render filter controls based on report configuration
 * Returns a Promise that resolves when all filter options are loaded
 */
async function renderFilterControls() {
  const filtersBar = document.getElementById("preview-filters-bar");
  if (!filtersBar) return;

  filtersBar.innerHTML = "";

  // Get filter types from report state
  const filters = getPreviewFilterNames();
  if (filters.length === 0) return;

  // Determine primary table for filter options
  const primaryTable = findPrimaryTable();

  // Create filter dropdowns and collect load promises
  const loadPromises = [];
  filters.forEach((filterName) => {
    const { filterGroup, loadPromise } = createFilterGroup(
      filterName,
      primaryTable,
    );
    if (filterGroup) {
      filtersBar.appendChild(filterGroup);
    }
    if (loadPromise) {
      loadPromises.push(loadPromise);
    }
  });

  // Wait for all filter options to load
  await Promise.all(loadPromises);
}

function getPreviewFilterNames() {
  const filters = [...(reportState.filters || [])];
  for (const customFilter of reportState.customFilters || []) {
    if (customFilter.column && !filters.includes(customFilter.column)) {
      filters.push(customFilter.column);
    }
  }
  return filters;
}

/**
 * Find the primary table used in components.
 * Mirrors the server-side getPrimaryTable logic:
 * counts table occurrences from both query.table and raw SQL FROM clauses,
 * returns the most-used table (alphabetical tiebreaker).
 */
function findPrimaryTable() {
  const tableCounts = {};

  for (const section of reportState.sections) {
    for (const component of section.components || []) {
      if (component.query?.table) {
        const t = component.query.table;
        tableCounts[t] = (tableCounts[t] || 0) + 1;
      } else if (component.sql) {
        // Extract first FROM <schema.table> or FROM <table> from raw SQL
        const match = component.sql.match(
          /\bFROM\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)/i,
        );
        if (match) {
          const t = match[1].includes(".") ? match[1] : `report.${match[1]}`;
          tableCounts[t] = (tableCounts[t] || 0) + 1;
        }
      }
    }
  }

  let bestTable = "report.cht_form_097b"; // Default fallback
  let bestCount = 0;
  for (const [table, count] of Object.entries(tableCounts)) {
    if (count > bestCount || (count === bestCount && table < bestTable)) {
      bestTable = table;
      bestCount = count;
    }
  }
  return bestTable;
}

/**
 * Create a filter dropdown group
 * Returns both the filter group element and a promise for loading options
 */
function createFilterGroup(filterName, primaryTable) {
  const group = document.createElement("div");
  group.className = "preview-filter-group";

  const label = document.createElement("label");
  label.textContent = getFilterLabel(filterName);
  group.appendChild(label);

  const select = document.createElement("select");
  select.id = `preview-filter-${filterName}`;
  select.innerHTML = '<option value="">Loading...</option>';

  select.addEventListener("change", (e) => {
    previewFilters[filterName] = e.target.value;
    // Don't auto-refresh on every change - user clicks refresh button
  });

  group.appendChild(select);

  // Load filter options and return the promise
  const loadPromise = loadFilterOptions(filterName, primaryTable, select);

  return { filterGroup: group, loadPromise };
}

/**
 * Get display label for a filter
 */
function getFilterLabel(filterName) {
  const customFilter = reportState.customFilters?.find(
    (cf) => cf.column === filterName,
  );
  if (customFilter?.label) {
    return customFilter.label;
  }

  const labels = {
    year: "Year",
    month: "Month",
    week: "Week",
    quarter: "Quarter",
    district: "District",
    region: "Region",
    facility: "Facility",
    facility_select: "Facility",
  };
  return (
    labels[filterName] ||
    filterName.charAt(0).toUpperCase() + filterName.slice(1)
  );
}

/**
 * Load filter options from API
 */
async function loadFilterOptions(filterName, primaryTable, selectElement) {
  try {
    let endpoint;
    const customFilter = reportState.customFilters?.find(
      (cf) => cf.column === filterName,
    );
    switch (filterName) {
      case "year":
        endpoint = `${API_BASE}/filters/years?table=${encodeURIComponent(primaryTable)}`;
        break;
      case "month":
        endpoint = `${API_BASE}/filters/months`;
        break;
      case "week":
        endpoint = `${API_BASE}/filters/weeks?table=${encodeURIComponent(primaryTable)}`;
        break;
      case "quarter":
        endpoint = `${API_BASE}/filters/quarters`;
        break;
      case "district":
        endpoint = `${API_BASE}/filters/districts?table=${encodeURIComponent(primaryTable)}`;
        break;
      case "region":
        endpoint = `${API_BASE}/filters/regions?table=${encodeURIComponent(primaryTable)}`;
        break;
      case "facility":
        endpoint = `${API_BASE}/filters/facilities?table=${encodeURIComponent(primaryTable)}`;
        break;
      case "facility_select":
        endpoint = `${API_BASE}/filters/facilities?table=${encodeURIComponent(primaryTable)}${reportState.id ? "&report=" + encodeURIComponent(reportState.id) : ""}`;
        break;
      default:
        // Custom filter
        if (customFilter) {
          endpoint = `${API_BASE}/filters/custom?table=${encodeURIComponent(customFilter.table)}&column=${encodeURIComponent(filterName)}`;
        } else {
          selectElement.innerHTML = '<option value="">N/A</option>';
          return;
        }
    }

    const response = await fetch(endpoint);
    const data = await response.json();

    // Handle different response formats
    let options = [];
    if (Array.isArray(data)) {
      options = data;
    } else if (data.values) {
      options = data.values;
    }

    // Build options HTML
    const isCustomSingleSelect = customFilter?.type === "select";
    let optionsHtml = isCustomSingleSelect
      ? ""
      : '<option value="">All</option>';
    options.forEach((opt) => {
      const value = typeof opt === "object" ? opt.value || opt : opt;
      const display = formatFilterOption(filterName, value);
      optionsHtml += `<option value="${escapeHtml(String(value))}">${escapeHtml(String(display))}</option>`;
    });

    selectElement.innerHTML =
      optionsHtml || '<option value="">No values</option>';

    // Set default value if we have one
    if (previewFilters[filterName]) {
      selectElement.value = previewFilters[filterName];
    } else if (isCustomSingleSelect && options.length > 0) {
      const optionValues = options.map((opt) =>
        String(typeof opt === "object" ? opt.value || opt : opt),
      );
      const configuredDefault = customFilter.defaultValue
        ? String(customFilter.defaultValue)
        : "";
      const defaultValue =
        configuredDefault && optionValues.includes(configuredDefault)
          ? configuredDefault
          : optionValues[0];
      selectElement.value = defaultValue;
      previewFilters[filterName] = defaultValue;
    }
  } catch (error) {
    console.error(`Error loading ${filterName} options:`, error);
    selectElement.innerHTML = '<option value="">Error loading</option>';
  }
}

/**
 * Format a filter option for display
 */
function formatFilterOption(filterName, value) {
  if (filterName === "month") {
    const months = [
      "",
      "January",
      "February",
      "March",
      "April",
      "May",
      "June",
      "July",
      "August",
      "September",
      "October",
      "November",
      "December",
    ];
    return months[parseInt(value)] || value;
  }
  return String(value);
}

/**
 * Set default filter values
 * Called after filter options have been loaded
 *
 * Time filters (year, month, week, quarter) default to "All" to show all available data.
 * This matches the behavior when viewing reports from configs directly.
 * Users can manually select a specific period if needed.
 */
function setDefaultFilters() {
  // Clear any previously set preview filters
  previewFilters = {};

  // Time filters default to "All" (empty value) - don't auto-set
  // This ensures the preview shows data regardless of what periods exist in the database

  // Only set non-time filters if needed in the future
  // For now, all filters default to "All"
}

/**
 * Detect default filters based on most recent data
 * @deprecated Use setDefaultFilters instead - kept for backward compatibility
 */
export async function detectDefaultFilters() {
  setDefaultFilters();
}

/**
 * Calculate ISO week number
 */
function getISOWeek(date) {
  const d = new Date(date.getTime());
  d.setHours(0, 0, 0, 0);
  d.setDate(d.getDate() + 3 - ((d.getDay() + 6) % 7));
  const week1 = new Date(d.getFullYear(), 0, 4);
  return (
    1 +
    Math.round(
      ((d.getTime() - week1.getTime()) / 86400000 -
        3 +
        ((week1.getDay() + 6) % 7)) /
        7,
    )
  );
}

/**
 * Refresh the preview by executing the report
 */
export async function refreshPreview() {
  const renderArea = document.getElementById("preview-render-area");
  const loadingState = document.getElementById("preview-loading");
  const emptyState = document.getElementById("preview-empty-state");

  // Check if there are components
  const hasComponents =
    reportState.sections &&
    reportState.sections.length > 0 &&
    reportState.sections.some((s) => s.components && s.components.length > 0);

  console.log(
    "refreshPreview - hasComponents:",
    hasComponents,
    "sections:",
    reportState.sections?.length,
  );

  if (!hasComponents) {
    emptyState?.classList.remove("hidden");
    renderArea?.classList.add("hidden");
    loadingState?.classList.add("hidden");
    return;
  }

  // Show loading state
  emptyState?.classList.add("hidden");
  loadingState?.classList.remove("hidden");
  renderArea?.classList.add("hidden");

  // Summarise what we're about to do
  const compCount = reportState.sections.reduce(
    (n, s) => n + (s.components?.length || 0),
    0,
  );
  const sectionCount = reportState.sections.length;
  resetLoadingSteps();
  addLoadingStep(
    "Building report from " +
      sectionCount +
      " section" +
      (sectionCount !== 1 ? "s" : "") +
      " (" +
      compCount +
      " component" +
      (compCount !== 1 ? "s" : "") +
      ")",
  );
  setLoadingStatus("Preparing preview...");

  // Get current filter values from select elements
  const currentFilters = {};
  const activeFilters = [];
  for (const filterName of getPreviewFilterNames()) {
    const select = document.getElementById(`preview-filter-${filterName}`);
    if (select && select.value) {
      currentFilters[filterName] = select.value;
      activeFilters.push(getFilterLabel(filterName) + ": " + select.value);
    }
  }
  if (activeFilters.length > 0) {
    addLoadingStep("Applying filters \u2014 " + activeFilters.join(", "));
  }

  // Build report object from state
  const report = buildReportFromState();

  addLoadingStep("Generating SQL and querying database...");
  setLoadingStatus("Running queries...");

  try {
    const result = await authPostJSON(`${API_BASE}/report/builder-preview`, {
      report: report,
      filters: currentFilters,
    });

    addLoadingStep("Validating design and checking warnings...");
    setLoadingStatus("Rendering results...");

    // Hide loading state
    loadingState?.classList.add("hidden");

    if (result.error && !result.report) {
      // Fatal error - show in render area
      renderArea?.classList.remove("hidden");
      renderArea.innerHTML = `
                <div class="component-error">
                    <div class="component-error-header">
                        <span class="component-error-title">
                            <span class="component-error-badge">Error</span>
                            Preview Failed
                        </span>
                    </div>
                    <div class="component-error-message">${escapeHtml(result.error)}</div>
                </div>
            `;
      return;
    }

    // Render the report with any component errors
    renderArea?.classList.remove("hidden");
    renderPreviewContent(
      result.report,
      result.errors || [],
      result.warnings || [],
    );
  } catch (error) {
    console.error("Preview error:", error);
    loadingState?.classList.add("hidden");
    renderArea?.classList.remove("hidden");
    renderArea.innerHTML = `
            <div class="component-error">
                <div class="component-error-header">
                    <span class="component-error-title">
                        <span class="component-error-badge">Error</span>
                        Network Error
                    </span>
                </div>
                <div class="component-error-message">${escapeHtml(error.message)}</div>
            </div>
        `;
  }
}

/**
 * Build a Report object from the current builder state
 */
function buildReportFromState() {
  return {
    id: reportState.id || "preview",
    title: reportState.title || "Preview Report",
    description: reportState.description || "",
    datasource: reportState.datasource || "",
    filters: reportState.filters || [],
    customFilters: reportState.customFilters || [],
    timeColumns: reportState.timeColumns || {},
    locationColumns: reportState.locationColumns || {},
    sections: reportState.sections.map((section) => ({
      id: section.id || "",
      title: section.title || "",
      description: section.description || "",
      layout: section.layout || "single",
      components: (section.components || []).map((comp) => ({
        type: comp.type,
        title: comp.title || "",
        description: comp.description || "",
        content: comp.content || "",
        sql: comp.sql || "",
        filterTable: comp.filterTable || "",
        query: comp.query || null,
        facet: comp.facet || null,
        stacked: comp.stacked || false,
        horizontal: comp.horizontal || false,
        showValues: comp.showValues || false,
        showValuesWithAxis: comp.showValuesWithAxis || false,
        showAxisTitles:
          comp.showAxisTitles !== undefined
            ? comp.showAxisTitles
            : !comp.hideAxes,
        comparison: comp.comparison || "",
        target: comp.target || null,
        referenceLines: comp.referenceLines || [],
        axisLabels: comp.axisLabels || null,
        axisTooltips: comp.axisTooltips || null,
        axisFormulas: comp.axisFormulas || null,
        yLimit: comp.yLimit || null,
        columnMapping: comp.columnMapping || null,
        heatmap: comp.heatmap || null,
        data: comp.data || {},
        pivot: comp.pivot || null,
        firstRowIsHeader: comp.firstRowIsHeader || false,
        headerTooltips: comp.headerTooltips || null,
        headerFormulas: comp.headerFormulas || null,
        splitColumns: comp.splitColumns || null,
        splitChildHeaderMode: comp.splitChildHeaderMode || "",
        tableHeaderWrap: comp.tableHeaderWrap || false,
        tableHeaderStyle: comp.tableHeaderStyle || "",
        tableCellBoundaries:
          comp.tableCellBoundaries || comp.tableVerticalLines || false,
        tableVerticalLines:
          comp.tableVerticalLines || comp.tableCellBoundaries || false,
        pageSize: comp.pageSize || undefined,
        trendLine: comp.trendLine || false,
        wrapLabels: comp.wrapLabels || false,
        infoboxBody: comp.infoboxBody || "",
        sql: comp.sql || "",
        accentColor: comp.accentColor || "",
        unit: comp.unit || "",
        tone: comp.tone || "",
        lowerIsBetter: comp.lowerIsBetter || false,
        valueTemplate: comp.valueTemplate || "",
      })),
    })),
  };
}

/**
 * Render the preview content with report data and errors
 */
function renderPreviewContent(report, errors, warnings = []) {
  const renderArea = document.getElementById("preview-render-area");
  if (!renderArea) return;

  // Clean up any existing chart/map instances (global function from components.js)
  if (typeof window.cleanupAllInstances === "function") {
    window.cleanupAllInstances();
  }

  renderArea.innerHTML = "";

  // Render warnings panel if any
  if (warnings.length > 0) {
    const panel = document.createElement("div");
    panel.className = "design-warnings-panel";
    panel.innerHTML = `
            <div class="design-warnings-header" role="button" tabindex="0">
                <span class="design-warnings-badge">${warnings.length}</span>
                <span>Design Warning${warnings.length !== 1 ? "s" : ""}</span>
                <span class="design-warnings-toggle">▼</span>
            </div>
            <ul class="design-warnings-list">
                ${warnings.map((w) => `<li${w.severity === "low" ? ' class="severity-low"' : ""}>${escapeHtml(w.message)}${w.component ? ` <span class="design-warning-location">(${escapeHtml(w.component)})</span>` : ""}${w.tip ? ` <button class="design-warning-tip-btn" title="More info">?</button><div class="design-warning-tip">${escapeHtml(w.tip)}</div>` : ""}</li>`).join("")}
            </ul>
        `;
    const header = panel.querySelector(".design-warnings-header");
    const list = panel.querySelector(".design-warnings-list");
    const toggle = panel.querySelector(".design-warnings-toggle");
    header.addEventListener("click", () => {
      const collapsed = list.classList.toggle("collapsed");
      toggle.textContent = collapsed ? "▶" : "▼";
    });
    panel.querySelectorAll(".design-warning-tip-btn").forEach((btn) => {
      btn.addEventListener("click", (e) => {
        e.stopPropagation();
        const tip = btn.nextElementSibling;
        tip.classList.toggle("visible");
      });
    });
    renderArea.appendChild(panel);
  }

  // Build error lookup for quick access
  const errorMap = {};
  for (const err of errors) {
    const key = `${err.sectionIndex}-${err.componentIndex}`;
    errorMap[key] = err;
  }

  // Render each section
  for (let sectionIdx = 0; sectionIdx < report.sections.length; sectionIdx++) {
    const section = report.sections[sectionIdx];
    const sectionDiv = document.createElement("div");
    sectionDiv.className = "section";

    // Section title
    if (section.title) {
      const titleDiv = document.createElement("div");
      titleDiv.className = "section-title";
      titleDiv.textContent = section.title;
      sectionDiv.appendChild(titleDiv);
    }

    // Section description
    if (section.description) {
      const descDiv = document.createElement("div");
      descDiv.className = "section-description";
      descDiv.innerHTML = section.description;
      sectionDiv.appendChild(descDiv);
    }

    // Components container
    const componentsDiv = document.createElement("div");
    const layout = normalizeLayout(section.layout);
    componentsDiv.className = `section-components layout-${layout}`;

    // Render each component
    for (let compIdx = 0; compIdx < section.components.length; compIdx++) {
      const component = section.components[compIdx];
      const errorKey = `${sectionIdx}-${compIdx}`;

      if (errorMap[errorKey]) {
        // Render error state for this component
        const errorDiv = renderComponentError(
          component,
          errorMap[errorKey],
          sectionIdx,
          compIdx,
        );
        componentsDiv.appendChild(errorDiv);
      } else {
        // Render successful component using global function from components.js
        if (typeof window.renderComponent === "function") {
          window.renderComponent(componentsDiv, component);
        } else {
          // Fallback if components.js not loaded
          const placeholder = document.createElement("div");
          placeholder.className = "component";
          placeholder.innerHTML = `
                        <h3>${escapeHtml(component.title || "Component")}</h3>
                        <p>Type: ${escapeHtml(component.type)}</p>
                        <p style="color: var(--color-text-secondary);">Preview rendering not available</p>
                    `;
          componentsDiv.appendChild(placeholder);
        }
      }
    }

    sectionDiv.appendChild(componentsDiv);
    renderArea.appendChild(sectionDiv);
  }
}

/**
 * Render an error state for a failed component
 */
function renderComponentError(component, error, sectionIdx, compIdx) {
  const div = document.createElement("div");
  div.className = "component-error";

  div.innerHTML = `
        <div class="component-error-header">
            <span class="component-error-title">
                <span class="component-error-badge">${escapeHtml(component.type || "Component")}</span>
                ${escapeHtml(component.title || "Untitled Component")}
            </span>
        </div>
        <div class="component-error-message">${escapeHtml(error.error)}</div>
        <div class="component-error-actions">
            <button class="btn-edit-component" data-section="${sectionIdx}" data-component="${compIdx}">
                Edit Component
            </button>
        </div>
    `;

  // Add click handler for edit button
  const editBtn = div.querySelector(".btn-edit-component");
  editBtn?.addEventListener("click", () => {
    editComponentFromPreview(sectionIdx, compIdx);
  });

  return div;
}

/**
 * Navigate to sections tab and open component for editing
 */
function editComponentFromPreview(sectionIdx, compIdx) {
  // Switch to sections tab
  if (showTabCallback) {
    showTabCallback("sections");
  }

  // Open component modal after a short delay to let the tab render
  setTimeout(() => {
    if (showComponentModalCallback) {
      showComponentModalCallback(sectionIdx, compIdx);
    }
  }, 100);
}

/**
 * Normalize layout value
 */
function normalizeLayout(layout) {
  const layoutMap = {
    "one-column": "single",
    "single-column": "single",
    full: "single",
    single: "single",
    "two-column": "two-column",
    "three-column": "three-column",
    "four-column": "four-column",
  };
  return layoutMap[layout] || "single";
}

/**
 * Update the loading status text
 */
function setLoadingStatus(text) {
  const el = document.getElementById("preview-loading-status");
  if (el) el.textContent = text;
}

/**
 * Clear the loading steps list
 */
function resetLoadingSteps() {
  const el = document.getElementById("preview-loading-steps");
  if (el) el.innerHTML = "";
}

/**
 * Append a completed step to the loading steps list
 */
function addLoadingStep(text) {
  const el = document.getElementById("preview-loading-steps");
  if (!el) return;
  const li = document.createElement("li");
  li.textContent = text;
  el.appendChild(li);
}
