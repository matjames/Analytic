import React, { useState, useEffect } from 'react';

export default function DataGovernanceTab({ apiBase, token, onRefreshDashboard }) {
  const [dataAssets, setDataAssets] = useState([]);
  const [privacyAssessments, setPrivacyAssessments] = useState([]);
  const [activeSubView, setActiveSubView] = useState('assets');
  const [showModal, setShowModal] = useState(false);

  const [formData, setFormData] = useState({
    dataset_name: '',
    dataset_source: 'StatCollect',
    data_owner: '',
    data_steward: '',
    classification: 'Confidential',
    sensitivity_level: 'Medium',
    retention_period: '7 Years',
    access_rules: 'Restricted to accredited statistical researchers'
  });

  const fetchDataGovernance = async () => {
    try {
      const [astRes, priRes] = await Promise.all([
        fetch(`${apiBase}/api/data-governance`, { headers: { 'Authorization': `Bearer ${token}` } }),
        fetch(`${apiBase}/api/data-governance/privacy`, { headers: { 'Authorization': `Bearer ${token}` } })
      ]);
      if (astRes.ok) setDataAssets(await astRes.json());
      if (priRes.ok) setPrivacyAssessments(await priRes.json());
    } catch (err) {
      console.error('Error loading data governance:', err);
    }
  };

  useEffect(() => {
    fetchDataGovernance();
  }, [apiBase, token]);

  const handleCreateAsset = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/data-governance`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(formData)
      });
      if (res.ok) {
        setShowModal(false);
        fetchDataGovernance();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating data asset record:', err);
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>🗄️</span> Data Governance, Stewardship & Privacy Registers
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Maintain dataset classification, data stewardship, retention schedules, and DPIA compliance across StatGate.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Register Data Asset
        </button>
      </div>

      <div style={{ display: 'flex', gap: 10, marginBottom: 16 }}>
        <button
          className={`btn ${activeSubView === 'assets' ? 'btn-primary' : 'btn-secondary'} btn-sm`}
          onClick={() => setActiveSubView('assets')}
        >
          Data Assets & Stewardship ({dataAssets.length})
        </button>
        <button
          className={`btn ${activeSubView === 'privacy' ? 'btn-primary' : 'btn-secondary'} btn-sm`}
          onClick={() => setActiveSubView('privacy')}
        >
          Privacy & DPIA Registers ({privacyAssessments.length})
        </button>
      </div>

      {activeSubView === 'assets' && (
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>Dataset Name</th>
                <th>Source Application</th>
                <th>Data Owner / Steward</th>
                <th>Classification</th>
                <th>Sensitivity</th>
                <th>Retention</th>
                <th>Access Rules</th>
              </tr>
            </thead>
            <tbody>
              {dataAssets.length === 0 ? (
                <tr>
                  <td colSpan="7" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                    No dataset governance records registered.
                  </td>
                </tr>
              ) : (
                dataAssets.map(d => (
                  <tr key={d.id}>
                    <td style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{d.dataset_name}</td>
                    <td><span className="badge badge-draft">{d.dataset_source}</span></td>
                    <td>
                      <div style={{ fontSize: 12.5, fontWeight: 500 }}>{d.data_owner}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Steward: {d.data_steward}</div>
                    </td>
                    <td>
                      <span className={`badge ${d.classification === 'Public' ? 'badge-active' : d.classification === 'Restricted' ? 'badge-high' : 'badge-draft'}`}>
                        {d.classification}
                      </span>
                    </td>
                    <td><span className="badge badge-review">{d.sensitivity_level}</span></td>
                    <td style={{ fontSize: 12 }}>{d.retention_period}</td>
                    <td style={{ fontSize: 12, maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {d.access_rules}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {activeSubView === 'privacy' && (
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>Activity / Processing Operation</th>
                <th>Data Controller</th>
                <th>Lawful Basis</th>
                <th>DPIA Status</th>
                <th>Retention Rules</th>
              </tr>
            </thead>
            <tbody>
              {privacyAssessments.map(p => (
                <tr key={p.id}>
                  <td style={{ fontWeight: 600 }}>{p.activity_name}</td>
                  <td>{p.data_controller}</td>
                  <td><span className="badge badge-draft">{p.lawful_basis}</span></td>
                  <td><span className="badge badge-approved">{p.dpia_status}</span></td>
                  <td style={{ fontSize: 12 }}>{p.retention_rules}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Modal */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Register Governed Data Asset</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateAsset}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Dataset Name *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. National Household Survey Microdata 2026"
                      value={formData.dataset_name}
                      onChange={e => setFormData({ ...formData, dataset_name: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Source Application</label>
                    <select
                      className="form-control"
                      value={formData.dataset_source}
                      onChange={e => setFormData({ ...formData, dataset_source: e.target.value })}
                    >
                      <option value="StatCollect">StatCollect</option>
                      <option value="RMS">RMS (Research)</option>
                      <option value="Analytics">Analytics Hub</option>
                      <option value="PMS">PMS (Projects)</option>
                      <option value="Registry">Field Operations Registry</option>
                    </select>
                  </div>
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Data Owner *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Chief Statistician"
                      value={formData.data_owner}
                      onChange={e => setFormData({ ...formData, data_owner: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Data Steward *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Principal Data Engineer"
                      value={formData.data_steward}
                      onChange={e => setFormData({ ...formData, data_steward: e.target.value })}
                    />
                  </div>
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Classification</label>
                    <select
                      className="form-control"
                      value={formData.classification}
                      onChange={e => setFormData({ ...formData, classification: e.target.value })}
                    >
                      <option value="Public">Public</option>
                      <option value="Internal">Internal</option>
                      <option value="Confidential">Confidential</option>
                      <option value="Highly Restricted">Highly Restricted</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Sensitivity Level</label>
                    <select
                      className="form-control"
                      value={formData.sensitivity_level}
                      onChange={e => setFormData({ ...formData, sensitivity_level: e.target.value })}
                    >
                      <option value="Low">Low</option>
                      <option value="Medium">Medium</option>
                      <option value="High">High</option>
                      <option value="Special Category (PII/Health)">Special Category (PII/Health)</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Access Control Rules *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Specify role-based permissions, anonymization criteria, and sharing constraints..."
                    value={formData.access_rules}
                    onChange={e => setFormData({ ...formData, access_rules: e.target.value })}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Data Asset</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
