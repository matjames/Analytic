// YAML Handler Module
// YAML generation, import, and parsing

import { elements, reportState, markClean } from "./state.js";
import {
  generateReportYAML,
  parseReportYAML,
  loadAvailableTables,
} from "./api.js";
import { showError, showSuccess, showCopyFeedback } from "./utils.js";
import { validateComponents } from "./validation.js";

// Callbacks
let onShowTab = () => {};

export function setCallbacks(callbacks) {
  if (callbacks.showTab) onShowTab = callbacks.showTab;
}

/**
 * Populate all builder form fields and reportState from a parsed report object.
 * Shared by both importYAML (Import YAML) and selectReportForEdit (Edit Existing).
 */
export async function populateFormFromReport(report) {
  elements.reportId.value = report.id;
  elements.reportTitle.value = report.title;
  if (elements.reportDatasource)
    elements.reportDatasource.value = report.datasource || "";
  if (elements.reportKeywords)
    elements.reportKeywords.value = report.keywords || "";
  if (elements.reportAuthor) elements.reportAuthor.value = report.author || "";

  // Load filters
  const filters = report.filters || [];
  elements.filterCheckboxes.forEach((cb) => {
    if (cb.id === "filter-facility") {
      cb.checked =
        filters.includes("facility") || filters.includes("facility_select");
    } else {
      cb.checked = filters.includes(cb.value);
    }
  });
  reportState.filters = filters;

  // Set facility type radio and show/hide sub-options
  const facilityChecked =
    filters.includes("facility") || filters.includes("facility_select");
  const facilityTypeOptions = document.getElementById("facility-type-options");
  if (facilityTypeOptions)
    facilityTypeOptions.style.display = facilityChecked ? "block" : "none";
  if (facilityChecked) {
    const radioId = filters.includes("facility_select")
      ? "facility-type-single"
      : "facility-type-multi";
    const radio = document.getElementById(radioId);
    if (radio) radio.checked = true;
  }

  // Load timeColumns
  if (report.timeColumns) {
    const yearEl = document.getElementById("time-column-year");
    const monthEl = document.getElementById("time-column-month");
    const weekEl = document.getElementById("time-column-week");
    const quarterEl = document.getElementById("time-column-quarter");
    if (yearEl) yearEl.value = report.timeColumns.year || "";
    if (monthEl) monthEl.value = report.timeColumns.month || "";
    if (weekEl) weekEl.value = report.timeColumns.week || "";
    if (quarterEl) quarterEl.value = report.timeColumns.quarter || "";
  }

  // Expand time filters section if any time filters or columns are set
  if (
    filters.some((f) => ["year", "month", "week", "quarter"].includes(f)) ||
    (report.timeColumns &&
      (report.timeColumns.year ||
        report.timeColumns.month ||
        report.timeColumns.week ||
        report.timeColumns.quarter))
  ) {
    const timeContent = document.getElementById("time-filters-content");
    const timeHeader = document.getElementById("time-filters-header");
    if (timeContent) timeContent.style.display = "block";
    if (timeHeader) {
      const icon = timeHeader.querySelector(".collapse-icon");
      if (icon) icon.textContent = "▼";
    }
  }

  // Load locationColumns
  if (report.locationColumns) {
    const districtEl = document.getElementById("location-column-district");
    const regionEl = document.getElementById("location-column-region");
    const facilityEl = document.getElementById("location-column-facility");
    if (districtEl) districtEl.value = report.locationColumns.district || "";
    if (regionEl) regionEl.value = report.locationColumns.region || "";
    if (facilityEl) facilityEl.value = report.locationColumns.facility || "";
  }

  // Expand location filters section if any location filters or columns are set
  if (
    filters.some((f) =>
      ["district", "region", "facility", "facility_select"].includes(f),
    ) ||
    (report.locationColumns &&
      (report.locationColumns.district ||
        report.locationColumns.region ||
        report.locationColumns.facility))
  ) {
    const locationContent = document.getElementById("location-filters-content");
    const locationHeader = document.getElementById("location-filters-header");
    if (locationContent) locationContent.style.display = "block";
    if (locationHeader) {
      const icon = locationHeader.querySelector(".collapse-icon");
      if (icon) icon.textContent = "▼";
    }
  }

  // Expand custom filters section if there are custom filters
  if (report.customFilters && report.customFilters.length > 0) {
    const customContent = document.getElementById("custom-filters-content");
    const customHeader = document.getElementById("custom-filters-header");
    if (customContent) customContent.style.display = "block";
    if (customHeader) {
      const icon = customHeader.querySelector(".collapse-icon");
      if (icon) icon.textContent = "▼";
    }
  }

  // Update reportState
  reportState.id = report.id;
  reportState.title = report.title;
  reportState.keywords = report.keywords || "";
  reportState.author = report.author || "";
  reportState.datasource = report.datasource || "";
  reportState.sections = (report.sections || []).map((s) => ({
    ...s,
    _userLayout: !!s.layout,
  }));
  reportState.timeColumns = report.timeColumns || {};
  reportState.locationColumns = report.locationColumns || {};
  reportState.customFilters = report.customFilters || [];

  // Render custom filters list
  renderCustomFiltersList();

  // Reload available tables and register any from component queries
  await loadAvailableTables();
  for (const section of report.sections || []) {
    for (const component of section.components || []) {
      if (component.query && component.query.table) {
        const t = component.query.table;
        if (
          !reportState.availableTables.some(
            (existing) => existing.toLowerCase() === t.toLowerCase(),
          )
        ) {
          reportState.availableTables.push(t);
        }
      }
    }
  }
}

/**
 * Build a clean report object for YAML serialization
 */
export function buildReportObject() {
  const report = {
    id: reportState.id,
    title: reportState.title,
  };

  // Add datasource (named .env connection) if selected
  if (reportState.datasource) {
    report.datasource = reportState.datasource;
  }

  // Add keywords if provided (for search discoverability)
  if (reportState.keywords) {
    report.keywords = reportState.keywords;
  }

  // Add author if provided (for audit trail - not displayed)
  if (reportState.author) {
    report.author = reportState.author;
  }
  // Note: category is not included - it's determined by folder location

  if (reportState.filters.length > 0) {
    report.filters = reportState.filters;
  }

  // Add timeColumns only if any are set
  const hasTimeColumns =
    reportState.timeColumns &&
    (reportState.timeColumns.year ||
      reportState.timeColumns.month ||
      reportState.timeColumns.week ||
      reportState.timeColumns.quarter);
  if (hasTimeColumns) {
    report.timeColumns = {};
    if (reportState.timeColumns.year)
      report.timeColumns.year = reportState.timeColumns.year;
    if (reportState.timeColumns.month)
      report.timeColumns.month = reportState.timeColumns.month;
    if (reportState.timeColumns.week)
      report.timeColumns.week = reportState.timeColumns.week;
    if (reportState.timeColumns.quarter)
      report.timeColumns.quarter = reportState.timeColumns.quarter;
  }

  // Add locationColumns only if any are set
  const hasLocationColumns =
    reportState.locationColumns &&
    (reportState.locationColumns.district ||
      reportState.locationColumns.region ||
      reportState.locationColumns.facility);
  if (hasLocationColumns) {
    report.locationColumns = {};
    if (reportState.locationColumns.district)
      report.locationColumns.district = reportState.locationColumns.district;
    if (reportState.locationColumns.region)
      report.locationColumns.region = reportState.locationColumns.region;
    if (reportState.locationColumns.facility)
      report.locationColumns.facility = reportState.locationColumns.facility;
  }

  // Add customFilters if any are defined
  if (reportState.customFilters && reportState.customFilters.length > 0) {
    report.customFilters = reportState.customFilters.map((cf) => {
      const filter = {
        column: cf.column,
        table: cf.table,
        label: cf.label,
        type: cf.type,
      };
      if (cf.type === "select" && cf.defaultValue) {
        filter.defaultValue = cf.defaultValue;
      }
      if (cf.noDefault) {
        filter.noDefault = true;
      }
      if (cf.hideNone) {
        filter.hideNone = true;
      }
      return filter;
    });
  }

  // Build sections with components
  report.sections = reportState.sections.map((section) => {
    const sectionObj = {
      id: section.id,
      title: section.title,
    };

    // Only include description if set
    if (section.description) {
      sectionObj.description = section.description;
    }

    sectionObj.layout = section.layout;
    sectionObj.components = section.components.map((comp) => {
      const compObj = {
        type: comp.type,
      };

      if (comp.title) compObj.title = comp.title;
      if (comp.description) compObj.description = comp.description;
      if (comp.content) compObj.content = comp.content;
      if (comp.infoboxBody) compObj.infoboxBody = comp.infoboxBody;
      if (comp.accentColor) compObj.accentColor = comp.accentColor;
      if (comp.thresholdColumn) compObj.thresholdColumn = comp.thresholdColumn;
      if (comp.thresholds && comp.thresholds.length > 0)
        compObj.thresholds = comp.thresholds;
      if (comp.unit) compObj.unit = comp.unit;
      if (comp.tone) compObj.tone = comp.tone;
      if (comp.lowerIsBetter) compObj.lowerIsBetter = comp.lowerIsBetter;
      if (comp.valueTemplate) compObj.valueTemplate = comp.valueTemplate;

      // Bar chart options
      if (comp.stacked) compObj.stacked = true;
      if (comp.horizontal) compObj.horizontal = true;
      if (comp.showValues) compObj.showValues = true;
      if (comp.showValuesWithAxis) compObj.showValuesWithAxis = true;
      const showAxisTitles =
        comp.showAxisTitles !== undefined
          ? comp.showAxisTitles
          : !comp.hideAxes;
      compObj.showAxisTitles = showAxisTitles;

      // Trend line for bar/line charts
      if (comp.trendLine) {
        compObj.trendLine = true;
        if (comp.trendLineColor) compObj.trendLineColor = comp.trendLineColor;
      }

      // Period comparison for text/KPI components
      if (comp.comparison) compObj.comparison = comp.comparison;

      // Target comparison for text/KPI components
      if (comp.target && comp.target.value !== undefined) {
        compObj.target = { value: comp.target.value };
        if (comp.target.label) {
          compObj.target.label = comp.target.label;
        }
      }

      // Axis labels for bar/line/bar_line charts
      if (
        comp.axisLabels &&
        (comp.axisLabels.x || comp.axisLabels.y || comp.axisLabels.y1)
      ) {
        compObj.axisLabels = {};
        if (comp.axisLabels.x) compObj.axisLabels.x = comp.axisLabels.x;
        if (comp.axisLabels.y) compObj.axisLabels.y = comp.axisLabels.y;
        if (comp.axisLabels.y1) compObj.axisLabels.y1 = comp.axisLabels.y1;
      }

      if (comp.axisTooltips && (comp.axisTooltips.x || comp.axisTooltips.y)) {
        compObj.axisTooltips = {};
        if (comp.axisTooltips.x) compObj.axisTooltips.x = comp.axisTooltips.x;
        if (comp.axisTooltips.y) compObj.axisTooltips.y = comp.axisTooltips.y;
      }

      if (comp.axisFormulas && (comp.axisFormulas.x || comp.axisFormulas.y)) {
        compObj.axisFormulas = {};
        if (comp.axisFormulas.x) compObj.axisFormulas.x = comp.axisFormulas.x;
        if (comp.axisFormulas.y) compObj.axisFormulas.y = comp.axisFormulas.y;
      }

      // Y-axis limit for bar/line/bar_line charts
      if (comp.yLimit) {
        const yl = {};
        if (typeof comp.yLimit.min === "number") yl.min = comp.yLimit.min;
        if (typeof comp.yLimit.max === "number") yl.max = comp.yLimit.max;
        if (comp.yLimit.y1) {
          const y1 = {};
          if (typeof comp.yLimit.y1.min === "number")
            y1.min = comp.yLimit.y1.min;
          if (typeof comp.yLimit.y1.max === "number")
            y1.max = comp.yLimit.y1.max;
          if (Object.keys(y1).length > 0) yl.y1 = y1;
        }
        if (Object.keys(yl).length > 0) compObj.yLimit = yl;
      }

      // Reference lines for bar/line/bar_line charts
      if (comp.referenceLines && comp.referenceLines.length > 0) {
        compObj.referenceLines = comp.referenceLines.map((line) => ({
          value: line.value,
          label: line.label,
          color: line.color,
          style: line.style,
        }));
      }

      if (comp.heatmap) {
        if (
          Array.isArray(comp.heatmap.rules) &&
          comp.heatmap.rules.length > 0
        ) {
          compObj.heatmap = {
            rules: comp.heatmap.rules.map((r) => ({
              column: r.column,
              thresholds: r.thresholds,
              colors: r.colors,
            })),
          };
        } else if (comp.heatmap.scale) {
          compObj.heatmap = { scale: comp.heatmap.scale };
          if (
            Array.isArray(comp.heatmap.columns) &&
            comp.heatmap.columns.length > 0
          ) {
            compObj.heatmap.columns = comp.heatmap.columns.map((c) =>
              typeof c === "string" ? c : { name: c.name, scale: c.scale },
            );
          }
        }
      }

      if (comp.headerTooltips && Object.keys(comp.headerTooltips).length > 0) {
        compObj.headerTooltips = { ...comp.headerTooltips };
      }

      if (comp.headerFormulas && Object.keys(comp.headerFormulas).length > 0) {
        compObj.headerFormulas = { ...comp.headerFormulas };
      }

      if (comp.tableHeaderWrap) {
        compObj.tableHeaderWrap = true;
      }

      if (comp.tableHeaderStyle && comp.tableHeaderStyle !== "default") {
        compObj.tableHeaderStyle = comp.tableHeaderStyle;
      }

      if (comp.tableCellBoundaries || comp.tableVerticalLines) {
        compObj.tableCellBoundaries = true;
        compObj.tableVerticalLines = true;
      }

      const pageSize = Math.floor(Number(comp.pageSize));
      if (Number.isFinite(pageSize) && pageSize > 0 && pageSize !== 20) {
        compObj.pageSize = pageSize;
      }

      if (Array.isArray(comp.splitColumns) && comp.splitColumns.length > 0) {
        compObj.splitColumns = comp.splitColumns.map((g) => ({
          parent: g.parent,
          children: Array.isArray(g.children) ? [...g.children] : [],
        }));
      }

      if (comp.splitChildHeaderMode) {
        compObj.splitChildHeaderMode = comp.splitChildHeaderMode;
      }

      // firstRowIsHeader for table_advanced (legacy)
      if (comp.firstRowIsHeader) compObj.firstRowIsHeader = true;

      // Pivot configuration for table_advanced
      if (
        comp.pivot &&
        comp.pivot.rows &&
        comp.pivot.columns &&
        comp.pivot.values
      ) {
        const pv = {
          rows: comp.pivot.rows,
          columns: comp.pivot.columns,
          values: comp.pivot.values,
        };
        if (comp.pivot.format && comp.pivot.format !== "auto")
          pv.format = comp.pivot.format;
        if (
          Array.isArray(comp.pivot.extraColumns) &&
          comp.pivot.extraColumns.length > 0
        )
          pv.extraColumns = comp.pivot.extraColumns.map((e) => ({
            label: e.label,
            formula: e.formula,
          }));
        if (
          Array.isArray(comp.pivot.extraRows) &&
          comp.pivot.extraRows.length > 0
        )
          pv.extraRows = comp.pivot.extraRows.map((e) => ({
            label: e.label,
            formula: e.formula,
          }));
        compObj.pivot = pv;
      }

      if (comp.facet) {
        compObj.facet = {
          periodType: comp.facet.periodType,
          count: comp.facet.count,
        };
      }

      if (comp.valueType === "categorical") {
        compObj.valueType = "categorical";
      }

      // Handle table_advanced and choropleth with raw SQL
      if (comp.sql) {
        compObj.sql = comp.sql;
      }
      // Handle choropleth columnMapping for custom SQL
      if (comp.columnMapping) {
        compObj.columnMapping = {
          district: comp.columnMapping.district,
          value: comp.columnMapping.value,
        };
      }
      // Note: filterTable is auto-detected from SQL by backend

      if (
        comp.data &&
        (comp.data.datasets || comp.data.geojson_path || comp.data.legend)
      ) {
        compObj.data = {};

        if (comp.data.datasets && comp.data.datasets.length > 0) {
          compObj.data.datasets = comp.data.datasets.map((ds) => ({
            label: ds.label,
            data: [],
            ...(ds.color && { color: ds.color }),
            ...(ds.chartType && { chartType: ds.chartType }),
          }));
        }

        if (comp.data.geojson_path) {
          compObj.data.geojson_path = comp.data.geojson_path;
          compObj.data.district_values = {};
          if (comp.data.color_scheme && comp.data.color_scheme.length > 0) {
            compObj.data.color_scheme = comp.data.color_scheme;
          }
          if (comp.data.bin_labels && comp.data.bin_labels.length > 0) {
            compObj.data.bin_labels = comp.data.bin_labels;
          }
          if (comp.data.legend_title) {
            compObj.data.legend_title = comp.data.legend_title;
          }
          if (comp.data.bin_edges && comp.data.bin_edges.length > 0) {
            compObj.data.bin_edges = comp.data.bin_edges;
          }
        }

        if (comp.data.legend) {
          compObj.data.legend = {
            show: comp.data.legend.show,
            position: comp.data.legend.position || "top",
          };
        }
      }

      if (comp.query) {
        compObj.query = {
          table: comp.query.table,
        };

        if (comp.query.where) {
          compObj.query.where = comp.query.where;
        }

        if (comp.query.aggregations && comp.query.aggregations.length > 0) {
          compObj.query.aggregations = comp.query.aggregations.map((agg) => {
            const aggObj = {
              column: agg.column,
              function: agg.function,
              alias: agg.alias,
            };
            // Always preserve color if set
            if (agg.color) {
              aggObj.color = agg.color;
            }
            if (agg.roundTo !== undefined && agg.roundTo !== null) {
              aggObj.roundTo = agg.roundTo;
            }
            return aggObj;
          });
        }

        if (comp.query.groupBy && comp.query.groupBy.length > 0) {
          compObj.query.groupBy = comp.query.groupBy.map((gb) => {
            const gbObj = { field: gb.field };
            if (gb.format) gbObj.format = gb.format;
            if (gb.alias) gbObj.alias = gb.alias;
            return gbObj;
          });
        }

        if (comp.query.calculate && comp.query.calculate.length > 0) {
          compObj.query.calculate = comp.query.calculate.map((calc) => {
            const calcObj = { formula: calc.formula };
            if (calc.roundTo !== undefined) calcObj.roundTo = calc.roundTo;
            if (calc.whenZero !== undefined) calcObj.whenZero = calc.whenZero;
            if (calc.resultAlias) calcObj.resultAlias = calc.resultAlias;
            if (calc.color) calcObj.color = calc.color;
            return calcObj;
          });
        }

        if (comp.query.orderBy && comp.query.orderBy.length > 0) {
          compObj.query.orderBy = comp.query.orderBy.map((ob) => {
            const obObj = { field: ob.field };
            if (ob.direction) obObj.direction = ob.direction;
            return obObj;
          });
        }

        if (comp.query.selectColumns && comp.query.selectColumns.length > 0) {
          compObj.query.selectColumns = comp.query.selectColumns;
        }

        if (comp.query.transpose && comp.query.transpose.enabled) {
          compObj.query.transpose = {
            enabled: true,
          };
          // Include rowLabel if specified
          if (comp.query.transpose.rowLabel) {
            compObj.query.transpose.rowLabel = comp.query.transpose.rowLabel;
          }
        }

        if (comp.query.periodLimit && comp.query.periodLimit > 0) {
          compObj.query.periodLimit = comp.query.periodLimit;
        }
      }

      return compObj;
    });

    return sectionObj;
  });

  return report;
}

/**
 * Generate YAML via backend API and show in modal
 */
export async function generateYAML() {
  if (!validateComponents()) return;

  const request = {
    id: reportState.id,
    title: reportState.title,
    datasource: reportState.datasource || "",
    keywords: reportState.keywords || "", // For search discoverability
    // Note: category is determined by folder location, not YAML field
    filters: reportState.filters,
    customFilters: reportState.customFilters,
    timeColumns: reportState.timeColumns,
    locationColumns: reportState.locationColumns,
    sections: reportState.sections,
  };

  try {
    const result = await generateReportYAML(request);

    if (result.error) {
      const errorMsg = result.details
        ? `${result.error}: ${result.details}`
        : result.error;
      showError(errorMsg);
      return;
    }

    elements.yamlOutput.value = result.yaml;
    elements.yamlModal.classList.remove("hidden");

    // Show SQL processing feedback if any table_advanced components were processed
    if (result.sqlProcessed && result.sqlProcessed.length > 0) {
      showSQLProcessingFeedback(result.sqlProcessed);
    }
  } catch (err) {
    showError("Failed to generate YAML: " + err.message);
  }
}

/**
 * Show feedback about SQL processing (replacements, detected tables, etc.)
 */
function showSQLProcessingFeedback(sqlProcessed) {
  const messages = [];

  for (const info of sqlProcessed) {
    const parts = [];

    if (info.detectedTable) {
      parts.push(`Table: ${info.detectedTable}`);
    }

    if (info.replacements && info.replacements.length > 0) {
      parts.push(`Replaced: ${info.replacements.join(", ")}`);
    }

    if (info.filtersApplied && info.filtersApplied.length > 0) {
      parts.push(`Filters: ${info.filtersApplied.join(", ")}`);
    }

    if (info.warnings && info.warnings.length > 0) {
      parts.push(`Warnings: ${info.warnings.join(", ")}`);
    }

    if (parts.length > 0) {
      messages.push(
        `Component ${info.componentIndex + 1}: ${parts.join(" | ")}`,
      );
    }
  }

  if (messages.length > 0) {
    showSuccess("SQL processed: " + messages.join("; "));
  }
}

/**
 * Import YAML from input
 */
export async function importYAML() {
  const yaml = elements.importYaml.value.trim();
  if (!yaml) {
    showError("Please paste YAML content");
    return;
  }

  try {
    const result = await parseReportYAML(yaml);

    if (!result.valid) {
      showError("Invalid YAML: " + result.errors.join(", "));
      return;
    }

    // Load report into form
    const report = result.report;
    await populateFormFromReport(report);

    elements.importYaml.value = "";
    if (elements.importYamlModal) {
      elements.importYamlModal.classList.add("hidden");
    }

    // Mark as clean since we just loaded
    markClean();

    onShowTab("metadata");
    showSuccess("YAML imported successfully");
  } catch (err) {
    showError("Failed to parse YAML: " + err.message);
  }
}

/**
 * Close YAML modal
 */
export function closeYamlModal() {
  elements.yamlModal.classList.add("hidden");
}

/**
 * Copy YAML to clipboard
 */
export function copyYAMLToClipboard() {
  const yaml = elements.yamlOutput.value;
  navigator.clipboard
    .writeText(yaml)
    .then(() => {
      showCopyFeedback("yaml-copy-feedback");
    })
    .catch((err) => {
      showError("Failed to copy: " + err);
    });
}

/**
 * Download YAML file
 */
export function downloadYAML() {
  const yaml = elements.yamlOutput.value;
  const filename = `${reportState.id.replace(/\//g, "-")}.yaml`;
  const blob = new Blob([yaml], { type: "text/yaml" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

/**
 * Render the custom filters list in the UI
 */
export function renderCustomFiltersList() {
  const listContainer = document.getElementById("custom-filters-list");
  if (!listContainer) return;

  listContainer.innerHTML = "";

  if (reportState.customFilters.length === 0) {
    listContainer.innerHTML =
      '<p style="color: var(--color-text-muted); font-size: 0.85em; font-style: italic;">No custom filters defined yet.</p>';
    return;
  }

  reportState.customFilters.forEach((filter, index) => {
    const filterItem = document.createElement("div");
    filterItem.className = "custom-filter-item";
    filterItem.style.cssText =
      "display: flex; align-items: center; justify-content: space-between; padding: 10px; background: var(--color-bg); border: 1px solid var(--color-border); border-radius: 6px; margin-bottom: 8px;";

    const filterInfo = document.createElement("div");
    const displayDefaultText =
      filter.type === "select" && filter.defaultValue
        ? ` • Default: ${filter.defaultValue}`
        : filter.noDefault
          ? " • starts blank"
          : filter.hideNone
            ? ' • no "None" option'
            : "";
    filterInfo.innerHTML = `
            <strong style="font-size: 0.9em;">${filter.label}</strong>
            <div style="font-size: 0.8em; color: var(--color-text-muted);">
                ${filter.table}.${filter.column} • ${filter.type === "multiselect" ? "Multi-Select" : "Single Select"}${displayDefaultText}
            </div>
        `;

    const removeBtn = document.createElement("button");
    removeBtn.type = "button";
    removeBtn.className = "btn-small btn-danger";
    removeBtn.textContent = "✕";
    removeBtn.title = "Remove filter";
    removeBtn.style.cssText = "padding: 4px 8px; font-size: 0.8em;";
    removeBtn.addEventListener("click", () => {
      reportState.customFilters.splice(index, 1);
      renderCustomFiltersList();
    });

    filterItem.appendChild(filterInfo);
    filterItem.appendChild(removeBtn);
    listContainer.appendChild(filterItem);
  });
}
