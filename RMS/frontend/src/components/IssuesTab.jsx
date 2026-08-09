import React, { useState } from 'react';

export default function IssuesTab({ issues, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({
    title: '', description: '', category: '', priority: 'Medium',
    status: 'Open', owner: '', dueDate: ''
  });

  const handleSubmit = async (e) => {
    e.preventDefault();
    await fetch(`${API_URL}/api/issues`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...formData, researchId })
    });
    setShowForm(false);
    setFormData({ title: '', description: '', category: '', priority: 'Medium', status: 'Open', owner: '', dueDate: '' });
    onRefresh();
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Issue Tracker</h3>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Log Issue</button>
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>New Issue</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
              placeholder="Issue Title" required value={formData.title}
              onChange={e => setFormData({ ...formData, title: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Description" value={formData.description}
              onChange={e => setFormData({ ...formData, description: e.target.value })} />
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Category" value={formData.category}
                onChange={e => setFormData({ ...formData, category: e.target.value })} />
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.priority} onChange={e => setFormData({ ...formData, priority: e.target.value })}>
                <option>Low</option><option>Medium</option><option>High</option>
              </select>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Owner" value={formData.owner}
                onChange={e => setFormData({ ...formData, owner: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="date" value={formData.dueDate}
                onChange={e => setFormData({ ...formData, dueDate: e.target.value })} />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Log Issue</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {issues.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No issues logged.</p>
        ) : (
          issues.map(i => (
            <div key={i.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <h4 style={{ margin: '0 0 4px 0' }}>{i.title}</h4>
                  <p style={{ fontSize: '14px', margin: '4px 0', color: 'var(--text-secondary)' }}>{i.description}</p>
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px', flexWrap: 'wrap', fontSize: '12px' }}>
                    <span style={{ padding: '2px 8px', background: '#f0f0f0', borderRadius: '4px' }}>{i.category}</span>
                    <span style={{ padding: '2px 8px', background: i.priority === 'High' ? '#fce8e8' : i.priority === 'Medium' ? '#fff3cd' : '#e8f5e9', borderRadius: '4px' }}>{i.priority}</span>
                    <span style={{ padding: '2px 8px', background: '#e8f0fe', borderRadius: '4px' }}>{i.status}</span>
                    {i.owner && <span style={{ color: 'var(--text-secondary)' }}>Owner: {i.owner}</span>}
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}