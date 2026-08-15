import React, { useEffect, useState, useCallback } from 'react';
import { Activity, Info } from 'lucide-react';

const CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) || 'http://localhost:8096';

interface Sig { signal_id: string; signal_type: string; domain: string; severity: number; evidence: string; }
interface Dom { domain: string; severity: number; signal_count: number; }
interface Condition {
  condition_level: string; composite_score: number; calculation_timestamp: string;
  supporting_signals: Sig[]; affected_domains: Dom[]; explanation: string;
  calculation_version: string; correlation_id: string;
}

function badge(level: string): string {
  switch (level) {
    case 'EMERGENCY': return 'text-red-700 border-red-200 bg-red-100';
    case 'CRITICAL': case 'ELEVATED': return 'text-rose-700 border-rose-200 bg-rose-50';
    case 'ATTENTION': return 'text-amber-700 border-amber-200 bg-amber-50';
    case 'OPTIMAL': case 'NOMINAL': return 'text-emerald-700 border-emerald-200 bg-emerald-50';
    default: return 'text-gray-600 border-gray-200 bg-gray-100';
  }
}

export const InstitutionalConditionView: React.FC = () => {
  const [cond, setCond] = useState<Condition | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchIt = useCallback(async () => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
    try { const r = await fetch(`${CORE}/api/intelligence/condition`, { headers: { Authorization: `Bearer ${token}` } }); if (r.ok) setCond(await r.json()); }
    catch (e) { console.error(e); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchIt(); }, [fetchIt]);

  const level = cond?.condition_level ?? 'UNKNOWN';
  const st = badge(level);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <span className="px-2.5 py-0.5 text-xs font-bold bg-blue-100 text-blue-800 rounded-full">Phase XII</span>
        <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
          <Activity className="w-6 h-6 text-blue-600" /> Institutional Condition
        </h2>
      </div>

      {loading ? <p className="text-sm text-gray-500">Loading condition…</p>
      : cond ? (
        <>
          <div className={`border rounded-2xl p-6 ${st}`}>
            <div className="flex items-center gap-3">
              <div className="flex-1">
                <div className="text-2xl font-bold">{cond.condition_level}</div>
                <div className="text-sm">Composite severity: {cond.composite_score}/100</div>
              </div>
              <div className="text-right text-xs text-gray-500">
                <div className="font-mono">calc v{cond.calculation_version}</div>
                <div>{new Date(cond.calculation_timestamp).toLocaleString()}</div>
              </div>
            </div>
          </div>

          <div className="bg-white border border-gray-200 rounded-2xl p-6">
            <h3 className="text-sm font-bold text-gray-800 mb-2">Why is this the current condition?</h3>
            <p className="text-sm text-gray-600 leading-relaxed">{cond.explanation}</p>
          </div>

          <div className="bg-white border border-gray-200 rounded-2xl p-6">
            <h3 className="text-sm font-bold text-gray-800 mb-3">Supporting Signals (Evidence)</h3>
            {cond.supporting_signals?.length ? (
              <div className="space-y-2">
                {cond.supporting_signals.map((s, i) => (
                  <div key={i} className="flex items-start gap-3 border-b border-gray-100 pb-2 last:border-0">
                    <span className={`mt-0.5 px-2 py-0.5 rounded-full text-[11px] font-bold ${s.severity >= 70 ? 'bg-rose-100 text-rose-700' : s.severity >= 40 ? 'bg-amber-100 text-amber-700' : 'bg-emerald-100 text-emerald-700'}`}>{s.severity}</span>
                    <div>
                      <div className="text-sm font-semibold text-gray-800">{s.signal_type} <span className="text-gray-400">·</span> {s.domain}</div>
                      <div className="text-xs text-gray-500">{s.evidence}</div>
                    </div>
                  </div>
                ))}
              </div>
            ) : <p className="text-sm text-gray-400">NO DATA: no supporting signals recorded.</p>}
          </div>
        </>
      ) : (
        <div className="bg-gray-50 border border-dashed border-gray-300 rounded-2xl p-8 text-center">
          <Info className="w-6 h-6 text-gray-400 mx-auto mb-2" />
          <p className="text-sm text-gray-500 font-semibold">NO DATA</p>
          <p className="text-xs text-gray-400">No institutional condition computed yet. It is derived only from real intelligence signals, never manually entered.</p>
        </div>
      )}
    </div>
  );
};
