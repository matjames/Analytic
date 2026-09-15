import React from 'react';

const currency = value => `$${Number(value || 0).toLocaleString(undefined, { maximumFractionDigits: 0 })}`;
const percent = value => `${Number(value || 0).toFixed(1)}%`;

function Breakdown({ title, items }) {
  return (
    <div className="glass-panel" style={{ padding: '20px' }}>
      <h3 style={{ fontSize: '15px', fontWeight: '800', marginBottom: '14px' }}>{title}</h3>
      {items.length === 0 ? (
        <p style={{ fontSize: '12px', color: 'var(--text-muted)' }}>No grouped data in this scope.</p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {items.map(item => (
            <div key={item.name} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '10px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', gap: '12px', fontSize: '12px' }}>
                <strong style={{ color: 'var(--text-primary)' }}>{item.name}</strong>
                <span>{item.projectCount} project{item.projectCount === 1 ? '' : 's'}</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '5px', fontSize: '11px', color: 'var(--text-muted)' }}>
                <span>{percent(item.averageProgress)} average progress</span>
                <span>{currency(item.budgetTotal)} allocated</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function PortfolioDashboard({ data, portfolio, programme, onPortfolioChange, onProgrammeChange }) {
  if (!data) {
    return (
      <div className="glass-panel" style={{ padding: '20px', marginBottom: '24px' }}>
        <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>Loading portfolio and programme intelligence...</p>
      </div>
    );
  }

  const summary = data.summary || {};
  const portfolioBreakdown = data.portfolioBreakdown || [];
  const programmeBreakdown = data.programmeBreakdown || [];
  const projects = data.projects || [];
  const stages = Object.entries(data.stageDistribution || {});

  return (
    <section style={{ marginBottom: '24px' }} aria-label="Portfolio and programme dashboard">
      <div className="page-title-row" style={{ marginBottom: '14px' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800' }}>Portfolio &amp; Programme Control Tower</h2>
          <span className="page-subtitle">Workspace-scoped delivery health and investment view</span>
        </div>
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          <select value={portfolio} onChange={e => onPortfolioChange(e.target.value)} aria-label="Filter by portfolio">
            <option value="">All portfolios</option>
            {portfolioBreakdown.filter(item => item.name !== 'Unassigned').map(item => <option key={item.name} value={item.name}>{item.name}</option>)}
          </select>
          <select value={programme} onChange={e => onProgrammeChange(e.target.value)} aria-label="Filter by programme">
            <option value="">All programmes</option>
            {programmeBreakdown.filter(item => item.name !== 'Unassigned').map(item => <option key={item.name} value={item.name}>{item.name}</option>)}
          </select>
        </div>
      </div>

      <div className="stat-grid" style={{ marginBottom: '16px' }}>
        <div className="glass-panel stat-card">
          <span className="stat-label">Projects In Scope</span>
          <span className="stat-value">{summary.projectCount || 0}</span>
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{percent(summary.averageProgress)} average delivery progress</span>
        </div>
        <div className="glass-panel stat-card">
          <span className="stat-label">Allocated Budget</span>
          <span className="stat-value" style={{ color: 'var(--primary-color)' }}>{currency(summary.budgetTotal)}</span>
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{currency(summary.spentTotal)} spent</span>
        </div>
        <div className="glass-panel stat-card">
          <span className="stat-label">Delivery Attention</span>
          <span className="stat-value" style={{ color: summary.risksTotal || summary.issuesTotal ? 'var(--accent-warning)' : 'var(--success-color)' }}>
            {(summary.risksTotal || 0) + (summary.issuesTotal || 0)}
          </span>
          <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{summary.risksTotal || 0} risks, {summary.issuesTotal || 0} issues</span>
        </div>
      </div>

      <div className="portfolio-dashboard-columns" style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: '16px' }}>
        <Breakdown title="By Portfolio" items={portfolioBreakdown} />
        <Breakdown title="By Programme" items={programmeBreakdown} />
        <div className="glass-panel" style={{ padding: '20px' }}>
          <h3 style={{ fontSize: '15px', fontWeight: '800', marginBottom: '14px' }}>Lifecycle Distribution</h3>
          {stages.length === 0 ? (
            <p style={{ fontSize: '12px', color: 'var(--text-muted)' }}>No projects in this scope.</p>
          ) : stages.map(([stage, count]) => (
            <div key={stage} style={{ marginBottom: '11px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px', marginBottom: '4px' }}>
                <span>{stage}</span><strong>{count}</strong>
              </div>
              <div className="project-progress-bar"><div className="project-progress-fill" style={{ width: `${summary.projectCount ? (count / summary.projectCount) * 100 : 0}%` }} /></div>
            </div>
          ))}
        </div>
      </div>

      <div className="glass-panel" style={{ padding: '20px', marginTop: '16px' }}>
        <h3 style={{ fontSize: '15px', fontWeight: '800', marginBottom: '12px' }}>Projects In Current Scope</h3>
        {projects.length === 0 ? (
          <p style={{ fontSize: '12px', color: 'var(--text-muted)' }}>No projects match the selected filters.</p>
        ) : (
          <div style={{ display: 'grid', gap: '8px' }}>
            {projects.map(project => (
              <div key={project.id} className="portfolio-project-row" style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr 0.8fr 0.8fr', gap: '12px', alignItems: 'center', padding: '10px 0', borderBottom: '1px solid var(--border-light)', fontSize: '12px' }}>
                <div><strong>{project.name}</strong><span style={{ display: 'block', color: 'var(--text-muted)', marginTop: '3px' }}>{project.code} | {project.programme || 'No programme'}</span></div>
                <span>{project.portfolio || 'No portfolio'}</span>
                <span><strong>{percent(project.progress)}</strong><span style={{ display: 'block', color: 'var(--text-muted)' }}>{project.stage}</span></span>
                <span style={{ textAlign: 'right' }}>{currency(project.spentTotal)} / {currency(project.budgetTotal)}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
