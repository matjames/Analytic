import React, { useState } from 'react';

export default function PublicationsTab({ publications, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [formData, setFormData] = useState({
    title: '', pubType: 'Manuscript', authors: '', journal: '',
    affiliation: '', status: 'Drafting'
  });

  const resetForm = () => {
    setFormData({ title: '', pubType: 'Manuscript', authors: '', journal: '', affiliation: '', status: 'Drafting' });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const url = editingId
      ? `${import.meta.env.VITE_RMS_API_URL || ''}/api/publications/${editingId}`
      : `${import.meta.env.VITE_RMS_API_URL || ''}/api/publications`;
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

  const handleEdit = (p) => {
    setFormData({
      title: p.title || '', pubType: p.pubType || 'Manuscript', authors: p.authors || '',
      journal: p.journal || '', affiliation: p.affiliation || '', status: p.status || 'Drafting'
    });
    setEditingId(p.id);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this publication?')) return;
    await fetch(`${import.meta.env.VITE_RMS_API_URL || ''}/api/publications/${id}`, { method: 'DELETE' });
    onRefresh();
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'Published': return 'var(--success)';
      case 'Under Review': return 'var(--warning)';
      case 'Accepted': return 'var(--primary)';
      default: return 'var(--text-secondary)';
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Publications & Manuscripts</h3>
        {!showForm && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ New Publication</button>
        )}
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>{editingId ? 'Edit Publication' : 'New Publication'}</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Title" required value={formData.title}
                onChange={e => setFormData({ ...formData, title: e.target.value })} />
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.pubType}
                onChange={e => setFormData({ ...formData, pubType: e.target.value })}>
                <option value="Manuscript">Manuscript</option>
                <option value="Journal Article">Journal Article</option>
                <option value="Conference Abstract">Conference Abstract</option>
                <option value="Poster">Poster</option>
                <option value="Technical Report">Technical Report</option>
                <option value="Policy Brief">Policy Brief</option>
                <option value="Working Paper">Working Paper</option>
                <option value="Book Chapter">Book Chapter</option>
              </select>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Authors" value={formData.authors}
                onChange={e => setFormData({ ...formData, authors: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Journal / Venue" value={formData.journal}
                onChange={e => setFormData({ ...formData, journal: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Affiliation" value={formData.affiliation}
                onChange={e => setFormData({ ...formData, affiliation: e.target.value })} />
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.status}
                onChange={e => setFormData({ ...formData, status: e.target.value })}>
                <option value="Drafting">Drafting</option>
                <option value="Under Review">Under Review</option>
                <option value="Accepted">Accepted</option>
                <option value="Published">Published</option>
              </select>
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={resetForm}>Cancel</button>
              <button type="submit" className="btn btn-primary">{editingId ? 'Update' : 'Create'} Publication</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {publications.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No publications registered.</p>
        ) : (
          publications.map(p => (
            <div key={p.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '6px' }}>
                    <h4 style={{ margin: 0 }}>{p.title}</h4>
                    <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '4px', background: getStatusColor(p.status), color: '#fff' }}>
                      {p.status}
                    </span>
                  </div>
                  <p style={{ fontSize: '14px', margin: '4px 0', color: 'var(--text-secondary)' }}>
                    {p.authors && <span>By: {p.authors}</span>}
                    {p.journal && <span> | {p.journal}</span>}
                  </p>
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px', fontSize: '12px' }}>
                    <span style={{ padding: '2px 8px', background: '#f0f0f0', borderRadius: '4px' }}>{p.pubType}</span>
                    {p.doi && <span style={{ padding: '2px 8px', background: '#e8f0fe', borderRadius: '4px', color: 'var(--primary)' }}>DOI: {p.doi}</span>}
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