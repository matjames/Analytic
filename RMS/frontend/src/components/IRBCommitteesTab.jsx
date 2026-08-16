import React, { useState, useEffect } from 'react';

const STATUS_COLORS = {
  'Active': { bg: '#dcfce7', color: '#16a34a' },
  'Pending': { bg: '#fef9c3', color: '#ca8a04' },
  'Suspended': { bg: '#fee2e2', color: '#dc2626' },
};

export default function IRBCommitteesTab({ apiBase }) {
  const [committees, setCommittees] = useState([]);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ name: '', code: '', institution: '', chairPerson: '', email: '', phone: '', status: 'Active', approvalValidity: 12 });
  const [saving, setSaving] = useState(false);

  const load = () => {
    fetch(`${apiBase}/api/ethics/committees`)
      .then(r => r.json())
      .then(d => setCommittees(Array.isArray(d) ? d : []))
      .catch(() => setCommittees([]));
  };

  useEffect(() => { load(); }, []);

  const handleSubmit = (e) => {
    e.preventDefault();
    setSaving(true);
    fetch(`${apiBase}/api/ethics/committees`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...form, approvalValidity: Number(form.approvalValidity) }),
    })
      .then(() => { setShowForm(false); setForm({ name: '', code: '', institution: '', chairPerson: '', email: '', phone: '', status: 'Active', approvalValidity: 12 }); load(); })
      .finally(() => setSaving(false));
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
        <div>
          <h2 style={{ margin: 0, fontSize: '1.4rem', fontWeight: 700 }}>🏛️ IRB / Ethics Committees</h2>
          <p style={{ margin: '0.25rem 0 0', color: '#64748b', fontSize: '0.875rem' }}>Registered Institutional Review Boards and ethical oversight bodies</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowForm(v => !v)}>
          {showForm ? '✕ Close' : '➕ Register Committee'}
        </button>
      </div>

      {showForm && (
        <div style={{ background: '#f8fafc', padding: '1.5rem', borderRadius: '10px', border: '1px solid #e2e8f0', marginBottom: '1.5rem' }}>
          <h3 style={{ marginTop: 0, marginBottom: '1rem', fontSize: '1rem', fontWeight: 600 }}>New Ethics Committee Registration</h3>
          <form onSubmit={handleSubmit} style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: '0.875rem' }}>
            {[
              { key: 'name', label: 'Committee Name', placeholder: 'e.g. National IRB Tanzania' },
              { key: 'code', label: 'Committee Code', placeholder: 'e.g. NIMR-IRB-001' },
              { key: 'institution', label: 'Host Institution', placeholder: 'NIMR / University of Dar es Salaam' },
              { key: 'chairPerson', label: 'Chairperson', placeholder: 'Dr. Firstname Lastname' },
              { key: 'email', label: 'Contact Email', placeholder: 'irb@institution.ac.tz' },
            ].map(f => (
              <div key={f.key}>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>{f.label}</label>
                <input type="text" placeholder={f.placeholder} value={form[f.key]} onChange={e => setForm(p => ({ ...p, [f.key]: e.target.value }))}
                  style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }} />
              </div>
            ))}
            <div>
              <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Status</label>
              <select value={form.status} onChange={e => setForm(p => ({ ...p, status: e.target.value }))}
                style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }}>
                <option>Active</option><option>Pending</option><option>Suspended</option>
              </select>
            </div>
            <div style={{ gridColumn: 'span 3', display: 'flex', justifyContent: 'flex-end', gap: '0.5rem', paddingTop: '0.5rem', borderTop: '1px solid #e2e8f0' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Saving…' : 'Save Committee'}</button>
            </div>
          </form>
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(310px,1fr))', gap: '1.25rem' }}>
        {committees.length > 0 ? committees.map(c => (
          <div key={c.id} style={{ background: '#fff', padding: '1.25rem', borderRadius: '10px', border: '1px solid #e2e8f0', boxShadow: '0 1px 4px rgba(0,0,0,0.06)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.75rem' }}>
              <div>
                <h3 style={{ margin: 0, fontSize: '1.05rem', fontWeight: 700, color: '#0f172a' }}>{c.name}</h3>
                <span style={{ fontSize: '0.78rem', color: '#64748b' }}>Code: <strong>{c.code}</strong></span>
              </div>
              <span style={{ fontSize: '0.72rem', padding: '0.2rem 0.5rem', borderRadius: '4px', fontWeight: 600, background: STATUS_COLORS[c.status]?.bg || '#f1f5f9', color: STATUS_COLORS[c.status]?.color || '#475569' }}>
                {c.status}
              </span>
            </div>
            <div style={{ fontSize: '0.85rem', display: 'flex', flexDirection: 'column', gap: '0.35rem', color: '#334155' }}>
              <span>🏫 {c.institution}</span>
              <span>👤 {c.chairPerson || 'Chair not assigned'}</span>
              <span>📧 <a href={`mailto:${c.email}`} style={{ color: '#2563eb' }}>{c.email || 'N/A'}</a></span>
            </div>
          </div>
        )) : (
          <div style={{ gridColumn: 'span 3', padding: '3rem', textAlign: 'center', background: '#f8fafc', borderRadius: '10px', border: '1px dashed #cbd5e1', color: '#94a3b8' }}>
            No ethics committees registered yet.
          </div>
        )}
      </div>
    </div>
  );
}
