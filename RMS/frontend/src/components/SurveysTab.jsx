import React, { useEffect, useState } from 'react';

export default function SurveysTab({ apiBase, surveys, researchId, onRefresh }) {
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState({ name: '', status: 'Template', targetSample: 1000 });
  const [links, setLinks] = useState([]);
  const [linkForm, setLinkForm] = useState({ submissionId: '', formId: '' });
  const [linkError, setLinkError] = useState('');

  const loadLinks = () => {
    if (!researchId) return;
    fetch(`${apiBase}/api/research/${researchId}/statcollect-links`)
      .then(response => response.ok ? response.json() : Promise.reject(new Error('Could not load StatCollect links.')))
      .then(setLinks)
      .catch(error => setLinkError(error.message));
  };

  useEffect(loadLinks, [apiBase, researchId]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    await fetch(`${apiBase}/api/surveys`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...formData, researchId })
    });
    setShowForm(false);
    setFormData({ name: '', status: 'Template', targetSample: 1000 });
    onRefresh();
  };

  const handleLink = async (event) => {
    event.preventDefault();
    setLinkError('');
    const response = await fetch(`${apiBase}/api/research/${researchId}/statcollect-links`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(linkForm),
    });
    const body = await response.json();
    if (!response.ok) {
      setLinkError(body.error || 'Could not link the StatCollect submission.');
      return;
    }
    setLinks(current => [body, ...current.filter(link => link.id !== body.id)]);
    setLinkForm({ submissionId: '', formId: '' });
  };

  const removeLink = async (linkId) => {
    const response = await fetch(`${apiBase}/api/research/${researchId}/statcollect-links/${linkId}`, { method: 'DELETE' });
    if (response.ok) setLinks(current => current.filter(link => link.id !== linkId));
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

      <div className="glass-panel" style={{ padding: '24px', marginTop: '24px' }}>
        <h4>StatCollect evidence links</h4>
        <p style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Attach a collected submission to this workspace-owned study without copying raw response data into RMS. StatCollect events with explicit research and workspace context are linked automatically.</p>
        <form onSubmit={handleLink} style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', margin: '16px 0' }}>
          <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', flex: 1, minWidth: '220px' }} placeholder="Submission instance ID" required value={linkForm.submissionId} onChange={e => setLinkForm({ ...linkForm, submissionId: e.target.value })} />
          <input style={{ padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', flex: 1, minWidth: '180px' }} placeholder="Form ID (optional)" value={linkForm.formId} onChange={e => setLinkForm({ ...linkForm, formId: e.target.value })} />
          <button type="submit" className="btn btn-primary">Link submission</button>
        </form>
        {linkError && <p style={{ color: '#b91c1c' }}>{linkError}</p>}
        {links.length === 0 ? <p style={{ color: 'var(--text-secondary)' }}>No StatCollect submissions linked yet.</p> : links.map(link => (
          <div key={link.id} style={{ display: 'flex', justifyContent: 'space-between', gap: '12px', alignItems: 'center', borderTop: '1px solid var(--border-light)', padding: '12px 0' }}>
            <div><strong>{link.submissionId}</strong><div style={{ color: 'var(--text-secondary)', fontSize: '12px' }}>{link.formId || 'form not supplied'} | {link.status} | {link.sourceEventType || 'user-linked'}</div></div>
            <button type="button" className="btn btn-secondary" onClick={() => removeLink(link.id)}>Unlink</button>
          </div>
        ))}
      </div>
    </div>
  );
}
