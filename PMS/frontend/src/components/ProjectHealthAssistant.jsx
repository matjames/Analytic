import React, { useState } from 'react';

const actions = [
  { id: 'health_review', label: 'Run health review' },
  { id: 'risk_prediction', label: 'Predict delivery risk' },
  { id: 'schedule_optimization', label: 'Review schedule' },
  { id: 'budget_forecast', label: 'Forecast budget' },
];

export default function ProjectHealthAssistant({ project, apiBase }) {
  const [action, setAction] = useState('health_review');
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const runAssistant = () => {
    setLoading(true);
    setError('');
    fetch(`${apiBase}/api/projects/${project.id}/health-assistant`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action }),
    })
      .then(response => {
        if (!response.ok) throw new Error(`Assistant request failed: ${response.status}`);
        return response.json();
      })
      .then(setResult)
      .catch(requestError => setError(requestError.message))
      .finally(() => setLoading(false));
  };

  const assistant = result?.assistant;
  return (
    <div className="glass-panel" style={{ padding: '20px', marginBottom: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px', flexWrap: 'wrap' }}>
        <div>
          <h3 style={{ fontSize: '16px', fontWeight: '800', marginBottom: '5px' }}>AI Project Health Assistant</h3>
          <p style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Evidence-based advisory from the current project budget and delivery state.</p>
        </div>
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          <select value={action} onChange={event => setAction(event.target.value)} aria-label="Assistant action">
            {actions.map(item => <option key={item.id} value={item.id}>{item.label}</option>)}
          </select>
          <button className="btn btn-primary" onClick={runAssistant} disabled={loading}>{loading ? 'Analyzing...' : 'Generate advisory'}</button>
        </div>
      </div>

      {error && <p style={{ color: 'var(--accent-danger)', fontSize: '12px', marginTop: '12px' }}>{error}</p>}
      {assistant && (
        <div style={{ borderTop: '1px solid var(--border-light)', marginTop: '16px', paddingTop: '16px' }}>
          <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap', marginBottom: '10px' }}>
            <span className="badge badge-Implementation">Predicted risk: {assistant.predicted_risk}</span>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Confidence: {Math.round(Number(assistant.confidence || 0) * 100)}%</span>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{assistant.forecast_variance}</span>
          </div>
          {assistant.forecast && (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '8px', marginBottom: '12px' }}>
              <div className="metric-card"><span>Projected total</span><strong>${Number(assistant.forecast.projected_total || 0).toLocaleString(undefined, { maximumFractionDigits: 0 })}</strong></div>
              <div className="metric-card"><span>Forecast variance</span><strong>{Number(assistant.forecast.variance_percent || 0).toFixed(1)}%</strong></div>
              <div className="metric-card"><span>Budget consumed</span><strong>{Number(assistant.forecast.utilization_percent || 0).toFixed(1)}%</strong></div>
            </div>
          )}
          <ul style={{ margin: '0 0 10px 18px', color: 'var(--text-secondary)', fontSize: '12px', lineHeight: '1.7' }}>
            {(assistant.recommendations || []).map(recommendation => <li key={recommendation}>{recommendation}</li>)}
          </ul>
          <p style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{result.governanceNotice}</p>
        </div>
      )}
    </div>
  );
}
