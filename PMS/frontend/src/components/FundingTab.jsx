import React, { useState } from 'react';

const API_BASE = import.meta.env.VITE_PMS_API_URL || 'http://localhost:8091';

export default function FundingTab({ fundingSources = [], projectId, onRefresh }) {
  const [isAdding, setIsAdding] = useState(false);
  const [name, setName] = useState('');
  const [type, setType] = useState('Government');
  const [donor, setDonor] = useState('');
  const [amount, setAmount] = useState(0);
  const [received, setReceived] = useState(0);
  const [currency, setCurrency] = useState('USD');
  const [status, setStatus] = useState('Active');

  const handleAdd = (e) => {
    e.preventDefault();
    fetch(`${API_BASE}/api/funding`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId, name, type, donor, amount: Number(amount), received: Number(received), currency, status })
    })
      .then(res => res.json())
      .then(() => {
        onRefresh();
        setIsAdding(false);
        setName(''); setDonor(''); setAmount(0); setReceived(0);
      })
      .catch(err => console.error("Error creating funding source:", err));
  };

  const totalCommitted = fundingSources.reduce((a, f) => a + f.amount, 0);
  const totalReceived = fundingSources.reduce((a, f) => a + f.received, 0);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>💰 Funding & Donors</h2>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
            Funding sources, donors, grants, and financial commitments
          </p>
        </div>
        <button className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 16px' }} onClick={() => setIsAdding(true)}>
          + Add Funding Source
        </button>
      </div>

      {isAdding && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleAdd}>
            <h3>Add Funding Source</h3>
            <div className="form-group">
              <label>Funding Name</label>
              <input type="text" value={name} onChange={e => setName(e.target.value)} placeholder="e.g. WHO NCD Grant" required />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Type</label>
                <select value={type} onChange={e => setType(e.target.value)}>
                  <option value="Government">Government</option>
                  <option value="International">International</option>
                  <option value="NGO">NGO</option>
                  <option value="Private">Private</option>
                  <option value="Internal">Internal</option>
                </select>
              </div>
              <div className="form-group">
                <label>Donor</label>
                <input type="text" value={donor} onChange={e => setDonor(e.target.value)} placeholder="e.g. World Health Organization" />
              </div>
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Committed Amount</label>
                <input type="number" value={amount} onChange={e => setAmount(e.target.value)} required />
              </div>
              <div className="form-group">
                <label>Received Amount</label>
                <input type="number" value={received} onChange={e => setReceived(e.target.value)} />
              </div>
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Currency</label>
                <select value={currency} onChange={e => setCurrency(e.target.value)}>
                  <option value="USD">USD</option>
                  <option value="EUR">EUR</option>
                  <option value="GBP">GBP</option>
                  <option value="UGX">UGX</option>
                  <option value="KES">KES</option>
                </select>
              </div>
              <div className="form-group">
                <label>Status</label>
                <select value={status} onChange={e => setStatus(e.target.value)}>
                  <option value="Active">Active</option>
                  <option value="Pending">Pending</option>
                  <option value="Completed">Completed</option>
                  <option value="Cancelled">Cancelled</option>
                </select>
              </div>
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAdding(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Add Funding</button>
            </div>
          </form>
        </div>
      )}

      {/* Summary stats */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        {[
          { label: 'Funding Sources', value: fundingSources.length, icon: '🏦', color: 'var(--accent-primary)' },
          { label: 'Total Committed', value: `$${totalCommitted.toLocaleString()}`, icon: '📝', color: 'var(--accent-secondary)' },
          { label: 'Total Received', value: `$${totalReceived.toLocaleString()}`, icon: '💵', color: 'var(--accent-success)' },
          { label: 'Disbursement Rate', value: `${totalCommitted > 0 ? ((totalReceived / totalCommitted) * 100).toFixed(1) : 0}%`, icon: '📈', color: 'var(--accent-warning)' },
        ].map(stat => (
          <div key={stat.label} className="glass-panel" style={{ padding: '16px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ fontSize: '24px' }}>{stat.icon}</span>
            <div>
              <div style={{ fontSize: '20px', fontWeight: '800', color: stat.color }}>{stat.value}</div>
              <div style={{ fontSize: '10px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>{stat.label}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Funding sources list */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>Funding Sources & Donors</h3>
        {fundingSources.length === 0 ? (
          <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No funding sources defined.</div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            {fundingSources.map(fs => {
              const pct = fs.amount > 0 ? (fs.received / fs.amount) * 100 : 0;
              return (
                <div key={fs.id} style={{ padding: '14px', background: 'rgba(16,185,129,0.03)', borderRadius: '8px', border: '1px solid rgba(16,185,129,0.1)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                    <div>
                      <span style={{ fontSize: '13px', fontWeight: '700' }}>{fs.name}</span>
                      <span className="badge" style={{ fontSize: '9px', marginLeft: '8px', background: 'rgba(22,92,146,0.1)', color: 'var(--primary-color)' }}>{fs.type}</span>
                    </div>
                    <span className="badge" style={{ fontSize: '9px', background: fs.status === 'Active' ? 'rgba(16,185,129,0.1)' : 'rgba(245,158,11,0.1)', color: fs.status === 'Active' ? 'var(--accent-success)' : 'var(--accent-warning)' }}>{fs.status}</span>
                  </div>
                  <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>Donor: {fs.donor}</div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', color: 'var(--text-muted)', marginTop: '6px' }}>
                    <span>Committed: <strong>${fs.amount.toLocaleString()}</strong></span>
                    <span>Received: <strong>${fs.received.toLocaleString()}</strong></span>
                    <span>Currency: {fs.currency}</span>
                  </div>
                  <div style={{ marginTop: '8px' }}>
                    <div className="project-progress-bar" style={{ height: '4px' }}>
                      <div className="project-progress-fill" style={{ width: `${Math.min(100, pct)}%`, background: 'linear-gradient(90deg, #10b981, #34d399)' }} />
                    </div>
                    <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px' }}>{pct.toFixed(1)}% disbursed</div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}