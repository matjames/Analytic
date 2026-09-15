import React, { useEffect, useState } from 'react';

const authHeaders = () => {
  const token = localStorage.getItem('statgate_token') || localStorage.getItem('token');
  return token ? { Authorization: `Bearer ${token}` } : {};
};

export default function ProgressSummary({ project, apiBase }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    setError('');
    fetch(`${apiBase}/api/projects/${encodeURIComponent(project.id)}/progress-summary`, { headers: authHeaders() })
      .then(response => {
        if (!response.ok) throw new Error(`Progress summary failed: ${response.status}`);
        return response.json();
      })
      .then(nextData => {
        if (!cancelled) setData(nextData);
      })
      .catch(requestError => {
        if (!cancelled) setError(requestError.message);
      });
    return () => { cancelled = true; };
  }, [apiBase, project.id]);

  if (error) return <div className="glass-panel" style={{ padding: '20px', marginBottom: '24px', color: 'var(--accent-danger)', fontSize: '12px' }}>{error}</div>;
  if (!data) return <div className="glass-panel" style={{ padding: '20px', marginBottom: '24px', color: 'var(--text-muted)', fontSize: '12px' }}>Preparing progress summary...</div>;

  const delivery = data.delivery || {};
  const budget = data.budget || {};
  const nextMilestone = delivery.nextMilestone || {};
  const metrics = [
    ['Delivery progress', `${Number(delivery.progress || 0).toFixed(1)}%`],
    ['Tasks complete', `${delivery.completedTasks || 0}/${delivery.taskCount || 0}`],
    ['Milestones complete', `${delivery.completedMilestones || 0}/${delivery.milestoneCount || 0}`],
    ['Budget utilized', `${Number(budget.utilizationPercent || 0).toFixed(1)}%`],
    ['Open risks', `${data.openRisks || 0}`],
  ];

  return (
    <section className="glass-panel" style={{ padding: '20px', marginBottom: '24px' }} aria-labelledby="progress-summary-title">
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: '16px', flexWrap: 'wrap' }}>
        <div>
          <h3 id="progress-summary-title" style={{ fontSize: '16px', fontWeight: '800', marginBottom: '5px' }}>Automated Progress Summary</h3>
          <p style={{ fontSize: '12px', color: 'var(--text-secondary)', lineHeight: '1.5' }}>{data.summary}</p>
        </div>
        {nextMilestone.name && <div style={{ fontSize: '11px', color: 'var(--text-muted)', textAlign: 'right' }}>Next milestone<br /><strong style={{ color: 'var(--text-primary)' }}>{nextMilestone.name}</strong>{nextMilestone.dueDate ? ` · ${nextMilestone.dueDate}` : ''}</div>}
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(125px, 1fr))', gap: '8px', marginTop: '16px' }}>
        {metrics.map(([label, value]) => <div className="metric-card" key={label}><span>{label}</span><strong>{value}</strong></div>)}
      </div>
      <div style={{ marginTop: '16px' }}>
        <div style={{ fontSize: '11px', color: 'var(--text-muted)', marginBottom: '6px' }}>Recommended follow-up</div>
        <ul style={{ margin: 0, paddingLeft: '18px', color: 'var(--text-secondary)', fontSize: '12px', lineHeight: '1.7' }}>
          {(data.recommendations || []).map(item => <li key={item}>{item}</li>)}
        </ul>
      </div>
      <div style={{ marginTop: '12px', fontSize: '10px', color: 'var(--text-muted)' }}>Generated from current workspace PMS records. Confirm before taking consequential action.</div>
    </section>
  );
}
