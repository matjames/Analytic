import React, { useState } from 'react';

export default function SurveysTab({ surveys, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({ name: '', status: 'Template', targetSample: 1000 });

  const handleSubmit = async (e) => {
    e.preventDefault();
    await fetch(`${API_URL}/api/surveys`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...formData, researchId })
    });
    setShowForm(false);
    setFormData({ name: '', status: 'Template', targetSample: 1000 });
    onRefresh();
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Survey Instruments (StatCollect)</h3>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ New Survey</button>
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>New Survey Instrument</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Survey Name" required value={formData.name}
                onChange={e => setFormData({ ...formData, name: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="number" placeholder="Target Sample" value={formData.targetSample}
                onChange={e => setFormData({ ...formData, targetSample: parseInt(e.target.value) || 0 })} />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Create Survey</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {surveys.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No survey instruments created.</p>
        ) : (
          surveys.map(s => (
            <div key={s.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <h4 style={{ margin: '0 0 4px 0' }}>{s.name}</h4>
                  <div style={{ display: 'flex', gap: '12px', marginTop: '8px', flexWrap: 'wrap', fontSize: '13px', color: 'var(--text-secondary)' }}>
                    <span>Status: {s.status}</span>
                    <span>Target: {s.targetSample}</span>
                    <span>Submissions: {s.submissions}</span>
                    <span>Progress: {s.progress.toFixed(1)}%</span>
                  </div>
                  <div style={{ marginTop: '8px', width: '100%', height: '4px', background: 'var(--border-light)', borderRadius: '2px', overflow: 'hidden' }}>
                    <div style={{ width: `${s.progress}%`, height: '100%', background: 'var(--primary)', borderRadius: '2px' }} />
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