import React, { useState } from 'react';

export default function DocumentsTab({ documents, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({ name: '', type: '', size: '', uploadedBy: '', url: '', status: 'Draft' });

  const handleSubmit = async (e) => {
    e.preventDefault();
    await fetch(`${API_URL}/api/documents`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...formData, researchId })
    });
    setShowForm(false);
    setFormData({ name: '', type: '', size: '', uploadedBy: '', url: '', status: 'Draft' });
    onRefresh();
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Documents & Attachments</h3>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Upload Document</button>
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>Upload Document</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Document Name" required value={formData.name}
                onChange={e => setFormData({ ...formData, name: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Type (e.g. PDF, DOCX)" value={formData.type}
                onChange={e => setFormData({ ...formData, type: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Size (e.g. 2.4 MB)" value={formData.size}
                onChange={e => setFormData({ ...formData, size: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Uploaded By" value={formData.uploadedBy}
                onChange={e => setFormData({ ...formData, uploadedBy: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="URL / Path" value={formData.url}
                onChange={e => setFormData({ ...formData, url: e.target.value })} />
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.status} onChange={e => setFormData({ ...formData, status: e.target.value })}>
                <option>Draft</option><option>Active</option><option>Approved</option><option>Archived</option>
              </select>
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Upload</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {documents.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No documents uploaded.</p>
        ) : (
          documents.map(d => (
            <div key={d.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <h4 style={{ margin: '0 0 4px 0' }}>{d.name}</h4>
                  <div style={{ display: 'flex', gap: '12px', marginTop: '8px', flexWrap: 'wrap', fontSize: '13px', color: 'var(--text-secondary)' }}>
                    <span>Type: {d.type || '—'}</span>
                    <span>Size: {d.size || '—'}</span>
                    <span>By: {d.uploadedBy || '—'}</span>
                    <span style={{ padding: '2px 8px', background: '#e8f0fe', borderRadius: '4px' }}>{d.status}</span>
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