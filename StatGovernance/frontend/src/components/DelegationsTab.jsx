import React, { useState, useEffect } from 'react';

export default function DelegationsTab({ apiBase, token, onRefreshDashboard }) {
  const [delegations, setDelegations] = useState([]);
  const [showModal, setShowModal] = useState(false);

  const [formData, setFormData] = useState({
    delegate_name: '',
    delegate_id: '',
    role_scope: 'Acting Officer',
    scope_details: '',
    start_date: '',
    end_date: '',
    reason: ''
  });

  const fetchDelegations = async () => {
    try {
      const res = await fetch(`${apiBase}/api/delegations`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        setDelegations(await res.json());
      }
    } catch (err) {
      console.error('Error loading delegations:', err);
    }
  };

  useEffect(() => {
    fetchDelegations();
  }, [apiBase, token]);

  const handleCreateDelegation = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/delegations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({
          ...formData,
          start_date: new Date(formData.start_date).toISOString(),
          end_date: new Date(formData.end_date).toISOString()
        })
      });
      if (res.ok) {
        setShowModal(false);
        fetchDelegations();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating delegation:', err);
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>👥</span> Delegation of Governance Authority
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Manage temporary delegations for Acting Officers, Delegated Approvers, and Temporary Reviewers with automatic expiration.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Grant Delegation
        </button>
      </div>

      <div className="glass-panel table-container">
        <table className="gov-table">
          <thead>
            <tr>
              <th>Role Scope</th>
              <th>Delegator</th>
              <th>Delegate (Acting Authority)</th>
              <th>Scope & Details</th>
              <th>Validity Window</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {delegations.length === 0 ? (
              <tr>
                <td colSpan="6" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                  No delegations active or on record.
                </td>
              </tr>
            ) : (
              delegations.map(d => (
                <tr key={d.id}>
                  <td>
                    <span className="badge badge-approved" style={{ fontWeight: 700 }}>
                      {d.role_scope}
                    </span>
                  </td>
                  <td>{d.delegator_name}</td>
                  <td>
                    <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{d.delegate_name}</div>
                  </td>
                  <td>
                    <div style={{ fontSize: 12.5, color: 'var(--text-secondary)' }}>{d.scope_details}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Reason: {d.reason}</div>
                  </td>
                  <td style={{ fontSize: 12 }}>
                    {new Date(d.start_date).toLocaleDateString()} to {new Date(d.end_date).toLocaleDateString()}
                  </td>
                  <td>
                    <span className={`badge ${d.status === 'Active' ? 'badge-active' : 'badge-draft'}`}>
                      {d.status}
                    </span>
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
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Grant Temporary Governance Delegation</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateDelegation}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Delegate Name *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Dr. Florence Kigozi"
                      value={formData.delegate_name}
                      onChange={e => setFormData({ ...formData, delegate_name: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Delegation Scope *</label>
                    <select
                      className="form-control"
                      value={formData.role_scope}
                      onChange={e => setFormData({ ...formData, role_scope: e.target.value })}
                    >
                      <option value="Acting Officer">Acting Officer</option>
                      <option value="Delegated Approver">Delegated Approver</option>
                      <option value="Temporary Reviewer">Temporary Reviewer</option>
                      <option value="Audit Delegate">Audit Delegate</option>
                      <option value="Risk Owner Delegate">Risk Owner Delegate</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Scope & Authority Constraints *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Specify which policies, approvals, or departments this delegation covers..."
                    value={formData.scope_details}
                    onChange={e => setFormData({ ...formData, scope_details: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Institutional Reason *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. Annual leave of substantive Data Protection Officer"
                    value={formData.reason}
                    onChange={e => setFormData({ ...formData, reason: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Start Date & Time *</label>
                    <input
                      type="datetime-local"
                      className="form-control"
                      required
                      value={formData.start_date}
                      onChange={e => setFormData({ ...formData, start_date: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Expiration Date & Time *</label>
                    <input
                      type="datetime-local"
                      className="form-control"
                      required
                      value={formData.end_date}
                      onChange={e => setFormData({ ...formData, end_date: e.target.value })}
                    />
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Authorize Delegation</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
