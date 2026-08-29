/**
 * FilterManager Module
 * Consolidates all filter logic: creation, value read/write, persistence,
 * multiselect lifecycle, and document-level listener cleanup.
 *
 * State design: _values is the source of truth for multiselect filters.
 * The DOM is a view — it reflects _values, never drives it.
 * Single-select <select> elements and the facility autocomplete are
 * standard form elements; their DOM state is the authoritative value.
 *
 * Follows the apiCache.js IIFE pattern. Exposes window.filterManager.
 */

const filterManager = (function () {
  // ---- Internal state ----
  let _metadata = null; // Current report's metadata (filter config)
  let _reportId = null; // Current report ID (for localStorage keying)
  let _abortController = null; // AbortController for document-level listeners
  let _multiselectPanels = []; // Panels appended to document.body
  let _values = {}; // Source of truth: { filterName: ['val1','val2'] } for multiselects
  let _optionValues = {}; // All available options: { filterName: ['opt1','opt2',...] }
  let _fyMode = false; // true = fiscal year mode, false = calendar year mode
  let _fyValue = null; // Selected FY start year (e.g., 2024 for "FY 2024/25")

  // ---- Constants ----
  const BUILTIN_MULTISELECT = new Set([
    "month",
    "week",
    "quarter",
    "region",
    "district",
  ]);
  // facility_select is a single-select <select> dropdown variant of the facility filter;
  // it sends its value under the "facility" URL param (via filterDef.paramName).
  const BUILTIN_FILTERS = new Set([
    "year",
    "month",
    "week",
    "quarter",
    "region",
    "district",
    "facility",
    "facility_select",
  ]);

  const MULTI_PANEL_TITLE = {
    region: "Regions",
    facility: "Facilities",
    district: "Districts",
    month: "Months",
    week: "Weeks",
    quarter: "Quarters",
  };

  const MONTH_NAMES = [
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

  // ---- Private helpers ----

  function _escapeHtml(str) {
    const div = document.createElement("div");
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
  }

  function _capitalizeFirstLetter(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
  }

  function _multiselectPanelTitle(filterName, filterDef) {
    if (MULTI_PANEL_TITLE[filterName]) {
      return MULTI_PANEL_TITLE[filterName];
    }
    if (filterDef && filterDef.label) {
      return _shortFilterLabel(filterDef, filterName);
    }
    return _capitalizeFirstLetter(filterName) + "s";
  }

  function _dependentParentHint(parentFilter) {
    return `${_capitalizeFirstLetter(parentFilter)} first`;
  }

  /** Strip legacy "Select …" / "(ies)" wording from API labels. */
  function _shortFilterLabel(filterDef, filterName) {
    let raw =
      (filterDef && filterDef.label) || _capitalizeFirstLetter(filterName);
    raw = String(raw).trim();
    raw = raw.replace(/^Select\s+/i, "");
    raw = raw.replace(/\s*\(ies\)\s*$/i, "");
    raw = raw.replace(/\s*\(s\)\s*$/i, "");
    return raw.trim();
  }

  // Left-to-right filter bar order: period controls first, geography, facility last.
  // Custom / unknown filters keep YAML order in the middle (rank 100).
  const FILTER_BAR_DISPLAY_RANK = {
    year: 10,
    quarter: 20,
    month: 30,
    week: 40,
    region: 200,
    district: 210,
    facility: 300,
    facility_select: 300,
  };

  // Mobile summary: finer period first, then year, then geography (matches user scan order).
  const FILTER_SUMMARY_KEY_ORDER = {
    yearmonths: 5,
    quarter: 10,
    month: 20,
    week: 30,
    year: 40,
    region: 200,
    district: 210,
    facility: 300,
    facility_select: 300,
  };

  function _sortFiltersForDisplay(filters) {
    if (!filters || filters.length < 2) {
      return filters ? filters.slice() : [];
    }
    return filters
      .map((name, origIndex) => ({ name, origIndex }))
      .sort((a, b) => {
        const ra = FILTER_BAR_DISPLAY_RANK[a.name];
        const rb = FILTER_BAR_DISPLAY_RANK[b.name];
        const rankA = ra !== undefined ? ra : 100;
        const rankB = rb !== undefined ? rb : 100;
        if (rankA !== rankB) return rankA - rankB;
        return a.origIndex - b.origIndex;
      })
      .map((x) => x.name);
  }

  function _sortSummaryKeys(keys) {
    if (!keys || keys.length < 2) return keys ? keys.slice() : [];
    return keys.slice().sort((a, b) => {
      const ra = FILTER_SUMMARY_KEY_ORDER[a];
      const rb = FILTER_SUMMARY_KEY_ORDER[b];
      const rankA = ra !== undefined ? ra : 100;
      const rankB = rb !== undefined ? rb : 100;
      if (rankA !== rankB) return rankA - rankB;
      return keys.indexOf(a) - keys.indexOf(b);
    });
  }

  /**
   * Determine if a filter uses multiselect UI
   */
  function _isMultiselect(filterName, filterDef) {
    if (BUILTIN_MULTISELECT.has(filterName)) return true;
    if (
      filterDef &&
      filterDef.type === "multiselect" &&
      !BUILTIN_MULTISELECT.has(filterName) &&
      filterName !== "facility"
    )
      return true;
    return false;
  }

  function _isCustomSingleSelect(filterName, filterDef) {
    return !!(
      filterDef &&
      filterDef.type === "select" &&
      !BUILTIN_FILTERS.has(filterName)
    );
  }

  function _defaultSingleSelectValue(filterName, filterDef, optionValues) {
    if (
      !_isCustomSingleSelect(filterName, filterDef) ||
      !optionValues ||
      optionValues.length === 0
    ) {
      return "";
    }

    // If noDefault is set (and None option exists), start with blank selection
    if (filterDef.noDefault && !filterDef.hideNone) {
      return "";
    }

    const configuredDefault =
      filterDef.defaultValue !== undefined && filterDef.defaultValue !== null
        ? String(filterDef.defaultValue)
        : "";
    if (configuredDefault && optionValues.includes(configuredDefault)) {
      return configuredDefault;
    }
    return optionValues[0];
  }

  function _resetSingleSelectValue(filterName, filterDef) {
    const selectElement = document.getElementById(`filter-${filterName}`);
    if (!selectElement) return;

    if (!_isCustomSingleSelect(filterName, filterDef)) {
      selectElement.value = "";
      return;
    }

    const optionValues = Array.from(selectElement.options)
      .map((opt) => opt.value)
      .filter(Boolean);
    selectElement.value = _defaultSingleSelectValue(
      filterName,
      filterDef,
      optionValues,
    );
  }

  /**
   * Format a filter option value for display
   */
  function _formatOption(filterName, value) {
    if (filterName === "month" && !isNaN(value)) {
      return MONTH_NAMES[parseInt(value) - 1] || value;
    }
    return String(value);
  }

  // ---- Smart time defaults (pure functions) ----

  function _getSmartDefaultWeek() {
    const today = new Date();
    const dayOfWeek = today.getDay();
    const weeksAgo = dayOfWeek >= 4 || dayOfWeek === 0 ? 1 : 2;

    const targetDate = new Date(today);
    targetDate.setDate(today.getDate() - weeksAgo * 7);

    const d = new Date(targetDate.getTime());
    d.setHours(0, 0, 0, 0);
    d.setDate(d.getDate() + 3 - ((d.getDay() + 6) % 7));
    const week1 = new Date(d.getFullYear(), 0, 4);
    const weekNum =
      1 +
      Math.round(
        ((d.getTime() - week1.getTime()) / 86400000 -
          3 +
          ((week1.getDay() + 6) % 7)) /
          7,
      );
    const yearForWeek = d.getFullYear();

    return `${yearForWeek}W${String(weekNum).padStart(2, "0")}`;
  }

  function _getSmartDefaultMonth() {
    const today = new Date();
    const monthsAgo = today.getDate() > 20 ? 1 : 2;
    const targetDate = new Date(
      today.getFullYear(),
      today.getMonth() - monthsAgo,
      1,
    );
    return {
      year: targetDate.getFullYear(),
      month: targetDate.getMonth() + 1,
    };
  }

  function _getSmartDefaultQuarter() {
    const today = new Date();
    const quartersAgo = today.getDate() > 20 ? 1 : 2;
    const currentQuarter = Math.floor(today.getMonth() / 3) + 1;
    const currentYear = today.getFullYear();

    let targetQuarter = currentQuarter - quartersAgo;
    let targetYear = currentYear;
    while (targetQuarter <= 0) {
      targetQuarter += 4;
      targetYear--;
    }
    return { year: targetYear, quarter: targetQuarter };
  }

  function _quarterForMonth(monthValue) {
    const m = parseInt(monthValue, 10);
    if (isNaN(m) || m < 1 || m > 12) return null;
    return String(Math.floor((m - 1) / 3) + 1);
  }

  function _monthFromISOWeek(weekCode) {
    const match = /^([0-9]{4})W([0-9]{1,2})$/.exec(String(weekCode).trim());
    if (!match) return null;

    const year = parseInt(match[1], 10);
    const week = parseInt(match[2], 10);
    if (isNaN(year) || isNaN(week) || week < 1 || week > 53) return null;

    const jan4 = new Date(Date.UTC(year, 0, 4));
    const jan4Day = jan4.getUTCDay() || 7;
    const mondayWeek1 = new Date(jan4);
    mondayWeek1.setUTCDate(jan4.getUTCDate() - (jan4Day - 1));

    const weekStart = new Date(mondayWeek1);
    weekStart.setUTCDate(mondayWeek1.getUTCDate() + (week - 1) * 7);
    return String(weekStart.getUTCMonth() + 1);
  }

  function _hasFilterSelection(filterName) {
    const filterDef = _metadata?.filterDefinitions?.[filterName];
    if (!filterDef) return false;

    if (_isMultiselect(filterName, filterDef) || filterName === "facility") {
      return (
        Array.isArray(_values[filterName]) && _values[filterName].length > 0
      );
    }

    const selectElement = document.getElementById(`filter-${filterName}`);
    return !!(selectElement && selectElement.value);
  }

  function _autoFillTimeFromSelection(changedFilterName) {
    const availableFilters = _metadata?.filters || [];
    const hasQuarter = availableFilters.includes("quarter");
    const hasMonth = availableFilters.includes("month");
    const hasWeek = availableFilters.includes("week");

    if (changedFilterName === "month" && hasQuarter) {
      const selectedMonths = _values.month || [];
      const selectedQuarters = _values.quarter || [];
      if (selectedMonths.length > 0 && selectedQuarters.length === 0) {
        const inferredQuarters = Array.from(
          new Set(selectedMonths.map(_quarterForMonth).filter(Boolean)),
        );
        if (inferredQuarters.length > 0) {
          _values.quarter = inferredQuarters;
          _syncCheckboxesToValues("quarter");
        }
      }
    }

    if (changedFilterName === "week" && hasWeek) {
      const selectedWeeks = _values.week || [];
      if (selectedWeeks.length === 0) return;

      const yearSelect = document.getElementById("filter-year");
      if (yearSelect && !yearSelect.value) {
        const m = /^([0-9]{4})W[0-9]{1,2}$/.exec(
          String(selectedWeeks[0]).trim(),
        );
        if (m && yearSelect.querySelector(`option[value="${m[1]}"]`)) {
          yearSelect.value = m[1];
        }
      }

      if (hasMonth && (!_values.month || _values.month.length === 0)) {
        const inferredMonths = Array.from(
          new Set(selectedWeeks.map(_monthFromISOWeek).filter(Boolean)),
        );
        if (inferredMonths.length > 0) {
          _values.month = inferredMonths;
          _syncCheckboxesToValues("month");
        }
      }

      if (
        hasQuarter &&
        (!_values.quarter || _values.quarter.length === 0) &&
        _values.month &&
        _values.month.length > 0
      ) {
        const inferredQuarters = Array.from(
          new Set(_values.month.map(_quarterForMonth).filter(Boolean)),
        );
        if (inferredQuarters.length > 0) {
          _values.quarter = inferredQuarters;
          _syncCheckboxesToValues("quarter");
        }
      }
    }
  }

  // ---- Fiscal Year helpers ----
  // Uganda FY: July 1 - June 30. FY 2024/25 = Jul 2024 - Jun 2025.

  /**
   * Check if the current report supports FY mode.
   * Requires both 'year' and 'month' in filters list.
   */
  function _isFYEligible() {
    if (!_metadata || !_metadata.filters) return false;
    return (
      _metadata.filters.includes("year") && _metadata.filters.includes("month")
    );
  }

  /**
   * Build FY options from available year data in the year dropdown.
   * If years [2022, 2023, 2024, 2025] are available, FY options are:
   * FY 2022/23, FY 2023/24, FY 2024/25 (each needs data in both calendar years).
   */
  function _buildFYOptions() {
    const yearSelect = document.getElementById("filter-year");
    if (!yearSelect) return [];

    const years = [];
    yearSelect.querySelectorAll("option").forEach((opt) => {
      const val = parseInt(opt.value);
      if (!isNaN(val)) years.push(val);
    });
    years.sort((a, b) => a - b);

    const fyOptions = [];
    for (const y of years) {
      // FY needs both start year and end year to have data
      if (years.includes(y + 1)) {
        fyOptions.push(y);
      }
    }
    return fyOptions; // e.g., [2022, 2023, 2024] for FY 2022/23, 2023/24, 2024/25
  }

  /**
   * Get the yearmonths param value for a given FY start year.
   * FY 2024/25 → "2024:7,8,9,10,11,12|2025:1,2,3,4,5,6"
   */
  function _fyToYearMonths(startYear) {
    return `${startYear}:7,8,9,10,11,12|${startYear + 1}:1,2,3,4,5,6`;
  }

  /**
   * Format FY label: 2024 → "FY 2024/25"
   */
  function _formatFYLabel(startYear) {
    const endYearShort = String(startYear + 1).slice(-2);
    return `FY ${startYear}/${endYearShort}`;
  }

  /**
   * Get smart default FY: latest FY that could have data.
   * If current date is Jul+ → current FY; otherwise previous FY.
   */
  function _getSmartDefaultFY(availableFYs) {
    if (!availableFYs || availableFYs.length === 0) return null;
    const now = new Date();
    const currentFYStart =
      now.getMonth() >= 6 ? now.getFullYear() : now.getFullYear() - 1;
    // Pick the closest available FY that doesn't exceed current FY
    const valid = availableFYs.filter((fy) => fy <= currentFYStart);
    return valid.length > 0
      ? valid[valid.length - 1]
      : availableFYs[availableFYs.length - 1];
  }

  /**
   * Add CY|FY toggle to the year filter item.
   * Creates: [CY|FY toggle] [year select OR FY select]
   * In FY mode: hides CY year select, shows FY select, hides month filter.
   */
  function _addFYToggle(yearFilterItem) {
    const fyOptions = _buildFYOptions();
    if (fyOptions.length === 0) return; // Not enough year data for any FY

    const yearSelect = document.getElementById("filter-year");
    if (!yearSelect) return;

    // Create FY select (hidden by default)
    const fySelect = document.createElement("select");
    fySelect.className = "filter-item-select";
    fySelect.id = "filter-fy";
    fySelect.style.display = "none";

    fyOptions.forEach((startYear) => {
      const opt = document.createElement("option");
      opt.value = String(startYear);
      opt.textContent = _formatFYLabel(startYear);
      fySelect.appendChild(opt);
    });

    // Create toggle button
    const toggle = document.createElement("button");
    toggle.type = "button";
    toggle.className = "fy-toggle";
    toggle.innerHTML =
      '<span class="fy-toggle-opt" data-mode="cy">CY</span><span class="fy-toggle-opt" data-mode="fy">FY</span>';

    // Insert toggle before the label, and FY select after year select
    const label = yearFilterItem.querySelector(".filter-item-label");
    yearFilterItem.insertBefore(toggle, label);
    yearSelect.parentNode.insertBefore(fySelect, yearSelect.nextSibling);

    const updateToggleUI = () => {
      toggle.querySelectorAll(".fy-toggle-opt").forEach((opt) => {
        opt.classList.toggle(
          "active",
          opt.dataset.mode === (_fyMode ? "fy" : "cy"),
        );
      });

      yearSelect.style.display = _fyMode ? "none" : "";
      fySelect.style.display = _fyMode ? "" : "none";

      // Update label
      label.textContent = (_fyMode ? "Fiscal Year" : "Year") + ":";

      // Hide/show month filter item
      const monthItem = document.querySelector(
        '.filter-item[data-filter="month"]',
      );
      if (monthItem) {
        monthItem.style.display = _fyMode ? "none" : "";
      }
    };

    // Toggle click handler
    toggle.addEventListener("click", (e) => {
      const clickedOpt = e.target.closest(".fy-toggle-opt");
      if (!clickedOpt) return;

      const newMode = clickedOpt.dataset.mode === "fy";
      if (newMode === _fyMode) return;

      _fyMode = newMode;
      if (_fyMode) {
        // Entering FY mode: pick smart default
        const defaultFY = _getSmartDefaultFY(fyOptions);
        if (defaultFY !== null) {
          fySelect.value = String(defaultFY);
          _fyValue = defaultFY;
        }
      } else {
        // Leaving FY mode: clear FY state
        _fyValue = null;
      }

      updateToggleUI();
      _saveState();
      _updatePills();
      loadReport();
    });

    // FY select change handler
    fySelect.addEventListener("change", () => {
      _fyValue = parseInt(fySelect.value);
      _saveState();
      _updatePills();
      loadReport();
    });

    // Restore FY mode from localStorage if saved
    try {
      const saved = localStorage.getItem(`filterState-${_reportId}`);
      if (saved) {
        const state = JSON.parse(saved);
        if (state.fyMode) {
          _fyMode = true;
          _fyValue = state.fyValue || null;
          if (
            _fyValue &&
            fySelect.querySelector(`option[value="${_fyValue}"]`)
          ) {
            fySelect.value = String(_fyValue);
          } else {
            const defaultFY = _getSmartDefaultFY(fyOptions);
            if (defaultFY !== null) {
              fySelect.value = String(defaultFY);
              _fyValue = defaultFY;
            }
          }
        }
      }
    } catch (e) {
      /* ignore */
    }

    updateToggleUI();
  }

  // ---- Sync DOM checkboxes from _values (DOM as view, _values as source of truth) ----

  function _syncCheckboxesToValues(filterName) {
    const optionsContainer = document.getElementById(
      `filter-${filterName}-options`,
    );
    if (!optionsContainer) return;

    const selected = new Set(_values[filterName] || []);
    optionsContainer
      .querySelectorAll('input[type="checkbox"]')
      .forEach((cb) => {
        const checked = selected.has(cb.value);
        cb.checked = checked;
        cb.closest(".multiselect-option")?.classList.toggle(
          "selected",
          checked,
        );
      });
    _updateTrigger(filterName);
  }

  // ---- Multiselect trigger text (reads _values, not DOM) ----

  function _updateTrigger(filterName) {
    const trigger = document.getElementById(`filter-${filterName}-trigger`);
    if (!trigger) return;

    const selected = _values[filterName] || [];
    const count = selected.length;
    const total = _optionValues[filterName]?.length || 0;

    if (count === 0) {
      trigger.textContent = "None";
    } else if (count === total) {
      trigger.textContent = "All";
    } else if (count <= 2) {
      trigger.textContent = selected
        .map((v) => _formatOption(filterName, v))
        .join(", ");
    } else {
      trigger.textContent = `${count} selected`;
    }

    _updatePills();
  }

  // ---- Close all multiselect panels ----

  function _closeAllMultiselects(exceptPanel) {
    document.querySelectorAll(".multiselect-panel.open").forEach((panel) => {
      if (panel !== exceptPanel) {
        panel.classList.remove("open");
      }
    });
  }

  // ---- Dependent filter states ----

  function _updateDependentStates() {
    if (!_metadata) return;

    for (const filterName of _metadata.filters || []) {
      const filterDef = _metadata.filterDefinitions[filterName];
      if (
        !filterDef ||
        !filterDef.dependsOn ||
        filterDef.dependsOn.length === 0
      )
        continue;

      const trigger = document.getElementById(`filter-${filterName}-trigger`);
      if (!trigger) continue;

      let allParentsSet = true;
      let missingParent = null;
      for (const parentFilter of filterDef.dependsOn) {
        if (!_hasFilterSelection(parentFilter)) {
          allParentsSet = false;
          missingParent = parentFilter;
          break;
        }
      }

      trigger.disabled = !allParentsSet;
      trigger.style.opacity = allParentsSet ? "1" : "0.5";
      trigger.style.cursor = allParentsSet ? "pointer" : "not-allowed";

      if (!allParentsSet && missingParent) {
        trigger.textContent = _dependentParentHint(missingParent);
      }
    }
  }

  // ---- Clear All visibility ----

  function _updateClearAllVisibility() {
    const clearAllBtn = document.getElementById("clear-all-filters");
    if (!clearAllBtn || !_metadata) return;

    const hasActiveFilters = Object.keys(_getValues()).length > 0;
    clearAllBtn.style.display = hasActiveFilters ? "inline-block" : "none";
  }

  // ---- Filter summary for mobile ----

  function _updateFilterSummary() {
    const summaryEl = document.getElementById("filter-summary");
    if (!summaryEl || !_metadata) return;

    const values = _getValues();
    const parts = [];

    for (const key of _sortSummaryKeys(Object.keys(values))) {
      const value = values[key];
      if (key === "yearmonths") {
        // Show FY label instead of raw yearmonths param
        if (_fyValue) parts.push(_formatFYLabel(_fyValue));
        continue;
      }
      if (key === "district") {
        parts.push(value.replace(" District", ""));
      } else if (key === "year") {
        parts.push(value);
      } else if (key === "month") {
        const months = value.split(",").map((m) => {
          const idx = parseInt(m) - 1;
          return MONTH_NAMES[idx] ? MONTH_NAMES[idx].slice(0, 3) : m;
        });
        parts.push(months.join(", "));
      } else if (key === "week") {
        const weeks = value.split(",");
        if (weeks.length <= 2) {
          parts.push(weeks.join(", "));
        } else {
          parts.push(`${weeks.length} weeks`);
        }
      } else if (key === "quarter") {
        const quarters = value.split(",").map((q) => `Q${q}`);
        if (quarters.length <= 2) {
          parts.push(quarters.join(", "));
        } else {
          parts.push(`${quarters.length} quarters`);
        }
      } else {
        parts.push(value);
      }
    }

    summaryEl.textContent =
      parts.length > 0 ? parts.join(" \u2022 ") : "All filters cleared";
  }

  // ---- Filter count badge ----

  function _updateFilterCountBadge() {
    const badge = document.getElementById("filter-count-badge");
    if (!badge || !_metadata) return;

    const count = Object.keys(_getValues()).length;

    if (count > 0) {
      badge.textContent = count;
      badge.style.display = "inline-flex";
    } else {
      badge.style.display = "none";
    }
  }

  // ---- Combined pill update ----

  function _updatePills() {
    _updateClearAllVisibility();
    _updateFilterSummary();
    _updateFilterCountBadge();
  }

  // ---- Mobile filter toggle ----
  // 1200px (not phone-only 768): small laptops were cramped with the filter bar open
  // next to sidebar/header nav, so we use the same “compact chrome” breakpoint as app.js.

  function _initMobileToggle() {
    const filterBar = document.getElementById("filter-bar");
    const toggleHeader = document.getElementById("filter-bar-toggle");
    if (!toggleHeader || !filterBar) return;

    if (window.innerWidth <= 1200) {
      filterBar.classList.add("collapsed");
    }

    toggleHeader.addEventListener("click", () => {
      filterBar.classList.toggle("collapsed");
    });

    window.addEventListener("resize", () => {
      if (window.innerWidth > 1200) {
        filterBar.classList.remove("collapsed");
      }
    });
  }

  // ---- Create multiselect dropdown ----

  function _createMultiselectDropdown(filterName, filterDef) {
    const container = document.createElement("div");
    container.className = "multiselect-container";
    container.setAttribute("data-filter", filterName);

    const trigger = document.createElement("button");
    trigger.type = "button";
    trigger.className = "multiselect-trigger";
    trigger.id = `filter-${filterName}-trigger`;
    trigger.textContent = "All";

    const panel = document.createElement("div");
    panel.className = "multiselect-panel";
    panel.id = `filter-${filterName}-panel`;

    const header = document.createElement("div");
    header.className = "multiselect-header";
    header.innerHTML = `
            <span class="multiselect-title">${_escapeHtml(_multiselectPanelTitle(filterName, filterDef))}</span>
            <div class="multiselect-actions">
                <button type="button" class="multiselect-action" data-action="all">All</button>
                <button type="button" class="multiselect-action" data-action="clear">Clear</button>
            </div>
        `;

    const options = document.createElement("div");
    options.className = "multiselect-options";
    options.id = `filter-${filterName}-options`;

    const footer = document.createElement("div");
    footer.className = "multiselect-footer";
    footer.innerHTML = `<button type="button" class="multiselect-apply">Apply</button>`;

    panel.appendChild(header);
    panel.appendChild(options);
    panel.appendChild(footer);

    // Append panel to body for proper z-index stacking
    document.body.appendChild(panel);
    _multiselectPanels.push(panel);

    // Toggle panel on trigger click
    trigger.addEventListener("click", (e) => {
      e.stopPropagation();
      const isOpen = panel.classList.contains("open");
      _closeAllMultiselects(panel);

      if (!isOpen) {
        const rect = trigger.getBoundingClientRect();
        panel.style.top = `${rect.bottom + 4}px`;
        panel.style.left = `${rect.left}px`;
        panel.classList.add("open");
      } else {
        panel.classList.remove("open");
      }
    });

    // "All" button — set _values to every available option
    header
      .querySelector('[data-action="all"]')
      .addEventListener("click", (e) => {
        e.stopPropagation();
        _values[filterName] = [...(_optionValues[filterName] || [])];
        _syncCheckboxesToValues(filterName);
      });

    // "Clear" button — remove this filter from _values
    header
      .querySelector('[data-action="clear"]')
      .addEventListener("click", (e) => {
        e.stopPropagation();
        delete _values[filterName];
        _syncCheckboxesToValues(filterName);
      });

    // Apply button — save state and reload report
    footer
      .querySelector(".multiselect-apply")
      .addEventListener("click", async (e) => {
        e.stopPropagation();
        panel.classList.remove("open");

        _autoFillTimeFromSelection(filterName);
        _saveState();

        // Cascade to dependent filters (e.g., region → district)
        const filterDef = _metadata?.filterDefinitions?.[filterName];
        if (filterDef?.cascadesTo?.length > 0) {
          const currentValues = _getValues();
          for (const dependentFilter of filterDef.cascadesTo) {
            const dependentDef = _metadata.filterDefinitions[dependentFilter];
            if (!dependentDef) continue;

            // Clear facility tags when reloading (stale selections from a different district cause empty results)
            if (dependentFilter === "facility") {
              delete _values["facility"];
              const facilitySelected = document.getElementById(
                "filter-facility-selected",
              );
              if (facilitySelected) facilitySelected.innerHTML = "";
            }

            // _loadFilterOptions clears _values[dependentFilter] for multiselects
            await _loadFilterOptions(
              dependentFilter,
              dependentDef,
              currentValues,
            );

            // When region cascades to district, auto-select all loaded districts
            // then also refresh facility options using those freshly selected districts
            if (
              dependentFilter === "district" &&
              filterName === "region" &&
              _optionValues["district"]?.length > 0
            ) {
              _values["district"] = [..._optionValues["district"]];
              _syncCheckboxesToValues("district");
              _updateTrigger("district");

              // Propagate the new district selection down to facility
              const facilityDef = _metadata?.filterDefinitions?.["facility"];
              if (facilityDef) {
                delete _values["facility"];
                const facilitySelected = document.getElementById(
                  "filter-facility-selected",
                );
                if (facilitySelected) facilitySelected.innerHTML = "";
                // _getValues() now includes the auto-selected districts
                await _loadFilterOptions("facility", facilityDef, _getValues());
              }
            }

            // For single-select dependents, also clear the DOM element
            if (
              !_isMultiselect(dependentFilter, dependentDef) &&
              dependentFilter !== "facility"
            ) {
              _resetSingleSelectValue(dependentFilter, dependentDef);
            }
          }
        }

        await loadReport();
      });

    // Prevent panel clicks from closing
    panel.addEventListener("click", (e) => e.stopPropagation());

    // Delegated listener for checkbox changes — update _values, not just DOM.
    // Use a Set to avoid duplicates (e.g. re-firing a change on an already-selected item).
    options.addEventListener("change", (e) => {
      const checkbox = e.target.closest('input[type="checkbox"]');
      if (!checkbox) return;
      checkbox
        .closest(".multiselect-option")
        ?.classList.toggle("selected", checkbox.checked);

      const current = new Set(_values[filterName] || []);
      if (checkbox.checked) {
        current.add(checkbox.value);
      } else {
        current.delete(checkbox.value);
      }
      if (current.size > 0) {
        _values[filterName] = [...current];
      } else {
        delete _values[filterName];
      }
      _updateTrigger(filterName);
    });

    container.appendChild(trigger);
    return container;
  }

  // ---- Create facility autocomplete ----

  function _createFacilityAutocomplete(filterName, filterDef, reportMetadata) {
    const container = document.createElement("div");
    container.className = "facility-autocomplete-container";
    container.setAttribute("data-filter", filterName);

    const input = document.createElement("input");
    input.type = "text";
    input.className = "facility-autocomplete-input";
    input.id = `filter-${filterName}-input`;
    input.placeholder = "Search…";
    input.title = "Search facilities";
    input.autocomplete = "off";

    const selectedContainer = document.createElement("div");
    selectedContainer.className = "facility-selected-tags";
    selectedContainer.id = `filter-${filterName}-selected`;

    // Dropdown appended to body (like multiselect panels) to avoid
    // overflow clipping from .filter-bar-content's overflow-x: auto
    const dropdown = document.createElement("div");
    dropdown.className = "facility-autocomplete-dropdown";
    dropdown.id = `filter-${filterName}-dropdown`;
    document.body.appendChild(dropdown);
    _multiselectPanels.push(dropdown); // track for cleanup in destroy()

    let allFacilities = [];
    let debounceTimer = null;

    const positionDropdown = () => {
      const rect = input.getBoundingClientRect();
      dropdown.style.top = `${rect.bottom + 4}px`;
      dropdown.style.left = `${rect.left}px`;
      dropdown.style.width = `${Math.max(rect.width, 280)}px`;
    };

    const showSuggestions = (query) => {
      dropdown.innerHTML = "";
      const lowerQuery = query.toLowerCase();
      const filtered = allFacilities
        .filter((f) => f.toLowerCase().includes(lowerQuery))
        .slice(0, 20);

      if (filtered.length === 0) {
        const noResults = document.createElement("div");
        noResults.className = "facility-autocomplete-no-results";
        noResults.textContent = query
          ? "No matches found"
          : "Type to search...";
        dropdown.appendChild(noResults);
      } else {
        filtered.forEach((facility) => {
          const item = document.createElement("div");
          item.className = "facility-autocomplete-item";
          item.textContent = facility;
          item.addEventListener("click", (e) => {
            e.stopPropagation();
            addFacilityTag(facility);
            input.value = "";
            dropdown.classList.remove("open");
          });
          dropdown.appendChild(item);
        });
      }
      positionDropdown();
      dropdown.classList.add("open");
    };

    const addFacilityTag = (facility) => {
      const current = new Set(_values[filterName] || []);
      if (current.has(facility)) return;

      current.add(facility);
      _values[filterName] = [...current];

      const tag = document.createElement("span");
      tag.className = "facility-tag";
      tag.setAttribute("data-value", facility);
      tag.innerHTML = `${_escapeHtml(facility)} <button type="button" class="facility-tag-remove">&times;</button>`;

      tag
        .querySelector(".facility-tag-remove")
        .addEventListener("click", (e) => {
          e.stopPropagation();
          const updated = (_values[filterName] || []).filter(
            (v) => v !== facility,
          );
          if (updated.length > 0) {
            _values[filterName] = updated;
          } else {
            delete _values[filterName];
          }
          tag.remove();
          _updatePills();
        });

      selectedContainer.appendChild(tag);
      _updatePills();
    };

    input.addEventListener("input", () => {
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        showSuggestions(input.value);
      }, 150);
    });

    input.addEventListener("focus", () => {
      if (allFacilities.length > 0) {
        showSuggestions(input.value);
      }
    });

    // Close dropdown when clicking outside — uses AbortController
    document.addEventListener(
      "click",
      (e) => {
        if (!container.contains(e.target) && !dropdown.contains(e.target)) {
          dropdown.classList.remove("open");
        }
      },
      { signal: _abortController.signal },
    );

    // Prevent clicks inside dropdown from closing it
    dropdown.addEventListener("click", (e) => e.stopPropagation());

    container._loadFacilities = async (facilities) => {
      allFacilities = facilities || [];
    };

    container._addFacilityTag = addFacilityTag;

    container.appendChild(selectedContainer);
    container.appendChild(input);
    // dropdown is on document.body, not inside container

    return container;
  }

  // ---- Load filter options ----

  async function _loadFilterOptions(filterName, filterDef, parentValues) {
    try {
      let url = filterDef.apiEndpoint;
      if (url.startsWith("/api")) {
        url = (window.BASE_PATH || "") + url;
      }

      if (filterDef.parentParams && filterDef.parentParams.length > 0) {
        const params = new URLSearchParams();
        for (const parentParam of filterDef.parentParams) {
          const value = parentValues[parentParam];
          if (value) {
            params.append(parentParam, value);
          }
        }
        if (params.toString()) {
          const separator = url.includes("?") ? "&" : "?";
          url += `${separator}${params.toString()}`;
        }
      }

      const data = await apiCache.fetch(url);

      let options;
      if (Array.isArray(data)) {
        options = data;
      } else if (data && Array.isArray(data.values)) {
        options = data.values;
      } else {
        console.warn(`Unexpected response format for ${filterName}:`, data);
        options = [];
      }

      if (data && data.truncated) {
        console.warn(
          `Filter ${filterName} has ${data.total} values, showing first 20`,
        );
      }

      if (filterName === "month" && parentValues && parentValues.quarter) {
        const selectedQuarters = parentValues.quarter
          .split(",")
          .map((v) => parseInt(v.trim(), 10))
          .filter((v) => !isNaN(v) && v >= 1 && v <= 4);
        if (selectedQuarters.length > 0) {
          const allowedMonths = new Set();
          selectedQuarters.forEach((q) => {
            const start = (q - 1) * 3 + 1;
            allowedMonths.add(String(start));
            allowedMonths.add(String(start + 1));
            allowedMonths.add(String(start + 2));
          });
          options = options.filter((opt) => allowedMonths.has(String(opt)));
        }
      }

      console.log(
        `loadFilterOptions: ${filterName} got ${options?.length || 0} options from ${url}`,
      );

      const isMulti = _isMultiselect(filterName, filterDef);
      const isCustomMulti =
        filterDef.type === "multiselect" &&
        !BUILTIN_MULTISELECT.has(filterName) &&
        filterName !== "facility";

      if (isMulti) {
        const optionsContainer = document.getElementById(
          `filter-${filterName}-options`,
        );
        console.log(
          `loadFilterOptions: ${filterName} optionsContainer exists: ${!!optionsContainer}`,
        );
        if (!optionsContainer) return;

        optionsContainer.innerHTML = "";
        console.log(
          `loadFilterOptions: ${filterName} creating ${options.length} checkboxes`,
        );

        options.forEach((option, index) => {
          const optionDiv = document.createElement("div");
          optionDiv.className = "multiselect-option";

          const checkbox = document.createElement("input");
          checkbox.type = "checkbox";
          checkbox.value = String(option);
          checkbox.id = `${filterName}-opt-${index}`;

          const label = document.createElement("label");
          label.setAttribute("for", checkbox.id);
          label.textContent = _formatOption(filterName, option);

          // Checkbox change events are handled by the delegated listener on
          // the options container set up in _createMultiselectDropdown

          optionDiv.appendChild(checkbox);
          optionDiv.appendChild(label);
          optionsContainer.appendChild(optionDiv);
        });

        console.log(
          `loadFilterOptions: ${filterName} optionsContainer now has ${optionsContainer.children.length} children`,
        );

        // Track all available options (used by "All" button and trigger count)
        _optionValues[filterName] = options.map(String);

        // Preserve any still-valid previous selection(s) first.
        const optionSet = new Set(_optionValues[filterName]);
        const existingValid = (_values[filterName] || []).filter((v) =>
          optionSet.has(v),
        );

        // Leave time filters explicit (do not auto-select first option).
        const skipAutoSelect =
          filterName === "region" ||
          filterName === "district" ||
          filterName === "month" ||
          filterName === "quarter" ||
          filterName === "week" ||
          isCustomMulti;

        if (existingValid.length > 0) {
          _values[filterName] = existingValid;
        } else if (options.length > 0 && !skipAutoSelect) {
          _values[filterName] = [String(options[0])];
        } else {
          delete _values[filterName];
        }

        // Sync DOM checkboxes to reflect _values
        _syncCheckboxesToValues(filterName);
      } else if (filterName === "facility") {
        const container = document.querySelector(
          `.facility-autocomplete-container[data-filter="${filterName}"]`,
        );
        if (container && container._loadFacilities) {
          container._loadFacilities(options);
        }
      } else {
        const selectElement = document.getElementById(`filter-${filterName}`);
        if (!selectElement) return;

        const isCustomSingle = _isCustomSingleSelect(filterName, filterDef);
        const previousValue = selectElement.value;
        const optionValues = options.map((option) => String(option));

        selectElement.innerHTML = "";
        if (!isCustomSingle && !filterDef?.hideNone) {
          const blankOption = document.createElement("option");
          blankOption.value = "";
          blankOption.textContent = "All";
          selectElement.appendChild(blankOption);
        } else if (
          isCustomSingle &&
          filterDef.noDefault &&
          !filterDef?.hideNone
        ) {
          const blankOption = document.createElement("option");
          blankOption.value = "";
          blankOption.textContent = "All";
          selectElement.appendChild(blankOption);
        }
        options.forEach((option) => {
          const optElement = document.createElement("option");
          optElement.value = String(option);
          optElement.textContent = _formatOption(filterName, option);
          selectElement.appendChild(optElement);
        });

        if (isCustomSingle) {
          if (previousValue && optionValues.includes(previousValue)) {
            selectElement.value = previousValue;
          } else {
            selectElement.value = _defaultSingleSelectValue(
              filterName,
              filterDef,
              optionValues,
            );
          }
        }
      }
    } catch (error) {
      console.error(`Error loading ${filterName}:`, error);
    }
  }

  // ---- Get current filter values ----
  // Multiselects read from _values (no DOM queries).
  // Single-select <select> elements and facility tags read from DOM —
  // those are standard form elements where DOM state is the value.

  function _getValues() {
    if (!_metadata || !_metadata.filterDefinitions) return {};

    const values = {};
    const orderedNames = _sortFiltersForDisplay(_metadata.filters || []).filter(
      (name) => _metadata.filterDefinitions[name],
    );
    for (const filterName of orderedNames) {
      const filterDef = _metadata.filterDefinitions[filterName];

      // In FY mode, skip year and month — replaced by yearmonths param
      if (_fyMode && (filterName === "year" || filterName === "month"))
        continue;

      if (_isMultiselect(filterName, filterDef)) {
        const selected = _values[filterName];
        if (selected && selected.length > 0) {
          values[filterName] = selected.join(",");
        }
        continue;
      }

      if (filterName === "facility") {
        const selected = _values[filterName];
        if (selected && selected.length > 0) {
          values[filterName] = selected.join(",");
        }
        continue;
      }

      const selectElement = document.getElementById(`filter-${filterName}`);
      if (selectElement && selectElement.value) {
        // Use paramName as URL key so facility_select sends "facility=" to the backend
        const outputKey = filterDef?.paramName || filterName;
        values[outputKey] = selectElement.value;
      }
    }

    // In FY mode, add yearmonths param
    if (_fyMode && _fyValue) {
      values["yearmonths"] = _fyToYearMonths(_fyValue);
    }

    return values;
  }

  // ---- Save filter state ----

  function _saveState() {
    if (!_metadata || !_reportId) return;

    try {
      localStorage.setItem(
        `filterState-${_reportId}`,
        JSON.stringify({
          reportId: _reportId,
          filters: _getValues(),
          fyMode: _fyMode,
          fyValue: _fyValue,
          savedAt: new Date().toISOString(),
        }),
      );
      console.log("Filter state saved");
    } catch (error) {
      console.warn("Failed to save filter state to localStorage:", error);
    }
  }

  // ---- Restore filter state ----

  function _restoreState() {
    try {
      const saved = localStorage.getItem(`filterState-${_reportId}`);
      if (!saved) return false;

      const filterState = JSON.parse(saved);
      if (filterState.reportId !== _reportId) return false;

      console.log("Restoring filter state:", filterState.filters);

      for (const [filterName, filterValue] of Object.entries(
        filterState.filters,
      )) {
        const filterDef = _metadata?.filterDefinitions?.[filterName];
        if (_isMultiselect(filterName, filterDef)) {
          // Set _values and sync DOM in one step
          _values[filterName] = filterValue.split(",").filter(Boolean);
          _syncCheckboxesToValues(filterName);
        } else if (filterName === "facility") {
          const container = document.querySelector(
            '.facility-autocomplete-container[data-filter="facility"]',
          );
          if (container?._addFacilityTag) {
            for (const facility of filterValue.split(",").filter(Boolean)) {
              container._addFacilityTag(facility);
            }
          }
        } else {
          // If filterName not found directly (e.g. state saved as "facility" but
          // report uses "facility_select" alias), look up by paramName.
          let selectElement = document.getElementById(`filter-${filterName}`);
          if (!selectElement) {
            const aliasName = (_metadata?.filters || []).find((fn) => {
              const fd = _metadata?.filterDefinitions?.[fn];
              return fd?.paramName === filterName;
            });
            if (aliasName)
              selectElement = document.getElementById(`filter-${aliasName}`);
          }
          if (
            selectElement &&
            selectElement.querySelector(`option[value="${filterValue}"]`)
          ) {
            selectElement.value = filterValue;
          }
        }
      }

      _updatePills();
      _updateDependentStates();
      return true;
    } catch (error) {
      console.warn("Failed to restore filter state from localStorage:", error);
      return false;
    }
  }

  // ---- Clear filter state ----

  function _clearState() {
    try {
      localStorage.removeItem(`filterState-${_reportId}`);
      console.log("Filter state cleared");
    } catch (error) {
      console.warn("Failed to clear filter state:", error);
    }
  }

  // ---- Build filter bar ----

  async function _buildFilterBar() {
    const filterBar = document.getElementById("filter-bar");
    const activeFiltersEl = document.getElementById("active-filters");

    if (!_metadata || !_metadata.filters || _metadata.filters.length === 0) {
      filterBar.style.display = "none";
      return;
    }

    activeFiltersEl.innerHTML = "";

    // Clean up orphaned multiselect panels from previous filter bar
    // (panels not tracked by this instance, e.g. from a page that didn't call destroy)
    document.querySelectorAll(".multiselect-panel").forEach((panel) => {
      if (!_multiselectPanels.includes(panel)) {
        panel.remove();
      }
    });

    const displayFilters = _sortFiltersForDisplay(_metadata.filters);
    for (const filterName of displayFilters) {
      const filterDef = _metadata.filterDefinitions[filterName];
      if (!filterDef) continue;

      const filterItem = document.createElement("div");
      filterItem.className = "filter-item";
      filterItem.setAttribute("data-filter", filterName);

      const label = document.createElement("span");
      label.className = "filter-item-label";
      label.textContent = _shortFilterLabel(filterDef, filterName) + ":";
      filterItem.appendChild(label);

      const isMulti = _isMultiselect(filterName, filterDef);

      if (isMulti) {
        const multiselectContainer = _createMultiselectDropdown(
          filterName,
          filterDef,
        );
        filterItem.appendChild(multiselectContainer);

        // Setup cascading listeners for dependent multiselect filters
        if (filterDef.dependsOn && filterDef.dependsOn.length > 0) {
          const checkAllParentsHaveValues = () => {
            for (const parentName of filterDef.dependsOn) {
              if (!_hasFilterSelection(parentName)) {
                return { allSet: false, missingParent: parentName };
              }
            }
            return { allSet: true, missingParent: null };
          };

          for (const parentFilter of filterDef.dependsOn) {
            const parentSelect = document.getElementById(
              `filter-${parentFilter}`,
            );
            if (parentSelect) {
              parentSelect.addEventListener("change", async () => {
                const trig = document.getElementById(
                  `filter-${filterName}-trigger`,
                );
                if (trig) {
                  const { allSet, missingParent: mp } =
                    checkAllParentsHaveValues();
                  trig.disabled = !allSet;
                  trig.style.opacity = allSet ? "1" : "0.5";
                  trig.style.cursor = allSet ? "pointer" : "not-allowed";

                  if (!allSet) {
                    delete _values[filterName];
                    _syncCheckboxesToValues(filterName);
                    trig.textContent = _dependentParentHint(mp);
                    _updateFilterSummary();
                  }
                }
                const currentValues = _getValues();
                await _loadFilterOptions(filterName, filterDef, currentValues);
              });
            }
          }
        }
      } else if (filterName === "facility") {
        const autocompleteContainer = _createFacilityAutocomplete(
          filterName,
          filterDef,
          _metadata,
        );
        filterItem.appendChild(autocompleteContainer);
      } else {
        const select = document.createElement("select");
        select.className = "filter-item-select";
        select.setAttribute("id", `filter-${filterName}`);

        const defaultOption = document.createElement("option");
        defaultOption.value = "";
        defaultOption.textContent = _isCustomSingleSelect(filterName, filterDef)
          ? "Loading..."
          : "All";
        select.appendChild(defaultOption);

        filterItem.appendChild(select);

        select.addEventListener("change", async () => {
          _updatePills();
          _saveState();
          const currentValues = _getValues();

          if (filterDef.cascadesTo && filterDef.cascadesTo.length > 0) {
            for (const dependentFilter of filterDef.cascadesTo) {
              const dependentDef = _metadata.filterDefinitions[dependentFilter];
              if (dependentDef) {
                await _loadFilterOptions(
                  dependentFilter,
                  dependentDef,
                  currentValues,
                );
                if (!_isMultiselect(dependentFilter, dependentDef)) {
                  _resetSingleSelectValue(dependentFilter, dependentDef);
                }
              }
            }
          }
        });
      }

      activeFiltersEl.appendChild(filterItem);

      const currentFilterValues = _getValues();
      await _loadFilterOptions(filterName, filterDef, currentFilterValues);

      // After year filter is loaded and options are populated, add FY toggle
      if (filterName === "year" && _isFYEligible()) {
        _addFYToggle(filterItem);
      }
    }

    _updateDependentStates();
    filterBar.style.display = "flex";
  }

  // ---- Set filter values from state object ----

  function _setValues(filters) {
    for (const [filterName, filterValue] of Object.entries(filters)) {
      const filterDef = _metadata?.filterDefinitions?.[filterName];
      if (_isMultiselect(filterName, filterDef)) {
        _values[filterName] = filterValue.split(",").filter(Boolean);
        _syncCheckboxesToValues(filterName);
      } else if (filterName === "facility") {
        const container = document.querySelector(
          '.facility-autocomplete-container[data-filter="facility"]',
        );
        if (container?._addFacilityTag) {
          for (const facility of filterValue.split(",").filter(Boolean)) {
            container._addFacilityTag(facility);
          }
        }
      } else {
        // If filterName not found directly (e.g. URL has "facility" but report uses
        // "facility_select" alias with paramName:"facility"), look up by paramName.
        let actualName = filterName;
        if (!_metadata?.filterDefinitions?.[filterName]) {
          const aliasName = (_metadata?.filters || []).find((fn) => {
            const fd = _metadata?.filterDefinitions?.[fn];
            return fd?.paramName === filterName;
          });
          if (aliasName) actualName = aliasName;
        }
        const select = document.getElementById(`filter-${actualName}`);
        const actualDef = _metadata?.filterDefinitions?.[actualName];
        if (select) {
          select.value = filterValue;
          if (
            _isCustomSingleSelect(actualName, actualDef) &&
            select.value !== filterValue
          ) {
            _resetSingleSelectValue(actualName, actualDef);
          }
        }
      }
    }

    if (_values.week && _values.week.length > 0) {
      _autoFillTimeFromSelection("week");
    }
    if (_values.month && _values.month.length > 0) {
      _autoFillTimeFromSelection("month");
    }

    _updatePills();
    _updateDependentStates();
  }

  // ---- Apply smart time defaults ----

  function _applySmartDefaults() {
    if (!_metadata || !_metadata.filters) return;

    // In FY mode, smart defaults are handled by _addFYToggle
    if (_fyMode) return;

    const filters = _metadata.filters;

    if (filters.includes("week")) {
      const defaultWeek = _getSmartDefaultWeek();
      _values["week"] = [defaultWeek];
      _syncCheckboxesToValues("week");

      if (filters.includes("year") && defaultWeek.length >= 4) {
        const yearSelect = document.getElementById("filter-year");
        if (yearSelect) yearSelect.value = defaultWeek.substring(0, 4);
      }
    } else if (filters.includes("month")) {
      const { year, month } = _getSmartDefaultMonth();
      _values["month"] = [String(month)];
      _syncCheckboxesToValues("month");

      if (filters.includes("year")) {
        const yearSelect = document.getElementById("filter-year");
        if (yearSelect) yearSelect.value = String(year);
      }
    } else if (filters.includes("quarter")) {
      const { year, quarter } = _getSmartDefaultQuarter();

      if (!_metadata.disableSmartDefaults) {
        _values["quarter"] = [String(quarter)];
        _syncCheckboxesToValues("quarter");
      }

      if (filters.includes("year")) {
        const yearSelect = document.getElementById("filter-year");
        if (yearSelect) yearSelect.value = String(year);
      }
    } else if (filters.includes("year")) {
      // Year-only report (no month/week/quarter): default to "All" so the
      // report shows all available data rather than guessing a year that may be empty.
      const yearSelect = document.getElementById("filter-year");
      if (yearSelect) yearSelect.value = "";
    }

    _updateDependentStates();
    _updatePills();
  }

  // ---- Remove a single filter value ----

  function _removeFilter(filterName, value) {
    const filterDef = _metadata?.filterDefinitions?.[filterName];
    if (_isMultiselect(filterName, filterDef)) {
      const updated = (_values[filterName] || []).filter((v) => v !== value);
      if (updated.length > 0) {
        _values[filterName] = updated;
      } else {
        delete _values[filterName];
      }
      _syncCheckboxesToValues(filterName);
    } else if (filterName === "facility") {
      const updated = (_values[filterName] || []).filter((v) => v !== value);
      if (updated.length > 0) {
        _values[filterName] = updated;
      } else {
        delete _values[filterName];
      }
      const facilitySelected = document.getElementById(
        "filter-facility-selected",
      );
      facilitySelected
        ?.querySelector(`[data-value="${CSS.escape(value)}"]`)
        ?.remove();
    } else {
      _resetSingleSelectValue(filterName, filterDef);
    }
    loadReport();
  }

  // ---- Clear all filters ----

  async function _clearAll() {
    // Reset FY mode
    _fyMode = false;
    _fyValue = null;
    const fySelect = document.getElementById("filter-fy");
    const yearSelect = document.getElementById("filter-year");
    if (fySelect) fySelect.style.display = "none";
    if (yearSelect) yearSelect.style.display = "";
    const fyToggle = document.querySelector(".fy-toggle");
    if (fyToggle) {
      fyToggle.querySelectorAll(".fy-toggle-opt").forEach((opt) => {
        opt.classList.toggle("active", opt.dataset.mode === "cy");
      });
    }
    const monthItem = document.querySelector(
      '.filter-item[data-filter="month"]',
    );
    if (monthItem) monthItem.style.display = "";
    const yearLabel = document.querySelector(
      '.filter-item[data-filter="year"] .filter-item-label',
    );
    if (yearLabel) {
      const yDef = _metadata?.filterDefinitions?.year;
      yearLabel.textContent = _shortFilterLabel(yDef, "year") + ":";
    }

    for (const filterName of _metadata?.filters || []) {
      const filterDef = _metadata?.filterDefinitions?.[filterName];
      if (_isMultiselect(filterName, filterDef)) {
        delete _values[filterName];
        _syncCheckboxesToValues(filterName);
      } else if (filterName === "facility") {
        delete _values["facility"];
        const facilitySelected = document.getElementById(
          "filter-facility-selected",
        );
        if (facilitySelected) facilitySelected.innerHTML = "";
      } else {
        _resetSingleSelectValue(filterName, filterDef);
      }
    }

    _updatePills();
    _clearState();
  }

  // ---- Apply button loading state ----

  function _setApplyLoading(isLoading) {
    const applyBtn = document.getElementById("apply-filters-btn");
    if (!applyBtn) return;

    if (isLoading) {
      applyBtn.disabled = true;
      applyBtn.dataset.originalText = applyBtn.textContent;
      applyBtn.textContent = "Loading...";
      applyBtn.classList.add("loading");
    } else {
      applyBtn.disabled = false;
      applyBtn.textContent = applyBtn.dataset.originalText || "Apply";
      applyBtn.classList.remove("loading");
    }
  }

  // ---- Show filter feedback banner ----

  function _showFeedback() {
    const banner = document.getElementById("filter-feedback-banner");
    banner.style.display = "block";
    setTimeout(() => {
      banner.style.display = "none";
    }, 2000);
  }

  // ---- Document-level listener for closing multiselect panels on outside click ----

  function _setupDocumentCloseListener() {
    document.addEventListener(
      "click",
      (e) => {
        for (const panel of _multiselectPanels) {
          if (!panel.classList.contains("open")) continue;
          const trigger = document.getElementById(
            panel.id.replace("-panel", "-trigger"),
          );
          if (trigger && !panel.contains(e.target) && e.target !== trigger) {
            panel.classList.remove("open");
          }
        }
      },
      { signal: _abortController.signal },
    );
  }

  // ======== Public API ========

  function init(metadata, reportId) {
    // Clean up previous session
    destroy();

    _metadata = metadata;
    _reportId = reportId;
    _abortController = new AbortController();
    _multiselectPanels = [];
    _values = {};
    _optionValues = {};
    _fyMode = false;
    _fyValue = null;

    // Set up document-level close listener for multiselect panels
    _setupDocumentCloseListener();
  }

  function destroy() {
    if (_abortController) {
      _abortController.abort();
    }
    _multiselectPanels.forEach((p) => p.remove());
    _multiselectPanels = [];
    const filterBar = document.getElementById("filter-bar");
    if (filterBar) filterBar.style.display = "none";
    _metadata = null;
    _reportId = null;
    _abortController = null;
    _values = {};
    _optionValues = {};
    _fyMode = false;
    _fyValue = null;
  }

  return {
    init: init,
    destroy: destroy,
    buildFilterBar: _buildFilterBar,
    getValues: _getValues,
    setValues: _setValues,
    applySmartDefaults: _applySmartDefaults,
    saveState: _saveState,
    restoreState: _restoreState,
    clearState: _clearState,
    updatePills: _updatePills,
    removeFilter: _removeFilter,
    clearAll: _clearAll,
    setApplyLoading: _setApplyLoading,
    showFeedback: _showFeedback,
    isMultiselect: _isMultiselect,
    formatOption: _formatOption,
    getMetadata: function () {
      return _metadata;
    },
    initMobileToggle: _initMobileToggle,
  };
})();
