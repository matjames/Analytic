import React, { useState, useEffect } from 'react';

export default function AuditsTab({ apiBase, token, onRefreshDashboard }) {
  const [audits, setAudits] = useState([]);
  const [showModal, setShowModal] = useState(false);

  const [formData, setFormData] = useState({
    audit_code: '',
    title: '',
    audit_type: 'Internal Audit',
    scope: '',
    objectives: '',
    lead_auditor: '',
    department: 'All Operational Units',
    start_date: '',
    end_date: '',
    status: 'Planned',
    summary: ''
  });

  const fetchAudits = async () => {
    try {
      const res = await fetch(`${apiBase}/api/audits`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        setAudits(await res.json());
      }
    } catch (err) {
      console.error('Error loading audits:', err);
    }
  };

  useEffect(() => {
    fetchAudits();
  }, [apiBase, token]);

  const handleCreateAudit = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/audits`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(formData)
      });
      if (res.ok) {
        setShowModal(false);
        fetchAudits();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating audit:', err);
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>📋</span> Institutional Audit Programme & Engagements
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Manage scheduled audits, independent evaluations, compliance scope, and lead auditor assignments.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Schedule Audit
        </button>
      </div>

      <div className="glass-panel table-container">
        <table className="gov-table">
          <thead>
            <tr>
              <th>Audit Code</th>
              <th>Engagement Title</th>
              <th>Type</th>
              <th>Lead Auditor / Scope</th>
              <th>Status</th>
              <th>Findings</th>
              <th>Schedule</th>
            </tr>
          </thead>
          <tbody>
            {audits.length === 0 ? (
              <tr>
                <td colSpan="7" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                  No audit engagements scheduled.
                </td>
              </tr>
            ) : (
              audits.map(a => (
                <tr key={a.id}>
                  <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, fontSize: 12 }}>
                    {a.audit_code}
                  </td>
                  <td>
                    <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{a.title}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Dept: {a.department}</div>
                  </td>
                  <td><span className="badge badge-draft">{a.audit_type}</span></td>
                  <td>
                    <div style={{ fontSize: 12.5, fontWeight: 500 }}>{a.lead_auditor}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)', maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {a.scope}
                    </div>
                  </td>
                  <td>
                    <span className={`badge ${a.status === 'In Progress' ? 'badge-review' : a.status === 'Closed' ? 'badge-approved' : 'badge-draft'}`}>
                      {a.status}
                    </span>
                  </td>
                  <td>
                    <span style={{ fontWeight: 700, color: a.critical_findings > 0 ? 'var(--accent-ruby)' : 'inherit' }}>
                      {a.findings_count} findings
                    </span>
                    {a.critical_findings > 0 && <span style={{ fontSize: 11, color: 'var(--accent-ruby)', display: 'block' }}>({a.critical_findings} critical)</span>}
                  </td>
                  <td style={{ fontSize: 12 }}>
                    {a.start_date || 'TBD'} to {a.end_date || 'TBD'}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {showModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Schedule Audit Engagement</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateAudit}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Audit Code</label>
                    <input
                      className="form-control"
                      placeholder="e.g. AUD-2026-Q3"
                      value={formData.audit_code}
                      onChange={e => setFormData({ ...formData, audit_code: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Audit Type</label>
                    <select
                      className="form-control"
                      value={formData.audit_type}
                      onChange={e => setFormData({ ...formData, audit_type: e.target.value })}
                    >
                      <option value="Internal Audit">Internal Audit</option>
                      <option value="External Compliance">External Compliance</option>
                      <option value="ISO/IEC 27001 Security">ISO/IEC 27001 Security</option>
                      <option value="Data Protection Audit">Data Protection Audit</option>
                      <option value="Financial & Operational">Financial & Operational</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Audit Engagement Title *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. Annual Microdata Security & Consent Protocol Audit"
                    value={formData.title}
                    onChange={e => setFormData({ ...formData, title: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Audit Scope & Coverage *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Specify repositories, infrastructure, and departments in scope..."
                    value={formData.scope}
                    onChange={e => setFormData({ ...formData, scope: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Lead Auditor *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Chief Internal Auditor"
                      value={formData.lead_auditor}
                      onChange={e => setFormData({ ...formData, lead_auditor: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Target Department</label>
                    <input
                      className="form-control"
                      value={formData.department}
                      onChange={e => setFormData({ ...formData, department: e.target.value })}
                    />
                  </div>
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Start Date</label>
                    <input
                      type="date"
                      className="form-control"
                      value={formData.start_date}
                      onChange={e => setFormData({ ...formData, start_date: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Target Completion Date</label>
                    <input
                      type="date"
                      className="form-control"
                      value={formData.end_date}
                      onChange={e => setFormData({ ...formData, end_date: e.target.value })}
                    />
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Schedule Engagement</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
