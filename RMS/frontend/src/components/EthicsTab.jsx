import React, { useState } from 'react';

export default function EthicsTab({ ethics, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [formData, setFormData] = useState({
    irbName: '', submissionDate: '', certificateNumber: '', expiryDate: '', comments: ''
  });

  const resetForm = () => {
    setFormData({ irbName: '', submissionDate: '', certificateNumber: '', expiryDate: '', comments: '' });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const url = editingId
      ? `${import.meta.env.VITE_RMS_API_URL || ''}/api/ethics/${editingId}`
      : `${import.meta.env.VITE_RMS_API_URL || ''}/api/ethics`;
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

  const handleEdit = (item) => {
    setFormData({
      irbName: item.irbName || '', submissionDate: item.submissionDate || '',
      certificateNumber: item.certificateNumber || '', expiryDate: item.expiryDate || '',
      comments: item.comments || ''
    });
    setEditingId(item.id);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this ethics application?')) return;
    await fetch(`${import.meta.env.VITE_RMS_API_URL || ''}/api/ethics/${id}`, { method: 'DELETE' });
    onRefresh();
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'Approved': return 'var(--success)';
      case 'Pending': return 'var(--warning)';
      case 'Rejected': return 'var(--danger)';
      default: return 'var(--text-secondary)';
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Ethics & Institutional Review Board (IRB)</h3>
        {!showForm && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ New Application</button>
        )}
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>{editingId ? 'Edit Application' : 'New Ethics Application'}</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="IRB Name" required value={formData.irbName}
                onChange={e => setFormData({ ...formData, irbName: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="date" placeholder="Submission Date" value={formData.submissionDate}
                onChange={e => setFormData({ ...formData, submissionDate: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Certificate Number" value={formData.certificateNumber}
                onChange={e => setFormData({ ...formData, certificateNumber: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="date" placeholder="Expiry Date" value={formData.expiryDate}
                onChange={e => setFormData({ ...formData, expiryDate: e.target.value })} />
            </div>
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Comments / Notes" value={formData.comments}
              onChange={e => setFormData({ ...formData, comments: e.target.value })} />
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={resetForm}>Cancel</button>
              <button type="submit" className="btn btn-primary">{editingId ? 'Update' : 'Submit'} Application</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {ethics.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No ethics applications registered.</p>
        ) : (
          ethics.map(e => (
            <div key={e.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '8px' }}>
                    <h4 style={{ margin: 0 }}>IRB: {e.irbName}</h4>
                    <span style={{ fontSize: '12px', padding: '2px 8px', borderRadius: '4px', background: getStatusColor(e.status), color: '#fff' }}>
                      {e.status}
                    </span>
                  </div>
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '12px', fontSize: '13px', color: 'var(--text-secondary)' }}>
                    <div><strong>Submitted:</strong> {e.submissionDate || '—'}</div>
                    <div><strong>Certificate:</strong> {e.certificateNumber || '—'}</div>
                    <div><strong>Expires:</strong> {e.expiryDate || '—'}</div>
                  </div>
                  {e.comments && <p style={{ marginTop: '8px', fontSize: '13px', color: 'var(--text-secondary)' }}>{e.comments}</p>}
                </div>
                <div style={{ display: 'flex', gap: '8px', marginLeft: '16px' }}>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px' }} onClick={() => handleEdit(e)}>Edit</button>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px', color: 'var(--danger)' }} onClick={() => handleDelete(e.id)}>Delete</button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}