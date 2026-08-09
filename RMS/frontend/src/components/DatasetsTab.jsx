import React, { useState } from 'react';

export default function DatasetsTab({ datasets, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [formData, setFormData] = useState({
    name: '', description: '', version: '1.0', status: 'Draft',
    sourceType: '', collectionMethod: '', metadataInfo: '', variablesDict: '',
    accessLevel: 'Internal', statcollectId: ''
  });

  const resetForm = () => {
    setFormData({ name: '', description: '', version: '1.0', status: 'Draft',
      sourceType: '', collectionMethod: '', metadataInfo: '', variablesDict: '',
      accessLevel: 'Internal', statcollectId: '' });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const url = editingId
      ? `${import.meta.env.VITE_RMS_API_URL || ''}/api/datasets/${editingId}`
      : `${import.meta.env.VITE_RMS_API_URL || ''}/api/datasets`;
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

  const handleEdit = (d) => {
    setFormData({
      name: d.name || '', description: d.description || '', version: d.version || '1.0',
      status: d.status || 'Draft', sourceType: d.sourceType || '', collectionMethod: d.collectionMethod || '',
      metadataInfo: d.metadataInfo || '', variablesDict: d.variablesDict || '',
      accessLevel: d.accessLevel || 'Internal', statcollectId: d.statcollectId || ''
    });
    setEditingId(d.id);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this dataset?')) return;
    await fetch(`${import.meta.env.VITE_RMS_API_URL || ''}/api/datasets/${id}`, { method: 'DELETE' });
    onRefresh();
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'Validated': return 'var(--success)';
      case 'Draft': return 'var(--warning)';
      case 'Archived': return 'var(--text-secondary)';
      default: return 'var(--primary)';
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <div>
          <h3>Dataset Repository</h3>
          <p style={{ margin: '4px 0 0 0', fontSize: '14px', color: 'var(--text-secondary)' }}>
            {datasets.length} registered dataset{datasets.length !== 1 ? 's' : ''}
          </p>
        </div>
        {!showForm && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Register Dataset</button>
        )}
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>{editingId ? 'Edit Dataset' : 'Register New Dataset'}</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Dataset Name" required value={formData.name}
                onChange={e => setFormData({ ...formData, name: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Version" value={formData.version}
                onChange={e => setFormData({ ...formData, version: e.target.value })} />
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.status}
                onChange={e => setFormData({ ...formData, status: e.target.value })}>
                <option value="Draft">Draft</option>
                <option value="Validated">Validated</option>
                <option value="Archived">Archived</option>
              </select>
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.accessLevel}
                onChange={e => setFormData({ ...formData, accessLevel: e.target.value })}>
                <option value="Internal">Internal</option>
                <option value="Restricted">Restricted</option>
                <option value="Public">Public</option>
              </select>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Source Type" value={formData.sourceType}
                onChange={e => setFormData({ ...formData, sourceType: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Collection Method" value={formData.collectionMethod}
                onChange={e => setFormData({ ...formData, collectionMethod: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="StatCollect ID" value={formData.statcollectId}
                onChange={e => setFormData({ ...formData, statcollectId: e.target.value })} />
            </div>
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Description" value={formData.description}
              onChange={e => setFormData({ ...formData, description: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Metadata Information" value={formData.metadataInfo}
              onChange={e => setFormData({ ...formData, metadataInfo: e.target.value })} />
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Variables Dictionary" value={formData.variablesDict}
              onChange={e => setFormData({ ...formData, variablesDict: e.target.value })} />
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={resetForm}>Cancel</button>
              <button type="submit" className="btn btn-primary">{editingId ? 'Update' : 'Register'} Dataset</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {datasets.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No registered datasets.</p>
        ) : (
          datasets.map(d => (
            <div key={d.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '6px' }}>
                    <h4 style={{ margin: 0 }}>{d.name}</h4>
                    <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '4px', background: getStatusColor(d.status), color: '#fff' }}>
                      {d.status}
                    </span>
                  </div>
                  <p style={{ fontSize: '14px', margin: '4px 0', color: 'var(--text-secondary)' }}>{d.description}</p>
                  <div style={{ display: 'flex', gap: '12px', marginTop: '8px', flexWrap: 'wrap', fontSize: '13px', color: 'var(--text-secondary)' }}>
                    <span>Version: {d.version}</span>
                    <span>Source: {d.sourceType || '—'}</span>
                    <span>Method: {d.collectionMethod || '—'}</span>
                    <span>Access: {d.accessLevel}</span>
                    {d.statcollectId && <span style={{ color: 'var(--primary)' }}>StatCollect: {d.statcollectId}</span>}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '8px', marginLeft: '16px' }}>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px' }} onClick={() => handleEdit(d)}>Edit</button>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px', color: 'var(--danger)' }} onClick={() => handleDelete(d.id)}>Delete</button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}