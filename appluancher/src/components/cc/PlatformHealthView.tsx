import React, { useEffect, useState, useCallback } from 'react';
import {
  Server,
  CheckCircle,
  XCircle,
  AlertTriangle,
  RefreshCw,
  Activity,
  Database,
  Zap,
  Clock,
  Cpu,
  BarChart3,
} from 'lucide-react';

const ENTERPRISE_CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) ||
  'http://localhost:8096';

interface ServiceInfo {
  id: string;
  name: string;
  display_name: string;
  description: string;
  api_url: string;
  ui_url: string;
  health_url: string;
  version: string;
  status: string;
  last_heartbeat?: string;
}

interface PlatformMetrics {
  service: string;
  version: string;
  phase: string;
  uptime_s: number;
  timestamp: string;
  requests: { total: number; errors: number; error_rate: number };
  events: { total_published: number; redis_queue_depth: number; dlq_depth: number };
  database: { queries: number; pool: Record<string, unknown> };
  memory: { alloc_mb: number; sys_mb: number; heap_inuse_mb: number; gc_runs: number };
  goroutines: number;
}

interface ServiceHealthResult {
  id: string;
  name: string;
  display_name: string;
  status: 'healthy' | 'unhealthy' | 'unreachable' | 'checking';
  latency_ms?: number;
  version?: string;
  error?: string;
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { bg: string; text: string; icon: React.ReactNode }> = {
    healthy:    { bg: 'bg-emerald-50 border border-emerald-200', text: 'text-emerald-700', icon: <CheckCircle className="w-3.5 h-3.5" /> },
    active:     { bg: 'bg-emerald-50 border border-emerald-200', text: 'text-emerald-700', icon: <CheckCircle className="w-3.5 h-3.5" /> },
    unhealthy:  { bg: 'bg-red-50 border border-red-200',         text: 'text-red-700',     icon: <XCircle className="w-3.5 h-3.5" /> },
    unreachable:{ bg: 'bg-red-50 border border-red-200',         text: 'text-red-700',     icon: <XCircle className="w-3.5 h-3.5" /> },
    checking:   { bg: 'bg-amber-50 border border-amber-200',     text: 'text-amber-700',   icon: <AlertTriangle className="w-3.5 h-3.5" /> },
    degraded:   { bg: 'bg-amber-50 border border-amber-200',     text: 'text-amber-700',   icon: <AlertTriangle className="w-3.5 h-3.5" /> },
  };
  const cfg = map[status] ?? map['checking'];
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold ${cfg.bg} ${cfg.text}`}>
      {cfg.icon}
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
}

function MetricCard({ label, value, sub, icon: Icon, accent }: {
  label: string; value: string | number; sub?: string;
  icon: React.ComponentType<{ className?: string }>; accent: string;
}) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 p-4 flex items-start gap-3 shadow-sm">
      <div className={`p-2 rounded-lg ${accent}`}>
        <Icon className="w-5 h-5" />
      </div>
      <div className="flex-1 min-w-0">
        <div className="text-2xl font-bold text-gray-900 leading-tight">{value}</div>
        <div className="text-xs font-semibold text-gray-500 mt-0.5">{label}</div>
        {sub && <div className="text-xs text-gray-400 mt-1">{sub}</div>}
      </div>
    </div>
  );
}

function formatUptime(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

export const PlatformHealthView: React.FC = () => {
  const [metrics, setMetrics] = useState<PlatformMetrics | null>(null);
  const [services, setServices] = useState<ServiceInfo[]>([]);
  const [healthResults, setHealthResults] = useState<Record<string, ServiceHealthResult>>({});
  const [loading, setLoading] = useState(true);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const fetchMetrics = useCallback(async () => {
    try {
      const r = await fetch(`${ENTERPRISE_CORE}/metrics`);
      if (r.ok) setMetrics(await r.json());
    } catch { /* non-fatal */ }
  }, []);

  const fetchServices = useCallback(async () => {
    try {
      const r = await fetch(`${ENTERPRISE_CORE}/api/registry/services`, {
        headers: { Authorization: `Bearer ${localStorage.getItem('token') ?? ''}` },
      });
      if (r.ok) {
        const data = await r.json();
        setServices(data.services ?? []);
      }
    } catch { /* non-fatal */ }
  }, []);

  const checkServiceHealth = useCallback(async (svc: ServiceInfo) => {
    setHealthResults(prev => ({ ...prev, [svc.id]: { ...svc, status: 'checking' } }));
    const start = Date.now();
    try {
      const r = await fetch(`${ENTERPRISE_CORE}/api/registry/services/${svc.id}/health`, {
        headers: { Authorization: `Bearer ${localStorage.getItem('token') ?? ''}` },
      });
      const latency = Date.now() - start;
      const json = await r.json();
      const status = r.status < 400 ? 'healthy' : 'unhealthy';
      setHealthResults(prev => ({
        ...prev,
        [svc.id]: { ...svc, status, latency_ms: latency, version: json.version },
      }));
    } catch (e: unknown) {
      setHealthResults(prev => ({
        ...prev,
        [svc.id]: { ...svc, status: 'unreachable', error: String(e) },
      }));
    }
  }, []);

  const refresh = useCallback(async () => {
    setRefreshing(true);
    await Promise.all([fetchMetrics(), fetchServices()]);
    setLastRefresh(new Date());
    setRefreshing(false);
    setLoading(false);
  }, [fetchMetrics, fetchServices]);

  useEffect(() => {
    refresh();
    const interval = setInterval(fetchMetrics, 15000);
    return () => clearInterval(interval);
  }, [refresh, fetchMetrics]);

  useEffect(() => {
    if (services.length > 0) {
      services.forEach(svc => checkServiceHealth(svc));
    }
  }, [services, checkServiceHealth]);

  const healthyCount = Object.values(healthResults).filter(r => r.status === 'healthy').length;
  const unhealthyCount = Object.values(healthResults).filter(r => r.status === 'unhealthy' || r.status === 'unreachable').length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
            <Server className="w-6 h-6 text-blue-600" />
            Platform Health
          </h2>
          <p className="text-sm text-gray-500 mt-0.5">
            Phase X — Sovereign Platform Observability
            {lastRefresh && (
              <span className="ml-2 text-xs text-gray-400">
                Last updated {lastRefresh.toLocaleTimeString()}
              </span>
            )}
          </p>
        </div>
        <button
          id="platform-health-refresh"
          onClick={refresh}
          disabled={refreshing}
          className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-600 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
        >
          <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-48 text-gray-400">
          <RefreshCw className="w-6 h-6 animate-spin mr-2" /> Loading platform metrics…
        </div>
      ) : (
        <>
          {/* Metrics Cards */}
          {metrics && (
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <MetricCard
                label="Uptime"
                value={formatUptime(metrics.uptime_s)}
                icon={Clock}
                accent="bg-blue-50 text-blue-600"
              />
              <MetricCard
                label="API Requests"
                value={metrics.requests.total.toLocaleString()}
                sub={`${metrics.requests.error_rate.toFixed(1)}% error rate`}
                icon={Activity}
                accent="bg-emerald-50 text-emerald-600"
              />
              <MetricCard
                label="Events Published"
                value={metrics.events.total_published.toLocaleString()}
                sub={`DLQ: ${metrics.events.dlq_depth} | Queue: ${metrics.events.redis_queue_depth}`}
                icon={Zap}
                accent="bg-purple-50 text-purple-600"
              />
              <MetricCard
                label="Memory (Heap)"
                value={`${metrics.memory.heap_inuse_mb} MB`}
                sub={`${metrics.goroutines} goroutines | ${metrics.memory.gc_runs} GC runs`}
                icon={Cpu}
                accent="bg-amber-50 text-amber-600"
              />
            </div>
          )}

          {/* Service Health Summary */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm">
            <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <BarChart3 className="w-4 h-4 text-gray-500" />
                <h3 className="text-sm font-semibold text-gray-800">Service Registry Health</h3>
              </div>
              <div className="flex items-center gap-3 text-xs">
                <span className="flex items-center gap-1 text-emerald-600 font-semibold">
                  <CheckCircle className="w-3.5 h-3.5" /> {healthyCount} healthy
                </span>
                {unhealthyCount > 0 && (
                  <span className="flex items-center gap-1 text-red-600 font-semibold">
                    <XCircle className="w-3.5 h-3.5" /> {unhealthyCount} issues
                  </span>
                )}
              </div>
            </div>
            <div className="divide-y divide-gray-50">
              {services.length === 0 ? (
                <div className="px-5 py-8 text-center text-sm text-gray-400">
                  <Database className="w-8 h-8 mx-auto mb-2 text-gray-300" />
                  No services registered. Ensure STATGATE_DB_URL is configured.
                </div>
              ) : (
                services.map(svc => {
                  const health = healthResults[svc.id];
                  return (
                    <div key={svc.id} className="px-5 py-3 flex items-center gap-4 hover:bg-gray-50 transition-colors">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-semibold text-gray-900">{svc.display_name || svc.name}</span>
                          {health?.version && (
                            <span className="text-xs text-gray-400 font-mono">v{health.version}</span>
                          )}
                        </div>
                        <div className="text-xs text-gray-500 truncate">{svc.description}</div>
                      </div>
                      <div className="flex items-center gap-3 flex-shrink-0">
                        {health?.latency_ms !== undefined && (
                          <span className={`text-xs font-mono ${health.latency_ms > 500 ? 'text-amber-600' : 'text-gray-400'}`}>
                            {health.latency_ms}ms
                          </span>
                        )}
                        <StatusBadge status={health?.status ?? 'checking'} />
                        {svc.ui_url && (
                          <a
                            href={svc.ui_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-xs text-blue-600 hover:underline"
                          >
                            Open →
                          </a>
                        )}
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>

          {/* Database Pool Stats */}
          {metrics?.database?.pool && typeof metrics.database.pool === 'object' && 'open_connections' in metrics.database.pool && (
            <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
              <div className="flex items-center gap-2 mb-4">
                <Database className="w-4 h-4 text-gray-500" />
                <h3 className="text-sm font-semibold text-gray-800">Database Connection Pool</h3>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {Object.entries(metrics.database.pool as Record<string, unknown>).map(([k, v]) => (
                  <div key={k} className="text-center">
                    <div className="text-2xl font-bold text-gray-900">{String(v)}</div>
                    <div className="text-xs text-gray-500 mt-0.5">{k.replace(/_/g, ' ')}</div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};
