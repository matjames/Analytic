/* ═══════════════════════════════════════════════
   catalog.js – Kaggle-style Dataset Exploration
   ═══════════════════════════════════════════════ */

let allDatasets = [];
let activeDataset = null;
let activeDatasetSummary = null;
let activeSortCol = null;
let activeSortDir = 'asc';
let currentTab = 'schema';
let rawDataRows = [];
let dataSortCol = null;
let dataSortDir = 'asc';

// ── Bootstrap ────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  loadDatasetSidebar();
});

// ── Sidebar ──────────────────────────────────────
async function loadDatasetSidebar() {
  const list = document.getElementById('datasetSidebarList');
  const countBadge = document.getElementById('catalogCountBadge');

  try {
    const res = await fetch('/api/kaggle/datasets');
    const data = await res.json();
    allDatasets = data.tables || [];
    countBadge.innerText = `${allDatasets.length} datasets`;
    renderSidebarList(allDatasets);
  } catch (e) {
    list.innerHTML = `<li class="dataset-list-placeholder" style="color:var(--accent-red);">Failed to load catalog: ${e.message}</li>`;
    countBadge.innerText = 'Error';
    countBadge.className = 'badge-tag badge-red';
  }
}

function renderSidebarList(datasets) {
  const list = document.getElementById('datasetSidebarList');
  list.innerHTML = '';

  if (datasets.length === 0) {
    list.innerHTML = '<li class="dataset-list-placeholder">No matching datasets.</li>';
    return;
  }

  datasets.forEach(t => {
    const li = document.createElement('li');
    li.className = 'dataset-list-item' + (t === activeDataset ? ' active' : '');
    li.onclick = () => selectDataset(t, li);
    li.innerHTML = `
      <span class="dataset-item-icon">🗃️</span>
      <span class="dataset-item-text">
        <span class="dataset-item-name" title="${t}">${t}</span>
        <span class="dataset-item-schema">ml_staging</span>
      </span>`;
    list.appendChild(li);
  });
}

function filterSidebarDatasets() {
  const q = document.getElementById('sidebarSearchInput').value.toLowerCase();
  const filtered = allDatasets.filter(t => t.toLowerCase().includes(q));
  renderSidebarList(filtered);
}

// ── Dataset Selection ─────────────────────────────
async function selectDataset(tableName, liEl) {
  // Mark active in sidebar
  document.querySelectorAll('.dataset-list-item').forEach(el => el.classList.remove('active'));
  if (liEl) liEl.classList.add('active');

  activeDataset = tableName;

  // Show workspace panel, hide welcome
  document.getElementById('workspaceWelcome').style.display = 'none';
  document.getElementById('workspacePanel').style.display = 'flex';

  // Reset tab to schema
  switchTab('schema');

  // Set header
  document.getElementById('panelDatasetName').innerText = tableName;
  document.getElementById('panelDatasetMeta').innerText = 'Fetching profile via StatGateAnalysisEngine…';
  document.getElementById('schemaTableBody').innerHTML = `<tr><td colspan="9" class="loading-cell">⚡ Profiling dataset with DuckDB in-memory engine…</td></tr>`;

  // Reset stat pills
  ['pillRows', 'pillCols', 'pillLatency'].forEach(id => document.getElementById(id).innerText = '…');

  // Fetch summary from engine
  try {
    const res = await fetch(`/api/kaggle/schema/${tableName}`);
    const summary = await res.json();
    activeDatasetSummary = summary;

    // Update header meta
    document.getElementById('panelDatasetMeta').innerText =
      `StatGateAnalysisEngine · DuckDB In-Memory · Profiled in ${summary.latency_ms} ms`;

    // Update stat pills
    document.getElementById('pillRows').innerText = summary.row_count.toLocaleString();
    document.getElementById('pillCols').innerText = (summary.profiling || []).length;
    document.getElementById('pillLatency').innerText = `${summary.latency_ms} ms`;

    // Render schema tab
    renderSchemaTab(summary.profiling || []);

    // Cache raw data rows
    rawDataRows = summary.sample_records || [];
    document.getElementById('rowCountLabel').innerText = `${rawDataRows.length} rows`;

  } catch (e) {
    document.getElementById('panelDatasetMeta').innerText = 'Error loading profile.';
    document.getElementById('schemaTableBody').innerHTML =
      `<tr><td colspan="9" class="loading-cell" style="color:var(--accent-red);">Failed: ${e.message}</td></tr>`;
  }
}

// ── Schema Tab ────────────────────────────────────
function renderSchemaTab(profiling) {
  const tbody = document.getElementById('schemaTableBody');
  tbody.innerHTML = '';

  if (!profiling || profiling.length === 0) {
    tbody.innerHTML = `<tr><td colspan="9" class="loading-cell">No schema metadata available for this dataset.</td></tr>`;
    return;
  }

  profiling.forEach((col, idx) => {
    const s = col.stats || {};
    const hasStats = s.mean !== undefined;
    const nullClass = col.null_count > 0 ? 'null-warn' : 'null-ok';

    const row = document.createElement('tr');
    row.innerHTML = `
      <td class="col-idx">${idx + 1}</td>
      <td class="col-name">${col.column_name}</td>
      <td><span class="badge-tag badge-gold" style="font-size:0.72rem;">${col.data_type}</span></td>
      <td class="${nullClass}">${col.null_count} ${col.null_count > 0 ? '⚠️' : '✓'}</td>
      <td class="stat-val">${col.distinct_count.toLocaleString()}</td>
      <td class="stat-val">${hasStats ? fmt(s.min) : '—'}</td>
      <td class="stat-val">${hasStats ? fmt(s.mean) : '—'}</td>
      <td class="stat-val">${hasStats ? fmt(s.max) : '—'}</td>
      <td class="stat-val">${hasStats ? fmt(s.std) : '—'}</td>
    `;
    tbody.appendChild(row);
  });
}

function fmt(val) {
  if (val === null || val === undefined) return '—';
  return typeof val === 'number' ? val.toLocaleString(undefined, { maximumFractionDigits: 4 }) : val;
}

// ── Data Grid Tab (Dynamic Active Projection) ─────────────────
async function fetchLiveDataGrid(tableName) {
  const wrap = document.getElementById('dataGridWrap');
  wrap.innerHTML = `<div class="loading-cell">⚡ Executing SELECT * FROM ${tableName} LIMIT 100 on live Kaggle database…</div>`;

  try {
    const res = await fetch(`/api/v1/datasets/fetch/${tableName}`);
    const data = await res.json();
    rawDataRows = data.records || [];
    document.getElementById('rowCountLabel').innerText = `${rawDataRows.length} rows (Live Projection)`;
    renderDataGrid(rawDataRows);
  } catch (e) {
    wrap.innerHTML = `<div class="loading-cell" style="color:var(--accent-red);">Failed to fetch live dataset grid: ${e.message}</div>`;
  }
}

function renderDataGrid(rows) {
  const wrap = document.getElementById('dataGridWrap');

  if (!rows || rows.length === 0) {
    wrap.innerHTML = `<div class="loading-cell">No records returned from live database query SELECT * FROM table LIMIT 100.</div>`;
    return;
  }

  // Schema-Aware Dynamic Rendering: Extract all N columns returned by query
  const cols = Object.keys(rows[0]);
  let html = `<table class="data-grid-table" id="dataGridTable">
    <thead><tr>
      <th class="row-num-cell">#</th>
      ${cols.map((c, i) => `<th onclick="sortDataGrid('${c}', ${i})" title="Sort by ${c}">${c}</th>`).join('')}
    </tr></thead>
    <tbody>`;

  rows.forEach((row, ri) => {
    html += `<tr><td class="row-num-cell">${ri + 1}</td>`;
    cols.forEach(c => {
      const val = row[c];
      const isNull = val === null || val === undefined || val === '';
      html += `<td class="${isNull ? 'null-cell' : ''}" title="${isNull ? 'NULL' : val}">${isNull ? 'NULL' : val}</td>`;
    });
    html += '</tr>';
  });

  html += '</tbody></table>';
  wrap.innerHTML = html;
}

function sortDataGrid(colName, colIdx) {
  if (!rawDataRows.length) return;

  if (dataSortCol === colName) {
    dataSortDir = dataSortDir === 'asc' ? 'desc' : 'asc';
  } else {
    dataSortCol = colName;
    dataSortDir = 'asc';
  }

  const sorted = [...rawDataRows].sort((a, b) => {
    const av = a[colName], bv = b[colName];
    if (av === null || av === undefined) return 1;
    if (bv === null || bv === undefined) return -1;
    const res = av < bv ? -1 : av > bv ? 1 : 0;
    return dataSortDir === 'asc' ? res : -res;
  });

  // Update header classes
  document.querySelectorAll('.data-grid-table th').forEach(th => {
    th.classList.remove('sorted-asc', 'sorted-desc');
  });
  const ths = document.querySelectorAll('.data-grid-table th');
  if (ths[colIdx + 1]) {
    ths[colIdx + 1].classList.add(dataSortDir === 'asc' ? 'sorted-asc' : 'sorted-desc');
  }

  renderDataGrid(sorted);
}

// ── Tab Switching ─────────────────────────────────
function switchTab(tab) {
  currentTab = tab;

  document.getElementById('tabBtnSchema').classList.toggle('tab-btn-active', tab === 'schema');
  document.getElementById('tabBtnData').classList.toggle('tab-btn-active', tab === 'data');
  document.getElementById('tabSchema').style.display = tab === 'schema' ? 'block' : 'none';
  document.getElementById('tabData').style.display = tab === 'data' ? 'block' : 'none';

  if (tab === 'data' && activeDataset) {
    fetchLiveDataGrid(activeDataset);
  }
}

// ── Notebook Bridge ───────────────────────────────
function launchAnalysisInNotebook() {
  if (!activeDataset) return;
  window.location.href = `/notebook?dataset=${encodeURIComponent(activeDataset)}`;
}
