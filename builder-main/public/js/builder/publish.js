// Publish Module
// Handles publishing reports to the server and editing existing reports

import {
  elements,
  reportState,
  markClean,
  API_BASE,
  isPublishingAllowed,
} from "./state.js";
import { showError, showSuccess, escapeHtml } from "./utils.js";
import { parseReportYAML } from "./api.js";
import { populateFormFromReport } from "./yaml-handler.js";
import { authPostJSON } from "../authFetch.js";

// Callbacks
let onShowTab = () => {};
let onMarkClean = () => {};

export function setPublishCallbacks(callbacks) {
  if (callbacks.showTab) onShowTab = callbacks.showTab;
  if (callbacks.markClean) onMarkClean = callbacks.markClean;
}

// Store categories data
let categoriesData = [];

// Store grouped reports for Edit tab
let groupedReports = {};

function formatFolderPath(path) {
  return path
    .split("/")
    .map((part) =>
      part
        .split("-")
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(" "),
    )
    .join(" / ");
}

// Validation state for two-step publish flow
let validationState = {
  validated: false,
  valid: false,
  errors: [],
  warnings: [],
  fileExists: false,
  targetPath: "",
  _yaml: "",
};

function resetValidationState() {
  validationState = {
    validated: false,
    valid: false,
    errors: [],
    warnings: [],
    fileExists: false,
    targetPath: "",
    _yaml: "",
  };
  const resultsDiv = document.getElementById("publish-validation-results");
  if (resultsDiv) {
    resultsDiv.innerHTML = "";
    resultsDiv.classList.add("hidden");
  }
  const actionsDiv = document.getElementById("publish-actions");
  if (actionsDiv) actionsDiv.classList.add("hidden");
  const overwriteSection = document.getElementById("publish-overwrite-section");
  if (overwriteSection) overwriteSection.classList.add("hidden");
  const overwriteCheckbox = document.getElementById("publish-overwrite");
  if (overwriteCheckbox) overwriteCheckbox.checked = false;
}

/**
 * Load categories from the server
 */
export async function loadCategories() {
  try {
    const data = await apiCache.fetch(`${API_BASE}/builder/categories`);
    categoriesData = data.categories || [];
    return categoriesData;
  } catch (err) {
    console.error("Failed to load categories:", err);
    return [];
  }
}

/**
 * Load all reports for the report picker
 */
export async function loadReportsForPicker() {
  try {
    const data = await apiCache.fetch(`${API_BASE}/reports`);
    return data.reports || [];
  } catch (err) {
    console.error("Failed to load reports:", err);
    return [];
  }
}

/**
 * Load report source (raw YAML) for editing
 */
export async function loadReportSource(reportId) {
  try {
    return await apiCache.fetch(`${API_BASE}/report/source/${reportId}`);
  } catch (err) {
    console.error("Failed to load report source:", err);
    throw err;
  }
}

/**
 * Publish report to the server
 */
export async function publishReport(
  yaml,
  category,
  subcategory,
  filename,
  overwrite = false,
) {
  try {
    return await authPostJSON(`${API_BASE}/report/publish`, {
      yaml,
      category,
      subcategory,
      filename,
      overwrite,
    });
  } catch (err) {
    console.error("Failed to publish report:", err);
    throw err;
  }
}

/**
 * Load reports for the Edit tab (full page)
 */
export async function loadReportsForEditTab() {
  const listContainer = document.getElementById("report-picker-list");
  const searchInput = document.getElementById("report-picker-search");

  if (!listContainer) return;

  listContainer.innerHTML =
    '<div class="loading-spinner"></div><p style="text-align: center; color: var(--color-text-muted);">Loading reports...</p>';

  // Load reports
  const reports = await loadReportsForPicker();

  if (reports.length === 0) {
    listContainer.innerHTML =
      '<p style="padding: 20px; text-align: center; color: var(--color-text-muted);">No reports found.</p>';
    return;
  }

  // Group reports by category
  groupedReports = {};
  reports.forEach((report) => {
    const category = report.category || "Uncategorized";
    if (!groupedReports[category]) {
      groupedReports[category] = [];
    }
    groupedReports[category].push(report);
  });

  // Render the grouped list
  renderReportList(groupedReports, listContainer);

  // Setup search (remove old listener first)
  if (searchInput) {
    searchInput.value = "";
    const newSearchInput = searchInput.cloneNode(true);
    searchInput.parentNode.replaceChild(newSearchInput, searchInput);
    newSearchInput.addEventListener("input", (e) => {
      const query = e.target.value.toLowerCase();
      renderReportList(groupedReports, listContainer, query);
    });
  }
}

/**
 * Render the grouped report list with collapsible categories
 */
function renderReportList(grouped, container, filter = "") {
  container.innerHTML = "";

  const sortedCategories = Object.keys(grouped).sort();
  // When filtering, expand all categories to show matches
  const expandAll = filter !== "";

  for (const category of sortedCategories) {
    const reports = grouped[category].filter(
      (r) =>
        filter === "" ||
        r.title.toLowerCase().includes(filter) ||
        r.id.toLowerCase().includes(filter),
    );

    if (reports.length === 0) continue;

    const categoryDiv = document.createElement("div");
    categoryDiv.className = "report-picker-category";

    const headerDiv = document.createElement("div");
    headerDiv.className = "report-picker-category-header";
    headerDiv.style.cursor = "pointer";
    headerDiv.style.display = "flex";
    headerDiv.style.justifyContent = "space-between";
    headerDiv.style.alignItems = "center";
    headerDiv.innerHTML = `
            <span>${category} <span style="opacity: 0.6; font-weight: normal; font-size: 0.85em;">(${reports.length})</span></span>
            <span class="collapse-icon" style="font-size: 12px;">${expandAll ? "▼" : "▶"}</span>
        `;
    categoryDiv.appendChild(headerDiv);

    const reportsContainer = document.createElement("div");
    reportsContainer.className = "report-picker-category-items";
    reportsContainer.style.display = expandAll ? "block" : "none";

    for (const report of reports) {
      const itemDiv = document.createElement("div");
      itemDiv.className = "report-picker-item";
      itemDiv.innerHTML = `
                <div class="report-picker-item-title">${report.title}</div>
                <div class="report-picker-item-id">${report.id}</div>
            `;
      itemDiv.addEventListener("click", (e) => {
        e.stopPropagation();
        selectReportForEdit(report.id);
      });
      reportsContainer.appendChild(itemDiv);
    }

    categoryDiv.appendChild(reportsContainer);

    // Toggle expand/collapse on header click
    headerDiv.addEventListener("click", () => {
      const isExpanded = reportsContainer.style.display !== "none";
      reportsContainer.style.display = isExpanded ? "none" : "block";
      const icon = headerDiv.querySelector(".collapse-icon");
      if (icon) {
        icon.textContent = isExpanded ? "▶" : "▼";
      }
    });

    container.appendChild(categoryDiv);
  }

  if (container.children.length === 0) {
    container.innerHTML =
      '<p style="padding: 20px; text-align: center; color: var(--color-text-muted);">No reports match your search.</p>';
  }
}

/**
 * Select a report to edit
 */
async function selectReportForEdit(reportId) {
  try {
    showSuccess("Loading report...");
    const source = await loadReportSource(reportId);

    // Check for API errors
    if (source.error) {
      showError("Failed to load report: " + source.error);
      return;
    }

    if (!source.yaml) {
      showError("Report source not found or empty");
      return;
    }

    // Parse the YAML and load it into the form
    const result = await parseReportYAML(source.yaml);

    if (!result.valid) {
      showError("Invalid YAML: " + result.errors.join(", "));
      return;
    }

    // Load report into form
    const report = result.report;
    await populateFormFromReport(report);

    // Store the original path for publishing
    reportState._editingPath = source.path;
    reportState._editingFilename = source.filename;

    // Mark as clean since we just loaded
    markClean();

    // Go to metadata tab
    onShowTab("metadata");
    showSuccess(`Loaded report: ${report.title}`);
  } catch (err) {
    showError("Failed to load report: " + err.message);
  }
}

/**
 * Load categories for the Publish tab (full page)
 */
export async function loadCategoriesForPublishTab() {
  const categorySelect = document.getElementById("publish-category");
  const subcategorySelect = document.getElementById("publish-subcategory");
  const filenameInput = document.getElementById("publish-filename");
  const fieldErrorDiv = document.getElementById("publish-field-error");

  if (!categorySelect) return;

  // Clear stale validation results when navigating to publish tab
  resetValidationState();
  fieldErrorDiv?.classList.add("hidden");

  // Load categories
  await loadCategories();

  // Populate category dropdown
  categorySelect.innerHTML = '<option value="">Select category...</option>';
  categoriesData.forEach((cat) => {
    const option = document.createElement("option");
    option.value = cat.name;
    option.textContent = cat.name;
    categorySelect.appendChild(option);
  });

  // Setup category change handler (remove old listener first)
  const newCategorySelect = categorySelect.cloneNode(true);
  if (!categorySelect.parentNode) {
    console.error(
      "[Builder] categorySelect.parentNode is null - element may have been removed from DOM",
    );
    return;
  }
  categorySelect.parentNode.replaceChild(newCategorySelect, categorySelect);

  newCategorySelect.addEventListener("change", () => {
    const selectedCategory = newCategorySelect.value;
    const subSelect = document.getElementById("publish-subcategory");
    subSelect.innerHTML = '<option value="">None</option>';

    const category = categoriesData.find((c) => c.name === selectedCategory);
    if (category && category.subcategories) {
      category.subcategories.forEach((sub) => {
        const option = document.createElement("option");
        option.value = sub;
        option.textContent = formatFolderPath(sub);
        subSelect.appendChild(option);
      });
    }

    // If editing, try to pre-select the subcategory
    if (reportState._editingPath) {
      const parts = reportState._editingPath.split("/");
      if (parts.length > 1 && parts[0] === selectedCategory) {
        subSelect.value = parts.slice(1).join("/");
      }
    }
  });

  // Pre-fill from editing path if available
  if (reportState._editingPath) {
    const parts = reportState._editingPath.split("/");
    if (parts.length > 0) {
      newCategorySelect.value = parts[0];
      newCategorySelect.dispatchEvent(new Event("change"));
    }
  }

  // Pre-fill filename from editing or generate from ID/title
  if (reportState._editingFilename) {
    filenameInput.value = reportState._editingFilename;
  } else if (reportState.id) {
    // Extract just the filename part from the ID
    const idParts = reportState.id.split("/");
    filenameInput.value = idParts[idParts.length - 1];
  } else if (reportState.title) {
    // Generate from title
    filenameInput.value = reportState.title
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-|-$/g, "");
  }
}

/**
 * Validate report before publishing (step 1 of two-step flow)
 */
export async function validateForPublish() {
  if (!isPublishingAllowed()) {
    showError("Publishing is disabled in production mode");
    return;
  }

  const categorySelect = document.getElementById("publish-category");
  const subcategorySelect = document.getElementById("publish-subcategory");
  const filenameInput = document.getElementById("publish-filename");
  const fieldErrorDiv = document.getElementById("publish-field-error");
  const validateBtn = document.getElementById("btn-validate-publish");

  const category = categorySelect?.value;
  const subcategory = subcategorySelect?.value || "";
  const filename = filenameInput?.value?.trim();

  // Reset previous results
  resetValidationState();
  fieldErrorDiv?.classList.add("hidden");

  // Frontend field validation
  if (!reportState.title || reportState.title.trim() === "") {
    if (fieldErrorDiv) {
      fieldErrorDiv.textContent =
        "Report title is required. Go to Metadata tab to set it.";
      fieldErrorDiv.classList.remove("hidden");
    }
    return;
  }
  if (!category) {
    if (fieldErrorDiv) {
      fieldErrorDiv.textContent = "Please select a category";
      fieldErrorDiv.classList.remove("hidden");
    }
    return;
  }
  if (!filename) {
    if (fieldErrorDiv) {
      fieldErrorDiv.textContent = "Please enter a filename";
      fieldErrorDiv.classList.remove("hidden");
    }
    return;
  }

  // Disable button during validation
  if (validateBtn) {
    validateBtn.disabled = true;
    validateBtn.textContent = "Validating...";
  }

  try {
    // Generate YAML
    const { generateReportYAML } = await import("./api.js");
    const request = {
      id: reportState.id,
      title: reportState.title,
      datasource: reportState.datasource || "",
      keywords: reportState.keywords || "",
      filters: reportState.filters,
      customFilters: reportState.customFilters,
      timeColumns: reportState.timeColumns,
      locationColumns: reportState.locationColumns,
      sections: reportState.sections,
    };

    const yamlResult = await generateReportYAML(request);
    if (yamlResult.error) {
      if (fieldErrorDiv) {
        fieldErrorDiv.textContent = yamlResult.error;
        fieldErrorDiv.classList.remove("hidden");
      }
      return;
    }

    // Collect design warnings from YAML generation
    const designWarnings = (yamlResult.warnings || []).map((w) => ({
      message: w.message,
      component: w.component || "",
      section: "",
      type: "warning",
    }));

    // Call server validation
    const result = await authPostJSON(
      `${API_BASE}/report/publish?validateOnly=true`,
      { yaml: yamlResult.yaml, category, subcategory, filename },
    );

    // Merge server errors/warnings with design warnings
    const serverErrors = (result.errors || []).map((e) => ({
      ...e,
      type: "error",
    }));
    const serverWarnings = (result.warnings || []).map((w) => ({
      ...w,
      type: "warning",
    }));
    const allWarnings = [...serverWarnings, ...designWarnings];

    validationState = {
      validated: true,
      valid: result.valid !== false && serverErrors.length === 0,
      errors: serverErrors,
      warnings: allWarnings,
      fileExists: result.fileExists || false,
      targetPath: result.path || "",
      _yaml: yamlResult.yaml,
    };

    if (!validationState.valid) {
      window.telemetry?.track("builder.validation_block", {
        errors: String(serverErrors.length),
        warnings: String(allWarnings.length),
      });
    }

    renderValidationResults();
  } catch (err) {
    if (fieldErrorDiv) {
      fieldErrorDiv.textContent = "Validation failed: " + err.message;
      fieldErrorDiv.classList.remove("hidden");
    }
  } finally {
    if (validateBtn) {
      validateBtn.disabled = false;
      validateBtn.textContent = "Validate";
    }
  }
}

/**
 * Render validation results panel
 */
function renderValidationResults() {
  const resultsDiv = document.getElementById("publish-validation-results");
  const actionsDiv = document.getElementById("publish-actions");
  if (!resultsDiv) return;

  const { errors, warnings, valid, fileExists } = validationState;
  const errorCount = errors.length;
  const warningCount = warnings.length;

  // Build summary
  let summaryClass, summaryText;
  if (errorCount > 0) {
    summaryClass = "has-errors";
    summaryText = `${errorCount} error${errorCount !== 1 ? "s" : ""}`;
    if (warningCount > 0)
      summaryText += `, ${warningCount} warning${warningCount !== 1 ? "s" : ""}`;
  } else if (warningCount > 0) {
    summaryClass = "warnings-only";
    summaryText = `Validation passed with ${warningCount} warning${warningCount !== 1 ? "s" : ""}`;
  } else {
    summaryClass = "valid";
    summaryText = "Validation passed — ready to publish";
  }

  let html = `<div class="publish-validation-results">`;
  html += `<div class="publish-validation-summary ${summaryClass}">${escapeHtml(summaryText)}</div>`;

  // Group errors by section
  if (errorCount > 0) {
    const grouped = {};
    for (const err of errors) {
      const key = err.section || "Report Configuration";
      if (!grouped[key]) grouped[key] = [];
      grouped[key].push(err);
    }
    for (const [section, items] of Object.entries(grouped)) {
      html += `<div class="publish-validation-group">`;
      html += `<div class="publish-validation-group-header" data-collapsed="false">`;
      html += `<span>${escapeHtml(section)} <span class="group-count">(${items.length})</span></span>`;
      html += `<span class="collapse-icon">▼</span>`;
      html += `</div>`;
      html += `<ul class="publish-validation-group-items">`;
      for (const item of items) {
        const comp = item.component
          ? ` <span class="item-component">${escapeHtml(item.component)}</span>`
          : "";
        html += `<li class="publish-validation-item error">${escapeHtml(item.message)}${comp}</li>`;
      }
      html += `</ul></div>`;
    }
  }

  // Warnings in a flat group
  if (warningCount > 0) {
    html += `<div class="publish-validation-group">`;
    html += `<div class="publish-validation-group-header" data-collapsed="false">`;
    html += `<span>Warnings <span class="group-count">(${warningCount})</span></span>`;
    html += `<span class="collapse-icon">▼</span>`;
    html += `</div>`;
    html += `<ul class="publish-validation-group-items">`;
    for (const item of warnings) {
      const comp = item.component
        ? ` <span class="item-component">${escapeHtml(item.component)}</span>`
        : "";
      html += `<li class="publish-validation-item warning">${escapeHtml(item.message)}${comp}</li>`;
    }
    html += `</ul></div>`;
  }

  html += `</div>`;

  resultsDiv.innerHTML = html;
  resultsDiv.classList.remove("hidden");

  // Attach collapse/expand handlers
  resultsDiv
    .querySelectorAll(".publish-validation-group-header")
    .forEach((header) => {
      header.addEventListener("click", () => {
        const items = header.nextElementSibling;
        const icon = header.querySelector(".collapse-icon");
        const collapsed = header.getAttribute("data-collapsed") === "true";
        if (collapsed) {
          items.style.display = "";
          icon.textContent = "▼";
          header.setAttribute("data-collapsed", "false");
        } else {
          items.style.display = "none";
          icon.textContent = "▶";
          header.setAttribute("data-collapsed", "true");
        }
      });
    });

  // Show/hide publish actions
  if (actionsDiv) {
    if (valid) {
      actionsDiv.classList.remove("hidden");

      // Show overwrite section if file exists
      const overwriteSection = document.getElementById(
        "publish-overwrite-section",
      );
      if (overwriteSection) {
        if (fileExists) {
          overwriteSection.classList.remove("hidden");
        } else {
          overwriteSection.classList.add("hidden");
        }
      }
    } else {
      actionsDiv.classList.add("hidden");
    }
  }
}

/**
 * Handle the publish confirmation (step 2 of two-step flow)
 */
export async function confirmPublish() {
  if (!isPublishingAllowed()) {
    showError("Publishing is disabled in production mode");
    return;
  }

  // Must validate first
  if (!validationState.validated || !validationState.valid) {
    showError("Please validate the report first");
    return;
  }

  const categorySelect = document.getElementById("publish-category");
  const subcategorySelect = document.getElementById("publish-subcategory");
  const filenameInput = document.getElementById("publish-filename");
  const fieldErrorDiv = document.getElementById("publish-field-error");
  const overwriteCheckbox = document.getElementById("publish-overwrite");
  const publishBtn = document.getElementById("btn-confirm-publish");

  const category = categorySelect?.value;
  const subcategory = subcategorySelect?.value || "";
  const filename = filenameInput?.value?.trim();
  const overwrite = overwriteCheckbox?.checked || false;

  // If file exists and overwrite not checked, show field error
  if (validationState.fileExists && !overwrite) {
    if (fieldErrorDiv) {
      fieldErrorDiv.textContent =
        'Check "Overwrite existing file" to replace the existing report.';
      fieldErrorDiv.classList.remove("hidden");
    }
    return;
  }

  fieldErrorDiv?.classList.add("hidden");

  if (publishBtn) {
    publishBtn.disabled = true;
    publishBtn.textContent = "Publishing...";
  }

  try {
    const result = await publishReport(
      validationState._yaml,
      category,
      subcategory,
      filename,
      overwrite,
    );

    if (!result.success) {
      if (result.error === "exists") {
        // Should not happen since we checked fileExists, but handle gracefully
        const overwriteSection = document.getElementById(
          "publish-overwrite-section",
        );
        if (overwriteSection) overwriteSection.classList.remove("hidden");
        return;
      }
      const msg =
        result.errors && result.errors.length > 0
          ? result.errors.map((e) => e.message).join(", ")
          : result.message || result.error || "Failed to publish";
      showError(msg);
      return;
    }

    window.telemetry?.track("builder.save", {
      overwrite: overwrite ? "1" : "0",
    });

    showSuccess(`Report published to ${result.path}`);
    onMarkClean();

    // Invalidate reports cache
    apiCache.invalidatePattern(/\/api\/reports$/);
    apiCache.invalidatePattern(/\/api\/report\/source\//);

    // Update state
    reportState._editingPath = subcategory
      ? `${category}/${subcategory}`
      : category;
    reportState._editingFilename = filename;

    resetValidationState();
  } catch (err) {
    showError("Failed to publish: " + err.message);
  } finally {
    if (publishBtn) {
      publishBtn.disabled = false;
      publishBtn.textContent = "Publish Report";
    }
  }
}
