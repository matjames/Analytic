import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { Activity, CheckCircle2, AlertTriangle, XCircle, RefreshCw } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const ServiceHealthView: React.FC<{ user: User }> = ({ user: _user }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchHealth = () => {
    setRefreshing(true);
    fetch(`${API_BASE}/api/command-centre/service-health`)
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((res) => {
        setData(res);
        setLoading(false);
        setRefreshing(false);
      });
  };

  useEffect(() => {
    fetchHealth();
  }, []);

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-28 bg-gray-100 rounded-xl" />
        <div className="h-80 bg-gray-100 rounded-xl" />
      </div>
    );
  }

  const services = data?.services || [];
  const summary = data?.summary || {};
  const overall = data?.overall || 'healthy';

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <Activity className="w-6 h-6 text-blue-600" />
            Service Health & Observability
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Real-time operational status, network latency, and availability across all StatGate application domains.
          </p>
        </div>
        <button
          onClick={fetchHealth}
          disabled={refreshing}
          className="btn btn-secondary text-xs flex items-center gap-1.5"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh Health Checks
        </button>
      </div>

      {/* Health Overview Banner */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-5">
        <div className="stat-card border-l-4 border-l-blue-600 shadow-sm">
          <span className="stat-label">Platform State</span>
          <span className="text-xl font-bold uppercase tracking-wider text-gray-900 mt-1">
            {overall}
          </span>
          <span className="text-xs text-gray-400 mt-1">Overall infrastructure</span>
        </div>

        <div className="stat-card border-l-4 border-l-green-500 shadow-sm">
          <span className="stat-label">Healthy Services</span>
          <span className="stat-val text-green-600">{summary.healthy ?? 0}</span>
          <span className="text-xs text-gray-400 mt-1">Operational</span>
        </div>

        <div className="stat-card border-l-4 border-l-amber-500 shadow-sm">
          <span className="stat-label">Degraded Services</span>
          <span className="stat-val text-amber-600">{summary.degraded ?? 0}</span>
          <span className="text-xs text-gray-400 mt-1">Latency / partial errors</span>
        </div>

        <div className="stat-card border-l-4 border-l-red-500 shadow-sm">
          <span className="stat-label">Unreachable</span>
          <span className="stat-val text-red-600">{summary.unreachable ?? 0}</span>
          <span className="text-xs text-gray-400 mt-1">Offline or container down</span>
        </div>
      </div>

      {/* Services Table */}
      <div className="glass-panel">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Application Endpoints ({services.length})</h3>
          <span className="text-xs text-gray-400">
            Last checked: {data?.timestamp ? new Date(data.timestamp).toLocaleTimeString() : 'Just now'}
          </span>
        </div>

        {services.length === 0 ? (
          <p className="text-sm text-gray-500 py-8 text-center">No service endpoints configured.</p>
        ) : (
          <div className="divide-y divide-gray-100">
            {services.map((svc: any) => (
              <div key={svc.name} className="py-3.5 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  {svc.status === 'healthy' ? (
                    <CheckCircle2 className="w-5 h-5 text-green-500 flex-shrink-0" />
                  ) : svc.status === 'degraded' ? (
                    <AlertTriangle className="w-5 h-5 text-amber-500 flex-shrink-0" />
                  ) : (
                    <XCircle className="w-5 h-5 text-red-500 flex-shrink-0" />
                  )}
                  <div>
                    <div className="text-sm font-semibold text-gray-900 uppercase tracking-wide">
                      {svc.name}
                    </div>
                    <div className="text-xs text-gray-400 font-mono">{svc.url}</div>
                  </div>
                </div>

                <div className="flex items-center gap-6">
                  {svc.latency_ms !== undefined && (
                    <span className="text-xs text-gray-500 font-mono">{svc.latency_ms} ms</span>
                  )}
                  <span
                    className={`badge text-[11px] font-semibold uppercase tracking-wider ${
                      svc.status === 'healthy'
                        ? 'badge-success'
                        : svc.status === 'degraded'
                        ? 'badge-high'
                        : 'badge-critical'
                    }`}
                  >
                    {svc.status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
