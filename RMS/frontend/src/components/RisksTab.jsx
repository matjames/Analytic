import React, { useState } from 'react';

export default function RisksTab({ risks, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({
    description: '', category: '', probability: 'Medium', impact: 'Medium',
    mitigation: '', owner: '', dueDate: ''
  });

  const handleSubmit = async (e) => {
    e.preventDefault();
    await fetch(`${API_URL}/api/risks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...formData, researchId })
    });
    setShowForm(false);
    setFormData({ description: '', category: '', probability: 'Medium', impact: 'Medium', mitigation: '', owner: '', dueDate: '' });
    onRefresh();
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Risk Register</h3>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Log Risk</button>
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>New Risk</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Risk Description" required value={formData.description}
              onChange={e => setFormData({ ...formData, description: e.target.value })} />
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Category" value={formData.category}
                onChange={e => setFormData({ ...formData, category: e.target.value })} />
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.probability} onChange={e => setFormData({ ...formData, probability: e.target.value })}>
                <option>Low</option><option>Medium</option><option>High</option>
              </select>
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.impact} onChange={e => setFormData({ ...formData, impact: e.target.value })}>
                <option>Low</option><option>Medium</option><option>High</option>
              </select>
            </div>
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '60px' }}
              placeholder="Mitigation Strategy" value={formData.mitigation}
              onChange={e => setFormData({ ...formData, mitigation: e.target.value })} />
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Owner" value={formData.owner}
                onChange={e => setFormData({ ...formData, owner: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="date" value={formData.dueDate}
                onChange={e => setFormData({ ...formData, dueDate: e.target.value })} />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Log Risk</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {risks.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No risks logged.</p>
        ) : (
          risks.map(r => (
            <div key={r.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <p style={{ fontSize: '14px', fontWeight: '600', margin: '0 0 4px 0' }}>{r.description}</p>
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px', flexWrap: 'wrap', fontSize: '12px' }}>
                    <span style={{ padding: '2px 8px', background: '#fff3cd', borderRadius: '4px' }}>{r.category}</span>
                    <span style={{ padding: '2px 8px', background: '#f0f0f0', borderRadius: '4px' }}>P: {r.probability}</span>
                    <span style={{ padding: '2px 8px', background: '#f0f0f0', borderRadius: '4px' }}>I: {r.impact}</span>
                    <span style={{ padding: '2px 8px', background: '#e8f0fe', borderRadius: '4px' }}>{r.status}</span>
                    {r.owner && <span style={{ color: 'var(--text-secondary)' }}>Owner: {r.owner}</span>}
                  </div>
                  {r.mitigation && <p style={{ fontSize: '13px', marginTop: '8px', color: 'var(--text-secondary)' }}>Mitigation: {r.mitigation}</p>}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}