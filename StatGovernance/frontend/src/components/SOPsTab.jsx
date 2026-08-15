import React, { useState, useEffect } from 'react';

export default function SOPsTab({ apiBase, token, onRefreshDashboard }) {
  const [sops, setSops] = useState([]);
  const [selectedSOP, setSelectedSOP] = useState(null);
  const [showModal, setShowModal] = useState(false);

  const [formData, setFormData] = useState({
    sop_number: '',
    title: '',
    process: 'Statistical Data Pipeline',
    purpose: '',
    scope: 'All Operational Field Units',
    responsibilities: '',
    review_frequency: 'Annual',
    department: 'Statistical Methodology',
    procedure_steps: [
      { step_number: 1, title: 'Input Data Validation', description: 'Run checksum integrity check on raw survey batch.', role: 'Data Engineer' },
      { step_number: 2, title: 'Anonymization Verification', description: 'Apply k-anonymity protocol on PII fields.', role: 'Privacy Steward' }
    ]
  });

  const fetchSOPs = async () => {
    try {
      const res = await fetch(`${apiBase}/api/sops`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setSops(data);
        if (data.length > 0 && !selectedSOP) {
          setSelectedSOP(data[0]);
        }
      }
    } catch (err) {
      console.error('Error loading SOPs:', err);
    }
  };

  useEffect(() => {
    fetchSOPs();
  }, [apiBase, token]);

  const handleCreateSOP = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/sops`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(formData)
      });
      if (res.ok) {
        setShowModal(false);
        fetchSOPs();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating SOP:', err);
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>📑</span> Standard Operating Procedures (SOPs)
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Operationalize policies with step-by-step procedures, role accountabilities, and required evidence artifacts.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Author SOP
        </button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1.3fr 1fr', gap: 24 }}>
        {/* SOPs Table */}
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>SOP No</th>
                <th>Title & Process</th>
                <th>Department</th>
                <th>Frequency</th>
                <th>Version</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {sops.length === 0 ? (
                <tr>
                  <td colSpan="6" style={{ textAlign: 'center', padding: 24, color: 'var(--text-muted)' }}>
                    No standard operating procedures registered.
                  </td>
                </tr>
              ) : (
                sops.map(s => (
                  <tr
                    key={s.id}
                    onClick={() => setSelectedSOP(s)}
                    style={{
                      cursor: 'pointer',
                      background: selectedSOP?.id === s.id ? '#f1f5f9' : 'transparent'
                    }}
                  >
                    <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>{s.sop_number}</td>
                    <td>
                      <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{s.title}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Process: {s.process}</div>
                    </td>
                    <td>{s.department}</td>
                    <td>{s.review_frequency}</td>
                    <td>v{s.version}</td>
                    <td><span className="badge badge-active">{s.status}</span></td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Selected SOP Inspection */}
        <div className="glass-panel" style={{ padding: 22 }}>
          {selectedSOP ? (
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, fontWeight: 700, color: 'var(--primary-color)' }}>
                  {selectedSOP.sop_number}
                </span>
                <span className="badge badge-active">{selectedSOP.status}</span>
              </div>

              <h3 style={{ fontSize: 18, fontWeight: 800, marginBottom: 8 }}>{selectedSOP.title}</h3>
              <p style={{ fontSize: 13, color: 'var(--text-secondary)', marginBottom: 14 }}>{selectedSOP.purpose}</p>

              <div style={{ fontSize: 12, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 8 }}>
                Step-by-Step Procedure Workflow
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 10, marginBottom: 16 }}>
                {(selectedSOP.procedure_steps || []).map((st, idx) => (
                  <div key={idx} style={{ background: '#f8fafc', padding: 10, borderRadius: 6, border: '1px solid var(--border-light)', fontSize: 12.5 }}>
                    <div style={{ fontWeight: 700, color: 'var(--primary-dark)', marginBottom: 2 }}>
                      Step {st.step_number || idx + 1}: {st.title}
                    </div>
                    <div style={{ color: 'var(--text-dark)', marginBottom: 4 }}>{st.description}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Role Responsible: <strong>{st.role}</strong></div>
                  </div>
                ))}
              </div>

              <div style={{ fontSize: 12, color: 'var(--text-muted)', borderTop: '1px solid var(--border-light)', paddingTop: 12 }}>
                Owner: <strong>{selectedSOP.owner}</strong> • Dept: <strong>{selectedSOP.department}</strong>
              </div>
            </div>
          ) : (
            <div style={{ textAlign: 'center', padding: 40, color: 'var(--text-muted)' }}>
              Select an SOP to view procedural steps.
            </div>
          )}
        </div>
      </div>

      {showModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Author Standard Operating Procedure</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateSOP}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>SOP Number / Code</label>
                    <input
                      className="form-control"
                      placeholder="e.g. SOP-DATA-01"
                      value={formData.sop_number}
                      onChange={e => setFormData({ ...formData, sop_number: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Target Process *</label>
                    <input
                      className="form-control"
                      required
                      placeholder="e.g. Microdata Ingestion & Quality Cleaning"
                      value={formData.process}
                      onChange={e => setFormData({ ...formData, process: e.target.value })}
                    />
                  </div>
                </div>

                <div className="form-group">
                  <label>SOP Title *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. Statistical Data Anonymization Procedure"
                    value={formData.title}
                    onChange={e => setFormData({ ...formData, title: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Purpose & Objectives *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Specify why this procedure exists..."
                    value={formData.purpose}
                    onChange={e => setFormData({ ...formData, purpose: e.target.value })}
                  />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Responsible Department</label>
                    <input
                      className="form-control"
                      value={formData.department}
                      onChange={e => setFormData({ ...formData, department: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Review Frequency</label>
                    <select
                      className="form-control"
                      value={formData.review_frequency}
                      onChange={e => setFormData({ ...formData, review_frequency: e.target.value })}
                    >
                      <option value="Bi-Annual">Bi-Annual</option>
                      <option value="Annual">Annual</option>
                      <option value="Quarterly">Quarterly</option>
                    </select>
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save SOP</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
