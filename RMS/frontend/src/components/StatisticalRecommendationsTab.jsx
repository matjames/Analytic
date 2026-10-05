import React, { useEffect, useState } from 'react';

function RecommendationCard({ recommendation }) {
  return <article className="data-card">
    <div>
      <strong>{recommendation.method}</strong>
      <p>{recommendation.rationale}</p>
      <p><strong>Data requirements:</strong> {recommendation.dataRequirements.join(' | ')}</p>
      <p><strong>Assumptions:</strong> {recommendation.assumptions.join(' | ')}</p>
    </div>
    <span className="badge">{recommendation.priority}</span>
  </article>;
}

export default function StatisticalRecommendationsTab({ apiBase, researchId }) {
  const [result, setResult] = useState(null);
  const [datasetId, setDatasetId] = useState('');
  const [outcomeType, setOutcomeType] = useState('continuous');
  const [question, setQuestion] = useState('');
  const [rowVariable, setRowVariable] = useState('');
  const [colVariable, setColVariable] = useState('');
  const [aggregation, setAggregation] = useState('COUNT');
  const [tabulation, setTabulation] = useState(null);
  const [tabulationError, setTabulationError] = useState('');
  const [tabulationLoading, setTabulationLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const load = () => {
    if (!researchId) return setResult(null);
    setLoading(true);
    setError('');
    fetch(`${apiBase}/api/research/${researchId}/statistical-recommendations`)
      .then(async response => {
        const body = await response.json();
        if (!response.ok) throw new Error(body.error || 'Recommendations unavailable.');
        return body;
      })
      .then(body => {
        setResult(body);
        if (!datasetId && body.datasets?.length) setDatasetId(body.datasets[0].id);
      })
      .catch(loadError => setError(loadError.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    setDatasetId('');
    setOutcomeType('continuous');
    setQuestion('');
    setRowVariable('');
    setColVariable('');
    setTabulation(null);
    setTabulationError('');
    load();
  }, [researchId]);

  const generate = (event) => {
    event.preventDefault();
    setLoading(true);
    setError('');
    fetch(`${apiBase}/api/research/${researchId}/statistical-recommendations`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ datasetId, outcomeType, question }),
    }).then(async response => {
      const body = await response.json();
      if (!response.ok) throw new Error(body.error || 'Recommendations unavailable.');
      return body;
    }).then(setResult).catch(generateError => setError(generateError.message)).finally(() => setLoading(false));
  };

  const runTabulation = (event) => {
    event.preventDefault();
    setTabulationLoading(true);
    setTabulationError('');
    fetch(`${apiBase}/api/research/${researchId}/statistical-recommendations/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ datasetId, rowVariable, colVariable, aggregation }),
    }).then(async response => {
      const body = await response.json();
      if (!response.ok) throw new Error(body.error || 'Analytics Core execution failed.');
      return body;
    }).then(setTabulation).catch(error => {
      setTabulation(null);
      setTabulationError(error.message);
    }).finally(() => setTabulationLoading(false));
  };

  if (!researchId) return <div className="empty-state">Select a research study to plan statistical analysis.</div>;
  if (loading && !result) return <div className="empty-state">Preparing transparent statistical recommendations...</div>;

  return <div>
    <div className="section-heading">
      <div><h2>Statistical Analysis Planning</h2><p>Transparent planning guidance from registered dataset metadata. Recommendations are not executed tests and require statistician review.</p></div>
      <button className="btn btn-primary" onClick={load}>Refresh Plan</button>
    </div>
    <form onSubmit={generate} className="inline-form">
      <label>Dataset<select value={datasetId} onChange={event => setDatasetId(event.target.value)}><option value="">All registered datasets</option>{(result?.datasets || []).map(dataset => <option key={dataset.id} value={dataset.id}>{dataset.name} ({dataset.status})</option>)}</select></label>
      <label>Outcome type<select value={outcomeType} onChange={event => setOutcomeType(event.target.value)}><option value="continuous">Continuous</option><option value="binary">Binary</option><option value="categorical">Categorical</option><option value="count">Count</option><option value="time-to-event">Time to event</option></select></label>
      <label>Research question<textarea value={question} onChange={event => setQuestion(event.target.value)} rows="4" maxLength="2000" placeholder="What outcome, exposure, or comparison should the analysis address?" /></label>
      <div><button className="btn btn-primary" type="submit" disabled={loading}>{loading ? 'Preparing...' : 'Generate Plan'}</button></div>
    </form>
    {error && <div className="data-card"><strong>Planning unavailable</strong><p>{error}</p></div>}
    {result && <>
      <div className="data-card"><div><strong>{result.researchName}</strong><p>{result.datasets.length} registered dataset(s) considered.</p></div><span className="badge">{result.mode}</span></div>
      <div className="data-card"><strong>Analytics Core handoff: {result.analyticsCore.status}</strong><p>{result.analyticsCore.nextStep}</p></div>
      <div className="data-list">{result.recommendations.map(recommendation => <RecommendationCard key={recommendation.key} recommendation={recommendation} />)}</div>
      <div className="section-heading"><div><h3>Run a registered tabulation</h3><p>This sends only the selected durable dataset identifier and variables to Analytics Core. It does not accept raw request data.</p></div></div>
      <form onSubmit={runTabulation} className="inline-form">
        <label>Row variable<input value={rowVariable} onChange={event => setRowVariable(event.target.value)} required placeholder="e.g. sex" /></label>
        <label>Column variable<input value={colVariable} onChange={event => setColVariable(event.target.value)} placeholder="Optional, e.g. outcome" /></label>
        <label>Aggregation<select value={aggregation} onChange={event => setAggregation(event.target.value)}><option value="COUNT">Count</option><option value="SUM">Sum</option><option value="MEAN">Mean</option><option value="PERCENTAGE">Percentage</option></select></label>
        <div><button className="btn btn-primary" type="submit" disabled={tabulationLoading || !datasetId}>{tabulationLoading ? 'Running...' : 'Run in Analytics Core'}</button></div>
      </form>
      {tabulationError && <div className="data-card"><strong>Analytics Core unavailable</strong><p>{tabulationError}</p></div>}
      {tabulation && <div className="data-card"><div><strong>Analytics Core result</strong><p>{tabulation.result?.title || 'Tabulation completed'} | {tabulation.result?.total_record_count || 0} records</p></div><span className="badge">{tabulation.mode}</span></div>}
      <small>{result.disclaimer}</small>
    </>}
  </div>;
}

