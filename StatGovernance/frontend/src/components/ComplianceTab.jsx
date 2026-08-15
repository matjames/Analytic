import React, { useState, useEffect } from 'react';

export default function ComplianceTab({ apiBase, token, onRefreshDashboard }) {
  const [obligations, setObligations] = useState([]);
  const [regulations, setRegulations] = useState([]);
  const [assessments, setAssessments] = useState([]);
  const [activeSubView, setActiveSubView] = useState('obligations'); // obligations, regulations, assessments
  const [showModal, setShowModal] = useState(false);
  const [showAssessmentModal, setShowAssessmentModal] = useState(false);

  const [obligationForm, setObligationForm] = useState({
    obligation_code: '',
    regulation_id: '',
    requirement: '',
    applicable_scope: 'All Systems',
    responsible_dept: 'Compliance & Legal',
    responsible_person: '',
    frequency: 'Continuous',
    compliance_status: 'Compliant',
    violation_risk: 'High',
    deadline: '',
    evidence_requirement: ''
  });

  const [assessmentForm, setAssessmentForm] = useState({
    assessment_code: '',
    title: '',
    scope: 'Comprehensive Annual Governance Assessment',
    assessor: '',
    recommendations: '',
    start_date: '',
    completion_date: ''
  });

  const fetchComplianceData = async () => {
    try {
      const [oblRes, regRes, asmRes] = await Promise.all([
        fetch(`${apiBase}/api/compliance/obligations`, { headers: { 'Authorization': `Bearer ${token}` } }),
        fetch(`${apiBase}/api/regulations`, { headers: { 'Authorization': `Bearer ${token}` } }),
        fetch(`${apiBase}/api/compliance/assessments`, { headers: { 'Authorization': `Bearer ${token}` } })
      ]);

      if (oblRes.ok) setObligations(await oblRes.json());
      if (regRes.ok) setRegulations(await regRes.json());
      if (asmRes.ok) setAssessments(await asmRes.json());
    } catch (err) {
      console.error('Error loading compliance data:', err);
    }
  };

  useEffect(() => {
    fetchComplianceData();
  }, [apiBase, token]);

  const handleCreateObligation = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/compliance/obligations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(obligationForm)
      });
      if (res.ok) {
        setShowModal(false);
        fetchComplianceData();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating obligation:', err);
    }
  };

  const handleCreateAssessment = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/compliance/assessments`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(assessmentForm)
      });
      if (res.ok) {
        setShowAssessmentModal(false);
        fetchComplianceData();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating assessment:', err);
    }
  };

  const getStatusBadge = (status) => {
    switch (status) {
      case 'Compliant':
        return <span className="badge badge-compliant">Compliant</span>;
      case 'Partially Compliant':
      case 'Under Review':
        return <span className="badge badge-partially">{status}</span>;
      case 'Non-Compliant':
        return <span className="badge badge-non-compliant">Non-Compliant</span>;
      default:
        return <span className="badge badge-draft">{status || 'Not Assessed'}</span>;
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>⚖️</span> Regulatory Obligations & Compliance Register
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Track statutory obligations, regulatory authorities, compliance assessments, and audit evidence.
          </p>
        </div>
        <div style={{ display: 'flex', gap: 10 }}>
          <button className="btn btn-secondary" onClick={() => setShowAssessmentModal(true)}>
            + Run Assessment
          </button>
          <button className="btn btn-primary" onClick={() => setShowModal(true)}>
            + Add Obligation
          </button>
        </div>
      </div>

      {/* Sub-view navigation tabs */}
      <div style={{ display: 'flex', gap: 10, marginBottom: 16 }}>
        <button
          className={`btn ${activeSubView === 'obligations' ? 'btn-primary' : 'btn-secondary'} btn-sm`}
          onClick={() => setActiveSubView('obligations')}
        >
          Compliance Obligations ({obligations.length})
        </button>
        <button
          className={`btn ${activeSubView === 'regulations' ? 'btn-primary' : 'btn-secondary'} btn-sm`}
          onClick={() => setActiveSubView('regulations')}
        >
          Regulations & Authorities ({regulations.length})
        </button>
        <button
          className={`btn ${activeSubView === 'assessments' ? 'btn-primary' : 'btn-secondary'} btn-sm`}
          onClick={() => setActiveSubView('assessments')}
        >
          Compliance Assessments ({assessments.length})
        </button>
      </div>

      {/* 1. Obligations Table View */}
      {activeSubView === 'obligations' && (
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>Code</th>
                <th>Requirement</th>
                <th>Regulation</th>
                <th>Responsible Person / Dept</th>
                <th>Status</th>
                <th>Violation Risk</th>
                <th>Deadline</th>
              </tr>
            </thead>
            <tbody>
              {obligations.length === 0 ? (
                <tr>
                  <td colSpan="7" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                    No compliance obligations registered.
                  </td>
                </tr>
              ) : (
                obligations.map(o => (
                  <tr key={o.id}>
                    <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, fontSize: 12 }}>
                      {o.obligation_code}
                    </td>
                    <td>
                      <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{o.requirement}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Scope: {o.applicable_scope}</div>
                    </td>
                    <td>
                      <span className="badge badge-draft">{o.regulation_name || 'Regulation'}</span>
                    </td>
                    <td>
                      <div style={{ fontSize: 12.5, fontWeight: 500 }}>{o.responsible_person}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{o.responsible_dept}</div>
                    </td>
                    <td>{getStatusBadge(o.compliance_status)}</td>
                    <td>
                      <span style={{
                        fontSize: 11.5,
                        fontWeight: 700,
                        color: o.violation_risk === 'Critical' || o.violation_risk === 'High' ? 'var(--accent-ruby)' : 'inherit'
                      }}>
                        {o.violation_risk}
                      </span>
                    </td>
                    <td style={{ fontSize: 12, fontWeight: 500 }}>{o.deadline || 'Continuous'}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* 2. Regulations View */}
      {activeSubView === 'regulations' && (
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>Code</th>
                <th>Regulation Name</th>
                <th>Authority</th>
                <th>Category</th>
                <th>Jurisdiction</th>
                <th>Effective Date</th>
              </tr>
            </thead>
            <tbody>
              {regulations.map(r => (
                <tr key={r.id}>
                  <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>{r.code}</td>
                  <td style={{ fontWeight: 600 }}>{r.name}</td>
                  <td>{r.regulatory_authority}</td>
                  <td><span className="badge badge-draft">{r.category}</span></td>
                  <td>{r.jurisdiction}</td>
                  <td>{r.effective_date || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* 3. Assessments View */}
      {activeSubView === 'assessments' && (
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>Code</th>
                <th>Assessment Title</th>
                <th>Assessor</th>
                <th>Status</th>
                <th>Score</th>
                <th>Findings</th>
                <th>Period</th>
              </tr>
            </thead>
            <tbody>
              {assessments.map(a => (
                <tr key={a.id}>
                  <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>{a.assessment_code}</td>
                  <td style={{ fontWeight: 600 }}>{a.title}</td>
                  <td>{a.assessor}</td>
                  <td><span className="badge badge-review">{a.status}</span></td>
                  <td style={{ fontWeight: 700 }}>{a.score_percentage}%</td>
                  <td>{a.findings_count} findings</td>
                  <td style={{ fontSize: 12 }}>{a.start_date} to {a.completion_date || 'Ongoing'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Add Obligation Modal */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Register Compliance Obligation</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateObligation}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Obligation Code</label>
                    <input
                      className="form-control"
                      placeholder="e.g. OBL-STAT-2026"
                      value={obligationForm.obligation_code}
                      onChange={e => setObligationForm({ ...obligationForm, obligation_code: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Applicable Regulation *</label>
                    <select
                      className="form-control"
                      required
                      value={obligationForm.regulation_id}
                      onChange={e => setObligationForm({ ...obligationForm, regulation_id: e.target.value })}
                    >
                      <option value="">Select Regulation...</option>
                      {regulations.map(r => (
                        <option key={r.id} value={r.id}>{r.name} ({r.code})</option>
                      ))}
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Obligation Requirement *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Describe specific statutory requirement or mandate..."
                    value={obligationForm.requirement}
                    onChange={e => setObligationForm({ ...obligationForm, requirement: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Responsible Department *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Data Governance Unit"
                      value={obligationForm.responsible_dept}
                      onChange={e => setObligationForm({ ...obligationForm, responsible_dept: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Responsible Officer *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Compliance Officer"
                      value={obligationForm.responsible_person}
                      onChange={e => setObligationForm({ ...obligationForm, responsible_person: e.target.value })}
                    />
                  </div>
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Compliance Status</label>
                    <select
                      className="form-control"
                      value={obligationForm.compliance_status}
                      onChange={e => setObligationForm({ ...obligationForm, compliance_status: e.target.value })}
                    >
                      <option value="Compliant">Compliant</option>
                      <option value="Partially Compliant">Partially Compliant</option>
                      <option value="Non-Compliant">Non-Compliant</option>
                      <option value="Under Review">Under Review</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Violation Risk</label>
                    <select
                      className="form-control"
                      value={obligationForm.violation_risk}
                      onChange={e => setObligationForm({ ...obligationForm, violation_risk: e.target.value })}
                    >
                      <option value="Critical">Critical</option>
                      <option value="High">High</option>
                      <option value="Medium">Medium</option>
                      <option value="Low">Low</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Evidence Requirement</label>
                  <input
                    className="form-control"
                    placeholder="e.g. Signed Data Protection Officer Audit Certificate"
                    value={obligationForm.evidence_requirement}
                    onChange={e => setObligationForm({ ...obligationForm, evidence_requirement: e.target.value })}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Obligation</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Assessment Modal */}
      {showAssessmentModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Initiate Compliance Assessment</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowAssessmentModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateAssessment}>
              <div className="modal-body">
                <div className="form-group">
                  <label>Assessment Title *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. Q3 Statutory Statistical Standards Assessment"
                    value={assessmentForm.title}
                    onChange={e => setAssessmentForm({ ...assessmentForm, title: e.target.value })}
                  />
                </div>
                <div className="form-group">
                  <label>Scope & System Coverage *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Detail the repositories, surveys, and applications covered..."
                    value={assessmentForm.scope}
                    onChange={e => setAssessmentForm({ ...assessmentForm, scope: e.target.value })}
                  />
                </div>
                <div className="form-row">
                  <div className="form-group">
                    <label>Lead Assessor</label>
                    <input
                      className="form-control"
                      placeholder="e.g. Senior Compliance Lead"
                      value={assessmentForm.assessor}
                      onChange={e => setAssessmentForm({ ...assessmentForm, assessor: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Start Date</label>
                    <input
                      type="date"
                      className="form-control"
                      value={assessmentForm.start_date}
                      onChange={e => setAssessmentForm({ ...assessmentForm, start_date: e.target.value })}
                    />
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowAssessmentModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Start Assessment</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
