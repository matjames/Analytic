import React, { useEffect, useState, useCallback } from 'react';
import { Brain, ShieldCheck, Clock, AlertCircle } from 'lucide-react';

const CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) || 'http://localhost:8096';

interface Rec { id: string; provider: string; model: string; status: string; recommendation: string; confidence: number; risk_level: string; ai_label: string; recommended_actions: string[]; limitations: string[]; reasoning_summary: string; created_at: string; reviewed_by: string; }

const STATUS_STYLES: Record<string, string> = {
  GENERATED: 'text-blue-700 bg-blue-50 border-blue-200',
  PENDING_REVIEW: 'text-amber-700 bg-amber-50 border-amber-200',
  AUTHORIZED: 'text-emerald-700 bg-emerald-50 border-emerald-200',
  REJECTED: 'text-rose-700 bg-rose-50 border-rose-200',
  EXECUTED: 'text-teal-700 bg-teal-50 border-teal-200',
  CANCELLED: 'text-gray-600 bg-gray-100 border-gray-200',
  FAILED: 'text-rose-700 bg-rose-100 border-rose-200',
};

export const AIRecommendationsView: React.FC = () => {
  const [recs, setRecs] = useState<Rec[]>([]);
  const [loading, setLoading] = useState(true);

  const headers = () => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
    return { headers: { Authorization: `Bearer ${token}` } };
  };

  const fetchAll = useCallback(async () => {
    try {
      const r = await fetch(`${CORE}/api/ai/recommendations`, headers());
      if (r.ok) { const d = await r.json(); setRecs(d.recommendations ?? []); }
    } catch (e) { console.error(e); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchAll(); }, [fetchAll]);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <span className="px-2.5 py-0.5 text-xs font-bold bg-blue-100 text-blue-800 rounded-full">Phase XII</span>
        <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2"><Brain className="w-6 h-6 text-blue-600" /> AI Recommendations</h2>
      </div>
      <p className="text-xs text-gray-500">AI is advisory only. Every recommendation requires human review (directive §19); AI never directly mutates institutional records.</p>

      <div className="bg-white border border-gray-200 rounded-2xl p-5">
        <h3 className="text-sm font-bold text-gray-800 mb-3">Recommendations ({recs.length})</h3>
        {loading ? <p className="text-sm text-gray-500">Loading…</p>
        : recs.length ? (
          <div className="space-y-3">
            {recs.map((rec) => (
              <div key={rec.id} className="border border-gray-100 rounded-xl p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className={`text-[10px] px-2 py-0.5 rounded-full border font-bold ${STATUS_STYLES[rec.status] ?? STATUS_STYLES.GENERATED}`}>{rec.status}</span>
                    <span className="text-[10px] px-2 py-0.5 rounded-full bg-purple-100 text-purple-700 font-bold">{rec.ai_label}</span>
                  </div>
                  <span className="text-xs font-mono text-gray-400">confidence {Math.round(rec.confidence * 100)}%</span>
                </div>
                <p className="text-sm text-gray-700 mt-2">{rec.recommendation || 'NO DATA: no recommendation content.'}</p>
                {rec.reasoning_summary && <p className="text-xs text-gray-500 mt-1">{rec.reasoning_summary}</p>}
                {rec.risk_level && <p className="text-xs text-gray-400 mt-1">risk: {rec.risk_level}</p>}
                {rec.recommended_actions?.length > 0 && (
                  <ul className="list-disc list-inside text-xs text-gray-600 mt-2">{rec.recommended_actions.map((a, i) => <li key={i}>{a}</li>)}</ul>
                )}
                <div className="text-[11px] text-gray-400 mt-2 font-mono">
                  {rec.id} · {rec.provider}/{rec.model} · {new Date(rec.created_at).toLocaleString()}
                </div>
                {rec.status === 'PENDING_REVIEW' && (
                  <p className="text-xs text-amber-600 font-semibold mt-1 flex items-center gap-1"><Clock className="w-3 h-3" /> Awaiting human review.</p>
                )}
              </div>
            ))}
          </div>
        ) : (<p className="text-sm text-gray-400">NO DATA: no AI recommendations for this tenant.</p>)}
      </div>

      <div className="bg-white border border-gray-200 rounded-2xl p-5">
        <h3 className="text-sm font-bold text-gray-800 mb-2 flex items-center gap-2"><AlertCircle className="w-4 h-4 text-gray-500" /> Governance Note</h3>
        <p className="text-xs text-gray-500 flex items-center gap-1"><ShieldCheck className="w-3.5 h-3.5 text-emerald-500" /> Authorization and execution are privileged actions (admin / institutional_lead) performed by human operators. Every request is written to the dedicated AI audit trail.</p>
      </div>
    </div>
  );
};
