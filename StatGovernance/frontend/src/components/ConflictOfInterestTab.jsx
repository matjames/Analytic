import React, { useState, useEffect } from 'react';

export default function ConflictOfInterestTab({ apiBase }) {
  const [declarations, setDeclarations] = useState([]);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    declarantName: '',
    declarantRole: '',
    conflictType: 'Financial Interest',
    description: '',
    projectName: '',
    hasConflict: true,
    mitigationPlan: '',
  });

  const load = () => {
    fetch(`${apiBase}/api/governance/coi`)
      .then(r => r.json())
      .then(d => setDeclarations(Array.isArray(d) ? d : []))
      .catch(() => setDeclarations([]));
  };

  useEffect(() => { load(); }, []);

  const handleSubmit = (e) => {
    e.preventDefault();
    fetch(`${apiBase}/api/governance/coi`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form),
    }).then(() => {
      setShowForm(false);
      setForm({ declarantName: '', declarantRole: '', conflictType: 'Financial Interest', description: '', projectName: '', hasConflict: true, mitigationPlan: '' });
      load();
    });
  };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
        <div>
          <h2 style={{ margin: 0, fontSize: '1.4rem', fontWeight: 700 }}>⚖️ Conflict of Interest Declarations</h2>
          <p style={{ margin: '0.25rem 0 0', fontSize: '0.875rem', color: '#64748b' }}>
            Staff and board members must declare any potential conflicts of interest annually and per-project
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowForm(v => !v)}>
          {showForm ? '✕ Close' : '➕ New Declaration'}
        </button>
      </div>

      {/* Compliance stats */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: '1rem', marginBottom: '1.5rem' }}>
        {[
          { label: 'Total Declarations', value: declarations.length, icon: '📋', color: '#3b82f6' },
          { label: 'Conflicts Declared', value: declarations.filter(d => d.hasConflict).length, icon: '⚠️', color: '#f59e0b' },
          { label: 'No Conflict', value: declarations.filter(d => !d.hasConflict).length, icon: '✅', color: '#22c55e' },
        ].map(s => (
          <div key={s.label} style={{ background: '#fff', padding: '1.25rem', borderRadius: '10px', border: '1px solid #e2e8f0', display: 'flex', alignItems: 'center', gap: '0.875rem' }}>
            <span style={{ fontSize: '1.75rem' }}>{s.icon}</span>
            <div>
              <div style={{ fontSize: '1.75rem', fontWeight: 800, color: s.color, lineHeight: 1 }}>{s.value}</div>
              <div style={{ fontSize: '0.75rem', color: '#64748b', marginTop: '0.2rem' }}>{s.label}</div>
            </div>
          </div>
        ))}
      </div>

      {showForm && (
        <div style={{ background: '#f8fafc', padding: '1.5rem', borderRadius: '10px', border: '1px solid #e2e8f0', marginBottom: '1.5rem' }}>
          <h3 style={{ margin: '0 0 1rem', fontSize: '1rem', fontWeight: 600 }}>Submit COI Declaration</h3>
          <form onSubmit={handleSubmit} style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: '0.875rem' }}>
            {[
              { key: 'declarantName', label: 'Declarant Full Name', placeholder: 'Dr. Firstname Lastname' },
              { key: 'declarantRole', label: 'Role / Position', placeholder: 'Principal Investigator' },
              { key: 'projectName', label: 'Related Project (if any)', placeholder: 'National Survey 2024' },
            ].map(f => (
              <div key={f.key}>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>{f.label}</label>
                <input type="text" placeholder={f.placeholder} value={form[f.key]} onChange={e => setForm(p => ({ ...p, [f.key]: e.target.value }))}
                  style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }} />
              </div>
            ))}
            <div>
              <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Conflict Type</label>
              <select value={form.conflictType} onChange={e => setForm(p => ({ ...p, conflictType: e.target.value }))}
                style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }}>
                {['Financial Interest','Family Relationship','Prior Employment','Board Membership','Intellectual Property','Other'].map(t => <option key={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Has Conflict?</label>
              <select value={form.hasConflict ? 'yes' : 'no'} onChange={e => setForm(p => ({ ...p, hasConflict: e.target.value === 'yes' }))}
                style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }}>
                <option value="yes">Yes — I declare a conflict</option>
                <option value="no">No — No conflict to declare</option>
              </select>
            </div>
            <div style={{ gridColumn: 'span 3' }}>
              <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Description of Conflict / Statement</label>
              <textarea rows={3} value={form.description} onChange={e => setForm(p => ({ ...p, description: e.target.value }))}
                placeholder="Describe the nature of the conflict or provide a clear statement of no conflict"
                style={{ width: '100%', padding: '0.5rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem', resize: 'vertical' }} />
            </div>
            {form.hasConflict && (
              <div style={{ gridColumn: 'span 3' }}>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Mitigation Plan</label>
                <textarea rows={2} value={form.mitigationPlan} onChange={e => setForm(p => ({ ...p, mitigationPlan: e.target.value }))}
                  placeholder="How will this conflict be managed or mitigated?"
                  style={{ width: '100%', padding: '0.5rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem', resize: 'vertical' }} />
              </div>
            )}
            <div style={{ gridColumn: 'span 3', display: 'flex', justifyContent: 'flex-end', gap: '0.5rem', borderTop: '1px solid #e2e8f0', paddingTop: '0.75rem' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Submit Declaration</button>
            </div>
          </form>
        </div>
      )}

      <div style={{ display: 'flex', flexDirection: 'column', gap: '0.875rem' }}>
        {declarations.length > 0 ? declarations.map(d => (
          <div key={d.id} style={{ background: '#fff', padding: '1.25rem', borderRadius: '10px', border: `1px solid ${d.hasConflict ? '#fed7aa' : '#bbf7d0'}`, boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <h3 style={{ margin: '0 0 0.2rem', fontSize: '1rem', fontWeight: 700 }}>{d.declarantName}</h3>
                <span style={{ fontSize: '0.8rem', color: '#64748b' }}>{d.declarantRole}{d.projectName ? ` · ${d.projectName}` : ''}</span>
              </div>
              <span style={{ padding: '0.25rem 0.6rem', borderRadius: '6px', fontSize: '0.78rem', fontWeight: 700, background: d.hasConflict ? '#fee2e2' : '#dcfce7', color: d.hasConflict ? '#dc2626' : '#16a34a' }}>
                {d.hasConflict ? '⚠️ Conflict Declared' : '✅ No Conflict'}
              </span>
            </div>
            {d.hasConflict && (
              <div style={{ marginTop: '0.75rem', padding: '0.75rem', background: '#fff7ed', borderRadius: '6px', fontSize: '0.85rem', color: '#92400e', borderLeft: '3px solid #f59e0b' }}>
                <strong>Type:</strong> {d.conflictType} — {d.description}
                {d.mitigationPlan && <div style={{ marginTop: '0.35rem' }}><strong>Mitigation:</strong> {d.mitigationPlan}</div>}
              </div>
            )}
          </div>
        )) : (
          <div style={{ padding: '3rem', textAlign: 'center', background: '#f8fafc', borderRadius: '10px', border: '1px dashed #cbd5e1', color: '#94a3b8' }}>
            No conflict of interest declarations on record yet.
          </div>
        )}
      </div>
    </div>
  );
}
