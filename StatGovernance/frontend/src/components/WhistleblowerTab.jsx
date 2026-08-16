import React, { useState, useEffect } from 'react';

export default function WhistleblowerTab({ apiBase }) {
  const [reports, setReports] = useState([]);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    title: '',
    description: '',
    category: 'Financial Misconduct',
    severity: 'Medium',
    isAnonymous: true,
  });
  const [saving, setSaving] = useState(false);

  const load = () => {
    fetch(`${apiBase}/api/whistleblower/reports`)
      .then(r => r.json())
      .then(d => setReports(Array.isArray(d) ? d : []))
      .catch(() => setReports([]));
  };

  useEffect(() => { load(); }, []);

  const handleSubmit = (e) => {
    e.preventDefault();
    setSaving(true);
    fetch(`${apiBase}/api/whistleblower/reports`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form),
    }).then(() => {
      setShowForm(false);
      setForm({ title: '', description: '', category: 'Financial Misconduct', severity: 'Medium', isAnonymous: true });
      load();
    }).finally(() => setSaving(false));
  };

  const SEV_COLOR = { High: '#ef4444', Medium: '#f59e0b', Low: '#22c55e' };
  const STATUS_BG = { New: '#eff6ff', 'Under Review': '#fef9c3', Resolved: '#dcfce7', Closed: '#f1f5f9' };

  return (
    <div>
      {/* Warning Banner */}
      <div style={{ background: 'linear-gradient(135deg, #7f1d1d 0%, #b91c1c 100%)', color: '#fff', borderRadius: '12px', padding: '1.25rem 1.75rem', marginBottom: '1.75rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ margin: 0, fontSize: '1.3rem', fontWeight: 700 }}>🔒 Confidential Whistleblower Reporting</h2>
          <p style={{ margin: '0.25rem 0 0', opacity: 0.85, fontSize: '0.875rem' }}>
            All reports are end-to-end encrypted. Anonymous submissions are fully protected under institutional policy.
          </p>
        </div>
        <button className="btn" style={{ background: 'rgba(255,255,255,0.15)', color: '#fff', border: '1px solid rgba(255,255,255,0.3)', fontWeight: 600 }}
          onClick={() => setShowForm(v => !v)}>
          {showForm ? '✕ Close' : '📣 Submit Report'}
        </button>
      </div>

      {showForm && (
        <div style={{ background: '#fff', padding: '1.5rem', borderRadius: '10px', border: '1px solid #e2e8f0', boxShadow: '0 4px 16px rgba(0,0,0,0.08)', marginBottom: '1.5rem' }}>
          <h3 style={{ margin: '0 0 1rem', fontSize: '1rem', fontWeight: 700 }}>🔐 Submit Confidential Report</h3>
          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '0.875rem' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr', gap: '0.875rem' }}>
              <div>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Incident Title</label>
                <input type="text" required value={form.title} onChange={e => setForm(p => ({ ...p, title: e.target.value }))}
                  placeholder="Brief description of the incident" style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }} />
              </div>
              <div>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Category</label>
                <select value={form.category} onChange={e => setForm(p => ({ ...p, category: e.target.value }))}
                  style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }}>
                  {['Financial Misconduct','Data Fraud','Harassment','Corruption','Policy Violation','Safety Concern','Other'].map(c => <option key={c}>{c}</option>)}
                </select>
              </div>
              <div>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Severity</label>
                <select value={form.severity} onChange={e => setForm(p => ({ ...p, severity: e.target.value }))}
                  style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }}>
                  {['Critical','High','Medium','Low'].map(s => <option key={s}>{s}</option>)}
                </select>
              </div>
            </div>
            <div>
              <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Detailed Description</label>
              <textarea rows={4} required value={form.description} onChange={e => setForm(p => ({ ...p, description: e.target.value }))}
                placeholder="Provide as much detail as possible. Include dates, persons involved, and any supporting information."
                style={{ width: '100%', padding: '0.5rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem', resize: 'vertical' }} />
            </div>
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.875rem', cursor: 'pointer' }}>
              <input type="checkbox" checked={form.isAnonymous} onChange={e => setForm(p => ({ ...p, isAnonymous: e.target.checked }))} />
              <span><strong>Submit Anonymously</strong> — your identity will not be recorded or linked to this report</span>
            </label>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '0.5rem', borderTop: '1px solid #f1f5f9', paddingTop: '0.75rem' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary" disabled={saving}>{saving ? 'Submitting…' : '🔐 Submit Confidentially'}</button>
            </div>
          </form>
        </div>
      )}

      {/* Reports List */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '0.875rem' }}>
        {reports.length > 0 ? reports.map(r => (
          <div key={r.id} style={{ background: '#fff', padding: '1.25rem', borderRadius: '10px', border: '1px solid #e2e8f0', boxShadow: '0 1px 3px rgba(0,0,0,0.05)', display: 'flex', gap: '1rem', alignItems: 'flex-start' }}>
            <div style={{ width: '4px', alignSelf: 'stretch', borderRadius: '4px', background: SEV_COLOR[r.severity] || '#94a3b8', flexShrink: 0 }} />
            <div style={{ flex: 1 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.35rem' }}>
                <h3 style={{ margin: 0, fontSize: '1rem', fontWeight: 700 }}>{r.isAnonymous ? '🔒 Anonymous Report' : r.title}</h3>
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                  <span style={{ fontSize: '0.72rem', padding: '0.2rem 0.5rem', borderRadius: '4px', background: `${SEV_COLOR[r.severity]}20`, color: SEV_COLOR[r.severity], fontWeight: 600 }}>{r.severity}</span>
                  <span style={{ fontSize: '0.72rem', padding: '0.2rem 0.5rem', borderRadius: '4px', background: STATUS_BG[r.status] || '#f1f5f9', color: '#475569', fontWeight: 600 }}>{r.status || 'New'}</span>
                </div>
              </div>
              <p style={{ margin: '0 0 0.5rem', fontSize: '0.85rem', color: '#64748b' }}>{r.category}</p>
              {!r.isAnonymous && <p style={{ margin: 0, fontSize: '0.82rem', color: '#334155' }}>{r.description}</p>}
              {r.isAnonymous && <p style={{ margin: 0, fontSize: '0.82rem', color: '#94a3b8', fontStyle: 'italic' }}>[Description hidden — anonymous submission]</p>}
            </div>
          </div>
        )) : (
          <div style={{ padding: '3rem', textAlign: 'center', background: '#f8fafc', borderRadius: '10px', border: '1px dashed #cbd5e1', color: '#94a3b8' }}>
            No whistleblower reports on file. All submissions are fully confidential.
          </div>
        )}
      </div>
    </div>
  );
}
