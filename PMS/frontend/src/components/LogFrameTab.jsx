import React, { useState, useEffect } from 'react';

export default function LogFrameTab({ projectId, apiBase }) {
  const [logframes, setLogframes] = useState([]);
  const [toc, setToc] = useState(null);
  const [activeSubView, setActiveSubView] = useState('logframe');
  const [newItemLevel, setNewItemLevel] = useState('Outcome');
  const [newItemDesc, setNewItemDesc] = useState('');
  const [newItemIndicator, setNewItemIndicator] = useState('');
  const [newItemMoV, setNewItemMoV] = useState('');
  const [newItemAssumptions, setNewItemAssumptions] = useState('');

  const loadLogframes = () => {
    if (!projectId) return;
    fetch(`${apiBase}/api/projects/${projectId}/logframes`)
      .then(res => res.json())
      .then(data => setLogframes(Array.isArray(data) ? data : []))
      .catch(err => console.error("Error loading logframes:", err));

    fetch(`${apiBase}/api/projects/${projectId}/theory-of-change`)
      .then(res => res.json())
      .then(data => {
        if (data && data.title) setToc(data);
      })
      .catch(err => console.error("Error loading ToC:", err));
  };

  useEffect(() => {
    loadLogframes();
  }, [projectId]);

  const handleAddItem = (e) => {
    e.preventDefault();
    if (!logframes.length) {
      // Create parent logframe first if needed
      fetch(`${apiBase}/api/logframes`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          projectId: projectId,
          title: "Project Results Framework",
          description: "Authoritative Logical Framework matrix aligning project outputs with high-level institutional goals."
        })
      })
        .then(res => res.json())
        .then(lf => {
          postLogFrameItem(lf.id);
        });
    } else {
      postLogFrameItem(logframes[0].id);
    }
  };

  const postLogFrameItem = (logframeId) => {
    fetch(`${apiBase}/api/logframe-items`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        logframeId: logframeId,
        level: newItemLevel,
        code: `${newItemLevel.toUpperCase().slice(0, 3)}-${Date.now() % 1000}`,
        description: newItemDesc,
        indicators: [newItemIndicator],
        meansOfVerification: [newItemMoV],
        assumptions: [newItemAssumptions]
      })
    })
      .then(() => {
        setNewItemDesc('');
        setNewItemIndicator('');
        setNewItemMoV('');
        setNewItemAssumptions('');
        loadLogframes();
      })
      .catch(err => console.error("Error creating item:", err));
  };

  return (
    <div className="tab-pane-container">
      <div className="tab-header-row" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
        <div>
          <h2 style={{ margin: 0, fontSize: '1.4rem', fontWeight: 600 }}>Results & Impact Architecture</h2>
          <p style={{ margin: '0.25rem 0 0', color: '#64748b', fontSize: '0.9rem' }}>
            Logical Framework (LogFrame) Matrix & Theory of Change (ToC)
          </p>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button 
            className={`btn ${activeSubView === 'logframe' ? 'btn-primary' : 'btn-outline'}`}
            onClick={() => setActiveSubView('logframe')}
          >
            📋 LogFrame Matrix
          </button>
          <button 
            className={`btn ${activeSubView === 'toc' ? 'btn-primary' : 'btn-outline'}`}
            onClick={() => setActiveSubView('toc')}
          >
            🔄 Theory of Change (ToC)
          </button>
        </div>
      </div>

      {activeSubView === 'logframe' ? (
        <div>
          {/* Quick Add Form */}
          <div style={{ background: '#f8fafc', padding: '1.25rem', borderRadius: '8px', border: '1px solid #e2e8f0', marginBottom: '1.5rem' }}>
            <h4 style={{ margin: '0 0 1rem 0', fontSize: '1rem', color: '#1e293b' }}>➕ Add Results Chain Indicator</h4>
            <form onSubmit={handleAddItem} style={{ display: 'grid', gridTemplateColumns: '150px 1fr 1fr 1fr 1fr 100px', gap: '0.75rem', alignItems: 'end' }}>
              <div>
                <label style={{ fontSize: '0.75rem', fontWeight: 600, color: '#475569' }}>Level</label>
                <select 
                  value={newItemLevel} 
                  onChange={e => setNewItemLevel(e.target.value)}
                  style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #cbd5e1' }}
                >
                  <option value="Goal">🎯 Goal / Impact</option>
                  <option value="Outcome">🌟 Outcome</option>
                  <option value="Output">📦 Output</option>
                  <option value="Activity">⚡ Activity</option>
                </select>
              </div>
              <div>
                <label style={{ fontSize: '0.75rem', fontWeight: 600, color: '#475569' }}>Description</label>
                <input 
                  type="text" 
                  placeholder="e.g. Health facilities reporting regularly"
                  value={newItemDesc}
                  onChange={e => setNewItemDesc(e.target.value)}
                  required
                  style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #cbd5e1' }}
                />
              </div>
              <div>
                <label style={{ fontSize: '0.75rem', fontWeight: 600, color: '#475569' }}>Objectively Verifiable Indicator</label>
                <input 
                  type="text" 
                  placeholder="e.g. % of monthly reports received on time (Target: 95%)"
                  value={newItemIndicator}
                  onChange={e => setNewItemIndicator(e.target.value)}
                  required
                  style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #cbd5e1' }}
                />
              </div>
              <div>
                <label style={{ fontSize: '0.75rem', fontWeight: 600, color: '#475569' }}>Means of Verification</label>
                <input 
                  type="text" 
                  placeholder="e.g. StatCollect Submission Audits"
                  value={newItemMoV}
                  onChange={e => setNewItemMoV(e.target.value)}
                  style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #cbd5e1' }}
                />
              </div>
              <div>
                <label style={{ fontSize: '0.75rem', fontWeight: 600, color: '#475569' }}>Assumptions & Risks</label>
                <input 
                  type="text" 
                  placeholder="e.g. Continuous field tablet battery/solar"
                  value={newItemAssumptions}
                  onChange={e => setNewItemAssumptions(e.target.value)}
                  style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #cbd5e1' }}
                />
              </div>
              <div>
                <button type="submit" className="btn btn-primary" style={{ width: '100%', padding: '0.5rem' }}>
                  Save
                </button>
              </div>
            </form>
          </div>

          {/* Matrix Table */}
          <div style={{ overflowX: 'auto', background: '#fff', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '0.875rem' }}>
              <thead>
                <tr style={{ background: '#f1f5f9', borderBottom: '2px solid #cbd5e1' }}>
                  <th style={{ padding: '0.75rem 1rem', width: '120px' }}>Level</th>
                  <th style={{ padding: '0.75rem 1rem', width: '25%' }}>Intervention Logic</th>
                  <th style={{ padding: '0.75rem 1rem', width: '30%' }}>Indicators (OVI)</th>
                  <th style={{ padding: '0.75rem 1rem', width: '20%' }}>Means of Verification</th>
                  <th style={{ padding: '0.75rem 1rem', width: '25%' }}>Critical Assumptions</th>
                </tr>
              </thead>
              <tbody>
                {logframes.length > 0 && logframes[0].items && logframes[0].items.length > 0 ? (
                  logframes[0].items.map((item, idx) => (
                    <tr key={item.id || idx} style={{ borderBottom: '1px solid #e2e8f0', background: idx % 2 === 0 ? '#fff' : '#fafafa' }}>
                      <td style={{ padding: '0.75rem 1rem' }}>
                        <span style={{ 
                          display: 'inline-block', 
                          padding: '0.25rem 0.5rem', 
                          borderRadius: '4px', 
                          fontWeight: 600, 
                          fontSize: '0.75rem',
                          background: item.level === 'Goal' ? '#dbeafe' : item.level === 'Outcome' ? '#dcfce7' : item.level === 'Output' ? '#fef3c7' : '#f1f5f9',
                          color: item.level === 'Goal' ? '#1d4ed8' : item.level === 'Outcome' ? '#15803d' : item.level === 'Output' ? '#b45309' : '#475569'
                        }}>
                          {item.level}
                        </span>
                      </td>
                      <td style={{ padding: '0.75rem 1rem', fontWeight: 500, color: '#1e293b' }}>
                        {item.description}
                      </td>
                      <td style={{ padding: '0.75rem 1rem', color: '#334155' }}>
                        {item.indicators && item.indicators.join(', ')}
                      </td>
                      <td style={{ padding: '0.75rem 1rem', color: '#64748b' }}>
                        {item.meansOfVerification && item.meansOfVerification.join(', ')}
                      </td>
                      <td style={{ padding: '0.75rem 1rem', color: '#64748b', fontStyle: 'italic' }}>
                        {item.assumptions && item.assumptions.join(', ')}
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan="5" style={{ padding: '2rem', textAlign: 'center', color: '#94a3b8' }}>
                      No LogFrame matrix items defined yet. Use the form above to add your first Goal, Outcome, or Output indicator.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        /* Theory of Change View */
        <div style={{ background: '#fff', padding: '1.5rem', borderRadius: '8px', border: '1px solid #e2e8f0' }}>
          <h3 style={{ marginTop: 0, color: '#0f172a' }}>{toc ? toc.title : "Theory of Change (ToC)"}</h3>
          <p style={{ color: '#64748b', fontSize: '0.9rem', marginBottom: '2rem' }}>
            {toc ? toc.narrative : "Causal pathway showing how project inputs lead to activities, outputs, intermediate outcomes, and long-term socio-economic impact."}
          </p>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: '1rem' }}>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #64748b' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#475569' }}>1. INPUTS</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                <li>Dedicated research budget</li>
                <li>Certified field enumerators</li>
                <li>StatGate platform infrastructure</li>
              </ul>
            </div>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #3b82f6' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#2563eb' }}>2. ACTIVITIES</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                <li>Tablet field survey collection</li>
                <li>Real-time automated QA validation</li>
                <li>Stakeholder dissemination workshops</li>
              </ul>
            </div>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #eab308' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#ca8a04' }}>3. OUTPUTS</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                <li>Validated district datasets</li>
                <li>Policy brief publications</li>
                <li>Open science repository deposit</li>
              </ul>
            </div>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #10b981' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#059669' }}>4. OUTCOMES</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                <li>Ministries adopt evidence in budgeting</li>
                <li>Accelerated resource allocation</li>
                <li>Increased survey data trust</li>
              </ul>
            </div>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #8b5cf6' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#7c3aed' }}>5. IMPACT</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                <li>Measurable progress towards national development plan & SDGs</li>
              </ul>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
