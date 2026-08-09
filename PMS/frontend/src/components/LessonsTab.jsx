import React, { useState } from 'react';

const API_BASE = import.meta.env.VITE_PMS_API_URL || 'http://localhost:8091';

export default function LessonsTab({ lessons = [], projectId, onRefresh }) {
  const [isAdding, setIsAdding] = useState(false);
  const [title, setTitle] = useState('');
  const [desc, setDesc] = useState('');
  const [category, setCategory] = useState('Operational');
  const [impact, setImpact] = useState('Medium');
  const [owner, setOwner] = useState('');

  const handleAdd = (e) => {
    e.preventDefault();
    fetch(`${API_BASE}/api/lessons`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId, title, description: desc, category, impact, owner })
    })
      .then(res => res.json())
      .then(() => {
        onRefresh();
        setIsAdding(false);
        setTitle(''); setDesc(''); setOwner('');
      })
      .catch(err => console.error("Error creating lesson:", err));
  };

  const getImpactColor = (impact) => {
    if (impact === 'High') return 'var(--accent-danger)';
    if (impact === 'Medium') return 'var(--accent-warning)';
    return 'var(--accent-success)';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>📚 Lessons Learned</h2>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
            Knowledge capture and lessons learned from project execution
          </p>
        </div>
        <button className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 16px' }} onClick={() => setIsAdding(true)}>
          + Add Lesson
        </button>
      </div>

      {isAdding && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleAdd}>
            <h3>Add Lesson Learned</h3>
            <div className="form-group">
              <label>Lesson Title</label>
              <input type="text" value={title} onChange={e => setTitle(e.target.value)} required />
            </div>
            <div className="form-group">
              <label>Description</label>
              <textarea rows={3} value={desc} onChange={e => setDesc(e.target.value)} required />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Category</label>
                <select value={category} onChange={e => setCategory(e.target.value)}>
                  <option value="Operational">Operational</option>
                  <option value="Technical">Technical</option>
                  <option value="Financial">Financial</option>
                  <option value="Community Engagement">Community Engagement</option>
                  <option value="Technology">Technology</option>
                  <option value="Management">Management</option>
                </select>
              </div>
              <div className="form-group">
                <label>Impact</label>
                <select value={impact} onChange={e => setImpact(e.target.value)}>
                  <option value="Low">Low</option>
                  <option value="Medium">Medium</option>
                  <option value="High">High</option>
                </select>
              </div>
            </div>
            <div className="form-group">
              <label>Owner</label>
              <input type="text" value={owner} onChange={e => setOwner(e.target.value)} />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAdding(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Add Lesson</button>
            </div>
          </form>
        </div>
      )}

      {/* Summary stats */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        {[
          { label: 'Total Lessons', value: lessons.length, icon: '📚', color: 'var(--accent-primary)' },
          { label: 'High Impact', value: lessons.filter(l => l.impact === 'High').length, icon: '🔥', color: 'var(--accent-danger)' },
          { label: 'Medium Impact', value: lessons.filter(l => l.impact === 'Medium').length, icon: '⚡', color: 'var(--accent-warning)' },
          { label: 'Low Impact', value: lessons.filter(l => l.impact === 'Low').length, icon: '💡', color: 'var(--accent-success)' },
        ].map(stat => (
          <div key={stat.label} className="glass-panel" style={{ padding: '16px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ fontSize: '24px' }}>{stat.icon}</span>
            <div>
              <div style={{ fontSize: '22px', fontWeight: '800', color: stat.color }}>{stat.value}</div>
              <div style={{ fontSize: '10px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>{stat.label}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Lessons list */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>Captured Lessons</h3>
        {lessons.length === 0 ? (
          <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No lessons learned captured yet.</div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            {lessons.map(lesson => (
              <div key={lesson.id} style={{ padding: '14px', background: 'rgba(99,102,241,0.03)', borderRadius: '8px', border: '1px solid rgba(99,102,241,0.1)' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                  <span style={{ fontSize: '13px', fontWeight: '700' }}>{lesson.title}</span>
                  <div style={{ display: 'flex', gap: '6px' }}>
                    <span className="badge" style={{ fontSize: '9px', background: 'rgba(22,92,146,0.1)', color: 'var(--primary-color)' }}>{lesson.category}</span>
                    <span className="badge" style={{ fontSize: '9px', background: `${getImpactColor(lesson.impact)}20`, color: getImpactColor(lesson.impact) }}>{lesson.impact} Impact</span>
                  </div>
                </div>
                <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{lesson.description}</div>
                <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '6px' }}>
                  Owner: {lesson.owner} | {new Date(lesson.createdTime).toLocaleDateString()}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}