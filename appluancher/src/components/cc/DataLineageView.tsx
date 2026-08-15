import React, { useState, useCallback } from 'react';
import { GitBranch, Search, FileSearch } from 'lucide-react';

const CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) || 'http://localhost:8096';

interface Path { hops: number; nodes: string[]; }
interface QueryRes { query: string; canonical_id?: string; count: number; paths: Path[]; notes?: string[]; }

export const DataLineageView: React.FC = () => {
  const [traceId, setTraceId] = useState('');
  const [res, setRes] = useState<QueryRes | null>(null);

  const headers = () => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
    return { headers: { Authorization: `Bearer ${token}` } };
  };

  const trace = useCallback(async (id?: string) => {
    const target = id ?? traceId;
    if (!target) return;
    try {
      const r = await fetch(`${CORE}/api/graph/queries/object-neighborhood/${encodeURIComponent(target)}`, headers());
      if (r.ok) setRes(await r.json());
    } catch (e) { console.error(e); }
  }, [traceId]);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <span className="px-2.5 py-0.5 text-xs font-bold bg-blue-100 text-blue-800 rounded-full">Phase XII</span>
        <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2"><GitBranch className="w-6 h-6 text-blue-600" /> Data Lineage</h2>
      </div>
      <p className="text-xs text-gray-500">Reuses the Phase IX lineage infrastructure and the Phase XII knowledge graph to trace the full chain: KPI → Dataset → Transformation → Source, and Report → Indicator → Dataset → Research Study (directive §23).</p>

      <div className="bg-white border border-gray-200 rounded-2xl p-5">
        <div className="flex gap-2">
          <input value={traceId} onChange={(e) => setTraceId(e.target.value)} placeholder="Canonical ID to trace (e.g. uoi_kpi_…)" className="flex-1 text-sm border border-gray-200 rounded-lg px-3 py-2" />
          <button onClick={() => trace()} className="px-4 py-2 text-sm font-semibold bg-blue-600 text-white rounded-lg flex items-center gap-2"><Search className="w-4 h-4" /> Trace</button>
        </div>
      </div>

      {res && (
        <div className="bg-white border border-gray-200 rounded-2xl p-5">
          <h3 className="text-sm font-bold text-gray-800 mb-3 flex items-center gap-2"><FileSearch className="w-4 h-4 text-gray-500" /> Lineage Trace</h3>
          {res.notes?.length ? (
            res.notes.map((n, i) => <p key={i} className="text-xs text-amber-600 font-semibold">{n}</p>)
          ) : (
            <div className="space-y-2">
              {res.paths.map((p, i) => (
                <div key={i} className="border border-gray-100 rounded-lg p-3">
                  <div className="flex flex-col gap-1">
                    {p.nodes.map((n, j) => (
                      <div key={j} className="flex items-center gap-2">
                        {j > 0 && <span className="text-gray-300 text-xs">▼</span>}
                        <span className="text-xs bg-gray-50 border border-gray-200 px-2 py-1 rounded font-mono">{n}</span>
                      </div>
                    ))}
                  </div>
                  <div className="text-[10px] text-gray-400 mt-1">{p.hops} hop(s)</div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
