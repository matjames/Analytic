import React, { useState } from 'react';

export default function LiteratureTab({ literature, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [formData, setFormData] = useState({
    title: '', authors: '', journal: '', doi: '', pubYear: '',
    citation: '', keywords: '', category: '', tags: [], notes: '', url: '', readingStatus: 'Unread'
  });

  const resetForm = () => {
    setFormData({ title: '', authors: '', journal: '', doi: '', pubYear: '',
      citation: '', keywords: '', category: '', tags: [], notes: '', url: '', readingStatus: 'Unread' });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const url = editingId
      ? `${import.meta.env.VITE_RMS_API_URL || ''}/api/literature/${editingId}`
      : `${import.meta.env.VITE_RMS_API_URL || ''}/api/literature`;
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
      title: item.title || '', authors: item.authors || '', journal: item.journal || '',
      doi: item.doi || '', pubYear: item.pubYear ? String(item.pubYear) : '',
      citation: item.citation || '', keywords: item.keywords || '', category: item.category || '',
      tags: item.tags || [], notes: item.notes || '', url: item.url || '', readingStatus: item.readingStatus || 'Unread'
    });
    setEditingId(item.id);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this literature item?')) return;
    await fetch(`${import.meta.env.VITE_RMS_API_URL || ''}/api/literature/${id}`, { method: 'DELETE' });
    onRefresh();
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Literature Library</h3>
        {!showForm && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Add Literature</button>
        )}
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>{editingId ? 'Edit Literature' : 'New Literature Item'}</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
              placeholder="Title" required value={formData.title}
              onChange={e => setFormData({ ...formData, title: e.target.value })} />
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Authors" value={formData.authors}
                onChange={e => setFormData({ ...formData, authors: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Journal / Source" value={formData.journal}
                onChange={e => setFormData({ ...formData, journal: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="DOI" value={formData.doi}
                onChange={e => setFormData({ ...formData, doi: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Publication Year" type="number" value={formData.pubYear}
                onChange={e => setFormData({ ...formData, pubYear: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Category" value={formData.category}
                onChange={e => setFormData({ ...formData, category: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Keywords (comma separated)" value={formData.keywords}
                onChange={e => setFormData({ ...formData, keywords: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="URL" value={formData.url}
                onChange={e => setFormData({ ...formData, url: e.target.value })} />
            </div>
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Citation" value={formData.citation}
              onChange={e => setFormData({ ...formData, citation: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Notes" value={formData.notes}
              onChange={e => setFormData({ ...formData, notes: e.target.value })} />
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={resetForm}>Cancel</button>
              <button type="submit" className="btn btn-primary">{editingId ? 'Update' : 'Add'} Literature</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {literature.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No literature added to this study.</p>
        ) : (
          literature.map(l => (
            <div key={l.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <h4 style={{ margin: '0 0 4px 0' }}>{l.title}</h4>
                  <p style={{ fontSize: '14px', margin: '4px 0', color: 'var(--text-secondary)' }}>
                    Authors: {l.authors || '—'} | Journal: {l.journal || '—'}
                  </p>
                  <div style={{ display: 'flex', gap: '8px', marginTop: '8px', flexWrap: 'wrap', fontSize: '12px' }}>
                    {l.pubYear && <span style={{ padding: '2px 8px', background: '#f0f0f0', borderRadius: '4px' }}>{l.pubYear}</span>}
                    {l.doi && <span style={{ padding: '2px 8px', background: '#e8f0fe', borderRadius: '4px', color: 'var(--primary)' }}>DOI: {l.doi}</span>}
                    {l.category && <span style={{ padding: '2px 8px', background: '#fff3cd', borderRadius: '4px' }}>{l.category}</span>}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '8px', marginLeft: '16px' }}>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px' }} onClick={() => handleEdit(l)}>Edit</button>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px', color: 'var(--danger)' }} onClick={() => handleDelete(l.id)}>Delete</button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}