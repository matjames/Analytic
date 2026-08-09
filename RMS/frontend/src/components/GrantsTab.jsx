import React, { useState } from 'react';

export default function GrantsTab({ grants, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [formData, setFormData] = useState({
    opportunityName: '', donorName: '', contractNumber: '',
    budgetAllocated: 0, currency: 'USD', startDate: '', endDate: '', notes: ''
  });

  const resetForm = () => {
    setFormData({ opportunityName: '', donorName: '', contractNumber: '',
      budgetAllocated: 0, currency: 'USD', startDate: '', endDate: '', notes: '' });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const url = editingId
      ? `${import.meta.env.VITE_RMS_API_URL || ''}/api/grants/${editingId}`
      : `${import.meta.env.VITE_RMS_API_URL || ''}/api/grants`;
    const method = editingId ? 'PUT' : 'POST';

    const body = editingId ? formData : { ...formData, researchId };

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

  const handleEdit = (g) => {
    setFormData({
      opportunityName: g.opportunityName || '', donorName: g.donorName || '',
      contractNumber: g.contractNumber || '', budgetAllocated: g.budgetAllocated || 0,
      currency: g.currency || 'USD', startDate: g.startDate || '', endDate: g.endDate || '',
      notes: g.notes || ''
    });
    setEditingId(g.id);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this grant record?')) return;
    await fetch(`${import.meta.env.VITE_RMS_API_URL || ''}/api/grants/${id}`, { method: 'DELETE' });
    onRefresh();
  };

  const totalAllocated = grants.reduce((sum, g) => sum + (g.budgetAllocated || 0), 0);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <div>
          <h3>Funding & Grants</h3>
          <p style={{ margin: '4px 0 0 0', fontSize: '14px', color: 'var(--text-secondary)' }}>
            Total Allocated: <strong>${totalAllocated.toLocaleString()}</strong>
          </p>
        </div>
        {!showForm && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Add Grant</button>
        )}
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>{editingId ? 'Edit Grant' : 'New Grant'}</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Opportunity Name" required value={formData.opportunityName}
                onChange={e => setFormData({ ...formData, opportunityName: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Donor Name" value={formData.donorName}
                onChange={e => setFormData({ ...formData, donorName: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Contract Number" value={formData.contractNumber}
                onChange={e => setFormData({ ...formData, contractNumber: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="number" placeholder="Budget Allocated" value={formData.budgetAllocated}
                onChange={e => setFormData({ ...formData, budgetAllocated: parseFloat(e.target.value) || 0 })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Currency" value={formData.currency}
                onChange={e => setFormData({ ...formData, currency: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="date" value={formData.startDate}
                onChange={e => setFormData({ ...formData, startDate: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="date" value={formData.endDate}
                onChange={e => setFormData({ ...formData, endDate: e.target.value })} />
            </div>
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Notes" value={formData.notes}
              onChange={e => setFormData({ ...formData, notes: e.target.value })} />
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={resetForm}>Cancel</button>
              <button type="submit" className="btn btn-primary">{editingId ? 'Update' : 'Create'} Grant</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {grants.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No grant records.</p>
        ) : (
          grants.map(g => (
            <div key={g.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <h4 style={{ margin: '0 0 4px 0' }}>{g.opportunityName}</h4>
                  <p style={{ fontSize: '14px', margin: '4px 0', color: 'var(--text-secondary)' }}>
                    Donor: {g.donorName || '—'} {g.contractNumber && `| Contract: ${g.contractNumber}`}
                  </p>
                  <div style={{ display: 'flex', gap: '12px', marginTop: '8px', flexWrap: 'wrap', fontSize: '13px' }}>
                    <span style={{ fontWeight: '600', color: 'var(--success)' }}>
                      {g.currency} {g.budgetAllocated.toLocaleString()}
                    </span>
                    {g.startDate && <span style={{ color: 'var(--text-secondary)' }}>From: {g.startDate}</span>}
                    {g.endDate && <span style={{ color: 'var(--text-secondary)' }}>To: {g.endDate}</span>}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '8px', marginLeft: '16px' }}>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px' }} onClick={() => handleEdit(g)}>Edit</button>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px', color: 'var(--danger)' }} onClick={() => handleDelete(g.id)}>Delete</button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}