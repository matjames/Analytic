import React, { useState } from 'react';

export default function RiskManagement({ risks, onRiskAdded, projectId }) {
  const [isAdding, setIsAdding] = useState(false);
  const [desc, setDesc] = useState('');
  const [cat, setCat] = useState('Operational');
  const [prob, setProb] = useState('Medium');
  const [imp, setImp] = useState('Medium');
  const [mit, setMit] = useState('');
  const [owner, setOwner] = useState('');

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!desc.trim()) return;

    const payload = {
      projectId,
      description: desc,
      category: cat,
      probability: prob,
      impact: imp,
      mitigation: mit,
      owner
    };

    fetch('http://localhost:8091/api/risks', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
      .then(res => res.json())
      .then(() => {
        onRiskAdded();
        setIsAdding(false);
        setDesc('');
        setMit('');
        setOwner('');
      })
      .catch(err => console.error("Error creating risk:", err));
  };

  const getHeatColor = (prob, imp) => {
    if (prob === 'High' && imp === 'High') return 'var(--accent-danger)';
    if (prob === 'High' || imp === 'High') return 'var(--accent-warning)';
    return 'var(--accent-success)';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
      <div style={{ display: 'flex', justifySelf: 'stretch', justifyContent: 'space-between', alignItems: 'center' }}>
        <h3 style={{ fontSize: '18px', fontWeight: '700' }}>Risk Register & Mitigation Log</h3>
        <button className="btn btn-primary" onClick={() => setIsAdding(true)}>+ Log Risk</button>
      </div>

      {isAdding && (
        <div className="modal-overlay">
          <form className="modal-content glass-panel" onSubmit={handleSubmit}>
            <h3>Log Project Risk</h3>

            <div className="form-group">
              <label>Risk Description</label>
              <textarea rows={2} value={desc} onChange={e => setDesc(e.target.value)} required />
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Category</label>
                <select value={cat} onChange={e => setCat(e.target.value)}>
                  <option value="Technical">Technical</option>
                  <option value="Financial">Financial</option>
                  <option value="Operational">Operational</option>
                  <option value="External">External</option>
                </select>
              </div>
              <div className="form-group">
                <label>Owner / Coordinator</label>
                <input type="text" value={owner} onChange={e => setOwner(e.target.value)} placeholder="e.g. David Chen" />
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Probability</label>
                <select value={prob} onChange={e => setProb(e.target.value)}>
                  <option value="Low">Low</option>
                  <option value="Medium">Medium</option>
                  <option value="High">High</option>
                </select>
              </div>
              <div className="form-group">
                <label>Impact</label>
                <select value={imp} onChange={e => setImp(e.target.value)}>
                  <option value="Low">Low</option>
                  <option value="Medium">Medium</option>
                  <option value="High">High</option>
                </select>
              </div>
            </div>

            <div className="form-group">
              <label>Mitigation Plan</label>
              <textarea rows={2} value={mit} onChange={e => setMit(e.target.value)} placeholder="Explain preventative steps..." />
            </div>

            <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '10px' }}>
              <button type="button" className="btn btn-secondary" onClick={() => setIsAdding(false)}>Cancel</button>
              <button type="submit" className="btn btn-primary">Save Risk</button>
            </div>
          </form>
        </div>
      )}

      <div className="glass-panel" style={{ padding: '24px' }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: '16px' }}>
          {risks && risks.length === 0 ? (
            <div style={{ textAlign: 'center', color: 'var(--text-secondary)' }}>No active risks logged.</div>
          ) : (
            risks && risks.map(risk => (
              <div key={risk.id} style={{ display: 'flex', gap: '16px', borderBottom: '1px solid var(--border-glass)', paddingBottom: '16px' }}>
                {/* Heat Indicator */}
                <div style={{
                  width: '60px',
                  height: '60px',
                  borderRadius: '12px',
                  background: 'rgba(255,255,255,0.03)',
                  border: `2px solid ${getHeatColor(risk.probability, risk.impact)}`,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '9px',
                  fontWeight: '700',
                  color: getHeatColor(risk.probability, risk.impact)
                }}>
                  <span>P: {risk.probability[0]}</span>
                  <span>I: {risk.impact[0]}</span>
                </div>

                <div style={{ flexGrow: 1, display: 'flex', flexDirection: 'column', gap: '6px' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span style={{ fontSize: '13px', fontWeight: '700' }}>{risk.description}</span>
                    <span className="badge" style={{ background: 'rgba(255,255,255,0.05)', color: 'var(--text-secondary)' }}>{risk.category}</span>
                  </div>

                  <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                    <strong style={{ color: 'var(--accent-secondary)' }}>Mitigation:</strong> {risk.mitigation}
                  </div>

                  <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', color: 'var(--text-muted)' }}>
                    <span>Owner: {risk.owner}</span>
                    <span>Status: <strong style={{ color: risk.status === 'Active' ? 'var(--accent-warning)' : 'var(--text-secondary)' }}>{risk.status}</strong></span>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
