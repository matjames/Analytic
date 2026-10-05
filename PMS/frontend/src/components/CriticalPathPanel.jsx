import React, { useEffect, useState } from 'react';

const authHeaders = () => {
  const token = localStorage.getItem('statgate_token') || localStorage.getItem('token');
  return token ? { Authorization: `Bearer ${token}` } : {};
};

export default function CriticalPathPanel({ project, apiBase }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    fetch(`${apiBase}/api/projects/${encodeURIComponent(project.id)}/critical-path`, { headers: authHeaders() })
      .then(response => {
        if (!response.ok) throw new Error(`Critical path failed: ${response.status}`);
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
  if (!data) return <div className="glass-panel" style={{ padding: '20px', marginBottom: '24px', color: 'var(--text-muted)', fontSize: '12px' }}>Calculating critical path...</div>;

  const path = data.criticalPath || {};
  return (
    <section className="glass-panel" style={{ padding: '20px', marginBottom: '24px' }} aria-labelledby="critical-path-title">
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: '16px', flexWrap: 'wrap' }}>
        <div>
          <h3 id="critical-path-title" style={{ fontSize: '16px', fontWeight: '800', marginBottom: '5px' }}>Critical Path Analysis</h3>
          <p style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{path.tasks?.length || 0} task(s) determine the longest schedule chain at {path.durationDays || 0} day(s).</p>
        </div>
        {data.missingDependencies?.length > 0 && <span className="badge badge-Planning">Missing links: {data.missingDependencies.length}</span>}
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '14px' }}>
        {(path.tasks || []).map(task => (
          <div key={task.id} style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '9px 10px', border: '1px solid var(--border-light)', borderRadius: '6px', background: 'var(--bg-light)' }}>
            <strong style={{ color: 'var(--primary-color)', fontFamily: 'monospace', fontSize: '11px' }}>{task.wbsCode || task.id}</strong>
            <span style={{ flex: 1, fontSize: '12px' }}>{task.title}</span>
            <span style={{ color: 'var(--text-muted)', fontSize: '11px' }}>{task.durationDays} day(s)</span>
          </div>
        ))}
      </div>
      {data.missingDependencies?.length > 0 && <p style={{ marginTop: '12px', color: 'var(--accent-warning)', fontSize: '11px' }}>Unresolved dependency references: {data.missingDependencies.join(', ')}</p>}
      <p style={{ marginTop: '12px', color: 'var(--text-muted)', fontSize: '10px' }}>{data.method}. Confirm dates and dependencies before changing the baseline.</p>
    </section>
  );
}
