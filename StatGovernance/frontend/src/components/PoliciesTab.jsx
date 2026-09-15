import React, { useState, useEffect } from 'react';
import DiscussionButton from './DiscussionButton';

export default function PoliciesTab({ apiBase, token, onRefreshDashboard }) {
  const [policies, setPolicies] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedPolicy, setSelectedPolicy] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [showVersions, setShowVersions] = useState(false);
  const [versions, setVersions] = useState([]);

  const [formData, setFormData] = useState({
    title: '',
    policy_number: '',
    category: 'Governance',
    scope: 'Organization-Wide',
    classification: 'Internal',
    department: 'Executive Board',
    summary: '',
    content: '',
    effective_date: '',
    review_date: ''
  });

  const fetchPolicies = async () => {
    try {
      setLoading(true);
      const res = await fetch(`${apiBase}/api/policies`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setPolicies(data);
        if (data.length > 0 && !selectedPolicy) {
          setSelectedPolicy(data[0]);
        }
      }
    } catch (err) {
      console.error('Failed to load policies:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPolicies();
  }, [apiBase, token]);

  const handleCreatePolicy = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/policies`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(formData)
      });
      if (res.ok) {
        setShowModal(false);
        setFormData({
          title: '', policy_number: '', category: 'Governance', scope: 'Organization-Wide',
          classification: 'Internal', department: 'Executive Board', summary: '', content: '',
          effective_date: '', review_date: ''
        });
        fetchPolicies();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating policy:', err);
    }
  };

  const handleApprove = async (id) => {
    try {
      const res = await fetch(`${apiBase}/api/policies/${id}/approve`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        fetchPolicies();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error approving policy:', err);
    }
  };

  const handlePublish = async (id) => {
    try {
      const res = await fetch(`${apiBase}/api/policies/${id}/publish`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        fetchPolicies();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error publishing policy:', err);
    }
  };

  const handleViewVersions = async (id) => {
    try {
      const res = await fetch(`${apiBase}/api/policies/${id}/versions`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setVersions(data);
        setShowVersions(true);
      }
    } catch (err) {
      console.error('Failed to load policy versions:', err);
    }
  };

  const getStatusBadge = (status) => {
    switch (status) {
      case 'Active':
      case 'Published':
        return <span className="badge badge-active">{status}</span>;
      case 'Approved':
        return <span className="badge badge-approved">{status}</span>;
      case 'Under Review':
      case 'Approval':
        return <span className="badge badge-review">{status}</span>;
      default:
        return <span className="badge badge-draft">{status}</span>;
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>📜</span> Institutional Policy Lifecycle Management
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Author, version, review, approve, enforce, and archive institutional policies.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + New Policy
        </button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 24 }}>
        {/* Policies List Table */}
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>Policy No</th>
                <th>Title & Category</th>
                <th>Owner / Dept</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {policies.length === 0 ? (
                <tr>
                  <td colSpan="5" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                    No institutional policies found.
                  </td>
                </tr>
              ) : (
                policies.map(p => (
                  <tr
                    key={p.id}
                    onClick={() => setSelectedPolicy(p)}
                    style={{
                      cursor: 'pointer',
                      background: selectedPolicy?.id === p.id ? '#f1f5f9' : 'transparent'
                    }}
                  >
                    <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, fontSize: 12 }}>
                      {p.policy_number}
                    </td>
                    <td>
                      <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{p.title}</div>
                      <div style={{ fontSize: 11.5, color: 'var(--text-muted)' }}>{p.category} • v{p.version}</div>
                    </td>
                    <td>
                      <div style={{ fontSize: 12.5, fontWeight: 500 }}>{p.owner}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{p.department}</div>
                    </td>
                    <td>{getStatusBadge(p.status)}</td>
                    <td>
                      <div style={{ display: 'flex', gap: 6 }}>
                        {p.status === 'Draft' && (
                          <button
                            className="btn btn-secondary btn-sm"
                            onClick={(e) => { e.stopPropagation(); handleApprove(p.id); }}
                            title="Submit for Approval"
                          >
                            Approve
                          </button>
                        )}
                        {p.status === 'Approved' && (
                          <button
                            className="btn btn-success btn-sm"
                            onClick={(e) => { e.stopPropagation(); handlePublish(p.id); }}
                            title="Publish Policy"
                          >
                            Publish
                          </button>
                        )}
                        <button
                          className="btn btn-secondary btn-sm"
                          onClick={(e) => { e.stopPropagation(); handleViewVersions(p.id); }}
                          title="Version History"
                        >
                          vHistory
                        </button>
                        <DiscussionButton apiBase={apiBase} token={token} objectType="policy" objectId={p.id} name={p.title} />
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Selected Policy Detail Drawer */}
        <div className="glass-panel" style={{ padding: 22, display: 'flex', flexDirection: 'column' }}>
          {selectedPolicy ? (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
                <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, fontWeight: 700, color: 'var(--primary-color)' }}>
                  {selectedPolicy.policy_number}
                </span>
                {getStatusBadge(selectedPolicy.status)}
              </div>

              <h3 style={{ fontSize: 18, fontWeight: 800, marginBottom: 8, color: 'var(--text-dark)' }}>
                {selectedPolicy.title}
              </h3>

              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, marginBottom: 16 }}>
                <span className="badge badge-draft">Category: {selectedPolicy.category}</span>
                <span className="badge badge-draft">Scope: {selectedPolicy.scope}</span>
                <span className="badge badge-draft">Version: {selectedPolicy.version}</span>
                <span className="badge badge-draft">Classification: {selectedPolicy.classification}</span>
              </div>

              <div style={{ background: '#f8fafc', padding: 14, borderRadius: 8, border: '1px solid var(--border-light)', marginBottom: 16 }}>
                <div style={{ fontSize: 11, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 4 }}>
                  Executive Summary
                </div>
                <div style={{ fontSize: 13, color: 'var(--text-dark)' }}>
                  {selectedPolicy.summary || 'No summary provided.'}
                </div>
              </div>

              <div style={{ marginBottom: 16 }}>
                <div style={{ fontSize: 11, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 6 }}>
                  Policy Content & Governance Provisions
                </div>
                <div style={{
                  background: '#ffffff',
                  border: '1px solid var(--border-light)',
                  borderRadius: 8,
                  padding: 14,
                  fontSize: 13,
                  maxHeight: 240,
                  overflowY: 'auto',
                  whiteSpace: 'pre-wrap',
                  fontFamily: 'inherit',
                  lineHeight: 1.6
                }}>
                  {selectedPolicy.content}
                </div>
              </div>

              <div style={{ fontSize: 12, color: 'var(--text-muted)', borderTop: '1px solid var(--border-light)', paddingTop: 12, display: 'flex', justifyContent: 'space-between' }}>
                <span>Owner: <strong>{selectedPolicy.owner}</strong></span>
                <span>Review Date: <strong>{selectedPolicy.review_date || 'None'}</strong></span>
              </div>
            </div>
          ) : (
            <div style={{ textAlign: 'center', padding: 40, color: 'var(--text-muted)' }}>
              Select a policy to view governance details.
            </div>
          )}
        </div>
      </div>

      {/* New Policy Modal */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Draft New Institutional Policy</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreatePolicy}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Policy Number / Code</label>
                    <input
                      className="form-control"
                      placeholder="e.g. POL-SEC-2026-02"
                      value={formData.policy_number}
                      onChange={e => setFormData({ ...formData, policy_number: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Category</label>
                    <select
                      className="form-control"
                      value={formData.category}
                      onChange={e => setFormData({ ...formData, category: e.target.value })}
                    >
                      <option value="Governance">Governance</option>
                      <option value="Data Governance">Data Governance</option>
                      <option value="Information Security">Information Security</option>
                      <option value="Operations">Operations</option>
                      <option value="Ethics & Compliance">Ethics & Compliance</option>
                      <option value="Financial">Financial</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Policy Title *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. Statistical Microdata Anonymization Policy"
                    value={formData.title}
                    onChange={e => setFormData({ ...formData, title: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Department / Authority *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Department of Statistical Standards"
                      value={formData.department}
                      onChange={e => setFormData({ ...formData, department: e.target.value })}
                    />
                  </div>
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
                      <option value="Restricted">Restricted</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Executive Summary</label>
                  <textarea
                    className="form-control"
                    placeholder="Summarize the purpose and institutional rationale..."
                    value={formData.summary}
                    onChange={e => setFormData({ ...formData, summary: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Full Policy Content & Provisions *</label>
                  <textarea
                    className="form-control"
                    required
                    style={{ minHeight: 120 }}
                    placeholder="Enter full markdown policy text, sections, and controls..."
                    value={formData.content}
                    onChange={e => setFormData({ ...formData, content: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Effective Date</label>
                    <input
                      type="date"
                      className="form-control"
                      value={formData.effective_date}
                      onChange={e => setFormData({ ...formData, effective_date: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Mandatory Review Date</label>
                    <input
                      type="date"
                      className="form-control"
                      value={formData.review_date}
                      onChange={e => setFormData({ ...formData, review_date: e.target.value })}
                    />
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Policy</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Version History Modal */}
      {showVersions && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Policy Version History & Audit Trail</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowVersions(false)}>✕</button>
            </div>
            <div className="modal-body">
              {versions.length === 0 ? (
                <div style={{ color: 'var(--text-muted)', textAlign: 'center', padding: 20 }}>No version records found.</div>
              ) : (
                versions.map(v => (
                  <div key={v.id} style={{ borderBottom: '1px solid var(--border-light)', paddingBottom: 12, marginBottom: 12 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                      <span style={{ fontWeight: 700, fontSize: 14 }}>Version {v.version}</span>
                      <span className="badge badge-approved">{v.status}</span>
                    </div>
                    <div style={{ fontSize: 12.5, color: 'var(--text-dark)' }}>{v.change_summary}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>
                      Changed by: <strong>{v.changed_by}</strong> on {new Date(v.created_time).toLocaleString()}
                    </div>
                  </div>
                ))
              )}
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowVersions(false)}>Close</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
