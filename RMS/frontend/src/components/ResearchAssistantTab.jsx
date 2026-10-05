import React, { useEffect, useState } from 'react';

export default function ResearchAssistantTab({ apiBase, researchId }) {
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [task, setTask] = useState('proposal');
  const [prompt, setPrompt] = useState('');
  const [assistantResult, setAssistantResult] = useState(null);
  const [assistantError, setAssistantError] = useState('');
  const [assistantLoading, setAssistantLoading] = useState(false);
  const load = () => {
    if (!researchId) return setResult(null);
    setLoading(true);
    fetch(`${apiBase}/api/research/${researchId}/assistant`).then(r => r.json()).then(setResult).catch(() => setResult(null)).finally(() => setLoading(false));
  };
  useEffect(load, [researchId]);
  useEffect(() => {
    setAssistantResult(null);
    setAssistantError('');
  }, [researchId]);

  const askAssistant = (event) => {
    event.preventDefault();
    if (!researchId) return;
    setAssistantLoading(true);
    setAssistantError('');
    fetch(`${apiBase}/api/research/${researchId}/assistant`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ task, prompt }),
    }).then(async response => {
      const body = await response.json();
      if (!response.ok) throw new Error(body.error || 'Research assistant is unavailable.');
      return body;
    }).then(setAssistantResult).catch(error => {
      setAssistantResult(null);
      setAssistantError(error.message);
    }).finally(() => setAssistantLoading(false));
  };

  if (loading) return <div className="empty-state">Generating readiness guidance...</div>;
  if (!result) return <div className="empty-state">Select a research study to generate guidance.</div>;
  return <div>
    <div className="section-heading"><div><h2>Research Readiness Assistant</h2><p>Evidence-based workflow guidance from the records in this study. Deterministic guidance is always available; provider-backed assistance is optional and grounded in the same workspace records.</p></div><button className="btn btn-primary" onClick={load}>Refresh Guidance</button></div>
    <div className="data-card"><div><strong>{result.readiness}</strong><p>{result.researchName} | {result.stage} | {Math.round(result.progress || 0)}% complete</p></div><span className="badge">{result.mode}</span></div>
    <div className="data-list">{(result.recommendations || []).map((recommendation, index) => <article className="data-card" key={`${recommendation}-${index}`}><strong>Next action {index + 1}</strong><p>{recommendation}</p></article>)}</div>
    <div className="data-card"><strong>Evidence used</strong><p>Pending proposals: {result.evidence?.pendingProposals || 0} | Pending ethics: {result.evidence?.pendingEthics || 0} | Overdue tasks: {result.evidence?.overdueTasks || 0} | Datasets: {result.evidence?.datasets || 0} | Publications: {result.evidence?.publications || 0}</p></div>
    <div className="section-heading"><div><h3>Provider-backed research assistance</h3><p>Ask for a proposal plan, literature synthesis, or gap analysis. The response is advisory and must be checked by qualified researchers.</p></div></div>
    <form onSubmit={askAssistant} className="inline-form">
      <label>Task<select value={task} onChange={event => setTask(event.target.value)}><option value="proposal">Proposal planning</option><option value="literature">Literature synthesis</option><option value="gap-analysis">Research gap analysis</option></select></label>
      <label>Researcher request<textarea value={prompt} onChange={event => setPrompt(event.target.value)} rows="5" maxLength="4000" placeholder="What should the assistant help you plan or review?" /></label>
      <div><button className="btn btn-primary" type="submit" disabled={assistantLoading}>{assistantLoading ? 'Generating...' : 'Ask Assistant'}</button></div>
    </form>
    {assistantError && <div className="data-card"><strong>Assistant unavailable</strong><p>{assistantError}</p><small>Configure RMS_LLM_API_URL and the approved provider credentials before using provider-backed assistance.</small></div>}
    {assistantResult && <div className="data-list"><article className="data-card"><div><strong>{assistantResult.task}</strong><p>{assistantResult.output}</p></div><span className="badge">{assistantResult.model}</span></article><small>{assistantResult.disclaimer}</small></div>}
  </div>;
}
