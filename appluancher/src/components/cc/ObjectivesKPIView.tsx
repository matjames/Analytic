import React, { useEffect, useState, useCallback } from 'react';
import { Target, Gauge } from 'lucide-react';

const CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) || 'http://localhost:8096';

interface Objective { id: string; canonical_id: string; name: string; description: string; owner: string; status: string; governance_ref: string; }
interface KPI { id: string; canonical_id: string; name: string; owner: string; target: number; actual_value?: number | null; unit: string; source: string; data_status: string; measurement_period: string; }

const STATUS_COLORS: Record<string, string> = {
  ACTUAL: 'text-emerald-700 bg-emerald-50 border-emerald-200',
  ESTIMATED: 'text-amber-700 bg-amber-50 border-amber-200',
  STALE: 'text-orange-700 bg-orange-50 border-orange-200',
  MISSING: 'text-gray-600 bg-gray-100 border-gray-200',
  INVALID: 'text-rose-700 bg-rose-50 border-rose-200',
};

export const ObjectivesKPIView: React.FC = () => {
  const [objectives, setObjectives] = useState<Objective[]>([]);
  const [kpis, setKpis] = useState<KPI[]>([]);

  const headers = () => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
    return { headers: { Authorization: `Bearer ${token}` } };
  };

  const fetchAll = useCallback(async () => {
    try {
      const [o, k] = await Promise.all([
        fetch(`${CORE}/api/objectives`, headers()),
        fetch(`${CORE}/api/kpis`, headers()),
      ]);
      if (o.ok) { const d = await o.json(); setObjectives(d.objectives ?? []); }
      if (k.ok) { const d = await k.json(); setKpis(d.kpis ?? []); }
    } catch (e) { console.error(e); }
  }, []);
  useEffect(() => { fetchAll(); }, [fetchAll]);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <span className="px-2.5 py-0.5 text-xs font-bold bg-blue-100 text-blue-800 rounded-full">Phase XII</span>
        <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2"><Target className="w-6 h-6 text-blue-600" /> Objectives & KPI</h2>
      </div>
      <p className="text-xs text-gray-500">Strategic performance projection. StatGovernance remains the source of truth for governance-owned objectives. Stale data is never shown as current (directive §13).</p>

      <div className="bg-white border border-gray-200 rounded-2xl p-5">
        <h3 className="text-sm font-bold text-gray-800 mb-3">Institutional Objectives ({objectives.length})</h3>
        {objectives.length ? (
          <div className="grid md:grid-cols-2 gap-2">
            {objectives.map((o) => (
              <div key={o.id} className="border border-gray-100 rounded-xl p-3">
                <div className="flex items-center justify-between"><span className="text-sm font-semibold text-gray-800">{o.name}</span><span className="text-[10px] px-2 py-0.5 rounded-full bg-blue-50 text-blue-700">{o.status}</span></div>
                <div className="text-xs text-gray-500 mt-1">{o.description || 'No description'}</div>
                <div className="text-[11px] text-gray-400 mt-1">owner: {o.owner || '—'}</div>
                {o.governance_ref && <div className="text-[11px] text-gray-400 font-mono">governance: {o.governance_ref}</div>}
              </div>
            ))}
          </div>
        ) : (<p className="text-sm text-gray-400">NO DATA: no objectives projected for this tenant.</p>)}
      </div>

      <div className="bg-white border border-gray-200 rounded-2xl p-5">
        <h3 className="text-sm font-bold text-gray-800 mb-3 flex items-center gap-2"><Gauge className="w-4 h-4 text-gray-500" /> KPI Framework ({kpis.length})</h3>
        {kpis.length ? (
          <div className="space-y-3">
            {kpis.map((k) => {
              const sc = STATUS_COLORS[k.data_status] ?? STATUS_COLORS.MISSING;
              const pct = k.target ? Math.round(((k.actual_value ?? 0) / k.target) * 100) : null;
              return (
                <div key={k.id} className="border border-gray-100 rounded-xl p-4">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-semibold text-gray-800">{k.name}</span>
                    <span className={`text-[10px] px-2 py-0.5 rounded-full border font-bold ${sc}`}>{k.data_status}</span>
                  </div>
                  <div className="text-[11px] text-gray-400">owner: {k.owner || '—'} · period: {k.measurement_period} · source: {k.source || '—'}</div>
                  {k.data_status === 'MISSING' ? (
                    <p className="text-xs text-gray-400 mt-2">NOT CONFIGURED — no measurement recorded.</p>
                  ) : (
                    <div className="mt-2">
                      <div className="flex items-center justify-between text-xs">
                        <span className="font-mono text-gray-700">{k.actual_value ?? '—'} {k.unit}</span>
                        <span className="text-gray-400">target: {k.target} {k.unit}</span>
                      </div>
                      {pct !== null && (
                        <div className="mt-1 bg-gray-100 h-2 rounded-full overflow-hidden">
                          <div className={`h-full rounded-full ${pct >= 70 ? 'bg-emerald-500' : pct >= 40 ? 'bg-amber-500' : 'bg-rose-500'}`} style={{ width: `${Math.min(pct, 100)}%` }} />
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        ) : (<p className="text-sm text-gray-400">NO DATA: no KPIs configured for this tenant.</p>)}
      </div>
    </div>
  );
};
