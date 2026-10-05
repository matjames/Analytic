import React, { useEffect, useState } from 'react';

const API_BASE = import.meta.env.VITE_PMS_API_URL || 'http://localhost:8091';

const emptyForm = {
  title: '',
  grantNumber: '',
  donorId: '',
  purpose: '',
  amount: 0,
  currency: 'USD',
  startDate: '',
  endDate: '',
  reportingDue: '',
  status: 'Draft',
};

export default function GrantManagement({ projectId, apiBase = API_BASE }) {
  const [grants, setGrants] = useState([]);
  const [donors, setDonors] = useState([]);
  const [form, setForm] = useState(emptyForm);
  const [isOpen, setIsOpen] = useState(false);
  const [error, setError] = useState('');

  const load = () => {
    if (!projectId) return;
    Promise.all([
      fetch(`${apiBase}/api/projects/${projectId}/grants`).then(res => res.ok ? res.json() : []),
      fetch(`${apiBase}/api/donors`).then(res => res.ok ? res.json() : []),
    ])
      .then(([grantData, donorData]) => {
        setGrants(Array.isArray(grantData) ? grantData : []);
        setDonors(Array.isArray(donorData) ? donorData : []);
      })
      .catch(() => setError('Grant records could not be loaded.'));
  };

  useEffect(load, [projectId, apiBase]);

  const update = (key, value) => setForm(current => ({ ...current, [key]: value }));

  const createGrant = async (event) => {
    event.preventDefault();
    setError('');
    try {
      const response = await fetch(`${apiBase}/api/grants`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, projectId, amount: Number(form.amount) }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Grant could not be saved.');
      setForm(emptyForm);
      setIsOpen(false);
      load();
    } catch (err) {
      setError(err.message || 'Grant could not be saved.');
    }
  };

  const removeGrant = async (id) => {
    if (!window.confirm('Delete this grant record?')) return;
    const response = await fetch(`${apiBase}/api/grants/${id}`, { method: 'DELETE' });
    if (!response.ok) {
      const data = await response.json().catch(() => ({}));
      setError(data.error || 'Grant could not be deleted.');
      return;
    }
    load();
  };

  return (
    <div className="glass-panel" style={{ padding: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '12px' }}>
        <div>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '4px' }}>Grant Management</h3>
          <p style={{ fontSize: '12px', color: 'var(--text-secondary)', margin: 0 }}>Award identifiers, donor ownership, dates, amounts, and reporting commitments.</p>
        </div>
        <button className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 14px' }} onClick={() => { setError(''); setIsOpen(true); }}>+ Add Grant</button>
      </div>

      {error && <div className="form-error" role="alert" style={{ marginTop: '14px' }}>{error}</div>}

      {isOpen && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={createGrant}>
            <h3>Add Grant Award</h3>
            <div className="form-group">
              <label>Grant title</label>
              <input value={form.title} onChange={event => update('title', event.target.value)} placeholder="e.g. Maternal Health Outcomes Grant" required />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Grant number</label>
                <input value={form.grantNumber} onChange={event => update('grantNumber', event.target.value)} placeholder="Award identifier" />
              </div>
              <div className="form-group">
                <label>Donor</label>
                <select value={form.donorId} onChange={event => update('donorId', event.target.value)}>
                  <option value="">Not linked</option>
                  {donors.map(donor => <option key={donor.id} value={donor.id}>{donor.name}</option>)}
                </select>
              </div>
            </div>
            <div className="form-group">
              <label>Purpose</label>
              <textarea value={form.purpose} onChange={event => update('purpose', event.target.value)} rows="3" placeholder="Funded outcomes and restrictions" />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Award amount</label>
                <input type="number" min="0" step="0.01" value={form.amount} onChange={event => update('amount', event.target.value)} required />
              </div>
              <div className="form-group">
                <label>Currency</label>
                <select value={form.currency} onChange={event => update('currency', event.target.value)}>
                  <option value="USD">USD</option><option value="EUR">EUR</option><option value="GBP">GBP</option><option value="UGX">UGX</option><option value="KES">KES</option>
                </select>
              </div>
              <div className="form-group">
                <label>Status</label>
                <select value={form.status} onChange={event => update('status', event.target.value)}>
                  <option value="Draft">Draft</option><option value="Pending">Pending</option><option value="Active">Active</option><option value="Completed">Completed</option><option value="Closed">Closed</option>
                </select>
              </div>
            </div>
            <div className="form-row">
              <div className="form-group"><label>Start date</label><input type="date" value={form.startDate} onChange={event => update('startDate', event.target.value)} /></div>
              <div className="form-group"><label>End date</label><input type="date" value={form.endDate} onChange={event => update('endDate', event.target.value)} /></div>
              <div className="form-group"><label>Reporting due</label><input type="date" value={form.reportingDue} onChange={event => update('reportingDue', event.target.value)} /></div>
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsOpen(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Save Grant</button>
            </div>
          </form>
        </div>
      )}

      {grants.length === 0 ? (
        <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px 0 4px' }}>No grant awards recorded for this project.</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', marginTop: '18px' }}>
          {grants.map(grant => (
            <div key={grant.id} style={{ display: 'grid', gridTemplateColumns: '1fr auto auto', gap: '14px', alignItems: 'center', padding: '12px', border: '1px solid var(--border-light)', borderRadius: '8px' }}>
              <div>
                <div style={{ fontSize: '13px', fontWeight: '700' }}>{grant.title}</div>
                <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{grant.grantNumber || 'No award number'}{grant.reportingDue ? ` | Report due ${grant.reportingDue}` : ''}</div>
              </div>
              <div style={{ fontSize: '12px', fontWeight: '700' }}>{grant.currency} {Number(grant.amount || 0).toLocaleString()}</div>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}><span className="badge">{grant.status}</span><button className="btn btn-secondary" style={{ padding: '4px 8px', fontSize: '11px' }} onClick={() => removeGrant(grant.id)}>Delete</button></div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
