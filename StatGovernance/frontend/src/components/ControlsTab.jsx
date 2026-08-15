import React, { useState, useEffect } from 'react';

export default function ControlsTab({ apiBase, token, onRefreshDashboard }) {
  const [controls, setControls] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedControl, setSelectedControl] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [showTestModal, setShowTestModal] = useState(false);

  const [formData, setFormData] = useState({
    control_code: '',
    name: '',
    description: '',
    objective: '',
    owner: '',
    department: 'ICT & Infrastructure',
    control_type: 'Preventive',
    frequency: 'Continuous',
    test_method: 'Automated Probe',
    evidence_requirement: '',
    effectiveness: 'Effective'
  });

  const [testForm, setTestForm] = useState({
    tester: '',
    result: 'Pass',
    effectiveness: 'Effective',
    findings: '',
    evidence_ref: ''
  });

  const fetchControls = async () => {
    try {
      setLoading(true);
      const res = await fetch(`${apiBase}/api/controls`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setControls(data);
        if (data.length > 0 && !selectedControl) {
          setSelectedControl(data[0]);
        }
      }
    } catch (err) {
      console.error('Error loading controls:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchControls();
  }, [apiBase, token]);

  const handleCreateControl = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${apiBase}/api/controls`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(formData)
      });
      if (res.ok) {
        setShowModal(false);
        fetchControls();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error creating control:', err);
    }
  };

  const handleExecuteTest = async (e) => {
    e.preventDefault();
    if (!selectedControl) return;

    try {
      const res = await fetch(`${apiBase}/api/controls/${selectedControl.id}/test`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(testForm)
      });
      if (res.ok) {
        setShowTestModal(false);
        setTestForm({ tester: '', result: 'Pass', effectiveness: 'Effective', findings: '', evidence_ref: '' });
        fetchControls();
        onRefreshDashboard();
      }
    } catch (err) {
      console.error('Error executing control test:', err);
    }
  };

  const getEffectivenessBadge = (eff) => {
    switch (eff) {
      case 'Effective':
        return <span className="badge badge-effective">Effective</span>;
      case 'Partially Effective':
        return <span className="badge badge-partially">Partially Effective</span>;
      case 'Ineffective':
        return <span className="badge badge-ineffective">Ineffective</span>;
      default:
        return <span className="badge badge-draft">Not Tested</span>;
    }
  };

  return (
    <div>
      <div className="section-header">
        <div>
          <h2 className="section-title">
            <span>🛡️</span> Institutional Control Library & Testing Engine
          </h2>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            Maintain preventive, detective, and corrective controls with automated audit test probes.
          </p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowModal(true)}>
          + Add Control
        </button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1.4fr 1fr', gap: 24 }}>
        {/* Controls Table */}
        <div className="glass-panel table-container">
          <table className="gov-table">
            <thead>
              <tr>
                <th>Code</th>
                <th>Name & Objective</th>
                <th>Type / Freq</th>
                <th>Effectiveness</th>
                <th>Last Tested</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {controls.map(c => (
                <tr
                  key={c.id}
                  onClick={() => setSelectedControl(c)}
                  style={{
                    cursor: 'pointer',
                    background: selectedControl?.id === c.id ? '#f1f5f9' : 'transparent'
                  }}
                >
                  <td style={{ fontFamily: 'var(--font-mono)', fontWeight: 600, fontSize: 12 }}>
                    {c.control_code}
                  </td>
                  <td>
                    <div style={{ fontWeight: 600, color: 'var(--text-dark)' }}>{c.name}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{c.department}</div>
                  </td>
                  <td>
                    <div style={{ fontSize: 12 }}>{c.control_type}</div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{c.frequency}</div>
                  </td>
                  <td>{getEffectivenessBadge(c.effectiveness)}</td>
                  <td style={{ fontSize: 12 }}>{c.last_tested ? new Date(c.last_tested).toLocaleDateString() : 'Never'}</td>
                  <td>
                    <button
                      className="btn btn-secondary btn-sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        setSelectedControl(c);
                        setShowTestModal(true);
                      }}
                    >
                      Run Test
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Selected Control Detail */}
        <div className="glass-panel" style={{ padding: 22 }}>
          {selectedControl ? (
            <div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
                <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, fontWeight: 700, color: 'var(--primary-color)' }}>
                  {selectedControl.control_code}
                </span>
                {getEffectivenessBadge(selectedControl.effectiveness)}
              </div>

              <h3 style={{ fontSize: 18, fontWeight: 800, marginBottom: 8 }}>
                {selectedControl.name}
              </h3>

              <div style={{ background: '#f8fafc', padding: 14, borderRadius: 8, border: '1px solid var(--border-light)', marginBottom: 16 }}>
                <div style={{ fontSize: 11, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 4 }}>
                  Control Objective
                </div>
                <div style={{ fontSize: 13, color: 'var(--text-dark)' }}>
                  {selectedControl.objective}
                </div>
              </div>

              <div style={{ marginBottom: 16 }}>
                <div style={{ fontSize: 11, fontWeight: 700, textTransform: 'uppercase', color: 'var(--text-secondary)', marginBottom: 4 }}>
                  Description & Implementation
                </div>
                <div style={{ fontSize: 13, color: 'var(--text-dark)', background: '#ffffff', padding: 12, borderRadius: 8, border: '1px solid var(--border-light)' }}>
                  {selectedControl.description}
                </div>
              </div>

              <div className="form-row" style={{ marginBottom: 16 }}>
                <div>
                  <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>Test Method:</span>
                  <div style={{ fontSize: 13, fontWeight: 600 }}>{selectedControl.test_method}</div>
                </div>
                <div>
                  <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>Frequency:</span>
                  <div style={{ fontSize: 13, fontWeight: 600 }}>{selectedControl.frequency}</div>
                </div>
              </div>

              <button
                className="btn btn-primary"
                style={{ width: '100%', marginTop: 8 }}
                onClick={() => setShowTestModal(true)}
              >
                Execute Control Test Probe
              </button>
            </div>
          ) : (
            <div style={{ textAlign: 'center', padding: 40, color: 'var(--text-muted)' }}>
              Select a control to view test parameters.
            </div>
          )}
        </div>
      </div>

      {/* Add Control Modal */}
      {showModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Define Internal Control</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowModal(false)}>✕</button>
            </div>
            <form onSubmit={handleCreateControl}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Control Code</label>
                    <input
                      className="form-control"
                      placeholder="e.g. CTL-SEC-2026"
                      value={formData.control_code}
                      onChange={e => setFormData({ ...formData, control_code: e.target.value })}
                    />
                  </div>
                  <div className="form-group">
                    <label>Control Type</label>
                    <select
                      className="form-control"
                      value={formData.control_type}
                      onChange={e => setFormData({ ...formData, control_type: e.target.value })}
                    >
                      <option value="Preventive">Preventive</option>
                      <option value="Detective">Detective</option>
                      <option value="Corrective">Corrective</option>
                      <option value="Directive">Directive</option>
                      <option value="Compensating">Compensating</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Control Name *</label>
                  <input
                    className="form-control"
                    required
                    placeholder="e.g. Automated Role-Based JWT Scope Verification"
                    value={formData.name}
                    onChange={e => setFormData({ ...formData, name: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Control Objective *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Specify the objective of this control..."
                    value={formData.objective}
                    onChange={e => setFormData({ ...formData, objective: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Detailed Control Description *</label>
                  <textarea
                    className="form-control"
                    required
                    placeholder="Detail the technical or operational mechanism..."
                    value={formData.description}
                    onChange={e => setFormData({ ...formData, description: e.target.value })}
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
                    <label>Test Frequency</label>
                    <select
                      className="form-control"
                      value={formData.frequency}
                      onChange={e => setFormData({ ...formData, frequency: e.target.value })}
                    >
                      <option value="Continuous">Continuous</option>
                      <option value="Daily">Daily</option>
                      <option value="Weekly">Weekly</option>
                      <option value="Monthly">Monthly</option>
                      <option value="Quarterly">Quarterly</option>
                    </select>
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Save Control</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Execute Test Modal */}
      {showTestModal && selectedControl && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Execute Test on {selectedControl.control_code}</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowTestModal(false)}>✕</button>
            </div>
            <form onSubmit={handleExecuteTest}>
              <div className="modal-body">
                <div className="form-row">
                  <div className="form-group">
                    <label>Test Result *</label>
                    <select
                      className="form-control"
                      value={testForm.result}
                      onChange={e => setTestForm({ ...testForm, result: e.target.value })}
                    >
                      <option value="Pass">Pass</option>
                      <option value="Partial Pass">Partial Pass</option>
                      <option value="Fail">Fail</option>
                    </select>
                  </div>
                  <div className="form-group">
                    <label>Assessed Effectiveness *</label>
                    <select
                      className="form-control"
                      value={testForm.effectiveness}
                      onChange={e => setTestForm({ ...testForm, effectiveness: e.target.value })}
                    >
                      <option value="Effective">Effective</option>
                      <option value="Partially Effective">Partially Effective</option>
                      <option value="Ineffective">Ineffective</option>
                    </select>
                  </div>
                </div>

                <div className="form-group">
                  <label>Auditor / Tester Name</label>
                  <input
                    className="form-control"
                    placeholder="Enter tester name..."
                    value={testForm.tester}
                    onChange={e => setTestForm({ ...testForm, tester: e.target.value })}
                  />
                </div>

                <div className="form-group">
                  <label>Findings & Observations</label>
                  <textarea
                    className="form-control"
                    placeholder="Record any test findings or failure anomalies..."
                    value={testForm.findings}
                    onChange={e => setTestForm({ ...testForm, findings: e.target.value })}
                  />
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => setShowTestModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Record Test Result</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
