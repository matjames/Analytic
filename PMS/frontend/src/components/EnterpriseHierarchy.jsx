import React, { useState } from 'react';

const API_BASE = import.meta.env.VITE_PMS_API_URL || 'http://localhost:8091';

export default function EnterpriseHierarchy({ components = [], activities = [], deliverables = [], milestones = [], projectId, onRefresh }) {
  const [isAddingComponent, setIsAddingComponent] = useState(false);
  const [isAddingActivity, setIsAddingActivity] = useState(false);
  const [isAddingDeliverable, setIsAddingDeliverable] = useState(false);
  const [isAddingMilestone, setIsAddingMilestone] = useState(false);

  const [compName, setCompName] = useState('');
  const [compCode, setCompCode] = useState('');
  const [compDesc, setCompDesc] = useState('');
  const [compType, setCompType] = useState('Core');

  const [actName, setActName] = useState('');
  const [actCode, setActCode] = useState('');
  const [actDesc, setActDesc] = useState('');
  const [actComponent, setActComponent] = useState('');

  const [delName, setDelName] = useState('');
  const [delDesc, setDelDesc] = useState('');
  const [delType, setDelType] = useState('Document');
  const [delDue, setDelDue] = useState('');

  const [msName, setMsName] = useState('');
  const [msDesc, setMsDesc] = useState('');
  const [msDue, setMsDue] = useState('');

  const handleAddComponent = (e) => {
    e.preventDefault();
    fetch(`${API_BASE}/api/components`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId, name: compName, code: compCode, description: compDesc, type: compType, order: components.length + 1 })
    })
      .then(res => res.json())
      .then(() => {
        onRefresh();
        setIsAddingComponent(false);
        setCompName(''); setCompCode(''); setCompDesc('');
      })
      .catch(err => console.error("Error creating component:", err));
  };

  const handleAddActivity = (e) => {
    e.preventDefault();
    fetch(`${API_BASE}/api/activities`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId, componentId: actComponent, name: actName, code: actCode, description: actDesc, status: 'Planned', progress: 0 })
    })
      .then(res => res.json())
      .then(() => {
        onRefresh();
        setIsAddingActivity(false);
        setActName(''); setActCode(''); setActDesc(''); setActComponent('');
      })
      .catch(err => console.error("Error creating activity:", err));
  };

  const handleAddDeliverable = (e) => {
    e.preventDefault();
    fetch(`${API_BASE}/api/deliverables`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId, name: delName, description: delDesc, type: delType, status: 'Pending', dueDate: delDue })
    })
      .then(res => res.json())
      .then(() => {
        onRefresh();
        setIsAddingDeliverable(false);
        setDelName(''); setDelDesc(''); setDelDue('');
      })
      .catch(err => console.error("Error creating deliverable:", err));
  };

  const handleAddMilestone = (e) => {
    e.preventDefault();
    fetch(`${API_BASE}/api/milestones`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ projectId, name: msName, description: msDesc, dueDate: msDue, status: 'Pending' })
    })
      .then(res => res.json())
      .then(() => {
        onRefresh();
        setIsAddingMilestone(false);
        setMsName(''); setMsDesc(''); setMsDue('');
      })
      .catch(err => console.error("Error creating milestone:", err));
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>🏗️ Enterprise Project Hierarchy</h2>
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
            Organization → Portfolio → Programme → Project → Component → Activity → Task → Deliverable → Output → Outcome → Impact
          </p>
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          <button className="btn btn-primary" style={{ fontSize: '11px', padding: '6px 12px' }} onClick={() => setIsAddingComponent(true)}>+ Component</button>
          <button className="btn btn-secondary" style={{ fontSize: '11px', padding: '6px 12px' }} onClick={() => setIsAddingActivity(true)}>+ Activity</button>
          <button className="btn btn-secondary" style={{ fontSize: '11px', padding: '6px 12px' }} onClick={() => setIsAddingDeliverable(true)}>+ Deliverable</button>
          <button className="btn btn-secondary" style={{ fontSize: '11px', padding: '6px 12px' }} onClick={() => setIsAddingMilestone(true)}>+ Milestone</button>
        </div>
      </div>

      {/* Modals */}
      {isAddingComponent && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleAddComponent}>
            <h3>Add Project Component</h3>
            <div className="form-row">
              <div className="form-group">
                <label>Component Name</label>
                <input type="text" value={compName} onChange={e => setCompName(e.target.value)} required />
              </div>
              <div className="form-group">
                <label>Code</label>
                <input type="text" value={compCode} onChange={e => setCompCode(e.target.value)} placeholder="C1" required />
              </div>
            </div>
            <div className="form-group">
              <label>Description</label>
              <textarea rows={2} value={compDesc} onChange={e => setCompDesc(e.target.value)} />
            </div>
            <div className="form-group">
              <label>Type</label>
              <select value={compType} onChange={e => setCompType(e.target.value)}>
                <option value="Core">Core</option>
                <option value="Design">Design</option>
                <option value="Operations">Operations</option>
                <option value="Analysis">Analysis</option>
                <option value="Support">Support</option>
              </select>
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAddingComponent(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Add Component</button>
            </div>
          </form>
        </div>
      )}

      {isAddingActivity && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleAddActivity}>
            <h3>Add Project Activity</h3>
            <div className="form-row">
              <div className="form-group">
                <label>Activity Name</label>
                <input type="text" value={actName} onChange={e => setActName(e.target.value)} required />
              </div>
              <div className="form-group">
                <label>Code</label>
                <input type="text" value={actCode} onChange={e => setActCode(e.target.value)} placeholder="A1.1" required />
              </div>
            </div>
            <div className="form-group">
              <label>Parent Component</label>
              <select value={actComponent} onChange={e => setActComponent(e.target.value)} required>
                <option value="">Select component...</option>
                {components.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <div className="form-group">
              <label>Description</label>
              <textarea rows={2} value={actDesc} onChange={e => setActDesc(e.target.value)} />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAddingActivity(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Add Activity</button>
            </div>
          </form>
        </div>
      )}

      {isAddingDeliverable && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleAddDeliverable}>
            <h3>Add Deliverable</h3>
            <div className="form-group">
              <label>Deliverable Name</label>
              <input type="text" value={delName} onChange={e => setDelName(e.target.value)} required />
            </div>
            <div className="form-row">
              <div className="form-group">
                <label>Type</label>
                <select value={delType} onChange={e => setDelType(e.target.value)}>
                  <option value="Document">Document</option>
                  <option value="Dataset">Dataset</option>
                  <option value="Software">Software</option>
                  <option value="Report">Report</option>
                  <option value="Service">Service</option>
                </select>
              </div>
              <div className="form-group">
                <label>Due Date</label>
                <input type="date" value={delDue} onChange={e => setDelDue(e.target.value)} />
              </div>
            </div>
            <div className="form-group">
              <label>Description</label>
              <textarea rows={2} value={delDesc} onChange={e => setDelDesc(e.target.value)} />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAddingDeliverable(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Add Deliverable</button>
            </div>
          </form>
        </div>
      )}

      {isAddingMilestone && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleAddMilestone}>
            <h3>Add Milestone</h3>
            <div className="form-group">
              <label>Milestone Name</label>
              <input type="text" value={msName} onChange={e => setMsName(e.target.value)} required />
            </div>
            <div className="form-group">
              <label>Due Date</label>
              <input type="date" value={msDue} onChange={e => setMsDue(e.target.value)} required />
            </div>
            <div className="form-group">
              <label>Description</label>
              <textarea rows={2} value={msDesc} onChange={e => setMsDesc(e.target.value)} />
            </div>
            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAddingMilestone(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Add Milestone</button>
            </div>
          </form>
        </div>
      )}

      {/* Hierarchy Visualization */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px' }}>
        {/* Components */}
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>📦 Components</h3>
          {components.length === 0 ? (
            <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No components defined yet.</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              {components.map(comp => (
                <div key={comp.id} style={{ padding: '14px', background: 'rgba(22,92,146,0.05)', borderRadius: '8px', border: '1px solid rgba(22,92,146,0.1)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                    <span style={{ fontSize: '13px', fontWeight: '700' }}>{comp.name}</span>
                    <span className="badge" style={{ fontSize: '9px', background: 'rgba(22,92,146,0.1)', color: 'var(--primary-color)' }}>{comp.code}</span>
                  </div>
                  <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{comp.description}</div>
                  <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '4px' }}>Type: {comp.type}</div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Activities */}
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>⚡ Activities</h3>
          {activities.length === 0 ? (
            <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No activities defined yet.</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              {activities.map(act => (
                <div key={act.id} style={{ padding: '14px', background: 'rgba(6,182,212,0.05)', borderRadius: '8px', border: '1px solid rgba(6,182,212,0.1)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                    <span style={{ fontSize: '13px', fontWeight: '700' }}>{act.name}</span>
                    <span className="badge" style={{ fontSize: '9px', background: 'rgba(6,182,212,0.1)', color: 'var(--accent-secondary)' }}>{act.code}</span>
                  </div>
                  <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{act.description}</div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '10px', color: 'var(--text-muted)', marginTop: '6px' }}>
                    <span>Status: {act.status}</span>
                    <span>Progress: {act.progress}%</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Deliverables */}
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>📄 Deliverables</h3>
          {deliverables.length === 0 ? (
            <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No deliverables defined yet.</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              {deliverables.map(del => (
                <div key={del.id} style={{ padding: '14px', background: 'rgba(16,185,129,0.05)', borderRadius: '8px', border: '1px solid rgba(16,185,129,0.1)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                    <span style={{ fontSize: '13px', fontWeight: '700' }}>{del.name}</span>
                    <span className="badge" style={{ fontSize: '9px', background: 'rgba(16,185,129,0.1)', color: 'var(--accent-success)' }}>{del.status}</span>
                  </div>
                  <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{del.description}</div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '10px', color: 'var(--text-muted)', marginTop: '6px' }}>
                    <span>Type: {del.type}</span>
                    <span>Due: {del.dueDate}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Milestones */}
        <div className="glass-panel" style={{ padding: '24px' }}>
          <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>🎯 Milestones</h3>
          {milestones.length === 0 ? (
            <div style={{ fontSize: '12px', color: 'var(--text-muted)', textAlign: 'center', padding: '20px' }}>No milestones defined yet.</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              {milestones.map(ms => (
                <div key={ms.id} style={{ padding: '14px', background: 'rgba(245,158,11,0.05)', borderRadius: '8px', border: '1px solid rgba(245,158,11,0.1)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '6px' }}>
                    <span style={{ fontSize: '13px', fontWeight: '700' }}>{ms.name}</span>
                    <span className="badge" style={{ fontSize: '9px', background: 'rgba(245,158,11,0.1)', color: 'var(--accent-warning)' }}>{ms.status}</span>
                  </div>
                  <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{ms.description}</div>
                  <div style={{ fontSize: '10px', color: 'var(--text-muted)', marginTop: '6px' }}>Due: {ms.dueDate} | Owner: {ms.owner}</div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}