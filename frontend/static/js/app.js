/* ══════════════════════════════════════
   app.js – Live Data Fabric Engine
   StatGate Analytical Orchestration Hub
   ══════════════════════════════════════ */

let currentTenant = 'tenant-alpha';

function getCurrentTenant() {
  const sel = document.getElementById('tenantSelect');
  return sel ? sel.value : currentTenant;
}

function initTheme() {
  setTheme('fieldops');
}

function setTheme(themeName) {
  document.documentElement.removeAttribute('data-theme');
  localStorage.setItem('statgate_theme', 'fieldops');
  const sel = document.getElementById('themeSelect');
  if (sel) sel.value = themeName;
}

function onThemeChange() {
  const sel = document.getElementById('themeSelect');
  if (sel) setTheme(sel.value);
}

// ── Bootstrap ─────────────────────────────────────
document.addEventListener('DOMContentLoaded', async () => {
  initTheme();
  await populateDashboardDatasets();
  await populateBuilderDatasets();
  checkEngineStatus();
  loadKaggleDatasets();
  loadSemanticIndicators();
  loadABACPolicies();
  loadPublishedWidgets();
  refreshIntelligenceFeed();

  // Auto-refresh Intelligence Feed every 30 seconds
  setInterval(refreshIntelligenceFeed, 30000);

  // Restore saved dashboard layout if it exists
  restoreDashboardLayout();

  // If notebook page was accessed with ?dataset=, populate the selector
  const urlParams = new URLSearchParams(window.location.search);
  const datasetParam = urlParams.get('dataset');
  if (datasetParam) {
    const sel = document.getElementById('kaggleDatasetSelect');
    if (sel) {
      setTimeout(() => { sel.value = datasetParam; onKaggleDatasetChange(); }, 500);
    }
  }
});

// ── Live Dataset Catalog Cards ─────────────────────
async function populateDashboardDatasets() {
  try {
    const res = await fetch('/api/kaggle/datasets');
    const data = await res.json();
    const tables = data.tables || [];

    // Update count card
    const countEl = document.getElementById('cardTotalDatasets');
    if (countEl) countEl.innerText = tables.length;

    // Populate dashboard quick-select dropdown
    const sel = document.getElementById('dashboardDatasetSelect');
    if (sel) {
      sel.innerHTML = '<option value="">— Select a live dataset to snapshot —</option>';
      tables.forEach(t => {
        const opt = document.createElement('option');
        opt.value = t;
        opt.innerText = t;
        sel.appendChild(opt);
      });
    }
  } catch (e) {
    console.error('[app.js] populateDashboardDatasets error:', e);
  }
}

async function checkEngineStatus() {
  const el = document.getElementById('cardEngineStatus');
  if (!el) return;
  try {
    const start = performance.now();
    const res = await fetch('/api/kaggle/datasets');
    const ms = Math.round(performance.now() - start);
    if (res.ok) {
      el.innerText = `Live ✓`;
      el.style.color = 'var(--accent-green)';
      el.title = `DuckDB + PostgreSQL active. Catalog ping: ${ms}ms`;
    }
  } catch {
    el.innerText = 'Offline';
    el.style.color = 'var(--accent-red)';
  }
}

// ── App Launcher UI ─────────────────────────────
function toggleAppLauncher() {
  const wrapper = document.querySelector('.app-launcher-wrapper');
  const menu = wrapper ? wrapper.querySelector('.app-launcher-menu') : null;
  if (!menu) return;
  const shouldOpen = menu.style.display !== 'block';
  document.querySelectorAll('.app-launcher-menu').forEach((entry) => {
    entry.style.display = 'none';
  });
  if (shouldOpen) {
    menu.style.display = 'block';
  }
}

document.addEventListener('click', function (event) {
  const wrapper = event.target.closest('.app-launcher-wrapper');
  if (!wrapper) {
    document.querySelectorAll('.app-launcher-menu').forEach((menu) => {
      menu.style.display = 'none';
    });
  }
});

// ── Service Launcher UI ───────────────────────────
function toggleServiceLauncher() {
  const menu = document.getElementById('serviceLauncherMenu');
  if (!menu) return;
  menu.style.display = (menu.style.display === 'none' || menu.style.display === '') ? 'block' : 'none';
  if (menu.style.display === 'block') populateServiceLauncher();
}

async function populateServiceLauncher() {
  try {
    const res = await fetch('/api/services');
    const data = await res.json();
    const items = data.services || [];
    const container = document.getElementById('serviceItems');
    if (!container) return;
    container.innerHTML = '';
    items.forEach(s => {
      const div = document.createElement('div');
      div.style.display = 'flex';
      div.style.justifyContent = 'space-between';
      div.style.alignItems = 'center';
      div.style.padding = '6px 0';
      const left = document.createElement('div');
      left.innerText = s.name;
      const right = document.createElement('div');
      right.style.display = 'flex';
      right.style.gap = '6px';
      if (s.ui) {
        const a = document.createElement('a');
        a.href = s.ui;
        a.target = '_blank';
        a.rel = 'noopener';
        a.innerText = 'Open';
        right.appendChild(a);
      }
      const statusSpan = document.createElement('span');
      statusSpan.id = `svc_status_${s.id}`;
      statusSpan.style.minWidth = '64px';
      statusSpan.style.textAlign = 'right';
      statusSpan.innerText = '—';
      right.appendChild(statusSpan);
      div.appendChild(left);
      div.appendChild(right);
      container.appendChild(div);
    });
  } catch (e) {
    console.error('populateServiceLauncher error', e);
  }
}

async function checkAllServices() {
  try {
    const res = await fetch('/api/services/health');
    const data = await res.json();
    const results = data.results || [];
    results.forEach(r => {
      const el = document.getElementById(`svc_status_${r.id}`);
      if (el) {
        el.innerText = r.status === 'ok' ? 'OK' : r.status.toUpperCase();
        el.style.color = r.status === 'ok' ? 'var(--accent-green)' : 'var(--accent-gold)';
      }
    });
  } catch (e) {
    console.error('checkAllServices error', e);
  }
}

async function onDashboardDatasetChange() {
  const sel = document.getElementById('dashboardDatasetSelect');
  if (!sel || !sel.value) return;
  const table = sel.value;

  // Update "active dataset" pill
  const activeEl = document.getElementById('cardActiveDataset');
  if (activeEl) activeEl.innerText = table;

  // Update notebook launch link
  const link = document.getElementById('launchNotebookLink');
  if (link) link.href = `/notebook?dataset=${encodeURIComponent(table)}`;

  // Update status pills
  document.getElementById('snapStatus').innerText = 'Fetching from engine…';
  document.getElementById('snapRows').innerText = '…';
  document.getElementById('snapCols').innerText = '…';
  document.getElementById('snapSLA').innerText = '…';
  document.getElementById('snapshotTableWrap').style.display = 'none';

  try {
    const res = await fetch(`/api/kaggle/schema/${table}`);
    const summary = await res.json();
    const cols = summary.profiling || [];

    document.getElementById('snapStatus').innerText = '✓ Live';
    document.getElementById('snapRows').innerText = (summary.row_count || 0).toLocaleString();
    document.getElementById('snapCols').innerText = cols.length;
    document.getElementById('snapSLA').innerText = `${summary.latency_ms} ms`;

    // Render schema preview table dynamically (zero hardcoded headers)
    const thead = document.getElementById('snapshotSchemaHead');
    const tbody = document.getElementById('snapshotSchemaBody');
    if (thead && tbody) {
      thead.innerHTML = `<tr>
        <th>#</th>
        <th>Column Name</th>
        <th>Data Type</th>
        <th>Null Count</th>
        <th>Distinct</th>
        <th>Min</th>
        <th>Mean</th>
        <th>Max</th>
      </tr>`;
      tbody.innerHTML = '';
      cols.forEach((c, i) => {
        const s = c.stats || {};
        const nullClass = c.null_count > 0 ? 'color:var(--accent-gold)' : 'color:var(--accent-green)';
        tbody.innerHTML += `<tr>
          <td style="color:var(--text-muted)">${i+1}</td>
          <td><code style="color:var(--accent-cyan)">${c.column_name}</code></td>
          <td><span class="badge-tag badge-gold" style="font-size:0.72rem">${c.data_type}</span></td>
          <td style="${nullClass}">${c.null_count}</td>
          <td>${c.distinct_count.toLocaleString()}</td>
          <td style="font-family:'JetBrains Mono',monospace;font-size:0.8rem">${s.min !== undefined ? (+s.min).toLocaleString(undefined,{maximumFractionDigits:3}) : '—'}</td>
          <td style="font-family:'JetBrains Mono',monospace;font-size:0.8rem">${s.mean !== undefined ? (+s.mean).toLocaleString(undefined,{maximumFractionDigits:3}) : '—'}</td>
          <td style="font-family:'JetBrains Mono',monospace;font-size:0.8rem">${s.max !== undefined ? (+s.max).toLocaleString(undefined,{maximumFractionDigits:3}) : '—'}</td>
        </tr>`;
      });
      document.getElementById('snapshotTableWrap').style.display = 'block';
    }
  } catch (e) {
    document.getElementById('snapStatus').innerText = `Error: ${e.message}`;
  }
}

// ── Builder Dataset Population ─────────────────────
async function populateBuilderDatasets() {
  try {
    const res = await fetch('/api/kaggle/datasets');
    const data = await res.json();
    const sel = document.getElementById('builderDatasetSelect');
    if (!sel) return;
    sel.innerHTML = '<option value="">— Select Dataset —</option>';
    (data.tables || []).forEach(t => {
      const opt = document.createElement('option');
      opt.value = t;
      opt.innerText = t;
      sel.appendChild(opt);
    });
  } catch (e) {
    console.error('[app.js] populateBuilderDatasets error:', e);
  }
}

// ── NLQ Interface ──────────────────────────────────
async function runNLQ() {
  const input = document.getElementById('nlqInput').value;
  const outputDiv = document.getElementById('nlqResult');
  if (!input) return;
  outputDiv.style.display = 'block';
  outputDiv.innerText = '⚡ Interpreting natural language via StatGate Semantic Layer…';
  try {
    const res = await fetch('/api/nlq', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt: input, tenant_id: getCurrentTenant() })
    });
    const data = await res.json();
    outputDiv.innerText = `[Semantic Translation]\n${data.explanation}\n\n[Generated SQL]\n${data.generated_sql}`;
  } catch (e) {
    outputDiv.innerText = 'NLQ engine error: ' + e.message;
  }
}

// ── Notebook Execution ─────────────────────────────
async function runNotebookCode() {
  const code = document.getElementById('notebookCode').value;
  const consoleDiv = document.getElementById('notebookConsole');
  const kaggleSelect = document.getElementById('kaggleDatasetSelect');

  // Pull table from current session state — not a hardcoded string
  const selectedTable = kaggleSelect ? kaggleSelect.value : '';
  const sessionId = `session_${getCurrentTenant()}_${Date.now()}`;

  consoleDiv.innerText = '⚡ Executing in persistent kernel session (StatGateAnalysisEngine + DuckDB)…';
  try {
    const res = await fetch('/api/notebook/execute', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        code: code,
        tenant_id: getCurrentTenant(),
        kaggle_table: selectedTable,
        session_id: sessionId
      })
    });
    const data = await res.json();
    consoleDiv.innerText = data.success ? data.output : `Error: ${data.error}`;
  } catch (e) {
    consoleDiv.innerText = 'Execution request failed: ' + e.message;
  }
}

// ── Kaggle Dataset Selector (Notebook page) ────────
async function loadKaggleDatasets() {
  const select = document.getElementById('kaggleDatasetSelect');
  if (!select) return;
  try {
    const res = await fetch('/api/kaggle/datasets');
    const data = await res.json();
    select.innerHTML = '<option value="">— Load Dataset from Database —</option>';
    (data.tables || []).forEach(t => {
      const opt = document.createElement('option');
      opt.value = t;
      opt.innerText = t;
      select.appendChild(opt);
    });
  } catch (e) { console.error(e); }
}

function onKaggleDatasetChange() {
  const sel = document.getElementById('kaggleDatasetSelect');
  const editor = document.getElementById('notebookCode');
  if (!sel || !editor || !sel.value) return;
  const t = sel.value;

  // Update notebook badge if present
  const badge = document.getElementById('notebookDatasetBadge');
  if (badge) badge.innerHTML = `✅ Dataset: <strong style="color:var(--accent-cyan)">${t}</strong> via StatGateAnalysisEngine`;

  editor.value =
`# StatGate Notebook — Live Dataset Analysis
# Dataset: ${t} | Engine: StatGateAnalysisEngine (DuckDB In-Memory)

from statgate import engine
import pandas as pd

df = engine.load_dataset('${t}')
print(f"Table: ${t}  |  Shape: {df.shape}")
print("\\n--- Column Types ---")
print(df.dtypes)
print("\\n--- Statistical Summary ---")
print(df.describe())
`;
}

// ── Semantic Registry ──────────────────────────────
async function loadSemanticIndicators() {
  const tbody = document.getElementById('indicatorsTableBody');
  if (!tbody) return;
  try {
    const res = await fetch(`/api/proxy/indicators?tenant_id=${getCurrentTenant()}`);
    const data = await res.json();
    tbody.innerHTML = '';
    (data.indicators || []).forEach(ind => {
      const row = document.createElement('tr');
      row.innerHTML = `
        <td><code>${ind.id}</code></td>
        <td><strong>${ind.name}</strong></td>
        <td><code>${ind.formula}</code></td>
        <td><span class="badge-tag badge-gold">${ind.unit}</span></td>
        <td>${ind.tenant_id}</td>
      `;
      tbody.appendChild(row);
    });
    if (!data.indicators?.length) tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;">No indicators registered yet.</td></tr>';
  } catch (e) { console.error(e); }
}

async function createIndicator() {
  const name    = document.getElementById('indName')?.value;
  const unit    = document.getElementById('indUnit')?.value;
  const formula = document.getElementById('indFormula')?.value;
  const desc    = document.getElementById('indDesc')?.value;
  if (!name || !formula) return;
  await fetch('/api/proxy/indicators', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': getCurrentTenant() },
    body: JSON.stringify({ name, unit, formula, description: desc })
  });
  loadSemanticIndicators();
}

// ── ABAC Policy Viewer ─────────────────────────────
async function loadABACPolicies() {
  const tbody = document.getElementById('policiesTableBody');
  if (!tbody) return;
  try {
    const res = await fetch('/api/proxy/policies');
    const data = await res.json();
    tbody.innerHTML = '';
    data.forEach(p => {
      const row = document.createElement('tr');
      row.innerHTML = `
        <td><code>${p.id}</code></td>
        <td><strong>${p.resource}</strong></td>
        <td><span class="badge-tag badge-green">${p.action}</span></td>
        <td>Level ${p.min_clearance}</td>
        <td>${p.allowed_roles.join(', ')}</td>
        <td>${p.tenant_id}</td>
      `;
      tbody.appendChild(row);
    });
  } catch (e) { console.error(e); }
}

// ── Dashboard Layout Persistence ───────────────────
async function restoreDashboardLayout() {
  try {
    const res = await fetch('/api/dashboard/load/default_layout');
    const meta = await res.json();
    if (meta.table) {
      const sel = document.getElementById('builderDatasetSelect');
      if (sel) {
        // Wait for options to be populated
        setTimeout(async () => {
          sel.value = meta.table;
          await onBuilderDatasetChange();
          if (meta.colX) document.getElementById('builderColXSelect').value = meta.colX;
          if (meta.colY) document.getElementById('builderColYSelect').value = meta.colY;
          if (meta.chartType) document.getElementById('builderChartTypeSelect').value = meta.chartType;
          renderPlotlyChart();
        }, 800);
      }
    }
  } catch { /* no saved layout — silent */ }
}

// ── Proactive Intelligence Feed ────────────────────────────
async function refreshIntelligenceFeed() {
  const container = document.getElementById('intelligenceFeedContainer');
  const badge = document.getElementById('alertBadge');
  if (!container) return;

  try {
    const res = await fetch('/api/v1/alerts');
    const data = await res.json();
    const alerts = data.alerts || [];

    // Update badge
    if (badge) {
      badge.style.display = alerts.length > 0 ? 'block' : 'none';
      badge.innerText = `${alerts.length} Alert${alerts.length !== 1 ? 's' : ''}`;
    }

    container.innerHTML = '';

    if (alerts.length === 0) {
      container.innerHTML = `<div style="color:var(--accent-green);text-align:center;padding:20px;font-size:0.85rem;">✅ All data streams within ±3σ bounds. No anomalies detected.</div>`;
      return;
    }

    alerts.forEach(a => {
      const isCritical = (a.severity === 'CRITICAL');
      const sigmaText = a.sigma_score ? `${a.sigma_score.toFixed(2)}σ` : `${a.z_score ? a.z_score.toFixed(2) : '?'}σ`;
      const severityColor = isCritical ? 'var(--accent-red)' : 'var(--accent-gold)';
      const ts = a.timestamp ? new Date(a.timestamp).toLocaleTimeString() : 'now';

      const card = document.createElement('div');
      card.style.cssText = `
        background: linear-gradient(135deg, rgba(${isCritical ? '220,53,69' : '255,193,7'},0.08), rgba(6,9,15,0.6));
        border: 1px solid ${severityColor};
        border-radius: 8px;
        padding: 14px 18px;
        margin-bottom: 10px;
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        gap: 16px;
      `;
      card.innerHTML = `
        <div style="flex:1;">
          <div style="display:flex;align-items:center;gap:10px;margin-bottom:6px;">
            <span style="background:${severityColor};color:#000;font-size:0.72rem;font-weight:700;padding:2px 8px;border-radius:10px;">${a.severity || 'HIGH'}</span>
            <span style="font-size:0.82rem;font-weight:600;color:#fff;">${a.metric_name || a.event_id || 'System Metric'}</span>
            <span style="font-size:0.75rem;color:var(--text-muted);">@ ${ts}</span>
          </div>
          <div style="font-size:0.8rem;color:#ccc;margin-bottom:6px;">${a.message}</div>
          <div style="font-size:0.75rem;color:var(--text-muted);">
            Dataset: <code style="color:var(--accent-cyan)">${a.dataset || 'stream'}</code> &nbsp;|&nbsp;
            Value: <strong style="color:${severityColor}">${typeof a.value === 'number' ? a.value.toFixed(2) : a.value}</strong> &nbsp;|&nbsp;
            μ: ${typeof a.mean === 'number' ? a.mean.toFixed(2) : '—'} &nbsp;|&nbsp;
            Deviation: <strong style="color:${severityColor}">${sigmaText}</strong>
          </div>
        </div>
        <a href="${a.notebook_url || '/notebook'}" class="btn-primary" style="white-space:nowrap;text-decoration:none;padding:6px 12px;font-size:0.78rem;background:${severityColor};color:#000;">
          🔍 Investigate
        </a>
      `;
      container.appendChild(card);
    });
  } catch (e) {
    if (container) container.innerHTML = `<div style="color:var(--accent-red);padding:16px;text-align:center;">Intelligence Feed error: ${e.message}</div>`;
  }
}

// ── Published Widgets Dynamic Canvas ───────────────────────
async function loadPublishedWidgets() {
  const canvas = document.getElementById('publishedWidgetsCanvas');
  if (!canvas) return;

  try {
    const res = await fetch('/api/v1/assets/list');
    const data = await res.json();
    const assets = data.assets || [];

    // Filter widgets
    const widgets = assets.filter(a => a.id && a.id.startsWith('widget_cell_'));
    canvas.innerHTML = '';

    if (widgets.length === 0) {
      canvas.innerHTML = `<div style="color:var(--text-muted); font-size:0.85rem; padding:20px; text-align:center; grid-column: 1 / -1;">No published notebook widgets yet. Open the <a href="/notebook" style="color:var(--accent-cyan);">Notebook</a> and click "🚀 Publish to Dashboard" on any cell.</div>`;
      return;
    }

    widgets.forEach(w => {
      const def = w.content_definition || {};
      const card = document.createElement('div');
      card.className = 'glass-panel';
      card.style.background = 'rgba(18, 24, 36, 0.7)';
      card.style.margin = '0';

      card.innerHTML = `
        <div style="font-size:0.88rem; font-weight:600; color:var(--accent-cyan); margin-bottom:6px;">
          ${def.title || 'Published Widget'}
        </div>
        <div style="font-size:0.75rem; color:var(--text-muted); margin-bottom:8px;">
          Version: <span class="badge-tag badge-gold">${w.version_tag || '1.0.0'}</span> | Dataset: <code>${def.dataset || 'unlinked'}</code>
        </div>
        <div class="console-output" style="max-height:120px; overflow-y:auto; font-size:0.78rem; background:rgba(6,9,15,0.8);">
          ${def.output || 'No output recorded.'}
        </div>
      `;
      canvas.appendChild(card);
    });
  } catch (e) {
    canvas.innerHTML = `<div style="color:var(--accent-red); padding:20px; text-align:center; grid-column: 1 / -1;">Error loading published widgets: ${e.message}</div>`;
  }
}

// legacy alias kept for backward compat
async function refreshDashboard() {
  await populateDashboardDatasets();
  checkEngineStatus();
}
