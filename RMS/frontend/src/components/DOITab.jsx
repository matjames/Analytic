import React, { useState, useEffect } from 'react';

const CITATION_STYLES = ['APA', 'Chicago', 'Harvard', 'Vancouver', 'BibTeX'];

export default function DOITab({ apiBase }) {
  const [dois, setDois] = useState([]);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({ title: '', authors: '', journal: '', year: new Date().getFullYear(), doi: '', url: '', abstract: '', keywords: '', outputType: 'Journal Article' });
  const [citInput, setCitInput] = useState('');
  const [citStyle, setCitStyle] = useState('APA');
  const [citation, setCitation] = useState('');
  const [citLoading, setCitLoading] = useState(false);

  const load = () => {
    fetch(`${apiBase}/api/dois`)
      .then(r => r.json())
      .then(d => setDois(Array.isArray(d) ? d : []))
      .catch(() => setDois([]));
  };

  useEffect(() => { load(); }, []);

  const handleSubmit = (e) => {
    e.preventDefault();
    fetch(`${apiBase}/api/dois`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...form, year: Number(form.year) }),
    }).then(() => { setShowForm(false); load(); });
  };

  const handleFormat = () => {
    if (!citInput.trim()) return;
    setCitLoading(true);
    fetch(`${apiBase}/api/citations/format`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ doi: citInput, style: citStyle }),
    })
      .then(r => r.json())
      .then(d => setCitation(d.citation || d.error || 'No result'))
      .catch(() => setCitation('Error contacting citation service.'))
      .finally(() => setCitLoading(false));
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
      {/* Citation Formatter */}
      <div style={{ background: 'linear-gradient(135deg, #1e3a5f 0%, #2563eb 100%)', borderRadius: '12px', padding: '1.5rem', color: '#fff' }}>
        <h3 style={{ margin: '0 0 1rem', fontSize: '1.1rem', fontWeight: 700 }}>📚 Citation Formatter</h3>
        <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <div style={{ flex: 1, minWidth: '220px' }}>
            <label style={{ fontSize: '0.78rem', fontWeight: 600, opacity: 0.85, display: 'block', marginBottom: '0.25rem' }}>DOI or Title</label>
            <input type="text" placeholder="e.g. 10.1000/xyz123" value={citInput} onChange={e => setCitInput(e.target.value)}
              style={{ width: '100%', padding: '0.5rem 0.75rem', borderRadius: '6px', border: 'none', fontSize: '0.9rem', background: 'rgba(255,255,255,0.15)', color: '#fff', outline: 'none' }} />
          </div>
          <div style={{ minWidth: '140px' }}>
            <label style={{ fontSize: '0.78rem', fontWeight: 600, opacity: 0.85, display: 'block', marginBottom: '0.25rem' }}>Citation Style</label>
            <select value={citStyle} onChange={e => setCitStyle(e.target.value)}
              style={{ width: '100%', padding: '0.5rem 0.75rem', borderRadius: '6px', border: 'none', background: 'rgba(255,255,255,0.15)', color: '#fff', fontSize: '0.9rem' }}>
              {CITATION_STYLES.map(s => <option key={s} style={{ color: '#0f172a' }}>{s}</option>)}
            </select>
          </div>
          <button onClick={handleFormat} disabled={citLoading}
            style={{ padding: '0.55rem 1.25rem', borderRadius: '6px', border: 'none', background: '#fff', color: '#1e40af', fontWeight: 700, cursor: 'pointer', fontSize: '0.9rem' }}>
            {citLoading ? 'Formatting…' : 'Format Citation'}
          </button>
        </div>
        {citation && (
          <div style={{ marginTop: '1rem', background: 'rgba(255,255,255,0.1)', borderRadius: '8px', padding: '1rem', fontSize: '0.875rem', lineHeight: 1.7 }}>
            <strong style={{ fontSize: '0.75rem', textTransform: 'uppercase', opacity: 0.8 }}>Formatted {citStyle} Citation</strong>
            <pre style={{ margin: '0.5rem 0 0', fontFamily: 'monospace', whiteSpace: 'pre-wrap', fontSize: '0.85rem' }}>{citation}</pre>
          </div>
        )}
      </div>

      {/* DOI Registry */}
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <div>
            <h2 style={{ margin: 0, fontSize: '1.3rem', fontWeight: 700 }}>📋 DOI Registry</h2>
            <p style={{ margin: '0.2rem 0 0', fontSize: '0.875rem', color: '#64748b' }}>Track and manage all research outputs with DOI identifiers</p>
          </div>
          <button className="btn btn-primary" onClick={() => setShowForm(v => !v)}>
            {showForm ? '✕ Close' : '➕ Register Output'}
          </button>
        </div>

        {showForm && (
          <div style={{ background: '#f8fafc', padding: '1.5rem', borderRadius: '10px', border: '1px solid #e2e8f0', marginBottom: '1.25rem' }}>
            <form onSubmit={handleSubmit} style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: '0.875rem' }}>
              {[
                { key: 'title', label: 'Title', placeholder: 'Full research output title', span: 2 },
                { key: 'authors', label: 'Authors (comma-separated)', placeholder: 'Doe, J., Smith, A.' },
                { key: 'journal', label: 'Journal / Publisher', placeholder: 'Journal of Statistics' },
                { key: 'year', label: 'Year', placeholder: '2024', type: 'number' },
                { key: 'doi', label: 'DOI', placeholder: '10.XXXX/...' },
                { key: 'url', label: 'URL', placeholder: 'https://doi.org/...' },
              ].map(f => (
                <div key={f.key} style={{ gridColumn: f.span ? `span ${f.span}` : undefined }}>
                  <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>{f.label}</label>
                  <input type={f.type || 'text'} placeholder={f.placeholder} value={form[f.key]} onChange={e => setForm(p => ({ ...p, [f.key]: e.target.value }))}
                    style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }} />
                </div>
              ))}
              <div>
                <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Output Type</label>
                <select value={form.outputType} onChange={e => setForm(p => ({ ...p, outputType: e.target.value }))}
                  style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }}>
                  {['Journal Article','Conference Paper','Book Chapter','Technical Report','Dataset','Thesis'].map(o => <option key={o}>{o}</option>)}
                </select>
              </div>
              <div style={{ gridColumn: 'span 3', display: 'flex', justifyContent: 'flex-end', gap: '0.5rem', borderTop: '1px solid #e2e8f0', paddingTop: '0.75rem' }}>
                <button type="button" className="btn btn-secondary" onClick={() => setShowForm(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Register DOI</button>
              </div>
            </form>
          </div>
        )}

        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.875rem' }}>
          {dois.length > 0 ? dois.map(d => (
            <div key={d.id} style={{ background: '#fff', padding: '1.25rem', borderRadius: '10px', border: '1px solid #e2e8f0', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div>
                  <h3 style={{ margin: '0 0 0.25rem', fontSize: '1rem', fontWeight: 700, color: '#0f172a' }}>{d.title}</h3>
                  <p style={{ margin: 0, fontSize: '0.8rem', color: '#64748b' }}>{d.authors} — <em>{d.journal}</em>, {d.year}</p>
                </div>
                <span style={{ fontSize: '0.72rem', padding: '0.2rem 0.5rem', background: '#eff6ff', color: '#2563eb', borderRadius: '4px', fontWeight: 600, whiteSpace: 'nowrap', marginLeft: '0.75rem' }}>
                  {d.outputType}
                </span>
              </div>
              {d.doi && (
                <div style={{ marginTop: '0.75rem', padding: '0.5rem 0.75rem', background: '#f8fafc', borderRadius: '6px', fontSize: '0.8rem', fontFamily: 'monospace' }}>
                  <span style={{ color: '#94a3b8' }}>DOI: </span>
                  <a href={`https://doi.org/${d.doi}`} target="_blank" rel="noopener noreferrer" style={{ color: '#2563eb', textDecoration: 'none' }}>{d.doi}</a>
                </div>
              )}
            </div>
          )) : (
            <div style={{ padding: '3rem', textAlign: 'center', background: '#f8fafc', borderRadius: '10px', border: '1px dashed #cbd5e1', color: '#94a3b8' }}>
              No research outputs registered. Add your first DOI to start tracking publications.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
