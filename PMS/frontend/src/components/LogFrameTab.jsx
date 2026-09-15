import React, { useState, useEffect } from 'react';

const toLines = (value) => Array.isArray(value) ? value.join('\n') : '';
const fromLines = (value) => value.split('\n').map(line => line.trim()).filter(Boolean);

export default function LogFrameTab({ projectId, apiBase }) {
  const [logframes, setLogframes] = useState([]);
  const [toc, setToc] = useState(null);
  const [tocForm, setTocForm] = useState({
    title: 'Theory of Change (ToC)',
    narrative: '',
    inputs: '',
    activities: '',
    outputs: '',
    shortTermOutcomes: '',
    longTermOutcomes: '',
    impact: '',
    assumptions: ''
  });
  const [tocSaving, setTocSaving] = useState(false);
  const [tocMessage, setTocMessage] = useState('');
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
        if (data && data.title) {
          setToc(data);
          setTocForm({
            title: data.title || '',
            narrative: data.narrative || '',
            inputs: toLines(data.inputs),
            activities: toLines(data.activities),
            outputs: toLines(data.outputs),
            shortTermOutcomes: toLines(data.shortTermOutcomes),
            longTermOutcomes: toLines(data.longTermOutcomes),
            impact: toLines(data.impact),
            assumptions: toLines(data.assumptions)
          });
        }
      })
      .catch(err => console.error("Error loading ToC:", err));
  };

  const handleSaveToc = (e) => {
    e.preventDefault();
    setTocSaving(true);
    setTocMessage('');
    fetch(`${apiBase}/api/theory-of-change`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        projectId,
        title: tocForm.title,
        narrative: tocForm.narrative,
        inputs: fromLines(tocForm.inputs),
        activities: fromLines(tocForm.activities),
        outputs: fromLines(tocForm.outputs),
        shortTermOutcomes: fromLines(tocForm.shortTermOutcomes),
        longTermOutcomes: fromLines(tocForm.longTermOutcomes),
        impact: fromLines(tocForm.impact),
        assumptions: fromLines(tocForm.assumptions)
      })
    })
      .then(res => {
        if (!res.ok) throw new Error(`Save failed (${res.status})`);
        return res.json();
      })
      .then(data => {
        setToc(data);
        setTocMessage('Theory of Change saved.');
      })
      .catch(err => setTocMessage(err.message))
      .finally(() => setTocSaving(false));
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
          <h3 style={{ marginTop: 0, color: '#0f172a' }}>{tocForm.title}</h3>
          <p style={{ color: '#64748b', fontSize: '0.9rem', marginBottom: '2rem' }}>
            {tocForm.narrative || "Causal pathway showing how project inputs lead to activities, outputs, intermediate outcomes, and long-term socio-economic impact."}
          </p>

          <form onSubmit={handleSaveToc} style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '1rem', marginBottom: '1.5rem', padding: '1rem', background: '#f8fafc', border: '1px solid #e2e8f0', borderRadius: '8px' }}>
            <label style={{ display: 'grid', gap: '0.35rem', fontWeight: 600 }}>Title
              <input value={tocForm.title} onChange={e => setTocForm({ ...tocForm, title: e.target.value })} required style={{ padding: '0.55rem', border: '1px solid #cbd5e1', borderRadius: '4px' }} />
            </label>
            <label style={{ display: 'grid', gap: '0.35rem', fontWeight: 600 }}>Narrative
              <input value={tocForm.narrative} onChange={e => setTocForm({ ...tocForm, narrative: e.target.value })} placeholder="Describe the causal pathway" style={{ padding: '0.55rem', border: '1px solid #cbd5e1', borderRadius: '4px' }} />
            </label>
            {[
              ['inputs', 'Inputs'],
              ['activities', 'Activities'],
              ['outputs', 'Outputs'],
              ['shortTermOutcomes', 'Short-term outcomes'],
              ['longTermOutcomes', 'Long-term outcomes'],
              ['impact', 'Impact'],
              ['assumptions', 'Assumptions']
            ].map(([field, label]) => (
              <label key={field} style={{ display: 'grid', gap: '0.35rem', fontWeight: 600 }}>{label}
                <textarea value={tocForm[field]} onChange={e => setTocForm({ ...tocForm, [field]: e.target.value })} placeholder="One item per line" rows="3" style={{ padding: '0.55rem', border: '1px solid #cbd5e1', borderRadius: '4px', resize: 'vertical' }} />
              </label>
            ))}
            <div style={{ gridColumn: '1 / -1', display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
              <button type="submit" className="btn btn-primary" disabled={tocSaving}>{tocSaving ? 'Saving...' : 'Save Theory of Change'}</button>
              {tocMessage && <span role="status" style={{ color: tocMessage.includes('saved') ? '#15803d' : '#b91c1c', fontSize: '0.85rem' }}>{tocMessage}</span>}
            </div>
          </form>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: '1rem' }}>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #64748b' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#475569' }}>1. INPUTS</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                {fromLines(tocForm.inputs).map(item => <li key={item}>{item}</li>)}
              </ul>
            </div>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #3b82f6' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#2563eb' }}>2. ACTIVITIES</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                {fromLines(tocForm.activities).map(item => <li key={item}>{item}</li>)}
              </ul>
            </div>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #eab308' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#ca8a04' }}>3. OUTPUTS</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                {fromLines(tocForm.outputs).map(item => <li key={item}>{item}</li>)}
              </ul>
            </div>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #10b981' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#059669' }}>4. OUTCOMES</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                {fromLines(tocForm.shortTermOutcomes).map(item => <li key={item}>{item}</li>)}
              </ul>
            </div>
            <div style={{ background: '#f8fafc', padding: '1rem', borderRadius: '6px', borderTop: '4px solid #8b5cf6' }}>
              <h4 style={{ margin: '0 0 0.5rem', fontSize: '0.9rem', color: '#7c3aed' }}>5. IMPACT</h4>
              <ul style={{ margin: 0, paddingLeft: '1.2rem', fontSize: '0.85rem', color: '#334155' }}>
                {fromLines(tocForm.impact).map(item => <li key={item}>{item}</li>)}
              </ul>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
