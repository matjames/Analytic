import React, { useState } from 'react';

const API_BASE = import.meta.env.VITE_PMS_API_URL || 'http://localhost:8091';

export default function IssuesTab({ issues = [], assumptions = [], correctiveActions = [], projectId, onRefresh }) {
  const [isAddingIssue, setIsAddingIssue] = useState(false);
  const [isAddingAssumption, setIsAddingAssumption] = useState(false);
  const [isAddingAction, setIsAddingAction] = useState(false);

  const [issueTitle, setIssueTitle] = useState('');
  const [issueDesc, setIssueDesc] = useState('');
  const [issueCat, setIssueCat] = useState('Operational');
  const [issuePriority, setIssuePriority] = useState('Medium');
  const [issueOwner, setIssueOwner] = useState('');

  const [assumptionDesc, setAssumptionDesc] = useState('');
  const [assumptionCat, setAssumptionCat] = useState('Financial');

  const [actionDesc, setActionDesc] = useState('');
  const [actionOwner, setActionOwner] = useState('');
  const [actionDue, setActionDue] = useState('');

  const handleAddIssue = (e) => {
    e.preventDefault();
    fetch(`${API_BASE}/api/issues`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId, title: issueTitle, description: issueDesc, category: issueCat, priority: issuePriority, status: 'Open', owner: issueOwner })
    })
      .then(res => res.json())
      .then(() => {
        onRefresh();
        setIsAddingIssue(false);
        setIssueTitle(''); setIssueDesc(''); setIssueOwner('');
      })
      .catch(err => console.error("Error creating issue:", err));
  };

  const handleAddAssumption = (e) => {
    e.preventDefault();
    fetch(`${API_BASE}/api/assumptions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId, description: assumptionDesc, category: assumptionCat, status: 'Active' })
    })
      .then(res => res.json())
      .then(() => {
        onRefresh();
        setIsAddingAssumption(false);
        setAssumptionDesc('');
      })
      .catch(err => console.error("Error creating assumption:", err));
  };

  const handleAddAction = (e) => {
    e.preventDefault();
    fetch(`${API_BASE}/api/corrective-actions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId, description: actionDesc, status: 'Open', owner: actionOwner, dueDate: actionDue })
    })
      .then(res => res.json())
      .then(() => {
        onRefresh();
        setIsAddingAction(false);
        setActionDesc(''); setActionOwner(''); setActionDue('');
      })
      .catch(err => console.error("Error creating corrective action:", err));
  };

  const getPriorityColor = (p) => {
    if (p === 'High') return 'var(--accent-danger)';
    if (p === 'Medium') return 'var(--accent-warning)';
    return 'var(--accent-success)';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>⚠️ Issues & Risk Management</h2>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
            Issues, assumptions, corrective actions, and mitigation plans
          </p>
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          <button className="btn btn-primary" style={{ fontSize: '11px', padding: '6px 12px' }} onClick={() => setIsAddingIssue(true)}>+ Issue</button>
          <button className="btn btn-secondary" style={{ fontSize: '11px', padding: '6px 12px' }} onClick={() => setIsAddingAssumption(true)}>+ Assumption</button>
          <button className="btn btn-secondary" style={{ fontSize: '11px', padding: '6px 12px' }} onClick={() => setIsAddingAction(true)}>+ Action</button>
        </div>
      </div>

      {/* Modals */}
      {isAddingIssue && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleAddIssue}>
            <h3>Log Project Issue</h3>
            <div className="form-group">
              <label>Issue Title</label>
              <input type="text" value={issueTitle} onChange={e => setIssueTitle(e.target.value)} required />
            </div>
            <div className="form-group">
              <label>Description</label>
              <textarea rows={2} value={issueDesc} onChange={e => setIssueDesc(e.target.value)} required />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Category</label>
                <select value={issueCat} onChange={e => setIssueCat(e.target.value)}>
                  <option value="Operational">Operational</option>
                  <option value="Technical">Technical</option>
                  <option value="Financial">Financial</option>
                  <option value="Human Resources">Human Resources</option>
                  <option value="Equipment">Equipment</option>
                  <option value="External">External</option>
                </select>
              </div>
              <div className="form-group">
                <label>Priority</label>
                <select value={issuePriority} onChange={e => setIssuePriority(e.target.value)}>
                  <option value="Low">Low</option>
                  <option value="Medium">Medium</option>
                  <option value="High">High</option>
                </select>
              </div>
            </div>
            <div className="form-group">
              <label>Owner</label>
              <input type="text" value={issueOwner} onChange={e => setIssueOwner(e.target.value)} placeholder="e.g. David Chen" />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAddingIssue(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Log Issue</button>
            </div>
          </form>
        </div>
      )}

      {isAddingAssumption && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleAddAssumption}>
            <h3>Add Assumption</h3>
            <div className="form-group">
              <label>Assumption</label>
              <textarea rows={2} value={assumptionDesc} onChange={e => setAssumptionDesc(e.target.value)} required />
            </div>
            <div className="form-group">
              <label>Category</label>
              <select value={assumptionCat} onChange={e => setAssumptionCat(e.target.value)}>
                <option value="Financial">Financial</option>
                <option value="Operational">Operational</option>
                <option value="Community">Community</option>
                <option value="Technical">Technical</option>
                <option value="Political">Political</option>
              </select>
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAddingAssumption(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Add Assumption</button>
            </div>
          </form>
        </div>
      )}

      {isAddingAction && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleAddAction}>
            <h3>Add Corrective Action</h3>
            <div className="form-group">
              <label>Action Description</label>
              <textarea rows={2} value={actionDesc} onChange={e => setActionDesc(e.target.value)} required />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Owner</label>
                <input type="text" value={actionOwner} onChange={e => setActionOwner(e.target.value)} />
              </div>
              <div className="form-group">
                <label>Due Date</label>
                <input type="date" value={actionDue} onChange={e => setActionDue(e.target.value)} />
              </div>
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAddingAction(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Add Action</button>
            </div>
          </form>
        </div>
      )}

      {/* Summary stats */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        {[
          { label: 'Open Issues', value: issues.filter(i => i.status !== 'Resolved').length, icon: '🔴', color: 'var(--accent-danger)' },
          { label: 'In Progress', value: issues.filter(i => i.status === 'In Progress').length, icon: '🟡', color: 'var(--accent-warning)' },
          { label: 'Assumptions', value: assumptions.length, icon: '📋', color: 'var(--accent-primary)' },
          { label: 'Actions', value: correctiveActions.filter(a => a.status !== 'Completed').length, icon: '🔧', color: 'var(--accent-secondary)' },
        ].map(stat => (
          <div key={stat.label} className="glass-panel" style={{ padding: '16px', display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ fontSize: '24px' }}>{stat.icon}</span>
            <div>
              <div style={{ fontSize: '22px', fontWeight: '800', color: stat.color }}>{stat.value}</div>
              <div style={{ fontSize: '10px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>{stat.label}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Issues list */}
      <div className="glass-panel" style={{ padding: '24px' }}>
        <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>Project Issues</h3>
        {issues.length === 0 ? (
          <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No issues logged.</div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
            {issues.map(issue => (
              <div key={issue.id} style={{ padding: '14px', background: 'rgba(239,68,68,0.03)', borderRadius: '8px', border: '1px solid rgba(239,68,68,0.1)' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                  <span style={{ fontSize: '13px', fontWeight: '700' }}>{issue.title}</span>
                  <div style={{ display: 'flex', gap: '6px' }}>
                    <span className="badge" style={{ fontSize: '9px', background: `${getPriorityColor(issue.priority)}20`, color: getPriorityColor(issue.priority) }}>{issue.priority}</span>
                    <span className="badge" style={{ fontSize: '9px', background: 'rgba(22,92,146,0.1)', color: 'var(--primary-color)' }}>{issue.status}</span>
                  </div>
                </div>
                <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{issue.description}</div>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '10px', color: 'var(--text-muted)', marginTop: '6px' }}>
                  <span>Owner: {issue.owner}</span>
                  <span>Due: {issue.dueDate}</span>
                </div>
                {issue.resolution && (
                  <div style={{ fontSize: '11px', color: 'var(--accent-success)', marginTop: '6px', borderTop: '1px solid rgba(16,185,129,0.1)', paddingTop: '6px' }}>
                    ✅ Resolution: {issue.resolution}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Assumptions + Corrective Actions */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px' }}>
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>📋 Assumptions</h3>
          {assumptions.length === 0 ? (
            <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No assumptions logged.</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              {assumptions.map(a => (
                <div key={a.id} style={{ padding: '12px', background: 'rgba(99,102,241,0.05)', borderRadius: '8px', border: '1px solid rgba(99,102,241,0.1)' }}>
                  <div style={{ fontSize: '12px', fontWeight: '600' }}>{a.description}</div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px' }}>
                    <span>Category: {a.category}</span>
                    <span>Status: {a.status}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>🔧 Corrective Actions</h3>
          {correctiveActions.length === 0 ? (
            <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No corrective actions defined.</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              {correctiveActions.map(ca => (
                <div key={ca.id} style={{ padding: '12px', background: 'rgba(6,182,212,0.05)', borderRadius: '8px', border: '1px solid rgba(6,182,212,0.1)' }}>
                  <div style={{ fontSize: '12px', fontWeight: '600' }}>{ca.description}</div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px' }}>
                    <span>Owner: {ca.owner}</span>
                    <span>Status: {ca.status} | Due: {ca.dueDate}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}