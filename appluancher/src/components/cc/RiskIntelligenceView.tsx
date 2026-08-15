import React, { useEffect, useState, useCallback } from 'react';
import { ShieldAlert, Radar } from 'lucide-react';

const CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) || 'http://localhost:8096';

interface RiskEvent { id: string; event_type: string; source_system: string; severity_estimate: number; risk_reference: string; correlated_objects: string[]; evidence: string; status: string; created_at: string; }

export const RiskIntelligenceView: React.FC = () => {
  const [events, setEvents] = useState<RiskEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const headers = () => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
    return { headers: { Authorization: `Bearer ${token}` } };
  };

  const fetchAll = useCallback(async () => {
    try {
      const r = await fetch(`${CORE}/api/risks/events`, headers());
      if (r.ok) { const d = await r.json(); setEvents(d.events ?? []); }
    } catch (e) { console.error(e); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchAll(); }, [fetchAll]);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <span className="px-2.5 py-0.5 text-xs font-bold bg-blue-100 text-blue-800 rounded-full">Phase XII</span>
        <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2"><ShieldAlert className="w-6 h-6 text-blue-600" /> Risk Intelligence</h2>
      </div>
      <p className="text-xs text-gray-500">Risk intelligence correlation layer. StatGovernance remains the authoritative risk register; Phase XII correlates events — it does not create a competing register (directive §14). Emerging risks surface as intelligence signals for human governance.</p>

      <div className="bg-white border border-gray-200 rounded-2xl p-5">
        <h3 className="text-sm font-bold text-gray-800 mb-3 flex items-center gap-2"><Radar className="w-4 h-4 text-gray-500" /> Correlated Risk Events ({events.length})</h3>
        {loading ? <p className="text-sm text-gray-500">Loading…</p>
        : events.length ? (
          <div className="space-y-2">
            {events.map((e) => (
              <div key={e.id} className="border border-gray-100 rounded-xl p-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold text-gray-800 ">{e.event_type}</span>
                  <span className={`px-2 py-0.5 rounded-full text-xs font-bold ${e.severity_estimate >= 70 ? 'bg-rose-100 text-rose-700' : e.severity_estimate >= 40 ? 'bg-amber-100 text-amber-700' : 'bg-emerald-100 text-emerald-700'}`}>{e.severity_estimate}</span>
                </div>
                <div className="text-xs text-gray-500 mt-1">{e.evidence}</div>
                <div className="text-[11px] text-gray-400 mt-1">source: {e.source_system} · status: {e.status} · created: {new Date(e.created_at).toLocaleString()}</div>
                {e.risk_reference && <div className="text-[11px] text-gray-400 font-mono">register ref: {e.risk_reference}</div>}
                {e.correlated_objects?.length > 0 && (
                  <div className="flex flex-wrap gap-1 mt-1">{e.correlated_objects.map((o, i) => <span key={i} className="text-[10px] px-1.5 py-0.5 bg-gray-100 rounded font-mono">{o}</span>)}</div>
                )}
              </div>
            ))}
          </div>
        ) : (<p className="text-sm text-gray-400">NO DATA: no correlated risk events for this tenant. Formal risk records continue to live in StatGovernance.</p>)}
      </div>
    </div>
  );
};
