/* ═════════════════════════════════════════════════
   jupyter_sandbox.js – Multi-Cell Jupyter-Style Notebook
   ═════════════════════════════════════════════════ */

let cellCounter = 0;
let activeSessionId = `jupyter_session_${Date.now()}`;

function createCell(initialCode = '') {
  cellCounter++;
  const cellId = cellCounter;
  const container = document.getElementById('cellsContainer');
  if (!container) return;

  const cellCard = document.createElement('div');
  cellCard.className = 'glass-panel';
  cellCard.id = `cell-card-${cellId}`;
  cellCard.style.marginBottom = '16px';

  cellCard.innerHTML = `
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px;">
      <div style="font-size:0.82rem; font-weight:600; color:var(--accent-cyan); font-family:'JetBrains Mono', monospace;">
        In [${cellId}]:
      </div>
      <div style="display:flex; gap:8px;">
        <button class="btn-primary" style="padding:4px 10px; font-size:0.78rem;" onclick="runCell(${cellId})">▶ Run Cell</button>
        <button class="btn-primary" style="padding:4px 10px; font-size:0.78rem; background:linear-gradient(135deg, var(--accent-gold), var(--accent-purple));" onclick="publishCellToDashboard(${cellId})">🚀 Publish to Dashboard</button>
        <button style="background:transparent; border:1px solid var(--border-color); color:var(--text-muted); padding:4px 10px; border-radius:5px; cursor:pointer; font-size:0.78rem;" onclick="deleteCell(${cellId})">🗑 Delete</button>
      </div>
    </div>
    <textarea id="cell-code-${cellId}" class="code-editor" style="height:140px; font-family:'JetBrains Mono', monospace;">${initialCode}</textarea>
    <div style="margin-top:8px;">
      <div style="font-size:0.75rem; color:var(--text-muted); margin-bottom:4px; text-transform:uppercase; letter-spacing:0.5px;">Out [${cellId}]:</div>
      <div id="cell-output-${cellId}" class="console-output" style="min-height:40px; background:rgba(6,9,15,0.7);">Cell idle. Press ▶ Run Cell to execute.</div>
    </div>
  `;

  container.appendChild(cellCard);
}

function addEmptyCell() {
  createCell('# Write custom Python code using `from statgate import engine` or pandas\n');
}

function deleteCell(cellId) {
  const card = document.getElementById(`cell-card-${cellId}`);
  if (card) card.remove();
}

async function runCell(cellId) {
  const codeEl = document.getElementById(`cell-code-${cellId}`);
  const outEl = document.getElementById(`cell-output-${cellId}`);
  if (!codeEl || !outEl) return;

  const code = codeEl.value;
  const dsSelect = document.getElementById('kaggleDatasetSelect');
  const selectedTable = dsSelect ? dsSelect.value : '';

  outEl.innerText = '⚡ Executing in persistent PyDuckDB kernel session...';

  try {
    const res = await fetch('/api/notebook/execute', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        code: code,
        tenant_id: typeof getCurrentTenant === 'function' ? getCurrentTenant() : 'tenant-alpha',
        kaggle_table: selectedTable,
        session_id: activeSessionId
      })
    });

    const data = await res.json();
    if (data.success) {
      outEl.innerText = data.output;
    } else {
      outEl.innerText = 'Error: ' + data.error;
    }
  } catch (e) {
    outEl.innerText = 'Execution request failed: ' + e.message;
  }
}

async function saveNotebookSession() {
  const dsSelect = document.getElementById('kaggleDatasetSelect');
  const selectedTable = dsSelect ? dsSelect.value : 'unlinked';

  const cells = [];
  document.querySelectorAll('[id^="cell-code-"]').forEach(el => {
    cells.push(el.value);
  });

  const sessionPayload = {
    session_id: activeSessionId,
    dataset: selectedTable,
    saved_at: new Date().toISOString(),
    cells: cells
  };

  try {
    const res = await fetch('/api/dashboard/save', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        dashboard_id: `notebook_${selectedTable}`,
        metadata: sessionPayload
      })
    });
    const data = await res.json();
    alert(`Notebook session saved successfully as asset linked to dataset '${selectedTable}'!`);
  } catch (e) {
    alert('Failed to save notebook session: ' + e.message);
  }
}

async function publishCellToDashboard(cellId) {
  const code = document.getElementById(`cell-code-${cellId}`)?.value || '';
  const output = document.getElementById(`cell-output-${cellId}`)?.innerText || '';
  const dsSelect = document.getElementById('kaggleDatasetSelect');
  const selectedTable = dsSelect ? dsSelect.value : 'unlinked';

  const widgetAsset = {
    id: `widget_cell_${cellId}_${Date.now()}`,
    asset_type: 'widget',
    content_definition: {
      title: `Notebook Analysis Widget (${selectedTable})`,
      code: code,
      output: output,
      published_at: new Date().toISOString(),
      dataset: selectedTable
    },
    version_tag: '1.0.0'
  };

  try {
    const res = await fetch('/api/v1/assets/save', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(widgetAsset)
    });
    const data = await res.json();
    alert(`🚀 Cell In [${cellId}] output successfully published to live Dashboard Canvas!`);
  } catch (e) {
    alert('Failed to publish cell to dashboard: ' + e.message);
  }
}
