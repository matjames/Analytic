import React, { useState } from 'react';

export default function MeetingsTab({ meetings, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({
    title: '', dateTime: '', location: '', agenda: '', attendees: []
  });

  const handleSubmit = async (e) => {
    e.preventDefault();
    await fetch(`${API_URL}/api/meetings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...formData, researchId })
    });
    setShowForm(false);
    setFormData({ title: '', dateTime: '', location: '', agenda: '', attendees: [] });
    onRefresh();
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px', alignItems: 'center' }}>
        <h3>Meetings & Minutes</h3>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>+ Schedule Meeting</button>
      </div>

      {showForm && (
        <div className="glass-panel" style={{ padding: '24px', marginBottom: '24px' }}>
          <h4 style={{ marginBottom: '16px' }}>New Meeting</h4>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
              placeholder="Meeting Title" required value={formData.title}
              onChange={e => setFormData({ ...formData, title: e.target.value })} />
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                type="datetime-local" value={formData.dateTime}
                onChange={e => setFormData({ ...formData, dateTime: e.target.value })} />
              <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                placeholder="Location" value={formData.location}
                onChange={e => setFormData({ ...formData, location: e.target.value })} />
            </div>
            <textarea style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
              placeholder="Agenda" value={formData.agenda}
              onChange={e => setFormData({ ...formData, agenda: e.target.value })} />
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Schedule</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        {meetings.length === 0 ? (
          <p style={{ color: 'var(--text-secondary)' }}>No meetings scheduled.</p>
        ) : (
          meetings.map(m => (
            <div key={m.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: '16px', marginBottom: '16px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ flex: 1 }}>
                  <h4 style={{ margin: '0 0 4px 0' }}>{m.title}</h4>
                  <p style={{ fontSize: '14px', margin: '4px 0', color: 'var(--text-secondary)' }}>
                    {m.dateTime && <span>📅 {m.dateTime}</span>}
                    {m.location && <span> | 📍 {m.location}</span>}
                  </p>
                  {m.agenda && <p style={{ fontSize: '13px', margin: '8px 0', color: 'var(--text-secondary)' }}>{m.agenda}</p>}
                  <span style={{ fontSize: '12px', padding: '2px 8px', background: '#e8f0fe', borderRadius: '4px' }}>{m.status}</span>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}