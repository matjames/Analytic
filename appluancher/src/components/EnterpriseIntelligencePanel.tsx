import React, { FormEvent, useEffect, useMemo, useState } from 'react';
import { User } from '@typings/index';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

type Finding = { kind: 'fact' | 'inference' | 'recommendation' | 'uncertainty'; statement: string; confidence: string; source_ids?: string[] };
type Source = { id: string; app: string; entity: string; record_id: string; title: string; timestamp: string; deep_link?: string };
type AIResponse = { id: string; status: string; findings: Finding[]; sources: Source[] };
type IntelligenceContext = { application?: string; object_type?: string; object_id?: string; project_id?: string };

export const EnterpriseIntelligencePanel: React.FC<{ user: User; context?: IntelligenceContext }> = ({ user, context }) => {
  const [question, setQuestion] = useState('What requires my attention today?');
  const [response, setResponse] = useState<AIResponse | null>(null);
  const [briefing, setBriefing] = useState<Finding[]>([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  const headers = useMemo(() => {
    const token = typeof window === 'undefined' ? null : localStorage.getItem('registry_jwt');
    return { 'Content-Type': 'application/json', 'X-User-ID': user.id, ...(token ? { Authorization: `Bearer ${token}` } : {}) };
  }, [user.id]);

  useEffect(() => {
    fetch(`${API_BASE}/api/ai/v1/briefings`, { headers })
      .then((r) => r.ok ? r.json() : null)
      .then((data) => setBriefing(data?.findings || []))
      .catch(() => setBriefing([]));
  }, [headers]);

  const ask = async (event: FormEvent) => {
    event.preventDefault();
    if (!question.trim()) return;
    setLoading(true); setMessage('');
    try {
      const result = await fetch(`${API_BASE}/api/ai/v1/query`, { method: 'POST', headers, body: JSON.stringify({ question, ...context }) });
      const data = await result.json();
      if (!result.ok) throw new Error(data.error || 'The intelligence request could not be completed.');
      setResponse(data);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'The intelligence request could not be completed.');
    } finally { setLoading(false); }
  };

  const createInvestigation = async () => {
    if (!response) return;
    setMessage('');
    try {
      const result = await fetch(`${API_BASE}/api/ai/v1/investigations`, {
        method: 'POST', headers,
        body: JSON.stringify({ title: question, question, response_id: response.id, assignee: user.id, project_id: context?.project_id }),
      });
      const data = await result.json();
      if (!result.ok) throw new Error(data.error || 'Unable to create investigation.');
      setMessage(`Investigation ${data.id} created. A task has been assigned for human follow-up.`);
    } catch (error) { setMessage(error instanceof Error ? error.message : 'Unable to create investigation.'); }
  };

  return (
    <section className="glass-panel" aria-label="Enterprise intelligence">
      <div className="flex items-start justify-between gap-4 mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">Enterprise Intelligence</h3>
          <p className="text-sm text-gray-500">Evidence-first decision support across your authorized StatGate data{context?.object_id ? ` for ${context.object_type || 'this object'}` : ''}.</p>
        </div>
        <span className="text-xs px-2 py-1 rounded-full bg-blue-50 text-blue-700">Governed AI</span>
      </div>
      {briefing.length > 0 && <div className="mb-4 rounded-lg border border-blue-100 bg-blue-50 p-3">
        <div className="text-xs font-semibold uppercase tracking-wide text-blue-700 mb-1">My daily briefing</div>
        {briefing.map((item, index) => <p className="text-sm text-blue-900" key={`${item.statement}-${index}`}>{item.statement}</p>)}
      </div>}
      <form onSubmit={ask} className="flex gap-2">
        <input value={question} onChange={(e) => setQuestion(e.target.value)} className="min-w-0 flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm" aria-label="Ask Enterprise Intelligence" />
        <button type="submit" disabled={loading} className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-60">{loading ? 'Analyzing…' : 'Analyze'}</button>
      </form>
      {message && <p className="mt-3 text-sm text-gray-600" role="status">{message}</p>}
      {response && <div className="mt-4 space-y-3">
        <div className="flex items-center justify-between"><span className="text-xs font-semibold uppercase tracking-wide text-gray-500">Evidence state: {response.status.replace(/_/g, ' ')}</span>{response.sources.length > 0 && <button type="button" onClick={createInvestigation} className="text-sm font-medium text-blue-700 hover:text-blue-900">Create investigation</button>}</div>
        {response.findings.map((finding, index) => <div key={`${finding.kind}-${index}`} className="rounded-lg border border-gray-200 p-3"><div className="text-xs font-semibold uppercase tracking-wide text-gray-500">{finding.kind} · {finding.confidence.replace(/_/g, ' ')}</div><p className="mt-1 text-sm text-gray-800">{finding.statement}</p></div>)}
        {response.sources.length > 0 && <details className="rounded-lg border border-gray-200 p-3"><summary className="cursor-pointer text-sm font-medium text-gray-700">View evidence ({response.sources.length})</summary><ul className="mt-2 space-y-1 text-xs text-gray-600">{response.sources.map((source) => <li key={source.id}>{source.deep_link ? <a href={source.deep_link} className="text-blue-700 hover:underline"><span className="font-medium">{source.app}</span> / {source.entity} / {source.record_id}</a> : <><span className="font-medium">{source.app}</span> / {source.entity} / {source.record_id}</>}{source.title ? ` — ${source.title}` : ''}</li>)}</ul></details>}
      </div>}
    </section>
  );
};
