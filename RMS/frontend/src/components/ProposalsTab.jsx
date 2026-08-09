import React, { useState } from 'react';

export default function ProposalsTab({ proposals, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [formData, setFormData] = useState({
    title: '', description: '', draftContent: '', background: '',
    objectives: '', methodology: '', timelineDetails: '', budgetDetails: '', submittedBy: ''
  });

  const resetForm = () => {
    setFormData({ title: '', description: '', draftContent: '', background: '',
      objectives: '', methodology: '', timelineDetails: '', budgetDetails: '', submittedBy: '' });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const url = editingId
      ? `${import.meta.env.VITE_RMS_API_URL || ''}/api/proposals/${editingId}`
      : `${import.meta.env.VITE_RMS_API_URL || ''}/api/proposals`;
    const method = editingId ? 'PUT' : 'POST';

    const body = editingId
      ? { ...formData, status: 'Submitted' }
      : { ...formData, researchId };

    const res = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (res.ok) {
      resetForm();
      onRefresh();
    }
  };

  const handleEdit = (p) => {
    setFormData({
      title: p.title, description: p.description, draftContent: p.draftContent || '',
      background: p.background || '', objectives: p.objectives || '',
      methodology: p.methodology || '', timelineDetails: p.timelineDetails || '',
      budgetDetails: p.budgetDetails || '', submittedBy: p.submittedBy || ''
    });
    setEditingId(p.id);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this proposal?')) return;
    await fetch(`${import.meta.env.VITE_RMS_API_URL || ''}/api/proposals/${id}`, { method: 'DELETE' });
    onRefresh();
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Research Proposals</h3>
        {!showForm && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ New Proposal</button>
        )}
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>{editingId ? 'Edit Proposal' : 'New Proposal'}</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
              placeholder="Title" required value={formData.title}
              onChange={e => setFormData({ ...formData, title: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Description" value={formData.description}
              onChange={e => setFormData({ ...formData, description: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Background" value={formData.background}
              onChange={e => setFormData({ ...formData, background: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Objectives" value={formData.objectives}
              onChange={e => setFormData({ ...formData, objectives: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Methodology" value={formData.methodology}
              onChange={e => setFormData({ ...formData, methodology: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Timeline Details" value={formData.timelineDetails}
              onChange={e => setFormData({ ...formData, timelineDetails: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Budget Details" value={formData.budgetDetails}
              onChange={e => setFormData({ ...formData, budgetDetails: e.target.value })} />
            <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
              placeholder="Submitted By" value={formData.submittedBy}
              onChange={e => setFormData({ ...formData, submittedBy: e.target.value })} />
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={resetForm}>Cancel</button>
              <button type="submit" className="btn btn-primary">{editingId ? 'Update' : 'Create'} Proposal</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {proposals.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No proposals created yet.</p>
        ) : (
          proposals.map(p => (
            <div key={p.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <h4 style={{ margin: '0 0 4px 0' }}>{p.title}</h4>
                  <p style={{ fontSize: '14px', margin: '4px 0', color: 'var(--text-secondary)' }}>{p.description}</p>
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px', flexWrap: 'wrap' }}>
                    <span style={{ fontSize: '12px', padding: '2px 8px', background: '#e8f0fe', borderRadius: '4px' }}>{p.status}</span>
                    <span style={{ fontSize: '12px', padding: '2px 8px', background: '#f0f0f0', borderRadius: '4px' }}>v{p.version}</span>
                    {p.submittedBy && <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>By: {p.submittedBy}</span>}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '8px', marginLeft: '16px' }}>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px' }} onClick={() => handleEdit(p)}>Edit</button>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px', color: 'var(--danger)' }} onClick={() => handleDelete(p.id)}>Delete</button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}