import React, { useState, useEffect } from 'react';

export default function FindingsTab({ apiBase, token, onRefreshDashboard }) {
  const [findings, setFindings] = useState([]);
  const [capas, setCapas] = useState([]);
  const [activeSubView, setActiveSubView] = useState('findings'); // findings, capas
  const [showModal, setShowModal] = useState(false);
  const [showCloseModal, setShowCloseModal] = useState(false);
  const [selectedFinding, setSelectedFinding] = useState(null);
  const [closureEvidence, setClosureEvidence] = useState('');

  const [formData, setFormData] = useState({
    finding_code: '',
    title: '',
    description: '',
    severity: 'High',
    source: 'Audit',
    owner: '',
    department: 'Data Operations',
    recommendation: '',
    due_date: '',
    status: 'Open'
  });

  const fetchFindingsData = async () => {
    try {
      const [fndRes, capaRes] = await Promise.all([
        fetch(`${apiBase}/api/findings`, { headers: { 'Authorization': `Bearer ${token}` } }),
        fetch(`${apiBase}/api/corrective-actions`, { headers: { 'Authorization': `Bearer ${token}` } })
      ]);
      if (fndRes.ok) setFindings(await fndRes.json());
      if (capaRes.ok) setCapas(await capaRes.json());
    } catch (err) {
      console.error('Error loading findings:', err);
    }
  };

  useEffect(() => {
    fetchFindingsData();
  }, [apiBase, token]);

  const handleCreateFinding = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/findings`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(formData)
      });
      if (res.ok) {
        setShowModal(false);
        fetchFindingsData();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating finding:', err);
    }
  };

  const handleCloseFinding = async (e) => {
    e.preventDefault();
    if (!selectedFinding) return;

    try {
      const res = await fetch(`${apiBase}/api/findings/${selectedFinding.id}/close`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ evidence: closureEvidence })
      });
      if (res.ok) {
        setShowCloseModal(false);
        setClosureEvidence('');
        fetchFindingsData();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error closing finding:', err);
    }
  };

  const getSeverityBadge = (severity) => {
    switch (severity) {
      case 'Critical':
        return <span className="badge badge-critical">Critical</span>;
      case 'High':
        return <span className="badge badge-high">High</span>;
      case 'Medium':
        return <span className="badge badge-medium">Medium</span>;
      case 'Low':
        return <span className="badge badge-low">Low</span>;
      default:
        return <span className="badge badge-observation">Observation</span>;
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>🔍</span> Actionable Findings & Corrective Action (CAPA)
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Turn audit findings, control failures, and risk assessments into actionable, tracked corrective tasks.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Record Finding
        </button>
      </div>

      <div style={{ display: 'flex', gap: 10, marginBottom: 16 }}>
        <button
          className={`btn ${activeSubView === 'findings' ? 'btn-primary' : 'btn-secondary'} btn-sm`}
          onClick={() => setActiveSubView('findings')}
        >
          Audit Findings ({findings.length})
        </button>
        <button
          className={`btn ${activeSubView === 'capas' ? 'btn-primary' : 'btn-secondary'} btn-sm`}
          onClick={() => setActiveSubView('capas')}
        >
          Corrective Actions / CAPA ({capas.length})
        </button>
      </div>

      {activeSubView === 'findings' && (
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>Finding Code</th>
                <th>Finding Title & Description</th>
                <th>Severity</th>
                <th>Source</th>
                <th>Owner / Dept</th>
                <th>Due Date</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {findings.length === 0 ? (
                <tr>
                  <td colSpan="8" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                    No actionable findings recorded.
                  </td>
                </tr>
              ) : (
                findings.map(f => (
                  <tr key={f.id}>
                    <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, fontSize: 12 }}>
                      {f.finding_code}
                    </td>
                    <td>
                      <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{f.title}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{f.recommendation}</div>
                    </td>
                    <td>{getSeverityBadge(f.severity)}</td>
                    <td><span className="badge badge-draft">{f.source}</span></td>
                    <td>
                      <div style={{ fontSize: 12.5, fontWeight: 500 }}>{f.owner}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{f.department}</div>
                    </td>
                    <td style={{ fontSize: 12, fontWeight: 600 }}>{f.due_date}</td>
                    <td>
                      <span className={`badge ${f.status === 'Closed' ? 'badge-approved' : f.status === 'In Remediation' ? 'badge-review' : 'badge-draft'}`}>
                        {f.status}
                      </span>
                    </td>
                    <td>
                      {f.status !== 'Closed' && (
                        <button
                          className="btn btn-success btn-sm"
                          onClick={() => {
                            setSelectedFinding(f);
                            setShowCloseModal(true);
                          }}
                        >
                          Verify & Close
                        </button>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {activeSubView === 'capas' && (
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>CAPA Code</th>
                <th>Action Title</th>
                <th>Source</th>
                <th>Owner</th>
                <th>Priority</th>
                <th>Due Date</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {capas.map(c => (
                <tr key={c.id}>
                  <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>{c.action_code}</td>
                  <td style={{ fontWeight: 600 }}>{c.title}</td>
                  <td><span className="badge badge-draft">{c.source_type}</span></td>
                  <td>{c.owner}</td>
                  <td>{getSeverityBadge(c.priority)}</td>
                  <td>{c.due_date}</td>
                  <td><span className="badge badge-approved">{c.status}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Record Finding Modal */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Record Actionable Finding</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateFinding}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Finding Code</label>
                    <input
                      className="form-control"
                      placeholder="e.g. FND-2026-08"
                      value={formData.finding_code}
                      onChange={e => setFormData({ ...formData, finding_code: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Severity Level *</label>
                    <select
                      className="form-control"
                      value={formData.severity}
                      onChange={e => setFormData({ ...formData, severity: e.target.value })}
                    >
                      <option value="Critical">Critical (Immediate Escalation)</option>
                      <option value="High">High (7-day SLA)</option>
                      <option value="Medium">Medium (30-day SLA)</option>
                      <option value="Low">Low</option>
                      <option value="Observation">Observation</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Finding Title *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. Missing Data Subject Consent Logs in Rural Survey Registries"
                    value={formData.title}
                    onChange={e => setFormData({ ...formData, title: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Finding Description & Evidence Observed *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Document the exact non-conformance, evidence reviewed, and operational context..."
                    value={formData.description}
                    onChange={e => setFormData({ ...formData, description: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Remediation Recommendation *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Specify the required corrective and preventive measures..."
                    value={formData.recommendation}
                    onChange={e => setFormData({ ...formData, recommendation: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Action Owner *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Lead Survey Engineer"
                      value={formData.owner}
                      onChange={e => setFormData({ ...formData, owner: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Due Date *</label>
                    <input
                      type="date"
                      className="form-control"
                      required
                      value={formData.due_date}
                      onChange={e => setFormData({ ...formData, due_date: e.target.value })}
                    />
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Log Finding</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Verify & Close Modal */}
      {showCloseModal && selectedFinding && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Verify & Close Finding {selectedFinding.finding_code}</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowCloseModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCloseFinding}>
              <div className="modal-body">
                <div style={{ background: '#f8fafc', padding: 12, borderRadius: 8, fontSize: 13, border: '1px solid var(--border-light)' }}>
                  <strong>Finding:</strong> {selectedFinding.title}
                </div>
                <div className="form-group">
                  <label>Closure Evidence & Verification Notes *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Detail verification tests, re-audit results, or evidence documents demonstrating effective remediation..."
                    value={closureEvidence}
                    onChange={e => setClosureEvidence(e.target.value)}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowCloseModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-success">Confirm Verification & Close</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
