import React, { useState } from 'react';

export default function TasksTab({ tasks, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState(null);
  const [formData, setFormData] = useState({
    title: '', description: '', status: 'Todo', priority: 'Medium',
    startDate: '', endDate: '', assignedTo: '', progress: 0
  });

  const resetForm = () => {
    setFormData({ title: '', description: '', status: 'Todo', priority: 'Medium',
      startDate: '', endDate: '', assignedTo: '', progress: 0 });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    const url = editingId
      ? `${import.meta.env.VITE_RMS_API_URL || ''}/api/tasks/${editingId}`
      : `${import.meta.env.VITE_RMS_API_URL || ''}/api/tasks`;
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

  const handleEdit = (t) => {
    setFormData({
      title: t.title || '', description: t.description || '', status: t.status || 'Todo',
      priority: t.priority || 'Medium', startDate: t.startDate || '', endDate: t.endDate || '',
      assignedTo: t.assignedTo || '', progress: t.progress || 0
    });
    setEditingId(t.id);
    setShowForm(true);
  };

  const handleDelete = async (id) => {
    if (!confirm('Delete this task?')) return;
    await fetch(`${import.meta.env.VITE_RMS_API_URL || ''}/api/tasks/${id}`, { method: 'DELETE' });
    onRefresh();
  };

  const getPriorityColor = (priority) => {
    switch (priority) {
      case 'High': return 'var(--danger)';
      case 'Medium': return 'var(--warning)';
      case 'Low': return 'var(--success)';
      default: return 'var(--text-secondary)';
    }
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'Done': return 'var(--success)';
      case 'In Progress': return 'var(--primary)';
      case 'Todo': return 'var(--text-secondary)';
      default: return 'var(--text-secondary)';
    }
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Study Tasks</h3>
        {!showForm && (
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ New Task</button>
        )}
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>{editingId ? 'Edit Task' : 'New Task'}</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Task Title" required value={formData.title}
                onChange={e => setFormData({ ...formData, title: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Assigned To" value={formData.assignedTo}
                onChange={e => setFormData({ ...formData, assignedTo: e.target.value })} />
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.status}
                onChange={e => setFormData({ ...formData, status: e.target.value })}>
                <option value="Todo">Todo</option>
                <option value="In Progress">In Progress</option>
                <option value="Done">Done</option>
              </select>
              <select style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                value={formData.priority}
                onChange={e => setFormData({ ...formData, priority: e.target.value })}>
                <option value="Low">Low</option>
                <option value="Medium">Medium</option>
                <option value="High">High</option>
              </select>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="date" value={formData.startDate}
                onChange={e => setFormData({ ...formData, startDate: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="date" value={formData.endDate}
                onChange={e => setFormData({ ...formData, endDate: e.target.value })} />
              <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                <label style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Progress: {formData.progress}%</label>
                <input type="range" min="0" max="100" value={formData.progress}
                  onChange={e => setFormData({ ...formData, progress: parseInt(e.target.value) })} />
              </div>
            </div>
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Description" value={formData.description}
              onChange={e => setFormData({ ...formData, description: e.target.value })} />
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={resetForm}>Cancel</button>
              <button type="submit" className="btn btn-primary">{editingId ? 'Update' : 'Create'} Task</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {tasks.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No tasks assigned.</p>
        ) : (
          tasks.map(t => (
            <div key={t.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '6px' }}>
                    <h4 style={{ margin: 0 }}>{t.title}</h4>
                    <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '4px', background: getStatusColor(t.status), color: '#fff' }}>
                      {t.status}
                    </span>
                    <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '4px', background: getPriorityColor(t.priority), color: '#fff' }}>
                      {t.priority}
                    </span>
                  </div>
                  {t.description && <p style={{ fontSize: '14px', margin: '4px 0', color: 'var(--text-secondary)' }}>{t.description}</p>}
                  <div style={{ display: 'flex', gap: '16px', marginTop: '8px', flexWrap: 'wrap', fontSize: '13px', color: 'var(--text-secondary)' }}>
                    {t.assignedTo && <span>Assigned: {t.assignedTo}</span>}
                    {t.startDate && <span>Start: {t.startDate}</span>}
                    {t.endDate && <span>Due: {t.endDate}</span>}
                    <span>Progress: {t.progress}%</span>
                  </div>
                  {t.progress > 0 && (
                    <div style={{ marginTop: '8px', width: '100%', height: '4px', background: 'var(--border-light)', borderRadius: '2px', overflow: 'hidden' }}>
                      <div style={{ width: `${t.progress}%`, height: '100%', background: 'var(--primary)', borderRadius: '2px' }} />
                    </div>
                  )}
                </div>
                <div style={{ display: 'flex', gap: '8px', marginLeft: '16px' }}>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px' }} onClick={() => handleEdit(t)}>Edit</button>
                  <button className="btn btn-secondary" style={{ padding: '4px 12px', fontSize: '12px', color: 'var(--danger)' }} onClick={() => handleDelete(t.id)}>Delete</button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}