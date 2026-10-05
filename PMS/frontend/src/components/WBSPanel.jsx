import React, { useEffect, useState } from 'react';

const authHeaders = () => {
  const token = localStorage.getItem('statgate_token') || localStorage.getItem('token');
  return token ? { Authorization: `Bearer ${token}` } : {};
};

function WBSNode({ node, depth = 0 }) {
  return (
    <div style={{ marginLeft: `${depth * 16}px` }}>
      <div style={{ display: 'flex', gap: '10px', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid var(--border-light)' }}>
        <strong style={{ minWidth: '42px', color: 'var(--primary-color)', fontFamily: 'monospace', fontSize: '11px' }}>{node.wbsCode || node.id}</strong>
        <span style={{ flex: 1, fontSize: '12px' }}>{node.title}</span>
        <span style={{ color: 'var(--text-muted)', fontSize: '10px' }}>{node.assignedTo || 'Unassigned'}</span>
        <span style={{ fontSize: '11px' }}>{Number(node.progress || 0).toFixed(0)}%</span>
      </div>
      {(node.children || []).map(child => <WBSNode key={child.id} node={child} depth={depth + 1} />)}
    </div>
  );
}

export default function WBSPanel({ project, apiBase }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;
    fetch(`${apiBase}/api/projects/${encodeURIComponent(project.id)}/wbs`, { headers: authHeaders() })
      .then(response => {
        if (!response.ok) throw new Error(`WBS failed: ${response.status}`);
        return response.json();
      })
      .then(nextData => { if (!cancelled) setData(nextData); })
      .catch(requestError => { if (!cancelled) setError(requestError.message); });
    return () => { cancelled = true; };
  }, [apiBase, project.id]);

  if (error) return <div className="glass-panel" style={{ padding: '20px', marginBottom: '24px', color: 'var(--accent-danger)', fontSize: '12px' }}>{error}</div>;
  if (!data) return <div className="glass-panel" style={{ padding: '20px', marginBottom: '24px', color: 'var(--text-muted)', fontSize: '12px' }}>Loading work breakdown...</div>;

  return (
    <section className="glass-panel" style={{ padding: '20px', marginBottom: '24px' }} aria-labelledby="wbs-title">
      <h3 id="wbs-title" style={{ fontSize: '16px', fontWeight: '800', marginBottom: '5px' }}>Hierarchical Work Breakdown Structure</h3>
      <p style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '12px' }}>{data.taskCount || 0} task(s) projected from the current workspace task hierarchy.</p>
      {(data.roots || []).length === 0 ? <p style={{ color: 'var(--text-muted)', fontSize: '12px' }}>No WBS tasks recorded.</p> : (data.roots || []).map(node => <WBSNode key={node.id} node={node} />)}
      {(data.missingParents || []).length > 0 && <p style={{ marginTop: '12px', color: 'var(--accent-warning)', fontSize: '11px' }}>Unresolved parent references: {data.missingParents.join(', ')}</p>}
      <p style={{ marginTop: '12px', color: 'var(--text-muted)', fontSize: '10px' }}>{data.method}. Confirm hierarchy before changing task ownership or dates.</p>
    </section>
  );
}
