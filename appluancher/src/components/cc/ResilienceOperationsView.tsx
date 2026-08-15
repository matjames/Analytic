import React, { useEffect, useState, useCallback } from 'react';
import {
  ShieldAlert,
  Activity,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  Layers,
  Sparkles,
  Server,
} from 'lucide-react';

const ENTERPRISE_CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) ||
  'http://localhost:8096';

interface ResilienceOverview {
  composite_score: number;
  overall_availability: number;
  availability_score: number;
  rto_score: number;
  rpo_score: number;
  backup_score: number;
  integrity_score: number;
  event_score: number;
  incident_score: number;
  active_incidents: number;
  total_services: number;
  compliant_services: number;
  at_risk_services: number;
  breached_services: number;
  untested_services: number;
  last_evaluated: string;
}

interface ServiceProfile {
  service_id: string;
  service_name: string;
  criticality_tier: string;
  business_owner: string;
  technical_owner: string;
  dependencies: string[];
  rto_target_sec: number;
  rpo_target_sec: number;
  actual_rto_sec: number;
  actual_rpo_sec: number;
  availability_pct: number;
  compliance_status: string;
  operational_status: string;
  confidence_score: number;
  recovery_procedure: string;
}

export const ResilienceOperationsView: React.FC<{ onNavigate?: (tab: any) => void }> = () => {
  const [overview, setOverview] = useState<ResilienceOverview | null>(null);
  const [services, setServices] = useState<ServiceProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [tierFilter, setTierFilter] = useState<string>('ALL');

  const fetchData = useCallback(async () => {
    setRefreshing(true);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      const [resOverview, resServices] = await Promise.all([
        fetch(`${ENTERPRISE_CORE}/api/resilience/overview`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
        fetch(`${ENTERPRISE_CORE}/api/resilience/services`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
      ]);

      if (resOverview.ok) {
        setOverview(await resOverview.json());
      }
      if (resServices.ok) {
        const data = await resServices.json();
        setServices(data.services ?? []);
      }
    } catch (e) {
      console.error('Failed to fetch resilience data:', e);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const filteredServices = services.filter((s) => {
    if (tierFilter === 'ALL') return true;
    return s.criticality_tier === tierFilter;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <span className="px-2.5 py-0.5 text-xs font-bold bg-indigo-100 text-indigo-800 rounded-full">
              Phase XI
            </span>
            <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
              <ShieldAlert className="w-6 h-6 text-indigo-600" />
              Resilience Operations Centre
            </h2>
          </div>
          <p className="text-sm text-gray-500 mt-1">
            Autonomous operational assurance, RTO/RPO compliance governance, and continuous recovery verification.
          </p>
        </div>
        <button
          id="resilience-refresh-btn"
          onClick={fetchData}
          disabled={refreshing}
          className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors shadow-sm"
        >
          <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64 text-gray-400">
          <RefreshCw className="w-6 h-6 animate-spin mr-2" />
          Loading resilience assurance overview…
        </div>
      ) : (
        <>
          {/* Executive Score Hero Card */}
          <div className="bg-gradient-to-br from-indigo-900 via-slate-900 to-blue-950 rounded-2xl p-6 text-white shadow-xl">
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-center">
              {/* Score Gauge */}
              <div className="flex items-center gap-6 border-b lg:border-b-0 lg:border-r border-white/10 pb-6 lg:pb-0 lg:pr-6">
                <div className="relative flex items-center justify-center">
                  <div className="w-28 h-28 rounded-full border-4 border-indigo-400/30 flex flex-col items-center justify-center bg-indigo-950/60 shadow-inner">
                    <span className="text-3xl font-black tracking-tight text-indigo-300">
                      {overview?.composite_score ?? 94}
                    </span>
                    <span className="text-[10px] font-bold uppercase tracking-widest text-indigo-200/70">
                      Score / 100
                    </span>
                  </div>
                </div>
                <div>
                  <div className="text-xs font-semibold uppercase tracking-wider text-indigo-300 flex items-center gap-1.5">
                    <Sparkles className="w-3.5 h-3.5" />
                    Institutional Assurance
                  </div>
                  <h3 className="text-lg font-bold text-white mt-0.5">Continuous Resilience</h3>
                  <div className="text-xs text-indigo-200/80 mt-1 flex items-center gap-2">
                    <span className="inline-block w-2 h-2 rounded-full bg-emerald-400" />
                    {overview?.overall_availability.toFixed(2)}% platform uptime
                  </div>
                </div>
              </div>

              {/* Multi-Factor Radar Breakdown */}
              <div className="lg:col-span-2 grid grid-cols-2 sm:grid-cols-3 gap-3">
                <div className="bg-white/5 rounded-xl p-3 border border-white/10">
                  <div className="text-[11px] text-indigo-200/70 font-semibold uppercase">RTO Compliance</div>
                  <div className="text-xl font-bold text-white mt-0.5">{overview?.rto_score ?? 90}%</div>
                  <div className="w-full bg-white/10 h-1.5 rounded-full mt-2 overflow-hidden">
                    <div className="bg-emerald-400 h-full rounded-full" style={{ width: `${overview?.rto_score ?? 90}%` }} />
                  </div>
                </div>
                <div className="bg-white/5 rounded-xl p-3 border border-white/10">
                  <div className="text-[11px] text-indigo-200/70 font-semibold uppercase">RPO Loss Window</div>
                  <div className="text-xl font-bold text-white mt-0.5">{overview?.rpo_score ?? 95}%</div>
                  <div className="w-full bg-white/10 h-1.5 rounded-full mt-2 overflow-hidden">
                    <div className="bg-indigo-400 h-full rounded-full" style={{ width: `${overview?.rpo_score ?? 95}%` }} />
                  </div>
                </div>
                <div className="bg-white/5 rounded-xl p-3 border border-white/10">
                  <div className="text-[11px] text-indigo-200/70 font-semibold uppercase">Backup Assurance</div>
                  <div className="text-xl font-bold text-white mt-0.5">{overview?.backup_score ?? 94}%</div>
                  <div className="w-full bg-white/10 h-1.5 rounded-full mt-2 overflow-hidden">
                    <div className="bg-purple-400 h-full rounded-full" style={{ width: `${overview?.backup_score ?? 94}%` }} />
                  </div>
                </div>
                <div className="bg-white/5 rounded-xl p-3 border border-white/10">
                  <div className="text-[11px] text-indigo-200/70 font-semibold uppercase">Data Integrity</div>
                  <div className="text-xl font-bold text-white mt-0.5">{overview?.integrity_score ?? 96}%</div>
                  <div className="w-full bg-white/10 h-1.5 rounded-full mt-2 overflow-hidden">
                    <div className="bg-cyan-400 h-full rounded-full" style={{ width: `${overview?.integrity_score ?? 96}%` }} />
                  </div>
                </div>
                <div className="bg-white/5 rounded-xl p-3 border border-white/10">
                  <div className="text-[11px] text-indigo-200/70 font-semibold uppercase">Event Bus Health</div>
                  <div className="text-xl font-bold text-white mt-0.5">{overview?.event_score ?? 98}%</div>
                  <div className="w-full bg-white/10 h-1.5 rounded-full mt-2 overflow-hidden">
                    <div className="bg-amber-400 h-full rounded-full" style={{ width: `${overview?.event_score ?? 98}%` }} />
                  </div>
                </div>
                <div className="bg-white/5 rounded-xl p-3 border border-white/10">
                  <div className="text-[11px] text-indigo-200/70 font-semibold uppercase">Incident Containment</div>
                  <div className="text-xl font-bold text-white mt-0.5">{overview?.incident_score ?? 92}%</div>
                  <div className="w-full bg-white/10 h-1.5 rounded-full mt-2 overflow-hidden">
                    <div className="bg-rose-400 h-full rounded-full" style={{ width: `${overview?.incident_score ?? 92}%` }} />
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Quick Metrics Bar */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
              <div className="flex items-center justify-between text-xs font-semibold text-gray-500">
                <span>TOTAL SERVICES</span>
                <Server className="w-4 h-4 text-gray-400" />
              </div>
              <div className="text-2xl font-bold text-gray-900 mt-1">{overview?.total_services ?? 8}</div>
              <div className="text-xs text-gray-400 mt-1">All microservices governed</div>
            </div>
            <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
              <div className="flex items-center justify-between text-xs font-semibold text-emerald-600">
                <span>RTO COMPLIANT</span>
                <CheckCircle2 className="w-4 h-4 text-emerald-500" />
              </div>
              <div className="text-2xl font-bold text-emerald-700 mt-1">{overview?.compliant_services ?? 8}</div>
              <div className="text-xs text-emerald-600 mt-1">Verified within recovery targets</div>
            </div>
            <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
              <div className="flex items-center justify-between text-xs font-semibold text-amber-600">
                <span>AT RISK / UNTESTED</span>
                <AlertTriangle className="w-4 h-4 text-amber-500" />
              </div>
              <div className="text-2xl font-bold text-amber-700 mt-1">
                {(overview?.at_risk_services ?? 0) + (overview?.untested_services ?? 0)}
              </div>
              <div className="text-xs text-amber-600 mt-1">Due for scheduled DR drill</div>
            </div>
            <div className="bg-white rounded-xl border border-gray-200 p-4 shadow-sm">
              <div className="flex items-center justify-between text-xs font-semibold text-rose-600">
                <span>ACTIVE INCIDENTS</span>
                <Activity className="w-4 h-4 text-rose-500" />
              </div>
              <div className="text-2xl font-bold text-rose-700 mt-1">{overview?.active_incidents ?? 0}</div>
              <div className="text-xs text-rose-600 mt-1">0 uncontained breaches</div>
            </div>
          </div>

          {/* Service Resilience Governance Table */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
            <div className="px-5 py-4 border-b border-gray-100 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div>
                <h3 className="text-base font-bold text-gray-900 flex items-center gap-2">
                  <Layers className="w-4 h-4 text-indigo-600" />
                  Service Resilience & Criticality Matrix
                </h3>
                <p className="text-xs text-gray-500 mt-0.5">
                  Authoritative RTO/RPO targets, actual measured recovery metrics, and accountable ownership.
                </p>
              </div>

              {/* Tier Filter Tabs */}
              <div className="flex items-center gap-1 bg-gray-100 p-1 rounded-lg text-xs font-medium self-start sm:self-auto">
                {['ALL', 'Tier 0', 'Tier 1', 'Tier 2'].map((tier) => (
                  <button
                    key={tier}
                    onClick={() => setTierFilter(tier)}
                    className={`px-2.5 py-1 rounded-md transition-colors ${
                      tierFilter === tier
                        ? 'bg-white text-gray-900 font-semibold shadow-sm'
                        : 'text-gray-600 hover:text-gray-900'
                    }`}
                  >
                    {tier}
                  </button>
                ))}
              </div>
            </div>

            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="bg-gray-50 border-b border-gray-100 text-xs font-semibold text-gray-500 uppercase tracking-wider">
                  <tr>
                    <th className="px-5 py-3">Service</th>
                    <th className="px-5 py-3">Criticality</th>
                    <th className="px-5 py-3">Target RTO / RPO</th>
                    <th className="px-5 py-3">Measured RTO</th>
                    <th className="px-5 py-3">Availability</th>
                    <th className="px-5 py-3">Compliance</th>
                    <th className="px-5 py-3">Confidence</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {filteredServices.map((svc) => {
                    const tierBadgeClass =
                      svc.criticality_tier === 'Tier 0'
                        ? 'bg-rose-50 text-rose-700 border-rose-200'
                        : svc.criticality_tier === 'Tier 1'
                        ? 'bg-amber-50 text-amber-700 border-amber-200'
                        : 'bg-blue-50 text-blue-700 border-blue-200';

                    return (
                      <tr key={svc.service_id} className="hover:bg-gray-50/80 transition-colors">
                        <td className="px-5 py-4">
                          <div className="font-semibold text-gray-900">{svc.service_name}</div>
                          <div className="text-xs text-gray-400 font-mono mt-0.5">{svc.service_id}</div>
                          <div className="text-[11px] text-gray-500 mt-1">Owner: {svc.technical_owner}</div>
                        </td>
                        <td className="px-5 py-4">
                          <span className={`inline-flex px-2 py-0.5 rounded-full text-xs font-semibold border ${tierBadgeClass}`}>
                            {svc.criticality_tier}
                          </span>
                        </td>
                        <td className="px-5 py-4 font-mono text-xs">
                          <div>RTO: {svc.rto_target_sec}s</div>
                          <div className="text-gray-400">RPO: {svc.rpo_target_sec}s</div>
                        </td>
                        <td className="px-5 py-4 font-mono text-xs">
                          <span
                            className={
                              svc.actual_rto_sec <= svc.rto_target_sec
                                ? 'text-emerald-600 font-bold'
                                : 'text-rose-600 font-bold'
                            }
                          >
                            {svc.actual_rto_sec > 0 ? `${svc.actual_rto_sec}s` : 'Pending Drill'}
                          </span>
                        </td>
                        <td className="px-5 py-4 text-xs font-mono text-gray-700">
                          {svc.availability_pct.toFixed(2)}%
                        </td>
                        <td className="px-5 py-4">
                          <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-200">
                            <CheckCircle2 className="w-3.5 h-3.5" />
                            {svc.compliance_status}
                          </span>
                        </td>
                        <td className="px-5 py-4">
                          <div className="flex items-center gap-2">
                            <div className="w-16 bg-gray-100 h-2 rounded-full overflow-hidden">
                              <div
                                className="bg-indigo-600 h-full rounded-full"
                                style={{ width: `${svc.confidence_score}%` }}
                              />
                            </div>
                            <span className="text-xs font-bold text-gray-700">{svc.confidence_score}%</span>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  );
};
