import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { CheckCircle2, ArrowRight, Activity } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const OutcomeMonitorView: React.FC<{ user: User }> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    fetch(`${API_BASE}/api/command-centre/outcomes?user_id=${encodeURIComponent(user.id)}`, {
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
        <div className="h-80 bg-gray-100 rounded-xl" />
      </div>
    );
  }

  const monitoring = data?.monitoring || [];
  const closed = data?.closed || [];

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
          <CheckCircle2 className="w-6 h-6 text-emerald-600" />
          Institutional Outcome Monitoring & Closed-Loop Intelligence
        </h2>
        <p className="text-sm text-gray-500 mt-1">
          Verifying and monitoring whether executed decisions, tasks, and interventions solved the original institutional problems.
        </p>
      </div>

      {/* Closed-loop Pipeline Visualization */}
      <div className="glass-panel">
        <h3 className="text-sm font-semibold uppercase tracking-wider text-gray-500 mb-4">
          Closed-Loop Reasoning Chain
        </h3>
        <div className="flex flex-wrap items-center justify-between gap-2 p-3 bg-gray-50 rounded-lg text-xs font-medium text-gray-700">
          <span className="px-2.5 py-1 bg-white border border-gray-200 rounded-md">Data</span>
          <ArrowRight className="w-4 h-4 text-gray-400" />
          <span className="px-2.5 py-1 bg-white border border-gray-200 rounded-md">Anomaly</span>
          <ArrowRight className="w-4 h-4 text-gray-400" />
          <span className="px-2.5 py-1 bg-white border border-gray-200 rounded-md">Alert</span>
          <ArrowRight className="w-4 h-4 text-gray-400" />
          <span className="px-2.5 py-1 bg-white border border-gray-200 rounded-md">Investigation</span>
          <ArrowRight className="w-4 h-4 text-gray-400" />
          <span className="px-2.5 py-1 bg-white border border-gray-200 rounded-md">Evidence</span>
          <ArrowRight className="w-4 h-4 text-gray-400" />
          <span className="px-2.5 py-1 bg-purple-50 border border-purple-200 text-purple-800 rounded-md">AI Suggestion</span>
          <ArrowRight className="w-4 h-4 text-gray-400" />
          <span className="px-2.5 py-1 bg-amber-50 border border-amber-300 text-amber-900 rounded-md font-bold">Human Decision</span>
          <ArrowRight className="w-4 h-4 text-gray-400" />
          <span className="px-2.5 py-1 bg-blue-50 border border-blue-300 text-blue-800 rounded-md font-bold">
            Task Action
          </span>
          <ArrowRight className="w-4 h-4 text-gray-400" />
          <span className="px-2.5 py-1 bg-green-50 border border-green-300 text-green-800 rounded-md font-bold">
            Outcome & Knowledge
          </span>
        </div>
      </div>

      {/* Grid: Actioned & Monitored Decisions */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Monitored Decisions */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <Activity className="w-5 h-5 text-blue-600" />
              Decisions in Post-Action Monitoring ({monitoring.length})
            </h3>
          </div>

          {monitoring.length === 0 ? (
            <p className="text-sm text-gray-500 py-8 text-center">
              No decisions currently under active post-action monitoring.
            </p>
          ) : (
            <div className="divide-y divide-gray-100">
              {monitoring.map((d: any) => (
                <div
                  key={d.id}
                  onClick={() => openObjectContext('decision', d.id, { title: d.decision })}
                  className="py-3.5 p-2 rounded-lg hover:bg-blue-50/50 cursor-pointer transition-colors"
                >
                  <div className="text-sm font-semibold text-gray-900">{d.decision}</div>
                  <div className="text-xs text-gray-600 mt-1">
                    Responsible: {d.responsible_person || 'Assigned Lead'}
                  </div>
                  <div className="text-xs text-blue-700 mt-1 font-medium">
                    Outcome status: {d.outcome || 'Evaluating impact metrics'}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Closed Decisions with Confirmed Outcomes */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <CheckCircle2 className="w-5 h-5 text-green-600" />
              Institutional Outcomes Archive ({closed.length})
            </h3>
          </div>

          {closed.length === 0 ? (
            <p className="text-sm text-gray-500 py-8 text-center">
              No archived closed outcomes recorded yet.
            </p>
          ) : (
            <div className="divide-y divide-gray-100">
              {closed.map((d: any) => (
                <div
                  key={d.id}
                  onClick={() => openObjectContext('decision', d.id, { title: d.decision })}
                  className="py-3.5 p-2 rounded-lg hover:bg-green-50/50 cursor-pointer transition-colors"
                >
                  <div className="text-sm font-semibold text-gray-900">{d.decision}</div>
                  <div className="text-xs text-gray-600 mt-1">
                    Confirmed Outcome: {d.outcome || 'Action successfully resolved issue'}
                  </div>
                  <span className="badge badge-success text-[10px] mt-2">Closed & Verified</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
