import React, { useState, useEffect } from 'react';

export default function RisksTab({ apiBase, token, onRefreshDashboard }) {
  const [risks, setRisks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedRisk, setSelectedRisk] = useState(null);
  const [showModal, setShowModal] = useState(false);

  const [formData, setFormData] = useState({
    title: '',
    risk_code: '',
    category: 'Operational',
    department: 'Data Analytics',
    description: '',
    probability: 3,
    impact: 3,
    treatment_strategy: 'Mitigate',
    mitigation_actions: '',
    residual_probability: 2,
    residual_impact: 2,
    target_date: '',
    review_date: ''
  });

  const fetchRisks = async () => {
    try {
      setLoading(true);
      const res = await fetch(`${apiBase}/api/risks`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setRisks(data);
        if (data.length > 0 && !selectedRisk) {
          setSelectedRisk(data[0]);
        }
      }
    } catch (err) {
      console.error('Failed to load risks:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRisks();
  }, [apiBase, token]);

  const handleCreateRisk = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/risks`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          ...formData,
          probability: parseInt(formData.probability),
          impact: parseInt(formData.impact),
          residual_probability: parseInt(formData.residual_probability),
          residual_impact: parseInt(formData.residual_impact)
        })
      });
      if (res.ok) {
        setShowModal(false);
        setFormData({
          title: '', risk_code: '', category: 'Operational', department: 'Data Analytics',
          description: '', probability: 3, impact: 3, treatment_strategy: 'Mitigate',
          mitigation_actions: '', residual_probability: 2, residual_impact: 2,
          target_date: '', review_date: ''
        });
        fetchRisks();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating risk:', err);
    }
  };

  const handleEscalate = async (id) => {
    try {
      const res = await fetch(`${apiBase}/api/risks/${id}/escalate`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        alert('Risk successfully escalated to Enterprise Core and Executive Notice queue!');
        fetchRisks();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error escalating risk:', err);
    }
  };

  const getRiskBadge = (level) => {
    switch (level) {
      case 'Critical':
        return <span className="badge badge-critical">Critical</span>;
      case 'High':
        return <span className="badge badge-high">High</span>;
      case 'Medium':
        return <span className="badge badge-medium">Medium</span>;
      default:
        return <span className="badge badge-low">Low</span>;
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>🎯</span> Institutional Risk Register & Treatment System
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Identify, assess, treat, monitor, escalate, and close institutional risks.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Log New Risk
        </button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 24 }}>
        {/* Risk Register Table */}
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>Risk Code</th>
                <th>Title & Category</th>
                <th>Inherent Risk</th>
                <th>Residual Risk</th>
                <th>Strategy</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {risks.length === 0 ? (
                <tr>
                  <td colSpan="6" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                    No institutional risks registered.
                  </td>
                </tr>
              ) : (
                risks.map(r => (
                  <tr
                    key={r.id}
                    onClick={() => setSelectedRisk(r)}
                    style={{
                      cursor: 'pointer',
                      background: selectedRisk?.id === r.id ? '#f1f5f9' : 'transparent'
                    }}
                  >
                    <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, fontSize: 12 }}>
                      {r.risk_code}
                    </td>
                    <td>
                      <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{r.title}</div>
                      <div style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>{r.category} • {r.department}</div>
                    </td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        {getRiskBadge(r.inherent_risk_level)}
                        <span style={{ fontSize: 12, fontWeight: 700 }}>{r.inherent_risk_score}</span>
                      </div>
                    </td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        {r.residual_risk_level ? getRiskBadge(r.residual_risk_level) : '—'}
                        {r.residual_risk_score ? <span style={{ fontSize: 12, fontWeight: 700 }}>{r.residual_risk_score}</span> : ''}
                      </div>
                    </td>
                    <td>
                      <span className="badge badge-draft">{r.treatment_strategy}</span>
                    </td>
                    <td>
                      <button
                        className="btn btn-danger btn-sm"
                        onClick={(e) => { e.stopPropagation(); handleEscalate(r.id); }}
                        title="Escalate Risk to Enterprise Core"
                      >
                        Escalate
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Selected Risk Inspection */}
        <div className="glass-panel" style={{ padding: 22 }}>
          {selectedRisk ? (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
                <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, fontWeight: 700, color: 'var(--primary-color)' }}>
                  {selectedRisk.risk_code}
                </span>
                {getRiskBadge(selectedRisk.inherent_risk_level)}
              </div>

              <h3 style={{ fontSize: 18, fontWeight: 800, marginBottom: 12, color: 'var(--text-dark)' }}>
                {selectedRisk.title}
              </h3>

              <div style={{ background: '#f8fafc', padding: 14, borderRadius: 8, border: '1px solid var(--border-light)', marginBottom: 16 }}>
                <div style={{ fontSize: 11, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 4 }}>
                  Risk Description & Context
                </div>
                <div style={{ fontSize: 13, color: 'var(--text-dark)', lineHeight: 1.6 }}>
                  {selectedRisk.description}
                </div>
              </div>

              <div className="form-row" style={{ marginBottom: 16 }}>
                <div style={{ background: '#ffffff', padding: 12, borderRadius: 8, border: '1px solid var(--border-light)' }}>
                  <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)' }}>Inherent Risk</div>
                  <div style={{ fontSize: 18, fontWeight: 800, color: 'var(--accent-ruby)', marginTop: 2 }}>
                    {selectedRisk.probability} × {selectedRisk.impact} = {selectedRisk.inherent_risk_score}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>Level: {selectedRisk.inherent_risk_level}</div>
                </div>

                <div style={{ background: '#ffffff', padding: 12, borderRadius: 8, border: '1px solid var(--border-light)' }}>
                  <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-muted)' }}>Residual Risk</div>
                  <div style={{ fontSize: 18, fontWeight: 800, color: 'var(--accent-emerald)', marginTop: 2 }}>
                    {selectedRisk.residual_probability || '—'} × {selectedRisk.residual_impact || '—'} = {selectedRisk.residual_risk_score || '—'}
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>Level: {selectedRisk.residual_risk_level || 'Not Assessed'}</div>
                </div>
              </div>

              <div style={{ marginBottom: 16 }}>
                <div style={{ fontSize: 11, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 4 }}>
                  Mitigation & Treatment Action Plan
                </div>
                <div style={{ fontSize: 13, color: 'var(--text-dark)', background: '#ffffff', padding: 12, borderRadius: 8, border: '1px solid var(--border-light)' }}>
                  {selectedRisk.mitigation_actions || 'No treatment action recorded.'}
                </div>
              </div>

              <div style={{ fontSize: 12, color: 'var(--text-muted)', borderTop: '1px solid var(--border-light)', paddingTop: 12, display: 'flex', justifyContent: 'space-between' }}>
                <span>Owner: <strong>{selectedRisk.owner}</strong></span>
                <span>Review: <strong>{selectedRisk.review_date || 'Ongoing'}</strong></span>
              </div>
            </div>
          ) : (
            <div style={{ textAlign: 'center', padding: 40, color: 'var(--text-muted)' }}>
              Select a risk to inspect matrix calculation and treatment plans.
            </div>
          )}
        </div>
      </div>

      {/* New Risk Modal */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Log Institutional Risk</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateRisk}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Risk Code</label>
                    <input
                      className="form-control"
                      placeholder="e.g. RSK-SEC-2026-09"
                      value={formData.risk_code}
                      onChange={e => setFormData({ ...formData, risk_code: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Category</label>
                    <select
                      className="form-control"
                      value={formData.category}
                      onChange={e => setFormData({ ...formData, category: e.target.value })}
                    >
                      <option value="Operational">Operational</option>
                      <option value="Information Security">Information Security</option>
                      <option value="Compliance">Compliance</option>
                      <option value="Data Integrity">Data Integrity</option>
                      <option value="Strategic">Strategic</option>
                      <option value="Financial">Financial</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Risk Title *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. Unauthorized Access to Field Microdata"
                    value={formData.title}
                    onChange={e => setFormData({ ...formData, title: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Risk Description & Institutional Threat *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Describe the cause, scenario, and potential organizational impact..."
                    value={formData.description}
                    onChange={e => setFormData({ ...formData, description: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Inherent Probability (1-5) *</label>
                    <select
                      className="form-control"
                      value={formData.probability}
                      onChange={e => setFormData({ ...formData, probability: e.target.value })}
                    >
                      <option value="1">1 - Rare</option>
                      <option value="2">2 - Unlikely</option>
                      <option value="3">3 - Possible</option>
                      <option value="4">4 - Likely</option>
                      <option value="5">5 - Almost Certain</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Inherent Impact (1-5) *</label>
                    <select
                      className="form-control"
                      value={formData.impact}
                      onChange={e => setFormData({ ...formData, impact: e.target.value })}
                    >
                      <option value="1">1 - Insignificant</option>
                      <option value="2">2 - Minor</option>
                      <option value="3">3 - Moderate</option>
                      <option value="4">4 - Major</option>
                      <option value="5">5 - Catastrophic</option>
                    </select>
                  </div>
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Treatment Strategy</label>
                    <select
                      className="form-control"
                      value={formData.treatment_strategy}
                      onChange={e => setFormData({ ...formData, treatment_strategy: e.target.value })}
                    >
                      <option value="Mitigate">Mitigate (Implement Controls)</option>
                      <option value="Avoid">Avoid (Discontinue Activity)</option>
                      <option value="Transfer">Transfer (Insurance / Third-Party)</option>
                      <option value="Accept">Accept (Residual Tolerance)</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Department / Function</label>
                    <input
                      className="form-control"
                      placeholder="e.g. Information Technology"
                      value={formData.department}
                      onChange={e => setFormData({ ...formData, department: e.target.value })}
                    />
                  </div>
                </div>

                <div className="form-group">
                  <label>Mitigation Action Plan</label>
                  <textarea
                    className="form-control"
                    placeholder="Specify internal controls and preventive tasks..."
                    value={formData.mitigation_actions}
                    onChange={e => setFormData({ ...formData, mitigation_actions: e.target.value })}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Risk Record</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
