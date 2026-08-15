import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import {
  Sliders,
  CheckSquare,
  AlertCircle,
  Headphones,
  GitPullRequest,
  Clock,
  ArrowUpRight,
  Database,
} from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const OperationsView: React.FC<{ user: User }> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [filterSource, setFilterSource] = useState<string>('all');

  useEffect(() => {
    let cancelled = false;
    setLoading(true);

    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    fetch(`${API_BASE}/api/command-centre/operations?user_id=${encodeURIComponent(user.id)}`, {
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
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-28 bg-gray-100 rounded-xl" />
          ))}
        </div>
        <div className="h-96 bg-gray-100 rounded-xl" />
      </div>
    );
  }

  const tasks = data?.tasks || [];
  const overdueTasks = data?.overdue_tasks || [];
  const tickets = data?.tickets || [];
  const workflows = data?.workflows || {};
  const dataQuality = data?.data_quality || [];
  const approvals = data?.approvals || [];

  const filteredTasks = tasks.filter((t: any) => {
    if (filterSource === 'all') return true;
    return (t.source || '').toLowerCase() === filterSource.toLowerCase();
  });

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <Sliders className="w-6 h-6 text-indigo-600" />
            Operations Console & Execution Hub
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Real-time tracking of operational queues, team task execution, HelpDesk SLAs, and workflow instances.
          </p>
        </div>

        {/* Source Filter Tabs */}
        <div className="flex items-center bg-gray-100 p-1 rounded-lg">
          {['all', 'pms', 'rms', 'enterprise'].map((s) => (
            <button
              key={s}
              onClick={() => setFilterSource(s)}
              className={`px-3 py-1.5 text-xs font-semibold rounded-md uppercase tracking-wider transition-all ${
                filterSource === s
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-500 hover:text-gray-900'
              }`}
            >
              {s}
            </button>
          ))}
        </div>
      </div>

      {/* Operational Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        <div className="stat-card border-l-4 border-l-blue-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Assigned Tasks</span>
            <CheckSquare className="w-5 h-5 text-blue-500" />
          </div>
          <span className="stat-val">{tasks.length}</span>
          <span className="text-xs text-gray-400 mt-1">Active team queue</span>
        </div>

        <div className="stat-card border-l-4 border-l-red-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Overdue Tasks</span>
            <AlertCircle className="w-5 h-5 text-red-500" />
          </div>
          <span className="stat-val">{overdueTasks.length}</span>
          <span className="text-xs text-red-600 font-medium mt-1">SLA Attention Required</span>
        </div>

        <div className="stat-card border-l-4 border-l-purple-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Open Tickets</span>
            <Headphones className="w-5 h-5 text-purple-500" />
          </div>
          <span className="stat-val">{tickets.length}</span>
          <span className="text-xs text-gray-400 mt-1">HelpDesk support queue</span>
        </div>

        <div className="stat-card border-l-4 border-l-amber-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Workflows</span>
            <GitPullRequest className="w-5 h-5 text-amber-500" />
          </div>
          <span className="stat-val">{workflows.running_count ?? 0}</span>
          <span className="text-xs text-gray-400 mt-1">
            {workflows.failed_count ?? 0} failed workflows
          </span>
        </div>

        <div className="stat-card border-l-4 border-l-green-500 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Pending Approvals</span>
            <Clock className="w-5 h-5 text-green-500" />
          </div>
          <span className="stat-val">{approvals.length}</span>
          <span className="text-xs text-gray-400 mt-1">Awaiting sign-off</span>
        </div>
      </div>

      {/* Grid: Task Queue & HelpDesk / Data Quality */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Main Task Queue */}
        <div className="lg:col-span-2 glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
              <CheckSquare className="w-5 h-5 text-blue-600" />
              Operational Task Queue
            </h3>
            <span className="text-xs text-gray-500">{filteredTasks.length} items</span>
          </div>

          {filteredTasks.length === 0 ? (
            <p className="text-sm text-gray-500 py-12 text-center">
              No tasks currently queued for this filter.
            </p>
          ) : (
            <div className="divide-y divide-gray-100">
              {filteredTasks.map((t: any) => (
                <div
                  key={t.id}
                  onClick={() => openObjectContext('task', t.id, { title: t.title })}
                  className="py-3 flex items-start justify-between gap-4 p-2 hover:bg-blue-50/50 rounded-lg cursor-pointer transition-colors group"
                >
                  <div className="min-w-0 flex-1">
                    <div className="text-sm font-medium text-gray-900 group-hover:text-blue-600 transition-colors">{t.title}</div>
                    <div className="flex items-center gap-2 text-xs text-gray-500 mt-0.5">
                      <span className="font-semibold uppercase tracking-wider text-blue-700">
                        {t.source}
                      </span>
                      {t.due && (
                        <>
                          <span>•</span>
                          <span className="flex items-center gap-1">
                            <Clock className="w-3 h-3" /> Due {t.due}
                          </span>
                        </>
                      )}
                    </div>
                  </div>
                  <span
                    className={`badge text-[10px] ${
                      t.status === 'completed'
                        ? 'badge-success'
                        : t.status === 'in_progress'
                        ? 'badge-high'
                        : 'badge-low'
                    }`}
                  >
                    {t.status || 'Pending'}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Operational Issues: HelpDesk & Data Quality */}
        <div className="space-y-6">
          {/* HelpDesk Tickets */}
          <div className="glass-panel">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-base font-semibold text-gray-900 flex items-center gap-2">
                <Headphones className="w-4 h-4 text-purple-600" />
                HelpDesk Operations
              </h3>
              <a
                href="http://localhost:3005"
                className="text-xs text-blue-600 hover:text-blue-800 flex items-center gap-0.5"
              >
                HelpDesk <ArrowUpRight className="w-3 h-3" />
              </a>
            </div>

            {tickets.length === 0 ? (
              <p className="text-xs text-gray-500 py-4 text-center">No open tickets assigned.</p>
            ) : (
              <div className="space-y-2">
                {tickets.slice(0, 4).map((tk: any) => (
                  <div
                    key={tk.id}
                    onClick={() => openObjectContext('ticket', tk.id, { title: tk.title })}
                    className="p-2.5 rounded border border-gray-100 bg-gray-50/50 hover:bg-purple-50/50 cursor-pointer transition-colors"
                  >
                    <div className="text-xs font-medium text-gray-900 truncate">{tk.title}</div>
                    <div className="flex items-center justify-between text-[11px] text-gray-500 mt-1">
                      <span>{tk.status}</span>
                      <span className="font-semibold text-amber-700">{tk.priority}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Data Quality Issues */}
          <div className="glass-panel">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-base font-semibold text-gray-900 flex items-center gap-2">
                <Database className="w-4 h-4 text-emerald-600" />
                Data Quality Issues
              </h3>
              <span className="text-xs text-gray-400">{dataQuality.length}</span>
            </div>

            {dataQuality.length === 0 ? (
              <p className="text-xs text-gray-500 py-4 text-center">
                Data quality validation passed.
              </p>
            ) : (
              <div className="space-y-2">
                {dataQuality.slice(0, 4).map((dq: any) => (
                  <div
                    key={dq.id}
                    onClick={() => openObjectContext('quality_issue', dq.id, { title: dq.title })}
                    className="p-2.5 rounded border border-yellow-200 bg-yellow-50/60 hover:bg-yellow-100/60 cursor-pointer transition-colors"
                  >
                    <div className="text-xs font-semibold text-yellow-900">{dq.title}</div>
                    <div className="text-[11px] text-yellow-800 mt-0.5">{dq.message || dq.description}</div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
