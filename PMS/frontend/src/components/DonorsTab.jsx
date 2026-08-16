import React, { useState, useEffect } from 'react';

export default function DonorsTab({ apiBase }) {
  const [donors, setDonors] = useState([]);
  const [showAddModal, setShowAddModal] = useState(false);
  const [name, setName] = useState('');
  const [code, setCode] = useState('');
  const [type, setType] = useState('Bilateral');
  const [contactPerson, setContactPerson] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [website, setWebsite] = useState('');
  const [totalFunding, setTotalFunding] = useState(500000);
  const [currency, setCurrency] = useState('USD');

  const loadDonors = () => {
    fetch(`${apiBase}/api/donors`)
      .then(res => res.json())
      .then(data => setDonors(Array.isArray(data) ? data : []))
      .catch(err => console.error("Error loading donors:", err));
  };

  useEffect(() => {
    loadDonors();
  }, []);

  const handleCreateDonor = (e) => {
    e.preventDefault();
    fetch(`${apiBase}/api/donors`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name,
        code,
        type,
        contactPerson,
        email,
        phone,
        website,
        totalFunding: Number(totalFunding),
        currency,
        status: 'Active'
      })
    })
      .then(res => res.json())
      .then(() => {
        setName('');
        setCode('');
        setContactPerson('');
        setEmail('');
        setPhone('');
        setWebsite('');
        setShowAddModal(false);
        loadDonors();
      })
      .catch(err => console.error("Error creating donor:", err));
  };

  return (
    <div className="tab-pane-container">
      <div className="tab-header-row" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
        <div>
          <h2 style={{ margin: 0, fontSize: '1.4rem', fontWeight: 600 }}>Institutional Donors & Funding Partners</h2>
          <p style={{ margin: '0.25rem 0 0', color: '#64748b', fontSize: '0.9rem' }}>
            CRM directory for bilateral donors, multilateral institutions, foundations, and grantors
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowAddModal(true)}>
          ➕ Register Partner
        </button>
      </div>

      {showAddModal && (
        <div style={{ background: '#f8fafc', padding: '1.5rem', borderRadius: '8px', border: '1px solid #cbd5e1', marginBottom: '1.5rem' }}>
          <h3 style={{ marginTop: 0, fontSize: '1.1rem' }}>Register Institutional Donor</h3>
          <form onSubmit={handleCreateDonor} style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '1rem' }}>
            <div>
              <label style={{ fontSize: '0.8rem', fontWeight: 600 }}>Partner Name</label>
              <input type="text" value={name} onChange={e => setName(e.target.value)} required placeholder="e.g. USAID / Foreign Dev Agency" style={{ width: '100%', padding: '0.5rem' }} />
            </div>
            <div>
              <label style={{ fontSize: '0.8rem', fontWeight: 600 }}>Code</label>
              <input type="text" value={code} onChange={e => setCode(e.target.value)} required placeholder="e.g. DON-USAID-01" style={{ width: '100%', padding: '0.5rem' }} />
            </div>
            <div>
              <label style={{ fontSize: '0.8rem', fontWeight: 600 }}>Category</label>
              <select value={type} onChange={e => setType(e.target.value)} style={{ width: '100%', padding: '0.5rem' }}>
                <option value="Bilateral">Bilateral Agency</option>
                <option value="Multilateral">Multilateral (UN / World Bank)</option>
                <option value="Foundation">Philanthropic Foundation</option>
                <option value="NGO">International NGO</option>
                <option value="Private">Private Corporate Partner</option>
              </select>
            </div>
            <div>
              <label style={{ fontSize: '0.8rem', fontWeight: 600 }}>Contact Person</label>
              <input type="text" value={contactPerson} onChange={e => setContactPerson(e.target.value)} placeholder="Lead Portfolio Officer" style={{ width: '100%', padding: '0.5rem' }} />
            </div>
            <div>
              <label style={{ fontSize: '0.8rem', fontWeight: 600 }}>Email</label>
              <input type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="grants@partner.org" style={{ width: '100%', padding: '0.5rem' }} />
            </div>
            <div>
              <label style={{ fontSize: '0.8rem', fontWeight: 600 }}>Committed Funding (USD)</label>
              <input type="number" value={totalFunding} onChange={e => setTotalFunding(e.target.value)} style={{ width: '100%', padding: '0.5rem' }} />
            </div>
            <div style={{ gridColumn: 'span 3', display: 'flex', justifyContent: 'flex-end', gap: '0.5rem' }}>
              <button type="button" className="btn btn-outline" onClick={() => setShowAddModal(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Save Partner</button>
            </div>
          </form>
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: '1.25rem' }}>
        {donors.length > 0 ? (
          donors.map(d => (
            <div key={d.id} style={{ background: '#fff', padding: '1.25rem', borderRadius: '8px', border: '1px solid #e2e8f0', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <h3 style={{ margin: '0 0 0.25rem', fontSize: '1.1rem', color: '#0f172a' }}>{d.name}</h3>
                <span style={{ fontSize: '0.75rem', padding: '0.2rem 0.5rem', background: '#e0f2fe', color: '#0369a1', borderRadius: '4px', fontWeight: 600 }}>
                  {d.type}
                </span>
              </div>
              <p style={{ fontSize: '0.8rem', color: '#64748b', margin: '0 0 1rem' }}>Code: {d.code || 'N/A'}</p>
              
              <div style={{ borderTop: '1px solid #f1f5f9', paddingTop: '0.75rem', fontSize: '0.85rem' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.4rem' }}>
                  <span style={{ color: '#64748b' }}>Contact:</span>
                  <span style={{ fontWeight: 500, color: '#334155' }}>{d.contactPerson || 'Not designated'}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.4rem' }}>
                  <span style={{ color: '#64748b' }}>Email:</span>
                  <span style={{ color: '#2563eb' }}>{d.email || 'N/A'}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '0.75rem', background: '#f8fafc', padding: '0.5rem', borderRadius: '4px' }}>
                  <span style={{ fontWeight: 600, color: '#475569' }}>Total Committed:</span>
                  <span style={{ fontWeight: 700, color: '#16a34a' }}>${(d.totalFunding || 0).toLocaleString()} {d.currency || 'USD'}</span>
                </div>
              </div>
            </div>
          ))
        ) : (
          <div style={{ gridColumn: 'span 3', padding: '3rem', textAlign: 'center', background: '#fff', borderRadius: '8px', border: '1px dashed #cbd5e1', color: '#94a3b8' }}>
            No institutional funding partners registered yet. Click "Register Partner" to add a donor profile.
          </div>
        )}
      </div>
    </div>
  );
}
