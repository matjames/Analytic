import React, { useState, useEffect } from 'react';

const ACCESS_COLORS = { Open: { bg: '#dcfce7', color: '#16a34a' }, Restricted: { bg: '#fee2e2', color: '#dc2626' }, Embargoed: { bg: '#fef9c3', color: '#ca8a04' } };

export default function OpenScienceTab({ apiBase }) {
  const [items, setItems] = useState([]);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ title: '', description: '', resourceType: 'Dataset', url: '', license: 'CC BY 4.0', accessLevel: 'Open', keywords: '', repoName: '' });

  const load = () => {
    fetch(`${apiBase}/api/open-science/repo`)
      .then(r => r.json())
      .then(d => setItems(Array.isArray(d) ? d : []))
      .catch(() => setItems([]));
  };

  useEffect(() => { load(); }, []);

  const handleSubmit = (e) => {
    e.preventDefault();
    fetch(`${apiBase}/api/open-science/repo`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form),
    }).then(() => { setShowForm(false); setForm({ title: '', description: '', resourceType: 'Dataset', url: '', license: 'CC BY 4.0', accessLevel: 'Open', keywords: '', repoName: '' }); load(); });
  };

  const ICONS = { Dataset: '🗂️', Code: '💻', Protocol: '📋', Preprint: '📄', Report: '📊', Other: '📦' };

  return (
    <div>
      {/* Hero Banner */}
      <div style={{ background: 'linear-gradient(135deg, #064e3b 0%, #059669 100%)', borderRadius: '12px', padding: '1.5rem 2rem', color: '#fff', marginBottom: '1.75rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ margin: '0 0 0.35rem', fontSize: '1.4rem', fontWeight: 700 }}>🌍 Open Science Repository</h2>
          <p style={{ margin: 0, opacity: 0.85, fontSize: '0.9rem' }}>FAIR-aligned open access datasets, preprints, protocols, and code artifacts</p>
          <div style={{ display: 'flex', gap: '1rem', marginTop: '0.75rem', fontSize: '0.875rem' }}>
            {[['Open', '#dcfce7', '#16a34a'], ['Embargoed', '#fef9c3', '#ca8a04'], ['Restricted', '#fee2e2', '#dc2626']].map(([lbl, bg, col]) => (
              <span key={lbl} style={{ padding: '0.2rem 0.6rem', borderRadius: '4px', background, color: col, fontWeight: 600, fontSize: '0.75rem' }}>
                {lbl}: {items.filter(i => i.accessLevel === lbl).length}
              </span>
            ))}
          </div>
        </div>
        <button className="btn" style={{ background: '#fff', color: '#064e3b', fontWeight: 700, padding: '0.6rem 1.25rem' }} onClick={() => setShowForm(v => !v)}>
          {showForm ? '✕ Close' : '➕ Add Resource'}
        </button>
      </div>

      {showForm && (
        <div style={{ background: '#f8fafc', padding: '1.5rem', borderRadius: '10px', border: '1px solid #e2e8f0', marginBottom: '1.5rem' }}>
          <h3 style={{ margin: '0 0 1rem', fontSize: '1rem', fontWeight: 600 }}>Register Open Science Resource</h3>
          <form onSubmit={handleSubmit} style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: '0.875rem' }}>
            {[
              { key: 'title', label: 'Resource Title', span: 2, placeholder: 'National Household Survey Dataset 2023' },
              { key: 'repoName', label: 'Repository', placeholder: 'Zenodo / OSF / GitHub' },
              { key: 'url', label: 'URL / DOI Link', span: 2, placeholder: 'https://zenodo.org/record/...' },
              { key: 'license', label: 'License', placeholder: 'CC BY 4.0' },
              { key: 'keywords', label: 'Keywords (comma-separated)', span: 2, placeholder: 'survey, Tanzania, health' },
            ].map(f => (
              <div key={f.key} style={{ gridColumn: f.span ? `span ${f.span}` : undefined }}>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, display: 'block', marginBottom: '0.25rem', color: '#374151' }}>{f.label}</label>
                <input type="text" placeholder={f.placeholder} value={form[f.key]} onChange={e => setForm(p => ({ ...p, [f.key]: e.target.value }))}
                  style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }} />
              </div>
            ))}
            <div>
              <label style={{ fontSize: '0.78rem', fontWeight: 600, display: 'block', marginBottom: '0.25rem', color: '#374151' }}>Resource Type</label>
              <select value={form.resourceType} onChange={e => setForm(p => ({ ...p, resourceType: e.target.value }))}
                style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }}>
                {['Dataset','Code','Protocol','Preprint','Report','Other'].map(t => <option key={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label style={{ fontSize: '0.78rem', fontWeight: 600, display: 'block', marginBottom: '0.25rem', color: '#374151' }}>Access Level</label>
              <select value={form.accessLevel} onChange={e => setForm(p => ({ ...p, accessLevel: e.target.value }))}
                style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }}>
                {['Open','Embargoed','Restricted'].map(a => <option key={a}>{a}</option>)}
              </select>
            </div>
            <div style={{ gridColumn: 'span 3' }}>
              <label style={{ fontSize: '0.78rem', fontWeight: 600, display: 'block', marginBottom: '0.25rem', color: '#374151' }}>Description</label>
              <textarea rows={2} value={form.description} onChange={e => setForm(p => ({ ...p, description: e.target.value }))} placeholder="Brief abstract or description of this resource"
                style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem', resize: 'vertical' }} />
            </div>
            <div style={{ gridColumn: 'span 3', display: 'flex', justifyContent: 'flex-end', gap: '0.5rem', borderTop: '1px solid #e2e8f0', paddingTop: '0.75rem' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Save Resource</button>
            </div>
          </form>
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px,1fr))', gap: '1.25rem' }}>
        {items.length > 0 ? items.map(item => (
          <div key={item.id} style={{ background: '#fff', padding: '1.25rem', borderRadius: '10px', border: '1px solid #e2e8f0', boxShadow: '0 1px 4px rgba(0,0,0,0.05)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.75rem' }}>
              <span style={{ fontSize: '1.6rem' }}>{ICONS[item.resourceType] || '📦'}</span>
              <span style={{ fontSize: '0.72rem', padding: '0.2rem 0.5rem', borderRadius: '4px', fontWeight: 600, background: ACCESS_COLORS[item.accessLevel]?.bg || '#f1f5f9', color: ACCESS_COLORS[item.accessLevel]?.color || '#475569' }}>
                {item.accessLevel}
              </span>
            </div>
            <h3 style={{ margin: '0 0 0.35rem', fontSize: '1rem', fontWeight: 700, color: '#0f172a' }}>{item.title}</h3>
            <p style={{ margin: '0 0 0.75rem', fontSize: '0.8rem', color: '#64748b', lineHeight: 1.5 }}>{item.description}</p>
            <div style={{ fontSize: '0.78rem', color: '#94a3b8', display: 'flex', justifyContent: 'space-between' }}>
              <span>📦 {item.resourceType} · {item.repoName || 'Repository'}</span>
              <span>🔓 {item.license}</span>
            </div>
            {item.url && (
              <a href={item.url} target="_blank" rel="noopener noreferrer"
                style={{ display: 'block', marginTop: '0.75rem', fontSize: '0.8rem', color: '#2563eb', textDecoration: 'none', borderTop: '1px solid #f1f5f9', paddingTop: '0.6rem' }}>
                View Resource →
              </a>
            )}
          </div>
        )) : (
          <div style={{ gridColumn: 'span 3', padding: '3rem', textAlign: 'center', background: '#f8fafc', borderRadius: '10px', border: '1px dashed #cbd5e1', color: '#94a3b8' }}>
            No open science resources added yet. Share datasets, code, and preprints to support reproducibility.
          </div>
        )}
      </div>
    </div>
  );
}
