import React, { useState, useEffect } from 'react';

const TOGGLE_STYLE = (active) => ({
  display: 'inline-flex', alignItems: 'center', gap: '0.35rem',
  padding: '0.3rem 0.75rem', borderRadius: '6px', border: 'none', cursor: 'pointer',
  fontSize: '0.82rem', fontWeight: 600,
  background: active ? '#2563eb' : '#f1f5f9',
  color: active ? '#fff' : '#475569',
  transition: 'all 0.15s ease',
});

export default function AdminConfigTab({ apiBase }) {
  const [featureFlags, setFeatureFlags] = useState([]);
  const [sysParams, setSysParams] = useState([]);
  const [activeSection, setActiveSection] = useState('flags');
  const [showFlagForm, setShowFlagForm] = useState(false);
  const [showParamForm, setShowParamForm] = useState(false);
  const [flagForm, setFlagForm] = useState({ name: '', description: '', environment: 'Production', isEnabled: false });
  const [paramForm, setParamForm] = useState({ key: '', value: '', description: '', category: 'General' });

  const loadFlags = () => fetch(`${apiBase}/api/governance/feature-flags`).then(r => r.json()).then(d => setFeatureFlags(Array.isArray(d) ? d : [])).catch(() => setFeatureFlags([]));
  const loadParams = () => fetch(`${apiBase}/api/governance/system-params`).then(r => r.json()).then(d => setSysParams(Array.isArray(d) ? d : [])).catch(() => setSysParams([]));

  useEffect(() => { loadFlags(); loadParams(); }, []);

  const toggleFlag = (flag) => {
    fetch(`${apiBase}/api/governance/feature-flags/${flag.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...flag, isEnabled: !flag.isEnabled }),
    }).then(() => loadFlags());
  };

  const handleFlagSubmit = (e) => {
    e.preventDefault();
    fetch(`${apiBase}/api/governance/feature-flags`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(flagForm),
    }).then(() => { setShowFlagForm(false); setFlagForm({ name: '', description: '', environment: 'Production', isEnabled: false }); loadFlags(); });
  };

  const handleParamSubmit = (e) => {
    e.preventDefault();
    fetch(`${apiBase}/api/governance/system-params`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(paramForm),
    }).then(() => { setShowParamForm(false); setParamForm({ key: '', value: '', description: '', category: 'General' }); loadParams(); });
  };

  const ENV_COLOR = { Production: { bg: '#dcfce7', color: '#16a34a' }, Staging: { bg: '#fef9c3', color: '#ca8a04' }, Development: { bg: '#eff6ff', color: '#2563eb' } };

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
        <div>
          <h2 style={{ margin: 0, fontSize: '1.4rem', fontWeight: 700 }}>⚙️ System Administration & Configuration</h2>
          <p style={{ margin: '0.25rem 0 0', fontSize: '0.875rem', color: '#64748b' }}>Feature flags, runtime parameters, and system-wide configuration</p>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button style={TOGGLE_STYLE(activeSection === 'flags')} onClick={() => setActiveSection('flags')}>🚩 Feature Flags</button>
          <button style={TOGGLE_STYLE(activeSection === 'params')} onClick={() => setActiveSection('params')}>🔧 System Params</button>
        </div>
      </div>

      {activeSection === 'flags' && (
        <>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
            <div style={{ display: 'flex', gap: '0.75rem' }}>
              {[
                { label: 'Total Flags', value: featureFlags.length, color: '#3b82f6' },
                { label: 'Enabled', value: featureFlags.filter(f => f.isEnabled).length, color: '#22c55e' },
                { label: 'Disabled', value: featureFlags.filter(f => !f.isEnabled).length, color: '#94a3b8' },
              ].map(s => (
                <div key={s.label} style={{ background: '#fff', padding: '0.6rem 1rem', borderRadius: '8px', border: '1px solid #e2e8f0', display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                  <span style={{ fontWeight: 800, fontSize: '1.25rem', color: s.color }}>{s.value}</span>
                  <span style={{ fontSize: '0.78rem', color: '#64748b' }}>{s.label}</span>
                </div>
              ))}
            </div>
            <button className="btn btn-primary" onClick={() => setShowFlagForm(v => !v)}>{showFlagForm ? '✕ Close' : '➕ New Flag'}</button>
          </div>

          {showFlagForm && (
            <div style={{ background: '#f8fafc', padding: '1.25rem', borderRadius: '10px', border: '1px solid #e2e8f0', marginBottom: '1rem' }}>
              <form onSubmit={handleFlagSubmit} style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: '0.875rem' }}>
                <div style={{ gridColumn: 'span 2' }}>
                  <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Flag Name (snake_case)</label>
                  <input type="text" required placeholder="enable_spatial_module" value={flagForm.name} onChange={e => setFlagForm(p => ({ ...p, name: e.target.value }))}
                    style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem', fontFamily: 'monospace' }} />
                </div>
                <div>
                  <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Environment</label>
                  <select value={flagForm.environment} onChange={e => setFlagForm(p => ({ ...p, environment: e.target.value }))}
                    style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }}>
                    {['Production','Staging','Development'].map(v => <option key={v}>{v}</option>)}
                  </select>
                </div>
                <div style={{ gridColumn: 'span 2' }}>
                  <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>Description</label>
                  <input type="text" placeholder="What does this flag control?" value={flagForm.description} onChange={e => setFlagForm(p => ({ ...p, description: e.target.value }))}
                    style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem' }} />
                </div>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                  <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer', fontSize: '0.875rem', marginTop: '1.5rem' }}>
                    <input type="checkbox" checked={flagForm.isEnabled} onChange={e => setFlagForm(p => ({ ...p, isEnabled: e.target.checked }))} />
                    Enable immediately
                  </label>
                </div>
                <div style={{ gridColumn: 'span 3', display: 'flex', justifyContent: 'flex-end', gap: '0.5rem', borderTop: '1px solid #e2e8f0', paddingTop: '0.75rem' }}>
                  <button type="button" className="btn btn-secondary" onClick={() => setShowFlagForm(false)}>Cancel</button>
                  <button type="submit" className="btn btn-primary">Create Flag</button>
                </div>
              </form>
            </div>
          )}

          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            {featureFlags.length > 0 ? featureFlags.map(flag => (
              <div key={flag.id} style={{ background: '#fff', padding: '1rem 1.25rem', borderRadius: '10px', border: '1px solid #e2e8f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                    <code style={{ fontSize: '0.9rem', fontWeight: 700, color: '#1e3a5f', background: '#f1f5f9', padding: '0.15rem 0.5rem', borderRadius: '4px' }}>{flag.name}</code>
                    <span style={{ fontSize: '0.72rem', padding: '0.15rem 0.5rem', borderRadius: '4px', fontWeight: 600, background: ENV_COLOR[flag.environment]?.bg || '#f1f5f9', color: ENV_COLOR[flag.environment]?.color || '#475569' }}>
                      {flag.environment}
                    </span>
                  </div>
                  <p style={{ margin: '0.3rem 0 0', fontSize: '0.82rem', color: '#64748b' }}>{flag.description}</p>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
                  <span style={{ fontSize: '0.8rem', color: flag.isEnabled ? '#16a34a' : '#94a3b8', fontWeight: 600 }}>{flag.isEnabled ? 'ENABLED' : 'DISABLED'}</span>
                  <button onClick={() => toggleFlag(flag)}
                    style={{ width: '44px', height: '24px', borderRadius: '12px', border: 'none', cursor: 'pointer', position: 'relative', background: flag.isEnabled ? '#22c55e' : '#cbd5e1', transition: 'background 0.2s' }}>
                    <div style={{ position: 'absolute', top: '3px', left: flag.isEnabled ? '22px' : '3px', width: '18px', height: '18px', borderRadius: '50%', background: '#fff', transition: 'left 0.2s', boxShadow: '0 1px 3px rgba(0,0,0,0.2)' }} />
                  </button>
                </div>
              </div>
            )) : (
              <div style={{ padding: '2.5rem', textAlign: 'center', background: '#f8fafc', borderRadius: '10px', border: '1px dashed #cbd5e1', color: '#94a3b8' }}>
                No feature flags defined yet.
              </div>
            )}
          </div>
        </>
      )}

      {activeSection === 'params' && (
        <>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '1rem' }}>
            <button className="btn btn-primary" onClick={() => setShowParamForm(v => !v)}>{showParamForm ? '✕ Close' : '➕ New Parameter'}</button>
          </div>

          {showParamForm && (
            <div style={{ background: '#f8fafc', padding: '1.25rem', borderRadius: '10px', border: '1px solid #e2e8f0', marginBottom: '1rem' }}>
              <form onSubmit={handleParamSubmit} style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: '0.875rem' }}>
                {[
                  { key: 'key', label: 'Parameter Key', placeholder: 'MAX_FILE_UPLOAD_MB' },
                  { key: 'value', label: 'Value', placeholder: '50' },
                  { key: 'category', label: 'Category', placeholder: 'Storage' },
                  { key: 'description', label: 'Description', placeholder: 'Maximum upload size in megabytes', span: 3 },
                ].map(f => (
                  <div key={f.key} style={{ gridColumn: f.span ? `span ${f.span}` : undefined }}>
                    <label style={{ fontSize: '0.78rem', fontWeight: 600, color: '#374151', display: 'block', marginBottom: '0.25rem' }}>{f.label}</label>
                    <input type="text" placeholder={f.placeholder} value={paramForm[f.key]} onChange={e => setParamForm(p => ({ ...p, [f.key]: e.target.value }))}
                      style={{ width: '100%', padding: '0.45rem 0.75rem', borderRadius: '6px', border: '1px solid #cbd5e1', fontSize: '0.875rem', fontFamily: f.key === 'key' ? 'monospace' : 'inherit' }} />
                  </div>
                ))}
                <div style={{ gridColumn: 'span 3', display: 'flex', justifyContent: 'flex-end', gap: '0.5rem', borderTop: '1px solid #e2e8f0', paddingTop: '0.75rem' }}>
                  <button type="button" className="btn btn-secondary" onClick={() => setShowParamForm(false)}>Cancel</button>
                  <button type="submit" className="btn btn-primary">Save Parameter</button>
                </div>
              </form>
            </div>
          )}

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px,1fr))', gap: '0.875rem' }}>
            {sysParams.length > 0 ? sysParams.map(p => (
              <div key={p.id} style={{ background: '#fff', padding: '1rem 1.25rem', borderRadius: '10px', border: '1px solid #e2e8f0' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
                  <code style={{ fontSize: '0.8rem', fontWeight: 700, color: '#1e3a5f', background: '#f1f5f9', padding: '0.15rem 0.4rem', borderRadius: '4px' }}>{p.key}</code>
                  <span style={{ fontSize: '0.7rem', color: '#94a3b8', background: '#f8fafc', padding: '0.15rem 0.4rem', borderRadius: '4px' }}>{p.category}</span>
                </div>
                <div style={{ fontSize: '1.1rem', fontWeight: 800, color: '#0f172a', marginBottom: '0.35rem', fontFamily: 'monospace' }}>{p.value}</div>
                <p style={{ margin: 0, fontSize: '0.78rem', color: '#64748b' }}>{p.description}</p>
              </div>
            )) : (
              <div style={{ gridColumn: 'span 3', padding: '2.5rem', textAlign: 'center', background: '#f8fafc', borderRadius: '10px', border: '1px dashed #cbd5e1', color: '#94a3b8' }}>
                No system parameters configured yet.
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
