// Components Module
// Component modal, form rendering, and save functionality

// Returns a hint string listing custom filter placeholders defined on the report
function customFilterHint() {
  const cfs = reportState.customFilters;
  if (!cfs || cfs.length === 0) return "";
  const names = cfs.map(
    (cf) => `{{custom_filter_${cf.column}}} (${cf.label || cf.column})`,
  );
  return ` Custom filters: ${names.join(", ")}.`;
}

import { elements, reportState, currentEditingComponent } from "./state.js";
import {
  loadTableColumns,
  loadTablePreview,
  loadAvailableTables,
  previewAdvancedSQL,
} from "./api.js";
import { showError, escapeHtml } from "./utils.js";
import {
  addRecentTable,
  renderTableOptionsWithRecent,
} from "./recent-tables.js";
import {
  addAggregation,
  addCalculation,
  addGroupBy,
  renderCalculationRow,
  renderSelectColumnsCheckboxes,
  refreshSelectColumnsList,
  autoExpandCalculateIfNeeded,
  autoExpandOrderByIfNeeded,
  autoExpandPeriodLimitIfNeeded,
  autoExpandBarOptionsIfNeeded,
  autoExpandLegendIfNeeded,
  autoExpandTransposeIfNeeded,
  loadTransposeConfig,
  setupTransposeListeners,
  setupFacetListeners,
  populateColumnSuggestions,
  maybeSetDefaultPeriodLimit,
  getAvailableColumns,
  refreshCalcAliasChips,
  wireUpExistingCalcRows,
} from "./query-form.js";
import { initAllAutocomplete, refreshAutocomplete } from "./autocomplete.js";

// Callbacks set from main.js
let onUpdatePreview = () => {};
let onRenderSections = () => {};
let onPreviewQueryData = () => {};

function normalizeTablePageSize(value, fallback = 20) {
  const parsed = Math.floor(Number(value));
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function readTablePageSizeFromForm() {
  const input = document.getElementById("table-page-size");
  if (input) {
    return normalizeTablePageSize(input.value);
  }
  return normalizeTablePageSize(currentEditingComponent?.component?.pageSize);
}

export function setCallbacks(callbacks) {
  if (callbacks.updatePreview) onUpdatePreview = callbacks.updatePreview;
  if (callbacks.renderSections) onRenderSections = callbacks.renderSections;
  if (callbacks.previewQueryData)
    onPreviewQueryData = callbacks.previewQueryData;
}

/**
 * Populate column dropdowns for a table
 */
export async function populateColumnDropdowns(table) {
  await loadTableColumns(table);

  // Update column suggestions for WHERE, aggregations, and groupBy autocomplete
  populateColumnSuggestions();

  // Render sample data preview (fire-and-forget, no await needed)
  renderSampleDataPanel(table);

  // Restore saved groupBy values on text inputs
  const savedGroupBy = currentEditingComponent?.component?.query?.groupBy || [];
  document.querySelectorAll("#groupby-list .groupby-row").forEach((row, i) => {
    const fieldInput = row.querySelector(".groupby-field");
    const formatSelect = row.querySelector(".groupby-format");
    const aliasInput = row.querySelector(".groupby-alias");

    if (fieldInput && savedGroupBy[i] && savedGroupBy[i].field) {
      fieldInput.value = savedGroupBy[i].field;
    }

    // Restore format and alias (they should already be set from template, but ensure they are)
    if (formatSelect && savedGroupBy[i] && savedGroupBy[i].format) {
      formatSelect.value = savedGroupBy[i].format;
    }
    if (aliasInput && savedGroupBy[i] && savedGroupBy[i].alias) {
      aliasInput.value = savedGroupBy[i].alias;
    }
  });

  // Update select columns list to include groupBy aliases
  refreshSelectColumnsList();

  // Restore transpose config values if present
  const savedTranspose = currentEditingComponent?.component?.query?.transpose;
  if (savedTranspose) {
    const transposeEnabled = document.getElementById("transpose-enabled");
    const transposeOptions = document.getElementById("transpose-options");
    const rowLabel = document.getElementById("transpose-row-label");

    if (transposeEnabled) {
      transposeEnabled.checked = savedTranspose.enabled || false;
    }
    if (transposeOptions) {
      transposeOptions.style.opacity = savedTranspose.enabled ? "1" : "0.5";
    }
    if (rowLabel) {
      rowLabel.value = savedTranspose.rowLabel || "";
    }

    // Auto-expand transpose section if enabled
    if (savedTranspose.enabled) {
      const content = document.getElementById("transpose-content");
      const icon = document.getElementById("transpose-toggle-icon");
      if (content && icon) {
        content.style.display = "block";
        icon.innerHTML = "&#9660;";
        icon.textContent = "▼";
      }
      // Also expand select columns section
      const selectColsContent = document.getElementById("selectcols-content");
      const selectColsIcon = document.getElementById("selectcols-toggle-icon");
      if (selectColsContent && selectColsIcon) {
        selectColsContent.style.display = "block";
        selectColsIcon.textContent = "▼";
      }
    }
  }

  // Initialize autocomplete on column inputs
  initAllAutocomplete();
}

/**
 * Render sample data panel for the selected table
 */
async function renderSampleDataPanel(table) {
  const container = document.getElementById("sampledata-table-container");
  if (!container) return;

  container.innerHTML =
    '<p style="color: var(--color-text-secondary); font-size: 12px; padding: 12px; margin: 0;">Loading sample data...</p>';

  const data = await loadTablePreview(table);
  if (!data || !data.headers || data.headers.length === 0) {
    container.innerHTML =
      '<p style="color: var(--color-text-secondary); font-size: 12px; padding: 12px; margin: 0;">No data available.</p>';
    return;
  }

  const columns = reportState.availableColumns[table] || [];
  const typeMap = {};
  columns.forEach((col) => {
    typeMap[col.name] = col.type;
  });

  const headerCells = data.headers
    .map((h) => {
      const colType = typeMap[h] || "";
      const badge = colType
        ? `<span style="font-size: 9px; color: var(--color-text-muted); font-family: monospace; text-transform: uppercase; margin-left: 4px;">${colType}</span>`
        : "";
      return `<th style="padding: 6px 10px; text-align: left; font-size: 11px; font-weight: 600; color: var(--color-primary); cursor: pointer; white-space: nowrap; border-bottom: 2px solid var(--color-border); user-select: none;" data-column="${h}">${h}${badge}</th>`;
    })
    .join("");

  const bodyRows = (data.rows || [])
    .map((row) => {
      const cells = row
        .map((cell) => {
          const val =
            cell === null || cell === undefined
              ? '<span style="color: var(--color-text-muted); font-style: italic;">null</span>'
              : String(cell);
          return `<td style="padding: 5px 10px; font-size: 11px; max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; border-bottom: 1px solid var(--color-border-light);">${val}</td>`;
        })
        .join("");
      return `<tr>${cells}</tr>`;
    })
    .join("");

  container.innerHTML = `<table style="width: 100%; border-collapse: collapse; font-size: 11px;">
        <thead><tr>${headerCells}</tr></thead>
        <tbody>${bodyRows}</tbody>
    </table>`;

  // Wire up click-to-add on header cells
  container.querySelectorAll("th[data-column]").forEach((th) => {
    th.addEventListener("click", (e) => {
      handleSampleDataColumnClick(th.dataset.column, e);
    });
  });

  // Auto-expand the section on first load
  const content = document.getElementById("sampledata-content");
  const icon = document.getElementById("sampledata-toggle-icon");
  if (content && icon && content.style.display === "none") {
    content.style.display = "block";
    icon.textContent = "▼";
  }
}

/**
 * Handle click on a sample data column header — show popover to add as aggregation or groupBy
 */
function handleSampleDataColumnClick(columnName, event) {
  // Remove any existing popover
  const existing = document.getElementById("sampledata-popover");
  if (existing) existing.remove();

  const rect = event.target.getBoundingClientRect();

  const popover = document.createElement("div");
  popover.id = "sampledata-popover";
  popover.style.cssText = `position: fixed; top: ${rect.bottom + 4}px; left: ${rect.left}px; background: var(--color-card); border: 1px solid var(--color-border); border-radius: 6px; box-shadow: 0 4px 12px rgba(0,0,0,0.15); z-index: 10000; padding: 4px;`;
  popover.innerHTML = `
        <button id="sampledata-add-agg" style="display: block; width: 100%; padding: 8px 14px; border: none; background: transparent; cursor: pointer; text-align: left; font-size: 12px; border-radius: 4px; color: var(--color-text);">+ Aggregation</button>
        <button id="sampledata-add-groupby" style="display: block; width: 100%; padding: 8px 14px; border: none; background: transparent; cursor: pointer; text-align: left; font-size: 12px; border-radius: 4px; color: var(--color-text);">+ Group By</button>
    `;
  document.body.appendChild(popover);

  // Wire buttons
  popover.querySelector("#sampledata-add-agg").addEventListener("click", () => {
    addAggregation();
    const inputs = document.querySelectorAll("#aggregations-list .agg-column");
    const lastInput = inputs[inputs.length - 1];
    if (lastInput) lastInput.value = columnName;
    popover.remove();
  });

  popover
    .querySelector("#sampledata-add-groupby")
    .addEventListener("click", () => {
      addGroupBy();
      const inputs = document.querySelectorAll("#groupby-list .groupby-field");
      const lastInput = inputs[inputs.length - 1];
      if (lastInput) lastInput.value = columnName;
      popover.remove();
    });

  // Hover styles
  popover.querySelectorAll("button").forEach((btn) => {
    btn.addEventListener("mouseenter", () => {
      btn.style.background = "var(--color-hover)";
    });
    btn.addEventListener("mouseleave", () => {
      btn.style.background = "transparent";
    });
  });

  // Dismiss on outside click
  const dismiss = (e) => {
    if (!popover.contains(e.target)) {
      popover.remove();
      document.removeEventListener("click", dismiss);
    }
  };
  // Delay to avoid immediate dismiss from the click that opened it
  setTimeout(() => document.addEventListener("click", dismiss), 0);
}

/**
 * Show the component modal
 */
export async function showComponentModal(title) {
  // If the component has a table in a different schema, load the correct tables
  // before rendering so the table dropdown is accurate.
  const compTable = currentEditingComponent.component?.query?.table;
  if (compTable) {
    const compSchema = compTable.split(".")[0];
    if (compSchema !== reportState.selectedSchema) {
      reportState.selectedSchema = compSchema;
      await loadAvailableTables(compSchema);
    }
  }

  // Compute effective type (same mapping used by renderComponentForm)
  const comp = currentEditingComponent.component;
  let panelType = comp?.type || "bar";
  if (panelType === "text" && !comp?.query) panelType = "plain_text";

  document.getElementById("modal-title").textContent = title;

  // Populate left panel type picker
  const typePanelContainer = document.getElementById("component-type-panel");
  if (typePanelContainer) {
    typePanelContainer.innerHTML = renderTypePanelHTML(panelType);
  }

  // Populate right panel form body
  elements.componentFormContainer.innerHTML = renderComponentForm(comp);
  elements.componentModal.classList.remove("hidden");

  // Wire up form elements after rendering
  setTimeout(() => {
    // Wire up left-panel type picker items
    document.querySelectorAll(".type-panel-item").forEach((btn) => {
      btn.addEventListener("click", () => {
        document
          .querySelectorAll(".type-panel-item")
          .forEach((b) => b.classList.remove("selected"));
        btn.classList.add("selected");

        const typeInput = document.getElementById("comp-type");
        if (typeInput) typeInput.value = btn.dataset.type;

        updateFormFieldsVisibility();
        applySmartDefaults(btn.dataset.type);
        runInlineValidation();
      });
    });
    // (legacy grid .type-option — no-op now but kept for safety)
    // --- Axis Titles/Labels progressive disclosure wiring ---
    const showAxisTitlesCheckbox = document.getElementById(
      "bar-show-axis-titles",
    );
    const axisLabelsSection = document.getElementById("section-axislabels");
    if (showAxisTitlesCheckbox && axisLabelsSection) {
      // Initial state: show if checked
      if (showAxisTitlesCheckbox.checked) {
        axisLabelsSection.style.display = "block";
        // Focus first input
        const xInput = document.getElementById("axislabels-x");
        if (xInput) xInput.focus();
      } else {
        axisLabelsSection.style.display = "none";
      }
      showAxisTitlesCheckbox.addEventListener("change", (e) => {
        if (showAxisTitlesCheckbox.checked) {
          axisLabelsSection.style.display = "block";
          // Focus first input
          const xInput = document.getElementById("axislabels-x");
          if (xInput) xInput.focus();
        } else {
          axisLabelsSection.style.display = "none";
        }
      });
    }

    // Inline hint if axis labels missing and axis titles are shown
    if (showAxisTitlesCheckbox && axisLabelsSection) {
      const xInput = document.getElementById("axislabels-x");
      const yInput = document.getElementById("axislabels-y");
      const y1Input = document.getElementById("axislabels-y1");
      function updateAxisLabelsHint() {
        let missing = [];
        if (showAxisTitlesCheckbox.checked) {
          if (xInput && !xInput.value.trim()) missing.push("X-Axis");
          if (yInput && !yInput.value.trim()) missing.push("Y-Axis");
        }
        let hint = axisLabelsSection.querySelector(".axis-labels-hint");
        if (!hint && missing.length > 0) {
          hint = document.createElement("div");
          hint.className = "axis-labels-hint";
          hint.style.cssText =
            "color: #b45309; background: #fffbe8; border: 1px solid #fde68a; border-radius: 4px; padding: 7px 10px; margin-bottom: 10px; font-size: 12px;";
          axisLabelsSection.insertBefore(
            hint,
            axisLabelsSection.querySelector("h4")?.nextSibling,
          );
        }
        if (hint) {
          if (missing.length > 0) {
            hint.textContent = `Tip: Add ${missing.join(" and ")} label${missing.length > 1 ? "s" : ""} to help users interpret the chart.`;
            hint.style.display = "block";
          } else {
            hint.style.display = "none";
          }
        }
      }
      if (xInput) xInput.addEventListener("input", updateAxisLabelsHint);
      if (yInput) yInput.addEventListener("input", updateAxisLabelsHint);
      if (showAxisTitlesCheckbox)
        showAxisTitlesCheckbox.addEventListener("change", updateAxisLabelsHint);
      updateAxisLabelsHint();
    }

    // Keep value-label modes mutually exclusive for bar/bar_line charts.
    const showValuesCheckbox = document.getElementById("bar-show-values");
    const showValuesWithAxisCheckbox = document.getElementById(
      "bar-show-values-with-axis",
    );
    if (showValuesCheckbox && showValuesWithAxisCheckbox) {
      // Default mode is "Show Values" unless the y-axis-enabled mode is explicitly selected.
      if (!showValuesCheckbox.checked && !showValuesWithAxisCheckbox.checked) {
        showValuesCheckbox.checked = true;
      }

      // If both are somehow checked (legacy/stale state), prefer y-axis-enabled mode.
      if (showValuesCheckbox.checked && showValuesWithAxisCheckbox.checked) {
        showValuesCheckbox.checked = false;
      }

      showValuesCheckbox.addEventListener("change", () => {
        if (showValuesCheckbox.checked) {
          showValuesWithAxisCheckbox.checked = false;
        }
      });

      showValuesWithAxisCheckbox.addEventListener("change", () => {
        if (showValuesWithAxisCheckbox.checked) {
          showValuesCheckbox.checked = false;
        } else if (!showValuesCheckbox.checked) {
          // Always keep one mode selected; default back to Show Values.
          showValuesCheckbox.checked = true;
        }
      });
    }

    // Wire up tier toggle buttons
    document.querySelectorAll(".type-tier-toggle").forEach((toggle) => {
      toggle.addEventListener("click", () => {
        const tierIndex = toggle.dataset.tierIndex;
        const grid = document.querySelector(`[data-tier-grid="${tierIndex}"]`);
        const arrow = toggle.querySelector(".tier-arrow");
        if (grid && arrow) {
          const isHidden = grid.style.display === "none";
          grid.style.display = isHidden ? "grid" : "none";
          arrow.textContent = isHidden ? "▼" : "▶";
        }
      });
    });

    updateFormFieldsVisibility();

    // For infobox in query-builder mode, comp-type was set to 'text' to expose query builder
    // fields. Re-show the infobox body section and suppress comparison.
    if (
      document.querySelector(".type-panel-item.selected")?.dataset.type ===
      "infobox"
    ) {
      const infoboxSection = document.getElementById("section-infobox");
      if (infoboxSection) infoboxSection.style.display = "block";
      const compSection = document.getElementById("section-comparison");
      if (compSection) compSection.style.display = "none";
      // Infobox uses its own Body Text — never show the plain_text HTML Template.
      const contentSection = document.getElementById("section-content");
      if (contentSection) contentSection.style.display = "none";
    }

    // Auto-expand advanced options if editing component with advanced values
    autoExpandAdvancedIfNeeded();
    autoExpandCalculateIfNeeded();
    autoExpandOrderByIfNeeded();
    autoExpandPeriodLimitIfNeeded();
    autoExpandBarOptionsIfNeeded();
    autoExpandLegendIfNeeded();

    // Auto-expand WHERE section if component has a where clause
    if (currentEditingComponent?.component?.query?.where) {
      const whereContent = document.getElementById("where-content");
      const whereIcon = document.getElementById("where-toggle-icon");
      if (whereContent && whereIcon) {
        whereContent.style.display = "block";
        whereIcon.textContent = "▼";
      }
    }
    setupTransposeListeners();
    autoExpandTransposeIfNeeded();

    // Sync infobox color picker ↔ text input
    const infoboxColorPicker = document.getElementById("infobox-accent-color");
    const infoboxColorText = document.getElementById(
      "infobox-accent-color-text",
    );
    if (infoboxColorPicker && infoboxColorText) {
      infoboxColorPicker.addEventListener("input", () => {
        infoboxColorText.value = infoboxColorPicker.value;
      });
      infoboxColorText.addEventListener("input", () => {
        const val = infoboxColorText.value.trim();
        if (/^#[0-9a-fA-F]{6}$/.test(val)) {
          infoboxColorPicker.value = val;
        }
      });
    }

    // Threshold rules: wire existing rows + handle Add / Remove
    const thresholdList = document.getElementById("threshold-rules-list");
    if (thresholdList) {
      thresholdList.querySelectorAll(".threshold-rule-row").forEach((row) => {
        setupThresholdRowListeners(row);
      });
      thresholdList.addEventListener("click", (e) => {
        if (e.target.classList.contains("remove-threshold-rule")) {
          e.target.closest(".threshold-rule-row")?.remove();
        }
      });
    }
    const addThresholdBtn = document.getElementById("add-threshold-rule");
    if (addThresholdBtn) {
      addThresholdBtn.addEventListener("click", () => {
        const list = document.getElementById("threshold-rules-list");
        if (list) {
          list.insertAdjacentHTML("beforeend", renderThresholdRuleRow({}));
          const rows = list.querySelectorAll(".threshold-rule-row");
          setupThresholdRowListeners(rows[rows.length - 1]);
        }
      });
    }

    // Sync kpi card color picker ↔ text input
    const kpiColorPicker = document.getElementById("kpi-accent-color");
    const kpiColorText = document.getElementById("kpi-accent-color-text");
    if (kpiColorPicker && kpiColorText) {
      kpiColorPicker.addEventListener("input", () => {
        kpiColorText.value = kpiColorPicker.value;
      });
      kpiColorText.addEventListener("input", () => {
        const val = kpiColorText.value.trim();
        if (/^#[0-9a-fA-F]{6}$/.test(val)) {
          kpiColorPicker.value = val;
        }
      });
    }

    // Load existing transpose config if editing
    if (currentEditingComponent?.component?.query?.transpose) {
      loadTransposeConfig(currentEditingComponent.component.query.transpose);
    }

    const schemaSelect = document.getElementById("comp-schema");
    const tableSelect = document.getElementById("comp-table");

    // Wire up schema change
    if (schemaSelect) {
      schemaSelect.addEventListener("change", async () => {
        const schema = schemaSelect.value;
        reportState.selectedSchema = schema;
        await loadAvailableTables(schema);
        if (tableSelect) {
          tableSelect.innerHTML = renderTableOptionsWithRecent(
            reportState.availableTables,
            "",
          );
        }
      });
    }

    if (tableSelect) {
      tableSelect.addEventListener("change", async () => {
        const table = tableSelect.value;
        if (table) {
          // Track this table as recently used
          addRecentTable(table);
          await populateColumnDropdowns(table);
        }
      });

      if (tableSelect.value) {
        populateColumnDropdowns(tableSelect.value);
      }
    }

    // Wire up add aggregation button
    const addAggBtn = document.getElementById("btn-add-aggregation");
    if (addAggBtn) {
      addAggBtn.addEventListener("click", addAggregation);
    }

    // Wire up add calculation button
    const addCalcBtn = document.getElementById("btn-add-calculation");
    if (addCalcBtn) {
      addCalcBtn.addEventListener("click", addCalculation);
    }

    // Wire up add groupBy button
    const addGroupByBtn = document.getElementById("btn-add-groupby");
    if (addGroupByBtn) {
      addGroupByBtn.addEventListener("click", addGroupBy);
    }

    // Wire up existing groupBy remove buttons and change handlers
    document.querySelectorAll("#groupby-list .groupby-row").forEach((row) => {
      const removeBtn = row.querySelector(".btn-remove-groupby");
      if (removeBtn) {
        removeBtn.addEventListener("click", () => {
          row.remove();
          refreshSelectColumnsList();
        });
      }
      const aliasInput = row.querySelector(".groupby-alias");
      if (aliasInput) {
        aliasInput.addEventListener("change", () => {
          refreshSelectColumnsList();
        });
      }
      const fieldSelect = row.querySelector(".groupby-field");
      if (fieldSelect) {
        fieldSelect.addEventListener("change", () => {
          refreshSelectColumnsList();
        });
      }
      const formatSelect = row.querySelector(".groupby-format");
      if (formatSelect) {
        formatSelect.addEventListener("change", () => {
          maybeSetDefaultPeriodLimit(formatSelect.value);
        });
      }
    });

    // Wire up preview query button
    const previewQueryBtn = document.getElementById("btn-preview-query");
    if (previewQueryBtn) {
      previewQueryBtn.addEventListener("click", onPreviewQueryData);
    }

    // Wire up advanced SQL preview button
    const previewAdvancedSqlBtn = document.getElementById(
      "btn-preview-advanced-sql",
    );
    if (previewAdvancedSqlBtn) {
      previewAdvancedSqlBtn.addEventListener("click", handleAdvancedSQLPreview);
    }

    // Wire up choropleth SQL preview button
    const previewChoroplethSqlBtn = document.getElementById(
      "btn-preview-choropleth-sql",
    );
    if (previewChoroplethSqlBtn) {
      previewChoroplethSqlBtn.addEventListener(
        "click",
        handleChoroplethSQLPreview,
      );
    }

    // Wire up chart SQL preview button
    const previewChartSqlBtn = document.getElementById("btn-preview-chart-sql");
    if (previewChartSqlBtn) {
      previewChartSqlBtn.addEventListener("click", handleChartSQLPreview);
    }

    // Wire up chart SQL dataset mapping buttons (for bar_line)
    const addDsBtn = document.getElementById("btn-add-chart-sql-dataset");
    if (addDsBtn) {
      addDsBtn.addEventListener("click", () => addChartSqlDatasetRow());
    }
    const dsList = document.getElementById("chart-sql-datasets-list");
    if (dsList) {
      dsList.addEventListener("click", (e) => {
        if (e.target.classList.contains("btn-remove-chart-sql-ds")) {
          e.target.closest(".chart-sql-dataset-row")?.remove();
        }
      });
    }

    // Auto-expand chart SQL section if component has SQL and is a chart type
    const chartTypes = ["bar", "line", "bar_line", "pie", "pyramid"];
    if (
      currentEditingComponent?.component?.sql &&
      chartTypes.includes(currentEditingComponent?.component?.type)
    ) {
      const chartSqlContent = document.getElementById("chart-sql-content");
      const chartSqlIcon = document.getElementById("chart-sql-toggle-icon");
      if (chartSqlContent && chartSqlIcon) {
        chartSqlContent.style.display = "block";
        chartSqlIcon.textContent = "▼";
      }
      // Trigger the toggle to hide query sections
      window.toggleChartSqlMode();
    }

    // Auto-expand choropleth SQL section if component has SQL
    if (
      currentEditingComponent?.component?.sql &&
      currentEditingComponent?.component?.type === "choropleth"
    ) {
      const content = document.getElementById("choropleth-sql-content");
      const icon = document.getElementById("choropleth-sql-toggle-icon");
      if (content && icon) {
        content.style.display = "block";
        icon.textContent = "▼";
      }
      if (icon) icon.innerHTML = "&#9660;";
      storeChoroplethSQLPreviewHeaders([]);
      window.toggleChoroplethSqlMode();
    }

    // Wire up plain_text data source mode selector
    const textDataMode = document.getElementById("text-data-mode");
    const textSqlContent = document.getElementById("text-sql-content");
    if (textDataMode) {
      textDataMode.addEventListener("change", () => {
        const mode = textDataMode.value;
        if (textSqlContent)
          textSqlContent.style.display = mode === "sql" ? "block" : "none";
        const typeInput = document.getElementById("comp-type");
        const isInfoboxPanel =
          document.querySelector(".type-panel-item.selected")?.dataset.type ===
          "infobox";
        if (typeInput) {
          if (isInfoboxPanel) {
            // For infobox: switch to 'text' type to expose query-builder fields, keep infobox panel active.
            typeInput.value = mode === "builder" ? "text" : "infobox";
            updateFormFieldsVisibility();
            // Always keep the infobox body section visible.
            const infoboxSection = document.getElementById("section-infobox");
            if (infoboxSection) infoboxSection.style.display = "block";
            // Infobox doesn't use comparison.
            const compSection = document.getElementById("section-comparison");
            if (compSection) compSection.style.display = "none";
            // Infobox uses its own Body Text — never show the plain_text HTML Template.
            const contentSection = document.getElementById("section-content");
            if (contentSection) contentSection.style.display = "none";
          } else {
            // plain_text: switch type for form visibility
            typeInput.value = mode === "builder" ? "text" : "plain_text";
            updateFormFieldsVisibility();
            // Narrative text doesn't use comparison or target — hide them even in query builder mode
            if (mode === "builder") {
              const compSection = document.getElementById("section-comparison");
              if (compSection) compSection.style.display = "none";
            }
            // Keep left-panel selection in sync (plain_text either way)
            document
              .querySelectorAll(".type-panel-item")
              .forEach((b) => b.classList.remove("selected"));
            document
              .querySelector('.type-panel-item[data-type="plain_text"]')
              ?.classList.add("selected");
          }
        }
      });
    }

    // Wire up aggregation alias changes
    document
      .querySelectorAll("#aggregations-list .agg-alias")
      .forEach((input) => {
        input.addEventListener("change", () => {
          refreshSelectColumnsList();
          refreshCalcAliasChips();
        });
      });

    // Wire up existing calculation rows (templates, alias chips, validation)
    wireUpExistingCalcRows();

    setupFacetListeners();

    // Wire up grouping & series toggle
    const groupingToggleBtn = document.getElementById("btn-toggle-grouping");
    if (groupingToggleBtn) {
      const groupingContent = document.getElementById("grouping-content");
      const groupingArrow = document.getElementById("grouping-toggle-arrow");
      const isExpanded =
        localStorage.getItem("builder_grouping_expanded") === "true";
      if (isExpanded && groupingContent && groupingArrow) {
        groupingContent.style.display = "block";
        groupingArrow.textContent = "▼";
      }
      groupingToggleBtn.addEventListener("click", () => {
        if (groupingContent && groupingArrow) {
          const isHidden = groupingContent.style.display === "none";
          groupingContent.style.display = isHidden ? "block" : "none";
          groupingArrow.textContent = isHidden ? "▼" : "▶";
          localStorage.setItem(
            "builder_grouping_expanded",
            isHidden ? "true" : "false",
          );
        }
      });
    }

    // Wire up query configuration toggle
    const queryToggleBtn = document.getElementById("btn-toggle-query");
    if (queryToggleBtn) {
      const queryContent = document.getElementById("query-content");
      const queryArrow = document.getElementById("query-toggle-arrow");
      const isCollapsed =
        localStorage.getItem("builder_query_collapsed") === "true";
      if (isCollapsed && queryContent && queryArrow) {
        queryContent.style.display = "none";
        queryArrow.textContent = "▶";
      }
      queryToggleBtn.addEventListener("click", () => {
        if (queryContent && queryArrow) {
          const isHidden = queryContent.style.display === "none";
          queryContent.style.display = isHidden ? "block" : "none";
          queryArrow.textContent = isHidden ? "▼" : "▶";
          localStorage.setItem(
            "builder_query_collapsed",
            isHidden ? "false" : "true",
          );
        }
      });
    }

    // Wire up advanced options toggle
    const advancedToggleBtn = document.getElementById("btn-toggle-advanced");
    if (advancedToggleBtn) {
      const advancedContent = document.getElementById(
        "advanced-options-content",
      );
      const advancedArrow = document.getElementById("advanced-toggle-arrow");
      const isExpanded =
        localStorage.getItem("builder_advanced_expanded") === "true";
      if (isExpanded && advancedContent && advancedArrow) {
        advancedContent.style.display = "block";
        advancedArrow.textContent = "▼";
      }
      advancedToggleBtn.addEventListener("click", () => {
        if (advancedContent && advancedArrow) {
          const isHidden = advancedContent.style.display === "none";
          advancedContent.style.display = isHidden ? "block" : "none";
          advancedArrow.textContent = isHidden ? "▼" : "▶";
          localStorage.setItem(
            "builder_advanced_expanded",
            isHidden ? "true" : "false",
          );
        }
      });
    }

    // Wire up reference lines
    autoExpandReferenceLinesIfNeeded();
    const addRefLineBtn = document.getElementById("btn-add-refline");
    if (addRefLineBtn) {
      addRefLineBtn.addEventListener("click", addReferenceLine);
    }
    // Wire up existing reference line remove buttons
    document
      .querySelectorAll("#referencelines-list .btn-remove-refline")
      .forEach((btn) => {
        btn.addEventListener("click", () =>
          btn.closest(".refline-row").remove(),
        );
      });

    // Populate existing heatmap rules on edit (uses buildHeatmapRuleRowHTML for dropdowns)
    const heatmapRulesList = document.getElementById("heatmap-rules-list");
    const editingComp = currentEditingComponent?.component;
    if (heatmapRulesList && editingComp?.heatmap?.rules?.length) {
      editingComp.heatmap.rules.forEach((rule) => {
        heatmapRulesList.insertAdjacentHTML(
          "beforeend",
          buildHeatmapRuleRowHTML(
            rule.column,
            rule.thresholds?.[0],
            rule.thresholds?.[1],
            rule.colors,
          ),
        );
      });
    }

    // Populate existing per-column scale config on edit
    const heatmapColsList = document.getElementById("heatmap-cols-list");
    const useColumnsChk = document.getElementById("heatmap-use-columns");
    const columnsGroup = document.getElementById("heatmap-columns-group");
    if (heatmapColsList && editingComp?.heatmap?.columns?.length) {
      if (useColumnsChk) useColumnsChk.checked = true;
      if (columnsGroup) columnsGroup.style.display = "block";
      const defaultScale = editingComp.heatmap.scale || "green-red";
      editingComp.heatmap.columns.forEach((col) => {
        const name = typeof col === "string" ? col : col.name;
        const scale =
          typeof col === "string" ? defaultScale : col.scale || defaultScale;
        heatmapColsList.insertAdjacentHTML(
          "beforeend",
          buildHeatmapColumnRowHTML(name, scale),
        );
      });
    }

    // Wire up heatmap column add button
    const addHeatmapColBtn = document.getElementById("btn-add-heatmap-col");
    if (addHeatmapColBtn) {
      addHeatmapColBtn.addEventListener("click", () => {
        document
          .getElementById("heatmap-cols-list")
          ?.insertAdjacentHTML("beforeend", buildHeatmapColumnRowHTML("", ""));
      });
    }
    if (heatmapColsList) {
      heatmapColsList.addEventListener("click", (e) => {
        if (e.target.classList.contains("btn-remove-heatmap-col")) {
          e.target.closest(".heatmap-col-row")?.remove();
        }
      });
    }

    // Wire up heatmap rule add button
    const addHeatmapRuleBtn = document.getElementById("btn-add-heatmap-rule");
    if (addHeatmapRuleBtn) {
      addHeatmapRuleBtn.addEventListener("click", () => {
        const list = document.getElementById("heatmap-rules-list");
        if (!list) return;
        list.insertAdjacentHTML(
          "beforeend",
          buildHeatmapRuleRowHTML("", "", "", "red-yellow-green"),
        );
      });
    }

    // Wire up heatmap rule remove via event delegation
    if (heatmapRulesList) {
      heatmapRulesList.addEventListener("click", (e) => {
        if (e.target.classList.contains("btn-remove-heatmap-rule")) {
          e.target.closest(".heatmap-rule-row")?.remove();
        }
      });
      // Validate lo < hi on input
      heatmapRulesList.addEventListener("input", (e) => {
        if (
          e.target.classList.contains("heatmap-rule-low") ||
          e.target.classList.contains("heatmap-rule-high")
        ) {
          const row = e.target.closest(".heatmap-rule-row");
          if (!row) return;
          const lo = parseFloat(row.querySelector(".heatmap-rule-low")?.value);
          const hi = parseFloat(row.querySelector(".heatmap-rule-high")?.value);
          const warn = row.querySelector(".heatmap-rule-warning");
          if (warn) {
            warn.style.display =
              !isNaN(lo) && !isNaN(hi) && lo >= hi ? "block" : "none";
          }
        }
      });
    }

    const headerTooltipsList = document.getElementById("header-tooltips-list");
    const addHeaderTooltipBtn = document.getElementById(
      "btn-add-header-tooltip",
    );
    const splitColumnsList = document.getElementById("split-columns-list");
    const addSplitColumnBtn = document.getElementById("btn-add-split-column");
    if (addHeaderTooltipBtn) {
      addHeaderTooltipBtn.addEventListener("click", () => {
        if (!headerTooltipsList) return;
        ensureHeaderTooltipEmptyState();
        headerTooltipsList.insertAdjacentHTML(
          "beforeend",
          buildHeaderTooltipRowHTML(),
        );
        ensureHeaderTooltipEmptyState();
        refreshHeaderTooltipSuggestions();
        refreshSplitColumnSuggestions();
      });
    }
    if (headerTooltipsList) {
      headerTooltipsList.addEventListener("click", (e) => {
        if (e.target.classList.contains("btn-remove-header-tooltip")) {
          e.target.closest(".header-tooltip-row")?.remove();
          ensureHeaderTooltipEmptyState();
          refreshHeaderTooltipSuggestions();
          refreshSplitColumnSuggestions();
        }
      });
      headerTooltipsList.addEventListener("input", (e) => {
        if (e.target.classList.contains("header-tooltip-header")) {
          refreshHeaderTooltipSuggestions();
          refreshSplitColumnSuggestions();
        }
      });
    }

    if (addSplitColumnBtn) {
      addSplitColumnBtn.addEventListener("click", () => {
        if (!splitColumnsList) return;
        ensureSplitColumnEmptyState();
        splitColumnsList.insertAdjacentHTML(
          "beforeend",
          buildSplitColumnRowHTML(),
        );
        ensureSplitColumnEmptyState();
        refreshSplitColumnSuggestions();
      });
    }

    if (splitColumnsList) {
      splitColumnsList.addEventListener("click", (e) => {
        if (e.target.classList.contains("btn-remove-split-column")) {
          e.target.closest(".split-column-row")?.remove();
          ensureSplitColumnEmptyState();
          refreshSplitColumnSuggestions();
        }
      });

      splitColumnsList.addEventListener("input", (e) => {
        if (
          e.target.classList.contains("split-column-parent") ||
          e.target.classList.contains("split-column-children")
        ) {
          refreshSplitColumnSuggestions();
        }
      });
    }

    ensureHeaderTooltipEmptyState();
    refreshHeaderTooltipSuggestions();
    ensureSplitColumnEmptyState();
    refreshSplitColumnSuggestions();

    // Initialize pivot form state for table_advanced components that already have pivot config
    const editingCompType = currentEditingComponent?.component?.type;
    if (editingCompType === "table_advanced") {
      initPivotFormState(currentEditingComponent.component);
    }

    const selectColumnsList = document.getElementById("select-columns-list");
    if (selectColumnsList) {
      selectColumnsList.addEventListener("change", (e) => {
        if (!e.target.classList.contains("select-column-checkbox")) return;

        const type = document.getElementById("comp-type")?.value || "";
        const sqlEnabled = document.getElementById(
          "choropleth-sql-enabled",
        )?.checked;
        if (type !== "choropleth" || !sqlEnabled) return;

        if (e.target.checked) {
          selectColumnsList
            .querySelectorAll(".select-column-checkbox")
            .forEach((checkbox) => {
              if (checkbox !== e.target) checkbox.checked = false;
            });
        }

        syncChoroplethSQLValueSelectionFromCheckboxes();
      });
    }

    const choroplethDistrictSelect = document.getElementById(
      "choropleth-district-column",
    );
    const choroplethValueSelect = document.getElementById(
      "choropleth-value-column",
    );
    if (choroplethDistrictSelect) {
      choroplethDistrictSelect.addEventListener("change", () => {
        syncChoroplethSQLValueSelectionFromMapping();
      });
    }
    if (choroplethValueSelect) {
      choroplethValueSelect.addEventListener("change", () => {
        refreshSelectColumnsList();
      });
    }

    // Initialize autocomplete on column inputs if table is already selected
    if (tableSelect && tableSelect.value) {
      initAllAutocomplete();
    }
  }, 100);
}

/**
 * Close the component modal
 */
export function closeComponentModal() {
  elements.componentModal.classList.add("hidden");
}

/**
 * Handle advanced SQL preview
 */
async function handleAdvancedSQLPreview() {
  const sqlInput = document.getElementById("advanced-sql");
  const sql = sqlInput?.value?.trim();

  if (!sql) {
    showError("Please enter a SQL query");
    return;
  }

  const previewBtn = document.getElementById("btn-preview-advanced-sql");
  const resultContainer = document.getElementById(
    "advanced-sql-preview-result",
  );
  const infoDiv = document.getElementById("advanced-sql-preview-info");
  const tableDiv = document.getElementById("advanced-sql-preview-table");

  // Show loading state
  previewBtn.disabled = true;
  previewBtn.textContent = "⏳ Running...";

  // If pivot is enabled, build pivot config for the preview request
  const pivotEnabled = document.getElementById("pivot-enabled")?.checked;
  let pivotConfig = null;
  if (pivotEnabled) {
    const rows = document.getElementById("pivot-rows")?.value;
    const columns = document.getElementById("pivot-columns")?.value;
    const values = document.getElementById("pivot-values")?.value;
    if (rows && columns && values) {
      pivotConfig = buildPivotConfigFromForm();
    }
  }

  try {
    const requestBody = { sql, filters: {}, limit: 20 };
    if (pivotConfig) requestBody.pivot = pivotConfig;

    const result = await previewAdvancedSQL(requestBody);

    resultContainer.style.display = "block";

    if (result.error) {
      infoDiv.innerHTML = `<span style="color: var(--color-error);">Error: ${result.error}</span>`;
      tableDiv.innerHTML = "";
      if (!pivotEnabled) storeAdvancedPreviewHeaders([]);
    } else {
      const pivotLabel = pivotConfig
        ? ' <em style="font-size:11px;">(pivoted)</em>'
        : "";
      infoDiv.innerHTML = `<span style="color: var(--color-success);">✓ ${result.rowCount} rows · ${result.executionTime}${pivotLabel}</span>`;

      // Build preview table
      if (result.headers && result.rows) {
        let tableHtml =
          '<table style="width: 100%; border-collapse: collapse; font-size: 11px;">';
        tableHtml += "<thead><tr>";
        result.headers.forEach((h) => {
          tableHtml += `<th style="border: 1px solid var(--color-border); padding: 4px; text-align: left; background: var(--color-bg);">${escapeHtml(String(h))}</th>`;
        });
        tableHtml += "</tr></thead><tbody>";
        result.rows.forEach((row) => {
          tableHtml += "<tr>";
          row.forEach((cell) => {
            const displayVal =
              cell === null || cell === undefined
                ? "<em>null</em>"
                : escapeHtml(String(cell));
            tableHtml += `<td style="border: 1px solid var(--color-border); padding: 4px;">${displayVal}</td>`;
          });
          tableHtml += "</tr>";
        });
        tableHtml += "</tbody></table>";
        tableDiv.innerHTML = tableHtml;

        // When NOT in pivot mode, store raw SQL headers and populate pivot dropdowns
        if (!pivotConfig) {
          // Populate heatmap column suggestions from preview headers
          const datalist = document.getElementById(
            "heatmap-column-suggestions",
          );
          if (datalist) {
            datalist.innerHTML = result.headers
              .map((h) => `<option value="${escapeHtml(h)}">`)
              .join("");
          }
          storeAdvancedPreviewHeaders(result.headers);
          populatePivotDropdowns(result.headers);
        }
      } else {
        tableDiv.innerHTML = "<em>No data returned</em>";
        if (!pivotEnabled) storeAdvancedPreviewHeaders([]);
      }
    }
  } catch (err) {
    resultContainer.style.display = "block";
    infoDiv.innerHTML = `<span style="color: var(--color-error);">Error: ${err.message}</span>`;
    tableDiv.innerHTML = "";
    storeAdvancedPreviewHeaders([]);
  } finally {
    previewBtn.disabled = false;
    previewBtn.textContent = "▶ Preview SQL Results";
  }
}

/**
 * Show or hide the pivot configuration panel based on the Enable Pivot checkbox.
 */
window.togglePivotConfig = function togglePivotConfig() {
  const enabled = document.getElementById("pivot-enabled")?.checked;
  const panel = document.getElementById("pivot-config");
  if (panel) panel.style.display = enabled ? "block" : "none";
};

/**
 * Populate the three pivot axis dropdowns from the SQL preview headers.
 * Existing selections are preserved if they still exist in the new header list.
 */
function populatePivotDropdowns(headers) {
  ["pivot-rows", "pivot-columns", "pivot-values"].forEach((id) => {
    const sel = document.getElementById(id);
    if (!sel) return;
    const current = sel.value;
    sel.innerHTML =
      `<option value="">— select column —</option>` +
      headers
        .map(
          (h) =>
            `<option value="${escapeHtml(h)}" ${h === current ? "selected" : ""}>${escapeHtml(h)}</option>`,
        )
        .join("");
  });
}

/**
 * Read the pivot configuration from form elements.
 * Returns a pivot config object, or null if the feature is not enabled / not configured.
 */
function buildPivotConfigFromForm() {
  const enabled = document.getElementById("pivot-enabled")?.checked;
  if (!enabled) return null;

  const rows = document.getElementById("pivot-rows")?.value?.trim();
  const columns = document.getElementById("pivot-columns")?.value?.trim();
  const values = document.getElementById("pivot-values")?.value?.trim();
  if (!rows || !columns || !values) return null;

  const format = document.getElementById("pivot-format")?.value || "auto";

  // Read extra columns
  const extraColumns = [];
  document
    .querySelectorAll("#pivot-extra-columns-list .pivot-extra-item")
    .forEach((el) => {
      const label = el.querySelector(".pivot-extra-label")?.value?.trim();
      const formula = el.querySelector(".pivot-extra-formula")?.value?.trim();
      if (label && formula) extraColumns.push({ label, formula });
    });

  // Read extra rows
  const extraRows = [];
  document
    .querySelectorAll("#pivot-extra-rows-list .pivot-extra-item")
    .forEach((el) => {
      const label = el.querySelector(".pivot-extra-label")?.value?.trim();
      const formula = el.querySelector(".pivot-extra-formula")?.value?.trim();
      if (label && formula) extraRows.push({ label, formula });
    });

  const cfg = { rows, columns, values };
  if (format && format !== "auto") cfg.format = format;
  if (extraColumns.length > 0) cfg.extraColumns = extraColumns;
  if (extraRows.length > 0) cfg.extraRows = extraRows;
  return cfg;
}

/**
 * Render saved extra pivot items (extra columns or extra rows) into a container.
 */
function renderPivotExtraItems(items, containerId, formulaPlaceholder) {
  const container = document.getElementById(containerId);
  if (!container) return;
  container.innerHTML = "";
  (items || []).forEach((item) =>
    appendPivotExtraItemRow(
      container,
      item.label,
      item.formula,
      formulaPlaceholder,
    ),
  );
}

function appendPivotExtraItemRow(
  container,
  label,
  formula,
  formulaPlaceholder,
) {
  const div = document.createElement("div");
  div.className = "pivot-extra-item";
  div.style.cssText = "display: flex; gap: 6px; align-items: center;";
  div.innerHTML = `
        <input type="text" class="form-input pivot-extra-label" placeholder="Label" value="${escapeHtml(label || "")}" style="flex: 1; font-size: 12px;">
        <input type="text" class="form-input pivot-extra-formula" placeholder="${escapeHtml(formulaPlaceholder || "formula")}" value="${escapeHtml(formula || "")}" style="flex: 1; font-size: 12px;">
        <button type="button" class="btn-secondary" style="padding: 4px 8px; font-size: 11px; color: var(--color-error); border-color: var(--color-error);" onclick="this.closest('.pivot-extra-item').remove()">✕</button>
    `;
  container.appendChild(div);
}

window.addPivotExtraColumn = function addPivotExtraColumn() {
  const container = document.getElementById("pivot-extra-columns-list");
  if (container)
    appendPivotExtraItemRow(container, "", "", "e.g. sum or [-1]-[-2]");
};

window.addPivotExtraRow = function addPivotExtraRow() {
  const container = document.getElementById("pivot-extra-rows-list");
  if (container) appendPivotExtraItemRow(container, "", "", "sum or avg");
};

/**
 * After the component form is rendered, populate pivot dropdowns and extra-item lists
 * if the component already has a pivot config.
 */
function initPivotFormState(component) {
  if (!component.pivot) return;

  const cfg = component.pivot;
  const storedHeaders = getStoredAdvancedPreviewHeaders();

  // Build a minimal header list from the saved axis names so the dropdowns aren't empty
  // (real headers will be loaded when the user previews the SQL)
  const seedHeaders = [cfg.rows, cfg.columns, cfg.values].filter(Boolean);
  if (storedHeaders.length > 0) {
    populatePivotDropdowns(storedHeaders);
  } else if (seedHeaders.length > 0) {
    populatePivotDropdowns(seedHeaders);
  }

  // Set the saved selections
  const setVal = (id, val) => {
    const el = document.getElementById(id);
    if (el && val) el.value = val;
  };
  setVal("pivot-rows", cfg.rows);
  setVal("pivot-columns", cfg.columns);
  setVal("pivot-values", cfg.values);
  setVal("pivot-format", cfg.format || "auto");

  renderPivotExtraItems(
    cfg.extraColumns || [],
    "pivot-extra-columns-list",
    "e.g. sum or [-1]-[-2]",
  );
  renderPivotExtraItems(
    cfg.extraRows || [],
    "pivot-extra-rows-list",
    "sum or avg",
  );
}

/**
 * Update form field visibility based on component type
 */
export function updateFormFieldsVisibility() {
  const typeInput = document.getElementById("comp-type");
  const type =
    typeInput?.value || currentEditingComponent?.component?.type || "bar";

  const sectionVisibility = {
    text: {
      datasource: true,
      content: true,
      query: true,
      comparison: true,
      groupby: false,
      seriesby: false,
      where: true,
      selectcolumns: false,
      calculate: true,
      orderby: false,
      periodlimit: false,
      legend: false,
      referencelines: false,
      axislabels: false,
      ylimit: false,
      geojson: false,
      facet: false,
      baroptions: false,
      trendline: false,
      advancedsql: false,
      choroplethsql: false,
      chartsql: false,
      heatmap: false,
      headertooltips: false,
      advancedoptions: true,
    },
    kpi: {
      datasource: false,
      content: false,
      query: true,
      comparison: true,
      groupby: false,
      seriesby: false,
      where: true,
      selectcolumns: false,
      calculate: true,
      orderby: false,
      periodlimit: false,
      legend: false,
      referencelines: false,
      axislabels: false,
      ylimit: false,
      geojson: false,
      facet: false,
      baroptions: false,
      trendline: false,
      advancedsql: false,
      choroplethsql: false,
      chartsql: false,
      heatmap: false,
      headertooltips: false,
      kpi: true,
      advancedoptions: true,
    },
    bar: {
      datasource: false,
      content: false,
      query: true,
      comparison: false,
      groupby: true,
      seriesby: true,
      where: true,
      selectcolumns: true,
      calculate: true,
      orderby: true,
      periodlimit: true,
      legend: true,
      referencelines: true,
      axislabels: true,
      ylimit: true,
      geojson: false,
      facet: false,
      baroptions: true,
      trendline: true,
      advancedsql: false,
      choroplethsql: false,
      chartsql: true,
      heatmap: false,
      headertooltips: false,
      advancedoptions: true,
    },
    line: {
      datasource: false,
      content: false,
      query: true,
      comparison: false,
      groupby: true,
      seriesby: true,
      where: true,
      selectcolumns: true,
      calculate: true,
      orderby: true,
      periodlimit: true,
      legend: true,
      referencelines: true,
      axislabels: true,
      ylimit: true,
      geojson: false,
      facet: false,
      baroptions: true,
      trendline: true,
      advancedsql: false,
      choroplethsql: false,
      chartsql: true,
      heatmap: false,
      headertooltips: false,
      advancedoptions: true,
    },
    bar_line: {
      datasource: false,
      content: false,
      query: true,
      comparison: false,
      groupby: true,
      seriesby: true,
      where: true,
      selectcolumns: true,
      calculate: true,
      orderby: true,
      periodlimit: true,
      legend: true,
      referencelines: true,
      axislabels: true,
      ylimit: true,
      geojson: false,
      facet: false,
      baroptions: true,
      trendline: false,
      advancedsql: false,
      choroplethsql: false,
      chartsql: true,
      heatmap: false,
      headertooltips: false,
      advancedoptions: true,
    },
    pie: {
      datasource: false,
      content: false,
      query: true,
      comparison: false,
      groupby: false,
      seriesby: false,
      where: true,
      selectcolumns: true,
      calculate: true,
      orderby: false,
      periodlimit: false,
      legend: true,
      referencelines: false,
      axislabels: false,
      ylimit: false,
      geojson: false,
      facet: false,
      baroptions: false,
      trendline: false,
      advancedsql: false,
      choroplethsql: false,
      chartsql: true,
      heatmap: false,
      headertooltips: false,
      advancedoptions: true,
    },
    pyramid: {
      datasource: false,
      content: false,
      query: true,
      comparison: false,
      groupby: true,
      seriesby: false,
      where: true,
      selectcolumns: true,
      calculate: false,
      orderby: false,
      periodlimit: false,
      legend: true,
      referencelines: true,
      axislabels: true,
      ylimit: false,
      geojson: false,
      facet: false,
      baroptions: false,
      trendline: false,
      advancedsql: false,
      choroplethsql: false,
      chartsql: true,
      heatmap: false,
      headertooltips: false,
      advancedoptions: true,
    },
    table: {
      datasource: false,
      content: false,
      query: true,
      comparison: false,
      groupby: true,
      seriesby: false,
      where: true,
      selectcolumns: true,
      calculate: true,
      orderby: true,
      periodlimit: true,
      legend: false,
      referencelines: false,
      axislabels: false,
      ylimit: false,
      geojson: false,
      facet: false,
      baroptions: false,
      trendline: false,
      advancedsql: false,
      choroplethsql: false,
      chartsql: false,
      heatmap: true,
      headertooltips: true,
      advancedoptions: true,
    },
    table_advanced: {
      datasource: false,
      content: false,
      query: false,
      comparison: false,
      groupby: false,
      seriesby: false,
      where: false,
      selectcolumns: false,
      calculate: false,
      orderby: false,
      periodlimit: false,
      legend: false,
      referencelines: false,
      axislabels: false,
      ylimit: false,
      geojson: false,
      facet: false,
      baroptions: false,
      trendline: false,
      advancedsql: true,
      choroplethsql: false,
      chartsql: false,
      heatmap: true,
      headertooltips: true,
      advancedoptions: true,
    },
    choropleth: {
      datasource: false,
      content: false,
      query: true,
      comparison: false,
      groupby: true,
      seriesby: false,
      where: true,
      selectcolumns: true,
      calculate: true,
      orderby: false,
      periodlimit: false,
      legend: false,
      referencelines: false,
      axislabels: false,
      ylimit: false,
      geojson: true,
      facet: true,
      baroptions: false,
      trendline: false,
      advancedsql: false,
      choroplethsql: true,
      chartsql: false,
      heatmap: false,
      headertooltips: false,
      advancedoptions: true,
    },
    map: {
      datasource: false,
      content: false,
      query: true,
      comparison: false,
      groupby: true,
      seriesby: false,
      where: true,
      selectcolumns: false,
      calculate: false,
      orderby: false,
      periodlimit: false,
      legend: false,
      referencelines: false,
      axislabels: false,
      ylimit: false,
      geojson: true,
      facet: false,
      baroptions: false,
      trendline: false,
      advancedsql: false,
      choroplethsql: false,
      chartsql: false,
      heatmap: false,
      headertooltips: false,
      advancedoptions: true,
    },
    plain_text: {
      datasource: true,
      content: true,
      query: false,
      comparison: false,
      groupby: false,
      seriesby: false,
      where: false,
      selectcolumns: false,
      calculate: false,
      orderby: false,
      periodlimit: false,
      legend: false,
      referencelines: false,
      axislabels: false,
      ylimit: false,
      geojson: false,
      facet: false,
      baroptions: false,
      trendline: false,
      advancedsql: false,
      choroplethsql: false,
      chartsql: false,
      heatmap: false,
      headertooltips: false,
      advancedoptions: false,
    },
    infobox: {
      datasource: true,
      content: false,
      query: false,
      comparison: false,
      groupby: false,
      seriesby: false,
      where: false,
      selectcolumns: false,
      calculate: false,
      orderby: false,
      periodlimit: false,
      legend: false,
      referencelines: false,
      axislabels: false,
      ylimit: false,
      geojson: false,
      facet: false,
      baroptions: false,
      trendline: false,
      advancedsql: false,
      choroplethsql: false,
      chartsql: false,
      heatmap: false,
      headertooltips: false,
      advancedoptions: false,
      infobox: true,
    },
  };

  const visibility = sectionVisibility[type] || sectionVisibility.bar;

  const sections = {
    datasource: document.getElementById("section-datasource"),
    content: document.getElementById("section-content"),
    query: document.getElementById("section-query"),
    comparison: document.getElementById("section-comparison"),
    groupby: document.getElementById("section-groupby"),
    seriesby: document.getElementById("section-seriesby"),
    where: document.getElementById("section-where"),
    selectcolumns: document.getElementById("section-select-columns"),
    calculate: document.getElementById("section-calculate"),
    orderby: document.getElementById("section-orderby"),
    periodlimit: document.getElementById("section-periodlimit"),
    legend: document.getElementById("section-legend"),
    referencelines: document.getElementById("section-referencelines"),
    axislabels: document.getElementById("section-axislabels"),
    ylimit: document.getElementById("section-ylimit"),
    geojson: document.getElementById("section-geojson"),
    facet: document.getElementById("section-facet"),
    baroptions: document.getElementById("section-baroptions"),
    trendline: document.getElementById("section-trendline"),
    advancedsql: document.getElementById("section-advanced-sql"),
    choroplethsql: document.getElementById("section-choropleth-sql"),
    chartsql: document.getElementById("section-chart-sql"),
    heatmap: document.getElementById("section-heatmap"),
    headertooltips: document.getElementById("section-header-tooltips"),
    advancedoptions: document.getElementById("section-advanced-options"),
    infobox: document.getElementById("section-infobox"),
    kpi: document.getElementById("section-kpi"),
  };

  for (const [section, element] of Object.entries(sections)) {
    // Always show seriesBy for bar, line, bar_line (even with custom SQL)
    if (element) {
      if (
        section === "seriesby" &&
        ["bar", "line", "bar_line"].includes(type)
      ) {
        element.style.display = "block";
      } else {
        element.style.display = visibility[section] ? "block" : "none";
      }
    }
  }

  // Line charts reuse the Bar Options section but only expose the value-label
  // and axis-title controls. Hide the bar-specific controls and relabel the section.
  if (visibility.baroptions) {
    const isLine = type === "line";
    const baroptionsTitle = document.getElementById("baroptions-title");
    if (baroptionsTitle)
      baroptionsTitle.textContent = isLine ? "Line Options" : "Bar Options";
    [
      "bar-opt-stacked-group",
      "bar-opt-horizontal-group",
      "bar-opt-showvalues-group",
      "bar-opt-wrap-group",
    ].forEach((id) => {
      const el = document.getElementById(id);
      if (el) el.style.display = isLine ? "none" : "";
    });
  }

  // When chart SQL is enabled, re-apply overrides (hide query/groupby, show dataset mapping)
  const chartSqlEnabled = document.getElementById("chart-sql-enabled")?.checked;
  if (chartSqlEnabled) {
    const querySection = document.getElementById("section-query");
    const groupBySection = document.getElementById("section-groupby");
    if (querySection) querySection.style.display = "none";
    if (groupBySection) groupBySection.style.display = "none";
    // Do NOT hide seriesBy for bar/line/bar_line with custom SQL
    // Hide only query-builder sub-sections that have no meaning for raw SQL.
    // Period Range stays visible — backend honors periodLimit for raw SQL when {{*_filter}} placeholders are present.
    [
      "section-where",
      "section-calculate",
      "section-orderby",
      "section-select-columns",
    ].forEach((id) => {
      const el = document.getElementById(id);
      if (el) el.style.display = "none";
    });
  }
  const periodLimitSqlHint = document.getElementById("period-limit-sql-hint");
  if (periodLimitSqlHint) {
    periodLimitSqlHint.style.display = chartSqlEnabled ? "block" : "none";
  }
  const dsMapping = document.getElementById("chart-sql-dataset-mapping");
  if (dsMapping) {
    dsMapping.style.display =
      chartSqlEnabled && type === "bar_line" ? "block" : "none";
  }

  // When KPI SQL mode is enabled, hide query builder sections (they're not used)
  const kpiSqlEnabled =
    type === "kpi" && !!document.getElementById("kpi-sql-enabled")?.checked;
  if (kpiSqlEnabled) {
    ["section-query", "section-where", "section-calculate"].forEach((id) => {
      const el = document.getElementById(id);
      if (el) el.style.display = "none";
    });
  }

  // Show pyramid guidance hint when type is pyramid
  const pyramidGuidance = document.getElementById("pyramid-guidance");
  if (pyramidGuidance) {
    pyramidGuidance.style.display = type === "pyramid" ? "block" : "none";
  }

  if (visibility.selectcolumns) {
    refreshSelectColumnsList();

    const selectColsTitle = document.getElementById("selectcols-section-title");
    const selectColsHint = document.getElementById("selectcols-section-hint");

    const choroplethSqlEnabled =
      type === "choropleth" &&
      !!document.getElementById("choropleth-sql-enabled")?.checked;

    if (choroplethSqlEnabled) {
      if (selectColsTitle) selectColsTitle.textContent = "Map Value Column";
      if (selectColsHint)
        selectColsHint.textContent =
          "Run SQL Preview to load result columns, then choose one value column for the map. District selection stays in Column Mapping.";
    } else if (type === "bar_line") {
      if (selectColsTitle)
        selectColsTitle.textContent =
          "Columns, Colors & Chart Types (Required)";
      if (selectColsHint)
        selectColsHint.textContent =
          "Choose which columns to display, set colors, and whether each is a Bar or Line";
      const content = document.getElementById("selectcols-content");
      const icon = document.getElementById("selectcols-toggle-icon");
      if (content && icon && content.style.display === "none") {
        content.style.display = "block";
        icon.textContent = "▼";
      }
    } else if (type === "bar" || type === "line" || type === "pie") {
      if (selectColsTitle) selectColsTitle.textContent = "Columns & Colors";
      if (selectColsHint)
        selectColsHint.textContent =
          "All columns are shown by default. Uncheck to hide, use color pickers to customize.";
    } else {
      if (selectColsTitle)
        selectColsTitle.textContent = "Select Columns to Display";
      if (selectColsHint)
        selectColsHint.textContent =
          "All columns are shown by default. Uncheck any you want to hide.";
    }

    // Show/hide pivot subsection based on component type (only for tables)
    const pivotSubsection = document.getElementById("pivot-subsection");
    if (pivotSubsection) {
      pivotSubsection.style.display = type === "table" ? "block" : "none";
    }

    // Show/hide transpose subsection based on component type (only for tables)
    const transposeSubsection = document.getElementById("transpose-subsection");
    if (transposeSubsection) {
      transposeSubsection.style.display = type === "table" ? "block" : "none";
    }
  }

  refreshHeaderTooltipSuggestions();
}

/**
 * Read heatmap config from the builder form.
 * Returns { rules: [...] }, { scale: "..." }, or null.
 */
function readHeatmapConfig() {
  if (!document.getElementById("heatmap-enabled")?.checked) return null;

  const advanced = document.getElementById("heatmap-advanced")?.checked;
  if (advanced) {
    const ruleRows = document.querySelectorAll(
      "#heatmap-rules-list .heatmap-rule-row",
    );
    const rules = [];
    ruleRows.forEach((row) => {
      const column =
        row.querySelector(".heatmap-rule-column")?.value?.trim() || "";
      const lo = parseFloat(row.querySelector(".heatmap-rule-low")?.value);
      const hi = parseFloat(row.querySelector(".heatmap-rule-high")?.value);
      const colors =
        row.querySelector(".heatmap-rule-colors")?.value || "red-yellow-green";
      if (column && !isNaN(lo) && !isNaN(hi)) {
        rules.push({ column, thresholds: [lo, hi], colors });
      }
    });
    if (rules.length > 0) return { rules };
  }

  const scale = document.getElementById("heatmap-scale")?.value || "green-red";
  const useColumns = document.getElementById("heatmap-use-columns")?.checked;
  if (useColumns) {
    const colRows = document.querySelectorAll(
      "#heatmap-cols-list .heatmap-col-row",
    );
    const columns = [];
    colRows.forEach((row) => {
      const name = (row.querySelector(".heatmap-col-name")?.value || "").trim();
      const colScale = row.querySelector(".heatmap-col-scale")?.value || scale;
      if (name) columns.push({ name, scale: colScale });
    });
    if (columns.length > 0) return { scale, columns };
  }
  return { scale };
}

function buildHeatmapColumnRowHTML(selectedColumn, selectedScale) {
  const compType = document.getElementById("comp-type")?.value || "";
  const columns = compType === "table_advanced" ? [] : getAvailableColumns();
  let columnField;
  if (columns.length > 0) {
    const options = columns
      .map((col) => {
        const escaped = escapeHtml(col);
        return `<option value="${escaped}" ${col === selectedColumn ? "selected" : ""}>${escaped}</option>`;
      })
      .join("");
    columnField = `<select class="form-input heatmap-col-name" style="flex: 1; min-width: 120px;">
            <option value="">-- Column --</option>${options}</select>`;
  } else {
    columnField = `<input type="text" class="form-input heatmap-col-name" value="${escapeHtml(selectedColumn || "")}" placeholder="Column name" list="heatmap-column-suggestions" style="flex: 1; min-width: 120px;">`;
  }
  const scale = selectedScale || "green-red";
  return `
        <div class="heatmap-col-row" style="display: flex; gap: 8px; align-items: center; margin-bottom: 6px;">
            ${columnField}
            <select class="form-input heatmap-col-scale" style="width: 160px;">
                <option value="green-red" ${scale === "green-red" ? "selected" : ""}>Green→Red (high = bad)</option>
                <option value="red-green" ${scale === "red-green" ? "selected" : ""}>Red→Green (high = good)</option>
                <option value="blue" ${scale === "blue" ? "selected" : ""}>Blue (neutral)</option>
            </select>
            <button type="button" class="btn-small btn-remove-heatmap-col" style="background: #ff6b6b; color: white; padding: 6px 8px;">×</button>
        </div>`;
}

function buildHeaderTooltipRowHTML(header = "", tooltip = "", formula = "") {
  return `
        <div class="header-tooltip-row" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: flex-start;">
            <input type="text" class="form-input header-tooltip-header" value="${escapeHtml(header)}" placeholder="Column name (e.g., TPR %)" list="header-tooltip-suggestions" style="flex: 0 0 140px; margin: 0;">
            <div style="flex: 1; display: flex; flex-direction: column; gap: 4px;">
                <input type="text" class="form-input header-tooltip-text" value="${escapeHtml(tooltip)}" placeholder="Tooltip text (e.g., Test Positivity Rate)" style="margin: 0;">
                <input type="text" class="form-input header-tooltip-formula" value="${escapeHtml(formula)}" placeholder="Calculation formula (e.g., deaths / admissions × 100)" style="font-family: monospace; margin: 0;">
            </div>
            <button type="button" class="btn-small btn-remove-header-tooltip" aria-label="Remove header tooltip" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px; flex: 0 0 28px;">&times;</button>
        </div>
    `;
}

function buildSplitColumnRowHTML(parent = "", children = []) {
  const childText = Array.isArray(children) ? children.join(", ") : "";
  return `
        <div class="split-column-row" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center;">
            <input type="text" class="form-input split-column-parent" value="${escapeHtml(parent)}" placeholder="Parent header" list="split-column-suggestions" style="flex: 1; margin: 0;">
            <input type="text" class="form-input split-column-children" value="${escapeHtml(childText)}" placeholder="Child headers (comma-separated)" list="split-column-suggestions" style="flex: 2; margin: 0;">
            <button type="button" class="btn-small btn-remove-split-column" aria-label="Remove split column group" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">&times;</button>
        </div>
    `;
}

function ensureHeaderTooltipEmptyState() {
  const list = document.getElementById("header-tooltips-list");
  if (!list) return;

  const rows = list.querySelectorAll(".header-tooltip-row");
  const emptyMsg = list.querySelector("#no-header-tooltips-msg");

  if (rows.length === 0 && !emptyMsg) {
    list.innerHTML =
      '<p id="no-header-tooltips-msg" style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">No header tooltips yet. Add one to explain abbreviations like TPR % or CFR %.</p>';
  } else if (rows.length > 0 && emptyMsg) {
    emptyMsg.remove();
  }
}

function ensureSplitColumnEmptyState() {
  const list = document.getElementById("split-columns-list");
  if (!list) return;

  const rows = list.querySelectorAll(".split-column-row");
  const emptyMsg = list.querySelector("#no-split-columns-msg");

  if (rows.length === 0 && !emptyMsg) {
    list.innerHTML =
      '<p id="no-split-columns-msg" style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">No split column groups yet. Add one to create parent/child headers.</p>';
  } else if (rows.length > 0 && emptyMsg) {
    emptyMsg.remove();
  }
}

function getStoredAdvancedPreviewHeaders() {
  const datalist = document.getElementById("header-tooltip-suggestions");
  if (!datalist?.dataset.previewHeaders) return [];

  try {
    const parsed = JSON.parse(datalist.dataset.previewHeaders);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function storeAdvancedPreviewHeaders(headers) {
  const datalist = document.getElementById("header-tooltip-suggestions");
  if (!datalist) return;

  datalist.dataset.previewHeaders = JSON.stringify(
    Array.isArray(headers) ? headers : [],
  );
  refreshHeaderTooltipSuggestions();
}

function refreshHeaderTooltipSuggestions() {
  const datalist = document.getElementById("header-tooltip-suggestions");
  if (!datalist) return;

  const type =
    document.getElementById("comp-type")?.value ||
    currentEditingComponent?.component?.type ||
    "";
  const suggestions = new Set();

  if (type === "table") {
    getAvailableColumns().forEach((col) => {
      if (col) suggestions.add(col);
    });
  }

  if (type === "table_advanced") {
    getStoredAdvancedPreviewHeaders().forEach((header) => {
      if (header) suggestions.add(header);
    });
  }

  Object.keys(currentEditingComponent?.component?.headerTooltips || {}).forEach(
    (header) => {
      if (header) suggestions.add(header);
    },
  );

  document
    .querySelectorAll("#header-tooltips-list .header-tooltip-header")
    .forEach((input) => {
      const header = input.value?.trim();
      if (header) suggestions.add(header);
    });

  datalist.innerHTML = Array.from(suggestions)
    .sort((a, b) => a.localeCompare(b))
    .map((header) => `<option value="${escapeHtml(header)}">`)
    .join("");
}

function refreshSplitColumnSuggestions() {
  const datalist = document.getElementById("split-column-suggestions");
  if (!datalist) return;

  const suggestions = new Set();

  getStoredAdvancedPreviewHeaders().forEach((header) => {
    if (header) suggestions.add(header);
  });

  document
    .querySelectorAll("#header-tooltips-list .header-tooltip-header")
    .forEach((input) => {
      const header = input.value?.trim();
      if (header) suggestions.add(header);
    });

  document
    .querySelectorAll("#split-columns-list .split-column-parent")
    .forEach((input) => {
      const header = input.value?.trim();
      if (header) suggestions.add(header);
    });

  document
    .querySelectorAll("#split-columns-list .split-column-children")
    .forEach((input) => {
      const raw = input.value || "";
      raw
        .split(",")
        .map((v) => v.trim())
        .filter(Boolean)
        .forEach((v) => suggestions.add(v));
    });

  datalist.innerHTML = Array.from(suggestions)
    .sort((a, b) => a.localeCompare(b))
    .map((header) => `<option value="${escapeHtml(header)}">`)
    .join("");
}

function readHeaderTooltipMap() {
  const rows = document.querySelectorAll(
    "#header-tooltips-list .header-tooltip-row",
  );
  const tooltipMap = {};
  const formulaMap = {};

  rows.forEach((row) => {
    const headerInput = row.querySelector(".header-tooltip-header");
    const textInput = row.querySelector(".header-tooltip-text");
    const formulaInput = row.querySelector(".header-tooltip-formula");
    const header = (headerInput?.value || "").trim();
    const tooltip = (textInput?.value || "").trim();
    const formula = (formulaInput?.value || "").trim();
    if (header && (tooltip || formula)) {
      if (tooltip) {
        tooltipMap[header] = tooltip;
      }
      if (formula) {
        formulaMap[header] = formula;
      }
    }
  });

  return { headerTooltips: tooltipMap, headerFormulas: formulaMap };
}

function readSplitColumnGroups() {
  const rows = document.querySelectorAll(
    "#split-columns-list .split-column-row",
  );
  const groups = [];

  rows.forEach((row) => {
    const parent =
      row.querySelector(".split-column-parent")?.value?.trim() || "";
    const childrenRaw =
      row.querySelector(".split-column-children")?.value || "";
    const children = childrenRaw
      .split(",")
      .map((v) => v.trim())
      .filter(Boolean);

    if (parent && children.length > 0) {
      groups.push({ parent, children });
    }
  });

  return groups;
}

window.refreshBuilderHeaderTooltipSuggestions = refreshHeaderTooltipSuggestions;

function getStoredChoroplethSQLPreviewHeaders() {
  const section = document.getElementById("section-select-columns");
  if (!section?.dataset.choroplethSqlHeaders) return [];

  try {
    const parsed = JSON.parse(section.dataset.choroplethSqlHeaders);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function storeChoroplethSQLPreviewHeaders(headers) {
  const section = document.getElementById("section-select-columns");
  if (!section) return;

  section.dataset.choroplethSqlHeaders = JSON.stringify(
    Array.isArray(headers) ? headers : [],
  );
}

function syncChoroplethSQLValueSelectionFromCheckboxes() {
  const valueSelect = document.getElementById("choropleth-value-column");
  if (!valueSelect) return;

  const selectedCheckbox = document.querySelector(
    "#select-columns-list .select-column-checkbox:checked",
  );
  valueSelect.value = selectedCheckbox?.value || "";
}

function syncChoroplethSQLValueSelectionFromMapping() {
  const districtSelect = document.getElementById("choropleth-district-column");
  const valueSelect = document.getElementById("choropleth-value-column");
  if (!districtSelect || !valueSelect) return;

  if (districtSelect.value && districtSelect.value === valueSelect.value) {
    const fallbackValue =
      getStoredChoroplethSQLPreviewHeaders().find(
        (header) => header !== districtSelect.value,
      ) || "";
    valueSelect.value = fallbackValue;
  }

  refreshSelectColumnsList();
}

/**
 * Build a heatmap rule row HTML string.
 * Uses a <select> dropdown when columns are available, falls back to <input> for raw SQL types.
 */
function buildHeatmapRuleRowHTML(selectedColumn, low, high, selectedColors) {
  const compType = document.getElementById("comp-type")?.value || "";
  const columns = compType === "table_advanced" ? [] : getAvailableColumns();

  let columnField;
  if (columns.length > 0) {
    const options = columns
      .map((col) => {
        const escaped = escapeHtml(col);
        return `<option value="${escaped}" ${col === selectedColumn ? "selected" : ""}>${escaped}</option>`;
      })
      .join("");
    columnField = `<select class="form-input heatmap-rule-column" style="flex: 1; min-width: 100px;">
            <option value="">-- Column --</option>
            ${options}
        </select>`;
  } else {
    columnField = `<input type="text" class="form-input heatmap-rule-column" value="${escapeHtml(selectedColumn || "")}" placeholder="Column name (run preview for suggestions)" list="heatmap-column-suggestions" style="flex: 1; min-width: 100px;">`;
  }

  const colors = selectedColors || "red-yellow-green";
  const showWarn =
    low !== "" &&
    high !== "" &&
    !isNaN(low) &&
    !isNaN(high) &&
    Number(low) >= Number(high);
  return `
        <div class="heatmap-rule-row" style="margin-bottom: 8px;">
            <div style="display: flex; gap: 8px; align-items: center; flex-wrap: wrap;">
                ${columnField}
                <input type="number" class="form-input heatmap-rule-low" value="${low ?? ""}" placeholder="Low" style="width: 70px;">
                <input type="number" class="form-input heatmap-rule-high" value="${high ?? ""}" placeholder="High" style="width: 70px;">
                <select class="form-input heatmap-rule-colors" style="width: 130px;">
                    <option value="red-yellow-green" ${colors === "red-yellow-green" ? "selected" : ""}>Low = Bad</option>
                    <option value="green-yellow-red" ${colors === "green-yellow-red" ? "selected" : ""}>High = Bad</option>
                </select>
                <button class="btn-small btn-remove-heatmap-rule" style="background: #ff6b6b; color: white; padding: 6px 8px;">×</button>
            </div>
            <small class="heatmap-rule-warning" style="display: ${showWarn ? "block" : "none"}; color: #b45309; font-size: 11px; margin-top: 4px;">Low threshold should be less than High — the middle color band won't appear otherwise.</small>
        </div>
    `;
}

/**
 * Save component from form
 */
export function saveComponent() {
  const type = document.getElementById("comp-type").value;
  const title = document.getElementById("comp-title").value;
  const description = document.getElementById("comp-description")?.value || "";
  const table = document.getElementById("comp-table")?.value || "";
  const content = document.getElementById("comp-content")?.value || "";

  if (!type) {
    showError("Component type is required");
    return;
  }

  const component = {
    type,
    title,
    description,
    content: type === "text" ? content || "<p>{{value}}</p>" : "",
    data: {},
    query: null,
  };

  if (type === "table" || type === "table_advanced") {
    const { headerTooltips, headerFormulas } = readHeaderTooltipMap();
    if (Object.keys(headerTooltips).length > 0) {
      component.headerTooltips = headerTooltips;
    }
    if (Object.keys(headerFormulas).length > 0) {
      component.headerFormulas = headerFormulas;
    }

    const splitColumns = readSplitColumnGroups();
    if (splitColumns.length > 0) {
      component.splitColumns = splitColumns;
    }

    const tableHeaderWrap =
      document.getElementById("table-header-wrap")?.checked || false;
    if (tableHeaderWrap) {
      component.tableHeaderWrap = true;
    }

    const tableHeaderStyle =
      document.getElementById("table-header-style")?.value || "default";
    if (tableHeaderStyle !== "default") {
      component.tableHeaderStyle = tableHeaderStyle;
    }

    const tableCellBoundaries =
      document.getElementById("table-vertical-lines")?.checked || false;
    if (tableCellBoundaries) {
      component.tableCellBoundaries = true;
      component.tableVerticalLines = true;
    }

    const tablePageSize = readTablePageSizeFromForm();
    if (tablePageSize !== 20) {
      component.pageSize = tablePageSize;
    }
  }

  // Handle infobox type (titled callout, optional data source for {{placeholders}})
  if (type === "infobox") {
    component.type = "infobox";
    component.query = null;

    // Sync color inputs: text field overrides color picker if both differ
    const colorPicker = document.getElementById("infobox-accent-color");
    const colorText = document.getElementById("infobox-accent-color-text");
    const accentColor =
      colorText?.value?.trim() || colorPicker?.value || "#2563a8";
    component.accentColor = accentColor;
    component.infoboxBody =
      document.getElementById("infobox-body")?.value || "";

    // Threshold rules
    const thresholdColumn =
      document.getElementById("infobox-threshold-column")?.value?.trim() || "";
    if (thresholdColumn) component.thresholdColumn = thresholdColumn;
    else delete component.thresholdColumn;
    const thresholds = readThresholdsFromForm();
    if (thresholds.length > 0) component.thresholds = thresholds;
    else delete component.thresholds;

    // Optional SQL data source for {{placeholder}} substitution in infoboxBody
    const infoboxDataMode =
      document.getElementById("text-data-mode")?.value || "none";
    const infoboxSql = document.getElementById("text-sql")?.value?.trim() || "";
    if (infoboxDataMode === "sql" && infoboxSql) {
      component.sql = infoboxSql;
    } else {
      delete component.sql;
    }

    const { sectionIndex, compIndex } = currentEditingComponent;
    if (compIndex === null) {
      reportState.sections[sectionIndex].components.push(component);
    } else {
      reportState.sections[sectionIndex].components[compIndex] = component;
    }
    closeComponentModal();
    onRenderSections();
    onUpdatePreview();
    return;
  }

  // Handle plain_text type (static content, no query builder)
  if (type === "plain_text") {
    component.type = "text";
    component.content = content || "";
    component.query = null;

    const textDataMode =
      document.getElementById("text-data-mode")?.value || "none";
    const sql = document.getElementById("text-sql")?.value?.trim() || "";
    if (textDataMode === "sql" && sql) {
      component.sql = sql;
    } else {
      delete component.sql;
    }

    const { sectionIndex, compIndex } = currentEditingComponent;
    if (compIndex === null) {
      reportState.sections[sectionIndex].components.push(component);
    } else {
      reportState.sections[sectionIndex].components[compIndex] = component;
    }
    closeComponentModal();
    onRenderSections();
    onUpdatePreview();
    return;
  }

  // Add bar options for bar/bar_line charts
  if (type === "bar" || type === "bar_line") {
    const stacked = document.getElementById("bar-stacked")?.checked || false;
    const horizontal =
      document.getElementById("bar-horizontal")?.checked || false;
    const showValuesWithAxis =
      document.getElementById("bar-show-values-with-axis")?.checked || false;
    const showValues = showValuesWithAxis
      ? false
      : (document.getElementById("bar-show-values")?.checked ?? true);
    const showAxisTitles =
      document.getElementById("bar-show-axis-titles")?.checked ?? true;
    if (stacked) component.stacked = true;
    if (horizontal) component.horizontal = true;
    if (showValuesWithAxis) {
      component.showValuesWithAxis = true;
      delete component.showValues;
    } else {
      component.showValues = showValues;
      delete component.showValuesWithAxis;
    }
    component.showAxisTitles = showAxisTitles;
    delete component.hideAxes;
    const wrapLabels =
      document.getElementById("bar-wrap-labels")?.checked || false;
    if (wrapLabels) component.wrapLabels = true;
    else delete component.wrapLabels;
  }

  // Line charts reuse the value-label and axis-title controls from the bar options section.
  if (type === "line") {
    const showValuesWithAxis =
      document.getElementById("bar-show-values-with-axis")?.checked || false;
    if (showValuesWithAxis) component.showValuesWithAxis = true;
    else delete component.showValuesWithAxis;
    const showAxisTitles =
      document.getElementById("bar-show-axis-titles")?.checked ?? true;
    component.showAxisTitles = showAxisTitles;
    delete component.hideAxes;
  }

  // Pyramid: always stacked, horizontal, and flagged
  if (type === "pyramid") {
    component.stacked = true;
    component.horizontal = true;
    component.pyramid = true;
  }

  // Add comparison for text/KPI components
  if (type === "text" || type === "kpi") {
    const comparison = document.getElementById("comp-comparison")?.value || "";
    if (comparison) {
      component.comparison = comparison;
    }

    // Add target comparison
    const targetValue = parseFloat(
      document.getElementById("target-value")?.value,
    );
    const targetLabel =
      document.getElementById("target-label")?.value?.trim() || "";
    if (!isNaN(targetValue)) {
      component.target = { value: targetValue };
      if (targetLabel) {
        component.target.label = targetLabel;
      }
    }
  }

  // Read kpi-specific settings from the kpi form panel and save.
  if (type === "kpi") {
    const kpiSqlEnabled = document.getElementById("kpi-sql-enabled")?.checked;
    if (kpiSqlEnabled) {
      const kpiSql = document.getElementById("kpi-sql")?.value?.trim() || "";
      if (!kpiSql) {
        showError("SQL query is required when using custom SQL for KPI Card");
        return;
      }
      component.sql = kpiSql;
    } else {
      // Query builder mode — clear any stale SQL so the server uses component.query
      delete component.sql;
    }

    const kpiColorText = document.getElementById("kpi-accent-color-text");
    const kpiColorPicker = document.getElementById("kpi-accent-color");
    // Text input is the source of truth; color picker syncs to it.
    const accentColor =
      kpiColorText?.value?.trim() || kpiColorPicker?.value?.trim() || "";
    if (accentColor) component.accentColor = accentColor;
    else delete component.accentColor;

    const unit = document.getElementById("kpi-unit")?.value?.trim() || "";
    if (unit) component.unit = unit;
    else delete component.unit;

    const lowerIsBetter =
      document.getElementById("kpi-lower-is-better")?.checked || false;
    if (lowerIsBetter) component.lowerIsBetter = true;
    else delete component.lowerIsBetter;

    const valueTemplate =
      document.getElementById("kpi-value-template")?.value?.trim() || "";
    if (valueTemplate) component.valueTemplate = valueTemplate;
    else delete component.valueTemplate;

    const tone = document.getElementById("kpi-tone")?.value || "";
    if (tone) component.tone = tone;
    else delete component.tone;

    if (kpiSqlEnabled) {
      // SQL mode: save immediately without running query builder logic
      const { sectionIndex, compIndex } = currentEditingComponent;
      if (compIndex === null) {
        reportState.sections[sectionIndex].components.push(component);
      } else {
        reportState.sections[sectionIndex].components[compIndex] = component;
      }
      closeComponentModal();
      onRenderSections();
      onUpdatePreview();
      return;
    }
    // Query builder mode: fall through to the standard query save logic below
  }

  // For legacy type:text KPI cards there are no form inputs for lowerIsBetter — preserve it.
  if (type === "text" && currentEditingComponent?.component?.lowerIsBetter) {
    component.lowerIsBetter = currentEditingComponent.component.lowerIsBetter;
  }

  // Add reference lines for bar/line/bar_line/pyramid charts
  if (
    type === "bar" ||
    type === "line" ||
    type === "bar_line" ||
    type === "pyramid"
  ) {
    const refLineRows = document.querySelectorAll(
      "#referencelines-list .refline-row",
    );
    const referenceLines = [];
    refLineRows.forEach((row) => {
      const value = parseFloat(row.querySelector(".refline-value")?.value);
      const label = row.querySelector(".refline-label")?.value?.trim() || "";
      const color = row.querySelector(".refline-color")?.value || "#22c55e";
      const style = row.querySelector(".refline-style")?.value || "dashed";
      const axis = row.querySelector(".refline-axis")?.value || "y";

      if (!isNaN(value)) {
        const line = { value, label, color, style };
        if (axis === "y1") line.axis = "y1";
        referenceLines.push(line);
      }
    });
    if (referenceLines.length > 0) {
      component.referenceLines = referenceLines;
    }
  }

  // Add trend line for bar/line charts
  if (type === "bar" || type === "line") {
    const trendLine = document.getElementById("trend-line")?.checked || false;
    if (trendLine) {
      component.trendLine = true;
      const trendLineColor =
        document.getElementById("trend-line-color")?.value || "";
      if (trendLineColor) {
        component.trendLineColor = trendLineColor;
      }
    }
  }

  // Add axis labels for bar/line/bar_line charts
  if (type === "bar" || type === "line" || type === "bar_line") {
    const axisX = document.getElementById("axislabels-x")?.value?.trim() || "";
    const axisY = document.getElementById("axislabels-y")?.value?.trim() || "";
    const axisY1 =
      document.getElementById("axislabels-y1")?.value?.trim() || "";
    if (axisX || axisY || axisY1) {
      component.axisLabels = {};
      if (axisX) component.axisLabels.x = axisX;
      if (axisY) component.axisLabels.y = axisY;
      if (axisY1) component.axisLabels.y1 = axisY1;
    }

    const axisTooltipX =
      document.getElementById("axis-tooltip-x")?.value?.trim() || "";
    const axisTooltipY =
      document.getElementById("axis-tooltip-y")?.value?.trim() || "";
    if (axisTooltipX || axisTooltipY) {
      component.axisTooltips = {};
      if (axisTooltipX) component.axisTooltips.x = axisTooltipX;
      if (axisTooltipY) component.axisTooltips.y = axisTooltipY;
    }

    const axisFormulaX =
      document.getElementById("axis-formula-x")?.value?.trim() || "";
    const axisFormulaY =
      document.getElementById("axis-formula-y")?.value?.trim() || "";
    if (axisFormulaX || axisFormulaY) {
      component.axisFormulas = {};
      if (axisFormulaX) component.axisFormulas.x = axisFormulaX;
      if (axisFormulaY) component.axisFormulas.y = axisFormulaY;
    }
  }

  // Add y-axis limit for bar/line/bar_line charts
  if (type === "bar" || type === "line" || type === "bar_line") {
    const readNum = (id) => {
      const raw = document.getElementById(id)?.value?.trim();
      if (raw === undefined || raw === "") return undefined;
      const n = Number(raw);
      return Number.isFinite(n) ? n : undefined;
    };
    const yMin = readNum("ylimit-min");
    const yMax = readNum("ylimit-max");
    const y1Min = readNum("ylimit-y1-min");
    const y1Max = readNum("ylimit-y1-max");
    const yLimit = {};
    if (yMin !== undefined) yLimit.min = yMin;
    if (yMax !== undefined) yLimit.max = yMax;
    if (y1Min !== undefined || y1Max !== undefined) {
      yLimit.y1 = {};
      if (y1Min !== undefined) yLimit.y1.min = y1Min;
      if (y1Max !== undefined) yLimit.y1.max = y1Max;
    }
    if (Object.keys(yLimit).length > 0) {
      component.yLimit = yLimit;
    }
  }

  // Handle table_advanced type (raw SQL)
  if (type === "table_advanced") {
    const sql = document.getElementById("advanced-sql")?.value?.trim() || "";
    if (!sql) {
      showError("SQL query is required for Advanced Table");
      return;
    }
    component.sql = sql;
    // Note: filterTable is no longer needed - filters are auto-detected from SQL

    // firstRowIsHeader: promote first SQL row to column headers
    if (document.getElementById("first-row-is-header")?.checked) {
      component.firstRowIsHeader = true;
    }

    // Pivot mode
    const pivotCfg = buildPivotConfigFromForm();
    if (pivotCfg) {
      component.pivot = pivotCfg;
    }

    // Heatmap for table_advanced
    const advancedHeatmap = readHeatmapConfig();
    if (advancedHeatmap) component.heatmap = advancedHeatmap;

    // Save component
    const { sectionIndex, compIndex } = currentEditingComponent;
    if (compIndex === null) {
      reportState.sections[sectionIndex].components.push(component);
    } else {
      reportState.sections[sectionIndex].components[compIndex] = component;
    }

    closeComponentModal();
    onRenderSections();
    onUpdatePreview();
    return;
  }

  // Handle choropleth type with custom SQL
  if (type === "choropleth") {
    const sqlEnabled = document.getElementById(
      "choropleth-sql-enabled",
    )?.checked;

    if (sqlEnabled) {
      const sql =
        document.getElementById("choropleth-sql")?.value?.trim() || "";
      const districtColumn =
        document.getElementById("choropleth-district-column")?.value || "";
      const selectedValueColumn =
        document.querySelector(
          "#select-columns-list .select-column-checkbox:checked",
        )?.value || "";
      const valueColumn =
        selectedValueColumn ||
        document.getElementById("choropleth-value-column")?.value ||
        "";

      if (!sql) {
        showError("SQL query is required when using custom SQL for choropleth");
        return;
      }
      if (!districtColumn || !valueColumn) {
        showError(
          "Please preview the SQL and map the district and value columns",
        );
        return;
      }
      if (districtColumn === valueColumn) {
        showError(
          "District and value columns must be different for a choropleth map",
        );
        return;
      }

      component.sql = sql;
      component.columnMapping = {
        district: districtColumn,
        value: valueColumn,
      };
      // Clear query since we're using custom SQL
      component.query = null;

      // Add geojson and visual settings
      component.data = component.data || {};
      component.data.geojson_path = "/assets/uganda_districts.geojson";
      component.data.district_values = {};

      // Set value type (categorical or numeric)
      if (isCategoricalMode()) {
        component.valueType = "categorical";
      } else {
        delete component.valueType;
      }

      const binConfig = readBinConfig();
      component.data.color_scheme = binConfig.colorScheme;
      component.data.bin_labels = binConfig.binLabels;
      if (binConfig.binEdges) {
        component.data.bin_edges = binConfig.binEdges;
      } else {
        delete component.data.bin_edges;
      }

      const legendTitle =
        document.getElementById("legend-title")?.value?.trim() || "Value Range";
      component.data.legend_title = legendTitle;

      const facetEnabled = document.getElementById("facet-enabled")?.checked;
      if (facetEnabled) {
        const periodType =
          document.getElementById("facet-period-type")?.value || "month";
        const count =
          parseInt(document.getElementById("facet-count")?.value) || 4;
        component.facet = {
          periodType,
          count,
        };
      }

      // Save component
      const { sectionIndex, compIndex } = currentEditingComponent;
      if (compIndex === null) {
        reportState.sections[sectionIndex].components.push(component);
      } else {
        reportState.sections[sectionIndex].components[compIndex] = component;
      }

      closeComponentModal();
      onRenderSections();
      onUpdatePreview();
      return;
    }
  }

  // Handle chart types with custom SQL (bar, line, bar_line, pie, pyramid)
  const chartSqlTypes = ["bar", "line", "bar_line", "pie", "pyramid"];
  if (chartSqlTypes.includes(type)) {
    const chartSqlEnabled =
      document.getElementById("chart-sql-enabled")?.checked;
    if (chartSqlEnabled) {
      const sql = document.getElementById("chart-sql")?.value?.trim() || "";
      if (!sql) {
        showError("SQL query is required when using custom SQL");
        return;
      }
      component.sql = sql;
      // Backend honors component.query.periodLimit even in raw SQL mode
      // (server/reports.go ~line 1277). Preserve only periodLimit; drop
      // the rest of the structured query, which doesn't apply.
      const sqlPeriodLimit =
        parseInt(document.getElementById("period-limit")?.value, 10) || 0;
      component.query =
        sqlPeriodLimit > 0 ? { periodLimit: sqlPeriodLimit } : null;

      // For bar_line, build data.datasets from the mapping UI
      if (type === "bar_line") {
        const dsRows = document.querySelectorAll(
          "#chart-sql-datasets-list .chart-sql-dataset-row",
        );
        if (dsRows.length > 0) {
          const datasets = [];
          dsRows.forEach((row) => {
            const label =
              row.querySelector(".chart-sql-ds-label")?.value?.trim() || "";
            const chartType =
              row.querySelector(".chart-sql-ds-type")?.value || "bar";
            const color =
              row.querySelector(".chart-sql-ds-color")?.value || "#3498db";
            if (label) {
              datasets.push({ label, data: [], chartType, color });
            }
          });
          if (datasets.length > 0) {
            component.data = component.data || {};
            component.data.datasets = datasets;
          }
        }
      }

      // Collect legend config (same logic as the query path below)
      const sqlLegendShow = document.getElementById("legend-show")?.checked;
      const sqlLegendPosition =
        document.getElementById("legend-position")?.value || "top";
      if (sqlLegendShow === false || sqlLegendPosition !== "top") {
        component.data = component.data || {};
        component.data.legend = {
          show: sqlLegendShow !== false,
          position: sqlLegendPosition,
        };
      }

      // Save component
      const { sectionIndex, compIndex } = currentEditingComponent;
      if (compIndex === null) {
        reportState.sections[sectionIndex].components.push(component);
      } else {
        reportState.sections[sectionIndex].components[compIndex] = component;
      }

      closeComponentModal();
      onRenderSections();
      onUpdatePreview();
      return;
    }
  }

  // Build query if table is selected
  if (table) {
    const aggregations = [];

    // Build column selections map
    const columnSelections = {};
    document
      .querySelectorAll("#select-columns-list .select-column-row")
      .forEach((row) => {
        const col = row.dataset.column;
        if (col) {
          columnSelections[col] = {
            checked:
              row.querySelector(".select-column-checkbox")?.checked ?? true,
            color:
              row.querySelector(".select-column-color")?.value || "#3498db",
            chartType:
              row.querySelector(".select-column-charttype")?.value || "",
          };
        }
      });

    // Read aggregations from form
    const aggRows = document.querySelectorAll("#aggregations-list > div");
    aggRows.forEach((row, index) => {
      const colSelect = row.querySelector(".agg-column");
      const funcSelect = row.querySelector(".agg-function");
      const aliasInput = row.querySelector(".agg-alias");
      const roundInput = row.querySelector(".agg-roundto");

      if (colSelect && funcSelect && aliasInput) {
        const col = colSelect.value?.trim() || "";
        const func = funcSelect.value || "sum";
        const alias = aliasInput.value?.trim() || "";
        const selection = columnSelections[alias] || {};
        const color = selection.color || "#3498db";
        const chartType = selection.chartType || (index === 0 ? "bar" : "line");
        const roundToRaw = roundInput?.value;

        if (col && alias) {
          const agg = {
            column: col,
            function: func,
            alias: alias,
          };
          if (color) agg.color = color;
          if (chartType) agg.chartType = chartType;
          // roundTo is ignored for COUNT/COUNT_DISTINCT server-side,
          // so don't persist it there either.
          if (
            roundToRaw !== "" &&
            roundToRaw != null &&
            func !== "count" &&
            func !== "count_distinct"
          ) {
            const n = parseInt(roundToRaw, 10);
            if (!isNaN(n) && n >= 0) agg.roundTo = n;
          }
          aggregations.push(agg);
        }
      }
    });

    component.query = {
      table: table,
      aggregations: aggregations,
    };

    // Add WHERE clause if provided
    const whereClause =
      document.getElementById("comp-where")?.value?.trim() || "";
    if (whereClause) {
      component.query.where = whereClause;
    }

    // Read all groupBy fields from the list
    const groupByRows = document.querySelectorAll("#groupby-list .groupby-row");
    const groupByList = [];
    groupByRows.forEach((row) => {
      const field = row.querySelector(".groupby-field")?.value?.trim() || "";
      const format = row.querySelector(".groupby-format")?.value || "";
      const alias = row.querySelector(".groupby-alias")?.value?.trim() || "";
      if (field) {
        groupByList.push({
          field: field,
          format: format || undefined,
          alias: alias || undefined,
        });
      }
    });
    if (groupByList.length > 0) {
      component.query.groupBy = groupByList;
    }

    // Add seriesBy if provided
    const seriesByField =
      document.getElementById("seriesby-field")?.value?.trim() || "";
    if (seriesByField) {
      component.query.seriesBy = seriesByField;
    }

    // Add calculations
    const calculations = [];
    const calcRows = document.querySelectorAll(
      "#calculations-list .calculation-row",
    );
    calcRows.forEach((row) => {
      const formula = row.querySelector(".calc-formula")?.value?.trim();
      if (formula) {
        const calc = { formula: formula };
        const roundTo = row.querySelector(".calc-roundto")?.value;
        const whenZero = row.querySelector(".calc-whenzero")?.value;
        const resultAlias = row.querySelector(".calc-alias")?.value?.trim();

        if (roundTo !== "" && roundTo != null) calc.roundTo = parseInt(roundTo);
        if (whenZero) calc.whenZero = whenZero;
        if (resultAlias) {
          calc.resultAlias = resultAlias;
          const selection = columnSelections[resultAlias] || {};
          if (selection.color) calc.color = selection.color;
          if (selection.chartType) calc.chartType = selection.chartType;
        }

        calculations.push(calc);
      }
    });
    if (calculations.length > 0) {
      component.query.calculate = calculations;
    }

    // Add orderBy
    const orderByField =
      document.getElementById("orderby-field")?.value?.trim() || "";
    if (orderByField) {
      const orderByDirection =
        document.getElementById("orderby-direction")?.value || "DESC";
      component.query.orderBy = [
        {
          field: orderByField,
          direction: orderByDirection,
        },
      ];
    }

    // Add selectColumns only if user has deselected some AND seriesBy is not set
    // (seriesBy requires the series column to remain in the result — selectColumns strips it)
    const allColumnCheckboxes = document.querySelectorAll(
      "#select-columns-list .select-column-checkbox",
    );
    const checkedColumnCheckboxes = document.querySelectorAll(
      "#select-columns-list .select-column-checkbox:checked",
    );
    if (
      !seriesByField &&
      checkedColumnCheckboxes.length > 0 &&
      checkedColumnCheckboxes.length < allColumnCheckboxes.length
    ) {
      component.query.selectColumns = Array.from(checkedColumnCheckboxes).map(
        (cb) => cb.value,
      );
    }

    // Add transpose configuration (for tables only)
    if (type === "table") {
      const transposeEnabled =
        document.getElementById("transpose-enabled")?.checked || false;

      if (transposeEnabled) {
        const rowLabel =
          document.getElementById("transpose-row-label")?.value || "";

        component.query.transpose = {
          enabled: true,
        };
        // Add rowLabel if specified (otherwise backend defaults to "Indicator")
        if (rowLabel) {
          component.query.transpose.rowLabel = rowLabel;
        }
      }
    }

    // Add periodLimit
    const periodLimit =
      parseInt(document.getElementById("period-limit")?.value) || 0;
    if (periodLimit > 0) {
      component.query.periodLimit = periodLimit;
    }

    // Build data block for colors and legend
    if (
      type === "line" ||
      type === "bar" ||
      type === "pie" ||
      type === "bar_line"
    ) {
      const datasets = [];

      aggregations.forEach((agg, index) => {
        const ds = {
          label: agg.alias,
          data: [],
          color: agg.color || "#3498db",
        };
        if (type === "bar_line") {
          ds.chartType = agg.chartType || (index === 0 ? "bar" : "line");
        }
        datasets.push(ds);
      });

      if (type === "bar_line" && calculations.length > 0) {
        calculations.forEach((calc) => {
          if (calc.resultAlias) {
            const ds = {
              label: calc.resultAlias,
              data: [],
              color: calc.color || "#9b59b6",
              chartType: calc.chartType || "line",
            };
            datasets.push(ds);
          }
        });
      }

      if (datasets.length > 0) {
        component.data = { datasets };
      }

      // Add legend configuration
      const legendShow = document.getElementById("legend-show")?.checked;
      const legendPosition =
        document.getElementById("legend-position")?.value || "top";
      // Only add legend config if user has explicitly changed from defaults
      if (legendShow === false || legendPosition !== "top") {
        component.data = component.data || {};
        component.data.legend = {
          show: legendShow !== false,
          position: legendPosition,
        };
      }
    }

    // Add geojson_path for choropleth/map (always use Uganda districts)
    if (type === "choropleth" || type === "map") {
      component.data = component.data || {};
      component.data.geojson_path = "/assets/uganda_districts.geojson";
      component.data.district_values = {};

      // Set value type (categorical or numeric)
      if (isCategoricalMode()) {
        component.valueType = "categorical";
      } else {
        delete component.valueType;
      }

      const binConfig = readBinConfig();
      component.data.color_scheme = binConfig.colorScheme;
      component.data.bin_labels = binConfig.binLabels;
      if (binConfig.binEdges) {
        component.data.bin_edges = binConfig.binEdges;
      } else {
        delete component.data.bin_edges;
      }

      const legendTitle =
        document.getElementById("legend-title")?.value?.trim() || "Value Range";
      component.data.legend_title = legendTitle;

      if (type === "choropleth") {
        const facetEnabled = document.getElementById("facet-enabled")?.checked;
        if (facetEnabled) {
          const periodType =
            document.getElementById("facet-period-type")?.value || "month";
          const count =
            parseInt(document.getElementById("facet-count")?.value) || 4;
          component.facet = {
            periodType: periodType,
            count: count,
          };
        }
      }
    }

    if (aggregations.length === 0) {
      showError("At least one aggregation is required for " + type);
      return;
    }
  }

  // Heatmap for table type (general query path)
  if (type === "table") {
    const tableHeatmap = readHeatmapConfig();
    if (tableHeatmap) component.heatmap = tableHeatmap;
  }

  // If saving an infobox component in query-builder mode, restore infobox-specific properties.
  // (comp-type was temporarily set to 'text' so the query-builder form fields would be visible.)
  const _leftPanelType = document.querySelector(".type-panel-item.selected")
    ?.dataset.type;
  if (_leftPanelType === "infobox") {
    component.type = "infobox";
    component.infoboxBody =
      document.getElementById("infobox-body")?.value || "";
    const _colorText = document.getElementById("infobox-accent-color-text");
    const _colorPicker = document.getElementById("infobox-accent-color");
    component.accentColor =
      _colorText?.value?.trim() || _colorPicker?.value || "#2563a8";
    delete component.content;
    delete component.comparison;

    // Threshold rules (query-builder infobox path)
    const _thresholdColumn =
      document.getElementById("infobox-threshold-column")?.value?.trim() || "";
    if (_thresholdColumn) component.thresholdColumn = _thresholdColumn;
    else delete component.thresholdColumn;
    const _thresholds = readThresholdsFromForm();
    if (_thresholds.length > 0) component.thresholds = _thresholds;
    else delete component.thresholds;
  }

  // Save component
  const { sectionIndex, compIndex } = currentEditingComponent;
  if (compIndex === null) {
    reportState.sections[sectionIndex].components.push(component);
  } else {
    reportState.sections[sectionIndex].components[compIndex] = component;
  }

  closeComponentModal();
  onRenderSections();
  onUpdatePreview();
}

// Component type tiers — progressive disclosure of component types
const COMPONENT_TYPE_TIERS = [
  {
    label: "Essentials",
    collapsed: false,
    types: [
      {
        type: "bar",
        name: "Bar Chart",
        desc: "District or category comparison",
      },
      { type: "line", name: "Line Chart", desc: "Monthly or weekly trends" },
      { type: "table", name: "Table", desc: "Aggregated summary table" },
      { type: "kpi", name: "KPI Card", desc: "Single headline number" },
    ],
  },
  {
    label: "More",
    collapsed: true,
    types: [
      { type: "pie", name: "Pie Chart", desc: "Proportional breakdown" },
      {
        type: "bar_line",
        name: "Bar-Line Combo",
        desc: "Bars with trend overlay",
      },
      {
        type: "pyramid",
        name: "Population Pyramid",
        desc: "Age/sex mirrored bar chart",
      },
      { type: "choropleth", name: "Map", desc: "Geographic district heatmap" },
      {
        type: "table_advanced",
        name: "Advanced Table",
        desc: "Custom SQL data grid",
      },
      {
        type: "plain_text",
        name: "Plain Text",
        desc: "Static description or content",
      },
      {
        type: "infobox",
        name: "Info Box",
        desc: "Titled callout with accent colour",
      },
    ],
  },
];

// Flat lookup derived from tiers for backward compat
const COMPONENT_TYPES = {};
COMPONENT_TYPE_TIERS.forEach((tier) => {
  tier.types.forEach((t) => {
    COMPONENT_TYPES[t.type] = { name: t.name, desc: t.desc };
  });
});

/**
 * Render the tiered type selector HTML (legacy grid — kept for backward compat)
 */
function renderTypeSelectorHTML(selectedType) {
  return COMPONENT_TYPE_TIERS.map((tier, tierIndex) => {
    const tierContainsSelected = tier.types.some(
      (t) => t.type === selectedType,
    );
    const isCollapsible = tier.collapsed;
    const shouldExpand = tierContainsSelected;

    const buttons = tier.types
      .map(
        (t) => `
            <button type="button" class="type-option ${t.type === selectedType ? "selected" : ""}" data-type="${t.type}">
                <span class="type-name">${t.name}</span>
                <span class="type-desc">${t.desc}</span>
            </button>
        `,
      )
      .join("");

    if (!isCollapsible) {
      return `
                <div>
                    <div class="type-tier-label-static">${tier.label}</div>
                    <div class="type-tier-grid">${buttons}</div>
                </div>
            `;
    }

    return `
            <div>
                <button type="button" class="type-tier-toggle" data-tier-index="${tierIndex}">
                    <span>${tier.label}</span>
                    <span class="tier-arrow">${shouldExpand ? "▼" : "▶"}</span>
                </button>
                <div class="type-tier-grid" data-tier-grid="${tierIndex}" style="display: ${shouldExpand ? "grid" : "none"}; margin-top: 8px;">
                    ${buttons}
                </div>
            </div>
        `;
  }).join("");
}

/**
 * Render the left-panel vertical type list for the two-panel modal.
 */
function renderTypePanelHTML(selectedType) {
  const TYPE_META = {
    bar: { icon: "▊", color: "#3b82f6" },
    line: { icon: "∿", color: "#7c3aed" },
    table: { icon: "⊟", color: "#0891b2" },
    kpi: { icon: "#", color: "#d97706" },
    pie: { icon: "◍", color: "#16a34a" },
    bar_line: { icon: "⇅", color: "#4f46e5" },
    pyramid: { icon: "△", color: "#db2777" },
    choropleth: { icon: "◉", color: "#0f766e" },
    table_advanced: { icon: "⊞", color: "#475569" },
    plain_text: { icon: "¶", color: "#92400e" },
    infobox: { icon: "ⓘ", color: "#0369a1" },
  };

  return COMPONENT_TYPE_TIERS.map(
    (tier) => `
        <div class="type-panel-group">
            <div class="type-panel-group-label">${tier.label}</div>
            ${tier.types
              .map((t) => {
                const meta = TYPE_META[t.type] || {
                  icon: "□",
                  color: "#94a3b8",
                };
                return `
                <button type="button"
                    class="type-panel-item ${t.type === selectedType ? "selected" : ""}"
                    data-type="${t.type}"
                    title="${t.desc}">
                    <span class="type-panel-icon" style="background:${meta.color}">${meta.icon}</span>
                    <span class="type-panel-info">
                        <span class="type-panel-name">${t.name}</span>
                        <span class="type-panel-desc">${t.desc}</span>
                    </span>
                </button>`;
              })
              .join("")}
        </div>
    `,
  ).join("");
}

/**
 * Render the component form
 */
/**
 * Renders a single threshold rule row for the InfoBox Threshold Rules UI.
 */
function renderThresholdRuleRow(rule) {
  const color = escapeHtml(rule.color || "#922B21");
  const condition = escapeHtml(rule.condition || "");
  return `
    <div class="threshold-rule-row" style="display: flex; gap: 8px; align-items: center; margin-bottom: 8px;">
      <input type="text" class="form-input threshold-condition" value="${condition}" placeholder="e.g. &gt;= 1" style="width: 120px; font-family: monospace; font-size: 13px; flex-shrink: 0;">
      <input type="color" class="threshold-color-picker" value="${color}" style="width: 40px; height: 36px; border: 1px solid var(--color-border); border-radius: 4px; padding: 2px; cursor: pointer; flex-shrink: 0;">
      <input type="text" class="form-input threshold-color-text" value="${color}" placeholder="#922B21" style="width: 90px; font-family: monospace; font-size: 13px; flex-shrink: 0;">
      <button type="button" class="btn-secondary remove-threshold-rule" style="font-size: 11px; padding: 4px 10px; flex-shrink: 0;">✕</button>
    </div>`;
}

/**
 * Reads threshold rules from the current InfoBox form.
 */
function readThresholdsFromForm() {
  const rows = document.querySelectorAll(
    "#threshold-rules-list .threshold-rule-row",
  );
  const thresholds = [];
  rows.forEach((row) => {
    const condition =
      row.querySelector(".threshold-condition")?.value?.trim() || "";
    const colorText =
      row.querySelector(".threshold-color-text")?.value?.trim() || "";
    const colorPicker =
      row.querySelector(".threshold-color-picker")?.value || "";
    const color = colorText || colorPicker;
    if (condition) thresholds.push({ condition, color });
  });
  return thresholds;
}

/**
 * Wires color picker ↔ text input sync for a single threshold rule row.
 */
function setupThresholdRowListeners(row) {
  const picker = row.querySelector(".threshold-color-picker");
  const text = row.querySelector(".threshold-color-text");
  if (picker && text) {
    picker.addEventListener("input", () => {
      text.value = picker.value;
    });
    text.addEventListener("input", () => {
      const val = text.value.trim();
      if (/^#[0-9a-fA-F]{6}$/.test(val)) picker.value = val;
    });
  }
}

export function renderComponentForm(component) {
  const query = component.query || {};
  const headerTooltipEntries = Object.entries(component.headerTooltips || {});
  const splitColumnEntries = Array.isArray(component.splitColumns)
    ? component.splitColumns
    : [];
  const tablePageSizeValue = normalizeTablePageSize(component.pageSize);
  const hasCustomTableConfigs =
    headerTooltipEntries.length > 0 ||
    splitColumnEntries.length > 0 ||
    tablePageSizeValue !== 20;
  let componentType = component.type || "bar";
  if (componentType === "text" && !component.query) {
    componentType = "plain_text";
  }
  // For infobox in query-builder mode: use 'text' form type so query-builder fields are visible.
  // The left panel still shows 'infobox' selected; the fixup in saveComponent() restores the type.
  if (componentType === "infobox" && component.query && !component.sql) {
    componentType = "text";
  }

  return `
        <input type="hidden" id="comp-type" value="${componentType}">

        <div class="form-group">
            <label for="comp-title">Title <span class="help-tooltip" data-tooltip="Display title shown above the component in the report.">?</span></label>
            <input type="text" id="comp-title" class="form-input" value="${component.title || ""}" placeholder="Component title">
        </div>

        <div class="form-group">
            <label for="comp-description">Description <span class="help-tooltip" data-tooltip="Optional description shown below the title. Use for methodology notes, data sources, or interpretation guidance.">?</span></label>
            <input type="text" id="comp-description" class="form-input" value="${component.description || ""}" placeholder="Brief description (optional)">
        </div>

        <!-- Advanced SQL Section -->
        <div id="section-advanced-sql" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <h4 style="margin-bottom: 15px;">Advanced SQL Query <span class="help-tooltip" data-tooltip="Write custom SQL with CTEs, complex joins, etc. Column names from your SQL become table headers. Filters are automatically applied based on the FROM table.">?</span></h4>
            <small class="hint" style="display: block; margin-bottom: 10px;">
                Write raw SQL. Column aliases (AS "Name") become table headers.
                Use <code>{{year_filter}}</code>, <code>{{month_filter}}</code>, <code>{{district_filter}}</code>, <code>{{region_filter}}</code>, <code>{{facility_filter}}</code> for standard filter placeholders.
                ${customFilterHint()}
            </small>

            <div class="form-group">
                <label for="advanced-sql">SQL Query *</label>
                <textarea id="advanced-sql" class="form-textarea" style="font-family: monospace; font-size: 12px; min-height: 200px;" placeholder="WITH summary AS (
  SELECT district AS &quot;District&quot;,
         SUM(cases) AS &quot;Total Cases&quot;
  FROM report.my_table
  GROUP BY district
)
SELECT * FROM summary
ORDER BY &quot;Total Cases&quot; DESC">${component.sql || ""}</textarea>
            </div>

            <div style="display: flex; gap: 10px; align-items: center; margin-top: 10px;">
                <span style="font-size: 12px; color: var(--color-text-muted);">Timeout: 30s | Max rows: 1000</span>
            </div>

            <div style="margin-top: 15px;">
                <button type="button" class="btn-secondary" id="btn-preview-advanced-sql" style="width: 100%; background: var(--color-success); color: white; border-color: var(--color-success);">
                    ▶ Preview SQL Results
                </button>
            </div>

            <div id="advanced-sql-preview-result" style="margin-top: 15px; display: none;">
                <div style="background: var(--color-hover); padding: 10px; border-radius: 4px; font-size: 12px;">
                    <div id="advanced-sql-preview-info" style="margin-bottom: 10px;"></div>
                    <div id="advanced-sql-preview-table" style="max-height: 300px; overflow: auto;"></div>
                </div>
            </div>

            <!-- Pivot Mode Section -->
            <div class="form-group" style="margin-top: 18px; padding: 12px; background: var(--color-hover); border-radius: 6px; border: 1px solid var(--color-border);">
                <label class="checkbox-label" style="display: flex; align-items: flex-start; gap: 8px; cursor: pointer;">
                    <input type="checkbox" id="pivot-enabled" style="margin-top: 2px; flex-shrink: 0;" ${component.pivot ? "checked" : ""} onchange="togglePivotConfig()">
                    <span style="font-weight: 500;">Enable Pivot Mode <span class="help-tooltip" data-tooltip="Converts a 3-column SQL result into a cross-tab pivot table automatically. Preview SQL first to load column choices.">?</span></span>
                </label>
                <small class="hint" style="margin-top: 4px; display: block; padding-left: 24px;">Write SQL returning three columns (row labels, column headers, cell values) and map them below. Column order follows your SQL ORDER BY.</small>

                <div id="pivot-config" style="margin-top: 14px; display: ${component.pivot ? "block" : "none"};">
                    <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 10px; margin-bottom: 12px;">
                        <div>
                            <label style="font-size: 12px; font-weight: 500; display: block; margin-bottom: 4px;">Rows column *</label>
                            <select id="pivot-rows" class="form-input" style="font-size: 12px;">
                                <option value="">— preview SQL first —</option>
                            </select>
                        </div>
                        <div>
                            <label style="font-size: 12px; font-weight: 500; display: block; margin-bottom: 4px;">Columns column *</label>
                            <select id="pivot-columns" class="form-input" style="font-size: 12px;">
                                <option value="">— preview SQL first —</option>
                            </select>
                        </div>
                        <div>
                            <label style="font-size: 12px; font-weight: 500; display: block; margin-bottom: 4px;">Values column *</label>
                            <select id="pivot-values" class="form-input" style="font-size: 12px;">
                                <option value="">— preview SQL first —</option>
                            </select>
                        </div>
                    </div>

                    <div style="margin-bottom: 12px;">
                        <label style="font-size: 12px; font-weight: 500; display: block; margin-bottom: 4px;">Cell Format</label>
                        <select id="pivot-format" class="form-input" style="font-size: 12px; max-width: 220px;">
                            <option value="auto" ${!component.pivot?.format || component.pivot?.format === "auto" ? "selected" : ""}>Auto (as returned by SQL)</option>
                            <option value="integer" ${component.pivot?.format === "integer" ? "selected" : ""}>Integer (rounded, no decimals)</option>
                            <option value="decimal:1" ${component.pivot?.format === "decimal:1" ? "selected" : ""}>Decimal — 1 place</option>
                            <option value="decimal:2" ${component.pivot?.format === "decimal:2" ? "selected" : ""}>Decimal — 2 places</option>
                            <option value="percent:1" ${component.pivot?.format === "percent:1" ? "selected" : ""}>Percent — 1 place (%)</option>
                            <option value="percent:2" ${component.pivot?.format === "percent:2" ? "selected" : ""}>Percent — 2 places (%)</option>
                        </select>
                    </div>

                    <div style="margin-bottom: 12px;">
                        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px;">
                            <label style="font-size: 12px; font-weight: 500;">Extra Columns (computed) <span class="help-tooltip" data-tooltip="Columns appended after the pivot columns. Use formulas like [-1]-[-2] where [-1] is the last pivot column value and [-2] is the second-to-last.">?</span></label>
                            <button type="button" class="btn-secondary" style="font-size: 11px; padding: 2px 8px;" onclick="addPivotExtraColumn()">+ Add</button>
                        </div>
                        <div id="pivot-extra-columns-list" style="display: flex; flex-direction: column; gap: 6px;"></div>
                        <small class="hint" style="margin-top: 4px; display: block;">Formulas: <code>sum</code>, <code>avg</code>, or expression e.g. <code>[-1] - [-2]</code> ([-N] = Nth column from the right)</small>
                    </div>

                    <div>
                        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px;">
                            <label style="font-size: 12px; font-weight: 500;">Extra Rows (computed) <span class="help-tooltip" data-tooltip="Rows appended after the data rows, e.g. a Total row. Use sum or avg.">?</span></label>
                            <button type="button" class="btn-secondary" style="font-size: 11px; padding: 2px 8px;" onclick="addPivotExtraRow()">+ Add</button>
                        </div>
                        <div id="pivot-extra-rows-list" style="display: flex; flex-direction: column; gap: 6px;"></div>
                        <small class="hint" style="margin-top: 4px; display: block;">Formulas: <code>sum</code> (column total) or <code>avg</code> (column average)</small>
                    </div>
                </div>
            </div>

            <!-- Legacy: first row as header (hidden by default, kept for backward compatibility) -->
            <div class="form-group" id="legacy-first-row-is-header-section" style="margin-top: 8px; display: ${component.firstRowIsHeader ? "block" : "none"};">
                <label class="checkbox-label" style="display: flex; align-items: flex-start; gap: 8px; cursor: pointer; opacity: 0.65;">
                    <input type="checkbox" id="first-row-is-header" style="margin-top: 2px; flex-shrink: 0;" ${component.firstRowIsHeader ? "checked" : ""}>
                    <span style="font-size: 11px;">(Legacy) Use first row as column headers</span>
                </label>
            </div>
        </div>

        <!-- Text Data Source Section -->
        <div id="section-datasource" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <h4 style="margin-bottom: 15px;">Data Source</h4>
            <div class="form-group">
                <label for="text-data-mode">Data Source <span class="help-tooltip" data-tooltip="How to populate {{column_name}} placeholders in the HTML template. Raw SQL: write your own query. Query Builder: use the visual form below.">?</span></label>
                <select id="text-data-mode" class="form-input">
                    <option value="none" ${!component.sql && !component.query ? "selected" : ""}>None (static content only)</option>
                    <option value="sql" ${component.sql ? "selected" : ""}>Raw SQL query</option>
                    <option value="builder" ${component.query && !component.sql ? "selected" : ""}>Query Builder</option>
                </select>
                <small class="hint">Column aliases in your query (<code>AS total_vhts</code>) become <code>{{total_vhts}}</code> placeholders in the content below.</small>
            </div>
            <div id="text-sql-content" style="display: ${component.sql ? "block" : "none"};">
                <div class="form-group">
                    <label for="text-sql">SQL Query <span class="help-tooltip" data-tooltip="Write a SELECT query. The first result row is used. Column aliases (AS name) become the {{name}} placeholder in your HTML template.">?</span></label>
                    <textarea id="text-sql" class="form-textarea" style="font-family: monospace; font-size: 12px; min-height: 140px;" placeholder="SELECT\n  COUNT(DISTINCT fieldworker_id) AS total_vhts,\n  COUNT(*) AS total_forms\nFROM report.cht_fieldworker_weekly\nWHERE {{period_filter}}">${escapeHtml(component.sql || "")}</textarea>
                    <small class="hint">Use <code>{{period_filter}}</code>, <code>{{district_filter}}</code>, <code>{{facility_filter}}</code> for active dashboard filters.</small>
                </div>
            </div>
        </div>

        <!-- Period Comparison Section (for text/KPI cards) -->
        <div id="section-comparison" class="form-section" style="display: none;">
            <div class="form-group">
                <label for="comp-comparison">Period Comparison <span class="help-tooltip" data-tooltip="Compare the current value against a previous period. Shows an arrow indicator (up/down) with percentage change.">?</span></label>
                <select id="comp-comparison" class="form-input">
                    <option value="">None</option>
                    <option value="previous_period" ${component.comparison === "previous_period" ? "selected" : ""}>Previous Period</option>
                    <option value="previous_year" ${component.comparison === "previous_year" ? "selected" : ""}>Previous Year</option>
                </select>
                <small class="hint">Previous Period adapts to filter: week→week, month→month, quarter→quarter</small>
            </div>

            <hr style="margin: 15px 0; border: none; border-top: 1px dashed var(--color-border);">

            <div class="form-group">
                <label>Target Comparison <span class="help-tooltip" data-tooltip="Compare the current value against a static target. Shows percentage above/below target.">?</span></label>
                <small class="hint" style="display: block; margin-bottom: 10px;">Set a fixed target value to compare against (e.g., monthly goal of 100 visits)</small>

                <div style="display: flex; gap: 10px; align-items: flex-end;">
                    <div style="flex: 1;">
                        <label for="target-value" style="font-size: 12px;">Target Value</label>
                        <input type="number" id="target-value" class="form-input" value="${component.target?.value || ""}" placeholder="e.g., 100" step="any">
                    </div>
                    <div style="flex: 1;">
                        <label for="target-label" style="font-size: 12px;">Label (optional)</label>
                        <input type="text" id="target-label" class="form-input" value="${component.target?.label || ""}" placeholder="e.g., Monthly Target">
                    </div>
                </div>
                <small class="hint">Displays as "↑ 25% above Monthly Target" or "↓ 10% below target"</small>
            </div>
        </div>

        <!-- Query Section -->
        <div id="section-query" class="form-section">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <button type="button" class="advanced-toggle-btn" id="btn-toggle-query">
                <span>Query Configuration</span>
                <span id="query-toggle-arrow">▼</span>
            </button>
            <div id="query-content" style="display: block;">

            <div id="pyramid-guidance" style="display: none; margin-bottom: 15px; padding: 10px; background: var(--color-hover); border-radius: 4px; font-size: 12px; line-height: 1.5;">
                <strong>Pyramid setup:</strong> Add exactly 2 aggregations (e.g., Male and Female columns) and 1 group-by (e.g., age group). The first aggregation mirrors left, the second goes right.
            </div>

            <div class="form-group" style="display: flex; gap: 10px; align-items: flex-end;">
                <div style="flex: 0 0 120px;">
                    <label for="comp-schema">Schema <span class="help-tooltip" data-tooltip="Database schema. Defaults to 'report'. Change only if your data is in a different schema.">?</span></label>
                    <select id="comp-schema" class="form-input" style="font-size: 12px;">
                        ${reportState.availableSchemas.map((s) => `<option value="${s}" ${s === (query.table ? query.table.split(".")[0] : reportState.selectedSchema) ? "selected" : ""}>${s}</option>`).join("")}
                    </select>
                </div>
                <div style="flex: 1;">
                    <label for="comp-table">Table * <span class="help-tooltip" data-tooltip="Database table containing your data. Usually report.cht_form_097b for CHW reports.">?</span></label>
                    <select id="comp-table" class="form-input">
                        ${renderTableOptionsWithRecent(reportState.availableTables, query.table || "")}
                    </select>
                </div>
            </div>
            <small class="hint">Columns will appear after selecting a table</small>

            <!-- Shared column suggestions datalist -->
            <datalist id="column-suggestions"></datalist>

            <!-- Sample Data Section (Collapsible) -->
            <div id="section-sample-data" class="form-group" style="margin-top: 10px;">
                <div class="collapsible-header" onclick="toggleSampleDataSection()"
                     style="cursor: pointer; display: flex; align-items: center; justify-content: space-between; padding: 8px; background: var(--color-hover); border-radius: 4px;">
                    <span style="font-size: 12px; font-weight: 500;">Sample Data</span>
                    <span id="sampledata-toggle-icon" style="font-size: 12px; transition: transform 0.2s;">▶</span>
                </div>
                <div id="sampledata-content" style="display: none; margin-top: 8px;">
                    <div id="sampledata-table-container"
                         style="overflow-x: auto; max-height: 240px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-bg);">
                        <p style="color: var(--color-text-secondary); font-size: 12px; padding: 12px; margin: 0;">
                            Select a table to see sample data.
                        </p>
                    </div>
                    <small class="hint" style="margin-top: 4px; display: block;">
                        Click a column header to add it as an aggregation or group-by field.
                    </small>
                </div>
            </div>

            <div class="form-group">
                <label>Aggregations <span class="help-tooltip" data-tooltip="Define metrics to calculate: column name, function (SUM, COUNT, AVG), and an alias for the result.">?</span></label>
                <small class="hint" style="display: block; margin-bottom: 8px;">Enter column names or expressions (e.g., col1, col1 + col2)</small>
                <div id="aggregations-list" style="border: 1px solid var(--color-border); border-radius: 4px; padding: 10px; max-height: 250px; overflow-y: auto; margin-bottom: 10px; background: var(--color-bg);">
                    ${
                      query.aggregations && query.aggregations.length > 0
                        ? query.aggregations
                            .map(
                              (agg, i) => `
                            <div style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center;">
                                <input type="text" class="form-input agg-column" value="${agg.column || ""}" placeholder="Column or expression" list="column-suggestions" style="flex: 1; margin: 0;">
                                <select class="form-input agg-function" style="flex: 0.5; margin: 0;">
                                    <option value="sum" ${agg.function === "sum" ? "selected" : ""}>SUM</option>
                                    <option value="count" ${agg.function === "count" ? "selected" : ""}>COUNT</option>
                                    <option value="count_distinct" ${agg.function === "count_distinct" ? "selected" : ""}>COUNT DISTINCT</option>
                                    <option value="avg" ${agg.function === "avg" ? "selected" : ""}>AVG</option>
                                    <option value="max" ${agg.function === "max" ? "selected" : ""}>MAX</option>
                                    <option value="min" ${agg.function === "min" ? "selected" : ""}>MIN</option>
                                    <option value="variance" ${agg.function === "variance" ? "selected" : ""}>VARIANCE</option>
                                    <option value="stddev" ${agg.function === "stddev" ? "selected" : ""}>STDDEV</option>
                                </select>
                                <input type="text" class="form-input agg-alias" value="${agg.alias}" placeholder="Alias" style="flex: 0.8; margin: 0;">
                                <input type="number" class="form-input agg-roundto" value="${agg.roundTo != null ? agg.roundTo : 1}" placeholder="Round" min="0" max="6" style="flex: 0.4; margin: 0;" title="Decimal places (default 1; ignored for COUNT)">
                                <button class="btn-small" onclick="this.parentElement.remove()" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">×</button>
                            </div>
                          `,
                            )
                            .join("")
                        : '<p style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">No aggregations yet. Click "Add Aggregation" to start.</p>'
                    }
                </div>
                <button class="btn-secondary" id="btn-add-aggregation" style="width: 100%;">+ Add Aggregation</button>
            </div>

            <div style="margin-top: 15px;">
                <button class="btn-secondary" id="btn-preview-query" style="width: 100%; background: var(--color-success); color: white; border-color: var(--color-success);">
                    ▶ Preview Data (10 rows)
                </button>
            </div>

            </div><!-- end #query-content -->
        </div>

        <!-- Calculate Section -->
        <div id="section-calculate" class="form-section">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleCalculateSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;">Calculations<span class="help-tooltip" data-tooltip="Compute derived metrics after aggregation. E.g., (positive / tested * 100) for positivity rate.">?</span></h4>
                <span id="calc-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">▶</span>
            </div>
            <div id="calc-content" style="display: none; margin-top: 15px;">
                <small class="hint" style="display: block; margin-bottom: 10px;">Calculate derived metrics (e.g., percentages, rates). Use aggregation aliases in formulas.</small>
                <div id="calculations-list" style="border: 1px solid var(--color-border); border-radius: 4px; padding: 10px; max-height: 400px; overflow-y: auto; margin-bottom: 10px; background: var(--color-bg);">
                    ${
                      query.calculate && query.calculate.length > 0
                        ? query.calculate
                            .map((calc, i) => renderCalculationRow(calc))
                            .join("")
                        : '<p style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">No calculations yet. Click "Add Calculation" to start.</p>'
                    }
                </div>
                <button class="btn-secondary" id="btn-add-calculation" style="width: 100%;">+ Add Calculation</button>
            </div>
        </div>

        <!-- Grouping & Series Section -->
        <div id="section-groupby" class="form-section">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <button type="button" class="advanced-toggle-btn" id="btn-toggle-grouping">
                <span>Grouping &amp; Series</span>
                <span id="grouping-toggle-arrow">▶</span>
            </button>
            <div id="grouping-content" style="display: none; border-left: 3px solid var(--color-border); padding-left: 16px; margin-top: 12px;">

                <!-- Group By -->
                <div class="form-group" style="margin-top: 4px;">
                    <label>Group By <span class="help-tooltip" data-tooltip="Group results by one or more columns. For pivot tables, add multiple fields (e.g., district and month).">?</span></label>
                    <small class="hint" style="display: block; margin-bottom: 8px;">Add fields to group by. For pivot, you typically need 2 fields (e.g., district + month).</small>
                    <div id="groupby-list" style="border: 1px solid var(--color-border); border-radius: 4px; padding: 10px; max-height: 300px; overflow-y: auto; margin-bottom: 10px; background: var(--color-bg);">
                        ${
                          query.groupBy && query.groupBy.length > 0
                            ? query.groupBy
                                .map(
                                  (gb, i) => `
                                <div class="groupby-row" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center; flex-wrap: wrap;">
                                    <input type="text" class="form-input groupby-field" value="${gb.field || ""}" placeholder="Column name" list="column-suggestions" style="flex: 1; min-width: 120px; margin: 0;">
                                    <select class="form-input groupby-format" style="flex: 0 0 140px; margin: 0;">
                                        <optgroup label="Period">
                                            <option value="month" ${gb.format === "month" || gb.format === "month_year" ? "selected" : ""}>Month</option>
                                            <option value="quarter" ${gb.format === "quarter" || gb.format === "quarter_year" ? "selected" : ""}>Quarter</option>
                                            <option value="week" ${gb.format === "week" || gb.format === "week_year" ? "selected" : ""}>Week</option>
                                            <option value="year" ${gb.format === "year" ? "selected" : ""}>Year</option>
                                            <option value="date" ${gb.format === "date" ? "selected" : ""}>Date</option>
                                        </optgroup>
                                        <optgroup label="Period (no year)">
                                            <option value="month_noyear" ${gb.format === "month_noyear" ? "selected" : ""}>Month (no year)</option>
                                            <option value="week_noyear" ${gb.format === "week_noyear" ? "selected" : ""}>Week (no year)</option>
                                            <option value="quarter_noyear" ${gb.format === "quarter_noyear" ? "selected" : ""}>Quarter (no year)</option>
                                        </optgroup>
                                        <optgroup label="Other">
                                            <option value="" ${!gb.format ? "selected" : ""}>No Format</option>
                                        </optgroup>
                                    </select>
                                    <input type="text" class="form-input groupby-alias" value="${gb.alias || ""}" placeholder="Alias" style="flex: 0 0 100px; margin: 0;">
                                    <button class="btn-small btn-remove-groupby" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">×</button>
                                </div>
                              `,
                                )
                                .join("")
                            : '<p id="no-groupby-msg" style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">No group by fields yet. Click "Add Group By" to start.</p>'
                        }
                    </div>
                    <button class="btn-secondary" id="btn-add-groupby" style="width: 100%;">+ Add Group By</button>
                </div>

                <!-- Series By (bar/line/bar_line only) -->
                <div id="section-seriesby" class="form-section" style="display: none;">
                    <hr style="margin: 14px 0; border: none; border-top: 1px dashed var(--color-border);">
                    <div class="form-group">
                        <label>Series By <span class="help-tooltip" data-tooltip="Split data into multiple lines/bars by a category column. Each unique value becomes a separate dataset (e.g., one line per commodity).">?</span></label>
                        <small class="hint" style="display: block; margin-bottom: 8px;">Each unique value in this column becomes a separate line or bar group. Leave empty for standard behavior (one dataset per aggregation).</small>
                        <input type="text" id="seriesby-field" class="form-input" value="${query.seriesBy || ""}" placeholder="e.g. commodity_name, district, age_group" list="column-suggestions" style="margin: 0;">
                    </div>
                </div>

            </div>
        </div>

        <!-- Choropleth Custom SQL Section -->
        <div id="section-choropleth-sql" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleChoroplethSqlSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;">Custom SQL<span class="help-tooltip" data-tooltip="Use custom SQL instead of the query builder. After previewing, map which columns contain the district name and value.">?</span></h4>
                <span id="choropleth-sql-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">▶</span>
            </div>
            <div id="choropleth-sql-content" style="display: ${component.sql ? "block" : "none"}; margin-top: 15px;">
                <small class="hint" style="display: block; margin-bottom: 10px;">
                    Write a custom SQL query that returns district names and values. After preview, map which columns to use.
                </small>

                <div class="form-group">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer; margin-bottom: 10px;">
                        <input type="checkbox" id="choropleth-sql-enabled" ${component.sql ? "checked" : ""} onchange="toggleChoroplethSqlMode()">
                        <span>Use Custom SQL</span>
                    </label>
                </div>

                <div id="choropleth-sql-form" style="display: ${component.sql ? "block" : "none"};">
                    <div class="form-group">
                        <label for="choropleth-sql">SQL Query *</label>
                        <textarea id="choropleth-sql" class="form-textarea" style="font-family: monospace; font-size: 12px; min-height: 150px;" placeholder="SELECT district_name, SUM(cases) as total
FROM report.my_table
WHERE 1=1 {{district_filter}} {{year_filter}} {{month_filter}}
GROUP BY district_name">${component.sql || ""}</textarea>
                        <small class="hint">Use <code>{{district_filter}}</code>, <code>{{region_filter}}</code>, <code>{{facility_filter}}</code>, <code>{{year_filter}}</code>, <code>{{month_filter}}</code> for filter placeholders.${customFilterHint()}</small>
                    </div>

                    <div style="margin-top: 15px;">
                        <button type="button" class="btn-secondary" id="btn-preview-choropleth-sql" style="width: 100%; background: var(--color-success); color: white; border-color: var(--color-success);">
                            ▶ Preview SQL & Map Columns
                        </button>
                    </div>

                    <div id="choropleth-sql-preview-result" style="margin-top: 15px; display: none;">
                        <div style="background: var(--color-hover); padding: 10px; border-radius: 4px; font-size: 12px;">
                            <div id="choropleth-sql-preview-info" style="margin-bottom: 10px;"></div>
                            <div id="choropleth-sql-preview-table" style="max-height: 200px; overflow: auto;"></div>
                        </div>
                    </div>

                    <div id="choropleth-column-mapping" style="display: ${component.columnMapping ? "block" : "none"}; margin-top: 15px; padding: 15px; background: var(--color-hover); border-radius: 4px;">
                        <h5 style="margin: 0 0 12px 0; font-size: 14px;">Column Mapping *</h5>
                        <small class="hint" style="display: block; margin-bottom: 12px;">Map which columns contain the district name and value for the choropleth.</small>

                        <div class="form-group" style="margin-bottom: 12px;">
                            <label for="choropleth-district-column" style="font-size: 13px;">District Column *</label>
                            <select id="choropleth-district-column" class="form-input">
                                <option value="">-- Select column --</option>
                                ${component.columnMapping?.district ? `<option value="${component.columnMapping.district}" selected>${component.columnMapping.district}</option>` : ""}
                            </select>
                        </div>

                        <div class="form-group">
                            <label for="choropleth-value-column" style="font-size: 13px;">Value Column *</label>
                            <select id="choropleth-value-column" class="form-input">
                                <option value="">-- Select column --</option>
                                ${component.columnMapping?.value ? `<option value="${component.columnMapping.value}" selected>${component.columnMapping.value}</option>` : ""}
                            </select>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Chart Custom SQL Section -->
        <div id="section-chart-sql" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleChartSqlSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;">Custom SQL<span class="help-tooltip" data-tooltip="Use custom SQL instead of the query builder for this chart. First column is used as labels (for bar/line/pyramid), remaining columns become datasets. For pie charts, headers are labels and first row values are slices.">?</span></h4>
                <span id="chart-sql-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">▶</span>
            </div>
            <div id="chart-sql-content" style="display: ${component.sql && component.type !== "choropleth" && component.type !== "table_advanced" ? "block" : "none"}; margin-top: 15px;">
                <small class="hint" style="display: block; margin-bottom: 10px;">
                    Write a custom SQL query for this chart. For bar/line/bar_line/pyramid: first column = labels, remaining columns = datasets.
                    For pie: each column header = slice label, first row values = slice sizes.
                </small>

                <div class="form-group">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer; margin-bottom: 10px;">
                        <input type="checkbox" id="chart-sql-enabled" ${component.sql && component.type !== "choropleth" && component.type !== "table_advanced" ? "checked" : ""} onchange="toggleChartSqlMode()">
                        <span>Use Custom SQL</span>
                    </label>
                </div>

                <div id="chart-sql-form" style="display: ${component.sql && component.type !== "choropleth" && component.type !== "table_advanced" ? "block" : "none"};">
                    <div class="form-group">
                        <label for="chart-sql">SQL Query *</label>
                        <textarea id="chart-sql" class="form-textarea" style="font-family: monospace; font-size: 12px; min-height: 150px;" placeholder="SELECT period_month, SUM(cases) as total_cases, SUM(deaths) as total_deaths
FROM report.my_table
WHERE 1=1 {{district_filter}} {{year_filter}} {{month_filter}}
GROUP BY period_month
ORDER BY period_month">${component.sql && component.type !== "choropleth" && component.type !== "table_advanced" ? component.sql || "" : ""}</textarea>
                        <small class="hint">Use <code>{{district_filter}}</code>, <code>{{region_filter}}</code>, <code>{{facility_filter}}</code>, <code>{{year_filter}}</code>, <code>{{month_filter}}</code> for filter placeholders.${customFilterHint()} Max ${500} rows returned.</small>
                    </div>

                    <div style="margin-top: 15px;">
                        <button type="button" class="btn-secondary" id="btn-preview-chart-sql" style="width: 100%; background: var(--color-success); color: white; border-color: var(--color-success);">
                            ▶ Preview SQL Results
                        </button>
                    </div>

                    <div id="chart-sql-preview-result" style="margin-top: 15px; display: none;">
                        <div style="background: var(--color-hover); padding: 10px; border-radius: 4px; font-size: 12px;">
                            <div id="chart-sql-preview-info" style="margin-bottom: 10px;"></div>
                            <div id="chart-sql-preview-table" style="max-height: 200px; overflow: auto;"></div>
                        </div>
                    </div>

                    <div id="chart-sql-dataset-mapping" style="display: ${component.type === "bar_line" ? "block" : "none"}; margin-top: 15px;">
                        <label style="font-weight: 600; margin-bottom: 8px; display: block;">Dataset Mapping (Bar vs Line)</label>
                        <small class="hint" style="display: block; margin-bottom: 10px;">
                            Map each SQL column (by alias) to bar or line. The column name must match the SQL alias exactly.
                        </small>
                        <div id="chart-sql-datasets-list">
                            ${
                              component.type === "bar_line" &&
                              component.sql &&
                              component.data?.datasets?.length > 0
                                ? component.data.datasets
                                    .map(
                                      (ds, i) => `
                                <div class="chart-sql-dataset-row" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center;">
                                    <input type="text" class="form-input chart-sql-ds-label" value="${ds.label || ""}" placeholder="Column alias" style="flex: 1; margin: 0;">
                                    <select class="form-input chart-sql-ds-type" style="width: 80px; margin: 0;">
                                        <option value="bar" ${ds.chartType !== "line" ? "selected" : ""}>Bar</option>
                                        <option value="line" ${ds.chartType === "line" ? "selected" : ""}>Line</option>
                                    </select>
                                    <input type="color" class="form-input chart-sql-ds-color" value="${ds.color || "#3498db"}" style="width: 40px; height: 32px; padding: 2px; margin: 0;">
                                    <button type="button" class="btn-small btn-remove-chart-sql-ds" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">×</button>
                                </div>
                            `,
                                    )
                                    .join("")
                                : ""
                            }
                        </div>
                        <button type="button" id="btn-add-chart-sql-dataset" class="btn-secondary" style="width: 100%; margin-top: 4px;">
                            + Add Column Mapping
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Heatmap Section (table types only) -->
        <div id="section-heatmap" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="form-group">
                <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                    <input type="checkbox" id="heatmap-enabled" ${component.heatmap?.scale || component.heatmap?.rules?.length || component.heatmap?.columns?.length ? "checked" : ""} onchange="toggleHeatmapOptions()">
                    <span>Heatmap</span>
                    <span class="help-tooltip" data-tooltip="Color numeric cells by intensity. Useful for spotting high/low values at a glance — like conditional formatting in spreadsheets.">?</span>
                </label>
                <small class="hint">Apply background coloring to numeric columns based on value</small>
            </div>
            <div id="heatmap-options" style="display: ${component.heatmap?.scale || component.heatmap?.rules?.length || component.heatmap?.columns?.length ? "block" : "none"}; margin-top: 12px;">
                <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer; margin-bottom: 12px;">
                    <input type="checkbox" id="heatmap-advanced" ${component.heatmap?.rules?.length ? "checked" : ""} onchange="toggleHeatmapAdvanced()">
                    <span>Advanced (per-column thresholds)</span>
                </label>
                <div id="heatmap-simple-group" style="display: ${component.heatmap?.rules?.length ? "none" : "block"};">
                    <label for="heatmap-scale">Default Color Scale</label>
                    <select id="heatmap-scale" class="form-input">
                        <option value="green-red" ${(component.heatmap?.scale || "green-red") === "green-red" ? "selected" : ""}>Green → Red (high = bad)</option>
                        <option value="red-green" ${component.heatmap?.scale === "red-green" ? "selected" : ""}>Red → Green (high = good)</option>
                        <option value="blue" ${component.heatmap?.scale === "blue" ? "selected" : ""}>Blue intensity (neutral)</option>
                    </select>
                    <small class="hint">Green→Red for cases/deaths, Red→Green for coverage/rates</small>
                    <div style="margin-top: 10px;">
                        <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                            <input type="checkbox" id="heatmap-use-columns" ${component.heatmap?.columns?.length ? "checked" : ""} onchange="toggleHeatmapColumns()">
                            <span>Apply to specific columns only</span>
                        </label>
                        <div id="heatmap-columns-group" style="display: ${component.heatmap?.columns?.length ? "block" : "none"}; margin-top: 8px;">
                            <small class="hint" style="display: block; margin-bottom: 8px;">Each column can have its own scale direction. Columns not listed get no heatmap.</small>
                            <datalist id="heatmap-column-suggestions"></datalist>
                            <div id="heatmap-cols-list"></div>
                            <button type="button" id="btn-add-heatmap-col" class="btn-secondary" style="width: 100%; margin-top: 4px;">+ Add Column</button>
                        </div>
                    </div>
                </div>
                <div id="heatmap-advanced-group" style="display: ${component.heatmap?.rules?.length ? "block" : "none"};">
                    <datalist id="heatmap-column-suggestions"></datalist>
                    <div id="heatmap-rules-list"></div>
                    <button class="btn-secondary" id="btn-add-heatmap-rule" style="width: 100%;">+ Add Column Rule</button>
                </div>
            </div>
        </div>

        <!-- <div id="section-header-tooltips" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleHeaderTooltipsSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;">Header Tooltips <span class="help-tooltip" data-tooltip="Add hover definitions for table column headers such as TPR % → Test Positivity Rate %. Available in Table and Advanced Table components.">?</span></h4>
                <span id="headertooltips-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">${headerTooltipEntries.length > 0 ? "â–¼" : "â–¶"}</span>
            </div>
            <div id="headertooltips-content" style="display: ${headerTooltipEntries.length > 0 ? "block" : "none"}; margin-top: 15px;">
                <small class="hint" style="display: block; margin-bottom: 10px;">Add optional explanations for abbreviated or technical headers. For Advanced Table, run SQL Preview first to get exact header suggestions.</small>

                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-bottom: 12px; padding: 10px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-hover);">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer; margin: 0;">
                        <input type="checkbox" id="table-header-wrap" ${component.tableHeaderWrap ? "checked" : ""}>
                        <span>Wrap long headers</span>
                    </label>
                    <div>
                        <label for="table-header-style" style="font-size: 12px; font-weight: 500; margin-bottom: 4px; display: block;">Header style</label>
                        <select id="table-header-style" class="form-input" style="font-size: 12px; margin: 0;">
                            <option value="default" ${!component.tableHeaderStyle || component.tableHeaderStyle === "default" ? "selected" : ""}>Default</option>
                            <option value="enhanced-compact" ${component.tableHeaderStyle === "enhanced-compact" ? "selected" : ""}>Enhanced Compact</option>
                            <option value="enhanced-readable" ${component.tableHeaderStyle === "enhanced-readable" || component.tableHeaderStyle === "enhanced" ? "selected" : ""}>Enhanced Readable</option>
                        </select>
                    </div>
                </div>

                <datalist id="header-tooltip-suggestions"></datalist>
                <div id="header-tooltips-list" style="border: 1px solid var(--color-border); border-radius: 4px; padding: 10px; margin-bottom: 10px; background: var(--color-bg);">
                    ${
                      headerTooltipEntries.length > 0
                        ? headerTooltipEntries
                            .map(([header, tooltip]) =>
                              buildHeaderTooltipRowHTML(
                                header,
                                tooltip,
                                (component.headerFormulas || {})[header] || "",
                              ),
                            )
                            .join("")
                        : '<p id="no-header-tooltips-msg" style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">No header tooltips yet. Add one to explain abbreviations like TPR % or CFR %.</p>'
                    }
                </div>
                <button class="btn-secondary" id="btn-add-header-tooltip" style="width: 100%;">+ Add Header Tooltip</button>
            </div>
        </div> -->

        <!-- Advanced Options Wrapper -->
        <div id="section-advanced-options">
            <button type="button" class="advanced-toggle-btn" id="btn-toggle-advanced">
                <span>Advanced Options</span>
                <span id="advanced-toggle-arrow">▶</span>
            </button>
            <div id="advanced-options-content" style="display: none;">

            <!-- WHERE Clause Section (Collapsible) -->
            <div id="section-where" class="form-section">
                <div class="form-group" style="margin-top: 10px;">
                    <div class="collapsible-header" onclick="toggleWhereSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between; padding: 8px; background: var(--color-hover); border-radius: 4px;">
                        <span style="font-size: 12px; font-weight: 500;">Filter (WHERE clause)</span>
                        <span id="where-toggle-icon" style="font-size: 12px; transition: transform 0.2s;">▶</span>
                    </div>
                    <div id="where-content" style="display: none; margin-top: 8px;">
                        <input type="text" id="comp-where" class="form-input" style="font-family: monospace; font-size: 12px;"
                               value="${query.where || ""}"
                               placeholder="e.g., age_group = 'under_5' AND status = 'active'"
                               list="column-suggestions">
                        <small class="hint" style="font-size: 11px;">Available: =, !=, &gt;, &lt;, AND, OR, (). No subqueries.</small>
                    </div>
                </div>
            </div>


        <!-- KPI Card Appearance Section -->
        <div id="section-kpi" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <h4 style="margin-bottom: 15px;">KPI Card</h4>
            <div class="form-group">
                <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer; margin-bottom: 6px;">
                    <input type="checkbox" id="kpi-sql-enabled" ${component.sql ? "checked" : ""} onchange="toggleKpiSqlMode()">
                    <span>Use Custom SQL <span class="help-tooltip" data-tooltip="Write a raw SQL query instead of using the Query Builder above. Use when the Query Builder cannot express the metric (e.g. complex subqueries, CTEs, window functions).">?</span></span>
                </label>
                <small class="hint">When unchecked, the Query Builder above provides the data source. The first aggregation column's value populates the card.</small>
            </div>
            <div id="kpi-sql-form" style="display: ${component.sql ? "block" : "none"};">
                <div class="form-group">
                    <label for="kpi-sql">SQL Query <span class="help-tooltip" data-tooltip="SQL returning a single value. The first numeric result becomes the displayed KPI value. Use {{year_filter}}, {{month_filter}}, {{quarter_filter}}, {{region_filter}}, {{district_filter}} for filter placeholders.">?</span></label>
                    <textarea id="kpi-sql" class="form-textarea" style="font-family: monospace; font-size: 12px; min-height: 140px;" placeholder="SELECT COALESCE(SUM(emergency_cases), 0)::bigint AS &quot;Value&quot;&#10;FROM report.my_table&#10;WHERE 1=1&#10;  {{year_filter}} {{month_filter}}&#10;  {{region_filter}} {{district_filter}}">${escapeHtml(component.sql || "")}</textarea>
                    <small class="hint">Returns one row, one column. Use a descriptive alias (AS &quot;Label&quot;) — it is not displayed but helps with debugging.</small>
                </div>
            </div>
            <div class="form-group">
                <label for="kpi-accent-color">Accent Colour <span class="help-tooltip" data-tooltip="Sets the left border and tinted background colour of the KPI card. Leave blank for the default card style.">?</span></label>
                <div style="display: flex; gap: 10px; align-items: center;">
                    <input type="color" id="kpi-accent-color" value="${component.accentColor || "#118DFF"}" style="width: 48px; height: 36px; border: 1px solid var(--color-border); border-radius: 4px; cursor: pointer; padding: 2px;">
                    <input type="text" id="kpi-accent-color-text" class="form-input" value="${escapeHtml(component.accentColor || "")}" placeholder="#118DFF — leave blank for default" style="flex: 1; font-family: monospace;">
                    <button type="button" class="btn-secondary" style="font-size: 11px; white-space: nowrap;" onclick="document.getElementById('kpi-accent-color').value='#118DFF'; document.getElementById('kpi-accent-color-text').value='';">Clear</button>
                </div>
                <small class="hint">Hex colour code, e.g. #118DFF (blue), #D64550 (red), #E66C37 (orange). Leave text field blank for no accent.</small>
            </div>
            <div class="form-group">
                <label for="kpi-unit">Unit Suffix <span class="help-tooltip" data-tooltip="Suffix appended directly after the value, e.g. % or /100k.">?</span></label>
                <input type="text" id="kpi-unit" class="form-input" value="${escapeHtml(component.unit || "")}" placeholder="e.g. % or /100k" style="max-width: 200px;">
                <small class="hint">Displayed immediately after the number with no space. Leave blank for no unit.</small>
            </div>
            <div class="form-group">
                <label for="kpi-value-template">Value Template <span class="help-tooltip" data-tooltip="Custom display pattern for the value slot. Use {{value}} for the single query result. For multi-alias queries use each alias name, e.g. {{reported}} ({{pct}}%). Leave blank for the default.">?</span></label>
                <input type="text" id="kpi-value-template" class="form-input" value="${escapeHtml(component.valueTemplate || "")}" placeholder="Leave blank to show {{value}}">
                <small class="hint">Advanced: combine multiple aliases, e.g. <code>{{reported}} ({{pct}}%)</code>. Only needed for multi-column queries.</small>
            </div>
            <div class="form-group">
                <label class="checkbox-label" style="display: flex; align-items: flex-start; gap: 8px; cursor: pointer;">
                    <input type="checkbox" id="kpi-lower-is-better" style="margin-top: 2px; flex-shrink: 0;" ${component.lowerIsBetter ? "checked" : ""}>
                    <span>Lower is Better <span class="help-tooltip" data-tooltip="Inverts comparison arrow colours — a decrease shows green (good) and an increase shows red (bad). Use for rates such as mortality, complications, or case fatality.">?</span></span>
                </label>
                <small class="hint" style="padding-left: 24px;">E.g. for mortality rate, case fatality rate, complication rate.</small>
            </div>
            <div class="form-group">
                <label for="kpi-tone">Preset Tone <span class="help-tooltip" data-tooltip="Override the accent colour with a built-in colour preset. Takes precedence over accent colour. Leave blank to use the accent colour above.">?</span></label>
                <select id="kpi-tone" class="form-input" style="max-width: 220px;">
                    <option value="" ${!component.tone ? "selected" : ""}>None (use accent colour)</option>
                    <option value="good" ${component.tone === "good" ? "selected" : ""}>Good (green stripe)</option>
                    <option value="alert" ${component.tone === "alert" ? "selected" : ""}>Alert (red stripe)</option>
                    <option value="muted" ${component.tone === "muted" ? "selected" : ""}>Muted (grey, no stripe)</option>
                </select>
                <small class="hint">Tone takes precedence over accent colour. Use for static status indicators.</small>
            </div>
        </div>

        <!-- Period Limit Section -->
        <div id="section-periodlimit" class="form-section">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="togglePeriodLimitSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;">Period Range</h4>
                <span id="periodlimit-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">▶</span>
            </div>
            <div id="periodlimit-content" style="display: none; margin-top: 15px;">
                <div class="form-group">
                    <label for="period-limit">Show Last N Periods <span class="help-tooltip" data-tooltip="Expand the period range backwards. E.g., if user selects March and periodLimit is 4, shows Dec-Mar (4 months). Works for charts and tables grouped by month/week/quarter.">?</span></label>
                    <input type="number" id="period-limit" class="form-input" value="${query.periodLimit || ""}" min="0" max="24" placeholder="4">
                    <small class="hint">Common values: 4 (quarterly view), 6 (half-year), 12 (full year). If user selects March and periodLimit is 4, shows Dec–Mar.</small>
                    <small id="period-limit-sql-hint" class="hint" style="display: none; margin-top: 6px; color: var(--color-text-secondary);">In SQL mode, your query must include <code>{{month_filter}}</code>, <code>{{week_filter}}</code>, or <code>{{quarter_filter}}</code> for period expansion to take effect.</small>
                </div>
            </div>
        </div>

        <!-- Order By Section -->
        <div id="section-orderby" class="form-section">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleOrderBySection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;">Sort Order</h4>
                <span id="orderby-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">▶</span>
            </div>
            <div id="orderby-content" style="display: none; margin-top: 15px;">
                <div class="form-group">
                    <label for="orderby-field">Sort By Field</label>
                    <input type="text" id="orderby-field" class="form-input" value="${(query.orderBy && query.orderBy[0]?.field) || ""}" placeholder="e.g., alias_name">
                </div>

                <div class="form-group">
                    <label for="orderby-direction">Sort Direction</label>
                    <select id="orderby-direction" class="form-input">
                        <option value="ASC" ${query.orderBy && query.orderBy[0]?.direction === "ASC" ? "selected" : ""}>Ascending (ASC)</option>
                        <option value="DESC" ${query.orderBy && query.orderBy[0]?.direction === "ASC" ? "" : "selected"}>Descending (DESC)</option>
                    </select>
                </div>
            </div>
        </div>

        <!-- Select Columns Section -->
        <div id="section-select-columns" class="form-section">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleSelectColumnsSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 id="selectcols-section-title" style="margin: 0;">Select Columns to Display <span class="help-tooltip" data-tooltip="Choose which columns appear in the chart. For Bar-Line charts, also select whether each column is displayed as a bar or line.">?</span></h4>
                <span id="selectcols-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">▶</span>
            </div>
            <div id="selectcols-content" style="display: none; margin-top: 15px;">
                <small id="selectcols-section-hint" class="hint" style="display: block; margin-bottom: 10px;">All columns are shown by default. Uncheck any you want to hide.</small>
                <div id="select-columns-list">
                    ${renderSelectColumnsCheckboxes(query, componentType)}
                </div>

                <!-- Transpose Subsection (for tables only) -->
                <div id="transpose-subsection" style="margin-top: 20px; padding-top: 15px; border-top: 1px dashed var(--color-border);">
                    <div class="collapsible-header" onclick="toggleTransposeSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                        <h5 style="margin: 0; font-size: 0.95em;">Transpose <span class="help-tooltip" data-tooltip="Transpose the table so that aggregation names become rows and first column values (periods) become column headers.">?</span></h5>
                        <span id="transpose-toggle-icon" style="font-size: 16px; transition: transform 0.2s;">▶</span>
                    </div>
                    <div id="transpose-content" style="display: none; margin-top: 12px;">
                        <small class="hint" style="display: block; margin-bottom: 12px;">Swap rows and columns: metric names become rows, periods become column headers.</small>

                        <div style="display: flex; align-items: center; margin-bottom: 12px;">
                            <input type="checkbox" id="transpose-enabled" ${query.transpose?.enabled ? "checked" : ""} onchange="updateTransposeOptions()">
                            <label for="transpose-enabled" style="margin-left: 8px; cursor: pointer;">Enable transpose</label>
                        </div>

                        <div id="transpose-options" style="${query.transpose?.enabled ? "" : "opacity: 0.5;"}">
                            <div class="form-group" style="margin-bottom: 10px;">
                                <label for="transpose-row-label" style="font-size: 0.85em;">Row Label</label>
                                <input type="text" id="transpose-row-label" class="form-input" style="font-size: 0.9em;" placeholder="Indicator" value="${query.transpose?.rowLabel || ""}">
                                <small class="hint">Label for the first column (default: "Indicator")</small>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div id="section-header-tooltips" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleHeaderTooltipsSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;">Custom Table Configs <span class="help-tooltip" data-tooltip="Add custom per-column configs such as header explanations, abbreviations, or other advanced options. For Advanced Table, run SQL Preview first to get exact header suggestions.">?</span></h4>
                <span id="headertooltips-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">${hasCustomTableConfigs ? "&#9660;" : "&#9654;"}</span>
            </div>
            <div id="headertooltips-content" style="display: ${hasCustomTableConfigs ? "block" : "none"}; margin-top: 15px;">
                <small class="hint" style="display: block; margin-bottom: 10px;">Add optional per-column explanations, abbreviations, or advanced configs. For Advanced Table, run SQL Preview first to get exact header suggestions.</small>

                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 10px; margin-bottom: 12px; padding: 10px; border: 1px solid var(--color-border); border-radius: 4px; background: var(--color-hover);">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer; margin: 0;">
                        <input type="checkbox" id="table-header-wrap" ${component.tableHeaderWrap ? "checked" : ""}>
                        <span>Wrap long headers</span>
                    </label>
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer; margin: 0;">
                        <input type="checkbox" id="table-vertical-lines" ${component.tableCellBoundaries || component.tableVerticalLines ? "checked" : ""}>
                        <span>Enable cell boundaries</span>
                    </label>
                    <div>
                        <label for="table-header-style" style="font-size: 12px; font-weight: 500; margin-bottom: 4px; display: block;">Header style</label>
                        <select id="table-header-style" class="form-input" style="font-size: 12px; margin: 0;">
                            <option value="default" ${!component.tableHeaderStyle || component.tableHeaderStyle === "default" ? "selected" : ""}>Default</option>
                            <option value="enhanced" ${component.tableHeaderStyle === "enhanced" ? "selected" : ""}>Enhanced</option>
                        </select>
                    </div>
                    <div>
                        <label for="table-page-size" style="font-size: 12px; font-weight: 500; margin-bottom: 4px; display: block;">Rows per page</label>
                        <input type="number" id="table-page-size" class="form-input" min="1" step="1" value="${tablePageSizeValue}" placeholder="20" style="font-size: 12px; margin: 0;">
                    </div>
                </div>

                <datalist id="header-tooltip-suggestions"></datalist>
                <div id="header-tooltips-list" style="border: 1px solid var(--color-border); border-radius: 4px; padding: 10px; margin-bottom: 10px; background: var(--color-bg);">
                    ${
                      headerTooltipEntries.length > 0
                        ? headerTooltipEntries
                            .map(([header, tooltip]) =>
                              buildHeaderTooltipRowHTML(
                                header,
                                tooltip,
                                (component.headerFormulas || {})[header] || "",
                              ),
                            )
                            .join("")
                        : '<p id="no-header-tooltips-msg" style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">No custom table configs yet. Add one to explain abbreviations like TPR % or CFR %.</p>'
                    }
                </div>
                <button class="btn-secondary" id="btn-add-header-tooltip" style="width: 100%;">+ Add Header Tooltip</button>

                <div style="margin-top: 14px; border-top: 1px dashed var(--color-border); padding-top: 12px;">
                    <h5 style="margin: 0 0 8px; font-size: 13px;">Split Column Groups</h5>
                    <small class="hint" style="display: block; margin-bottom: 10px;">Create grouped headers by defining a parent and the child headers under it (comma-separated).</small>
                    <datalist id="split-column-suggestions"></datalist>
                    <div id="split-columns-list" style="border: 1px solid var(--color-border); border-radius: 4px; padding: 10px; margin-bottom: 10px; background: var(--color-bg);">
                        ${
                          splitColumnEntries.length > 0
                            ? splitColumnEntries
                                .map((entry) =>
                                  buildSplitColumnRowHTML(
                                    entry.parent || "",
                                    entry.children || [],
                                  ),
                                )
                                .join("")
                            : '<p id="no-split-columns-msg" style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">No split column groups yet. Add one to create parent/child headers.</p>'
                        }
                    </div>
                    <button class="btn-secondary" id="btn-add-split-column" style="width: 100%;">+ Add Split Column Group</button>
                </div>
            </div>
        </div>

        <!-- Bar Options Section -->
        <div id="section-baroptions" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleBarOptionsSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;"><span id="baroptions-title">Bar Options</span> <span class="help-tooltip" data-tooltip="Configure bar chart display: stacked bars, horizontal orientation, or show value labels on each bar.">?</span></h4>
                <span id="baroptions-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">▶</span>
            </div>
            <div id="baroptions-content" style="display: none; margin-top: 15px;">
                <div class="form-group" id="bar-opt-stacked-group">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                        <input type="checkbox" id="bar-stacked" ${component.stacked ? "checked" : ""}>
                        <span>Stacked</span>
                    </label>
                    <small class="hint">Stack datasets on top of each other instead of side-by-side</small>
                </div>
                <div class="form-group" id="bar-opt-horizontal-group" style="margin-top: 12px;">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                        <input type="checkbox" id="bar-horizontal" ${component.horizontal ? "checked" : ""}>
                        <span>Horizontal</span>
                    </label>
                    <small class="hint">Display bars horizontally (categories on Y-axis)</small>
                </div>
                <div class="form-group" id="bar-opt-showvalues-group" style="margin-top: 12px;">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                        <input type="checkbox" id="bar-show-values" ${(component.showValuesWithAxis ? false : component.showValues !== false) ? "checked" : ""}>
                        <span>Show Values</span>
                    </label>
                    <small class="hint">Display value labels on each bar and disable the Y-axis</small>
                </div>
                <div class="form-group" style="margin-top: 12px;">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                        <input type="checkbox" id="bar-show-values-with-axis" ${component.showValuesWithAxis ? "checked" : ""}>
                        <span>Show Values (y-axis enabled)</span>
                    </label>
                    <small class="hint">Display value labels on each bar with Y-axis visible</small>
                </div>
                <div class="form-group" style="margin-top: 12px;">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                        <input type="checkbox" id="bar-show-axis-titles" ${component.showAxisTitles !== false && !component.hideAxes ? "checked" : ""}>
                        <span>Show Axis Titles</span>
                    </label>
                    <small class="hint">Uncheck to hide axis titles on bar and bar-line charts</small>
                </div>
                <div class="form-group" id="bar-opt-wrap-group" style="margin-top: 12px;">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                        <input type="checkbox" id="bar-wrap-labels" ${component.wrapLabels ? "checked" : ""}>
                        <span>Wrap Long Labels</span>
                    </label>
                    <small class="hint">Wrap long category-axis labels onto multiple lines (splits on – separator)</small>
                </div>
            </div>
        </div>

        <!-- Axis Labels Section (moved up, immediately after Show Axis Titles) -->
        <div id="section-axislabels" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <h4>Axis Labels</h4>
            <div class="form-group">
                <label for="axislabels-x">X-Axis Label</label>
                <input type="text" id="axislabels-x" class="form-input" value="${component.axisLabels?.x || ""}" placeholder="e.g. District">
            </div>
            <div class="form-group" style="margin-top: 8px;">
                <label for="axis-tooltip-x">X-Axis Tooltip</label>
                <input type="text" id="axis-tooltip-x" class="form-input" value="${(component.axisTooltips?.x || "").replace(/"/g, "&quot;")}" placeholder="Optional description shown on hover">
            </div>
            <div class="form-group" style="margin-top: 8px;">
                <label for="axis-formula-x">X-Axis Formula</label>
                <input type="text" id="axis-formula-x" class="form-input" value="${(component.axisFormulas?.x || "").replace(/"/g, "&quot;")}" placeholder="Optional formula shown on hover">
            </div>
            <div class="form-group" style="margin-top: 12px;">
                <label for="axislabels-y">Left Y-Axis Label</label>
                <input type="text" id="axislabels-y" class="form-input" value="${component.axisLabels?.y || ""}" placeholder="e.g. Number of Cases">
            </div>
            <div class="form-group" style="margin-top: 8px;">
                <label for="axis-tooltip-y">Y-Axis Tooltip</label>
                <input type="text" id="axis-tooltip-y" class="form-input" value="${(component.axisTooltips?.y || "").replace(/"/g, "&quot;")}" placeholder="Optional description shown on hover">
            </div>
            <div class="form-group" style="margin-top: 8px;">
                <label for="axis-formula-y">Y-Axis Formula</label>
                <input type="text" id="axis-formula-y" class="form-input" value="${(component.axisFormulas?.y || "").replace(/"/g, "&quot;")}" placeholder="Optional formula shown on hover">
            </div>
            <div class="form-group" style="margin-top: 12px;">
                <label for="axislabels-y1">Right Y-Axis Label</label>
                <input type="text" id="axislabels-y1" class="form-input" value="${component.axisLabels?.y1 || ""}" placeholder="e.g. Percentage (%)">
            </div>
        </div>

        <!-- Trend Line Section (now after Axis Labels) -->
        <div id="section-trendline" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="form-group">
                <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                    <input type="checkbox" id="trend-line" ${component.trendLine ? "checked" : ""}>
                    <span>Trend Line</span>
                    <span class="help-tooltip" data-tooltip="Show a dashed linear regression line over each dataset to indicate the overall trend direction.">?</span>
                </label>
                <small class="hint">Display a best-fit line showing the overall data trend</small>
            </div>
            <div class="form-group" style="margin-top: 12px;">
                <label style="display: flex; align-items: center; gap: 8px;">
                    <span>Color</span>
                    <input type="color" id="trend-line-color" value="${component.trendLineColor || "#888888"}" style="width: 36px; height: 28px; padding: 0; border: 1px solid var(--color-border); border-radius: 4px; cursor: pointer;">
                </label>
                <small class="hint">Color for the trend line (leave default for dataset colors)</small>
            </div>
        </div>

        <!-- Y-Axis Limit Section -->
        <div id="section-ylimit" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <h4>Y-Axis Limit <span class="help-tooltip" data-tooltip="Fix the value-axis range so the chart looks consistent regardless of the data. Values above the max are clipped at the ceiling.">?</span></h4>
            <small class="hint">Leave blank for auto-fit. Set max for percentage charts (e.g. 100) or to keep multiple charts visually comparable.</small>
            <div class="form-group" style="margin-top: 12px; display: flex; gap: 12px;">
                <div style="flex: 1;">
                    <label for="ylimit-min">Min</label>
                    <input type="number" step="any" id="ylimit-min" class="form-input" value="${component.yLimit?.min ?? ""}" placeholder="auto">
                </div>
                <div style="flex: 1;">
                    <label for="ylimit-max">Max</label>
                    <input type="number" step="any" id="ylimit-max" class="form-input" value="${component.yLimit?.max ?? ""}" placeholder="auto">
                </div>
            </div>
            <div class="form-group" style="margin-top: 16px;">
                <label style="font-weight: 500;">Right Y-Axis (bar+line only)</label>
                <small class="hint">Only used for bar+line charts. Bar and line charts ignore this.</small>
            </div>
            <div class="form-group" style="margin-top: 8px; display: flex; gap: 12px;">
                <div style="flex: 1;">
                    <label for="ylimit-y1-min">Min</label>
                    <input type="number" step="any" id="ylimit-y1-min" class="form-input" value="${component.yLimit?.y1?.min ?? ""}" placeholder="auto">
                </div>
                <div style="flex: 1;">
                    <label for="ylimit-y1-max">Max</label>
                    <input type="number" step="any" id="ylimit-y1-max" class="form-input" value="${component.yLimit?.y1?.max ?? ""}" placeholder="auto">
                </div>
            </div>
        </div>

        <!-- Legend Section -->
        <div id="section-legend" class="form-section">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleLegendSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;">Legend <span class="help-tooltip" data-tooltip="Configure the chart legend visibility and position.">?</span></h4>
                <span id="legend-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">▶</span>
            </div>
            <div id="legend-content" style="display: none; margin-top: 15px;">
                <div class="form-group">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px; cursor: pointer;">
                        <input type="checkbox" id="legend-show" ${component.data?.legend?.show !== false ? "checked" : ""}>
                        <span>Show Legend</span>
                    </label>
                    <small class="hint">Display the legend on the chart</small>
                </div>
                <div class="form-group" style="margin-top: 12px;">
                    <label for="legend-position">Position</label>
                    <select id="legend-position" class="form-input">
                        <option value="top" ${component.data?.legend?.position === "top" || !component.data?.legend?.position ? "selected" : ""}>Top</option>
                        <option value="bottom" ${component.data?.legend?.position === "bottom" ? "selected" : ""}>Bottom</option>
                        <option value="left" ${component.data?.legend?.position === "left" ? "selected" : ""}>Left</option>
                        <option value="right" ${component.data?.legend?.position === "right" ? "selected" : ""}>Right</option>
                    </select>
                    <small class="hint">Position of the legend on the chart</small>
                </div>
            </div>
        </div>

        <!-- GeoJSON Section -->
        <div id="section-geojson" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleGeojsonSection()" style="cursor: pointer; display: flex; justify-content: space-between; align-items: center;">
                <h4 style="margin: 0;">Map Configuration</h4>
                <span id="geojson-toggle-icon" style="font-size: 12px;">▶</span>
            </div>
            <div id="geojson-content" style="margin-top: 15px; display: none;">
                <div class="form-group">
                    <label for="map-value-type">Value Type</label>
                    <select id="map-value-type" class="form-input" onchange="onMapValueTypeChange()">
                        <option value="numeric"${component.valueType !== "categorical" ? " selected" : ""}>Numeric (binned ranges)</option>
                        <option value="categorical"${component.valueType === "categorical" ? " selected" : ""}>Categorical (text values)</option>
                    </select>
                    <small class="hint">Numeric: districts colored by value ranges. Categorical: districts colored by text category (e.g. "Global Fund", "USAID").</small>
                </div>

                <div class="form-group">
                    <label for="legend-title">Legend Title</label>
                    <input type="text" id="legend-title" class="form-input" value="${component.data?.legend_title || "Value Range"}" placeholder="Value Range">
                    <small class="hint">Title displayed above the legend</small>
                </div>

                <div class="form-group">
                    <div class="collapsible-header" onclick="toggleBinConfigSection()" style="cursor: pointer; display: flex; justify-content: space-between; align-items: center;">
                        <label style="cursor: pointer; margin: 0;" id="bin-config-label">${component.valueType === "categorical" ? "Category Configuration" : "Bin Configuration"}</label>
                        <span id="bin-toggle-icon" style="font-size: 12px;">▶</span>
                    </div>
                    <small class="hint" id="bin-config-hint">${component.valueType === "categorical" ? "Define categories and their colors. Category names must exactly match the values returned by your query." : "Default: auto-calculated from data. Expand to customize colors, labels, and thresholds."}</small>

                    <div id="bin-config-content" style="display: none; margin-top: 12px;">
                        <div id="bin-rows-container">${buildBinRowsHTML(component)}</div>
                        <button type="button" id="add-bin-btn" onclick="addBinRow()" style="margin-top: 8px; padding: 4px 12px; font-size: 0.85em; cursor: pointer;"${(component.data?.color_scheme?.length || 3) >= 5 ? " disabled" : ""}>+ Add Category</button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Facet Section -->
        <div id="section-facet" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleFacetSection()" style="cursor: pointer; display: flex; justify-content: space-between; align-items: center;">
                <h4 style="margin: 0;">Faceted View (Multi-Period Maps) <span class="help-tooltip" data-tooltip="Display multiple maps in a grid, each showing data for a different time period. Great for comparing trends across months or quarters.">?</span></h4>
                <span id="facet-toggle-icon" style="font-size: 12px;">▶</span>
            </div>
            <div id="facet-content" style="margin-top: 15px; display: none;">
                <div class="form-group">
                    <label class="checkbox-label" style="display: flex; align-items: center; gap: 8px;">
                        <input type="checkbox" id="facet-enabled" ${component.facet ? "checked" : ""}>
                        <span>Enable Faceted View</span>
                    </label>
                    <small class="hint">Show 4 maps in a row, one for each time period</small>
                </div>

                <div id="facet-options" style="display: ${component.facet ? "block" : "none"};">
                    <div class="form-group">
                        <label for="facet-period-type">Period Type</label>
                        <select id="facet-period-type" class="form-input">
                            <option value="month" ${component.facet?.periodType === "month" ? "selected" : ""}>Month</option>
                            <option value="week" ${component.facet?.periodType === "week" ? "selected" : ""}>Week</option>
                            <option value="quarter" ${component.facet?.periodType === "quarter" ? "selected" : ""}>Quarter</option>
                            <option value="year" ${component.facet?.periodType === "year" ? "selected" : ""}>Year</option>
                        </select>
                        <small class="hint">Time period to facet by</small>
                        <div id="facet-quarter-hint" style="display: ${component.facet?.periodType === "quarter" ? "block" : "none"}; margin-top: 8px; padding: 8px 10px; background: var(--color-warning-subtle, #fffbeb); border: 1px solid var(--color-warning-border, #fcd34d); border-radius: 4px; font-size: 11px; line-height: 1.5;">
                            Quarter facets need <strong>Year</strong> + <strong>Quarter</strong> checked in the Report Filters panel. If your data uses a custom column name for quarter, set it in the <strong>Time Columns</strong> section of the report settings.
                        </div>
                    </div>

                    <div class="form-group">
                        <label for="facet-count">Number of Facets</label>
                        <select id="facet-count" class="form-input">
                            <option value="2" ${component.facet?.count === 2 ? "selected" : ""}>2 maps</option>
                            <option value="3" ${component.facet?.count === 3 ? "selected" : ""}>3 maps</option>
                            <option value="4" ${component.facet?.count === 4 || !component.facet?.count ? "selected" : ""}>4 maps (recommended)</option>
                        </select>
                        <small class="hint">Number of time periods to display (max 4)</small>
                    </div>
                </div>
            </div>
        </div>

        <!-- Reference Lines Section (for bar/line/bar_line charts) -->
        <div id="section-referencelines" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <div class="collapsible-header" onclick="toggleReferenceLinesSection()" style="cursor: pointer; display: flex; align-items: center; justify-content: space-between;">
                <h4 style="margin: 0;">Reference Lines <span class="help-tooltip" data-tooltip="Add horizontal reference lines to the chart (e.g., target lines, thresholds, baselines). Lines are drawn at a fixed Y-axis value.">?</span></h4>
                <span id="referencelines-toggle-icon" style="font-size: 18px; transition: transform 0.2s;">▶</span>
            </div>
            <div id="referencelines-content" style="display: none; margin-top: 15px;">
                <small class="hint" style="display: block; margin-bottom: 10px;">Add horizontal lines at specific values (e.g., target = 15000, baseline = 10000)</small>
                <div id="referencelines-list" style="border: 1px solid var(--color-border); border-radius: 4px; padding: 10px; max-height: 300px; overflow-y: auto; margin-bottom: 10px; background: var(--color-bg);">
                    ${
                      component.referenceLines &&
                      component.referenceLines.length > 0
                        ? component.referenceLines
                            .map(
                              (line, i) => `
                            <div class="refline-row" style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center; flex-wrap: wrap;">
                                <input type="number" class="form-input refline-value" value="${line.value || ""}" placeholder="Value" step="any" style="flex: 0 0 80px; margin: 0;">
                                <input type="text" class="form-input refline-label" value="${line.label || ""}" placeholder="Label (e.g., Target)" style="flex: 1; min-width: 100px; margin: 0;">
                                <input type="color" class="form-input refline-color" value="${line.color || "#22c55e"}" style="width: 40px; height: 32px; padding: 2px; margin: 0;">
                                <select class="form-input refline-style" style="flex: 0 0 80px; margin: 0;">
                                    <option value="dashed" ${line.style === "dashed" ? "selected" : ""}>Dashed</option>
                                    <option value="dotted" ${line.style === "dotted" ? "selected" : ""}>Dotted</option>
                                    <option value="solid" ${line.style === "solid" ? "selected" : ""}>Solid</option>
                                </select>
                                <select class="form-input refline-axis" title="Y-axis" style="flex: 0 0 60px; margin: 0;">
                                    <option value="y" ${!line.axis || line.axis === "y" ? "selected" : ""}>Left (y)</option>
                                    <option value="y1" ${line.axis === "y1" ? "selected" : ""}>Right (y1)</option>
                                </select>
                                <button class="btn-small btn-remove-refline" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">×</button>
                            </div>
                          `,
                            )
                            .join("")
                        : '<p id="no-reflines-msg" style="color: var(--color-text-secondary); font-size: 12px; margin: 0;">No reference lines yet. Click "Add Reference Line" to start.</p>'
                    }
                </div>
                <button class="btn-secondary" id="btn-add-refline" style="width: 100%;">+ Add Reference Line</button>
            </div>
        </div>

            </div><!-- end #advanced-options-content -->
        </div><!-- end #section-advanced-options -->

        <!-- HTML Template Section (shown after query settings so placeholders are known) -->
        <div id="section-content" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <h4 style="margin-bottom: 15px;">HTML Template</h4>
            <div class="form-group">
                <label for="comp-content">HTML Content</label>
                <textarea id="comp-content" class="form-textarea" rows="6" placeholder="<p>This week <strong>{{total_vhts}}</strong> FieldWorkers submitted <strong>{{total_forms}}</strong> forms.</p>">${component.content || ""}</textarea>
                <small class="hint">Use <code>{{column_name}}</code> placeholders — column aliases from your query above. Supports HTML tags.</small>
            </div>
        </div>

        <!-- Info Box Section (placed after query builder so placeholders are known) -->
        <div id="section-infobox" class="form-section" style="display: none;">
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <h4 style="margin-bottom: 15px;">Info Box Content</h4>
            <div class="form-group">
                <label for="infobox-body">Body Text</label>
                <textarea id="infobox-body" class="form-textarea" rows="4" placeholder="Enter the body text for this info box...">${escapeHtml(component.infoboxBody || "")}</textarea>
                <small class="hint">Plain text or simple HTML displayed below the title. Use <code>{{column_name}}</code> placeholders — populated from the Data Source above.</small>
            </div>
            <div class="form-group">
                <label for="infobox-accent-color">Accent Colour <span class="help-tooltip" data-tooltip="Sets the left border and title colour of the info box.">?</span></label>
                <div style="display: flex; gap: 10px; align-items: center;">
                    <input type="color" id="infobox-accent-color" value="${component.accentColor || "#2563a8"}" style="width: 48px; height: 36px; border: 1px solid var(--color-border); border-radius: 4px; cursor: pointer; padding: 2px;">
                    <input type="text" id="infobox-accent-color-text" class="form-input" value="${component.accentColor || "#2563a8"}" placeholder="#2563a8" style="flex: 1; font-family: monospace;">
                    <button type="button" class="btn-secondary" style="font-size: 11px; white-space: nowrap;" onclick="document.getElementById('infobox-accent-color').value='#2563a8'; document.getElementById('infobox-accent-color-text').value='#2563a8';">Reset</button>
                </div>
                <small class="hint">Hex colour code, e.g. #2563a8 (blue), #0e7c42 (green), #c0392b (red)</small>
            </div>
            <hr style="margin: 20px 0; border: none; border-top: 1px solid var(--color-border);">
            <h4 style="margin-bottom: 8px;">Threshold Rules <span class="help-tooltip" data-tooltip="Dynamically override the Accent Colour based on a data value. Rules are evaluated top-to-bottom; the first matching rule sets the colour.">?</span></h4>
            <div class="form-group">
                <label for="infobox-threshold-column">Threshold Column</label>
                <input type="text" id="infobox-threshold-column" class="form-input" value="${escapeHtml(component.thresholdColumn || "")}" placeholder="Leave blank to use the first column">
                <small class="hint">Name of the SQL result column whose value is tested. Leave blank to use the first returned column.</small>
            </div>
            <div id="threshold-rules-list">
                ${(component.thresholds || []).map((r) => renderThresholdRuleRow(r)).join("")}
            </div>
            <button type="button" id="add-threshold-rule" class="btn-secondary" style="margin-top: 4px; font-size: 12px;">+ Add Rule</button>
            <small class="hint" style="display: block; margin-top: 6px;">Operators: &gt;= &gt; &lt;= &lt; == !=&nbsp;&nbsp;Example: <code>&gt;= 1</code> → red, <code>== 0</code> → green</small>
        </div>

        <div id="type-specific-fields"></div>
    `;
}

/**
 * Apply smart defaults for new components only.
 * Skips if editing an existing component or if rows already exist.
 */
function applySmartDefaults(type) {
  // Only for new components — skip if editing existing
  if (currentEditingComponent?.compIndex !== null) return;

  const aggsList = document.getElementById("aggregations-list");
  const groupByList = document.getElementById("groupby-list");
  const existingAggs = aggsList
    ? aggsList.querySelectorAll(":scope > div").length
    : 0;
  const existingGroupBy = groupByList
    ? groupByList.querySelectorAll(".groupby-row").length
    : 0;

  // Skip if user already has rows (e.g., switched type after adding rows)
  if (existingAggs > 0 || existingGroupBy > 0) return;

  const defaults = {
    line: { aggs: 1, groupBys: 1, groupByFormat: "month", periodLimit: 6 },
    bar: { aggs: 1, groupBys: 1 },
    table: { aggs: 1, groupBys: 1 },
    pyramid: { aggs: 2, groupBys: 1 },
    choropleth: { aggs: 1, groupBys: 1 },
  };

  const def = defaults[type];
  if (!def) return;

  for (let i = 0; i < (def.aggs || 0); i++) addAggregation();
  for (let i = 0; i < (def.groupBys || 0); i++) addGroupBy();

  if (def.groupByFormat) {
    const formatSelect = groupByList?.querySelector(".groupby-format");
    if (formatSelect) formatSelect.value = def.groupByFormat;
  }

  if (def.periodLimit) {
    const periodInput = document.getElementById("period-limit");
    if (periodInput && !periodInput.value) {
      periodInput.value = def.periodLimit;
    }
  }
}

/**
 * Auto-expand the advanced options wrapper if the component being edited
 * has any advanced field values set.
 */
function autoExpandAdvancedIfNeeded() {
  const comp = currentEditingComponent?.component;
  if (!comp) return;

  const query = comp.query || {};
  const hasAdvanced =
    query.where ||
    (query.calculate && query.calculate.length > 0) ||
    (query.orderBy && query.orderBy.length > 0) ||
    query.periodLimit ||
    (query.selectColumns && query.selectColumns.length > 0) ||
    query.transpose?.enabled ||
    comp.stacked ||
    comp.horizontal ||
    comp.showValues ||
    comp.showValuesWithAxis ||
    comp.trendLine ||
    comp.axisLabels?.x ||
    comp.axisLabels?.y ||
    comp.axisLabels?.y1 ||
    comp.yLimit?.min !== undefined ||
    comp.yLimit?.max !== undefined ||
    comp.yLimit?.y1?.min !== undefined ||
    comp.yLimit?.y1?.max !== undefined ||
    comp.data?.legend?.show === false ||
    (comp.data?.legend?.position && comp.data.legend.position !== "top") ||
    (comp.referenceLines && comp.referenceLines.length > 0) ||
    (Array.isArray(comp.splitColumns) && comp.splitColumns.length > 0) ||
    normalizeTablePageSize(comp.pageSize) !== 20 ||
    comp.facet;

  if (hasAdvanced) {
    const advancedContent = document.getElementById("advanced-options-content");
    const advancedArrow = document.getElementById("advanced-toggle-arrow");
    if (advancedContent && advancedArrow) {
      advancedContent.style.display = "block";
      advancedArrow.textContent = "▼";
    }
  }
}

/**
 * Contextual hints — dismissible, localStorage-persisted
 */
function getDismissedHints() {
  try {
    return JSON.parse(localStorage.getItem("builder_dismissed_hints") || "[]");
  } catch {
    return [];
  }
}

function dismissHint(hintId) {
  const dismissed = getDismissedHints();
  if (!dismissed.includes(hintId)) {
    dismissed.push(hintId);
    localStorage.setItem("builder_dismissed_hints", JSON.stringify(dismissed));
  }
  const el = document.getElementById(`hint-${hintId}`);
  if (el) el.remove();
}

function showHintIfNotDismissed(hintId, container, message) {
  if (!container) return;
  const dismissed = getDismissedHints();
  if (dismissed.includes(hintId)) return;
  // Don't show duplicate
  if (document.getElementById(`hint-${hintId}`)) return;

  const div = document.createElement("div");
  div.id = `hint-${hintId}`;
  div.className = "builder-hint";
  div.innerHTML = `<span>${message}</span><button type="button" class="builder-hint-dismiss" data-hint-id="${hintId}">&times;</button>`;
  div
    .querySelector(".builder-hint-dismiss")
    .addEventListener("click", () => dismissHint(hintId));
  container.prepend(div);
}
window.showHintIfNotDismissed = showHintIfNotDismissed;

/**
 * Run inline validation — yellow warnings only, never blocks save.
 */
function runInlineValidation() {
  const type = document.getElementById("comp-type")?.value || "bar";

  // 1. Yellow border on empty aggregation aliases
  document
    .querySelectorAll("#aggregations-list .agg-alias")
    .forEach((input) => {
      const column = input
        .closest("div")
        ?.querySelector(".agg-column")
        ?.value?.trim();
      if (column && !input.value?.trim()) {
        input.classList.add("validation-warning");
      } else {
        input.classList.remove("validation-warning");
      }
    });

  // 2. Yellow banner for chart type with no groupBy
  const chartTypes = ["bar", "line", "bar_line"];
  const groupByRows = document.querySelectorAll("#groupby-list .groupby-row");
  const existingBanner = document.getElementById("validation-no-groupby");
  if (chartTypes.includes(type) && groupByRows.length === 0) {
    if (!existingBanner) {
      const groupBySection = document.getElementById("section-groupby");
      if (groupBySection) {
        const banner = document.createElement("div");
        banner.id = "validation-no-groupby";
        banner.className = "validation-banner-warning";
        banner.textContent =
          "Charts usually need at least one Group By field (e.g., period or district).";
        groupBySection.querySelector(".form-group")?.prepend(banner);
      }
    }
  } else if (existingBanner) {
    existingBanner.remove();
  }

  // 3. Red banner for pyramid != 2 aggregations
  const aggCount = document.querySelectorAll("#aggregations-list > div").length;
  const existingPyramidBanner = document.getElementById(
    "validation-pyramid-agg",
  );
  if (type === "pyramid" && aggCount > 0 && aggCount !== 2) {
    if (!existingPyramidBanner) {
      const aggSection = document.getElementById("aggregations-list");
      if (aggSection) {
        const banner = document.createElement("div");
        banner.id = "validation-pyramid-agg";
        banner.className = "validation-banner-error";
        banner.textContent =
          "Pyramid charts require exactly 2 aggregations (left and right sides).";
        aggSection.parentElement.insertBefore(banner, aggSection);
      }
    }
  } else if (existingPyramidBanner) {
    existingPyramidBanner.remove();
  }
}
window.runInlineValidation = runInlineValidation;

// --- Dynamic Bin Configuration ---

const BIN_COLOR_PALETTES = {
  3: ["#90EE90", "#FFD700", "#FF6347"],
  4: ["#90EE90", "#FFD700", "#FF8C00", "#FF6347"],
  5: ["#90EE90", "#ADFF2F", "#FFD700", "#FF8C00", "#FF6347"],
};
const BIN_DEFAULT_LABELS = ["Very Low", "Low", "Medium", "High", "Very High"];

const CATEGORICAL_COLOR_PALETTE = [
  "#e41a1c",
  "#377eb8",
  "#4daf4a",
  "#984ea3",
  "#ff7f00",
  "#a65628",
  "#f781bf",
  "#999999",
  "#66c2a5",
  "#fc8d62",
];

function isCategoricalMode() {
  const sel = document.getElementById("map-value-type");
  return sel && sel.value === "categorical";
}

function onMapValueTypeChange() {
  const categorical = isCategoricalMode();
  const label = document.getElementById("bin-config-label");
  const hint = document.getElementById("bin-config-hint");
  const addBtn = document.getElementById("add-bin-btn");

  if (label)
    label.textContent = categorical
      ? "Category Configuration"
      : "Bin Configuration";
  if (hint)
    hint.textContent = categorical
      ? "Define categories and their colors. Category names must exactly match the values returned by your query."
      : "Default: auto-calculated from data. Expand to customize colors, labels, and thresholds.";
  if (addBtn) {
    addBtn.textContent = categorical ? "+ Add Category" : "+ Add Bin";
    addBtn.disabled = false; // categorical has no 5-bin cap initially; re-check below
    if (!categorical) {
      const container = document.getElementById("bin-rows-container");
      if (container && container.querySelectorAll(".bin-row").length >= 5)
        addBtn.disabled = true;
    }
  }

  // Rebuild bin rows for the new mode
  rebuildBinRows();
}
window.onMapValueTypeChange = onMapValueTypeChange;

function buildBinRowsHTML(component) {
  const categorical = component.valueType === "categorical";
  const numBins = component.data?.color_scheme?.length || 3;
  const colors =
    component.data?.color_scheme ||
    (categorical
      ? CATEGORICAL_COLOR_PALETTE.slice(0, 3)
      : BIN_COLOR_PALETTES[3]);
  const labels =
    component.data?.bin_labels ||
    (categorical
      ? []
      : BIN_DEFAULT_LABELS.slice(BIN_DEFAULT_LABELS.length - numBins));
  const edges = component.data?.bin_edges || [];
  let html = "";
  for (let i = 0; i < numBins; i++) {
    const isLast = i === numBins - 1;
    const minRows = categorical ? 2 : 3;
    const removeBtn =
      i >= minRows
        ? `<button type="button" onclick="removeBinRow(${i})" style="padding: 2px 8px; font-size: 0.8em; cursor: pointer;">✕</button>`
        : "";

    if (categorical) {
      html += `<div class="bin-row" style="display: flex; gap: 8px; align-items: center; margin-bottom: 8px;">
                <input type="color" id="color-bin-${i}" class="form-input" value="${colors[i] || CATEGORICAL_COLOR_PALETTE[i] || "#cccccc"}" style="width: 40px; height: 32px; padding: 2px;">
                <input type="text" id="label-bin-${i}" class="form-input" value="${labels[i] || ""}" placeholder="Category name (exact match)" style="flex: 1;">
                ${removeBtn}
            </div>`;
    } else {
      html += `<div class="bin-row" style="display: flex; gap: 8px; align-items: center; margin-bottom: 8px;">
                <input type="color" id="color-bin-${i}" class="form-input" value="${colors[i] || BIN_COLOR_PALETTES[5][i] || "#cccccc"}" style="width: 40px; height: 32px; padding: 2px;">
                <input type="text" id="label-bin-${i}" class="form-input" value="${labels[i] || ""}" placeholder="${BIN_DEFAULT_LABELS[BIN_DEFAULT_LABELS.length - numBins + i]}" style="width: 80px;">
                ${
                  isLast
                    ? '<span style="color: #666; width: 100px;">&gt; threshold above</span>'
                    : `<span style="color: #666;">≤</span><input type="number" id="threshold-bin-${i}" class="form-input" value="${edges[i] != null ? edges[i] : ""}" placeholder="auto" style="width: 80px;">`
                }
                ${removeBtn}
            </div>`;
    }
  }
  return html;
}

/**
 * Rebuild bin rows from current form state when switching between numeric/categorical mode.
 */
function rebuildBinRows() {
  const container = document.getElementById("bin-rows-container");
  if (!container) return;
  const categorical = isCategoricalMode();

  // Read existing colors and labels to preserve them
  const rows = container.querySelectorAll(".bin-row");
  const count = rows.length || 3;
  const existingColors = [];
  const existingLabels = [];
  for (let i = 0; i < count; i++) {
    existingColors.push(document.getElementById(`color-bin-${i}`)?.value || "");
    existingLabels.push(document.getElementById(`label-bin-${i}`)?.value || "");
  }

  // Build a pseudo-component to reuse buildBinRowsHTML
  const pseudo = {
    valueType: categorical ? "categorical" : undefined,
    data: {
      color_scheme: existingColors,
      bin_labels: existingLabels,
    },
  };
  container.innerHTML = buildBinRowsHTML(pseudo);

  // Update add button cap
  const addBtn = document.getElementById("add-bin-btn");
  if (addBtn) {
    addBtn.textContent = categorical ? "+ Add Category" : "+ Add Bin";
    if (!categorical) {
      addBtn.disabled = count >= 5;
    } else {
      addBtn.disabled = count >= 10;
    }
  }
}

function addBinRow() {
  const container = document.getElementById("bin-rows-container");
  if (!container) return;
  const categorical = isCategoricalMode();
  const currentRows = container.querySelectorAll(".bin-row");
  const count = currentRows.length;
  const maxRows = categorical ? 10 : 5;
  if (count >= maxRows) return;

  if (categorical) {
    // Categorical: just add a new color + label row
    const newIdx = count;
    const color = CATEGORICAL_COLOR_PALETTE[newIdx] || "#cccccc";
    const minRows = 2;
    const div = document.createElement("div");
    div.className = "bin-row";
    div.style.cssText =
      "display: flex; gap: 8px; align-items: center; margin-bottom: 8px;";
    div.innerHTML = `
            <input type="color" id="color-bin-${newIdx}" class="form-input" value="${color}" style="width: 40px; height: 32px; padding: 2px;">
            <input type="text" id="label-bin-${newIdx}" class="form-input" value="" placeholder="Category name (exact match)" style="flex: 1;">
            ${newIdx >= minRows ? `<button type="button" onclick="removeBinRow(${newIdx})" style="padding: 2px 8px; font-size: 0.8em; cursor: pointer;">✕</button>` : ""}
        `;
    container.appendChild(div);
  } else {
    // Numeric: existing logic — convert last row's catch-all to threshold, add new catch-all row
    const lastRow = currentRows[count - 1];
    const lastCatchAll = lastRow.querySelector('span[style*="width: 100px"]');
    if (lastCatchAll) {
      const idx = count - 1;
      const span = document.createElement("span");
      span.style.color = "#666";
      span.textContent = "≤";
      const input = document.createElement("input");
      input.type = "number";
      input.id = `threshold-bin-${idx}`;
      input.className = "form-input";
      input.placeholder = "auto";
      input.style.width = "80px";
      lastCatchAll.replaceWith(span, input);
    }

    // Add remove button to last row if it's now row index >= 3
    if (count - 1 >= 3 && !lastRow.querySelector("button")) {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.onclick = () => removeBinRow(count - 1);
      btn.style.cssText =
        "padding: 2px 8px; font-size: 0.8em; cursor: pointer;";
      btn.textContent = "✕";
      lastRow.appendChild(btn);
    }

    // Create new last row (catch-all, no threshold)
    const newIdx = count;
    const palette = BIN_COLOR_PALETTES[count + 1] || BIN_COLOR_PALETTES[5];
    const defaultLabel =
      BIN_DEFAULT_LABELS[BIN_DEFAULT_LABELS.length - (count + 1) + newIdx] ||
      "";
    const div = document.createElement("div");
    div.className = "bin-row";
    div.style.cssText =
      "display: flex; gap: 8px; align-items: center; margin-bottom: 8px;";
    div.innerHTML = `
            <input type="color" id="color-bin-${newIdx}" class="form-input" value="${palette[newIdx] || "#cccccc"}" style="width: 40px; height: 32px; padding: 2px;">
            <input type="text" id="label-bin-${newIdx}" class="form-input" value="" placeholder="${defaultLabel}" style="width: 80px;">
            <span style="color: #666; width: 100px;">&gt; threshold above</span>
            <button type="button" onclick="removeBinRow(${newIdx})" style="padding: 2px 8px; font-size: 0.8em; cursor: pointer;">✕</button>
        `;
    container.appendChild(div);
  }

  // Disable "Add" if at max
  const addBtn = document.getElementById("add-bin-btn");
  if (addBtn && count + 1 >= maxRows) addBtn.disabled = true;
}
window.addBinRow = addBinRow;

function removeBinRow(index) {
  const container = document.getElementById("bin-rows-container");
  if (!container) return;
  const categorical = isCategoricalMode();
  const minRows = categorical ? 2 : 3;
  const rows = container.querySelectorAll(".bin-row");
  if (rows.length <= minRows) return;

  // Remove the requested row
  rows[index].remove();

  // Re-index all remaining rows
  const remaining = container.querySelectorAll(".bin-row");
  remaining.forEach((row, i) => {
    const colorInput = row.querySelector('input[type="color"]');
    if (colorInput) colorInput.id = `color-bin-${i}`;
    const labelInput = row.querySelector('input[type="text"]');
    if (labelInput) labelInput.id = `label-bin-${i}`;

    if (categorical) {
      // Categorical: just manage remove buttons (min 2 rows)
      const existingBtn = row.querySelector("button");
      if (i < minRows && existingBtn) {
        existingBtn.remove();
      } else if (i >= minRows && !existingBtn) {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.onclick = () => removeBinRow(i);
        btn.style.cssText =
          "padding: 2px 8px; font-size: 0.8em; cursor: pointer;";
        btn.textContent = "✕";
        row.appendChild(btn);
      } else if (existingBtn) {
        existingBtn.onclick = () => removeBinRow(i);
      }
    } else {
      // Numeric: manage thresholds and remove buttons
      const thresholdInput = row.querySelector('input[type="number"]');
      if (thresholdInput) thresholdInput.id = `threshold-bin-${i}`;

      const isLast = i === remaining.length - 1;

      // Last row: replace threshold with catch-all text
      if (isLast && thresholdInput) {
        const prevSpan = thresholdInput.previousElementSibling; // the "≤" span
        const catchAll = document.createElement("span");
        catchAll.style.cssText = "color: #666; width: 100px;";
        catchAll.innerHTML = "&gt; threshold above";
        if (prevSpan && prevSpan.textContent === "≤") prevSpan.remove();
        thresholdInput.replaceWith(catchAll);
      }

      // Remove buttons: only on rows index >= 3
      const existingBtn = row.querySelector("button");
      if (i < 3 && existingBtn) {
        existingBtn.remove();
      } else if (i >= 3 && !existingBtn) {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.onclick = () => removeBinRow(i);
        btn.style.cssText =
          "padding: 2px 8px; font-size: 0.8em; cursor: pointer;";
        btn.textContent = "✕";
        row.appendChild(btn);
      } else if (existingBtn) {
        existingBtn.onclick = () => removeBinRow(i);
      }
    }
  });

  // Re-enable "Add" button
  const addBtn = document.getElementById("add-bin-btn");
  if (addBtn) addBtn.disabled = false;
}
window.removeBinRow = removeBinRow;

/**
 * Read bin configuration from the dynamic bin rows in the form.
 * Returns { colorScheme, binLabels, binEdges } where binEdges may be null if all thresholds are empty.
 * In categorical mode, binEdges is always null and binLabels are category names.
 */
function readBinConfig() {
  const container = document.getElementById("bin-rows-container");
  const categorical = isCategoricalMode();

  if (!container) {
    return {
      colorScheme: categorical
        ? CATEGORICAL_COLOR_PALETTE.slice(0, 2)
        : ["#90EE90", "#FFD700", "#FF6347"],
      binLabels: categorical
        ? ["Category 1", "Category 2"]
        : ["Low", "Medium", "High"],
      binEdges: null,
    };
  }
  const rows = container.querySelectorAll(".bin-row");
  const numBins = rows.length;
  const colorScheme = [];
  const binLabels = [];

  if (categorical) {
    for (let i = 0; i < numBins; i++) {
      colorScheme.push(
        document.getElementById(`color-bin-${i}`)?.value ||
          CATEGORICAL_COLOR_PALETTE[i] ||
          "#cccccc",
      );
      binLabels.push(
        document.getElementById(`label-bin-${i}`)?.value?.trim() || "",
      );
    }
    return { colorScheme, binLabels, binEdges: null };
  }

  // Numeric mode
  const defaultLabels = BIN_DEFAULT_LABELS.slice(
    BIN_DEFAULT_LABELS.length - numBins,
  );
  const binEdges = [];
  let hasAnyEdge = false;

  for (let i = 0; i < numBins; i++) {
    colorScheme.push(
      document.getElementById(`color-bin-${i}`)?.value ||
        BIN_COLOR_PALETTES[3][i] ||
        "#cccccc",
    );
    binLabels.push(
      document.getElementById(`label-bin-${i}`)?.value?.trim() ||
        defaultLabels[i] ||
        "",
    );
    if (i < numBins - 1) {
      const val = document.getElementById(`threshold-bin-${i}`)?.value?.trim();
      if (val) {
        binEdges.push(parseFloat(val));
        hasAnyEdge = true;
      } else {
        binEdges.push(null);
      }
    }
  }

  return {
    colorScheme,
    binLabels,
    binEdges: hasAnyEdge ? binEdges.map((e) => (e != null ? e : 0)) : null,
  };
}

// Make updateFormFieldsVisibility available globally for onchange handler
window.updateFormFieldsVisibility = updateFormFieldsVisibility;

// Toggle choropleth SQL section visibility
window.toggleChoroplethSqlSection = function () {
  const content = document.getElementById("choropleth-sql-content");
  const icon = document.getElementById("choropleth-sql-toggle-icon");
  if (content && icon) {
    if (content.style.display === "none") {
      content.style.display = "block";
      icon.textContent = "▼";
    } else {
      content.style.display = "none";
      icon.textContent = "▶";
    }
  }
};

// Toggle between query builder and custom SQL mode for KPI cards
window.toggleKpiSqlMode = function () {
  const enabled = document.getElementById("kpi-sql-enabled")?.checked;
  const sqlForm = document.getElementById("kpi-sql-form");
  if (sqlForm) sqlForm.style.display = enabled ? "block" : "none";

  const builderSections = [
    "section-query",
    "section-where",
    "section-calculate",
  ];
  if (enabled) {
    builderSections.forEach((id) => {
      const el = document.getElementById(id);
      if (el) el.style.display = "none";
    });
  } else {
    // Restore query builder sections
    builderSections.forEach((id) => {
      const el = document.getElementById(id);
      if (el) el.style.display = "block";
    });
  }
};

// Toggle between query builder and custom SQL mode for choropleth
window.toggleChoroplethSqlMode = function () {
  const enabled = document.getElementById("choropleth-sql-enabled")?.checked;
  const sqlForm = document.getElementById("choropleth-sql-form");
  const querySection = document.getElementById("section-query");
  const groupBySection = document.getElementById("section-groupby");
  const queryBuilderSectionIds = [
    "section-where",
    "section-calculate",
    "section-orderby",
  ];
  const columnMappingSection = document.getElementById(
    "choropleth-column-mapping",
  );

  if (sqlForm) {
    sqlForm.style.display = enabled ? "block" : "none";
  }

  updateFormFieldsVisibility();

  if (enabled) {
    if (querySection) querySection.style.display = "none";
    if (groupBySection) groupBySection.style.display = "none";
    queryBuilderSectionIds.forEach((id) => {
      const section = document.getElementById(id);
      if (section) section.style.display = "none";
    });
    refreshSelectColumnsList();
  } else {
    if (columnMappingSection) {
      columnMappingSection.style.display = "none";
    }
    storeChoroplethSQLPreviewHeaders([]);
    refreshSelectColumnsList();
  }
};

// Handle choropleth SQL preview
async function handleChoroplethSQLPreview() {
  const sqlInput = document.getElementById("choropleth-sql");
  const sql = sqlInput?.value?.trim();

  if (!sql) {
    showError("Please enter a SQL query");
    return;
  }

  const previewBtn = document.getElementById("btn-preview-choropleth-sql");
  const resultContainer = document.getElementById(
    "choropleth-sql-preview-result",
  );
  const infoDiv = document.getElementById("choropleth-sql-preview-info");
  const tableDiv = document.getElementById("choropleth-sql-preview-table");
  const mappingDiv = document.getElementById("choropleth-column-mapping");
  const districtSelect = document.getElementById("choropleth-district-column");
  const valueSelect = document.getElementById("choropleth-value-column");

  // Show loading state
  previewBtn.disabled = true;
  previewBtn.textContent = "⏳ Running...";

  try {
    const result = await previewAdvancedSQL({
      sql: sql,
      filters: {},
      limit: 20,
    });

    resultContainer.style.display = "block";

    if (result.error) {
      infoDiv.innerHTML = `<span style="color: var(--color-error);">Error: ${result.error}</span>`;
      tableDiv.innerHTML = "";
      mappingDiv.style.display = "none";
      storeChoroplethSQLPreviewHeaders([]);
      refreshSelectColumnsList();
    } else {
      infoDiv.innerHTML = `<span style="color: var(--color-success);">✓ ${result.rowCount} rows · ${result.executionTime}</span>`;

      // Build preview table
      if (result.headers && result.rows) {
        let tableHtml =
          '<table style="width: 100%; border-collapse: collapse; font-size: 11px;">';
        tableHtml += "<thead><tr>";
        result.headers.forEach((h) => {
          tableHtml += `<th style="border: 1px solid var(--color-border); padding: 4px; text-align: left; background: var(--color-bg);">${h}</th>`;
        });
        tableHtml += "</tr></thead><tbody>";
        result.rows.slice(0, 5).forEach((row) => {
          tableHtml += "<tr>";
          row.forEach((cell) => {
            const displayVal = cell === null ? "<em>null</em>" : cell;
            tableHtml += `<td style="border: 1px solid var(--color-border); padding: 4px;">${displayVal}</td>`;
          });
          tableHtml += "</tr>";
        });
        tableHtml += "</tbody></table>";
        tableDiv.innerHTML = tableHtml;

        // Populate column mapping dropdowns
        const columnOptions = result.headers
          .map((h) => `<option value="${h}">${h}</option>`)
          .join("");

        districtSelect.innerHTML = `<option value="">-- Select column --</option>${columnOptions}`;
        valueSelect.innerHTML = `<option value="">-- Select column --</option>${columnOptions}`;

        const existingDistrict =
          currentEditingComponent?.component?.columnMapping?.district;
        const existingValue =
          currentEditingComponent?.component?.columnMapping?.value;

        if (existingDistrict && result.headers.includes(existingDistrict)) {
          districtSelect.value = existingDistrict;
        }

        if (existingValue && result.headers.includes(existingValue)) {
          valueSelect.value = existingValue;
        }

        // Auto-select first column as district, second as value
        if (result.headers.length >= 2) {
          if (!districtSelect.value) districtSelect.value = result.headers[0];
          if (
            !valueSelect.value ||
            valueSelect.value === districtSelect.value
          ) {
            valueSelect.value =
              result.headers.find(
                (header) => header !== districtSelect.value,
              ) || "";
          }
        }

        storeChoroplethSQLPreviewHeaders(result.headers);
        refreshSelectColumnsList();

        // Show the column mapping section
        mappingDiv.style.display = "block";
      } else {
        tableDiv.innerHTML = "<em>No data returned</em>";
        mappingDiv.style.display = "none";
        storeChoroplethSQLPreviewHeaders([]);
        refreshSelectColumnsList();
      }
    }
  } catch (err) {
    resultContainer.style.display = "block";
    infoDiv.innerHTML = `<span style="color: var(--color-error);">Error: ${err.message}</span>`;
    tableDiv.innerHTML = "";
    mappingDiv.style.display = "none";
    storeChoroplethSQLPreviewHeaders([]);
    refreshSelectColumnsList();
  } finally {
    previewBtn.disabled = false;
    previewBtn.textContent = "▶ Preview SQL & Map Columns";
  }
}

// Toggle chart SQL section visibility (collapsible header)
window.toggleChartSqlSection = function () {
  const content = document.getElementById("chart-sql-content");
  const icon = document.getElementById("chart-sql-toggle-icon");
  if (content && icon) {
    if (content.style.display === "none") {
      content.style.display = "block";
      icon.textContent = "▼";
    } else {
      content.style.display = "none";
      icon.textContent = "▶";
    }
  }
};

// Toggle between query builder and custom SQL mode for chart types
window.toggleChartSqlMode = function () {
  const enabled = document.getElementById("chart-sql-enabled")?.checked;
  const sqlForm = document.getElementById("chart-sql-form");
  const querySection = document.getElementById("section-query");
  const groupBySection = document.getElementById("section-groupby");

  if (sqlForm) {
    sqlForm.style.display = enabled ? "block" : "none";
  }

  // Show/hide dataset mapping for bar_line
  const dsMapping = document.getElementById("chart-sql-dataset-mapping");
  if (dsMapping) {
    const type = document.getElementById("comp-type")?.value || "";
    dsMapping.style.display = enabled && type === "bar_line" ? "block" : "none";
  }

  // Hide/show query builder sections when SQL mode is enabled
  if (enabled) {
    if (querySection) querySection.style.display = "none";
    if (groupBySection) groupBySection.style.display = "none";
    const seriesBySection = document.getElementById("section-seriesby");
    if (seriesBySection) seriesBySection.style.display = "none";
    // Hide only query-builder sub-sections that have no meaning for raw SQL.
    // Period Range stays visible — backend honors periodLimit for raw SQL when {{*_filter}} placeholders are present.
    [
      "section-where",
      "section-calculate",
      "section-orderby",
      "section-select-columns",
    ].forEach((id) => {
      const el = document.getElementById(id);
      if (el) el.style.display = "none";
    });
  } else {
    // Restore sections based on the current component type's visibility
    updateFormFieldsVisibility();
  }
  const periodLimitSqlHint = document.getElementById("period-limit-sql-hint");
  if (periodLimitSqlHint) {
    periodLimitSqlHint.style.display = enabled ? "block" : "none";
  }
};

window.toggleHeaderTooltipsSection = function () {
  const content = document.getElementById("headertooltips-content");
  const icon = document.getElementById("headertooltips-toggle-icon");
  if (content && icon) {
    if (content.style.display === "none") {
      content.style.display = "block";
      icon.innerHTML = "&#9660;";
    } else {
      content.style.display = "none";
      icon.innerHTML = "&#9654;";
    }
  }
};

// Handle chart SQL preview
async function handleChartSQLPreview() {
  const sqlInput = document.getElementById("chart-sql");
  const sql = sqlInput?.value?.trim();

  if (!sql) {
    showError("Please enter a SQL query");
    return;
  }

  const previewBtn = document.getElementById("btn-preview-chart-sql");
  const resultContainer = document.getElementById("chart-sql-preview-result");
  const infoDiv = document.getElementById("chart-sql-preview-info");
  const tableDiv = document.getElementById("chart-sql-preview-table");

  // Show loading state
  previewBtn.disabled = true;
  previewBtn.textContent = "⏳ Running...";

  try {
    const result = await previewAdvancedSQL({
      sql: sql,
      filters: {},
      limit: 20,
    });

    resultContainer.style.display = "block";

    if (result.error) {
      infoDiv.innerHTML = `<span style="color: var(--color-error);">Error: ${result.error}</span>`;
      tableDiv.innerHTML = "";
    } else {
      infoDiv.innerHTML = `<span style="color: var(--color-success);">✓ ${result.rowCount} rows · ${result.headers?.length || 0} columns · ${result.executionTime}</span>`;

      // Build preview table
      if (result.headers && result.rows) {
        let tableHtml =
          '<table style="width: 100%; border-collapse: collapse; font-size: 11px;">';
        tableHtml += "<thead><tr>";
        result.headers.forEach((h) => {
          tableHtml += `<th style="border: 1px solid var(--color-border); padding: 4px; text-align: left; background: var(--color-bg);">${h}</th>`;
        });
        tableHtml += "</tr></thead><tbody>";
        result.rows.slice(0, 5).forEach((row) => {
          tableHtml += "<tr>";
          row.forEach((cell) => {
            const displayVal = cell === null ? "<em>null</em>" : cell;
            tableHtml += `<td style="border: 1px solid var(--color-border); padding: 4px;">${displayVal}</td>`;
          });
          tableHtml += "</tr>";
        });
        tableHtml += "</tbody></table>";
        tableDiv.innerHTML = tableHtml;

        // Auto-populate dataset mapping for bar_line (skip first column = labels)
        const type = document.getElementById("comp-type")?.value || "";
        if (type === "bar_line" && result.headers.length > 1) {
          const list = document.getElementById("chart-sql-datasets-list");
          if (list) {
            // Only auto-populate if no rows exist yet
            const existingRows = list.querySelectorAll(
              ".chart-sql-dataset-row",
            );
            if (existingRows.length === 0) {
              const dataHeaders = result.headers.slice(1);
              dataHeaders.forEach((h, i) => {
                addChartSqlDatasetRow(h, i === 0 ? "bar" : "line");
              });
            }
          }
        }
      } else {
        tableDiv.innerHTML = "<em>No data returned</em>";
      }
    }
  } catch (err) {
    resultContainer.style.display = "block";
    infoDiv.innerHTML = `<span style="color: var(--color-error);">Error: ${err.message}</span>`;
    tableDiv.innerHTML = "";
  } finally {
    previewBtn.disabled = false;
    previewBtn.textContent = "▶ Preview SQL Results";
  }
}

// Add a dataset mapping row for bar_line custom SQL
const defaultDsPalette = [
  "#0f766e",
  "#e74c3c",
  "#2ecc71",
  "#d97706",
  "#6366f1",
  "#14b8a6",
  "#334155",
  "#f59e0b",
];
function addChartSqlDatasetRow(label, chartType, color) {
  const list = document.getElementById("chart-sql-datasets-list");
  if (!list) return;
  const idx = list.querySelectorAll(".chart-sql-dataset-row").length;
  const defaultColor = color || defaultDsPalette[idx % defaultDsPalette.length];
  const defaultType = chartType || (idx === 0 ? "bar" : "line");
  const row = document.createElement("div");
  row.className = "chart-sql-dataset-row";
  row.style.cssText =
    "display: flex; gap: 8px; margin-bottom: 8px; align-items: center;";
  row.innerHTML = `
        <input type="text" class="form-input chart-sql-ds-label" value="${label || ""}" placeholder="Column alias" style="flex: 1; margin: 0;">
        <select class="form-input chart-sql-ds-type" style="width: 80px; margin: 0;">
            <option value="bar" ${defaultType !== "line" ? "selected" : ""}>Bar</option>
            <option value="line" ${defaultType === "line" ? "selected" : ""}>Line</option>
        </select>
        <input type="color" class="form-input chart-sql-ds-color" value="${defaultColor}" style="width: 40px; height: 32px; padding: 2px; margin: 0;">
        <button type="button" class="btn-small btn-remove-chart-sql-ds" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">×</button>
    `;
  list.appendChild(row);
}

// Toggle reference lines section visibility
window.toggleReferenceLinesSection = function () {
  const content = document.getElementById("referencelines-content");
  const icon = document.getElementById("referencelines-toggle-icon");
  if (content && icon) {
    if (content.style.display === "none") {
      content.style.display = "block";
      icon.textContent = "▼";
    } else {
      content.style.display = "none";
      icon.textContent = "▶";
    }
  }
};

// Add a new reference line row
export function addReferenceLine() {
  const list = document.getElementById("referencelines-list");
  if (!list) return;

  // Remove "no reference lines" message if present
  const noMsg = document.getElementById("no-reflines-msg");
  if (noMsg) noMsg.remove();

  const row = document.createElement("div");
  row.className = "refline-row";
  row.style.cssText =
    "display: flex; gap: 8px; margin-bottom: 8px; align-items: center; flex-wrap: wrap;";
  row.innerHTML = `
        <input type="number" class="form-input refline-value" placeholder="Value" step="any" style="flex: 0 0 80px; margin: 0;">
        <input type="text" class="form-input refline-label" placeholder="Label (e.g., Target)" style="flex: 1; min-width: 100px; margin: 0;">
        <input type="color" class="form-input refline-color" value="#22c55e" style="width: 40px; height: 32px; padding: 2px; margin: 0;">
        <select class="form-input refline-style" style="flex: 0 0 80px; margin: 0;">
            <option value="dashed">Dashed</option>
            <option value="dotted">Dotted</option>
            <option value="solid">Solid</option>
        </select>
        <select class="form-input refline-axis" title="Y-axis" style="flex: 0 0 60px; margin: 0;">
            <option value="y">Left (y)</option>
            <option value="y1">Right (y1)</option>
        </select>
        <button class="btn-small btn-remove-refline" style="background: #ff6b6b; color: white; margin: 0; padding: 6px 8px;">×</button>
    `;

  // Add remove handler
  const removeBtn = row.querySelector(".btn-remove-refline");
  if (removeBtn) {
    removeBtn.addEventListener("click", () => row.remove());
  }

  list.appendChild(row);
}

// Auto-expand reference lines section if component has reference lines
export function autoExpandReferenceLinesIfNeeded() {
  if (currentEditingComponent?.component?.referenceLines?.length > 0) {
    const content = document.getElementById("referencelines-content");
    const icon = document.getElementById("referencelines-toggle-icon");
    if (content && icon) {
      content.style.display = "block";
      icon.textContent = "▼";
    }
  }
}

// Export for use in showComponentModal
export { handleChoroplethSQLPreview, handleChartSQLPreview };
