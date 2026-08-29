/
function geoFeatureName(feature) {
  if (!feature || !feature.properties) return '';
  const p = feature.properties;
  return (
    p.name || p.NAME || p.admin || p.ADMIN || p.admin_name ||
    p.COUNTRY || p.Country || p.sovereignt || p.SOVEREIGNT ||
    (feature.id ? String(feature.id) : '')
  ) || '';
}

// Resolve the display name for a choropleth feature (country or district).
function styleFeatureName(feature) {
  return geoFeatureName(feature);
}/ API base URL - uses BASE_PATH from server for subpath deployments
const ASSET_BASE = window.BASE_PATH || "";

// Global chart instances to prevent memory leaks
const chartInstances = {};

// GeoJSON cache: keyed by path, stores the fetch Promise so multiple maps share one parse
const _geojsonCache = {};
window._geojsonCache = _geojsonCache;
function fetchGeoJSON(path) {
  if (!path || typeof path !== "string") {
    return Promise.reject(new Error("GeoJSON path is missing"));
  }
  if (!_geojsonCache[path]) {
    _geojsonCache[path] = fetch(ASSET_BASE + path)
      .then((r) => {
        if (!r.ok) {
          throw new Error(`GeoJSON ${path}: HTTP ${r.status}`);
        }
        return r.json();
      })
      .catch((err) => {
        delete _geojsonCache[path];
        throw err;
      });
  }
  return _geojsonCache[path];
}

// Generation counter: incremented on cleanup so pending rAF callbacks can detect stale renders
let _renderGeneration = 0;

// =============================================================================
// PUBLICATION-QUALITY CHART PALETTE
// Professional institutional blue palette for health/government reports
// =============================================================================
function getChartPalette() {
  return {
    // Primary blue progression (dark to light)
    blues: ["#1e3a5f", "#2d5a87", "#4a7c9b", "#7ba3be", "#a8c5d8"],
    // Accent colors for contrast/highlights
    accents: ["#c17f59", "#5a7d5a", "#8b4d4d"],
    // Chart styling
    border: "#1a1a1a",
    stackedBorder: "#9ca3af",
    borderLight: "#333333",
    grid: "#e5e5e5",
    axis: "#1a1a1a",
    background: "#ffffff",
    text: "#1a1a1a",
    textSecondary: "#666666",
    labelBg: "rgba(255, 255, 255, 0.85)",
  };
}

// Backward-compatible constant — used at chart render time so always fresh
const CHART_PALETTE = new Proxy(
  {},
  {
    get(_, prop) {
      return getChartPalette()[prop];
    },
  },
);

function getBarBorderColor(isStacked) {
  return isStacked ? "transparent" : CHART_PALETTE.border;
}

// =============================================================================
// COLORBLIND-ACCESSIBLE PATTERNS
// Subtle patterns to help distinguish datasets without relying on color alone
// Uses canvas pattern API - patterns are very subtle (10% opacity lines)
// =============================================================================
const CHART_PATTERNS = {
  /**
   * Create a subtle diagonal line pattern
   * @param {string} baseColor - The base fill color
   * @param {CanvasRenderingContext2D} ctx - Canvas context for pattern creation
   * @returns {CanvasPattern|string} Pattern or original color if creation fails
   */
  diagonal: function (baseColor, ctx) {
    try {
      const patternCanvas = document.createElement("canvas");
      patternCanvas.width = 10;
      patternCanvas.height = 10;
      const pCtx = patternCanvas.getContext("2d");

      // Fill with base color
      pCtx.fillStyle = baseColor;
      pCtx.fillRect(0, 0, 10, 10);

      // Add subtle diagonal line (10% opacity)
      pCtx.strokeStyle = "rgba(255, 255, 255, 0.15)";
      pCtx.lineWidth = 1;
      pCtx.beginPath();
      pCtx.moveTo(0, 10);
      pCtx.lineTo(10, 0);
      pCtx.stroke();

      return ctx.createPattern(patternCanvas, "repeat");
    } catch (e) {
      return baseColor;
    }
  },

  /**
   * Create a subtle dot pattern
   */
  dots: function (baseColor, ctx) {
    try {
      const patternCanvas = document.createElement("canvas");
      patternCanvas.width = 8;
      patternCanvas.height = 8;
      const pCtx = patternCanvas.getContext("2d");

      pCtx.fillStyle = baseColor;
      pCtx.fillRect(0, 0, 8, 8);

      // Add subtle dot (15% opacity)
      pCtx.fillStyle = "rgba(255, 255, 255, 0.2)";
      pCtx.beginPath();
      pCtx.arc(4, 4, 1.5, 0, Math.PI * 2);
      pCtx.fill();

      return ctx.createPattern(patternCanvas, "repeat");
    } catch (e) {
      return baseColor;
    }
  },

  /**
   * Create a subtle horizontal line pattern
   */
  horizontal: function (baseColor, ctx) {
    try {
      const patternCanvas = document.createElement("canvas");
      patternCanvas.width = 6;
      patternCanvas.height = 6;
      const pCtx = patternCanvas.getContext("2d");

      pCtx.fillStyle = baseColor;
      pCtx.fillRect(0, 0, 6, 6);

      // Add subtle horizontal line
      pCtx.strokeStyle = "rgba(255, 255, 255, 0.15)";
      pCtx.lineWidth = 1;
      pCtx.beginPath();
      pCtx.moveTo(0, 3);
      pCtx.lineTo(6, 3);
      pCtx.stroke();

      return ctx.createPattern(patternCanvas, "repeat");
    } catch (e) {
      return baseColor;
    }
  },
};

/**
 * Get pattern for a dataset index (cycles through: solid, diagonal, dots, horizontal)
 * @param {string} baseColor - The base color
 * @param {number} index - Dataset index
 * @param {CanvasRenderingContext2D} ctx - Canvas context
 * @returns {CanvasPattern|string} Pattern or solid color
 */
function getPatternForIndex(baseColor, index, ctx) {
  const patternTypes = [null, "diagonal", "dots", "horizontal"];
  const patternType = patternTypes[index % patternTypes.length];

  if (!patternType || !ctx) {
    return baseColor; // First dataset is always solid
  }

  return CHART_PATTERNS[patternType](baseColor, ctx);
}

/**
 * Get color for a dataset - respects YAML-specified colors, falls back to palette
 * @param {Object} dataset - The dataset object from component data
 * @param {number} index - Dataset index for palette cycling
 * @param {string} type - 'primary' for blues, 'accent' for accent colors
 * @returns {string} Hex color code
 */
function getDatasetColor(dataset, index, type = "primary") {
  // If user specified a color in YAML, respect it
  if (dataset && dataset.color) {
    return dataset.color;
  }
  // Fall back to palette (cycle if more datasets than colors)
  const palette =
    type === "accent" ? CHART_PALETTE.accents : CHART_PALETTE.blues;
  return palette[index % palette.length];
}

// =============================================================================
// ACCESSIBILITY HELPERS
// ARIA labels and screen reader support for charts
// =============================================================================

/**
 * Generate an accessible description for a chart (used in aria-label)
 * Summarizes the data so screen readers can announce meaningful content
 * @param {Object} component - The chart component
 * @returns {string} Human-readable description of the chart data
 */
function generateChartAriaLabel(component) {
  const title = component.title || "Chart";
  const data = component.data || {};
  const datasets = data.datasets || [];
  const labels = data.labels || [];

  if (datasets.length === 0 || labels.length === 0) {
    return `${title}. No data available.`;
  }

  // Build a concise summary
  const parts = [title];

  // Add period/category info
  if (labels.length > 0) {
    if (labels.length <= 3) {
      parts.push(`showing ${labels.join(", ")}`);
    } else {
      parts.push(
        `showing ${labels.length} periods from ${labels[0]} to ${labels[labels.length - 1]}`,
      );
    }
  }

  // Add dataset summaries (max 3 to keep it brief)
  const summaries = datasets
    .slice(0, 3)
    .map((ds) => {
      const values = ds.data || [];
      if (values.length === 0) return null;

      const numericValues = values.filter(
        (v) => typeof v === "number" && !isNaN(v),
      );
      if (numericValues.length === 0) return `${ds.label}: no numeric data`;

      const total = numericValues.reduce((a, b) => a + b, 0);
      const avg = Math.round(total / numericValues.length);
      return `${ds.label}: average ${formatNumber(avg)}`;
    })
    .filter(Boolean);

  if (summaries.length > 0) {
    parts.push(summaries.join("; "));
  }

  return parts.join(". ") + ".";
}

/**
 * Create a consistent "no data" message element with proper ARIA attributes
 * role="alert" ensures screen readers announce this immediately
 * @param {string} title - Optional chart title for context
 * @returns {HTMLElement} The no-data message element
 */
function createNoDataMessage(title) {
  const noData = document.createElement("div");
  noData.setAttribute("role", "alert");
  noData.setAttribute("aria-live", "polite");
  noData.className = "no-data-message";
  noData.innerHTML = `
        <p style="margin:0 0 4px 0;font-weight:500;">No data available</p>
        <p style="margin:0;font-size:0.9em;opacity:0.7;">Try adjusting the filters</p>
    `;
  noData.style.cssText =
    "text-align: center; padding: 40px; color: var(--text-secondary);";
  return noData;
}

/**
 * Set up accessibility attributes on a chart canvas
 * @param {HTMLCanvasElement} canvas - The canvas element
 * @param {Object} component - The chart component for label generation
 */
function setChartAccessibility(canvas, component) {
  canvas.setAttribute("role", "img");
  canvas.setAttribute("aria-label", generateChartAriaLabel(component));
}

/**
 * Create a Chart.js plugin for drawing horizontal reference lines
 * @param {Array} referenceLines - Array of reference line configs from component.data.referenceLines
 * @param {boolean} isHorizontal - Whether chart has horizontal bars (swaps x/y for line position)
 * @returns {Object} Chart.js plugin object
 */
function createReferenceLinesPlugin(referenceLines, isHorizontal = false) {
  if (!referenceLines || referenceLines.length === 0) {
    return null;
  }

  // Default colors: first line green (target), others gray (benchmark)
  const defaultColors = ["#22c55e", "#9ca3af", "#6b7280"];

  return {
    id: "referenceLines",
    afterDatasetsDraw: function (chart) {
      const ctx = chart.ctx;
      const xScale = chart.scales.x;
      const chartArea = chart.chartArea;

      referenceLines.forEach((line, index) => {
        const value = line.value;
        const label = line.label || "";
        const color = line.color || defaultColors[index % defaultColors.length];
        const style = line.style || "dashed";

        // Resolve which y-axis scale to use: "y1" (right) or "y" (left, default).
        // Falls back to "y" when the requested scale is not present in the chart.
        const axisId = line.axis === "y1" && chart.scales.y1 ? "y1" : "y";
        const yScale = chart.scales[axisId];

        // Calculate Y position for the line
        const yPos = isHorizontal
          ? xScale.getPixelForValue(value)
          : yScale.getPixelForValue(value);

        // Skip if line is outside chart area
        if (isHorizontal) {
          if (yPos < chartArea.left || yPos > chartArea.right) return;
        } else {
          if (yPos < chartArea.top || yPos > chartArea.bottom) return;
        }

        ctx.save();

        // Set line style
        ctx.strokeStyle = color;
        ctx.lineWidth = 1.5;

        if (style === "dotted") {
          ctx.setLineDash([2, 4]);
        } else {
          // dashed (default)
          ctx.setLineDash([6, 4]);
        }

        // Draw the horizontal line
        ctx.beginPath();
        if (isHorizontal) {
          // Vertical line for horizontal bar charts
          ctx.moveTo(yPos, chartArea.top);
          ctx.lineTo(yPos, chartArea.bottom);
        } else {
          // Horizontal line for normal charts
          ctx.moveTo(chartArea.left, yPos);
          ctx.lineTo(chartArea.right, yPos);
        }
        ctx.stroke();

        // Draw label on left edge (inside chart)
        if (label) {
          ctx.setLineDash([]); // Reset dash for text
          ctx.fillStyle = color;
          ctx.font = "600 10px 'Inter', -apple-system, sans-serif";
          ctx.textAlign = "left";
          ctx.textBaseline = "bottom";

          // Position label slightly above the line, on left edge
          const labelX = chartArea.left + 4;
          const labelY = isHorizontal ? chartArea.top + 12 : yPos - 3;

          // Draw small background for readability
          const textWidth = ctx.measureText(label).width;
          ctx.fillStyle = getChartPalette().labelBg;
          ctx.fillRect(labelX - 2, labelY - 10, textWidth + 4, 12);

          // Draw text
          ctx.fillStyle = color;
          ctx.fillText(label, labelX, labelY);
        }

        ctx.restore();
      });
    },
  };
}

/**
 * Creates a Chart.js plugin that draws linear regression trend lines for each dataset.
 * Uses least squares method to compute the best-fit line.
 * @param {Array} datasets - Chart.js dataset objects
 * @param {string} [colorOverride] - Optional color to use for all trend lines
 * @returns {Object|null} Chart.js plugin object or null if no valid datasets
 */
function createTrendLinePlugin(datasets, colorOverride) {
  if (!datasets || datasets.length === 0) return null;

  return {
    id: "trendLines",
    afterDatasetsDraw: function (chart) {
      const ctx = chart.ctx;
      const xScale = chart.scales.x;
      const yScale = chart.scales.y;

      chart.data.datasets.forEach((dataset, datasetIndex) => {
        const meta = chart.getDatasetMeta(datasetIndex);
        if (meta.hidden) return;

        // Collect non-null data points with their indices
        const points = [];
        const data = dataset.data;
        for (let i = 0; i < data.length; i++) {
          if (data[i] != null) {
            points.push({ x: i, y: Number(data[i]) });
          }
        }

        // Need at least 2 points for a trend line
        if (points.length < 2) return;

        // Least squares linear regression
        const n = points.length;
        let sumX = 0,
          sumY = 0,
          sumXY = 0,
          sumX2 = 0;
        for (const p of points) {
          sumX += p.x;
          sumY += p.y;
          sumXY += p.x * p.y;
          sumX2 += p.x * p.x;
        }
        const denom = n * sumX2 - sumX * sumX;
        if (denom === 0) return;

        const slope = (n * sumXY - sumX * sumY) / denom;
        const intercept = (sumY - slope * sumX) / n;

        // Draw trend line from first to last data index
        const firstX = points[0].x;
        const lastX = points[points.length - 1].x;
        const startPixelX = xScale.getPixelForValue(firstX);
        const startPixelY = yScale.getPixelForValue(slope * firstX + intercept);
        const endPixelX = xScale.getPixelForValue(lastX);
        const endPixelY = yScale.getPixelForValue(slope * lastX + intercept);

        const color =
          colorOverride ||
          dataset.borderColor ||
          dataset.backgroundColor ||
          "#666";
        ctx.save();
        ctx.strokeStyle = color;
        ctx.globalAlpha = 0.5;
        ctx.lineWidth = 1.5;
        ctx.setLineDash([6, 4]);
        ctx.beginPath();
        ctx.moveTo(startPixelX, startPixelY);
        ctx.lineTo(endPixelX, endPixelY);
        ctx.stroke();

        // Draw "Trend Line" label at the end of the line
        ctx.setLineDash([]);
        ctx.globalAlpha = 0.7;
        ctx.fillStyle = color;
        ctx.font = "600 10px 'Inter', -apple-system, sans-serif";
        ctx.textAlign = "right";
        ctx.textBaseline = "bottom";
        const visibleDatasets = chart.data.datasets.filter(
          (_, i) => !chart.getDatasetMeta(i).hidden,
        );
        const labelY =
          endPixelY - 4 - (visibleDatasets.length > 1 ? datasetIndex * 14 : 0);
        const labelText =
          visibleDatasets.length > 1 ? `Trend: ${dataset.label}` : "Trend Line";
        const textWidth = ctx.measureText(labelText).width;
        // Background for readability
        const bgAlpha = ctx.globalAlpha;
        ctx.globalAlpha = 0.8;
        ctx.fillStyle = getChartPalette().labelBg;
        ctx.fillRect(endPixelX - textWidth - 4, labelY - 10, textWidth + 6, 12);
        ctx.globalAlpha = bgAlpha;
        ctx.fillStyle = color;
        ctx.fillText(labelText, endPixelX - 1, labelY);

        ctx.restore();
      });
    },
  };
}

// Draw a translucent label background rectangle at (x, y), sized to text and
// honoring ctx.textAlign. Assumes textBaseline = 'middle' and ~10px label font.
// Caller is responsible for setting the final text color and calling fillText after.
function drawValueLabelBg(ctx, text, x, y, palette) {
  const textWidth = ctx.measureText(text).width;
  const padX = 3;
  const padY = 2;
  const halfH = 5 + padY;
  let bgX = x - padX;
  if (ctx.textAlign === "center") bgX = x - textWidth / 2 - padX;
  else if (ctx.textAlign === "right") bgX = x - textWidth - padX;
  ctx.fillStyle = palette.labelBg;
  ctx.fillRect(bgX, y - halfH, textWidth + padX * 2, halfH * 2);
}

/**
 * Create a Chart.js plugin for drawing value labels on bars
 * @param {boolean} isHorizontal - Whether chart has horizontal bars
 * @param {Object} [componentData] - component.data (used to read columnFormats for per-dataset decimals)
 * @returns {Object} Chart.js plugin object
 */
function createValueLabelsPlugin(isHorizontal = false, componentData = null) {
  return {
    id: "valueLabels",
    afterDatasetsDraw: function (chart) {
      const ctx = chart.ctx;
      const palette = getChartPalette();
      const valueAxis = isHorizontal ? "x" : "y";
      const isStacked = !!(
        chart.options.scales &&
        chart.options.scales[valueAxis] &&
        chart.options.scales[valueAxis].stacked
      );

      chart.data.datasets.forEach((dataset, datasetIndex) => {
        // Skip line datasets in bar_line charts
        if (dataset.type === "line") return;

        const meta = chart.getDatasetMeta(datasetIndex);
        if (!meta || meta.hidden) return;

        const decimals = getColumnDecimals(componentData, dataset.label);

        meta.data.forEach((bar, index) => {
          const value = dataset.data[index];
          if (value == null || value === 0) return;

          const label = formatNumber(value, { decimals });

          ctx.save();
          ctx.font = "600 10px 'Inter', -apple-system, sans-serif";
          ctx.textBaseline = "middle";

          let lx, ly;
          if (isStacked) {
            // Center label inside each stacked segment; skip segments too small to fit text
            if (isHorizontal) {
              const segmentWidth = Math.abs(bar.x - bar.base);
              if (segmentWidth < 16) {
                ctx.restore();
                return;
              }
              ctx.textAlign = "center";
              lx = (bar.x + bar.base) / 2;
              ly = bar.y;
            } else {
              const segmentHeight = Math.abs(bar.base - bar.y);
              if (segmentHeight < 12) {
                ctx.restore();
                return;
              }
              ctx.textAlign = "center";
              lx = bar.x;
              ly = (bar.y + bar.base) / 2;
            }
          } else if (isHorizontal) {
            // Label to the right of the bar
            ctx.textAlign = "left";
            lx = bar.x + 8;
            ly = bar.y;
          } else {
            // Label above the bar
            ctx.textAlign = "center";
            lx = bar.x;
            ly = bar.y - 10;
          }

          if (isStacked) {
            ctx.font = "700 10px 'Inter', -apple-system, sans-serif";
            ctx.fillStyle = "#000000";
          } else {
            drawValueLabelBg(ctx, label, lx, ly, palette);
            ctx.fillStyle = palette.text;
          }
          ctx.fillText(label, lx, ly);

          ctx.restore();
        });
      });
    },
  };
}

/**
 * Get publication-quality scale configuration for charts
 * @param {boolean} isHorizontal - Whether chart is horizontal (swaps x/y grid logic)
 * @param {Object} [axisLabels] - Optional x/y axis label config
 * @param {Object} [componentData] - component.data (used to resolve tick decimals from columnFormats).
 *   Axis ticks share one precision across all datasets on that axis; we use the first
 *   dataset's decimals. Multi-dataset charts with mixed precision are uncommon.
 * @param {Object} [yLimit] - Optional value-axis range {min, max}; clips data to ceiling.
 *   Maps to the value axis (x when horizontal, y otherwise) so authors think in
 *   "y-axis stops at N" regardless of chart orientation.
 * @returns {Object} Chart.js scales configuration
 */
function getPublicationScales(
  isHorizontal = false,
  axisLabels = null,
  componentData = null,
  yLimit = null,
) {
  const palette = getChartPalette();
  const axisTitleFont = {
    size: 12,
    family: "'Inter', -apple-system, sans-serif",
    weight: "500",
  };
  // Resolve value-axis decimals from the first dataset's column format.
  let axisDecimals;
  if (
    componentData &&
    componentData.datasets &&
    componentData.datasets.length > 0
  ) {
    axisDecimals = getColumnDecimals(
      componentData,
      componentData.datasets[0].label,
    );
  }
  const valueAxis = {
    beginAtZero: true,
    border: { display: false },
    grid: {
      color: palette.grid,
      lineWidth: 0.5,
      drawTicks: false,
    },
    ticks: {
      color: palette.text,
      font: { size: 11, family: "'Inter', -apple-system, sans-serif" },
      padding: 8,
      callback: function (value) {
        return formatNumber(value, { decimals: axisDecimals });
      },
    },
  };

  if (yLimit) {
    if (typeof yLimit.min === "number") valueAxis.min = yLimit.min;
    if (typeof yLimit.max === "number") valueAxis.max = yLimit.max;
  }

  // Ensure reference lines are always visible: extend scale to cover them when data falls short.
  // Use suggestedMax so data exceeding the target still shows without clipping.
  if (!yLimit || typeof yLimit.max !== "number") {
    const refLines = componentData && componentData.referenceLines;
    if (refLines && refLines.length > 0) {
      const maxRef = Math.max(...refLines.map((r) => r.value || 0));
      if (maxRef > 0) valueAxis.suggestedMax = maxRef;
    }
  }

  const categoryAxis = {
    border: { color: palette.axis },
    grid: { display: false },
    ticks: {
      color: palette.text,
      font: { size: 11, family: "'Inter', -apple-system, sans-serif" },
      padding: 6,
      autoSkip: !isHorizontal,
      maxRotation: 45,
      minRotation: 0,
    },
  };

  const scales = {
    y: isHorizontal ? categoryAxis : valueAxis,
    x: isHorizontal ? valueAxis : categoryAxis,
  };

  // Apply axis labels if provided
  if (axisLabels) {
    if (axisLabels.x) {
      scales.x.title = {
        display: true,
        text: axisLabels.x,
        color: palette.text,
        font: axisTitleFont,
        padding: { top: 8 },
      };
    }
    if (axisLabels.y) {
      scales.y.title = {
        display: true,
        text: axisLabels.y,
        color: palette.text,
        font: axisTitleFont,
        padding: { bottom: 8 },
      };
    }
  }

  return scales;
}

// Lazy loading observer for charts
const lazyChartObserver = new IntersectionObserver(
  (entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        const placeholder = entry.target;
        const renderFn = placeholder._lazyRenderFn;
        if (renderFn) {
          renderFn();
          delete placeholder._lazyRenderFn;
        }
        lazyChartObserver.unobserve(placeholder);
      }
    });
  },
  {
    rootMargin: "100px", // Start loading 100px before visible
    threshold: 0.1,
  },
);

/**
 * Cleanup all chart and map instances to prevent memory leaks
 * Call this before loading a new report
 */
function cleanupAllInstances() {
  _renderGeneration++;

  // Destroy all Chart.js instances
  for (const chartId in chartInstances) {
    if (chartInstances[chartId]) {
      try {
        chartInstances[chartId].destroy();
      } catch (e) {
        console.warn("Error destroying chart:", chartId, e);
      }
      delete chartInstances[chartId];
    }
  }

  // Destroy all Leaflet map instances
  if (window.choroplethMaps) {
    for (const mapId in window.choroplethMaps) {
      if (window.choroplethMaps[mapId]) {
        try {
          window.choroplethMaps[mapId].remove();
        } catch (e) {
          console.warn("Error destroying map:", mapId, e);
        }
        delete window.choroplethMaps[mapId];
      }
    }
  }

  // Disconnect lazy observer so stale placeholders don't trigger renders
  lazyChartObserver.disconnect();
}

/**
 * Normalize layout value to supported CSS classes
 * Handles legacy values: "one-column" → "single", "single-column" → "single", "full" → "single"
 * @param {string} layout - The layout value from YAML
 * @returns {string} Normalized layout value
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
  return layoutMap[layout] || "single"; // Default to single if unknown
}

/**
 * Format a number with thousand separators.
 * @param {number|string} value - The value to format
 * @param {{decimals?: number}} [opts] - Optional precision override (from the component's
 *   columnFormats). When set, caps fractional digits at `decimals`; otherwise caps at 1.
 *   Trailing zeros are never padded — `83.2` with decimals=2 stays `83.2`, not `83.20`.
 * @returns {string} Formatted number with commas
 */
function formatNumber(value, opts) {
  if (value === null || value === undefined || value === "") {
    return "";
  }
  const trimmed = String(value).trim();
  const num = Number(trimmed);
  if (trimmed === "" || isNaN(num)) {
    return String(value);
  }
  const decimals =
    opts && Number.isInteger(opts.decimals) && opts.decimals >= 0
      ? opts.decimals
      : 1;
  return num.toLocaleString("en-US", {
    maximumFractionDigits: decimals,
    minimumFractionDigits: 0,
  });
}

/**
 * Format numbers inside HTML content with thousand separators.
 * Caps fractional digits at 1 as a safety net for arbitrary numbers authors
 * type into text templates. Substituted DB values are pre-formatted server-side
 * using the column's roundTo, so this cap only affects literal numbers in the
 * content string.
 * @param {string} html - The HTML content to format
 * @returns {string} HTML with formatted numbers
 */
function formatNumbersInHtml(html) {
  return html.replace(/(?<![\d.])(\d+(?:\.\d+)?)\b(?![^<]*>)/g, (match) => {
    const num = Number(match);
    if (isNaN(num)) return match;
    const hasFraction = match.includes(".");
    if (!hasFraction && num < 1000) return match;
    return num.toLocaleString("en-US", { maximumFractionDigits: 1 });
  });
}

/**
 * Look up the decimals hint for a column/dataset from component.data.columnFormats.
 * Returns undefined when no hint exists — formatters fall back to their default.
 * @param {Object} componentData - component.data (may be null/undefined)
 * @param {string} columnName - header or dataset label
 * @returns {number|undefined}
 */
function getColumnDecimals(componentData, columnName) {
  if (!componentData || !componentData.columnFormats || !columnName)
    return undefined;
  const fmt = componentData.columnFormats[columnName];
  if (!fmt || !Number.isInteger(fmt.decimals) || fmt.decimals < 0)
    return undefined;
  return fmt.decimals;
}

/**
 * Render an info box component — a titled callout with an accent colour left border.
 */
function renderInfobox(element, component) {
  const accentColor = component.accentColor || "#2563a8";
  const div = document.createElement("div");
  div.className = "component component-infobox";
  div.style.setProperty("--infobox-accent", accentColor);

  if (component.title) {
    const titleEl = document.createElement("div");
    titleEl.className = "infobox-title";
    titleEl.textContent = component.title;
    div.appendChild(titleEl);
  }

  if (component.infoboxBody) {
    const bodyEl = document.createElement("div");
    bodyEl.className = "infobox-body";
    // If the body is plain text (no HTML tags), convert newlines to <br> so
    // YAML literal-block scalars (|-) render line breaks correctly.
    const body = component.infoboxBody;
    const isHtml = /<[a-zA-Z][\s\S]*?>/.test(body);
    bodyEl.innerHTML = isHtml ? body : body.replace(/\n/g, "<br>");
    div.appendChild(bodyEl);
  }

  element.appendChild(div);
}

/**
 * Render a text component with optional comparison indicators
 * Supports both period comparison (vs previous month/year) and target comparison (vs static value)
 */
function renderText(element, component) {
  const div = document.createElement("div");
  div.className = "component component-text";

  let html = formatNumbersInHtml(component.content || "");

  // Container for comparison indicators
  let comparisonsHtml = "";

  // Add period comparison indicator if present (e.g., "vs last month")
  const comparison = component.data && component.data.comparison;
  const invert = component.lowerIsBetter;
  if (
    comparison &&
    comparison.direction &&
    comparison.percentageChange !== undefined
  ) {
    const arrow =
      comparison.direction === "up"
        ? "↑"
        : comparison.direction === "down"
          ? "↓"
          : "→";
    let colorClass;
    if (comparison.direction === "unchanged") {
      colorClass = "comparison-unchanged";
    } else if (invert) {
      colorClass =
        comparison.direction === "down" ? "comparison-up" : "comparison-down";
    } else {
      colorClass =
        comparison.direction === "up" ? "comparison-up" : "comparison-down";
    }
    const pctDisplay = Math.abs(comparison.percentageChange).toFixed(1);

    comparisonsHtml += `<span class="comparison-indicator ${colorClass}">${arrow} ${pctDisplay}% vs prior</span>`;
  }

  // Add target comparison indicator if present (e.g., "vs Target")
  const targetComparison = component.data && component.data.targetComparison;
  if (
    targetComparison &&
    targetComparison.direction &&
    targetComparison.percentageChange !== undefined
  ) {
    const arrow =
      targetComparison.direction === "up"
        ? "↑"
        : targetComparison.direction === "down"
          ? "↓"
          : "→";
    let colorClass;
    if (targetComparison.direction === "unchanged") {
      colorClass = "comparison-unchanged";
    } else if (invert) {
      colorClass =
        targetComparison.direction === "down"
          ? "comparison-up"
          : "comparison-down";
    } else {
      colorClass =
        targetComparison.direction === "up"
          ? "comparison-up"
          : "comparison-down";
    }
    const pctDisplay = Math.abs(targetComparison.percentageChange).toFixed(1);
    const label = targetComparison.targetLabel || "Target";
    const preposition =
      targetComparison.direction === "up"
        ? "above"
        : targetComparison.direction === "down"
          ? "below"
          : "at";

    comparisonsHtml += `<span class="comparison-indicator ${colorClass}">${arrow} ${pctDisplay}% ${preposition} ${label}</span>`;
  }

  if (comparisonsHtml) {
    html += `<div class="comparison-indicators">${comparisonsHtml}</div>`;
  }

  div.innerHTML = html;
  element.appendChild(div);
}

/**
 * Render a line chart using Chart.js
 */
function renderLineChart(element, component) {
  const div = document.createElement("div");
  div.className = "component component-chart";

  const canvas = document.createElement("canvas");
  canvas.id = "chart-" + Math.random().toString(36).substr(2, 9);

  // Set accessibility attributes
  setChartAccessibility(canvas, component);

  if (component.title) {
    const title = document.createElement("h3");
    title.className = "component-title";
    title.textContent = component.title;
    div.appendChild(title);
  }

  if (component.description) {
    const desc = document.createElement("p");
    desc.className = "component-description";
    desc.textContent = component.description;
    div.appendChild(desc);
  }

  div.appendChild(canvas);
  element.appendChild(div);

  // Defer chart creation until visible (lazy rendering)
  const createChart = () => {
    // Prepare data
    const data = component.data || { labels: [], datasets: [] };

    // Check if we have data to render
    if (!data.labels || data.labels.length === 0) {
      console.warn("Line chart has no labels:", component.title);
      div.appendChild(createNoDataMessage(component.title));
      return;
    }

    const datasets = (data.datasets || []).map((dataset, index) => {
      const color = getDatasetColor(dataset, index);
      return {
        label: dataset.label,
        data: dataset.data,
        borderColor: color,
        backgroundColor: color + "15",
        borderWidth: 2.5,
        fill: false,
        tension: 0.3,
        pointRadius: 3,
        pointBackgroundColor: color,
        pointBorderColor: CHART_PALETTE.background,
        pointBorderWidth: 1.5,
        pointHoverRadius: 5,
        pointHoverBorderWidth: 2,
      };
    });

    // Use requestAnimationFrame to ensure DOM is ready
    const gen1 = _renderGeneration;
    requestAnimationFrame(() => {
      if (gen1 !== _renderGeneration) return;
      // Create chart
      const ctx = canvas.getContext("2d");
      const chartId = canvas.id;

      // Destroy existing chart if it exists
      if (chartInstances[chartId]) {
        chartInstances[chartId].destroy();
      }

      // Legend configuration from component data
      const legendConfig = data.legend || {};
      const showLegend = legendConfig.show !== false; // Default to true
      const legendPosition = legendConfig.position || "top";

      // Build plugins array (reference lines if configured)
      const chartPlugins = [];
      const refLinesPlugin = createReferenceLinesPlugin(
        data.referenceLines,
        false,
      );
      if (refLinesPlugin) {
        chartPlugins.push(refLinesPlugin);
      }
      if (data.trendLine) {
        const trendPlugin = createTrendLinePlugin(
          datasets,
          data.trendLineColor,
        );
        if (trendPlugin) chartPlugins.push(trendPlugin);
      }
      const axisTooltipPlugin = createAxisTooltipPlugin(component);
      if (axisTooltipPlugin) chartPlugins.push(axisTooltipPlugin);

      const showValuesWithAxis = component.showValuesWithAxis || false;
      if (showValuesWithAxis) {
        chartPlugins.push(createValueLabelsPlugin(false, data));
      }

      // Get scales and hide axis titles if requested
      const scales = getPublicationScales(
        false,
        data.axisLabels,
        data,
        data.yLimit,
      );
      const showAxisTitles =
        component.showAxisTitles !== undefined
          ? component.showAxisTitles !== false
          : !component.hideAxes;
      if (!showAxisTitles) {
        if (scales.x.title) scales.x.title.display = false;
        if (scales.y.title) scales.y.title.display = false;
      }

      chartInstances[chartId] = new Chart(ctx, {
        type: "line",
        data: {
          labels: data.labels,
          datasets: datasets,
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          layout: {
            padding: {
              top: showValuesWithAxis ? 24 : 10,
              right: 20,
              bottom: 10,
              left: 10,
            },
          },
          plugins: {
            legend: {
              display: showLegend,
              position: legendPosition,
              labels: {
                color: CHART_PALETTE.text,
                usePointStyle: true,
                pointStyle: "circle",
                padding: 16,
                font: {
                  size: 11,
                  family: "'Inter', -apple-system, sans-serif",
                },
              },
            },
            title: {
              display: false,
            },
            tooltip: {
              backgroundColor: "rgba(30, 58, 95, 0.95)",
              titleFont: { size: 12, weight: "600" },
              bodyFont: { size: 11 },
              padding: 10,
              cornerRadius: 4,
              callbacks: {
                label: function (context) {
                  const decimals = getColumnDecimals(
                    data,
                    context.dataset.label,
                  );
                  return (
                    context.dataset.label +
                    ": " +
                    formatNumber(context.parsed.y, { decimals })
                  );
                },
              },
            },
          },
          scales: scales,
        },
        plugins: chartPlugins,
      });
    }); // end requestAnimationFrame
  };

  div._lazyRenderFn = createChart;
  lazyChartObserver.observe(div);
}

/**
 * Render a bar chart using Chart.js
 */
function renderBarChart(element, component) {
  const div = document.createElement("div");
  div.className = "component component-chart";

  const canvas = document.createElement("canvas");
  canvas.id = "chart-" + Math.random().toString(36).substr(2, 9);

  // Set accessibility attributes
  setChartAccessibility(canvas, component);

  if (component.title) {
    const title = document.createElement("h3");
    title.className = "component-title";
    title.textContent = component.title;
    div.appendChild(title);
  }

  if (component.description) {
    const desc = document.createElement("p");
    desc.className = "component-description";
    desc.textContent = component.description;
    div.appendChild(desc);
  }

  div.appendChild(canvas);
  element.appendChild(div);

  // Defer chart creation until visible (lazy rendering)
  const createChart = () => {
    // Prepare data
    const data = component.data || { labels: [], datasets: [] };

    // Check if we have data to render
    if (!data.labels || data.labels.length === 0) {
      console.warn("Bar chart has no labels:", component.title);
      div.appendChild(createNoDataMessage(component.title));
      return;
    }

    // Get canvas context for pattern creation (needed before chart init)
    const ctx = canvas.getContext("2d");

    // Bar chart options
    const isStacked = component.stacked || false;
    const barBorderColor = getBarBorderColor(isStacked);
    const barBorderWidth = isStacked ? 0 : 1;

    let mappedDatasets = (data.datasets || []).map((dataset, index) => {
      const color = getDatasetColor(dataset, index);
      // Use per-bar color array when server provides one; otherwise apply pattern
      let bgColor;
      if (dataset.colors && dataset.colors.length > 0) {
        bgColor = dataset.colors.map((c, ci) => getPatternForIndex(c, ci, ctx));
      } else {
        bgColor =
          data.datasets.length > 1
            ? getPatternForIndex(color, index, ctx)
            : color;
      }
      return {
        label: dataset.label,
        data: dataset.data,
        backgroundColor: bgColor,
        borderColor: barBorderColor,
        borderWidth: barBorderWidth,
        borderRadius: 2,
        borderSkipped: false,
      };
    });

    // For grouped bars, sort datasets so larger values appear first (left)
    if (!isStacked && mappedDatasets.length > 1) {
      mappedDatasets.sort((a, b) => {
        const sumA = (a.data || []).reduce((s, v) => s + (Number(v) || 0), 0);
        const sumB = (b.data || []).reduce((s, v) => s + (Number(v) || 0), 0);
        return sumB - sumA;
      });
    }
    const datasets = mappedDatasets;
    const isHorizontal = component.horizontal || false;

    // Legend configuration from component data
    const legendConfig = data.legend || {};
    const showLegend = legendConfig.show !== false; // Default to true
    const legendPosition = legendConfig.position || "top";

    // Use requestAnimationFrame to ensure DOM is ready
    const gen2 = _renderGeneration;
    requestAnimationFrame(() => {
      if (gen2 !== _renderGeneration) return;
      const chartId = canvas.id;

      // Destroy existing chart if it exists
      if (chartInstances[chartId]) {
        chartInstances[chartId].destroy();
      }

      // Get publication scales and add stacked option
      const scales = getPublicationScales(
        isHorizontal,
        data.axisLabels,
        data,
        data.yLimit,
      );
      scales.y.stacked = isStacked;
      scales.x.stacked = isStacked;

      // Wrap long category-axis labels onto multiple lines when enabled.
      // Labels are split on the en-dash separator first (e.g. "District – Facility"),
      // then fall back to word-boundary wrapping at ~18 characters per line.
      if (component.wrapLabels) {
        const catAxis = isHorizontal ? scales.y : scales.x;
        catAxis.ticks.autoSkip = false;
        catAxis.ticks.maxRotation = 45;
        catAxis.ticks.minRotation = 45;
        catAxis.ticks.callback = function (value) {
          const label = this.getLabelForValue(value);
          if (typeof label !== "string") return label;
          if (label.includes(" \u2013 ")) return label.split(" \u2013 ");
          if (label.length <= 18) return label;
          const words = label.split(" ");
          const lines = [];
          let current = "";
          for (const word of words) {
            if (current && current.length + word.length + 1 > 18) {
              lines.push(current);
              current = word;
            } else {
              current = current ? current + " " + word : word;
            }
          }
          if (current) lines.push(current);
          return lines;
        };
      }

      const showValues = component.showValues || false;
      const showValuesWithAxis = component.showValuesWithAxis || false;
      const showAxisTitles =
        component.showAxisTitles !== undefined
          ? component.showAxisTitles !== false
          : !component.hideAxes;

      // Hide the numeric (value) axis if showValues is enabled (but not if showValuesWithAxis is enabled).
      // For horizontal charts, the value axis is X; for vertical charts, the value axis is Y.
      if (showValues && !showValuesWithAxis) {
        if (isHorizontal && scales.x) {
          scales.x.display = false;
        } else if (!isHorizontal && scales.y) {
          scales.y.display = false;
        }
      }

      // Hide axis titles if requested.
      if (!showAxisTitles) {
        if (scales.x.title) scales.x.title.display = false;
        if (scales.y.title) scales.y.title.display = false;
      }

      // Build plugins array (reference lines if configured)
      const chartPlugins = [];
      const refLinesPlugin = createReferenceLinesPlugin(
        data.referenceLines,
        isHorizontal,
      );
      if (refLinesPlugin) {
        chartPlugins.push(refLinesPlugin);
      }
      if (showValues || showValuesWithAxis) {
        chartPlugins.push(createValueLabelsPlugin(isHorizontal, data));
      }
      if (data.trendLine) {
        const trendPlugin = createTrendLinePlugin(
          datasets,
          data.trendLineColor,
        );
        if (trendPlugin) chartPlugins.push(trendPlugin);
      }
      const axisTooltipPlugin = createAxisTooltipPlugin(component);
      if (axisTooltipPlugin) chartPlugins.push(axisTooltipPlugin);

      chartInstances[chartId] = new Chart(ctx, {
        type: "bar",
        data: {
          labels: data.labels,
          datasets: datasets,
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          indexAxis: isHorizontal ? "y" : data.indexAxis || "x",
          layout: {
            padding: {
              top: showValues || showValuesWithAxis ? 24 : 10,
              right:
                (showValues || showValuesWithAxis) && isHorizontal ? 50 : 20,
              bottom: 10,
              left: 10,
            },
          },
          plugins: {
            legend: {
              display: showLegend,
              position: legendPosition,
              labels: {
                color: CHART_PALETTE.text,
                usePointStyle: true,
                pointStyle: "rectRounded",
                padding: 16,
                font: {
                  size: 11,
                  family: "'Inter', -apple-system, sans-serif",
                },
              },
            },
            title: {
              display: false,
            },
            tooltip: {
              backgroundColor: "rgba(30, 58, 95, 0.95)",
              titleFont: { size: 12, weight: "600" },
              bodyFont: { size: 11 },
              padding: 10,
              cornerRadius: 4,
              callbacks: {
                label: function (context) {
                  const value = isHorizontal
                    ? context.parsed.x
                    : context.parsed.y;
                  const decimals = getColumnDecimals(
                    data,
                    context.dataset.label,
                  );
                  return (
                    context.dataset.label +
                    ": " +
                    formatNumber(value, { decimals })
                  );
                },
              },
            },
          },
          scales: scales,
        },
        plugins: chartPlugins,
      });
    }); // end requestAnimationFrame
  };

  div._lazyRenderFn = createChart;
  lazyChartObserver.observe(div);
}

function normalizePieLabelAngle(angle) {
  const tau = Math.PI * 2;
  return ((angle % tau) + tau) % tau;
}

function spreadPieLabelsAroundCircle(items, labelRadius, minArcLength) {
  if (items.length === 0) return;

  if (items.length === 1) {
    items[0].labelAngle = items[0].idealAngle;
    return;
  }

  const tau = Math.PI * 2;
  const gap = Math.min(
    minArcLength / Math.max(labelRadius, 1),
    (tau - 0.01) / items.length,
  );

  items.sort((a, b) => a.idealAngle - b.idealAngle);

  let largestGap = -1;
  let largestGapIndex = 0;
  for (let i = 0; i < items.length; i++) {
    const currentAngle = items[i].idealAngle;
    const nextAngle =
      items[(i + 1) % items.length].idealAngle +
      (i === items.length - 1 ? tau : 0);
    const angleGap = nextAngle - currentAngle;
    if (angleGap > largestGap) {
      largestGap = angleGap;
      largestGapIndex = i;
    }
  }

  const ordered = [];
  for (let offset = 1; offset <= items.length; offset++) {
    ordered.push(items[(largestGapIndex + offset) % items.length]);
  }

  ordered.forEach((item, index) => {
    item.unwrappedAngle = item.idealAngle;
    if (index > 0) {
      while (item.unwrappedAngle <= ordered[index - 1].unwrappedAngle) {
        item.unwrappedAngle += tau;
      }
    }
  });

  ordered[0].labelAngle = ordered[0].unwrappedAngle;
  for (let i = 1; i < ordered.length; i++) {
    ordered[i].labelAngle = Math.max(
      ordered[i].unwrappedAngle,
      ordered[i - 1].labelAngle + gap,
    );
  }

  const maxEndAngle = ordered[0].unwrappedAngle + tau - gap;
  const overflow = ordered[ordered.length - 1].labelAngle - maxEndAngle;
  if (overflow > 0) {
    ordered.forEach((item) => {
      item.labelAngle -= overflow;
    });
  }

  for (let i = ordered.length - 2; i >= 0; i--) {
    ordered[i].labelAngle = Math.min(
      ordered[i].labelAngle,
      ordered[i + 1].labelAngle - gap,
    );
  }

  ordered.forEach((item) => {
    item.labelAngle = normalizePieLabelAngle(item.labelAngle);
  });
}

function getPieLegendLayout(containerWidth, itemCount) {
  const safeWidth = Math.max(280, Number(containerWidth) || 600);
  const count = Math.max(1, itemCount || 1);
  const spacePerKey = safeWidth / count;

  return {
    padding: Math.round(Math.max(8, Math.min(28, spacePerKey * 0.15))),
    boxSize: count > 10 || safeWidth < 420 ? 8 : 10,
    fontSize: count > 12 || safeWidth < 360 ? 10 : 11,
  };
}

function getPieOuterLabelBounds(chart, labelBlockHeight, overflow = 26) {
  const halfLabelHeight = labelBlockHeight / 2;
  const edgePadding = 4;
  const chartHeight = Number(chart.height) || 0;
  const chartArea = chart.chartArea || { top: 0, bottom: chartHeight };
  const legend = chart.legend;
  const legendOptions = legend?.options || {};
  const legendDisplayed = !!legend && legendOptions.display !== false;
  const legendPosition = legendOptions.position || legend.position || "";

  const topOverflow =
    legendDisplayed && legendPosition === "top" ? 0 : overflow;
  const bottomOverflow =
    legendDisplayed && legendPosition === "bottom"
      ? -(halfLabelHeight + edgePadding)
      : overflow;

  const minY = halfLabelHeight + edgePadding;
  const maxY = chartHeight - halfLabelHeight - edgePadding;
  const labelTop = Math.max(minY, chartArea.top - topOverflow);

  const labelBottom = Math.min(maxY, chartArea.bottom + bottomOverflow);

  return {
    labelTop,
    labelBottom: Math.max(labelTop, labelBottom),
  };
}

/**
 * Chart.js plugin that draws value + percentage labels outside each pie/doughnut slice.
 * Labels are nudged around the full outside ring so tiny adjacent slices do not
 * stack their values on top of each other or gather in one column.
 */
function createPieOuterLabelsPlugin(allData, labels, decimals) {
  return {
    id: "pieOuterLabels",
    afterDatasetsDraw: function (chart) {
      const ctx = chart.ctx;
      const dataset = chart.data.datasets[0];
      const meta = chart.getDatasetMeta(0);
      const total = allData.reduce((s, v, index) => {
        if (chart.getDataVisibility && !chart.getDataVisibility(index))
          return s;
        return s + (parseFloat(v) || 0);
      }, 0);
      if (total === 0) return;

      const palette = getChartPalette();
      const cx = (chart.chartArea.left + chart.chartArea.right) / 2;
      const cy = (chart.chartArea.top + chart.chartArea.bottom) / 2;
      // Use the outer radius of the arcs as the base for leader lines.
      const outerR = meta.data[0] ? meta.data[0].outerRadius : 0;
      const lineEnd = outerR + 22;
      const lineHeight = 11;
      const labelBlockHeight = lineHeight * 2;
      const labelRadius = outerR + 24;
      const { labelTop, labelBottom } = getPieOuterLabelBounds(
        chart,
        labelBlockHeight,
      );

      ctx.save();
      ctx.font = "600 10px 'Inter', -apple-system, sans-serif";
      ctx.textBaseline = "middle";

      const labelItems = [];

      meta.data.forEach((arc, i) => {
        if (chart.getDataVisibility && !chart.getDataVisibility(i)) return;
        const val = parseFloat(allData[i]) || 0;
        if (val === 0) return;
        const pct = ((val / total) * 100).toFixed(1);
        const labelLines = [formatNumber(val, { decimals }), "(" + pct + "%)"];
        const textWidth = Math.max(
          ...labelLines.map((line) => ctx.measureText(line).width),
        );

        // Mid-angle of the slice (Chart.js stores startAngle/endAngle on each arc element)
        const midAngle = arc.startAngle + (arc.endAngle - arc.startAngle) / 2;
        const cos = Math.cos(midAngle);
        const sin = Math.sin(midAngle);
        if (!Number.isFinite(cos) || !Number.isFinite(sin)) return;

        // Leader line: from edge of slice to just outside
        const x1 = cx + cos * (outerR + 2);
        const y1 = cy + sin * (outerR + 2);
        const x2 = cx + cos * lineEnd;
        const y2 = cy + sin * lineEnd;

        const rawColor = dataset.backgroundColor[i];
        // backgroundColor may be a canvas pattern; fall back to the base palette color
        const strokeColor =
          typeof rawColor === "string" ? rawColor : palette.text;

        labelItems.push({
          labelLines,
          textWidth,
          x1,
          y1,
          x2,
          y2,
          idealAngle: normalizePieLabelAngle(midAngle),
          strokeColor,
        });
      });

      const widestLabel = labelItems.reduce(
        (max, item) => Math.max(max, item.textWidth),
        0,
      );
      const minArcLength = Math.max(
        labelBlockHeight + 4,
        Math.min(58, widestLabel * 0.75),
      );
      spreadPieLabelsAroundCircle(labelItems, labelRadius, minArcLength);

      labelItems.forEach((item) => {
        const labelAngle = item.labelAngle ?? item.idealAngle;
        const labelCos = Math.cos(labelAngle);
        const labelSin = Math.sin(labelAngle);
        const textWidth = item.textWidth;
        let textX = cx + labelCos * labelRadius;
        let textY = cy + labelSin * labelRadius;
        let textAlign = "center";

        if (Math.abs(labelCos) >= 0.25) {
          const isRightSide = labelCos > 0;
          const outsideEdgeX = isRightSide ? cx + outerR + 8 : cx - outerR - 8;
          const boundedTextX = isRightSide
            ? Math.min(textX, chart.width - textWidth - 4)
            : Math.max(textX, textWidth + 4);

          textX = isRightSide
            ? Math.max(outsideEdgeX, boundedTextX)
            : Math.min(outsideEdgeX, boundedTextX);
          textAlign = isRightSide ? "left" : "right";
        } else {
          textX = Math.min(
            Math.max(textX, textWidth / 2 + 4),
            chart.width - textWidth / 2 - 4,
          );
        }

        textY = Math.min(
          Math.max(textY, labelTop),
          Math.max(labelTop, labelBottom),
        );

        const distanceFromCenter = Math.hypot(textX - cx, textY - cy);
        const minClearance = outerR + 8 + labelBlockHeight / 2;
        if (distanceFromCenter < minClearance) {
          const scale = minClearance / Math.max(distanceFromCenter, 1);
          textX = cx + (textX - cx) * scale;
          textY = cy + (textY - cy) * scale;
          // Re-clamp: the clearance push can move textY past the legend boundary.
          textY = Math.min(
            Math.max(textY, labelTop),
            Math.max(labelTop, labelBottom),
          );
        }

        const labelLineEndX =
          textAlign === "left"
            ? textX - 4
            : textAlign === "right"
              ? textX + 4
              : textX;

        ctx.beginPath();
        ctx.moveTo(item.x1, item.y1);
        ctx.lineTo(item.x2, item.y2);
        ctx.lineTo(labelLineEndX, textY);
        ctx.strokeStyle = item.strokeColor;
        ctx.lineWidth = 1;
        ctx.stroke();

        ctx.fillStyle = palette.text;
        ctx.textAlign = textAlign;
        item.labelLines.forEach((line, lineIndex) => {
          const lineY =
            textY + (lineIndex - (item.labelLines.length - 1) / 2) * lineHeight;
          ctx.fillText(line, textX, lineY);
        });
      });

      ctx.restore();
    },
  };
}

/**
 * Render a pie chart using Chart.js
 */
function renderPieChart(element, component) {
  const div = document.createElement("div");
  div.className = "component component-chart";

  const canvas = document.createElement("canvas");
  canvas.id = "chart-" + Math.random().toString(36).substr(2, 9);

  // Set accessibility attributes
  setChartAccessibility(canvas, component);

  if (component.title) {
    const title = document.createElement("h3");
    title.className = "component-title";
    title.textContent = component.title;
    div.appendChild(title);
  }

  if (component.description) {
    const desc = document.createElement("p");
    desc.className = "component-description";
    desc.textContent = component.description;
    div.appendChild(desc);
  }

  const canvasWrapper = document.createElement("div");
  canvasWrapper.className = "pie-canvas-wrapper";
  canvasWrapper.appendChild(canvas);
  div.appendChild(canvasWrapper);
  element.appendChild(div);

  // Defer chart creation until visible (lazy rendering)
  const createChart = () => {
    // Prepare data
    const data = component.data || { labels: [], datasets: [] };

    // For pie charts, combine all dataset values
    if (!data.datasets || data.datasets.length === 0) {
      div.appendChild(createNoDataMessage(component.title));
      return;
    }

    // Get canvas context for pattern creation
    const ctx = canvas.getContext("2d");

    // Combine all dataset values and colors for pie chart
    const allData = [];
    const backgroundColor = [];
    const plainColors = []; // plain hex colors for the HTML legend swatches

    data.datasets.forEach((ds, index) => {
      // Take the first value from each dataset (for aggregation-based pie charts)
      if (ds.data && ds.data.length > 0) {
        allData.push(parseFloat(ds.data[0]) || 0);
      }
      const color = getDatasetColor(ds, index);
      plainColors.push(color);
      // Apply subtle pattern for colorblind accessibility (pie charts benefit most)
      backgroundColor.push(getPatternForIndex(color, index, ctx));
    });

    // Create chart
    const gen3 = _renderGeneration;
    requestAnimationFrame(() => {
      if (gen3 !== _renderGeneration) return;
      const chartId = canvas.id;

      // Destroy existing chart if it exists
      if (chartInstances[chartId]) {
        chartInstances[chartId].destroy();
      }

      const legendConfig = data.legend || {};
      const showLegend = legendConfig.show !== false;

      const decimals =
        data.datasets && data.datasets.length > 0
          ? getColumnDecimals(data, data.datasets[0].label)
          : 0;
      const total = allData.reduce((s, v) => s + (parseFloat(v) || 0), 0);

      chartInstances[chartId] = new Chart(ctx, {
        type: "doughnut",
        data: {
          labels: data.labels,
          datasets: [
            {
              data: allData,
              backgroundColor: backgroundColor,
              borderColor: CHART_PALETTE.border,
              borderWidth: 1,
              hoverBorderWidth: 2,
            },
          ],
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          cutout: "55%",
          layout: {
            padding: { top: 10, right: 20, bottom: 10, left: 20 },
          },
          plugins: {
            legend: { display: false },
            title: { display: false },
            tooltip: {
              backgroundColor: "rgba(30, 58, 95, 0.95)",
              titleFont: { size: 12, weight: "600" },
              bodyFont: { size: 11 },
              padding: 10,
              cornerRadius: 4,
              callbacks: {
                label: function (context) {
                  const label = context.label || "";
                  const value = context.parsed || 0;
                  const percentage =
                    total > 0 ? ((value / total) * 100).toFixed(1) : "0.0";
                  const decimals = getColumnDecimals(data, label);
                  return (
                    label +
                    ": " +
                    formatNumber(value, { decimals }) +
                    " (" +
                    percentage +
                    "%)"
                  );
                },
              },
            },
          },
        },
        plugins: [],
      });

      // Build HTML legend so the value/percentage can be styled differently
      // from the label name — canvas text can't be partially styled.
      if (showLegend) {
        const legendEl = document.createElement("div");
        legendEl.className = "pie-html-legend";
        (data.labels || []).forEach((label, i) => {
          const val = parseFloat(allData[i]) || 0;
          const pct = total > 0 ? ((val / total) * 100).toFixed(1) : "0.0";
          const color = plainColors[i] || "#888";
          const item = document.createElement("div");
          item.className = "pie-legend-item";
          const swatch = document.createElement("span");
          swatch.className = "pie-legend-swatch";
          swatch.style.background = color;
          const name = document.createElement("span");
          name.className = "pie-legend-name";
          name.textContent = label;
          const value = document.createElement("span");
          value.className = "pie-legend-value";
          value.textContent =
            ": " + formatNumber(val, { decimals }) + " (" + pct + "%)";
          item.appendChild(swatch);
          item.appendChild(name);
          item.appendChild(value);
          legendEl.appendChild(item);
        });
        div.appendChild(legendEl);
      }
    });
  };

  div._lazyRenderFn = createChart;
  lazyChartObserver.observe(div);
}

/**
 * Render a population pyramid chart (horizontal mirrored bar)
 * First dataset values are already negated by the server.
 */
function renderPyramidChart(element, component) {
  const div = document.createElement("div");
  div.className = "component component-chart";

  const canvas = document.createElement("canvas");
  canvas.id = "chart-" + Math.random().toString(36).substr(2, 9);

  setChartAccessibility(canvas, component);

  if (component.title) {
    const title = document.createElement("h3");
    title.className = "component-title";
    title.textContent = component.title;
    div.appendChild(title);
  }

  if (component.description) {
    const desc = document.createElement("p");
    desc.className = "component-description";
    desc.textContent = component.description;
    div.appendChild(desc);
  }

  div.appendChild(canvas);
  element.appendChild(div);

  const createChart = () => {
    const data = component.data || { labels: [], datasets: [] };

    if (!data.labels || data.labels.length === 0) {
      div.appendChild(createNoDataMessage(component.title));
      return;
    }

    const pyramidColors = ["#2d5a87", "#c06070"];
    const datasets = (data.datasets || []).map((dataset, index) => ({
      label: dataset.label,
      data: dataset.data,
      backgroundColor: pyramidColors[index] || getDatasetColor(dataset, index),
      borderColor: CHART_PALETTE.stackedBorder,
      borderWidth: 1,
      borderRadius: 2,
      borderSkipped: false,
    }));

    const legendConfig = data.legend || {};
    const showLegend = legendConfig.show !== false;
    const legendPosition = legendConfig.position || "top";

    const gen = _renderGeneration;
    requestAnimationFrame(() => {
      if (gen !== _renderGeneration) return;
      const chartId = canvas.id;

      if (chartInstances[chartId]) {
        chartInstances[chartId].destroy();
      }

      const scales = getPublicationScales(true, data.axisLabels, data);
      scales.y.stacked = true;
      scales.x.stacked = true;
      // Pyramid axis shares precision across both sides (left is negated); use first dataset.
      const axisDecimals =
        data.datasets && data.datasets.length > 0
          ? getColumnDecimals(data, data.datasets[0].label)
          : undefined;
      scales.x.ticks.callback = function (value) {
        return formatNumber(Math.abs(value), { decimals: axisDecimals });
      };

      const chartPlugins = [];
      const refLinesPlugin = createReferenceLinesPlugin(
        data.referenceLines,
        true,
      );
      if (refLinesPlugin) {
        chartPlugins.push(refLinesPlugin);
      }
      chartPlugins.push({
        id: "pyramidValueLabels",
        afterDatasetsDraw: function (chart) {
          const ctx = chart.ctx;
          const palette = getChartPalette();
          chart.data.datasets.forEach((dataset, datasetIndex) => {
            const meta = chart.getDatasetMeta(datasetIndex);
            if (!meta || meta.hidden) return;
            const dsDecimals = getColumnDecimals(data, dataset.label);
            meta.data.forEach((bar, index) => {
              const value = dataset.data[index];
              if (value == null || value === 0) return;
              const label = formatNumber(Math.abs(value), {
                decimals: dsDecimals,
              });
              ctx.save();
              ctx.font = "600 10px 'Inter', -apple-system, sans-serif";
              ctx.textBaseline = "middle";
              let lx,
                ly = bar.y;
              if (value < 0) {
                ctx.textAlign = "right";
                lx = bar.x - 4;
              } else {
                ctx.textAlign = "left";
                lx = bar.x + 4;
              }
              drawValueLabelBg(ctx, label, lx, ly, palette);
              ctx.fillStyle = palette.text;
              ctx.fillText(label, lx, ly);
              ctx.restore();
            });
          });
        },
      });
      const axisTooltipPlugin = createAxisTooltipPlugin(component);
      if (axisTooltipPlugin) chartPlugins.push(axisTooltipPlugin);

      chartInstances[chartId] = new Chart(canvas.getContext("2d"), {
        type: "bar",
        data: {
          labels: data.labels,
          datasets: datasets,
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          indexAxis: "y",
          layout: {
            padding: { top: 10, right: 50, bottom: 10, left: 50 },
          },
          plugins: {
            legend: {
              display: showLegend,
              position: legendPosition,
              labels: {
                color: CHART_PALETTE.text,
                usePointStyle: true,
                pointStyle: "rectRounded",
                padding: 16,
                font: {
                  size: 11,
                  family: "'Inter', -apple-system, sans-serif",
                },
              },
            },
            title: { display: false },
            tooltip: {
              backgroundColor: "rgba(30, 58, 95, 0.95)",
              titleFont: { size: 12, weight: "600" },
              bodyFont: { size: 11 },
              padding: 10,
              cornerRadius: 4,
              callbacks: {
                label: function (context) {
                  const value = Math.abs(context.parsed.x);
                  const decimals = getColumnDecimals(
                    data,
                    context.dataset.label,
                  );
                  return (
                    context.dataset.label +
                    ": " +
                    formatNumber(value, { decimals })
                  );
                },
              },
            },
          },
          scales: scales,
        },
        plugins: chartPlugins,
      });
    });
  };

  div._lazyRenderFn = createChart;
  lazyChartObserver.observe(div);
}

/**
 * Render a combined bar and line chart
 */
function renderBarLineChart(element, component) {
  const data = component.data || {};

  // Create container and title first (so we can show "no data" message)
  const div = document.createElement("div");
  div.className = "component component-chart";

  if (component.title) {
    const title = document.createElement("h3");
    title.className = "component-title";
    title.textContent = component.title;
    div.appendChild(title);
  }

  if (component.description) {
    const desc = document.createElement("p");
    desc.className = "component-description";
    desc.textContent = component.description;
    div.appendChild(desc);
  }

  // Check for valid data
  if (
    !data.labels ||
    data.labels.length === 0 ||
    !data.datasets ||
    data.datasets.length === 0
  ) {
    console.warn(
      "Bar-line chart has no data:",
      component.title,
      "labels:",
      data.labels,
      "datasets:",
      data.datasets,
    );
    div.appendChild(createNoDataMessage(component.title));
    element.appendChild(div);
    return;
  }

  // Create canvas
  const canvas = document.createElement("canvas");

  // Set accessibility attributes
  setChartAccessibility(canvas, component);

  element.appendChild(div);
  div.appendChild(canvas);

  // Use requestAnimationFrame to ensure DOM is ready
  const gen3 = _renderGeneration;
  requestAnimationFrame(() => {
    if (gen3 !== _renderGeneration) return;
    const ctx = canvas.getContext("2d");

    // Destroy existing chart if present
    const chartId = `chart-${(component.title || "chart").replace(/[^a-z0-9]/gi, "-").toLowerCase()}`;
    if (chartInstances[chartId]) {
      chartInstances[chartId].destroy();
      delete chartInstances[chartId];
    }
    canvas.id = chartId;

    // Count bar datasets for pattern assignment
    let barIndex = 0;

    // Map datasets with type-specific styling
    const datasets = (data.datasets || []).map((dataset, index) => {
      const isLine = dataset.chartType === "line";
      const color = getDatasetColor(dataset, index);

      if (isLine) {
        // Line styling - publication quality
        return {
          label: dataset.label,
          data: dataset.data,
          type: "line",
          borderColor: color,
          backgroundColor: color + "15",
          borderWidth: 2.5,
          fill: false,
          tension: 0.3,
          pointRadius: 3,
          pointStyle: "circle",
          pointBackgroundColor: color,
          pointBorderColor: CHART_PALETTE.background,
          pointBorderWidth: 1.5,
          pointHoverRadius: 5,
          yAxisID: "y1",
          order: 0,
        };
      } else {
        // Bar styling — use per-bar color array when server provides one (single metric
        // column where data.datasets labels match x-axis category values, e.g. AWaRe chart)
        let bgColor;
        if (dataset.colors && dataset.colors.length > 0) {
          bgColor = dataset.colors.map((c, ci) =>
            getPatternForIndex(c, ci, ctx),
          );
        } else {
          bgColor = getPatternForIndex(color, barIndex++, ctx);
        }
        return {
          label: dataset.label,
          data: dataset.data,
          type: "bar",
          backgroundColor: bgColor,
          borderColor: CHART_PALETTE.border,
          borderWidth: 1,
          borderRadius: 2,
          yAxisID: "y",
          order: 1,
        };
      }
    });

    // Legend configuration from component data
    const legendConfig = data.legend || {};
    const showLegend = legendConfig.show !== false; // Default to true
    const legendPosition = legendConfig.position || "top";

    // Show value labels on bars and optionally hide Y-axis and axis titles.
    const showValues = component.showValues || false;
    const showValuesWithAxis = component.showValuesWithAxis || false;
    const showAxisTitles =
      component.showAxisTitles !== undefined
        ? component.showAxisTitles !== false
        : !component.hideAxes;

    // Build plugins array (reference lines if configured)
    const chartPlugins = [];
    const refLinesPlugin = createReferenceLinesPlugin(
      data.referenceLines,
      false,
    );
    if (refLinesPlugin) {
      chartPlugins.push(refLinesPlugin);
    }
    if (showValues || showValuesWithAxis) {
      chartPlugins.push(createValueLabelsPlugin(false, data));
    }
    const axisTooltipPlugin = createAxisTooltipPlugin(component);
    if (axisTooltipPlugin) chartPlugins.push(axisTooltipPlugin);

    // Pre-resolve per-axis decimals: y → first bar dataset, y1 → first line dataset.
    const firstBar = (data.datasets || []).find(
      (ds) => ds.chartType !== "line",
    );
    const firstLine = (data.datasets || []).find(
      (ds) => ds.chartType === "line",
    );
    const yDecimals = firstBar
      ? getColumnDecimals(data, firstBar.label)
      : undefined;
    const y1Decimals = firstLine
      ? getColumnDecimals(data, firstLine.label)
      : undefined;

    // Ensure y1 scale max covers any y1-targeted reference lines so they are not
    // clipped when the benchmark value exceeds the auto-scaled data maximum.
    let y1ComputedMax = undefined;
    const y1RefLines = (data.referenceLines || []).filter(
      (rl) => rl.axis === "y1",
    );
    if (y1RefLines.length > 0) {
      const maxRefVal = Math.max(...y1RefLines.map((rl) => rl.value));
      // Find the max value in line datasets so we don't shrink the axis
      let dataMax = 0;
      (data.datasets || [])
        .filter((ds) => ds.chartType === "line")
        .forEach((ds) => {
          (ds.data || []).forEach((v) => {
            if (typeof v === "number" && v > dataMax) dataMax = v;
          });
        });
      const needed = Math.max(maxRefVal, dataMax);
      // Only override if the reference line would be at or above auto max
      if (maxRefVal >= dataMax) {
        y1ComputedMax = needed * 1.15; // 15% headroom above the benchmark
      }
    }

    chartInstances[chartId] = new Chart(ctx, {
      type: "bar", // Base type (Chart.js requires a base type for mixed charts)
      data: {
        labels: data.labels,
        datasets: datasets,
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        layout: {
          padding: {
            top: showValues || showValuesWithAxis ? 24 : 10,
            right: 20,
            bottom: 10,
            left: 10,
          },
        },
        interaction: {
          mode: "index",
          intersect: false,
        },
        plugins: {
          legend: {
            display: showLegend,
            position: legendPosition,
            labels: {
              color: CHART_PALETTE.text,
              usePointStyle: true,
              padding: 16,
              font: {
                size: 11,
                family: "'Inter', -apple-system, sans-serif",
              },
            },
          },
          tooltip: {
            enabled: true,
            backgroundColor: "rgba(30, 58, 95, 0.95)",
            titleFont: { size: 12, weight: "600" },
            bodyFont: { size: 11 },
            padding: 10,
            cornerRadius: 4,
            callbacks: {
              title: function (tooltipItems) {
                return tooltipItems[0].label;
              },
              label: function (context) {
                const decimals = getColumnDecimals(data, context.dataset.label);
                return (
                  context.dataset.label +
                  ": " +
                  formatNumber(context.parsed.y, { decimals })
                );
              },
            },
          },
        },
        scales: {
          y: {
            display: !(showValues && !showValuesWithAxis),
            type: "linear",
            position: "left",
            beginAtZero: true,
            border: { display: false },
            grid: {
              color: CHART_PALETTE.grid,
              lineWidth: 0.5,
              drawTicks: false,
              display: true,
            },
            ticks: {
              color: CHART_PALETTE.text,
              font: { size: 11, family: "'Inter', -apple-system, sans-serif" },
              padding: 8,
              display: true,
              callback: function (value) {
                return formatNumber(value, { decimals: yDecimals });
              },
            },
            ...(data.axisLabels && data.axisLabels.y
              ? {
                  title: {
                    display: showAxisTitles,
                    text: data.axisLabels.y,
                    color: CHART_PALETTE.text,
                    font: {
                      size: 12,
                      family: "'Inter', -apple-system, sans-serif",
                      weight: "500",
                    },
                    padding: { bottom: 8 },
                  },
                }
              : {}),
            ...(data.yLimit && typeof data.yLimit.min === "number"
              ? { min: data.yLimit.min }
              : {}),
            ...(data.yLimit && typeof data.yLimit.max === "number"
              ? { max: data.yLimit.max }
              : {}),
          },
          y1: {
            display: !(showValues && !showValuesWithAxis),
            type: "linear",
            position: "right",
            beginAtZero: true,
            border: { display: false },
            grid: { display: false },
            ticks: {
              color: CHART_PALETTE.text,
              font: { size: 11, family: "'Inter', -apple-system, sans-serif" },
              padding: 8,
              display: true,
              callback: function (value) {
                return formatNumber(value, { decimals: y1Decimals });
              },
            },
            ...(data.axisLabels && data.axisLabels.y1
              ? {
                  title: {
                    display: showAxisTitles,
                    text: data.axisLabels.y1,
                    color: CHART_PALETTE.text,
                    font: {
                      size: 12,
                      family: "'Inter', -apple-system, sans-serif",
                      weight: "500",
                    },
                    padding: { bottom: 8 },
                  },
                }
              : {}),
            ...(data.yLimit &&
            data.yLimit.y1 &&
            typeof data.yLimit.y1.min === "number"
              ? { min: data.yLimit.y1.min }
              : {}),
            ...(data.yLimit &&
            data.yLimit.y1 &&
            typeof data.yLimit.y1.max === "number"
              ? { max: data.yLimit.y1.max }
              : y1ComputedMax !== undefined
                ? { max: y1ComputedMax }
                : {}),
          },
          x: {
            border: { color: CHART_PALETTE.axis },
            grid: { display: false },
            ticks: {
              color: CHART_PALETTE.text,
              autoSkip: true,
              maxRotation: 45,
              minRotation: 0,
              padding: 6,
              font: { size: 11, family: "'Inter', -apple-system, sans-serif" },
              display: true,
            },
            ...(data.axisLabels && data.axisLabels.x
              ? {
                  title: {
                    display: showAxisTitles,
                    text: data.axisLabels.x,
                    color: CHART_PALETTE.text,
                    font: {
                      size: 12,
                      family: "'Inter', -apple-system, sans-serif",
                      weight: "500",
                    },
                    padding: { top: 8 },
                  },
                }
              : {}),
          },
        },
      },
      plugins: chartPlugins,
    });
  }); // end requestAnimationFrame
}

/**
 * Export table data to CSV
 */
function exportTableToCSV(headers, rows, filename) {
  // Build CSV content
  const csvRows = [];

  // Add headers
  csvRows.push(
    headers.map((h) => `"${String(h).replace(/"/g, '""')}"`).join(","),
  );

  // Add data rows
  rows.forEach((row) => {
    csvRows.push(
      row
        .map((cell) => {
          const cellStr =
            cell === null || cell === undefined ? "" : String(cell);
          return `"${cellStr.replace(/"/g, '""')}"`;
        })
        .join(","),
    );
  });

  const csvContent = csvRows.join("\n");
  const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);

  const link = document.createElement("a");
  link.setAttribute("href", url);
  link.setAttribute("download", filename || "table-export.csv");
  link.style.display = "none";
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Fire-and-forget CSV download tracking
 */
function trackCSVDownload(reportId, componentTitle, rowCount) {
  const base = (window.BASE_PATH || "") + "/api/metrics/csv";
  fetch(base, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      reportId: reportId || "",
      componentTitle: componentTitle || "",
      rowCount: rowCount || 0,
    }),
  }).catch(() => {});
}

/**
 * Return a CSS background color for a heatmap cell.
 * ratio: 0 (minimum) to 1 (maximum)
 * scale: "green-red", "red-green", or "blue"
 */
function heatmapColor(ratio, scale) {
  if (scale === "red-green") {
    return `hsl(${ratio * 120}, 55%, 88%)`;
  }
  if (scale === "blue") {
    return `rgba(49, 130, 189, ${(0.06 + ratio * 0.3).toFixed(2)})`;
  }
  // default: green-red
  return `hsl(${120 - ratio * 120}, 55%, 88%)`;
}

/**
 * Return a discrete background color for a cell based on per-column threshold rule.
 * rule: { thresholds: [lo, hi], colors: "red-yellow-green" | "green-yellow-red" }
 */
function heatmapRuleColor(value, rule) {
  const [lo, hi] = rule.thresholds;
  const red = "hsl(0, 55%, 88%)";
  const yellow = "hsl(48, 55%, 88%)";
  const green = "hsl(120, 55%, 88%)";

  if (rule.colors === "green-yellow-red") {
    if (value < lo) return green;
    if (value < hi) return yellow;
    return red;
  }
  // default: red-yellow-green
  if (value < lo) return red;
  if (value < hi) return yellow;
  return green;
}

/**
 * Map rule column names to header indices.
 * Returns { [colIndex]: rule } for valid rules (column found, thresholds length 2).
 * Matching is case-insensitive so manually-typed column names (e.g. table_advanced) still work.
 */
function buildHeatmapRuleIndex(rules, headers) {
  const index = {};
  const lowerHeaders = headers.map((h) => h.toLowerCase());
  for (const rule of rules) {
    if (
      !rule.column ||
      !Array.isArray(rule.thresholds) ||
      rule.thresholds.length !== 2
    )
      continue;
    const colIndex = lowerHeaders.indexOf(rule.column.toLowerCase());
    if (colIndex === -1) continue;
    index[colIndex] = rule;
  }
  return index;
}

/**
 * Parse a heatmap config from a component. Returns { scale }, { rules }, or null.
 */
function parseHeatmapConfig(heatmap) {
  if (!heatmap) return null;
  if (Array.isArray(heatmap.rules) && heatmap.rules.length > 0)
    return { rules: heatmap.rules };
  if (heatmap.scale) {
    const defaultScale = heatmap.scale;
    // columns may be strings (use default scale) or objects { name, scale }
    let columns = null;
    if (Array.isArray(heatmap.columns) && heatmap.columns.length > 0) {
      columns = heatmap.columns.map((c) =>
        typeof c === "string"
          ? { name: c, scale: defaultScale }
          : { name: c.name, scale: c.scale || defaultScale },
      );
    }
    return { scale: defaultScale, columns };
  }
  return null;
}

/**
 * Compute per-column min/max for numeric columns in one pass.
 * Returns { [colIndex]: { min, max } } for columns where max > min.
 */
function computeHeatmapStats(
  rows,
  isNumericFn,
  colCount,
  headers,
  columnConfigs,
  defaultScale,
) {
  const stats = {};
  // Build a lookup from header name → { scale } when per-column config is provided
  const colConfigByName = {};
  if (columnConfigs && headers) {
    columnConfigs.forEach((cfg) => {
      colConfigByName[cfg.name] = cfg;
    });
  }
  for (let c = 0; c < colCount; c++) {
    if (!isNumericFn(c)) continue;
    const headerName = headers ? headers[c] : null;
    // If an allow-list is configured, skip columns not in it
    if (columnConfigs && headerName && !colConfigByName[headerName]) continue;
    let min = Infinity,
      max = -Infinity;
    for (let r = 0; r < rows.length; r++) {
      const val = parseFloat(rows[r][c]);
      if (isNaN(val)) continue;
      if (val < min) min = val;
      if (val > max) max = val;
    }
    if (min < max) {
      const scale =
        (colConfigByName[headerName] && colConfigByName[headerName].scale) ||
        defaultScale;
      stats[c] = { min, max, scale };
    }
  }
  return stats;
}

/**
 * Render a table component with search, sorting, and pagination
 */
function getOrCreateHeaderTooltipPortal() {
  let tooltip = document.getElementById("header-tooltip-portal");
  if (tooltip) {
    return tooltip;
  }

  tooltip = document.createElement("div");
  tooltip.id = "header-tooltip-portal";
  tooltip.className = "header-tooltip-portal";
  tooltip.setAttribute("role", "tooltip");

  const content = document.createElement("div");
  content.className = "header-tooltip-portal-content";
  tooltip.appendChild(content);

  document.body.appendChild(tooltip);
  return tooltip;
}

function showHeaderTooltipPortal(target, text, formula) {
  if (!target || (!text && !formula)) {
    return;
  }
  showHeaderTooltipPortalRect(target.getBoundingClientRect(), text, formula);
}

function createAxisTooltipPlugin(component) {
  const tooltips = (component && component.axisTooltips) || {};
  const formulas = (component && component.axisFormulas) || {};
  const hasX = !!(tooltips.x || formulas.x);
  const hasY = !!(tooltips.y || formulas.y);
  if (!hasX && !hasY) {
    return null;
  }

  let currentAxis = null;
  const TITLE_THICKNESS = 28;

  return {
    id: "axisTitleTooltip",
    afterEvent(chart, args) {
      const e = args && args.event;
      if (!e) return;
      const type = e.type;
      if (type !== "mousemove" && type !== "mouseout" && type !== "mouseleave")
        return;

      if (type === "mouseout" || type === "mouseleave") {
        if (currentAxis) {
          hideHeaderTooltipPortal();
          currentAxis = null;
          if (chart.canvas) chart.canvas.style.cursor = "";
        }
        return;
      }

      const ca = chart.chartArea;
      const sx = chart.scales && chart.scales.x;
      const sy = chart.scales && chart.scales.y;
      if (!ca) return;

      let hit = null;
      if (hasX && sx) {
        const box = {
          left: ca.left,
          right: ca.right,
          top: sx.bottom - TITLE_THICKNESS,
          bottom: sx.bottom,
        };
        if (
          e.x >= box.left &&
          e.x <= box.right &&
          e.y >= box.top &&
          e.y <= box.bottom
        ) {
          hit = { axis: "x", box };
        }
      }
      if (!hit && hasY && sy) {
        const box = {
          left: sy.left,
          right: sy.left + TITLE_THICKNESS,
          top: ca.top,
          bottom: ca.bottom,
        };
        if (
          e.x >= box.left &&
          e.x <= box.right &&
          e.y >= box.top &&
          e.y <= box.bottom
        ) {
          hit = { axis: "y", box };
        }
      }

      if (hit) {
        if (currentAxis !== hit.axis) {
          currentAxis = hit.axis;
          const canvasRect = chart.canvas.getBoundingClientRect();
          const scaleX =
            (canvasRect.width / chart.canvas.width) *
            (window.devicePixelRatio || 1);
          const scaleY =
            (canvasRect.height / chart.canvas.height) *
            (window.devicePixelRatio || 1);
          // Chart.js box coords are in CSS pixels relative to canvas; no DPR scaling needed.
          const viewportRect = {
            left: canvasRect.left + hit.box.left,
            top: canvasRect.top + hit.box.top,
            right: canvasRect.left + hit.box.right,
            bottom: canvasRect.top + hit.box.bottom,
            width: hit.box.right - hit.box.left,
            height: hit.box.bottom - hit.box.top,
          };
          showHeaderTooltipPortalRect(
            viewportRect,
            tooltips[hit.axis] || "",
            formulas[hit.axis] || "",
          );
        }
        if (chart.canvas) chart.canvas.style.cursor = "help";
      } else if (currentAxis) {
        hideHeaderTooltipPortal();
        currentAxis = null;
        if (chart.canvas) chart.canvas.style.cursor = "";
      }
    },
  };
}

function showHeaderTooltipPortalRect(rect, text, formula) {
  if (!rect || (!text && !formula)) {
    return;
  }

  const tooltip = getOrCreateHeaderTooltipPortal();
  const content = tooltip.querySelector(".header-tooltip-portal-content");
  if (!content) {
    return;
  }

  content.innerHTML = "";

  if (text) {
    const textEl = document.createElement("div");
    textEl.className = "header-tooltip-text-section";
    textEl.textContent = text;
    content.appendChild(textEl);
  }

  if (formula) {
    if (text) {
      const divider = document.createElement("div");
      divider.className = "header-tooltip-divider";
      content.appendChild(divider);
    }
    const formulaSection = document.createElement("div");
    formulaSection.className = "header-tooltip-formula-section";
    formulaSection.innerHTML =
      '<strong style="display: block; font-size: 10px; letter-spacing: 0.5px; text-transform: uppercase; opacity: 0.7; margin-bottom: 4px;">Formula:</strong><code style="display: block; background: rgba(0, 0, 0, 0.2); padding: 6px; border-radius: 4px; font-family: monospace; font-size: 10px; white-space: normal; overflow-wrap: anywhere;"></code>';
    const codeEl = formulaSection.querySelector("code");
    // Escape, then insert <wbr> after arithmetic operators so long formulas
    // prefer to wrap right after a +, -, ×, ÷, *, /, %, or = symbol.
    const escaped = formula.replace(
      /[&<>"']/g,
      (c) =>
        ({
          "&": "&amp;",
          "<": "&lt;",
          ">": "&gt;",
          '"': "&quot;",
          "'": "&#39;",
        })[c],
    );
    codeEl.innerHTML = escaped.replace(/([+\-×÷*/%=])(\s*)/g, "$1$2<wbr>");
    content.appendChild(formulaSection);
  }

  tooltip.classList.add("is-visible");
  tooltip.style.left = "0px";
  tooltip.style.top = "0px";

  requestAnimationFrame(() => {
    const tooltipRect = tooltip.getBoundingClientRect();
    const gap = 12;
    const viewportPadding = 8;

    let left = rect.left + rect.width / 2 - tooltipRect.width / 2;
    left = Math.max(
      viewportPadding,
      Math.min(left, window.innerWidth - tooltipRect.width - viewportPadding),
    );

    let top = rect.top - tooltipRect.height - gap;
    top = Math.max(viewportPadding, top);

    const arrowLeft = Math.max(
      14,
      Math.min(rect.left + rect.width / 2 - left, tooltipRect.width - 14),
    );

    tooltip.style.left = `${left}px`;
    tooltip.style.top = `${top}px`;
    tooltip.style.setProperty("--header-tooltip-arrow-left", `${arrowLeft}px`);
  });
}

function hideHeaderTooltipPortal() {
  const tooltip = document.getElementById("header-tooltip-portal");
  if (!tooltip) {
    return;
  }
  tooltip.classList.remove("is-visible");
}

function resolveTablePageSize(pageSize) {
  const parsed = Math.floor(Number(pageSize));
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 20;
}

function renderTable(element, component) {
  const div = document.createElement("div");
  div.className = "component component-table";
  const headerWrapEnabled = !!component.tableHeaderWrap;
  const cellBoundariesEnabled = !!(
    component.tableCellBoundaries || component.tableVerticalLines
  );
  const headerStyle = component.tableHeaderStyle || "default";
  const normalizedHeaderStyle =
    headerStyle === "enhanced" ? "enhanced-readable" : headerStyle;
  if (headerWrapEnabled) {
    div.classList.add("table-header-wrap-enabled");
  }
  if (cellBoundariesEnabled) {
    div.classList.add("table-cell-boundaries-enabled");
    div.classList.add("table-vertical-lines-enabled");
  }
  if (
    normalizedHeaderStyle === "enhanced-compact" ||
    normalizedHeaderStyle === "enhanced-readable"
  ) {
    div.classList.add("table-header-style-enhanced");
    div.classList.add(`table-header-style-${normalizedHeaderStyle}`);
  }

  // Create header container for title and export button
  const headerContainer = document.createElement("div");
  headerContainer.className = "component-header";

  if (component.title) {
    const title = document.createElement("h3");
    title.className = "component-title";
    title.textContent = component.title;
    headerContainer.appendChild(title);
  }

  // Add download button
  const exportBtn = document.createElement("button");
  exportBtn.className = "table-export-btn";
  exportBtn.setAttribute("data-tooltip", "Download as CSV");
  exportBtn.innerHTML =
    '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>';
  exportBtn.addEventListener("click", async () => {
    const reportTitle =
      typeof currentReportMetadata !== "undefined" &&
      currentReportMetadata?.title
        ? currentReportMetadata.title
        : "";
    const tableTitle = component.title || "";
    const baseName = reportTitle
      ? tableTitle
        ? `${reportTitle} - ${tableTitle}`
        : reportTitle
      : tableTitle || "table";
    const filename =
      baseName
        .replace(/[^a-z0-9]+/gi, "-")
        .replace(/^-|-$/g, "")
        .toLowerCase() + "-table.csv";

    // For table_advanced components use the server-side unlimited download endpoint
    if (
      component.type === "table_advanced" &&
      component._sectionId != null &&
      typeof currentReportId !== "undefined" &&
      currentReportId
    ) {
      try {
        exportBtn.disabled = true;
        const params = new URLSearchParams(window.location.search);
        params.set("report", currentReportId);
        params.set("sectionId", component._sectionId);
        params.set("componentIndex", String(component._componentIndex));

        const downloadUrl = `${API_BASE}/download/csv?${params.toString()}`;
        const token =
          typeof window.getAuthToken === "function"
            ? await window.getAuthToken()
            : null;
        const fetchOpts = token
          ? { headers: { Authorization: `Bearer ${token}` } }
          : {};
        const resp = await fetch(downloadUrl, fetchOpts);
        if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
        const blob = await resp.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        trackCSVDownload(currentReportId, component.title, null);
      } catch (err) {
        console.error(
          "[Download] CSV server download failed, falling back to client export:",
          err,
        );
        exportTableToCSV(headers, filteredRows, filename);
        trackCSVDownload(currentReportId, component.title, filteredRows.length);
      } finally {
        exportBtn.disabled = false;
      }
      return;
    }

    exportTableToCSV(headers, filteredRows, filename);
    trackCSVDownload(currentReportId, component.title, filteredRows.length);
  });
  headerContainer.appendChild(exportBtn);

  div.appendChild(headerContainer);

  if (component.description) {
    const desc = document.createElement("p");
    desc.className = "component-description";
    desc.textContent = component.description;
    div.appendChild(desc);
  }

  const originalRows =
    component.data && component.data.rows ? component.data.rows : [];
  const headers =
    component.data && component.data.headers ? component.data.headers : [];
  const headerTooltips = component.headerTooltips || {};
  const headerFormulas = component.headerFormulas || {};

  function normalizeSplitColumnGroups(rawGroups, headerList) {
    if (
      !Array.isArray(rawGroups) ||
      rawGroups.length === 0 ||
      headerList.length === 0
    ) {
      return [];
    }

    function getChildTailLabel(child) {
      const normalized = String(child || "").trim();
      if (normalized.endsWith("%")) {
        return "%";
      }
      if (normalized.toLowerCase().endsWith("emergency cases")) {
        return "Emergency Cases";
      }
      if (normalized.toLowerCase().endsWith("total")) {
        return "Total";
      }
      return normalized;
    }

    function findContiguousByTail(children, usedIndicesSet) {
      const tails = children.map(getChildTailLabel);
      const span = tails.length;
      for (let start = 0; start <= headerList.length - span; start++) {
        const windowIndices = Array.from(
          { length: span },
          (_, offset) => start + offset,
        );
        const overlaps = windowIndices.some((idx) => usedIndicesSet.has(idx));
        if (overlaps) {
          continue;
        }

        const matches = windowIndices.every((idx, offset) => {
          const header = String(headerList[idx] || "").trim();
          const tail = tails[offset];
          if (!tail) {
            return false;
          }
          if (tail === "%") {
            return header.endsWith("%");
          }
          return header.toLowerCase().endsWith(tail.toLowerCase());
        });

        if (matches) {
          return windowIndices.map((index, offset) => ({
            child: children[offset],
            index,
          }));
        }
      }
      return [];
    }

    const groups = [];
    const usedIndices = new Set();

    rawGroups.forEach((rawGroup) => {
      const parent =
        typeof rawGroup?.parent === "string" ? rawGroup.parent.trim() : "";
      const children = Array.isArray(rawGroup?.children)
        ? rawGroup.children
            .map((child) => String(child || "").trim())
            .filter(Boolean)
        : [];

      if (!parent || children.length === 0) {
        return;
      }

      const indexed = children
        .map((child) => ({ child, index: headerList.indexOf(child) }))
        .filter((entry) => entry.index >= 0)
        .sort((a, b) => a.index - b.index);

      const resolved =
        indexed.length === children.length
          ? indexed
          : findContiguousByTail(children, usedIndices);

      if (resolved.length === 0) {
        return;
      }

      const contiguous = resolved.every(
        (entry, idx) =>
          idx === 0 || entry.index === resolved[idx - 1].index + 1,
      );
      if (!contiguous) {
        return;
      }

      const overlaps = resolved.some((entry) => usedIndices.has(entry.index));
      if (overlaps) {
        return;
      }

      resolved.forEach((entry) => usedIndices.add(entry.index));

      groups.push({
        parent,
        firstIndex: resolved[0].index,
        childIndices: resolved.map((entry) => entry.index),
        childSet: new Set(resolved.map((entry) => entry.index)),
        span: resolved.length,
      });
    });

    return groups.sort((a, b) => a.firstIndex - b.firstIndex);
  }

  const splitColumnGroups = normalizeSplitColumnGroups(
    component.splitColumns,
    headers,
  );
  const hasAgeGroupsParent = splitColumnGroups.some(
    (group) =>
      String(group.parent || "")
        .trim()
        .toLowerCase() === "age groups",
  );

  // State management
  let searchTerm = "";
  let sortColumn = null;
  let sortDirection = "asc"; // 'asc' or 'desc'
  let filteredRows = [...originalRows];

  // Pagination state
  const PAGE_SIZE = resolveTablePageSize(component.pageSize);
  let currentPage = 1;

  // Detect if a column contains mostly numbers
  function isNumericColumn(colIndex) {
    const sampleSize = Math.min(10, originalRows.length);
    let numericCount = 0;
    for (let i = 0; i < sampleSize; i++) {
      const val = originalRows[i][colIndex];
      const trimmed = String(val).trim();
      if (trimmed !== "" && !isNaN(Number(trimmed))) {
        numericCount++;
      }
    }
    return numericCount / sampleSize > 0.7; // If 70%+ are numbers
  }

  // Filter rows based on search term
  function filterRows(term) {
    if (!term.trim()) {
      return [...originalRows];
    }
    const lowerTerm = term.toLowerCase();
    return originalRows.filter((row) =>
      row.some((cell) => String(cell).toLowerCase().includes(lowerTerm)),
    );
  }

  // Sort rows by column
  function sortRows(rows, colIndex, direction) {
    if (colIndex === null) return rows;

    const isNumeric = isNumericColumn(colIndex);
    const sorted = [...rows].sort((a, b) => {
      const aVal = a[colIndex];
      const bVal = b[colIndex];

      if (isNumeric) {
        const aNum = parseFloat(aVal) || 0;
        const bNum = parseFloat(bVal) || 0;
        return direction === "asc" ? aNum - bNum : bNum - aNum;
      } else {
        const aStr = aVal.toString().toLowerCase();
        const bStr = bVal.toString().toLowerCase();
        return direction === "asc"
          ? aStr.localeCompare(bStr)
          : bStr.localeCompare(aStr);
      }
    });
    return sorted;
  }

  // Apply filter and sort to get final display rows
  function getDisplayRows() {
    let rows = filterRows(searchTerm);
    rows = sortRows(rows, sortColumn, sortDirection);
    return rows;
  }

  // Get rows for current page (PDF mode bypasses pagination to show all rows)
  function getPageRows() {
    if (div._pdfMode) return filteredRows;
    const startIndex = (currentPage - 1) * PAGE_SIZE;
    const endIndex = startIndex + PAGE_SIZE;
    return filteredRows.slice(startIndex, endIndex);
  }

  // Get total pages
  function getTotalPages() {
    return Math.ceil(filteredRows.length / PAGE_SIZE);
  }

  // Heatmap config (parsed once)
  const heatmapCfg = parseHeatmapConfig(component.heatmap);

  // Keep column widths visually consistent: use the widest unwrapped header
  // as a shared baseline so wrapped and unwrapped columns align cleanly.
  // On narrow viewports, skip equal-percentage distribution and let the browser
  // size columns naturally (gives the first text column more room).
  function normalizeColumnWidths(tableEl) {
    if (!headerWrapEnabled) {
      return;
    }
    if (window.innerWidth < 600) {
      return;
    }

    const headerCells = Array.from(
      tableEl.querySelectorAll("thead th.sortable-header"),
    );
    if (headerCells.length === 0) {
      return;
    }

    headerCells.forEach((th) => {
      th.style.width = "";
      th.style.minWidth = "";
    });

    tableEl.style.tableLayout = "fixed";
    tableEl.style.width = "100%";

    const equalWidth = `${(100 / headerCells.length).toFixed(4)}%`;
    headerCells.forEach((th) => {
      th.style.width = equalWidth;
      th.style.minWidth = "0";
    });
  }

  // Function to render table content
  function renderTableContent() {
    // Compute heatmap data from full filteredRows (not page rows) for consistent scale
    let heatmapStats = null;
    let heatmapRuleIndex = null;
    if (heatmapCfg) {
      if (heatmapCfg.rules) {
        heatmapRuleIndex = buildHeatmapRuleIndex(heatmapCfg.rules, headers);
      } else {
        heatmapStats = computeHeatmapStats(
          filteredRows,
          isNumericColumn,
          headers.length,
          headers,
          heatmapCfg.columns,
          heatmapCfg.scale,
        );
      }
    }

    // Remove existing table and pagination if any
    const existingTable = div.querySelector("table");
    if (existingTable) {
      existingTable.remove();
    }
    const existingPagination = div.querySelector(".table-pagination");
    if (existingPagination) {
      existingPagination.remove();
    }

    const table = document.createElement("table");

    // Create header with clickable columns
    const thead = document.createElement("thead");

    function buildSortableHeaderCell(header, colIndex, displayLabel = null) {
      const th = document.createElement("th");
      th.className = "sortable-header";
      th.style.cursor = "pointer";
      th.setAttribute("data-col-index", colIndex);

      if (isNumericColumn(colIndex)) {
        th.classList.add("col-numeric");
      } else {
        th.classList.add("col-text");
      }

      const tooltipText = headerTooltips[header] || "";
      const formulaText = headerFormulas[header] || "";
      if (tooltipText || formulaText) {
        th.classList.add("has-header-tooltip");
        if (tooltipText) {
          th.setAttribute("data-header-tooltip", tooltipText);
        }
        if (formulaText) {
          th.setAttribute("data-header-formula", formulaText);
        }
        th.setAttribute("tabindex", "0");
      }

      const headerText = document.createElement("span");
      headerText.className = "table-header-label";
      const effectiveLabel = displayLabel || header;
      headerText.textContent = effectiveLabel;
      if (headerWrapEnabled && String(header).trim().length >= 16) {
        headerText.classList.add("header-long");
        th.classList.add("header-long-col");
      }

      const headerMeta = document.createElement("span");
      headerMeta.className = "table-header-meta";
      headerMeta.appendChild(headerText);

      if (tooltipText || formulaText) {
        const infoIcon = document.createElement("span");
        infoIcon.className = "header-info-icon";
        infoIcon.setAttribute("aria-hidden", "true");
        infoIcon.innerHTML =
          '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/><line x1="12" y1="8" x2="12" y2="8"/><line x1="12" y1="12" x2="12" y2="16"/></svg>';
        headerMeta.appendChild(infoIcon);
      }

      th.appendChild(headerMeta);

      const indicator = document.createElement("span");
      indicator.className = "sort-indicator";
      if (sortColumn === colIndex) {
        indicator.textContent = sortDirection === "asc" ? " ▲" : " ▼";
        indicator.style.color = "var(--accent-color)";
        th.style.backgroundColor = "rgba(191, 160, 148, 0.15)";
      } else {
        indicator.textContent = " ⋮";
        indicator.style.opacity = "0.3";
      }
      th.appendChild(indicator);

      return th;
    }

    function getSplitChildDisplayLabel(header, colIndex) {
      const childHeaderMode = String(
        component.splitChildHeaderMode || "symbols",
      ).toLowerCase();
      if (hasAgeGroupsParent) {
        return header;
      }

      if (childHeaderMode === "labels") {
        const normalized = String(header || "").trim();
        const group = splitColumnGroups.find((candidate) =>
          candidate.childSet.has(colIndex),
        );
        if (group) {
          const parent = String(group.parent || "").trim();
          if (parent) {
            const prefix = `${parent} `;
            if (normalized.toLowerCase().startsWith(prefix.toLowerCase())) {
              return normalized.slice(prefix.length).trim() || header;
            }
          }
        }

        // Fallback: strip leading period token when parent and child period text differ
        // (for example parent "Q4 2025" with child "Q4 2024 Total").
        const periodPrefixMatch = normalized.match(
          /^(Q[1-4]\s+\d{4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{4})\s+(.+)$/i,
        );
        if (periodPrefixMatch && periodPrefixMatch[2]) {
          return periodPrefixMatch[2].trim();
        }

        return header;
      }

      const normalized = String(header || "").trim();
      return normalized.includes("%") ? "%" : "Total";
    }

    if (splitColumnGroups.length > 0) {
      const topRow = document.createElement("tr");
      topRow.className = "split-header-parent-row";
      const childRow = document.createElement("tr");
      childRow.className = "split-header-child-row";

      const groupByFirstIndex = new Map(
        splitColumnGroups.map((group) => [group.firstIndex, group]),
      );

      headers.forEach((header, colIndex) => {
        const startsGroup = groupByFirstIndex.get(colIndex);
        const inAnyGroup = splitColumnGroups.some((group) =>
          group.childSet.has(colIndex),
        );

        if (startsGroup) {
          const parentTh = document.createElement("th");
          parentTh.className = "split-parent-header";
          parentTh.colSpan = startsGroup.span;

          const parentLabel = document.createElement("span");
          parentLabel.className = "table-header-label";
          parentLabel.textContent = startsGroup.parent;
          parentTh.appendChild(parentLabel);

          topRow.appendChild(parentTh);
          childRow.appendChild(
            buildSortableHeaderCell(
              header,
              colIndex,
              getSplitChildDisplayLabel(header, colIndex),
            ),
          );
          return;
        }

        if (inAnyGroup) {
          childRow.appendChild(
            buildSortableHeaderCell(
              header,
              colIndex,
              getSplitChildDisplayLabel(header, colIndex),
            ),
          );
          return;
        }

        const th = buildSortableHeaderCell(header, colIndex);
        th.rowSpan = 2;
        topRow.appendChild(th);
      });

      thead.appendChild(topRow);
      thead.appendChild(childRow);
    } else {
      const headerRow = document.createElement("tr");
      headers.forEach((header, colIndex) => {
        headerRow.appendChild(buildSortableHeaderCell(header, colIndex));
      });
      thead.appendChild(headerRow);
    }

    table.appendChild(thead);

    // Create body with paginated rows
    const tbody = document.createElement("tbody");
    const pageRows = getPageRows();

    if (filteredRows.length === 0) {
      const tr = document.createElement("tr");
      const td = document.createElement("td");
      td.colSpan = headers.length;
      td.style.textAlign = "center";
      td.style.padding = "30px 20px";
      td.innerHTML =
        '<p style="margin:0 0 4px 0;color:var(--text-secondary);font-weight:500;">No records to display</p><p style="margin:0;color:var(--text-light);font-size:0.9em;">This may be due to the selected filters</p>';
      tr.appendChild(td);
      tbody.appendChild(tr);
    } else {
      pageRows.forEach((row) => {
        const tr = document.createElement("tr");
        row.forEach((cell, colIndex) => {
          const td = document.createElement("td");
          if (isNumericColumn(colIndex)) {
            td.classList.add("col-numeric");
          } else {
            td.classList.add("col-text");
          }
          // Format numeric cells with thousand separators.
          // Per-column decimals come from the server's columnFormats (author's roundTo).
          if (isNumericColumn(colIndex)) {
            const decimals = getColumnDecimals(
              component.data,
              headers[colIndex],
            );
            td.textContent = formatNumber(cell, { decimals });
          } else {
            td.textContent = cell;
          }
          // Apply heatmap coloring
          if (heatmapRuleIndex && heatmapRuleIndex[colIndex]) {
            const val = parseFloat(cell);
            if (!isNaN(val)) {
              td.style.backgroundColor = heatmapRuleColor(
                val,
                heatmapRuleIndex[colIndex],
              );
            }
          } else if (heatmapStats && heatmapStats[colIndex]) {
            const val = parseFloat(cell);
            if (!isNaN(val)) {
              const { min, max, scale } = heatmapStats[colIndex];
              const ratio = (val - min) / (max - min);
              td.style.backgroundColor = heatmapColor(ratio, scale);
            }
          }
          tr.appendChild(td);
        });
        tbody.appendChild(tr);
      });
    }
    table.appendChild(tbody);
    div.appendChild(table);
    normalizeColumnWidths(table);

    // Add pagination controls if more than one page
    const totalPages = getTotalPages();
    if (totalPages > 1) {
      const paginationDiv = document.createElement("div");
      paginationDiv.className = "table-pagination";

      // Previous button
      const prevBtn = document.createElement("button");
      prevBtn.className = "pagination-btn";
      prevBtn.setAttribute("data-page-action", "prev");
      prevBtn.textContent = "← Previous";
      prevBtn.disabled = currentPage === 1;
      // Click handled by delegated listener on div
      paginationDiv.appendChild(prevBtn);

      // Page info — long form on desktop, short form on mobile via CSS
      const pageInfo = document.createElement("span");
      pageInfo.className = "pagination-info";
      const startRow = (currentPage - 1) * PAGE_SIZE + 1;
      const endRow = Math.min(currentPage * PAGE_SIZE, filteredRows.length);
      pageInfo.innerHTML = `<span class="pag-long">${startRow}\u2013${endRow} of ${filteredRows.length} rows (Page ${currentPage} of ${totalPages})</span><span class="pag-short">Pg ${currentPage}/${totalPages}</span>`;
      paginationDiv.appendChild(pageInfo);

      // Next button
      const nextBtn = document.createElement("button");
      nextBtn.className = "pagination-btn";
      nextBtn.setAttribute("data-page-action", "next");
      nextBtn.textContent = "Next →";
      nextBtn.disabled = currentPage === totalPages;
      // Click handled by delegated listener on div
      paginationDiv.appendChild(nextBtn);

      div.appendChild(paginationDiv);
    }
  }

  // Delegated click listener on the stable wrapper div for sort + pagination
  div.addEventListener("click", (e) => {
    // Sort header click
    const th = e.target.closest(".sortable-header");
    if (th) {
      const colIndex = parseInt(th.getAttribute("data-col-index"));
      if (!isNaN(colIndex)) {
        if (sortColumn === colIndex) {
          sortDirection = sortDirection === "asc" ? "desc" : "asc";
        } else {
          sortColumn = colIndex;
          sortDirection = "asc";
        }
        filteredRows = getDisplayRows();
        currentPage = 1;
        renderTableContent();
      }
      return;
    }

    // Pagination button click
    const pageBtn = e.target.closest(".pagination-btn");
    if (pageBtn && !pageBtn.disabled) {
      const action = pageBtn.getAttribute("data-page-action");
      if (action === "prev" && currentPage > 1) {
        currentPage--;
        renderTableContent();
      } else if (action === "next" && currentPage < getTotalPages()) {
        currentPage++;
        renderTableContent();
      }
    }
  });

  // Touch: tap a tooltip-able header to pin/dismiss the tooltip portal.
  // Uses { passive: false } so e.preventDefault() can suppress the sort tap.
  div.addEventListener(
    "touchstart",
    (e) => {
      const header = e.target.closest(".sortable-header.has-header-tooltip");
      if (!header || !div.contains(header)) {
        return;
      }
      e.preventDefault();
      const alreadyOpen = header.classList.contains("tooltip-touch-open");
      div
        .querySelectorAll(".tooltip-touch-open")
        .forEach((h) => h.classList.remove("tooltip-touch-open"));
      hideHeaderTooltipPortal();
      if (!alreadyOpen) {
        header.classList.add("tooltip-touch-open");
        showHeaderTooltipPortal(
          header,
          header.getAttribute("data-header-tooltip"),
          header.getAttribute("data-header-formula"),
        );
      }
    },
    { passive: false },
  );

  div.addEventListener("mouseover", (e) => {
    const header = e.target.closest(".sortable-header.has-header-tooltip");
    if (!header || !div.contains(header)) {
      return;
    }
    showHeaderTooltipPortal(
      header,
      header.getAttribute("data-header-tooltip"),
      header.getAttribute("data-header-formula"),
    );
  });

  div.addEventListener("mouseout", (e) => {
    const header = e.target.closest(".sortable-header.has-header-tooltip");
    if (!header) {
      return;
    }
    const nextTarget = e.relatedTarget;
    if (nextTarget && header.contains(nextTarget)) {
      return;
    }
    hideHeaderTooltipPortal();
  });

  div.addEventListener("focusin", (e) => {
    const header = e.target.closest(".sortable-header.has-header-tooltip");
    if (!header || !div.contains(header)) {
      return;
    }
    showHeaderTooltipPortal(
      header,
      header.getAttribute("data-header-tooltip"),
      header.getAttribute("data-header-formula"),
    );
  });

  div.addEventListener("focusout", (e) => {
    const header = e.target.closest(".sortable-header.has-header-tooltip");
    if (!header) {
      return;
    }
    const nextTarget = e.relatedTarget;
    if (nextTarget && header.contains(nextTarget)) {
      return;
    }
    hideHeaderTooltipPortal();
  });

  div.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      hideHeaderTooltipPortal();
    }
  });

  // Initial render
  filteredRows = getDisplayRows();
  renderTableContent();

  // PDF hooks: expand all rows / restore normal pagination
  div._pdfRenderAll = () => {
    if (filteredRows.length <= PAGE_SIZE) return;
    div._pdfMode = true;
    renderTableContent();
  };
  div._pdfRestorePage = () => {
    if (!div._pdfMode) return;
    div._pdfMode = false;
    renderTableContent();
  };

  element.appendChild(div);
}

/**
 * Render a map component using Leaflet.js
 */
function renderMap(element, component) {
  const div = document.createElement("div");
  div.className = "component component-map";

  if (component.title) {
    const title = document.createElement("h3");
    title.className = "component-title";
    title.textContent = component.title;
    div.appendChild(title);
  }

  if (component.description) {
    const desc = document.createElement("p");
    desc.className = "component-description";
    desc.textContent = component.description;
    div.appendChild(desc);
  }

  const unmappedFacilities = component.data?.unmapped || [];
  if (unmappedFacilities.length > 0) {
    const details = document.createElement("details");
    details.className = "facility-map-unmapped";

    const summary = document.createElement("summary");
    summary.textContent = `${unmappedFacilities.length} ${unmappedFacilities.length === 1 ? "facility has" : "facilities have"} no verified coordinates — view totals`;
    details.appendChild(summary);

    const list = document.createElement("div");
    list.className = "facility-map-unmapped-list";
    unmappedFacilities.forEach((facility) => {
      const row = document.createElement("div");
      row.className = "facility-map-unmapped-row";
      const location = facility.district
        ? `${facility.label} (${facility.district})`
        : facility.label;
      row.textContent = `${location}: ${formatNumber(facility.value)} screened`;
      list.appendChild(row);
    });
    details.appendChild(list);
    div.appendChild(details);
  }

  const mapDiv = document.createElement("div");
  mapDiv.id = "map-" + Math.random().toString(36).substr(2, 9);
  mapDiv.style.width = "100%";
  mapDiv.style.height = "100%"; // Fill container - CSS controls the height
  div.appendChild(mapDiv);
  element.appendChild(div);

  // Initialize an interactive map so users can inspect a specific area.
  setTimeout(() => {
    const data = component.data;
    const map = L.map(mapDiv.id, {
      dragging: true,
      scrollWheelZoom: true,
      doubleClickZoom: true,
      boxZoom: true,
      touchZoom: true,
      tap: true,
      keyboard: true,
      zoomControl: true,
    }).setView(data.center, data.zoom);

    mapDiv.setAttribute(
      "aria-label",
      "Interactive facility map. Drag to move, scroll or use the controls to zoom, and select a marker for details.",
    );

    // Add tile layer
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "© OpenStreetMap contributors",
      maxZoom: 19,
    }).addTo(map);

    const numericValues = (data.markers || [])
      .map((marker) => Number(marker.value))
      .filter(Number.isFinite);
    const maxValue = numericValues.length ? Math.max(...numericValues) : 1;

    const markerGroup = L.featureGroup().addTo(map);

    // Add proportional facility markers
    (data.markers || []).forEach((marker) => {
      const value = Number(marker.value) || 0;
      const ratio = maxValue > 0 ? value / maxValue : 0;
      const color = ratio >= 0.66 ? "#165c92" : ratio >= 0.33 ? "#f59e0b" : "#ef4444";
      const radius = 6 + Math.sqrt(ratio) * 16;
      const facilityMarker = L.circleMarker([marker.lat, marker.lng], {
        radius,
        color: "#ffffff",
        weight: 1.5,
        fillColor: color,
        fillOpacity: 0.8,
      })
        .bindTooltip(
          `<strong>${marker.label}</strong><br>${data.value_label || "Value"}: ${formatNumber(marker.value)}`,
          {
            direction: "top",
            offset: [0, -6],
            className: "facility-map-label",
          },
        )
        .bindPopup(
          `<strong>${marker.label}</strong><br/>${data.value_label || "Value"}: ${formatNumber(marker.value)}`,
        )
        .on("click", (event) => {
          map.flyTo(event.latlng, Math.max(map.getZoom(), 11), {
            duration: 0.45,
          });
        })
        .addTo(markerGroup);

      facilityMarker._facilityLabel = true;
    });

    map.invalidateSize();
    if (markerGroup.getLayers().length > 0) {
      map.fitBounds(markerGroup.getBounds(), { padding: [28, 28], maxZoom: 12 });
    }

    // Reveal facility names and totals as users zoom into a particular area.
    // At wider views, labels remain available on hover to avoid overlap.
    const updateFacilityLabels = () => {
      const showLabels = map.getZoom() >= 10;
      markerGroup.eachLayer((layer) => {
        if (!layer._facilityLabel) return;
        if (showLabels) layer.openTooltip();
        else layer.closeTooltip();
      });
    };
    map.on("zoomend moveend", updateFacilityLabels);
    updateFacilityLabels();

    L.control.scale({ imperial: false, position: "bottomleft" }).addTo(map);
  }, 100);
}

/**
 * Get color based on value (for markers)
 */
function getMarkerColor(value) {
  if (value >= 85) return "#27ae60"; // Green
  if (value >= 70) return "#f39c12"; // Orange
  if (value >= 50) return "#e67e22"; // Dark orange
  return "#e74c3c"; // Red
}

/**
 * Render a choropleth map using Leaflet.js
 */
function renderChoropleth(element, component) {
  const div = document.createElement("div");
  div.className = "component component-choropleth";

  if (component.title) {
    const title = document.createElement("h3");
    title.className = "component-title";
    title.textContent = component.title;
    div.appendChild(title);
  }

  if (component.description) {
    const desc = document.createElement("p");
    desc.className = "component-description";
    desc.textContent = component.description;
    div.appendChild(desc);
  }

  const mapDiv = document.createElement("div");
  mapDiv.id = "choropleth-" + Math.random().toString(36).substr(2, 9);
  mapDiv.className = "choropleth-map-area";
  mapDiv.style.backgroundColor = "transparent";
  div.appendChild(mapDiv);

  // Legend bar below the map (populated after data loads)
  const legendBar = document.createElement("div");
  legendBar.className = "choropleth-legend-bar";
  div.appendChild(legendBar);

  element.appendChild(div);

  setTimeout(() => {
    // Initialize global map storage if needed
    if (!window.choroplethMaps) {
      window.choroplethMaps = {};
    }

    // No fixed center/zoom - will auto-fit to data with fitBounds()
    const map = L.map(mapDiv.id, {
      dragging: false,
      scrollWheelZoom: false,
      doubleClickZoom: false,
      boxZoom: false,
      touchZoom: false,
      tap: false,
      keyboard: false,
      zoomControl: false,
      attributionControl: false, // Clean up UI
      zoomSnap: 0, // Enable fractional zoom (decimal zoom levels like 6.4)
      zoomDelta: 1, // Allow smooth zoom transitions
      renderer: L.canvas(), // Canvas renderer: 1 canvas element instead of 146 SVG paths
    });

    // Store reference
    window.choroplethMaps[mapDiv.id] = map;

    const isCategorical = component.valueType === "categorical";

    if (!component.data || !component.data.geojson_path) {
      console.error("Choropleth missing data.geojson_path:", component.title);
      div.innerHTML = `<div style="padding:24px;text-align:center;font-size:0.85em;color:var(--color-error,#dc2626);">Map configuration incomplete (missing GeoJSON path)</div>`;
      return;
    }

    const isUganda = component.data.geojson_path && /uganda/i.test(component.data.geojson_path);
    Promise.all([
      fetchGeoJSON(component.data.geojson_path),
      isUganda ? fetchGeoJSON("/assets/uganda_outline.geojson") : Promise.resolve(null),
      isUganda ? fetchGeoJSON("/assets/uganda_regions.geojson") : Promise.resolve(null),
    ])
      .then(([geojson, outlineGeoJSON, regionsGeoJSON]) => {
        const numBins = component.data.color_scheme?.length || 3;
        const bins = isCategorical
          ? null
          : calculateQuantileBins(component.data.district_values, numBins);
        const customEdges = isCategorical ? null : component.data.bin_edges;
        const categoryColorMap = isCategorical
          ? buildCategoryColorMap(
              component.data.bin_labels,
              component.data.color_scheme,
            )
          : null;

        // Layer 1: Country outline with water-blue fill (bottom)
        if (outlineGeoJSON) {
        L.geoJSON(outlineGeoJSON, {
          style: {
            fillColor: "#B3D4E8",
            fillOpacity: 1,
            color: "#999",
            weight: 0.5,
            opacity: 0.5,
          },
        }).addTo(map);

        // Layer 2: District polygons colored by data (middle)
        const styleFn = isCategorical
          ? (feature) =>
              styleCategoricalDistrict(
                feature,
                component.data.district_values,
                categoryColorMap,
              )
          : (feature) =>
              styleDistrict(
                feature,
                component.data.district_values,
                bins,
                component.data.color_scheme,
                customEdges,
              );

        const geoJsonLayer = L.geoJSON(geojson, {
          style: styleFn,
          onEachFeature: (feature, layer) => {
            const districtName = geoFeatureName(feature);
            const value =
              component.data.district_values == null
                ? undefined
                : component.data.district_values[districtName];
            const valueStr =
              value !== null && value !== undefined
                ? isCategorical
                  ? value
                  : formatNumber(value)
                : "No data";

            layer.bindTooltip(
              `<strong>${districtName}</strong><br/>${valueStr}`,
              {
                permanent: false,
                direction: "top",
                className: "choropleth-tooltip",
              },
            );

            layer.bindPopup(
              `<strong>${districtName}</strong><br/>Value: ${valueStr}`,
            );

            layer.on("mouseover", function () {
              this.setStyle({ weight: 2, color: "#333" });
              this.openTooltip();
            });
            layer.on("mouseout", function () {
              this.setStyle({ weight: 0.5, color: "#666" });
              this.closeTooltip();
            });
          },
        }).addTo(map);

        // Layer 3: Region boundaries (top) - black lines, thick
        if (regionsGeoJSON) {
        L.geoJSON(regionsGeoJSON, {
          style: { fill: false, color: "#000", weight: 1.5, opacity: 1 },
          interactive: false,
        }).addTo(map);

        // AUTO-FIT: This is the magic - fits Uganda perfectly to the container
        map.fitBounds(geoJsonLayer.getBounds(), {
          paddingTopLeft: [20, 20],
          paddingBottomRight: [20, 20],
        });

        if (isCategorical) {
          renderCategoricalLegendBar(
            legendBar,
            component.data.bin_labels,
            component.data.color_scheme,
            component.data.legend_title,
          );
        } else {
          renderChoroplethLegendBar(
            legendBar,
            bins,
            component.data.color_scheme,
            component.data.bin_labels,
            component.data.legend_title,
            customEdges,
          );
        }
        map.invalidateSize();
      })
      .catch((error) => {
        console.error("Error loading GeoJSON:", error);
        div.innerHTML = `<div style="padding:24px;text-align:center;font-size:0.85em;color:var(--color-error,#dc2626);">Map data failed to load</div>`;
      });

    map.invalidateSize();
  }, 100);
}

/**
 * Calculate quantile bins from district values
 */
function calculateQuantileBins(districtValues, numBins = 3) {
  const fallback = new Array(numBins - 1).fill(0);
  if (!districtValues) return fallback;
  const values = Object.values(districtValues)
    .filter((v) => v !== null && v !== undefined)
    .map((v) => parseFloat(v))
    .sort((a, b) => a - b);

  if (values.length === 0) return fallback;

  const thresholds = [];
  for (let i = 1; i < numBins; i++) {
    const idx = Math.floor((i * values.length) / numBins);
    thresholds.push(values[Math.min(idx, values.length - 1)]);
  }
  return thresholds;
}

/**
 * Get style for a district polygon
 * @param {Object} feature - GeoJSON feature
 * @param {Object} districtValues - Map of district names to values
 * @param {Array} bins - Auto-calculated quantile bins [q1, q2, max]
 * @param {Array} colorScheme - Array of 3 colors for low/mid/high
 * @param {Array} customEdges - Optional custom bin edges [threshold1, threshold2]
 */
function styleDistrict(
  feature,
  districtValues,
  bins,
  colorScheme,
  customEdges,
) {
  const districtName = geoFeatureName(feature);
  const value =
    districtValues == null ? undefined : districtValues[districtName];

  const scheme =
    colorScheme && colorScheme.length > 0
      ? colorScheme
      : ["#e8e8e8", "#9e9e9e", "#424242"];

  // Use custom edges if provided, otherwise use auto-calculated bins
  const thresholds =
    customEdges && customEdges.length >= 2 ? customEdges : bins;

  const hasData = value !== null && value !== undefined;

  if (!hasData) {
    return {
      fillColor: "#f5f5f5",
      fillOpacity: 1,
      color: "#666",
      weight: 0.5,
      opacity: 1,
    };
  }

  let fillColor = scheme[scheme.length - 1];
  const numValue = parseFloat(value);
  for (let i = 0; i < thresholds.length; i++) {
    if (numValue <= thresholds[i]) {
      fillColor = scheme[i];
      break;
    }
  }

  return {
    fillColor: fillColor,
    fillOpacity: 0.7,
    color: "#666",
    weight: 0.5,
    opacity: 1,
  };
}

/**
 * Render a horizontal legend bar below a choropleth map
 * @param {HTMLElement} container - Legend bar div to populate
 * @param {Array} bins - Auto-calculated quantile bins [q1, q2, max]
 * @param {Array} colorScheme - Array of 3 colors for low/mid/high
 * @param {Array} binLabels - Custom labels for bins
 * @param {string} legendTitle - Title for the legend
 * @param {Array} customEdges - Optional custom bin edges [threshold1, threshold2]
 */
function renderChoroplethLegendBar(
  container,
  bins,
  colorScheme,
  binLabels,
  legendTitle,
  customEdges,
) {
  const scheme =
    colorScheme && colorScheme.length > 0
      ? colorScheme
      : ["#e8e8e8", "#9e9e9e", "#424242"];
  const numBins = scheme.length;
  const defaultLabels = ["Very Low", "Low", "Medium", "High", "Very High"];
  const labels =
    binLabels && binLabels.length >= numBins
      ? binLabels
      : defaultLabels.slice(defaultLabels.length - numBins);

  const title = legendTitle || "Value Range";
  let thresholds =
    customEdges && customEdges.length >= numBins - 1 ? customEdges : bins;
  if (!thresholds || thresholds.length < numBins - 1) {
    thresholds = Array.from({ length: numBins - 1 }, (_, j) => j + 1);
  }

  let html = `<strong>${title}</strong>`;

  for (let i = 0; i < numBins; i++) {
    let rangeText;
    if (i === 0) {
      rangeText = `≤${formatNumber(Number(thresholds[0]).toFixed(0))}`;
    } else if (i === numBins - 1) {
      rangeText = `>${formatNumber(Number(thresholds[i - 1]).toFixed(0))}`;
    } else {
      rangeText = `${formatNumber(Number(thresholds[i - 1]).toFixed(0))} - ${formatNumber(Number(thresholds[i]).toFixed(0))}`;
    }
    html += `
        <span class="legend-item">
            <span class="legend-swatch" style="background: ${scheme[i]};"></span>
            ${labels[i]} (${rangeText})
        </span>`;
  }

  html += `
        <span class="legend-item legend-nodata">
            <span class="legend-swatch" style="background: #f5f5f5; border: 1px solid #ccc;"></span>
            No data
        </span>`;

  container.innerHTML = html;
}

/**
 * Build a map of category string values to colors.
 * bin_labels[i] is the category name, color_scheme[i] is its color.
 */
function buildCategoryColorMap(binLabels, colorScheme) {
  const map = {};
  if (!binLabels || !colorScheme) return map;
  for (let i = 0; i < binLabels.length; i++) {
    map[binLabels[i]] = colorScheme[i] || "#cccccc";
  }
  return map;
}

/**
 * Style a district polygon using categorical (string) values.
 * Matches the district's value against categoryColorMap for direct color lookup.
 */
function styleCategoricalDistrict(feature, districtValues, categoryColorMap) {
  const districtName = geoFeatureName(feature);
  const value =
    districtValues == null ? undefined : districtValues[districtName];
  const hasData = value !== null && value !== undefined;

  if (!hasData || !categoryColorMap[value]) {
    return {
      fillColor: "#f5f5f5",
      fillOpacity: 1,
      color: "#666",
      weight: 0.5,
      opacity: 1,
    };
  }

  return {
    fillColor: categoryColorMap[value],
    fillOpacity: 0.7,
    color: "#666",
    weight: 0.5,
    opacity: 1,
  };
}

/**
 * Render a categorical legend bar (color swatch + category name, no numeric ranges)
 */
function renderCategoricalLegendBar(
  container,
  binLabels,
  colorScheme,
  legendTitle,
) {
  const title = legendTitle || "Category";
  let html = `<strong>${title}</strong>`;

  if (binLabels && colorScheme) {
    for (let i = 0; i < binLabels.length; i++) {
      html += `
            <span class="legend-item">
                <span class="legend-swatch" style="background: ${colorScheme[i] || "#cccccc"};"></span>
                ${binLabels[i]}
            </span>`;
    }
  }

  html += `
        <span class="legend-item legend-nodata">
            <span class="legend-swatch" style="background: #f5f5f5; border: 1px solid #ccc;"></span>
            No data
        </span>`;

  container.innerHTML = html;
}

/**
 * Render a faceted choropleth map (2x2 grid of maps by period)
 */
function renderFacetedChoropleth(element, component) {
  const div = document.createElement("div");
  div.className = "component component-faceted-choropleth";

  if (component.title) {
    const title = document.createElement("h3");
    title.className = "component-title";
    // Remove {{facet_label}} placeholder from main title
    title.textContent = component.title.replace(
      /\s*-?\s*\{\{facet_label\}\}/g,
      "",
    );
    div.appendChild(title);
  }

  if (component.description) {
    const desc = document.createElement("p");
    desc.className = "component-description";
    desc.textContent = component.description;
    div.appendChild(desc);
  }

  // Create grid container
  const gridContainer = document.createElement("div");
  gridContainer.className = "facet-grid";

  const facets = component.data.facets || [];

  // Shared legend bar below the grid (populated by last facet's callback)
  const legendBar = document.createElement("div");
  legendBar.className = "choropleth-legend-bar";

  facets.forEach((facet, index) => {
    const facetDiv = document.createElement("div");
    facetDiv.className = "facet-item";

    // Create facet title with period label
    const facetTitle = document.createElement("div");
    facetTitle.className = "facet-title";
    facetTitle.textContent = facet.label;
    facetDiv.appendChild(facetTitle);

    // Create map container
    const mapContainer = document.createElement("div");
    mapContainer.className = "facet-map-container";
    facetDiv.appendChild(mapContainer);

    // Get choropleth data for this facet
    const facetData = facet.data;

    // Build a mini component for this facet's choropleth
    const facetComponent = {
      type: "choropleth",
      valueType: component.valueType,
      title: "", // Title handled by facet container
      data: {
        geojson_path: facetData.geojson_path || component.data.geojson_path,
        district_values: facetData.district_values || {},
        color_scheme: facetData.color_scheme || component.data.color_scheme,
        bin_labels: facetData.bin_labels || component.data.bin_labels,
        bin_edges: facetData.bin_edges || component.data.bin_edges,
        legend_title: facetData.legend_title || component.data.legend_title,
      },
    };

    // Last facet populates the shared legend
    const isLast = index === facets.length - 1;
    renderFacetChoroplethMap(
      mapContainer,
      facetComponent,
      index,
      isLast ? legendBar : null,
    );

    gridContainer.appendChild(facetDiv);
  });

  div.appendChild(gridContainer);
  div.appendChild(legendBar);
  element.appendChild(div);
}

/**
 * Render a single choropleth map within a facet (smaller, no title)
 */
function renderFacetChoroplethMap(element, component, facetIndex, legendBar) {
  const mapDiv = document.createElement("div");
  mapDiv.id =
    "facet-choropleth-" +
    facetIndex +
    "-" +
    Math.random().toString(36).substr(2, 9);
  mapDiv.style.width = "100%";
  mapDiv.style.height = "100%";
  mapDiv.style.backgroundColor = "transparent";
  mapDiv.style.position = "relative";
  element.appendChild(mapDiv);

  setTimeout(() => {
    // Initialize global map storage if needed
    if (!window.choroplethMaps) {
      window.choroplethMaps = {};
    }

    const map = L.map(mapDiv.id, {
      dragging: false,
      scrollWheelZoom: false,
      doubleClickZoom: false,
      boxZoom: false,
      touchZoom: false,
      tap: false,
      keyboard: false,
      zoomControl: false,
      attributionControl: false,
      zoomSnap: 0,
      zoomDelta: 1,
      renderer: L.canvas(), // Canvas renderer: 1 canvas element instead of 146 SVG paths
    });

    window.choroplethMaps[mapDiv.id] = map;

    const isUganda = component.data.geojson_path && /uganda/i.test(component.data.geojson_path);
    Promise.all([
      fetchGeoJSON(component.data.geojson_path),
      isUganda ? fetchGeoJSON("/assets/uganda_outline.geojson") : Promise.resolve(null),
      isUganda ? fetchGeoJSON("/assets/uganda_regions.geojson") : Promise.resolve(null),
    ])
      .then(([geojson, outlineGeoJSON, regionsGeoJSON]) => {
        const isCatFacet = component.valueType === "categorical";
        const numBins = component.data.color_scheme?.length || 3;
        const bins = isCatFacet
          ? null
          : calculateQuantileBins(component.data.district_values, numBins);
        const customEdges = isCatFacet ? null : component.data.bin_edges;
        const categoryColorMap = isCatFacet
          ? buildCategoryColorMap(
              component.data.bin_labels,
              component.data.color_scheme,
            )
          : null;

        // Layer 1: Country outline with water-blue fill (bottom)
        if (outlineGeoJSON) {
        L.geoJSON(outlineGeoJSON, {
          style: {
            fillColor: "#B3D4E8",
            fillOpacity: 1,
            color: "#999",
            weight: 0.5,
            opacity: 0.5,
          },
        }).addTo(map);

        // Layer 2: District polygons colored by data (middle)
        const styleFn = isCatFacet
          ? (feature) =>
              styleCategoricalDistrict(
                feature,
                component.data.district_values,
                categoryColorMap,
              )
          : (feature) =>
              styleDistrict(
                feature,
                component.data.district_values,
                bins,
                component.data.color_scheme,
                customEdges,
              );

        const geoJsonLayer = L.geoJSON(geojson, {
          style: styleFn,
          onEachFeature: (feature, layer) => {
            const districtName = geoFeatureName(feature);
            const value =
              component.data.district_values == null
                ? undefined
                : component.data.district_values[districtName];
            const valueStr =
              value !== null && value !== undefined
                ? isCatFacet
                  ? value
                  : formatNumber(value)
                : "No data";

            layer.bindTooltip(
              `<strong>${districtName}</strong><br/>${valueStr}`,
              {
                permanent: false,
                direction: "top",
                className: "choropleth-tooltip",
              },
            );

            layer.bindPopup(
              `<strong>${districtName}</strong><br/>Value: ${valueStr}`,
            );

            layer.on("mouseover", function () {
              this.setStyle({ weight: 2, color: "#333" });
              this.openTooltip();
            });
            layer.on("mouseout", function () {
              this.setStyle({ weight: 0.5, color: "#666" });
              this.closeTooltip();
            });
          },
        }).addTo(map);

        // Layer 3: Region boundaries (top) - black lines, thick
        if (regionsGeoJSON) {
        L.geoJSON(regionsGeoJSON, {
          style: { fill: false, color: "#000", weight: 1.5, opacity: 1 },
          interactive: false,
        }).addTo(map);

        // Auto-fit with smaller padding for faceted view
        map.fitBounds(geoJsonLayer.getBounds(), {
          padding: [20, 20, 40, 20],
        });

        // Populate shared legend bar from last facet
        if (legendBar) {
          if (isCatFacet) {
            renderCategoricalLegendBar(
              legendBar,
              component.data.bin_labels,
              component.data.color_scheme,
              component.data.legend_title,
            );
          } else {
            renderChoroplethLegendBar(
              legendBar,
              bins,
              component.data.color_scheme,
              component.data.bin_labels,
              component.data.legend_title,
              customEdges,
            );
          }
        }

        map.invalidateSize();
      })
      .catch((error) => {
        console.error("Error loading GeoJSON for facet:", error);
        facetDiv.innerHTML = `<div style="padding:16px;text-align:center;font-size:0.8em;color:var(--color-error,#dc2626);">Map data failed to load</div>`;
      });

    map.invalidateSize();
  }, 100);
}

/**
 * Create a lazy loading placeholder for charts
 */
function createChartPlaceholder(element, component, renderFn) {
  const placeholder = document.createElement("div");
  placeholder.className = "component component-chart chart-placeholder";

  // Add title if present
  if (component.title) {
    const title = document.createElement("div");
    title.className = "component-title";
    title.textContent = component.title;
    placeholder.appendChild(title);
  }

  // Add skeleton loader
  const skeleton = document.createElement("div");
  skeleton.className = "skeleton skeleton-chart";
  skeleton.style.height = "350px";
  placeholder.appendChild(skeleton);

  element.appendChild(placeholder);

  // Store render function and observe
  placeholder._lazyRenderFn = () => {
    // Create a temporary container to capture the rendered chart
    const tempContainer = document.createElement("div");
    renderFn(tempContainer, component);

    // Replace placeholder with the rendered chart (maintains grid position)
    const renderedChart = tempContainer.firstElementChild;
    if (renderedChart) {
      renderedChart.classList.add("loaded");
      placeholder.replaceWith(renderedChart);
    }
  };

  lazyChartObserver.observe(placeholder);
}

/**
 * Render a component based on its type
 */
function renderComponent(element, component) {
  try {
    console.log(
      "Rendering component:",
      component.type,
      component.title,
      "data:",
      component.data,
    );

    // Show error state if the backend reported a failure for this component
    if (component.error) {
      const errDiv = document.createElement("div");
      errDiv.className = "component component-error";
      const inner = document.createElement("div");
      inner.className = "component-error-content";
      const titleEl = document.createElement("div");
      titleEl.className = "component-error-title";
      titleEl.textContent = component.title || "Component";
      const msgEl = document.createElement("div");
      msgEl.className = "component-error-message";
      msgEl.textContent = "Failed to load data";
      inner.appendChild(titleEl);
      inner.appendChild(msgEl);
      errDiv.appendChild(inner);
      element.appendChild(errDiv);
      return;
    }

    // Check for faceted choropleth (lazy-loaded)
    if (
      component.type === "choropleth" &&
      component.data &&
      component.data.facets &&
      component.data.facets.length > 0
    ) {
      createChartPlaceholder(element, component, renderFacetedChoropleth);
      return;
    }

    // Chart and map types use lazy loading
    const lazyTypes = [
      "line",
      "bar",
      "pie",
      "bar_line",
      "choropleth",
      "pyramid",
    ];

    if (lazyTypes.includes(component.type)) {
      const renderFns = {
        line: renderLineChart,
        bar: renderBarChart,
        pie: renderPieChart,
        bar_line: renderBarLineChart,
        choropleth: renderChoropleth,
        pyramid: renderPyramidChart,
      };
      createChartPlaceholder(element, component, renderFns[component.type]);
      return;
    }

    // Eager loading for tables, text, maps
    const childCountBefore = element.children.length;
    switch (component.type) {
      case "text":
      case "kpi":
        renderText(element, component);
        break;
      case "infobox":
        renderInfobox(element, component);
        break;
      case "table":
      case "table_advanced":
        renderTable(element, component);
        break;
      case "map":
        renderMap(element, component);
        break;
      default:
        console.warn("Unknown component type:", component.type);
    }
    // Add fade-in animation to newly appended component
    if (element.children.length > childCountBefore) {
      element.lastElementChild.classList.add("loaded");
    }
  } catch (error) {
    console.error("Error rendering component:", component.type, error);
    // Show error in UI
    const errorDiv = document.createElement("div");
    errorDiv.className = "component component-error";
    const errTitle = document.createElement("h3");
    errTitle.textContent = component.title || "Component";
    const errMsg = document.createElement("p");
    errMsg.className = "component-error-message";
    errMsg.textContent = "Render error: " + error.message;
    errorDiv.appendChild(errTitle);
    errorDiv.appendChild(errMsg);
    element.appendChild(errorDiv);
  }
}

/**
 * Render a section with its components
 */
function renderSection(element, section) {
  const sectionDiv = document.createElement("div");
  sectionDiv.className = "section";

  // Section header: title + inline feedback button
  const header = document.createElement("div");
  header.className = "section-header";

  if (section.title) {
    const title = document.createElement("div");
    title.className = "section-title";
    title.textContent = section.title;
    header.appendChild(title);
  }

  const fbBtn = document.createElement("button");
  fbBtn.type = "button";
  fbBtn.className = "section-feedback-btn";
  fbBtn.title = "Give feedback on this section";
  fbBtn.setAttribute("aria-label", "Give feedback on this section");
  fbBtn.innerHTML =
    '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="M14 1H2a1 1 0 0 0-1 1v8a1 1 0 0 0 1 1h3v3l3-3h6a1 1 0 0 0 1-1V2a1 1 0 0 0-1-1z" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round" fill="none"/></svg><span>Feedback</span>';
  const sectionTitle = section.title || "";
  fbBtn.addEventListener("click", () => {
    if (typeof window.openSectionFeedbackModal === "function") {
      window.openSectionFeedbackModal(sectionTitle);
    }
  });
  header.appendChild(fbBtn);

  sectionDiv.appendChild(header);

  if (section.description) {
    const desc = document.createElement("div");
    desc.className = "section-description";
    desc.innerHTML = section.description;
    sectionDiv.appendChild(desc);
  }

  const components = section.components || [];

  if (components.length > 0) {
    const componentsDiv = document.createElement("div");
    const normalizedLayout = normalizeLayout(section.layout);
    componentsDiv.className = `section-components layout-${normalizedLayout}`;

    components.forEach((component, idx) => {
      component._sectionId = section.id;
      component._componentIndex = idx;
      renderComponent(componentsDiv, component);
    });

    sectionDiv.appendChild(componentsDiv);
  }

  element.appendChild(sectionDiv);
}

/**
 * Format a date as relative time with short format (e.g., "2m ago")
 */
function formatTimeAgo(date) {
  const now = new Date();
  const seconds = Math.floor((now - date) / 1000);

  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}

/**
 * Render report footer with category siblings and timestamp
 * @param {Object} report - The current report object
 * @param {Array} allReports - All available reports (for finding siblings)
 * @param {string} fullReportId - Full report ID with path (e.g., "Programs/Malaria/report")
 */
function renderReportFooter(report, allReports, fullReportId) {
  const footer = document.getElementById("report-footer");
  const alsoInEl = document.getElementById("also-in-category");
  const timestampEl = document.getElementById("report-generated-at");

  if (!footer) return;

  // Render "Also in [Category]" siblings
  if (alsoInEl) {
    const siblings = findCategorySiblings(fullReportId, allReports);
    if (siblings.length > 0) {
      const categoryName = getCategoryDisplayName(fullReportId);
      const links = siblings
        .map(
          (r) =>
            `<a href="#" onclick="selectReport('${r.id}'); return false;">${r.title}</a>`,
        )
        .join(" · ");
      alsoInEl.innerHTML = `Also in ${categoryName}: ${links}`;
    } else {
      alsoInEl.innerHTML = "";
    }
  }

  // Render timestamp with offline freshness indicator
  if (timestampEl) {
    const savedDate =
      typeof pwa !== "undefined" && typeof currentReportId !== "undefined"
        ? pwa.getOfflineSavedDate(currentReportId)
        : null;

    if (typeof pwa !== "undefined" && pwa.isOffline() && savedDate) {
      // Offline: show when it was saved
      const d = new Date(savedDate);
      const label = d.toLocaleDateString(undefined, {
        month: "short",
        day: "numeric",
      });
      timestampEl.innerHTML =
        '<span class="offline-freshness-dot"></span> Saved for offline on ' +
        label;
      timestampEl.title = "This report was saved on " + d.toLocaleString();
    } else if (report.generated_at) {
      const date = new Date(report.generated_at);
      const timeAgo = formatTimeAgo(date);
      timestampEl.textContent = "Retrieved " + timeAgo;
      timestampEl.title = date.toLocaleString();
    } else {
      timestampEl.textContent = "";
    }
  }

  footer.style.display = "block";
}

/**
 * Find other reports in the same category (folder)
 * @param {string} currentId - Current report ID (e.g., "Programs/Malaria/weekly-report")
 * @param {Array} allReports - All available reports
 * @returns {Array} Sibling reports (excluding current)
 */
function findCategorySiblings(currentId, allReports) {
  if (!currentId || !allReports) return [];

  // Extract category path (everything except the last segment)
  const parts = currentId.split("/");
  if (parts.length < 2) return []; // No category structure

  const categoryPath = parts.slice(0, -1).join("/");

  // Find siblings in same category
  return allReports
    .filter((r) => {
      if (r.id === currentId) return false; // Exclude current report
      const rParts = r.id.split("/");
      if (rParts.length < 2) return false;
      const rCategoryPath = rParts.slice(0, -1).join("/");
      return rCategoryPath === categoryPath;
    })
    .slice(0, 5); // Limit to 5 siblings
}

/**
 * Get display name for category from report ID
 * @param {string} reportId - Report ID (e.g., "Programs/Malaria/weekly-report")
 * @returns {string} Category display name (e.g., "Malaria")
 */
function getCategoryDisplayName(reportId) {
  const parts = reportId.split("/");
  if (parts.length < 2) return "";
  // Use the last folder name (e.g., "Malaria" from "Programs/Malaria/report")
  return parts[parts.length - 2].replace(/-/g, " ");
}

/**
 * Hide report footer
 */
function hideReportFooter() {
  const footer = document.getElementById("report-footer");
  if (footer) footer.style.display = "none";
}

// Expose functions globally for use in other modules (e.g., builder preview)
window.renderComponent = renderComponent;
window.renderSection = renderSection;
window.cleanupAllInstances = cleanupAllInstances;
window.normalizeLayout = normalizeLayout;
