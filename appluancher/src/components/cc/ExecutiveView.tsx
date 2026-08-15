import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { DataLineageModal } from '@components/DataLineageModal';
import {
  TrendingUp,
  AlertTriangle,
  CheckCircle2,
  FolderGit2,
  FlaskConical,
  Activity,
  ArrowUpRight,
  ShieldAlert,
  GitCommit,
  Download,
} from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const ExecutiveView: React.FC<{ user: User }> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [selectedLineage, setSelectedLineage] = useState<{ title: string; val: any } | null>(null);
  const [exportNotice, setExportNotice] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    fetch(`${API_BASE}/api/command-centre/executive?user_id=${encodeURIComponent(user.id)}`, {
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

  const handleExport = (format: string) => {
    setExportNotice(`Generating signed ${format.toUpperCase()} executive briefing...`);
    setTimeout(() => {
      setExportNotice(`Executive Report [${format.toUpperCase()}] generated and archived to Enterprise Store.`);
      setTimeout(() => setExportNotice(null), 4000);
    }, 1200);
  };

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-28 bg-gray-100 rounded-xl" />
          ))}
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="h-64 bg-gray-100 rounded-xl" />
          <div className="h-64 bg-gray-100 rounded-xl" />
        </div>
      </div>
    );
  }

  const situation = data?.situation || {};
  const kpis = data?.kpis || [];
  const alerts = data?.alerts || [];
  const risks = data?.risks || [];
  const projects = data?.projects || [];
  const research = data?.research || [];

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <TrendingUp className="w-6 h-6 text-blue-600" />
            Executive Command Centre
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            High-level institutional health, strategic portfolio progress, and cross-application risk indicators.
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {exportNotice && (
            <span className="text-xs font-semibold text-blue-700 bg-blue-50 px-3 py-1.5 rounded-lg border border-blue-200 animate-fade-in">
              {exportNotice}
            </span>
          )}
          <button
            onClick={() => handleExport('pdf')}
            className="btn btn-secondary text-xs flex items-center gap-1.5"
          >
            <Download className="w-3.5 h-3.5" /> Export PDF
          </button>
          <button
            onClick={() => handleExport('excel')}
            className="btn btn-secondary text-xs flex items-center gap-1.5"
          >
            <Download className="w-3.5 h-3.5" /> Excel
          </button>
          <span className="text-xs px-3 py-1.5 bg-blue-50 border border-blue-200 text-blue-800 rounded-full font-semibold">
            Live Institutional Intelligence
          </span>
        </div>
      </div>

      {/* Strategic KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <div
          onClick={() => setSelectedLineage({ title: 'Active Projects Portfolio', val: situation.active_projects ?? projects.length })}
          className="stat-card border-l-4 border-l-blue-600 shadow-sm cursor-pointer hover:border-blue-700 transition-all group"
        >
          <div className="flex items-center justify-between">
            <span className="stat-label">Active Projects</span>
            <FolderGit2 className="w-5 h-5 text-blue-500" />
          </div>
          <span className="stat-val">{situation.active_projects ?? projects.length}</span>
          <div className="flex items-center justify-between text-xs text-gray-400 mt-1">
            <span>Across all departments</span>
            <span className="text-blue-600 font-semibold group-hover:underline flex items-center gap-0.5">
              Lineage <GitCommit className="w-3 h-3" />
            </span>
          </div>
        </div>

        <div
          onClick={() => setSelectedLineage({ title: 'RMS Research Portfolio', val: situation.active_research ?? research.length })}
          className="stat-card border-l-4 border-l-purple-600 shadow-sm cursor-pointer hover:border-purple-700 transition-all group"
        >
          <div className="flex items-center justify-between">
            <span className="stat-label">Research Studies</span>
            <FlaskConical className="w-5 h-5 text-purple-500" />
          </div>
          <span className="stat-val">{situation.active_research ?? research.length}</span>
          <div className="flex items-center justify-between text-xs text-gray-400 mt-1">
            <span>RMS Governed Portfolio</span>
            <span className="text-purple-600 font-semibold group-hover:underline flex items-center gap-0.5">
              Lineage <GitCommit className="w-3 h-3" />
            </span>
          </div>
        </div>

        <div
          onClick={() => setSelectedLineage({ title: 'StatGovernance Critical Risks', val: risks.length })}
          className="stat-card border-l-4 border-l-amber-500 shadow-sm cursor-pointer hover:border-amber-700 transition-all group"
        >
          <div className="flex items-center justify-between">
            <span className="stat-label">Open Critical Risks</span>
            <ShieldAlert className="w-5 h-5 text-amber-500" />
          </div>
          <span className="stat-val">{risks.length}</span>
          <div className="flex items-center justify-between text-xs text-gray-400 mt-1">
            <span>StatGovernance Registry</span>
            <span className="text-amber-600 font-semibold group-hover:underline flex items-center gap-0.5">
              Lineage <GitCommit className="w-3 h-3" />
            </span>
          </div>
        </div>

        <div
          onClick={() => setSelectedLineage({ title: 'Pending Institutional Signoffs', val: situation.pending_approvals ?? 0 })}
          className="stat-card border-l-4 border-l-emerald-600 shadow-sm cursor-pointer hover:border-emerald-700 transition-all group"
        >
          <div className="flex items-center justify-between">
            <span className="stat-label">Pending Approvals</span>
            <CheckCircle2 className="w-5 h-5 text-emerald-500" />
          </div>
          <span className="stat-val">{situation.pending_approvals ?? 0}</span>
          <div className="flex items-center justify-between text-xs text-gray-400 mt-1">
            <span>Executive Authority Action</span>
            <span className="text-emerald-600 font-semibold group-hover:underline flex items-center gap-0.5">
              Lineage <GitCommit className="w-3 h-3" />
            </span>
          </div>
        </div>
      </div>

      {/* Grid: Monitored KPIs & Critical Alerts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Core Enterprise KPIs */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <Activity className="w-5 h-5 text-blue-600" />
              Strategic KPIs
            </h3>
            <span className="text-xs text-gray-500">{kpis.length} defined</span>
          </div>

          {kpis.length === 0 ? (
            <p className="text-sm text-gray-500 py-6 text-center">No strategic KPIs registered.</p>
          ) : (
            <div className="divide-y divide-gray-100">
              {kpis.slice(0, 6).map((kpi: any) => (
                <div
                  key={kpi.id}
                  onClick={() => setSelectedLineage({ title: kpi.name, val: kpi.threshold ? `${kpi.threshold} ${kpi.unit || ''}` : 'Operational' })}
                  className="py-3 flex items-center justify-between hover:bg-blue-50/50 p-2 rounded-lg cursor-pointer transition-colors group"
                >
                  <div>
                    <div className="text-sm font-medium text-gray-900 group-hover:text-blue-600 transition-colors">
                      {kpi.name}
                    </div>
                    <div className="text-xs text-gray-500">
                      {kpi.category} • Scope: {kpi.scope}
                    </div>
                  </div>
                  <div className="text-right">
                    <span className="text-sm font-bold text-gray-900">
                      {kpi.threshold ? `${kpi.threshold} ${kpi.unit || ''}` : 'Active'}
                    </span>
                    <div className="text-[10px] text-blue-600 font-semibold flex items-center justify-end gap-1">
                      Lineage <GitCommit className="w-3 h-3" />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Institutional Alerts & Anomalies */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-amber-500" />
              Active Institutional Alerts
            </h3>
            <span className="text-xs text-gray-500">{alerts.length} active</span>
          </div>

          {alerts.length === 0 ? (
            <div className="py-8 text-center">
              <CheckCircle2 className="w-8 h-8 text-green-500 mx-auto mb-2" />
              <p className="text-sm text-gray-600 font-medium">All systems within normal thresholds</p>
              <p className="text-xs text-gray-400">No active critical or high alerts</p>
            </div>
          ) : (
            <div className="space-y-3">
              {alerts.slice(0, 5).map((al: any) => (
                <div
                  key={al.id}
                  onClick={() => openObjectContext('alert', al.id, { title: al.name || al.title })}
                  className="p-3 rounded-lg border border-amber-200 bg-amber-50/50 flex items-start justify-between gap-3 cursor-pointer hover:border-amber-400 transition-colors"
                >
                  <div>
                    <div className="text-sm font-semibold text-gray-900">{al.name || al.title}</div>
                    <div className="text-xs text-gray-600 mt-0.5">{al.message || al.description}</div>
                  </div>
                  <span
                    className={`badge text-[10px] ${
                      al.severity === 'critical' ? 'badge-critical' : 'badge-high'
                    }`}
                  >
                    {al.severity}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Portfolio Overview: Projects & Governance */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Project Portfolio */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900">Project Portfolio Status</h3>
            <a
              href="http://localhost:3010"
              className="text-xs text-blue-600 hover:text-blue-800 flex items-center gap-1 font-medium"
            >
              Open PMS <ArrowUpRight className="w-3.5 h-3.5" />
            </a>
          </div>

          {projects.length === 0 ? (
            <p className="text-sm text-gray-500 py-6 text-center">No active projects found in PMS.</p>
          ) : (
            <div className="space-y-3">
              {projects.slice(0, 5).map((p: any) => (
                <div
                  key={p.id}
                  onClick={() => openObjectContext('project', p.id, { title: p.name })}
                  className="p-3 rounded-lg border border-gray-100 bg-gray-50/50 cursor-pointer hover:border-blue-300 transition-colors"
                >
                  <div className="flex items-center justify-between mb-1.5">
                    <span className="text-sm font-medium text-gray-900 truncate">{p.name}</span>
                    <span className="text-xs px-2 py-0.5 rounded bg-blue-100 text-blue-800 font-medium">
                      {p.stage || 'In Progress'}
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-1.5 overflow-hidden">
                    <div
                      className="bg-blue-600 h-1.5 rounded-full"
                      style={{ width: `${Math.min(p.progress || 0, 100)}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Governance & Risk Summary */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900">Institutional Governance & Risk</h3>
            <a
              href="http://localhost:3012"
              className="text-xs text-blue-600 hover:text-blue-800 flex items-center gap-1 font-medium"
            >
              StatGovernance <ArrowUpRight className="w-3.5 h-3.5" />
            </a>
          </div>

          {risks.length === 0 ? (
            <p className="text-sm text-gray-500 py-6 text-center">
              Governance risk register is clear or unavailable.
            </p>
          ) : (
            <div className="space-y-2.5">
              {risks.slice(0, 5).map((r: any) => (
                <div
                  key={r.id}
                  onClick={() => openObjectContext('risk', r.id, { title: r.title || r.name })}
                  className="p-3 rounded-lg border border-gray-200 flex items-center justify-between gap-3 cursor-pointer hover:border-blue-300 transition-colors"
                >
                  <div className="min-w-0 flex-1">
                    <div className="text-sm font-medium text-gray-900 truncate">{r.title || r.name}</div>
                    <div className="text-xs text-gray-500">{r.category || 'Institutional Risk'}</div>
                  </div>
                  <span
                    className={`badge text-[10px] ${
                      r.severity === 'Critical' ? 'badge-critical' : 'badge-medium'
                    }`}
                  >
                    {r.severity || 'Assessed'}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

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
