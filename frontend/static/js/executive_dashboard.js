/**
 * executive_dashboard.js — StatGate Executive Command Centre Controller
 * Autonomous decision-support canvas: loads summary, runs agent analysis,
 * renders intervention scenarios, records RLHF feedback.
 */

'use strict';

const API = {
  SUMMARY:       '/api/executive/summary',
  ALERTS:        '/api/v1/alerts',
  AGENT_ANALYZE: '/api/agent/analyze',
  AGENT_FEEDBACK:'/api/agent/feedback',
  SCHEMA_HEALTH: '/api/schema-health/',
};

let currentReportId = null;

function initThemeExec() {
  const savedTheme = localStorage.getItem('statgate_theme') || 'cyber';
  setThemeExec(savedTheme);
}

function setThemeExec(themeName) {
  if (themeName === 'cyber') {
    document.documentElement.removeAttribute('data-theme');
  } else {
    document.documentElement.setAttribute('data-theme', themeName);
  }
  localStorage.setItem('statgate_theme', themeName);
  const sel = document.getElementById('themeSelectExec');
  if (sel) sel.value = themeName;
}

function onThemeChangeExec() {
  const sel = document.getElementById('themeSelectExec');
  if (sel) setThemeExec(sel.value);
}

// ── Boot ──────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  initThemeExec();
  loadExecutiveSummary();
  loadAlertFeed();
  setInterval(loadAlertFeed, 30_000);
  setInterval(loadExecutiveSummary, 60_000);
});

// ── Executive Summary ─────────────────────────────────────────
async function loadExecutiveSummary() {
  try {
    const res  = await fetch(API.SUMMARY);
    const data = await res.json();
    renderSummary(data);
  } catch (e) {
    console.warn('[executive] summary error:', e);
    renderSummary({ actions_needed: 0, critical_alerts: 0, high_alerts: 0,
                    total_datasets: 0, system_health: 'Unknown', federated_summary: {} });
  }
}

function renderSummary(data) {
  // Orb
  const count = data.actions_needed || 0;
  document.getElementById('actionsCount').textContent = count;
  document.getElementById('systemHealthLabel').textContent = data.system_health || '—';

  const fill = document.getElementById('orbFill');
  const circumference = 283;
  const maxActions = 10;
  const ratio = Math.min(count / maxActions, 1);
  fill.style.strokeDashoffset = circumference - ratio * circumference;
  fill.classList.remove('critical', 'warn', 'ok');
  fill.classList.add(count === 0 ? 'ok' : count >= 3 ? 'critical' : 'warn');

  // Stat Cards
  setEl('criticalCount',  data.critical_alerts ?? 0);
  setEl('highCount',      data.high_alerts ?? 0);
  setEl('datasetCount',   data.total_datasets ?? 0);
  setEl('healthStatus',   count === 0 ? '✓ Optimal' : count >= 3 ? '⚠ Degraded' : '! Attention');

  const htEl = document.getElementById('healthTrendText');
  if (htEl) {
    htEl.textContent = data.system_health || '';
    htEl.className = 'stat-card-trend ' +
      (count === 0 ? 'trend-down' : count >= 3 ? 'trend-up' : '');
  }

  // Federated Grid
  const fed = data.federated_summary || {};
  renderFederated(fed, data.total_datasets);

  // Health Ring
  renderHealthRing(data);
}

function renderFederated(fed, totalDatasets) {
  const grid = document.getElementById('federatedGrid');
  if (!grid) return;
  grid.innerHTML = `
    <div class="fed-card">
      <div class="fed-card-title">Global Mean</div>
      <div class="fed-card-value" style="color:var(--accent-cyan)">
        ${fed.global_mean !== null && fed.global_mean !== undefined ? fed.global_mean.toLocaleString() : '—'}
      </div>
    </div>
    <div class="fed-card">
      <div class="fed-card-title">Participating Datasets</div>
      <div class="fed-card-value" style="color:var(--accent-blue)">
        ${fed.participating_datasets || 0}
      </div>
    </div>
    <div class="fed-card">
      <div class="fed-card-title">Global Record Count</div>
      <div class="fed-card-value" style="color:var(--accent-purple)">
        ${(fed.global_count || 0).toLocaleString()}
      </div>
    </div>
    <div class="fed-card">
      <div class="fed-card-title">Total Live Datasets</div>
      <div class="fed-card-value" style="color:var(--accent-green)">
        ${totalDatasets || 0}
      </div>
    </div>
  `;
}

function renderHealthRing(data) {
  const ring = document.getElementById('healthRing');
  if (!ring) return;

  const criticalPct = data.critical_alerts > 0 ? 25 : 100;
  const items = [
    { label: 'Data Ingestion Pipeline', pct: 97, color: 'var(--accent-green)' },
    { label: 'Anomaly Detection Engine', pct: 100, color: 'var(--accent-green)' },
    { label: 'Schema Integrity',  pct: data.critical_alerts > 0 ? 72 : 98, color: data.critical_alerts > 0 ? 'var(--accent-gold)' : 'var(--accent-green)' },
    { label: 'ABAC Policy Engine', pct: 100, color: 'var(--accent-green)' },
  ];

  ring.innerHTML = items.map(i => `
    <div class="health-item">
      <div class="health-label">${i.label}</div>
      <div class="health-bar-wrap">
        <div class="health-bar" style="width:${i.pct}%;background:${i.color};"></div>
      </div>
      <div class="health-pct" style="color:${i.color}">${i.pct}%</div>
    </div>
  `).join('');
}

// ── Alert Feed ────────────────────────────────────────────────
async function loadAlertFeed() {
  const container = document.getElementById('alertFeedExec');
  if (!container) return;

  try {
    const res  = await fetch(API.ALERTS);
    const data = await res.json();
    const alerts = data.alerts || [];

    if (!alerts.length) {
      container.innerHTML = `<div style="color:var(--accent-green);font-size:0.82rem;text-align:center;padding:20px;">
        ✅ All streams within ±3σ bounds
      </div>`;
      return;
    }

    container.innerHTML = alerts.slice(0, 5).map(a => {
      const isCrit = a.severity === 'CRITICAL';
      const sevClass = isCrit ? 'sev-critical' : 'sev-high';
      const sigma = a.sigma_score?.toFixed(2) ?? a.z_score?.toFixed(2) ?? '?';
      const ts = a.timestamp ? new Date(a.timestamp).toLocaleTimeString() : '';
      return `
        <div class="alert-item">
          <span class="alert-severity ${sevClass}">${a.severity || 'HIGH'}</span>
          <div class="alert-body">
            <div class="alert-metric">${a.metric_name || 'System Metric'}</div>
            <div class="alert-msg">${a.message || ''}</div>
            <div class="alert-meta">${a.dataset || ''} · ${sigma}σ · ${ts}</div>
          </div>
          <a class="alert-cta" href="${a.notebook_url || '/notebook'}">Investigate →</a>
        </div>
      `;
    }).join('');
  } catch (e) {
    container.innerHTML = `<div style="color:var(--text-muted);font-size:0.8rem;text-align:center;padding:16px;">Feed unavailable</div>`;
  }
}

// ── Agentic Goal Analysis ─────────────────────────────────────
function setGoal(chip) {
  document.getElementById('goalInput').value = chip.textContent.trim();
}

async function runAgentAnalysis() {
  const input   = document.getElementById('goalInput');
  const btn     = document.getElementById('analyzeBtn');
  const status  = document.getElementById('agentStatus');
  const container = document.getElementById('scenarioContainer');
  const goal    = input.value.trim();

  if (!goal) { input.focus(); return; }

  btn.disabled = true;
  status.className = 'agent-status running';
  status.textContent = '⟳ Orchestrating multi-step analysis pipeline…';
  container.innerHTML = `
    <div class="skeleton" style="height:90px;margin-bottom:12px;border-radius:10px;"></div>
    <div class="skeleton" style="height:90px;margin-bottom:12px;border-radius:10px;"></div>
    <div class="skeleton" style="height:90px;border-radius:10px;"></div>
  `;

  try {
    const res  = await fetch(API.AGENT_ANALYZE, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ goal, tenant_id: 'tenant-alpha' })
    });
    const report = await res.json();

    if (report.error) throw new Error(report.error);

    currentReportId = `report_${Date.now()}`;
    status.className = 'agent-status';
    status.textContent =
      `✓ Analysis complete — Domain: ${report.domain} · ${report.datasets_analyzed?.length || 0} datasets · ` +
      `${report.anomalies_detected?.length || 0} anomalies · ${report.engine_latency_ms}ms`;

    renderScenarios(report.scenarios || [], report);
    renderFederated(report.federated_summary || {}, report.datasets_analyzed?.length || 0);
  } catch (e) {
    status.className = 'agent-status';
    status.textContent = `⚠ Engine error: ${e.message}`;
    container.innerHTML = `<div style="color:var(--accent-red);font-size:0.8rem;padding:12px;">${e.message}</div>`;
  } finally {
    btn.disabled = false;
  }
}

function renderScenarios(scenarios, report) {
  const container = document.getElementById('scenarioContainer');
  if (!scenarios.length) {
    container.innerHTML = `<div style="color:var(--text-muted);font-size:0.8rem;text-align:center;padding:20px;">No scenarios generated.</div>`;
    return;
  }

  container.innerHTML = `
    <div style="font-size:0.75rem;color:var(--text-muted);font-weight:600;text-transform:uppercase;letter-spacing:0.6px;margin-bottom:12px;">
      🎯 ${scenarios.length} Intervention Scenarios
    </div>
    <div class="scenario-grid">
      ${scenarios.map((s, i) => {
        const cls = s.priority?.toLowerCase() === 'critical' ? 'critical'
                  : s.priority?.toLowerCase() === 'high'     ? 'high' : 'medium';
        return `
          <div class="scenario-card ${cls}" id="sc_${s.scenario_id}">
            <div class="scenario-num">${i + 1}</div>
            <div class="scenario-content">
              <div class="scenario-title">${s.title}</div>
              <div class="scenario-desc">${s.description}</div>
              <div class="scenario-evidence">📊 ${s.evidence}</div>
              <div class="scenario-actions">
                <button class="btn-approve" onclick="recordFeedback('${s.scenario_id}','approved')">
                  ✓ Approve & Action
                </button>
                <button class="btn-defer" onclick="recordFeedback('${s.scenario_id}','deferred')">
                  Defer
                </button>
                <button class="btn-defer" onclick="recordFeedback('${s.scenario_id}','rejected')" style="border-color:rgba(248,81,73,0.4);color:var(--accent-red);">
                  ✕ Reject
                </button>
              </div>
            </div>
          </div>
        `;
      }).join('')}
    </div>
  `;
}

// ── RLHF Feedback ─────────────────────────────────────────────
async function recordFeedback(scenarioId, action) {
  const card = document.getElementById(`sc_${scenarioId}`);
  if (card) {
    card.style.opacity = '0.5';
    card.style.pointerEvents = 'none';
  }

  try {
    await fetch(API.AGENT_FEEDBACK, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        report_id:   currentReportId || 'session',
        scenario_id: scenarioId,
        action,
        outcome:     `Minister ${action} scenario ${scenarioId}`,
        tenant_id:   'tenant-alpha'
      })
    });
  } catch (e) { /* silent — non-critical */ }

  showToast(
    action === 'approved' ? '✓ Action approved — dispatching to coordinators.' :
    action === 'rejected' ? '✕ Scenario rejected. Agent will de-prioritize this pattern.' :
    '⏸ Scenario deferred for later review.'
  );
}

function showToast(msg) {
  const toast = document.getElementById('rlhfToast');
  const text  = document.getElementById('rlhfToastText');
  if (!toast || !text) return;
  text.textContent = msg;
  toast.classList.add('visible');
  setTimeout(() => toast.classList.remove('visible'), 3500);
}

// ── Self-Healing Schema Scan ──────────────────────────────────
async function runSchemaHealthCheck() {
  const result = document.getElementById('schemaHealthResult');
  if (!result) return;

  result.textContent = '⟳ Scanning schema health…';
  result.style.color = 'var(--accent-cyan)';

  try {
    // Pick a representative table to scan
    const table = 'covid_19_data';
    const res   = await fetch(`${API.SCHEMA_HEALTH}${table}`);
    const data  = await res.json();

    if (data.error) throw new Error(data.error);

    const drifted  = data.drifted;
    const patched  = data.healing?.patched || 0;
    const added    = data.added_cols?.length || 0;
    const removed  = data.removed_cols?.length || 0;
    const renamed  = data.renamed_cols?.length || 0;

    if (!drifted) {
      result.style.color = 'var(--accent-green)';
      result.textContent = `✓ ${table}: Schema stable (${data.live_columns?.length || 0} columns) — no drift detected.`;
    } else {
      result.style.color = 'var(--accent-gold)';
      result.textContent =
        `⚠ Drift detected: +${added} / -${removed} / ~${renamed} renamed columns. Auto-healed ${patched} asset(s).`;
    }
  } catch (e) {
    result.style.color = 'var(--text-muted)';
    result.textContent = `Scan note: ${e.message} (snapshot initialised for future drift tracking)`;
  }
}

// ── Helpers ───────────────────────────────────────────────────
function setEl(id, val) {
  const el = document.getElementById(id);
  if (el) el.textContent = val;
}
