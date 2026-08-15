import React, { useState, useEffect } from 'react';

export default function DecisionsTab({ apiBase, token, onRefreshDashboard }) {
  const [decisions, setDecisions] = useState([]);
  const [showModal, setShowModal] = useState(false);

  const [formData, setFormData] = useState({
    decision_code: '',
    title: '',
    context: '',
    risks_evaluated: '',
    recommendation: '',
    final_decision: '',
    decision_maker: '',
    status: 'Approved',
    ai_advisory: false
  });

  const fetchDecisions = async () => {
    try {
      const res = await fetch(`${apiBase}/api/decisions`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        setDecisions(await res.json());
      }
    } catch (err) {
      console.error('Error loading decisions:', err);
    }
  };

  useEffect(() => {
    fetchDecisions();
  }, [apiBase, token]);

  const handleCreateDecision = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/decisions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(formData)
      });
      if (res.ok) {
        setShowModal(false);
        fetchDecisions();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating decision:', err);
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>📜</span> Institutional Decision Governance Register
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Maintain auditable records of executive determinations, evaluated options, and human authorization signatures.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Record Decision
        </button>
      </div>

      <div className="glass-panel table-container">
        <table className="gov-table">
          <thead>
            <tr>
              <th>Decision Code</th>
              <th>Decision Title & Context</th>
              <th>Determined Resolution</th>
              <th>Authority / Sign-off</th>
              <th>Type</th>
              <th>Status</th>
              <th>Date</th>
            </tr>
          </thead>
          <tbody>
            {decisions.length === 0 ? (
              <tr>
                <td colSpan="7" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                  No institutional decision records logged.
                </td>
              </tr>
            ) : (
              decisions.map(d => (
                <tr key={d.id}>
                  <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, fontSize: 12 }}>
                    {d.decision_code}
                  </td>
                  <td>
                    <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{d.title}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{d.context}</div>
                  </td>
                  <td>
                    <div style={{ fontSize: 12.5, color: 'var(--text-dark)', maxWidth: 280, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {d.final_decision}
                    </div>
                  </td>
                  <td>
                    <div style={{ fontSize: 12.5, fontWeight: 500 }}>{d.decision_maker}</div>
                    <div style={{ fontSize: 10.5, color: 'var(--accent-emerald)', fontWeight: 700 }}>✓ Human Authorized</div>
                  </td>
                  <td>
                    {d.ai_advisory ? (
                      <span className="badge badge-draft">AI Supported</span>
                    ) : (
                      <span className="badge badge-approved">Institutional</span>
                    )}
                  </td>
                  <td><span className="badge badge-approved">{d.status}</span></td>
                  <td style={{ fontSize: 12 }}>{d.effective_date || new Date(d.created_time).toLocaleDateString()}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Record Decision Modal */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Record Institutional Decision</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateDecision}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Decision Code</label>
                    <input
                      className="form-control"
                      placeholder="e.g. DEC-2026-004"
                      value={formData.decision_code}
                      onChange={e => setFormData({ ...formData, decision_code: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Authorized Decision Maker *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Director General / Management Committee"
                      value={formData.decision_maker}
                      onChange={e => setFormData({ ...formData, decision_maker: e.target.value })}
                    />
                  </div>
                </div>

                <div className="form-group">
                  <label>Decision Title *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. Mandatory Two-Factor Authentication Policy Approval"
                    value={formData.title}
                    onChange={e => setFormData({ ...formData, title: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Context & Problem Statement *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Explain background, operational necessity, and institutional context..."
                    value={formData.context}
                    onChange={e => setFormData({ ...formData, context: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Risks & Implications Evaluated</label>
                  <textarea
                    className="form-control"
                    placeholder="Document evaluated risks, operational trade-offs, and compliance impact..."
                    value={formData.risks_evaluated}
                    onChange={e => setFormData({ ...formData, risks_evaluated: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Formal Determination / Final Decision *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="State the binding institutional ruling and authorized mandates..."
                    value={formData.final_decision}
                    onChange={e => setFormData({ ...formData, final_decision: e.target.value })}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Authorize & Record Decision</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
