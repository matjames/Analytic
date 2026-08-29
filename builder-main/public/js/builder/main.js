// Main Entry Point
// Initializes the application and wires up event listeners

import {
  elements,
  initElements,
  reportState,
  API_BASE,
  setOnStateChange,
  markDirty,
  markClean,
  setPublishingAllowed,
  isPublishingAllowed,
} from "./state.js";
import {
  loadAvailableSchemas,
  loadAvailableTables,
  loadTableColumns,
} from "./api.js";
import { showSuccess } from "./utils.js";
import { validateReportIdField, validateRequiredField } from "./validation.js";
import {
  renderSections,
  setCallbacks as setSectionsCallbacks,
} from "./sections.js";
import {
  showComponentModal,
  closeComponentModal,
  saveComponent,
  setCallbacks as setComponentsCallbacks,
} from "./components.js";
import {
  importYAML,
  closeYamlModal,
  copyYAMLToClipboard,
  downloadYAML,
  setCallbacks as setYamlCallbacks,
  renderCustomFiltersList,
} from "./yaml-handler.js";
import {
  updatePreview,
  handleGenerateYAML,
  copyPreview,
  closePreviewModal,
  previewQueryData,
} from "./preview.js";
import {
  initPreviewTab,
  refreshPreview,
  setPreviewCallbacks,
} from "./preview-tab.js";
import {
  loadReportsForEditTab,
  loadCategoriesForPublishTab,
  validateForPublish,
  confirmPublish,
  setPublishCallbacks,
} from "./publish.js";
import { initKeywordSuggestions } from "./keywords.js";
import { getAuthConfig } from "../config.js";
import { initAuth } from "../auth.js";

/**
 * Disable the publish tab (when publishing is disabled in production)
 * Tab remains visible but greyed out, with an info banner explaining GitOps workflow.
 */
function disablePublishTab() {
  const publishTabBtn = document.querySelector(
    '.builder-tab[data-tab="publish"]',
  );
  if (publishTabBtn) {
    publishTabBtn.classList.add("disabled");
    publishTabBtn.setAttribute("aria-disabled", "true");
  }
  const publishTabContent = document.getElementById("tab-publish");
  if (publishTabContent) {
    publishTabContent.innerHTML =
      '<div class="publish-disabled-notice">' +
      "<h2>Publish Report</h2>" +
      '<div class="info-banner">' +
      "<strong>Publishing is disabled in production.</strong>" +
      "<p>Report changes are managed through version control. " +
      "Export your YAML from the Preview tab, then commit it to the configs/ directory.</p>" +
      "</div>" +
      "</div>";
  }
}

/**
 * Show a tab
 */
function showTab(tabName) {
  reportState.currentTab = tabName;

  // Hide all tab contents
  document.querySelectorAll(".tab-content").forEach((tab) => {
    tab.classList.add("hidden");
  });

  // Show current tab content
  const currentTab = document.getElementById(`tab-${tabName}`);
  if (currentTab) {
    currentTab.classList.remove("hidden");

    // Tab-specific initialization
    if (tabName === "sections") {
      renderSections();
    } else if (tabName === "preview") {
      initPreviewTab();
    } else if (tabName === "edit") {
      loadReportsForEditTab();
    } else if (tabName === "publish") {
      loadCategoriesForPublishTab();
    }
  }

  // Update tab button active state (new top tabs)
  document.querySelectorAll(".builder-tab").forEach((btn) => {
    btn.classList.remove("active");
  });
  const activeBtn = document.querySelector(
    `.builder-tab[data-tab="${tabName}"]`,
  );
  if (activeBtn) {
    activeBtn.classList.add("active");
  }

  window.scrollTo(0, 0);
}

/**
 * Update tab badges based on current state
 */
function updateTabBadges() {
  // Metadata badge
  const metadataBadge = document.getElementById("badge-metadata");
  const metadataTab = document.querySelector(
    '.builder-tab[data-tab="metadata"]',
  );
  if (metadataBadge && metadataTab) {
    if (reportState.title && reportState.id) {
      metadataBadge.textContent = "✓";
      metadataBadge.className = "tab-badge success";
      metadataTab.classList.add("completed");
    } else {
      metadataBadge.textContent = "";
      metadataBadge.className = "tab-badge";
      metadataTab.classList.remove("completed");
    }
  }

  // Sections badge
  const sectionsBadge = document.getElementById("badge-sections");
  const sectionsTab = document.querySelector(
    '.builder-tab[data-tab="sections"]',
  );
  if (sectionsBadge && sectionsTab) {
    const sectionCount = reportState.sections.length;
    const componentCount = reportState.sections.reduce(
      (sum, s) => sum + s.components.length,
      0,
    );

    if (componentCount > 0) {
      sectionsBadge.textContent = `${sectionCount}s · ${componentCount}c`;
      sectionsBadge.className = "tab-badge success";
      sectionsTab.classList.add("completed");
    } else if (sectionCount > 0) {
      sectionsBadge.textContent = `${sectionCount}`;
      sectionsBadge.className = "tab-badge warning";
      sectionsTab.classList.remove("completed");
    } else {
      sectionsBadge.textContent = "";
      sectionsBadge.className = "tab-badge";
      sectionsTab.classList.remove("completed");
    }
  }
}

/**
 * Reset builder to new report state
 */
function resetToNewReport() {
  if (
    reportState.isDirty &&
    !confirm("You have unsaved changes. Start a new report anyway?")
  ) {
    return;
  }

  // Reset state
  reportState.id = "";
  reportState.title = "";
  reportState.keywords = "";
  reportState.author = "";
  reportState.filters = [];
  reportState.sections = [];
  reportState.customFilters = [];
  reportState.timeColumns = {};
  reportState.locationColumns = {};
  reportState.datasource = "";
  reportState.isDirty = false;

  // Clear form fields
  elements.reportId.value = "";
  elements.reportTitle.value = "";
  if (elements.reportKeywords) elements.reportKeywords.value = "";
  if (elements.reportAuthor) elements.reportAuthor.value = "";
  if (elements.reportDatasource) elements.reportDatasource.value = "";
  elements.filterCheckboxes.forEach((cb) => (cb.checked = false));
  const resetFacilityTypeOptions = document.getElementById(
    "facility-type-options",
  );
  if (resetFacilityTypeOptions) resetFacilityTypeOptions.style.display = "none";
  const resetFacilityMultiRadio = document.getElementById(
    "facility-type-multi",
  );
  if (resetFacilityMultiRadio) resetFacilityMultiRadio.checked = true;

  // Clear time/location columns
  ["year", "month", "week", "quarter"].forEach((col) => {
    const el = document.getElementById(`time-column-${col}`);
    if (el) el.value = "";
  });
  ["district", "region", "facility"].forEach((col) => {
    const el = document.getElementById(`location-column-${col}`);
    if (el) el.value = "";
  });

  // Update UI
  updateTabBadges();
  showTab("metadata");
  showSuccess("Started new report");
}

/**
 * Setup all event listeners
 */
function setupEventListeners() {
  // Top tab navigation
  document.querySelectorAll(".builder-tab").forEach((btn) => {
    btn.addEventListener("click", () => {
      if (btn.classList.contains("disabled")) return;
      const tab = btn.getAttribute("data-tab");
      showTab(tab);
    });
  });

  // Header dropdown toggle
  const openReportBtn = document.getElementById("btn-open-report");
  const openReportMenu = document.getElementById("open-report-menu");
  if (openReportBtn && openReportMenu) {
    openReportBtn.addEventListener("click", (e) => {
      e.stopPropagation();
      openReportMenu.classList.toggle("hidden");
    });

    // Close dropdown when clicking outside
    document.addEventListener("click", () => {
      openReportMenu.classList.add("hidden");
    });

    // Dropdown item actions
    openReportMenu.querySelectorAll(".dropdown-item").forEach((item) => {
      item.addEventListener("click", () => {
        const action = item.dataset.action;
        openReportMenu.classList.add("hidden");
        if (action === "import") {
          showTab("import");
        } else if (action === "edit") {
          showTab("edit");
        }
      });
    });
  }

  // New report button
  const newReportBtn = document.getElementById("btn-new-report");
  if (newReportBtn) {
    newReportBtn.addEventListener("click", resetToNewReport);
  }

  // YAML Panel Toggle
  const panelToggle = document.getElementById("panel-toggle");
  const yamlPanel = document.getElementById("yaml-panel");
  const builderContainer = document.querySelector(".builder-container");
  if (panelToggle && yamlPanel) {
    panelToggle.addEventListener("click", () => {
      yamlPanel.classList.toggle("collapsed");
      builderContainer?.classList.toggle("panel-collapsed");
    });
  }

  // Collapsible Time Filters Section
  const timeFiltersHeader = document.getElementById("time-filters-header");
  const timeFiltersContent = document.getElementById("time-filters-content");
  if (timeFiltersHeader && timeFiltersContent) {
    timeFiltersHeader.addEventListener("click", () => {
      const isExpanded = timeFiltersContent.style.display !== "none";
      timeFiltersContent.style.display = isExpanded ? "none" : "block";
      const icon = timeFiltersHeader.querySelector(".collapse-icon");
      if (icon) {
        icon.textContent = isExpanded ? "▶" : "▼";
      }
    });
  }

  // Import YAML button (now on Import tab)
  elements.importYamlBtn?.addEventListener("click", importYAML);

  // Preview buttons
  elements.copyPreviewBtn.addEventListener("click", copyPreview);
  elements.generateYamlBtn.addEventListener("click", handleGenerateYAML);

  // Modal close buttons
  elements.closeComponentModalBtn.addEventListener(
    "click",
    closeComponentModal,
  );
  const closeComponentModalBtn2 = document.getElementById(
    "btn-close-component-modal-2",
  );
  if (closeComponentModalBtn2) {
    closeComponentModalBtn2.addEventListener("click", (e) => {
      e.preventDefault();
      closeComponentModal();
    });
  }
  elements.closeYamlModalBtn.addEventListener("click", closeYamlModal);
  elements.closePreviewModalBtn.addEventListener("click", closePreviewModal);

  // YAML modal buttons
  elements.copyYamlBtn.addEventListener("click", copyYAMLToClipboard);
  elements.downloadYamlBtn.addEventListener("click", downloadYAML);

  // Form inputs - update state with validation
  elements.reportId.addEventListener("input", (e) => {
    reportState.id = e.target.value;
    validateReportIdField(e.target);
    markDirty();
    updateTabBadges();
  });
  elements.reportTitle.addEventListener("input", (e) => {
    reportState.title = e.target.value;
    validateRequiredField(e.target, "Title");
    markDirty();
    updateTabBadges();
  });
  // Keywords input with suggestions
  if (elements.reportKeywords) {
    elements.reportKeywords.addEventListener("input", (e) => {
      reportState.keywords = e.target.value;
      markDirty();
    });
  }

  // Author input
  if (elements.reportAuthor) {
    elements.reportAuthor.addEventListener("input", (e) => {
      reportState.author = e.target.value;
      markDirty();
    });
  }

  // Filters
  elements.filterCheckboxes.forEach((checkbox) => {
    if (checkbox.id === "filter-facility") return; // handled separately below
    checkbox.addEventListener("change", (e) => {
      if (e.target.checked) {
        reportState.filters.push(e.target.value);
      } else {
        reportState.filters = reportState.filters.filter(
          (f) => f !== e.target.value,
        );
      }
      markDirty();
    });
  });

  // Facility filter — special handler supporting single-select and multi-select variants
  const facilityCheckbox = document.getElementById("filter-facility");
  const facilityTypeOptions = document.getElementById("facility-type-options");
  const facilityTypeRadios = document.querySelectorAll(
    'input[name="facility-type"]',
  );

  if (facilityCheckbox) {
    facilityCheckbox.addEventListener("change", (e) => {
      if (e.target.checked) {
        const selectedType =
          document.querySelector('input[name="facility-type"]:checked')
            ?.value || "facility";
        reportState.filters.push(selectedType);
        if (facilityTypeOptions) facilityTypeOptions.style.display = "block";
      } else {
        reportState.filters = reportState.filters.filter(
          (f) => f !== "facility" && f !== "facility_select",
        );
        if (facilityTypeOptions) facilityTypeOptions.style.display = "none";
      }
      markDirty();
    });
  }

  facilityTypeRadios.forEach((radio) => {
    radio.addEventListener("change", (e) => {
      const newValue = e.target.value;
      const oldValue = newValue === "facility" ? "facility_select" : "facility";
      const idx = reportState.filters.indexOf(oldValue);
      if (idx !== -1) {
        reportState.filters[idx] = newValue;
      } else if (!reportState.filters.includes(newValue)) {
        reportState.filters.push(newValue);
      }
      markDirty();
    });
  });

  // TimeColumns
  const timeColumnYear = document.getElementById("time-column-year");
  const timeColumnMonth = document.getElementById("time-column-month");
  const timeColumnWeek = document.getElementById("time-column-week");
  const timeColumnQuarter = document.getElementById("time-column-quarter");

  if (timeColumnYear) {
    timeColumnYear.addEventListener("change", (e) => {
      reportState.timeColumns.year = e.target.value || undefined;
    });
  }
  if (timeColumnMonth) {
    timeColumnMonth.addEventListener("change", (e) => {
      reportState.timeColumns.month = e.target.value || undefined;
    });
  }
  if (timeColumnWeek) {
    timeColumnWeek.addEventListener("change", (e) => {
      reportState.timeColumns.week = e.target.value || undefined;
    });
  }
  if (timeColumnQuarter) {
    timeColumnQuarter.addEventListener("change", (e) => {
      reportState.timeColumns.quarter = e.target.value || undefined;
    });
  }

  // LocationColumns
  const locationColumnDistrict = document.getElementById(
    "location-column-district",
  );
  const locationColumnRegion = document.getElementById(
    "location-column-region",
  );
  const locationColumnFacility = document.getElementById(
    "location-column-facility",
  );

  if (locationColumnDistrict) {
    locationColumnDistrict.addEventListener("change", (e) => {
      reportState.locationColumns.district = e.target.value || undefined;
    });
  }
  if (locationColumnRegion) {
    locationColumnRegion.addEventListener("change", (e) => {
      reportState.locationColumns.region = e.target.value || undefined;
    });
  }
  if (locationColumnFacility) {
    locationColumnFacility.addEventListener("change", (e) => {
      reportState.locationColumns.facility = e.target.value || undefined;
    });
  }

  // Collapsible Location Filters Section
  const locationFiltersHeader = document.getElementById(
    "location-filters-header",
  );
  const locationFiltersContent = document.getElementById(
    "location-filters-content",
  );
  if (locationFiltersHeader && locationFiltersContent) {
    locationFiltersHeader.addEventListener("click", () => {
      const isExpanded = locationFiltersContent.style.display !== "none";
      locationFiltersContent.style.display = isExpanded ? "none" : "block";
      const icon = locationFiltersHeader.querySelector(".collapse-icon");
      if (icon) {
        icon.textContent = isExpanded ? "▶" : "▼";
      }
    });
  }

  // Collapsible Custom Filters Section
  const customFiltersHeader = document.getElementById("custom-filters-header");
  const customFiltersContent = document.getElementById(
    "custom-filters-content",
  );
  if (customFiltersHeader && customFiltersContent) {
    customFiltersHeader.addEventListener("click", () => {
      const isExpanded = customFiltersContent.style.display !== "none";
      customFiltersContent.style.display = isExpanded ? "none" : "block";
      const icon = customFiltersHeader.querySelector(".collapse-icon");
      if (icon) {
        icon.textContent = isExpanded ? "▶" : "▼";
      }
    });
  }

  // Custom Filter Form Elements
  const customFilterSchema = document.getElementById("custom-filter-schema");
  const customFilterTable = document.getElementById("custom-filter-table");
  const customFilterColumn = document.getElementById("custom-filter-column");
  const customFilterLabel = document.getElementById("custom-filter-label");
  const customFilterType = document.getElementById("custom-filter-type");
  const customFilterDefaultGroup = document.getElementById(
    "custom-filter-default-group",
  );
  const customFilterDefault = document.getElementById("custom-filter-default");
  const btnAddCustomFilter = document.getElementById("btn-add-custom-filter");

  const resetCustomFilterDefault = (message = "Select a column first...") => {
    if (!customFilterDefault) return;
    customFilterDefault.innerHTML = "";
    const option = document.createElement("option");
    option.value = "";
    option.textContent = message;
    customFilterDefault.appendChild(option);
    customFilterDefault.disabled = true;
  };

  const syncCustomFilterDefaultVisibility = () => {
    const isSingleSelect = (customFilterType?.value || "select") === "select";
    if (customFilterDefaultGroup) {
      customFilterDefaultGroup.style.display = isSingleSelect
        ? "block"
        : "none";
    }
    if (!isSingleSelect) {
      resetCustomFilterDefault("Single-select filters only");
    }
  };

  const loadCustomFilterDefaultOptions = async () => {
    if (!customFilterDefault) return;

    const table = customFilterTable?.value;
    const column = customFilterColumn?.value;
    const isSingleSelect = (customFilterType?.value || "select") === "select";

    if (!isSingleSelect) {
      resetCustomFilterDefault("Single-select filters only");
      return;
    }

    if (!table || !column) {
      resetCustomFilterDefault("Select a column first...");
      return;
    }

    resetCustomFilterDefault("Loading values...");

    try {
      const url = `${API_BASE}/filters/custom?table=${encodeURIComponent(table)}&column=${encodeURIComponent(column)}`;
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const data = await response.json();
      const values = Array.isArray(data)
        ? data
        : Array.isArray(data.values)
          ? data.values
          : [];

      customFilterDefault.innerHTML = "";
      const firstOption = document.createElement("option");
      firstOption.value = "";
      firstOption.textContent = "Use first available value";
      customFilterDefault.appendChild(firstOption);

      values.forEach((value) => {
        const option = document.createElement("option");
        option.value = String(value);
        option.textContent = String(value);
        customFilterDefault.appendChild(option);
      });

      customFilterDefault.disabled = values.length === 0;
      if (values.length === 0) {
        firstOption.textContent = "No values found";
      }
    } catch (err) {
      console.error("Failed to load custom filter values:", err);
      resetCustomFilterDefault("Values unavailable");
    }
  };

  syncCustomFilterDefaultVisibility();

  // Populate schema dropdown
  if (customFilterSchema) {
    customFilterSchema.innerHTML = '<option value="">Select schema...</option>';
    reportState.availableSchemas.forEach((schema) => {
      const option = document.createElement("option");
      option.value = schema;
      option.textContent = schema;
      customFilterSchema.appendChild(option);
    });

    // Schema selection handler
    customFilterSchema.addEventListener("change", async (e) => {
      const schema = e.target.value;
      customFilterTable.innerHTML = '<option value="">Select table...</option>';
      customFilterColumn.innerHTML =
        '<option value="">Select column...</option>';
      customFilterTable.disabled = true;
      customFilterColumn.disabled = true;
      resetCustomFilterDefault();

      if (schema) {
        try {
          const response = await fetch(
            `${API_BASE}/schema/tables?schema=${encodeURIComponent(schema)}`,
          );
          const tables = await response.json();
          tables.forEach((t) => {
            const option = document.createElement("option");
            option.value = `${t.schema}.${t.table}`;
            option.textContent = t.table;
            customFilterTable.appendChild(option);
          });
          customFilterTable.disabled = false;
        } catch (err) {
          console.error("Failed to load tables:", err);
        }
      }
    });
  }

  // Table selection handler
  if (customFilterTable) {
    customFilterTable.addEventListener("change", async (e) => {
      const table = e.target.value;
      customFilterColumn.innerHTML =
        '<option value="">Select column...</option>';
      customFilterColumn.disabled = true;
      resetCustomFilterDefault();

      if (table) {
        try {
          const columns = await loadTableColumns(table);
          columns.forEach((col) => {
            const option = document.createElement("option");
            option.value = col.name;
            option.textContent = `${col.name} (${col.type})`;
            customFilterColumn.appendChild(option);
          });
          customFilterColumn.disabled = false;
        } catch (err) {
          console.error("Failed to load columns:", err);
        }
      }
    });
  }

  // Column selection handler - auto-fill label
  if (customFilterColumn) {
    customFilterColumn.addEventListener("change", async (e) => {
      const column = e.target.value;
      if (column && !customFilterLabel.value) {
        // Convert snake_case to Title Case for label
        const label = column
          .split("_")
          .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
          .join(" ");
        customFilterLabel.value = label;
      }
      await loadCustomFilterDefaultOptions();
    });
  }

  if (customFilterType) {
    customFilterType.addEventListener("change", async () => {
      syncCustomFilterDefaultVisibility();
      await loadCustomFilterDefaultOptions();
    });
  }

  // Add custom filter button
  if (btnAddCustomFilter) {
    btnAddCustomFilter.addEventListener("click", () => {
      const table = customFilterTable?.value;
      const column = customFilterColumn?.value;
      const label = customFilterLabel?.value?.trim();
      const type = customFilterType?.value || "select";
      const defaultValue =
        type === "select" ? customFilterDefault?.value || "" : "";
      const noDefault =
        type === "select" &&
        !!document.getElementById("custom-filter-no-default")?.checked;
      const hideNone =
        type === "select" &&
        !!document.getElementById("custom-filter-hide-none")?.checked;

      if (!table || !column || !label) {
        showSuccess("Please fill in all fields");
        return;
      }

      // Check for duplicate column
      const exists = reportState.customFilters.some(
        (f) => f.column === column && f.table === table,
      );
      if (exists) {
        showSuccess("This filter already exists");
        return;
      }

      // Add to state
      const newFilter = {
        column,
        table,
        label,
        type,
      };
      if (defaultValue) {
        newFilter.defaultValue = defaultValue;
      }
      if (noDefault) {
        newFilter.noDefault = true;
      }
      if (hideNone) {
        newFilter.hideNone = true;
      }
      reportState.customFilters.push(newFilter);

      // Reset form
      customFilterSchema.value = "";
      customFilterTable.innerHTML = '<option value="">Select table...</option>';
      customFilterTable.disabled = true;
      customFilterColumn.innerHTML =
        '<option value="">Select column...</option>';
      customFilterColumn.disabled = true;
      customFilterLabel.value = "";
      customFilterType.value = "select";
      const noDefaultCheckbox = document.getElementById(
        "custom-filter-no-default",
      );
      if (noDefaultCheckbox) noDefaultCheckbox.checked = false;
      const hideNoneCheckbox = document.getElementById(
        "custom-filter-hide-none",
      );
      if (hideNoneCheckbox) hideNoneCheckbox.checked = false;
      resetCustomFilterDefault();
      syncCustomFilterDefaultVisibility();

      // Re-render list
      renderCustomFiltersList();
      markDirty();
      showSuccess("Custom filter added");
    });
  }

  // Save component button
  elements.saveComponentBtn.addEventListener("click", (e) => {
    e.preventDefault();
    saveComponent();
  });

  // Preview tab buttons
  const refreshPreviewBtn = document.getElementById("btn-refresh-preview");
  if (refreshPreviewBtn) {
    refreshPreviewBtn.addEventListener("click", refreshPreview);
  }

  const gotoSectionsBtn = document.getElementById("btn-goto-sections");
  if (gotoSectionsBtn) {
    gotoSectionsBtn.addEventListener("click", () => showTab("sections"));
  }

  // Publish tab buttons (two-step: validate then publish)
  const validatePublishBtn = document.getElementById("btn-validate-publish");
  if (validatePublishBtn) {
    validatePublishBtn.addEventListener("click", validateForPublish);
  }
  const confirmPublishBtn = document.getElementById("btn-confirm-publish");
  if (confirmPublishBtn) {
    confirmPublishBtn.addEventListener("click", confirmPublish);
  }
}

/**
 * Initialize the application
 */
async function init() {
  // Initialize authentication first (Keycloak login if required)
  // This sets up the auth token provider for apiCache
  try {
    const authenticated = await initAuth();
    console.log("[Builder] Auth initialized, authenticated:", authenticated);
  } catch (err) {
    console.error("[Builder] Auth initialization failed:", err);
    // Show error to user - they need to authenticate
    document.body.innerHTML =
      '<div style="padding: 2rem; text-align: center;">' +
      "<h1>Authentication Required</h1>" +
      "<p>Failed to initialize authentication. Please refresh the page.</p>" +
      '<p style="color: #666;">' +
      err.message +
      "</p>" +
      "</div>";
    return;
  }

  // Check auth config to determine if publishing is allowed
  try {
    const authConfig = await getAuthConfig();
    setPublishingAllowed(authConfig.canPublish !== false);
    if (!isPublishingAllowed()) {
      disablePublishTab();
      console.log("[Builder] Publishing disabled (production mode)");
    }
  } catch (err) {
    console.warn(
      "[Builder] Failed to get auth config, assuming publishing allowed:",
      err,
    );
  }

  // Initialize DOM elements
  initElements();

  // Set up state change callback for tab badges
  setOnStateChange(() => {
    updateTabBadges();
  });

  // Set up callbacks to avoid circular dependencies
  setSectionsCallbacks({
    showComponentModal,
    updatePreview,
  });

  setComponentsCallbacks({
    updatePreview,
    renderSections: () => {
      renderSections();
      markDirty();
      updateTabBadges();
    },
    previewQueryData,
  });

  setYamlCallbacks({
    showTab,
  });

  setPreviewCallbacks({
    showTab,
    showComponentModal,
  });

  setPublishCallbacks({
    showTab,
    markClean, // Mark as clean after successful publish
  });

  // Load schemas and tables
  await loadAvailableSchemas();
  await loadAvailableTables(reportState.selectedSchema);

  // Setup event listeners
  setupEventListeners();

  // Initialize custom filter schema dropdown (after schemas are loaded)
  initCustomFilterSchemas();

  // Render initial custom filters list
  renderCustomFiltersList();

  // Initialize keyword suggestions
  const keywordsInput = document.getElementById("report-keywords");
  const suggestionsContainer = document.getElementById("keywords-suggestions");
  if (keywordsInput && suggestionsContainer) {
    initKeywordSuggestions(keywordsInput, suggestionsContainer, () => {
      // Get category from report ID path (e.g., "Programs/Malaria/report" -> "Programs/Malaria")
      const id = reportState.id || "";
      const parts = id.split("/");
      return parts.length > 1 ? parts.slice(0, -1).join("/") : "";
    });
  }

  // Set up unsaved changes warning
  window.addEventListener("beforeunload", (e) => {
    if (reportState.isDirty) {
      e.preventDefault();
      e.returnValue =
        "You have unsaved changes. Are you sure you want to leave?";
      return e.returnValue;
    }
  });

  // Initial tab badge update
  updateTabBadges();

  // Show metadata tab (first tab)
  showTab("metadata");
}

/**
 * Initialize custom filter schema dropdown
 */
function initCustomFilterSchemas() {
  const customFilterSchema = document.getElementById("custom-filter-schema");
  if (customFilterSchema && reportState.availableSchemas.length > 0) {
    customFilterSchema.innerHTML = '<option value="">Select schema...</option>';
    reportState.availableSchemas.forEach((schema) => {
      const option = document.createElement("option");
      option.value = schema;
      option.textContent = schema;
      customFilterSchema.appendChild(option);
    });
  }
}

// Initialize when DOM is ready
document.addEventListener("DOMContentLoaded", init);
