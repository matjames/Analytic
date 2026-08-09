/**
 * StatGate Dashboard Framework — Shared Visualization Library
 * Phase X: Universal reusable charting and KPI components.
 * Wraps Chart.js (loaded from CDN) with enterprise-ready defaults.
 *
 * Usage:
 *   <script src="https://cdn.jsdelivr.net/npm/chart.js@4/dist/chart.umd.min.js"></script>
 *   <script src="/admin/dashboard-framework.js"></script>
 *
 *   DashboardFramework.KPICard(el, value, label, color, trend, onClick);
 *   DashboardFramework.LineChart(el, labels, datasets, options);
 *   DashboardFramework.BarChart(el, labels, datasets, options);
 *   DashboardFramework.DonutChart(el, labels, data, options);
 *   DashboardFramework.HeatmapCalendar(el, dailyData);
 *   DashboardFramework.GaugeChart(el, value, max, label);
 *   DashboardFramework.DataTable(el, columns, rows, options);
 *   DashboardFramework.startAutoRefresh(fetchFn, intervalMs);
 */

const DashboardFramework = (() => {
  // ── Colour Palette ────────────────────────────────────────────────────────
  const PALETTE = {
    blue:   '#2563eb',
    green:  '#10b981',
    amber:  '#f59e0b',
    red:    '#ef4444',
    purple: '#8b5cf6',
    teal:   '#14b8a6',
    indigo: '#6366f1',
    rose:   '#f43f5e',
    sky:    '#0ea5e9',
    lime:   '#84cc16',
  };

  const PALETTE_ARRAY = Object.values(PALETTE);

  const CHART_DEFAULTS = {
    animation: { duration: 600, easing: 'easeInOutQuart' },
    plugins: {
      legend: { labels: { font: { family: "'Inter', sans-serif", size: 12 }, color: '#475569' } },
      tooltip: {
        backgroundColor: '#0f172a',
        titleFont: { family: "'Inter', sans-serif", size: 13, weight: '600' },
        bodyFont:  { family: "'Inter', sans-serif", size: 12 },
        padding: 12, cornerRadius: 8,
      },
    },
    scales: {
      x: {
        grid: { color: 'rgba(148,163,184,0.15)' },
        ticks: { color: '#64748b', font: { family: "'Inter', sans-serif", size: 11 } },
      },
      y: {
        grid: { color: 'rgba(148,163,184,0.15)' },
        ticks: { color: '#64748b', font: { family: "'Inter', sans-serif", size: 11 } },
        beginAtZero: true,
      },
    },
    responsive: true,
    maintainAspectRatio: false,
  };

  // Registry of Chart instances for destroy-before-recreate
  const _charts = new WeakMap();

  function _destroyChart(el) {
    if (_charts.has(el)) {
      _charts.get(el).destroy();
      _charts.delete(el);
    }
  }

  function _deepMerge(target, source) {
    for (const key of Object.keys(source)) {
      if (source[key] && typeof source[key] === 'object' && !Array.isArray(source[key])) {
        target[key] = target[key] || {};
        _deepMerge(target[key], source[key]);
      } else {
        target[key] = source[key];
      }
    }
    return target;
  }

  // ── KPI Card ──────────────────────────────────────────────────────────────
  /**
   * Renders an animated KPI card into `el`.
   * @param {HTMLElement} el - Target element (cleared and populated)
   * @param {number|string} value
   * @param {string} label
   * @param {'blue'|'green'|'amber'|'red'|'purple'|string} color
   * @param {string} [trend] - e.g. "+12% this week"
   * @param {Function} [onClick] - Drill-down callback
   */
  function KPICard(el, value, label, color = 'blue', trend = '', onClick = null) {
    const c = PALETTE[color] || color;
    el.innerHTML = `
      <div class="sgdf-kpi" style="
        background:#fff; border:1px solid #e2e8f0; border-radius:12px;
        padding:20px; box-shadow:0 1px 3px rgba(0,0,0,0.08);
        position:relative; overflow:hidden;
        cursor:${onClick ? 'pointer' : 'default'};
        transition: transform 0.15s, box-shadow 0.15s;
      " data-sgdf-kpi>
        <div style="position:absolute;top:0;left:0;right:0;height:3px;background:${c}"></div>
        <div style="font-size:32px;font-weight:800;color:${c};line-height:1;font-family:'Inter',sans-serif" class="sgdf-kpi-val">${value}</div>
        <div style="font-size:11px;font-weight:700;color:#64748b;text-transform:uppercase;letter-spacing:0.5px;margin-top:8px">${label}</div>
        ${trend ? `<div style="font-size:11px;color:#94a3b8;margin-top:4px">${trend}</div>` : ''}
      </div>
    `;
    const inner = el.querySelector('[data-sgdf-kpi]');
    inner.addEventListener('mouseenter', () => { inner.style.transform = 'translateY(-2px)'; inner.style.boxShadow = '0 8px 24px rgba(0,0,0,0.12)'; });
    inner.addEventListener('mouseleave', () => { inner.style.transform = ''; inner.style.boxShadow = ''; });
    if (onClick) {
      inner.addEventListener('click', onClick);
      el.dispatchEvent(new CustomEvent('dashboardDrilldown', { bubbles: true, detail: { label } }));
    }
    // Animate count-up
    _animateCounter(el.querySelector('.sgdf-kpi-val'), value);
  }

  function _animateCounter(el, target) {
    const parsed = parseInt(String(target).replace(/[^0-9]/g, ''));
    if (!el || isNaN(parsed) || parsed === 0) return;
    let start = 0;
    const step = Math.max(1, Math.ceil(parsed / 40));
    const timer = setInterval(() => {
      start = Math.min(start + step, parsed);
      el.textContent = start.toLocaleString();
      if (start >= parsed) clearInterval(timer);
    }, 16);
  }

  // ── Line Chart ────────────────────────────────────────────────────────────
  /**
   * @param {HTMLElement} el - Canvas container (sets height via style)
   * @param {string[]} labels
   * @param {{ label:string, data:number[], color?:string }[]} datasets
   * @param {object} [options] - Chart.js overrides
   */
  function LineChart(el, labels, datasets, options = {}) {
    _destroyChart(el);
    el.innerHTML = '<canvas></canvas>';
    const canvas = el.querySelector('canvas');
    const chartDatasets = datasets.map((d, i) => {
      const c = d.color || PALETTE_ARRAY[i % PALETTE_ARRAY.length];
      return {
        label: d.label,
        data: d.data,
        borderColor: c,
        backgroundColor: c + '18',
        borderWidth: 2.5,
        pointRadius: 3,
        pointHoverRadius: 6,
        fill: true,
        tension: 0.4,
      };
    });
    const cfg = _deepMerge(JSON.parse(JSON.stringify(CHART_DEFAULTS)), options);
    const chart = new Chart(canvas, {
      type: 'line',
      data: { labels, datasets: chartDatasets },
      options: cfg,
    });
    _charts.set(el, chart);
    return chart;
  }

  // ── Bar Chart ─────────────────────────────────────────────────────────────
  /**
   * @param {HTMLElement} el
   * @param {string[]} labels
   * @param {{ label:string, data:number[], color?:string }[]} datasets
   * @param {object} [options]
   */
  function BarChart(el, labels, datasets, options = {}) {
    _destroyChart(el);
    el.innerHTML = '<canvas></canvas>';
    const canvas = el.querySelector('canvas');
    const chartDatasets = datasets.map((d, i) => {
      const c = d.color || PALETTE_ARRAY[i % PALETTE_ARRAY.length];
      return {
        label: d.label,
        data: d.data,
        backgroundColor: c + 'cc',
        borderColor: c,
        borderWidth: 1.5,
        borderRadius: 6,
        borderSkipped: false,
      };
    });
    const cfg = _deepMerge(JSON.parse(JSON.stringify(CHART_DEFAULTS)), options);
    const chart = new Chart(canvas, {
      type: 'bar',
      data: { labels, datasets: chartDatasets },
      options: cfg,
    });
    _charts.set(el, chart);
    return chart;
  }

  // ── Horizontal Bar Chart ──────────────────────────────────────────────────
  function HorizontalBarChart(el, labels, data, colorKey = 'blue', options = {}) {
    _destroyChart(el);
    el.innerHTML = '<canvas></canvas>';
    const canvas = el.querySelector('canvas');
    const c = PALETTE[colorKey] || colorKey;
    const cfg = _deepMerge(JSON.parse(JSON.stringify(CHART_DEFAULTS)), options);
    if (cfg.scales) { cfg.scales.x = cfg.scales.x || {}; cfg.scales.y = cfg.scales.y || {}; }
    const chart = new Chart(canvas, {
      type: 'bar',
      data: {
        labels,
        datasets: [{ data, backgroundColor: c + 'cc', borderColor: c, borderWidth: 1.5, borderRadius: 4 }],
      },
      options: { ...cfg, indexAxis: 'y' },
    });
    _charts.set(el, chart);
    return chart;
  }

  // ── Donut / Pie Chart ─────────────────────────────────────────────────────
  /**
   * @param {HTMLElement} el
   * @param {string[]} labels
   * @param {number[]} data
   * @param {object} [options]
   */
  function DonutChart(el, labels, data, options = {}) {
    _destroyChart(el);
    el.innerHTML = '<canvas></canvas>';
    const canvas = el.querySelector('canvas');
    const colors = labels.map((_, i) => PALETTE_ARRAY[i % PALETTE_ARRAY.length]);
    const cfg = {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { position: 'right', labels: { font: { family: "'Inter', sans-serif", size: 12 }, color: '#475569', padding: 16 } },
        tooltip: CHART_DEFAULTS.plugins.tooltip,
      },
      cutout: '65%',
      animation: { duration: 700, easing: 'easeInOutQuart' },
    };
    _deepMerge(cfg, options);
    const chart = new Chart(canvas, {
      type: 'doughnut',
      data: { labels, datasets: [{ data, backgroundColor: colors.map(c => c + 'dd'), borderColor: colors, borderWidth: 2 }] },
      options: cfg,
    });
    _charts.set(el, chart);
    return chart;
  }

  // ── Gauge Chart ───────────────────────────────────────────────────────────
  /**
   * Renders an SVG-based arc gauge.
   * @param {HTMLElement} el
   * @param {number} value - 0-max
   * @param {number} max
   * @param {string} label
   * @param {string} [color]
   */
  function GaugeChart(el, value, max = 100, label = '', color = null) {
    const pct = Math.min(value / max, 1);
    const c = color || (pct > 0.75 ? PALETTE.green : pct > 0.4 ? PALETTE.amber : PALETTE.red);
    const angle = pct * 180;
    const rad = (angle - 90) * (Math.PI / 180);
    const cx = 100, cy = 100, r = 75;
    const x = cx + r * Math.cos(rad);
    const y = cy + r * Math.sin(rad);
    const largeArc = angle > 180 ? 1 : 0;

    el.innerHTML = `
      <svg viewBox="0 0 200 120" style="width:100%;max-width:240px;display:block;margin:0 auto">
        <!-- track -->
        <path d="M 25 100 A 75 75 0 0 1 175 100" fill="none" stroke="#e2e8f0" stroke-width="18" stroke-linecap="round"/>
        <!-- fill -->
        <path d="M 25 100 A 75 75 0 ${largeArc} 1 ${x.toFixed(2)} ${y.toFixed(2)}" fill="none" stroke="${c}" stroke-width="18" stroke-linecap="round"
          style="transition: stroke-dashoffset 1s ease">
          <animate attributeName="stroke-dasharray" from="0 ${Math.PI*r}" to="${pct*Math.PI*r} ${Math.PI*r}" dur="0.8s" fill="freeze" />
        </path>
        <!-- value text -->
        <text x="100" y="98" text-anchor="middle" font-size="28" font-weight="800" fill="${c}" font-family="Inter,sans-serif">${Math.round(value)}</text>
        <text x="100" y="115" text-anchor="middle" font-size="11" fill="#64748b" font-family="Inter,sans-serif">${label}</text>
      </svg>
    `;
  }

  // ── Calendar Heatmap ──────────────────────────────────────────────────────
  /**
   * Renders a GitHub-style calendar heatmap.
   * @param {HTMLElement} el
   * @param {Object.<string,number>} dailyData - { "YYYY-MM-DD": count }
   * @param {string} [colorKey]
   */
  function HeatmapCalendar(el, dailyData, colorKey = 'blue') {
    const base = PALETTE[colorKey] || PALETTE.blue;
    const max = Math.max(...Object.values(dailyData), 1);
    const today = new Date();
    const cells = [];

    for (let i = 89; i >= 0; i--) {
      const d = new Date(today);
      d.setDate(today.getDate() - i);
      const key = d.toISOString().slice(0, 10);
      const val = dailyData[key] || 0;
      const intensity = val / max;
      const bg = val === 0 ? '#f1f5f9' : _hexOpacity(base, 0.15 + intensity * 0.85);
      cells.push(`<div title="${key}: ${val} submissions" style="
        width:14px;height:14px;border-radius:3px;background:${bg};
        flex-shrink:0;cursor:default;transition:transform 0.1s;
      " onmouseenter="this.style.transform='scale(1.4)'" onmouseleave="this.style.transform=''"></div>`);
    }

    el.innerHTML = `
      <div style="display:flex;flex-wrap:wrap;gap:3px;padding:8px">
        ${cells.join('')}
      </div>
      <div style="display:flex;align-items:center;gap:6px;padding:4px 8px;font-size:11px;color:#64748b">
        Less <div style="display:flex;gap:3px">
          ${[0.1,0.3,0.5,0.75,1].map(v => `<div style="width:12px;height:12px;border-radius:2px;background:${_hexOpacity(base, v)}"></div>`).join('')}
        </div> More
      </div>
    `;
  }

  function _hexOpacity(hex, opacity) {
    const r = parseInt(hex.slice(1,3), 16);
    const g = parseInt(hex.slice(3,5), 16);
    const b = parseInt(hex.slice(5,7), 16);
    return `rgba(${r},${g},${b},${opacity.toFixed(2)})`;
  }

  // ── Data Table ────────────────────────────────────────────────────────────
  /**
   * Renders a sortable, filterable data table.
   * @param {HTMLElement} el
   * @param {{ key:string, label:string, format?:Function }[]} columns
   * @param {Object[]} rows
   * @param {{ searchable?:boolean, exportable?:boolean, maxRows?:number }} options
   */
  function DataTable(el, columns, rows, options = {}) {
    const { searchable = true, exportable = false, maxRows = 100 } = options;
    let sortCol = null, sortAsc = true, filterText = '';

    function render(data) {
      const displayed = data.slice(0, maxRows);
      el.innerHTML = `
        ${searchable ? `<div style="padding:8px 0 12px;display:flex;gap:8px;align-items:center">
          <input placeholder="Search…" id="sgdf-search" style="flex:1;padding:8px 12px;border:1px solid #e2e8f0;border-radius:8px;font-size:13px;font-family:Inter,sans-serif" value="${filterText}" />
          ${exportable ? `<button id="sgdf-export" style="padding:8px 14px;background:#2563eb;color:white;border:none;border-radius:8px;font-size:12px;font-weight:600;cursor:pointer">↓ CSV</button>` : ''}
        </div>` : ''}
        <div style="overflow-x:auto">
          <table style="width:100%;border-collapse:collapse;font-size:13px;font-family:Inter,sans-serif">
            <thead>
              <tr>
                ${columns.map(c => `<th data-key="${c.key}" style="padding:10px 14px;text-align:left;font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:0.5px;color:#64748b;border-bottom:2px solid #e2e8f0;background:#f8fafc;cursor:pointer;user-select:none;white-space:nowrap">
                  ${c.label} ${sortCol === c.key ? (sortAsc ? '↑' : '↓') : ''}
                </th>`).join('')}
              </tr>
            </thead>
            <tbody>
              ${displayed.length === 0 ? `<tr><td colspan="${columns.length}" style="text-align:center;padding:24px;color:#94a3b8">No records found</td></tr>` :
                displayed.map(row => `<tr style="transition:background 0.1s" onmouseenter="this.style.background='#f8fafc'" onmouseleave="this.style.background=''">
                  ${columns.map(c => `<td style="padding:11px 14px;border-bottom:1px solid #f1f5f9;vertical-align:middle">${c.format ? c.format(row[c.key], row) : (row[c.key] ?? '—')}</td>`).join('')}
                </tr>`).join('')}
            </tbody>
          </table>
        </div>
        ${data.length > maxRows ? `<div style="padding:8px 14px;font-size:11px;color:#94a3b8">Showing ${maxRows} of ${data.length} records</div>` : ''}
      `;

      el.querySelectorAll('th[data-key]').forEach(th => {
        th.addEventListener('click', () => {
          const key = th.dataset.key;
          if (sortCol === key) sortAsc = !sortAsc;
          else { sortCol = key; sortAsc = true; }
          const sorted = [...rows].filter(r => !filterText || columns.some(c => String(r[c.key] || '').toLowerCase().includes(filterText)));
          sorted.sort((a, b) => {
            const av = a[key], bv = b[key];
            return (sortAsc ? 1 : -1) * (av > bv ? 1 : av < bv ? -1 : 0);
          });
          render(sorted);
        });
      });

      const searchInput = el.querySelector('#sgdf-search');
      if (searchInput) {
        searchInput.addEventListener('input', () => {
          filterText = searchInput.value.toLowerCase();
          const filtered = rows.filter(r => columns.some(c => String(r[c.key] || '').toLowerCase().includes(filterText)));
          render(filtered);
        });
      }

      const expBtn = el.querySelector('#sgdf-export');
      if (expBtn) {
        expBtn.addEventListener('click', () => _exportCSV(columns, data));
      }
    }
    render(rows);
  }

  function _exportCSV(columns, rows) {
    const header = columns.map(c => `"${c.label}"`).join(',');
    const body   = rows.map(r => columns.map(c => `"${String(r[c.key] ?? '').replace(/"/g, '""')}"`).join(',')).join('\n');
    const blob = new Blob([header + '\n' + body], { type: 'text/csv' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'statcollect_export_' + new Date().toISOString().slice(0,10) + '.csv';
    a.click();
  }

  // ── Pivot Table ───────────────────────────────────────────────────────────
  /**
   * Renders a basic pivot table.
   * @param {HTMLElement} el
   * @param {Object[]} data
   * @param {string} rowField
   * @param {string} colField
   * @param {string} valueField
   * @param {'count'|'sum'|'avg'} [aggFn]
   */
  function PivotTable(el, data, rowField, colField, valueField, aggFn = 'count') {
    const rowVals = [...new Set(data.map(d => d[rowField]))].sort();
    const colVals = [...new Set(data.map(d => d[colField]))].sort();

    const matrix = {};
    rowVals.forEach(rv => {
      matrix[rv] = {};
      colVals.forEach(cv => { matrix[rv][cv] = []; });
    });
    data.forEach(d => {
      const rv = d[rowField], cv = d[colField];
      if (matrix[rv] && matrix[rv][cv] !== undefined) {
        matrix[rv][cv].push(d[valueField]);
      }
    });

    function agg(arr) {
      if (arr.length === 0) return 0;
      if (aggFn === 'count') return arr.length;
      if (aggFn === 'sum')   return arr.reduce((a,b)=>a+(+b||0),0);
      if (aggFn === 'avg')   return (arr.reduce((a,b)=>a+(+b||0),0)/arr.length).toFixed(1);
      return arr.length;
    }

    el.innerHTML = `
      <div style="overflow-x:auto">
        <table style="width:100%;border-collapse:collapse;font-size:12px;font-family:Inter,sans-serif">
          <thead>
            <tr>
              <th style="padding:8px 12px;background:#f8fafc;border:1px solid #e2e8f0;font-weight:700;color:#475569">${rowField} / ${colField}</th>
              ${colVals.map(cv => `<th style="padding:8px 12px;background:#f8fafc;border:1px solid #e2e8f0;font-weight:700;color:#475569;white-space:nowrap">${cv}</th>`).join('')}
              <th style="padding:8px 12px;background:#eff6ff;border:1px solid #e2e8f0;font-weight:700;color:#2563eb">Total</th>
            </tr>
          </thead>
          <tbody>
            ${rowVals.map(rv => {
              const rowTotal = colVals.reduce((t, cv) => t + (aggFn === 'count' ? matrix[rv][cv].length : 0), 0);
              return `<tr>
                <td style="padding:8px 12px;border:1px solid #e2e8f0;font-weight:600;color:#0f172a;background:#fafcff">${rv}</td>
                ${colVals.map(cv => {
                  const v = agg(matrix[rv][cv]);
                  const max = Math.max(...colVals.map(c2 => agg(matrix[rv][c2])), 1);
                  const pct = Math.round((v/max)*100);
                  return `<td style="padding:8px 12px;border:1px solid #e2e8f0;text-align:center;background:rgba(37,99,235,${(pct/100*0.12).toFixed(2)})">${v}</td>`;
                }).join('')}
                <td style="padding:8px 12px;border:1px solid #e2e8f0;text-align:center;font-weight:700;color:#2563eb;background:#eff6ff">${rowTotal}</td>
              </tr>`;
            }).join('')}
          </tbody>
        </table>
      </div>
    `;
  }

  // ── Spark Lines (inline mini chart) ──────────────────────────────────────
  function SparkLine(el, data, color = PALETTE.blue) {
    if (!data || data.length < 2) { el.innerHTML = ''; return; }
    const max = Math.max(...data), min = Math.min(...data);
    const w = 80, h = 28;
    const pts = data.map((v, i) => {
      const x = (i / (data.length - 1)) * w;
      const y = h - ((v - min) / (max - min || 1)) * h;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    }).join(' ');
    el.innerHTML = `<svg width="${w}" height="${h}" style="display:block">
      <polyline points="${pts}" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
    </svg>`;
  }

  // ── Auto-Refresh ──────────────────────────────────────────────────────────
  /**
   * Starts a recurring refresh of fetchFn every intervalMs milliseconds.
   * Returns a cancel function.
   * @param {Function} fetchFn - async function to call
   * @param {number} [intervalMs=30000]
   * @returns {Function} cancel
   */
  function startAutoRefresh(fetchFn, intervalMs = 30000) {
    fetchFn();
    const timer = setInterval(fetchFn, intervalMs);
    return () => clearInterval(timer);
  }

  // ── SSE Live Events Helper ────────────────────────────────────────────────
  /**
   * Connects to the analytics SSE stream and calls handler for each event.
   * @param {string} url - Defaults to /api/analytics/stream
   * @param {Function} handler - (event) => void
   * @returns {EventSource}
   */
  function connectLiveStream(url = '/api/analytics/stream', handler) {
    const es = new EventSource(url);
    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        handler(data);
      } catch (_) {}
    };
    es.onerror = () => {
      // EventSource auto-reconnects
    };
    return es;
  }

  // ── Map helpers (used with gis-widget.js) ────────────────────────────────

  // ── Distribution Bar (CSS only, no Chart.js needed) ───────────────────────
  /**
   * Renders a simple CSS distribution bar from a {value: count} map.
   * @param {HTMLElement} el
   * @param {Object.<string,number>} distribution
   * @param {string} [color]
   */
  function DistributionBars(el, distribution, color = PALETTE.blue) {
    const entries = Object.entries(distribution);
    const total = entries.reduce((sum, [,v]) => sum + v, 0);
    if (total === 0) { el.innerHTML = '<div style="color:#94a3b8;font-size:12px">No data</div>'; return; }
    el.innerHTML = entries.sort((a,b) => b[1]-a[1]).map(([label, count]) => {
      const pct = Math.round((count / total) * 100);
      return `<div style="margin-bottom:10px">
        <div style="display:flex;justify-content:space-between;font-size:12px;margin-bottom:3px;font-family:Inter,sans-serif">
          <span style="color:#334155;font-weight:500">${_esc(label)}</span>
          <span style="color:#64748b"><b>${count}</b> (${pct}%)</span>
        </div>
        <div style="background:#f1f5f9;border-radius:6px;height:8px;overflow:hidden">
          <div style="background:${color};width:${pct}%;height:100%;border-radius:6px;transition:width 0.6s ease"></div>
        </div>
      </div>`;
    }).join('');
  }

  function _esc(s) {
    return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  }

  // ── Public API ────────────────────────────────────────────────────────────
  return {
    PALETTE,
    KPICard,
    LineChart,
    BarChart,
    HorizontalBarChart,
    DonutChart,
    GaugeChart,
    HeatmapCalendar,
    DataTable,
    PivotTable,
    SparkLine,
    DistributionBars,
    startAutoRefresh,
    connectLiveStream,
  };
})();

// Make globally available
window.DashboardFramework = DashboardFramework;
