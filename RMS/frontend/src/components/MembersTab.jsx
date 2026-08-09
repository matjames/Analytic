import React, { useState } from 'react';

export default function MembersTab({ members, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [formData, setFormData] = useState({
    name: '', role: '', email: '', department: '', phone: '', location: ''
  });

  const resetForm = () => {
    setFormData({ name: '', role: '', email: '', department: '', phone: '', location: '' });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const url = editingId
      ? `${import.meta.env.VITE_RMS_API_URL || ''}/api/members/${editingId}`
      : `${import.meta.env.VITE_RMS_API_URL || ''}/api/members`;
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

  const handleEdit = (m) => {
    setFormData({
      name: m.name || '', role: m.role || '', email: m.email || '',
      department: m.department || '', phone: m.phone || '', location: m.location || ''
    });
    setEditingId(m.id);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!confirm('Remove this team member?')) return;
    await fetch(`${import.meta.env.VITE_RMS_API_URL || ''}/api/members/${id}`, { method: 'DELETE' });
    onRefresh();
  };

  const getRoleColor = (role) => {
    switch (role) {
      case 'Principal Investigator': return 'var(--primary)';
      case 'Co-Investigator': return 'var(--accent)';
      case 'Statistician': return 'var(--success)';
      case 'Data Manager': return 'var(--warning)';
      case 'Research Assistant': return 'var(--text-secondary)';
      default: return 'var(--text-secondary)';
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Research Team Members</h3>
        {!showForm && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Add Member</button>
        )}
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>{editingId ? 'Edit Member' : 'Add Team Member'}</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Full Name" required value={formData.name}
                onChange={e => setFormData({ ...formData, name: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Role (e.g. Co-Investigator)" value={formData.role}
                onChange={e => setFormData({ ...formData, role: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="email" placeholder="Email" value={formData.email}
                onChange={e => setFormData({ ...formData, email: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Department" value={formData.department}
                onChange={e => setFormData({ ...formData, department: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Phone" value={formData.phone}
                onChange={e => setFormData({ ...formData, phone: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Location" value={formData.location}
                onChange={e => setFormData({ ...formData, location: e.target.value })} />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={resetForm}>Cancel</button>
              <button type="submit" className="btn btn-primary">{editingId ? 'Update' : 'Add'} Member</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {members.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No team members added yet.</p>
        ) : (
          members.map(m => (
            <div key={m.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '6px' }}>
                    <h4 style={{ margin: 0 }}>{m.name}</h4>
                    <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '4px', background: getRoleColor(m.role), color: '#fff' }}>
                      {m.role || 'Member'}
                    </span>
                  </div>
                  <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap', fontSize: '13px', color: 'var(--text-secondary)' }}>
                    {m.email && <span>Email: {m.email}</span>}
                    {m.department && <span>Dept: {m.department}</span>}
                    {m.phone && <span>Phone: {m.phone}</span>}
                    {m.location && <span>Location: {m.location}</span>}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '8px', marginLeft: '16px' }}>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px' }} onClick={() => handleEdit(m)}>Edit</button>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px', color: 'var(--danger)' }} onClick={() => handleDelete(m.id)}>Remove</button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}