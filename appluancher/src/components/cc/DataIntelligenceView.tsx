import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { DataLineageModal } from '@components/DataLineageModal';
import { Database, Activity, AlertTriangle, ArrowUpRight, GitCommit } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const DataIntelligenceView: React.FC<{ user: User }> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [selectedLineage, setSelectedLineage] = useState<{ title: string; val: any } | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    fetch(`${API_BASE}/api/command-centre/data-intelligence?user_id=${encodeURIComponent(user.id)}`, {
      headers,
    })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((res) => {
        if (cancelled) return;
        setData(res);
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [user.id, user.role]);

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-28 bg-gray-100 rounded-xl" />
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="h-72 bg-gray-100 rounded-xl" />
          <div className="h-72 bg-gray-100 rounded-xl" />
        </div>
      </div>
    );
  }

  const kpis = data?.kpis || [];
  const alerts = data?.alerts || [];
  const anomalies = data?.anomalies || [];

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <Database className="w-6 h-6 text-cyan-600" />
            Data Intelligence & Lineage Engine
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Governed statistical intelligence, KPI monitoring, automated anomaly detection, and end-to-end evidence lineage.
          </p>
        </div>
        <a
          href="http://localhost:5000"
          className="btn btn-primary text-xs bg-cyan-700 hover:bg-cyan-800"
        >
          StatGate Analytics Workspace <ArrowUpRight className="w-3.5 h-3.5" />
        </a>
      </div>

      {/* Grid: KPIs, Anomalies, Alerts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* KPI Definitions */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <Activity className="w-5 h-5 text-cyan-600" />
              Governed KPIs ({kpis.length})
            </h3>
            <span className="text-xs text-gray-400">Click to trace lineage</span>
          </div>

          {kpis.length === 0 ? (
            <p className="text-sm text-gray-500 py-8 text-center">No KPI definitions available.</p>
          ) : (
            <div className="divide-y divide-gray-100">
              {kpis.slice(0, 8).map((kpi: any) => (
                <div
                  key={kpi.id}
                  onClick={() => setSelectedLineage({ title: kpi.name, val: kpi.scope || 'Operational' })}
                  className="py-3 flex items-center justify-between hover:bg-cyan-50/50 p-2 rounded-lg cursor-pointer transition-colors group"
                >
                  <div>
                    <div className="text-sm font-medium text-gray-900 group-hover:text-cyan-800 transition-colors">
                      {kpi.name}
                    </div>
                    <div className="text-xs text-gray-500">
                      {kpi.category} • Formula: {kpi.formula || 'Standard Ingestion Aggregate'}
                    </div>
                  </div>
                  <div className="text-right">
                    <span className="text-xs px-2 py-0.5 rounded bg-cyan-50 text-cyan-700 font-semibold group-hover:bg-cyan-100 flex items-center gap-1">
                      Lineage <GitCommit className="w-3 h-3" />
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Anomaly Detection Stream */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-amber-500" />
              Detected Anomalies ({anomalies.length})
            </h3>
          </div>

          {anomalies.length === 0 ? (
            <p className="text-sm text-gray-500 py-8 text-center">
              No anomalies detected in the last 24 hours. Statistical models within confidence bounds.
            </p>
          ) : (
            <div className="space-y-3">
              {anomalies.slice(0, 6).map((an: any) => (
                <div
                  key={an.id}
                  onClick={() => openObjectContext('anomaly', an.id, { title: an.type || 'Statistical Anomaly' })}
                  className="p-3 rounded-lg border border-amber-200 bg-amber-50/50 flex items-start justify-between gap-3 hover:border-amber-400 cursor-pointer transition-colors"
                >
                  <div>
                    <div className="text-sm font-medium text-gray-900">{an.type || 'Statistical Anomaly'}</div>
                    <div className="text-xs text-gray-600 mt-0.5">{an.description}</div>
                  </div>
                  <span className="badge badge-high text-[10px]">
                    {an.severity || 'Anomaly'}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Analytical Alerts */}
      {alerts.length > 0 && (
        <div className="glass-panel">
          <h3 className="text-base font-semibold text-gray-900 mb-3">Active Analytical Alerts ({alerts.length})</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {alerts.slice(0, 4).map((al: any) => (
              <div
                key={al.id}
                onClick={() => openObjectContext('alert', al.id, { title: al.name || al.title })}
                className="p-3 rounded border border-gray-200 bg-gray-50/50 hover:bg-amber-50/50 cursor-pointer transition-colors"
              >
                <div className="text-xs font-semibold text-gray-900">{al.name || al.title}</div>
                <div className="text-xs text-gray-600 mt-1">{al.message || al.description}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Data Lineage Modal */}
      {selectedLineage && (
        <DataLineageModal
          kpiTitle={selectedLineage.title}
          kpiValue={selectedLineage.val}
          onClose={() => setSelectedLineage(null)}
        />
      )}
    </div>
  );
};
