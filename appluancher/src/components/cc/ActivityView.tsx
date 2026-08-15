import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { History, ChevronRight } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const ActivityView: React.FC<{ user: User }> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [activity, setActivity] = useState<any[]>([]);
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

    fetch(`${API_BASE}/api/timeline?limit=50`, { headers })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((res) => {
        if (cancelled) return;
        setActivity(res?.timeline || res || []);
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

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
          <History className="w-6 h-6 text-blue-600" />
          Enterprise Activity Timeline & Sovereign Audit
        </h2>
        <p className="text-sm text-gray-500 mt-1">
          Unified audit trail and real-time event log across all connected applications in StatGate.
        </p>
      </div>

      {/* Timeline List */}
      <div className="glass-panel">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-semibold text-gray-900">Recent Enterprise Events</h3>
          <span className="text-xs text-gray-400">
            {Array.isArray(activity) ? activity.length : 0} events
          </span>
        </div>

        {!Array.isArray(activity) || activity.length === 0 ? (
          <p className="text-sm text-gray-500 py-12 text-center">No recent timeline events recorded.</p>
        ) : (
          <div className="relative border-l-2 border-blue-100 ml-4 space-y-6">
            {activity.map((item: any, idx: number) => (
              <div
                key={idx}
                onClick={() =>
                  item.entity && item.entity_id
                    ? openObjectContext(item.entity, item.entity_id, {
                        title: item.description || item.action,
                      })
                    : null
                }
                className={`relative pl-6 p-2 rounded-xl transition-colors ${
                  item.entity && item.entity_id ? 'hover:bg-blue-50/50 cursor-pointer group' : ''
                }`}
              >
                <span className="absolute -left-[9px] top-3 w-4 h-4 rounded-full bg-blue-600 border-2 border-white" />
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-1">
                  <div className="text-sm font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">
                    {item.action || item.description || 'Enterprise Event'}
                  </div>
                  <span className="text-[11px] text-gray-400">
                    {item.timestamp ? new Date(item.timestamp).toLocaleString() : ''}
                  </span>
                </div>
                <div className="flex items-center justify-between mt-0.5">
                  <div className="flex items-center gap-2 text-xs text-gray-500">
                    <span className="font-semibold uppercase tracking-wider text-blue-700">
                      {item.application || item.source || 'Enterprise'}
                    </span>
                    {item.user && <span>• By {item.user}</span>}
                    {item.entity && <span>• Entity: {item.entity}</span>}
                  </div>
                  {item.entity && item.entity_id && (
                    <span className="text-[10px] text-blue-600 font-semibold flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                      Inspect Context <ChevronRight className="w-3 h-3" />
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
