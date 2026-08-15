import React, { useState, useEffect } from 'react';

export default function EvidenceTab({ apiBase, token, onRefreshDashboard }) {
  const [evidenceRecords, setEvidenceRecords] = useState([]);
  const [showModal, setShowModal] = useState(false);

  const [formData, setFormData] = useState({
    title: '',
    description: '',
    category: 'Audit Report',
    source_application: 'StatGovernance',
    related_entity_type: 'policy',
    related_entity_id: 'pol-001',
    file_name: '',
    file_url: '',
    classification: 'Internal'
  });

  const fetchEvidence = async () => {
    try {
      const res = await fetch(`${apiBase}/api/evidence`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        setEvidenceRecords(await res.json());
      }
    } catch (err) {
      console.error('Error loading evidence:', err);
    }
  };

  useEffect(() => {
    fetchEvidence();
  }, [apiBase, token]);

  const handleUploadEvidence = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/evidence`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(formData)
      });
      if (res.ok) {
        setShowModal(false);
        fetchEvidence();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error uploading evidence:', err);
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>📦</span> Institutional Evidence Vault
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Cryptographically verified audit evidence, test certificates, minutes, and data lineage records.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Record Evidence
        </button>
      </div>

      <div className="glass-panel table-container">
        <table className="gov-table">
          <thead>
            <tr>
              <th>Title</th>
              <th>Category</th>
              <th>Source Application</th>
              <th>Linked Entity</th>
              <th>SHA256 Checksum</th>
              <th>Uploader</th>
              <th>Timestamp</th>
            </tr>
          </thead>
          <tbody>
            {evidenceRecords.length === 0 ? (
              <tr>
                <td colSpan="7" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                  No evidence records registered.
                </td>
              </tr>
            ) : (
              evidenceRecords.map(e => (
                <tr key={e.id}>
                  <td>
                    <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{e.title}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{e.file_name}</div>
                  </td>
                  <td><span className="badge badge-draft">{e.category}</span></td>
                  <td><span className="badge badge-review">{e.source_application}</span></td>
                  <td>
                    <span style={{ fontSize: 12, fontWeight: 600 }}>{e.related_entity_type}: {e.related_entity_id}</span>
                  </td>
                  <td style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-secondary)' }}>
                    {e.checksum_sha256 ? `${e.checksum_sha256.substring(0, 14)}...` : '—'}
                  </td>
                  <td>{e.uploaded_by}</td>
                  <td style={{ fontSize: 12 }}>{new Date(e.created_time).toLocaleString()}</td>
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
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Record Verification Evidence</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleUploadEvidence}>
              <div className="modal-body">
                <div className="form-group">
                  <label>Evidence Title *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. 2026 ISO 27001 Penetration Test Report Certificate"
                    value={formData.title}
                    onChange={e => setFormData({ ...formData, title: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Category</label>
                    <select
                      className="form-control"
                      value={formData.category}
                      onChange={e => setFormData({ ...formData, category: e.target.value })}
                    >
                      <option value="Audit Report">Audit Report</option>
                      <option value="Test Result">Test Result</option>
                      <option value="Approval Log">Approval Log</option>
                      <option value="Meeting Minutes">Meeting Minutes</option>
                      <option value="Regulatory Certificate">Regulatory Certificate</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Source Application</label>
                    <select
                      className="form-control"
                      value={formData.source_application}
                      onChange={e => setFormData({ ...formData, source_application: e.target.value })}
                    >
                      <option value="StatGovernance">StatGovernance</option>
                      <option value="PMS">PMS</option>
                      <option value="RMS">RMS</option>
                      <option value="StatCollect">StatCollect</option>
                      <option value="HelpDesk">HelpDesk</option>
                    </select>
                  </div>
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Linked Entity Type *</label>
                    <select
                      className="form-control"
                      value={formData.related_entity_type}
                      onChange={e => setFormData({ ...formData, related_entity_type: e.target.value })}
                    >
                      <option value="policy">Policy</option>
                      <option value="risk">Risk</option>
                      <option value="control">Control</option>
                      <option value="audit">Audit</option>
                      <option value="finding">Finding</option>
                      <option value="decision">Decision</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Linked Entity ID *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. pol-001 or ctl-001"
                      value={formData.related_entity_id}
                      onChange={e => setFormData({ ...formData, related_entity_id: e.target.value })}
                    />
                  </div>
                </div>

                <div className="form-group">
                  <label>File Reference / URL</label>
                  <input
                    className="form-control"
                    placeholder="e.g. /files/enterprise/audit-report-2026.pdf"
                    value={formData.file_url}
                    onChange={e => setFormData({ ...formData, file_url: e.target.value })}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Evidence</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
