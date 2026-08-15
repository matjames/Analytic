import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { AlertOctagon, CheckCircle2 } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const ActionCentreView: React.FC<{ user: User }> = ({ user }) => {
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

    fetch(`${API_BASE}/api/action-centre?user_id=${encodeURIComponent(user.id)}`, { headers })
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
        <div className="h-96 bg-gray-100 rounded-xl" />
      </div>
    );
  }

  const approvals = data?.approvals || [];
  const tasks = data?.tasks || [];
  const alerts = data?.alerts || [];
  const summary = data?.summary || {};

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
          <AlertOctagon className="w-6 h-6 text-red-600" />
          Universal Action Centre
        </h2>
        <p className="text-sm text-gray-500 mt-1">
          Aggregated actionable responsibilities requiring your urgent decision, review, or task execution.
        </p>
      </div>

      {/* Attention Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-5">
        <div className="stat-card border-l-4 border-l-gray-900 shadow-sm">
          <span className="stat-label">Total Action Items</span>
          <span className="stat-val">{summary.total_items ?? (approvals.length + tasks.length + alerts.length)}</span>
          <span className="text-xs text-gray-400 mt-1">Across all domains</span>
        </div>

        <div className="stat-card border-l-4 border-l-red-600 shadow-sm">
          <span className="stat-label">Critical Priority</span>
          <span className="stat-val text-red-600">{summary.critical_count ?? 0}</span>
          <span className="text-xs text-red-700 font-medium mt-1">Immediate intervention</span>
        </div>

        <div className="stat-card border-l-4 border-l-amber-500 shadow-sm">
          <span className="stat-label">High Priority</span>
          <span className="stat-val text-amber-600">{summary.high_count ?? 0}</span>
          <span className="text-xs text-gray-400 mt-1">Action due today</span>
        </div>

        <div className="stat-card border-l-4 border-l-blue-600 shadow-sm">
          <span className="stat-label">Pending Approvals</span>
          <span className="stat-val text-blue-600">{approvals.length}</span>
          <span className="text-xs text-gray-400 mt-1">Governed sign-offs</span>
        </div>
      </div>

      {/* Action Items List */}
      <div className="glass-panel">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Urgent Action Items</h3>
        </div>

        {approvals.length === 0 && tasks.length === 0 && alerts.length === 0 ? (
          <div className="py-12 text-center">
            <CheckCircle2 className="w-10 h-10 text-green-500 mx-auto mb-2" />
            <p className="text-base font-semibold text-gray-900">Action Centre is Clear</p>
            <p className="text-xs text-gray-500">You have no outstanding critical tasks or pending approvals.</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {/* Approvals */}
            {approvals.map((item: any) => (
              <div
                key={`appr-${item.id}`}
                onClick={() => openObjectContext('approval', item.id, { title: item.title })}
                className="py-3.5 flex items-start justify-between gap-4 p-2 hover:bg-blue-50/50 rounded-xl cursor-pointer transition-colors group"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">
                      {item.title}
                    </span>
                    <span className="badge badge-high text-[10px]">Pending Approval</span>
                  </div>
                  <div className="text-xs text-gray-500 mt-1">
                    Source: <strong className="uppercase">{item.source || 'Enterprise'}</strong> • {item.description || 'Approval required to advance workflow'}
                  </div>
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    openObjectContext('approval', item.id, { title: item.title });
                  }}
                  className="btn btn-primary text-xs flex-shrink-0"
                >
                  Review & Approve
                </button>
              </div>
            ))}

            {/* Tasks */}
            {tasks.map((task: any) => (
              <div
                key={`task-${task.id}`}
                onClick={() => openObjectContext('task', task.id, { title: task.title })}
                className="py-3.5 flex items-start justify-between gap-4 p-2 hover:bg-blue-50/50 rounded-xl cursor-pointer transition-colors group"
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">
                      {task.title}
                    </span>
                    <span className="badge badge-medium text-[10px]">Assigned Task</span>
                  </div>
                  <div className="text-xs text-gray-500 mt-1">
                    Source: <strong className="uppercase">{task.source || 'Enterprise'}</strong> {task.due_at ? `• Due: ${task.due_at}` : ''}
                  </div>
                </div>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    openObjectContext('task', task.id, { title: task.title });
                  }}
                  className="btn btn-secondary text-xs flex-shrink-0"
                >
                  View Details
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
