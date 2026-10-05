import React, { useEffect, useState } from 'react';

const authHeaders = () => {
  const token = localStorage.getItem('statgate_token') || localStorage.getItem('token');
  return token ? { Authorization: `Bearer ${token}` } : {};
};

export default function ResourcePlanner({ project, apiBase }) {
  const [allocations, setAllocations] = useState([]);
  const [form, setForm] = useState({ resourceName: '', role: '', allocationPercent: 100, startDate: '', endDate: '', notes: '' });
  const [error, setError] = useState('');

  const load = () => fetch(`${apiBase}/api/projects/${encodeURIComponent(project.id)}/resources`, { headers: authHeaders() })
    .then(response => {
      if (!response.ok) throw new Error(`Resource planner failed: ${response.status}`);
      return response.json();
    })
    .then(setAllocations)
    .catch(requestError => setError(requestError.message));

  useEffect(() => { load(); }, [apiBase, project.id]);

  const update = event => setForm({ ...form, [event.target.name]: event.target.value });
  const create = event => {
    event.preventDefault();
    setError('');
    fetch(`${apiBase}/api/resource-allocations`, {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...form, projectId: project.id, allocationPercent: Number(form.allocationPercent) }),
    })
      .then(response => {
        if (!response.ok) return response.json().catch(() => ({})).then(body => { throw new Error(body.error || `Allocation failed: ${response.status}`); });
        return response.json();
      })
      .then(() => {
        setForm({ resourceName: '', role: '', allocationPercent: 100, startDate: '', endDate: '', notes: '' });
        return load();
      })
      .catch(requestError => setError(requestError.message));
  };

  const remove = id => fetch(`${apiBase}/api/resource-allocations/${encodeURIComponent(id)}`, { method: 'DELETE', headers: authHeaders() })
    .then(response => {
      if (!response.ok) throw new Error(`Remove failed: ${response.status}`);
      return load();
    })
    .catch(requestError => setError(requestError.message));

  return (
    <section className="glass-panel" style={{ padding: '20px', marginBottom: '24px' }} aria-labelledby="resource-planner-title">
      <h3 id="resource-planner-title" style={{ fontSize: '16px', fontWeight: '800', marginBottom: '5px' }}>Resource Planner</h3>
      <p style={{ fontSize: '12px', color: 'var(--text-secondary)', marginBottom: '14px' }}>Assign people or teams to this project with a dated capacity allocation.</p>
      <form onSubmit={create} style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))', gap: '8px', marginBottom: '14px' }}>
        <input name="resourceName" value={form.resourceName} onChange={update} placeholder="Resource name" aria-label="Resource name" required />
        <input name="role" value={form.role} onChange={update} placeholder="Role" aria-label="Role" />
        <input name="allocationPercent" type="number" min="1" max="100" value={form.allocationPercent} onChange={update} placeholder="Capacity %" aria-label="Allocation percentage" required />
        <input name="startDate" type="date" value={form.startDate} onChange={update} aria-label="Allocation start date" />
        <input name="endDate" type="date" value={form.endDate} onChange={update} aria-label="Allocation end date" />
        <button className="btn btn-primary" type="submit">Add allocation</button>
      </form>
      {error && <p style={{ color: 'var(--accent-danger)', fontSize: '12px', marginBottom: '10px' }}>{error}</p>}
      {allocations.length === 0 ? <p style={{ color: 'var(--text-muted)', fontSize: '12px' }}>No resource allocations recorded.</p> : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {allocations.map(allocation => <div key={allocation.id} style={{ display: 'flex', alignItems: 'center', gap: '10px', padding: '9px 10px', border: '1px solid var(--border-light)', borderRadius: '6px', background: 'var(--bg-light)' }}>
            <strong style={{ flex: 1, fontSize: '12px' }}>{allocation.resourceName}</strong>
            <span style={{ color: 'var(--text-secondary)', fontSize: '11px' }}>{allocation.role || 'Unspecified role'}</span>
            <span style={{ fontSize: '11px' }}>{allocation.allocationPercent}%</span>
            <span style={{ color: 'var(--text-muted)', fontSize: '10px' }}>{allocation.startDate || 'Open'} to {allocation.endDate || 'Open'}</span>
            <button className="btn btn-danger" type="button" onClick={() => remove(allocation.id)} aria-label={`Remove ${allocation.resourceName}`}>Remove</button>
          </div>)}
        </div>
      )}
    </section>
  );
}
