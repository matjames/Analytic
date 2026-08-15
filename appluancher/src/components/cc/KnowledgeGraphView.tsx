import React, { useEffect, useState, useCallback } from 'react';
import { Network, Share2, Search } from 'lucide-react';

const CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) || 'http://localhost:8096';

interface Obj { canonical_id: string; object_type: string; source_system: string; display_name: string; projection_status: string; }
interface EdgeRef { id: string; relationship_type: string; from: string; to: string; }
interface Path { hops: number; nodes: string[]; edges: EdgeRef[]; }
interface QueryRes { query: string; canonical_id?: string; count: number; paths: Path[]; notes?: string[]; }

export const KnowledgeGraphView: React.FC = () => {
  const [objects, setObjects] = useState<Obj[]>([]);
  const [edges, setEdges] = useState<EdgeRef[]>([]);
  const [query, setQuery] = useState<QueryRes | null>(null);
  const [searchId, setSearchId] = useState('');

  const headers = () => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
    return { headers: { Authorization: `Bearer ${token}` } };
  };

  const fetchGraph = useCallback(async () => {
    try {
      const [o, e] = await Promise.all([
        fetch(`${CORE}/api/graph/objects`, headers()),
        fetch(`${CORE}/api/graph/edges`, headers()),
      ]);
      if (o.ok) { const d = await o.json(); setObjects(d.objects ?? []); }
      if (e.ok) { const d = await e.json(); setEdges(d.edges ?? []); }
    } catch (err) { console.error(err); }
  }, []);
  useEffect(() => { fetchGraph(); }, [fetchGraph]);

  const runQuery = async (name: string, id?: string) => {
    try {
      const suffix = id ? `/queries/${name}/${encodeURIComponent(id)}` : '/queries/objectives-at-risk';
      const r = await fetch(`${CORE}/api/graph${suffix}`, headers());
      if (r.ok) setQuery(await r.json());
    } catch (err) { console.error(err); }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <span className="px-2.5 py-0.5 text-xs font-bold bg-blue-100 text-blue-800 rounded-full">Phase XII</span>
        <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2"><Network className="w-6 h-6 text-blue-600" /> Knowledge Graph</h2>
      </div>
      <p className="text-xs text-gray-500">Named, parameterized queries only (directive §27). Arbitrary expressions are never accepted.</p>

      <div className="bg-white border border-gray-200 rounded-2xl p-5">
        <h3 className="text-sm font-bold text-gray-800 mb-3 flex items-center gap-2"><Share2 className="w-4 h-4 text-gray-500" /> Named Queries</h3>
        <div className="flex flex-wrap gap-2 mb-3">
          <button onClick={() => runQuery('objectives-at-risk')} className="px-3 py-1.5 text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-200 rounded-lg hover:bg-blue-100">objectives-at-risk</button>
          <button onClick={() => runQuery('projects-affected-by-incident', searchId)} className="px-3 py-1.5 text-xs font-semibold bg-gray-50 text-gray-700 border border-gray-200 rounded-lg hover:bg-gray-100">projects-affected-by-incident</button>
          <button onClick={() => runQuery('datasets-supporting-kpi', searchId)} className="px-3 py-1.5 text-xs font-semibold bg-gray-50 text-gray-700 border border-gray-200 rounded-lg hover:bg-gray-100">datasets-supporting-kpi</button>
          <button onClick={() => runQuery('object-neighborhood', searchId)} className="px-3 py-1.5 text-xs font-semibold bg-gray-50 text-gray-700 border border-gray-200 rounded-lg hover:bg-gray-100">object-neighborhood</button>
        </div>
        <div className="flex gap-2">
          <input value={searchId} onChange={(e) => setSearchId(e.target.value)} placeholder="Canonical ID" className="flex-1 text-sm border border-gray-200 rounded-lg px-3 py-2" />
          <button onClick={() => runQuery('object-neighborhood', searchId)} className="px-4 py-2 text-sm font-semibold bg-blue-600 text-white rounded-lg flex items-center gap-2"><Search className="w-4 h-4" /> Inspect</button>
        </div>
      </div>

      {query && (
        <div className="bg-white border border-gray-200 rounded-2xl p-5">
          <h3 className="text-sm font-bold text-gray-800 mb-2">Result: {query.query}</h3>
          {query.notes?.length ? (
            query.notes.map((n, i) => <p key={i} className="text-xs text-amber-600 font-semibold">{n}</p>)
          ) : (
            <div className="space-y-1 max-h-72 overflow-y-auto">
              {query.paths.map((p, i) => (
                <div key={i} className="text-xs text-gray-600 border border-gray-100 rounded-lg p-2 flex flex-wrap gap-1 items-center">
                  {p.nodes.map((n, j) => (
                    <span key={j} className="flex items-center gap-1"><span className="bg-gray-100 px-1.5 py-0.5 rounded font-mono">{n}</span>{j < p.nodes.length - 1 && <span className="text-gray-400">→</span>}</span>
                  ))}
                  <span className="ml-auto text-[10px] text-gray-400">{p.hops} hop(s)</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      <div className="bg-white border border-gray-200 rounded-2xl p-5">
        <h3 className="text-sm font-bold text-gray-800 mb-3">Institutional Object Registry ({objects.length})</h3>
        {objects.length ? (
          <div className="grid md:grid-cols-2 gap-2">
            {objects.map((o) => (
              <div key={o.canonical_id} className="border border-gray-100 rounded-xl p-3">
                <div className="flex items-center justify-between"><span className="text-sm font-semibold text-gray-800">{o.display_name || o.canonical_id}</span><span className="text-[10px] px-2 py-0.5 rounded-full bg-blue-50 text-blue-700">{o.object_type}</span></div>
                <div className="text-[11px] text-gray-400 font-mono mt-1">{o.canonical_id}</div>
                <div className="text-[11px] text-gray-500">source: {o.source_system} · {o.projection_status}</div>
              </div>
            ))}
          </div>
        ) : (<p className="text-sm text-gray-400">NO DATA: no institutional objects projected for this tenant.</p>)}
      </div>

      <div className="bg-white border border-gray-200 rounded-2xl p-5">
        <h3 className="text-sm font-bold text-gray-800 mb-3">Graph Edges ({edges.length})</h3>
        {edges.length ? (
          <div className="space-y-1 max-h-72 overflow-y-auto">
            {edges.map((e) => (
              <div key={e.id} className="text-xs text-gray-600 border-b border-gray-100 pb-1.5"><span className="font-mono">{e.from}</span><span className="mx-2 px-1.5 py-0.5 rounded bg-gray-100 text-gray-700 font-semibold">{e.relationship_type}</span><span className="font-mono">{e.to}</span></div>
            ))}
          </div>
        ) : (<p className="text-sm text-gray-400">NO DATA: no graph edges for this tenant.</p>)}
      </div>
    </div>
  );
};
