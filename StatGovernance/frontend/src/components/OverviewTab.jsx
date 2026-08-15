import React from 'react';

export default function OverviewTab({ kpis, onNavigate, onOpenNewRisk, onOpenNewPolicy }) {
  if (!kpis) {
    return <div style={{ padding: 32, textAlign: 'center' }}>Loading governance telemetry...</div>;
  }

  // 5x5 Matrix coordinates (Impact 1-5 on X axis, Probability 5-1 on Y axis)
  const probabilities = [5, 4, 3, 2, 1];
  const impacts = [1, 2, 3, 4, 5];

  const getMatrixCellClass = (prob, imp) => {
    const score = prob * imp;
    if (score >= 15) return 'critical';
    if (score >= 10) return 'high';
    if (score >= 5) return 'medium';
    return 'low';
  };

  return (
    <div>
      {/* Top Banner / Executive Insight */}
      <div className="glass-panel" style={{ padding: '20px 24px', marginBottom: 24, background: 'linear-gradient(135deg, #0f4c81 0%, #1e293b 100%)', color: 'white' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 16 }}>
          <div>
            <span style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: 1, color: '#93c5fd', fontWeight: 700 }}>
              Institutional Governance & Control Status
            </span>
            <h2 style={{ fontSize: 22, fontWeight: 800, marginTop: 4 }}>
              StatGate Institutional Governance Operating System
            </h2>
            <p style={{ fontSize: 13, color: '#cbd5e1', maxWidth: 650, marginTop: 4 }}>
              Live real-time monitoring of institutional policies, compliance obligations, control effectiveness, risk treatment pipelines, and committee decisions.
            </p>
          </div>
          <div style={{ display: 'flex', gap: 10 }}>
            <button className="btn btn-primary" onClick={onOpenNewPolicy} style={{ background: '#ffffff', color: '#0f4c81' }}>
              + Draft Policy
            </button>
            <button className="btn" onClick={onOpenNewRisk} style={{ background: 'rgba(255,255,255,0.15)', color: '#ffffff', border: '1px solid rgba(255,255,255,0.3)' }}>
              + Log Risk
            </button>
          </div>
        </div>
      </div>

      {/* KPI Cards Grid */}
      <div className="metrics-grid">
        <div className="glass-panel kpi-card indigo" onClick={() => onNavigate('policies')} style={{ cursor: 'pointer' }}>
          <span className="kpi-label">Active Policies</span>
          <span className="kpi-value">{kpis.policies_active ?? 0}</span>
          <span className="kpi-subtext">
            <span>{kpis.policies_under_review ?? 0} in review</span> • 
            <span style={{ color: kpis.policies_expired > 0 ? 'var(--accent-ruby)' : 'inherit' }}> {kpis.policies_expired ?? 0} overdue</span>
          </span>
        </div>

        <div className="glass-panel kpi-card emerald" onClick={() => onNavigate('compliance')} style={{ cursor: 'pointer' }}>
          <span className="kpi-label">Compliance Rate</span>
          <span className="kpi-value">{(kpis.compliance_rate ?? 100).toFixed(1)}%</span>
          <span className="kpi-subtext">
            <span>{kpis.compliance_compliant ?? 0} Compliant</span> • 
            <span> {kpis.compliance_non_compliant ?? 0} Gaps</span>
          </span>
        </div>

        <div className="glass-panel kpi-card ruby" onClick={() => onNavigate('risks')} style={{ cursor: 'pointer' }}>
          <span className="kpi-label">Active Risks</span>
          <span className="kpi-value">{(kpis.risks_critical ?? 0) + (kpis.risks_high ?? 0) + (kpis.risks_medium ?? 0) + (kpis.risks_low ?? 0)}</span>
          <span className="kpi-subtext">
            <span style={{ color: 'var(--accent-ruby)', fontWeight: 700 }}>{kpis.risks_critical ?? 0} Critical</span> • 
            <span> {kpis.risks_high ?? 0} High</span>
          </span>
        </div>

        <div className="glass-panel kpi-card gold" onClick={() => onNavigate('controls')} style={{ cursor: 'pointer' }}>
          <span className="kpi-label">Control Library</span>
          <span className="kpi-value">{kpis.controls_effective ?? 0}</span>
          <span className="kpi-subtext">
            <span>{kpis.controls_effective ?? 0} Effective</span> • 
            <span style={{ color: kpis.controls_failed > 0 ? 'var(--accent-ruby)' : 'inherit' }}> {kpis.controls_failed ?? 0} Failed</span>
          </span>
        </div>

        <div className="glass-panel kpi-card" onClick={() => onNavigate('findings')} style={{ cursor: 'pointer' }}>
          <span className="kpi-label">Audit Findings</span>
          <span className="kpi-value">{kpis.findings_open ?? 0}</span>
          <span className="kpi-subtext">
            <span style={{ color: kpis.findings_overdue > 0 ? 'var(--accent-ruby)' : 'inherit' }}>{kpis.findings_overdue ?? 0} Overdue Actions</span>
          </span>
        </div>
      </div>

      {/* Main Section Grid: 5x5 Risk Heatmap & Governance Activity */}
      <div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: 24, marginBottom: 24 }}>
        {/* 5x5 Risk Heatmap Matrix */}
        <div className="glass-panel" style={{ padding: 22 }}>
          <div className="section-header">
            <span className="section-title">
              <span>🎯</span> Institutional 5×5 Risk Matrix Heatmap
            </span>
            <button className="btn btn-secondary btn-sm" onClick={() => onNavigate('risks')}>
              View Risk Register →
            </button>
          </div>
          <p style={{ fontSize: 12.5, color: 'var(--text-secondary)', marginBottom: 14 }}>
            Distribution of identified risks by Probability (1-5) and Impact (1-5). Click any cell to inspect filtered risks.
          </p>

          <div className="risk-matrix-grid">
            <div className="matrix-axis-label">Prob \ Imp</div>
            <div className="matrix-axis-label">1 - Low</div>
            <div className="matrix-axis-label">2 - Minor</div>
            <div className="matrix-axis-label">3 - Mod</div>
            <div className="matrix-axis-label">4 - Major</div>
            <div className="matrix-axis-label">5 - Crit</div>

            {probabilities.map(prob => (
              <React.Fragment key={`prob-${prob}`}>
                <div className="matrix-axis-label">{prob} - {prob === 5 ? 'Almost Certain' : prob === 4 ? 'Likely' : prob === 3 ? 'Possible' : prob === 2 ? 'Unlikely' : 'Rare'}</div>
                {impacts.map(imp => {
                  const key = `${prob}_${imp}`;
                  const count = kpis.risk_heatmap_matrix?.[key] || 0;
                  const cellClass = getMatrixCellClass(prob, imp);
                  return (
                    <div
                      key={key}
                      className={`matrix-cell ${cellClass}`}
                      onClick={() => onNavigate('risks')}
                      title={`Probability: ${prob}, Impact: ${imp} (${count} active risks)`}
                    >
                      <span>{count > 0 ? count : '—'}</span>
                      <span className="matrix-cell-score">{prob * imp}</span>
                    </div>
                  );
                })}
              </React.Fragment>
            ))}
          </div>
        </div>

        {/* Governance Control Health & Telemetry */}
        <div className="glass-panel" style={{ padding: 22, display: 'flex', flexDirection: 'column' }}>
          <div className="section-header">
            <span className="section-title">
              <span>🛡️</span> Control & Compliance Health
            </span>
            <button className="btn btn-secondary btn-sm" onClick={() => onNavigate('compliance')}>
              Obligations Register →
            </button>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 16, marginTop: 8 }}>
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, fontWeight: 600, marginBottom: 6 }}>
                <span>Regulatory Obligations Status</span>
                <span>{kpis.compliance_compliant ?? 0} / {(kpis.compliance_compliant ?? 0) + (kpis.compliance_partially ?? 0) + (kpis.compliance_non_compliant ?? 0)} Compliant</span>
              </div>
              <div style={{ height: 10, background: '#e2e8f0', borderRadius: 6, overflow: 'hidden', display: 'flex' }}>
                <div style={{ width: `${kpis.compliance_rate ?? 100}%`, background: 'var(--accent-emerald)' }}></div>
                <div style={{ width: `${100 - (kpis.compliance_rate ?? 100)}%`, background: 'var(--accent-ruby)' }}></div>
              </div>
            </div>

            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13, fontWeight: 600, marginBottom: 6 }}>
                <span>Control Effectiveness Rate</span>
                <span>{kpis.controls_effective ?? 0} Effective Controls</span>
              </div>
              <div style={{ height: 10, background: '#e2e8f0', borderRadius: 6, overflow: 'hidden', display: 'flex' }}>
                <div style={{ width: '85%', background: 'var(--primary-color)' }}></div>
                <div style={{ width: '15%', background: 'var(--accent-gold)' }}></div>
              </div>
            </div>

            <div style={{ background: '#f8fafc', padding: 14, borderRadius: 8, border: '1px solid var(--border-light)', marginTop: 8 }}>
              <div style={{ fontSize: 12, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 8 }}>
                Institutional Governance Summary
              </div>
              <ul style={{ fontSize: 13, color: 'var(--text-dark)', paddingLeft: 18, lineHeight: 1.8 }}>
                <li><strong>{kpis.pending_approvals_count ?? 0}</strong> Policies awaiting executive approval</li>
                <li><strong>{kpis.findings_overdue ?? 0}</strong> Corrective Actions approaching SLA deadline</li>
                <li><strong>{kpis.audits_in_progress ?? 0}</strong> Audit engagements actively executing</li>
                <li><strong>{kpis.recent_decisions_count ?? 0}</strong> Verified Institutional Decision records logged</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
