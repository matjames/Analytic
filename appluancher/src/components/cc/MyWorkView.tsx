import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { CheckSquare, Clock, CheckCircle2, ChevronRight } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const MyWorkView: React.FC<{ user: User }> = ({ user }) => {
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

    fetch(`${API_BASE}/api/my-work?user_id=${encodeURIComponent(user.id)}`, { headers })
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

  const tasks = data?.tasks || data?.my_tasks || [];

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
          <CheckSquare className="w-6 h-6 text-blue-600" />
          My Work Queue
        </h2>
        <p className="text-sm text-gray-500 mt-1">
          Your consolidated personal work assignments, PMS milestones, RMS review tasks, and enterprise actions.
        </p>
      </div>

      {/* Task List */}
      <div className="glass-panel">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Personal Work Queue ({tasks.length})</h3>
        </div>

        {tasks.length === 0 ? (
          <div className="py-12 text-center">
            <CheckCircle2 className="w-10 h-10 text-green-500 mx-auto mb-2" />
            <p className="text-base font-semibold text-gray-900">All Caught Up!</p>
            <p className="text-xs text-gray-500">You have no tasks assigned to your queue.</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {tasks.map((t: any) => (
              <div
                key={t.id}
                onClick={() => openObjectContext('task', t.id, { title: t.title })}
                className="py-3.5 flex items-start justify-between gap-4 p-2 hover:bg-blue-50/50 rounded-xl cursor-pointer transition-colors group"
              >
                <div>
                  <div className="text-sm font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">{t.title}</div>
                  <div className="flex items-center gap-2 text-xs text-gray-500 mt-1">
                    <span className="font-semibold uppercase text-blue-700">{t.source || 'Enterprise'}</span>
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
                <div className="flex items-center gap-3">
                  <span
                    className={`badge text-[10px] ${
                      t.status === 'completed' ? 'badge-success' : 'badge-low'
                    }`}
                  >
                    {t.status || 'Assigned'}
                  </span>
                  <ChevronRight className="w-4 h-4 text-gray-400 group-hover:translate-x-1 transition-transform" />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
