import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { FlaskConical, ArrowUpRight, ChevronRight } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const ResearchView: React.FC<{ user: User }> = ({ user }) => {
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

    fetch(`${API_BASE}/api/command-centre/research?user_id=${encodeURIComponent(user.id)}`, {
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

  const research = data?.research || [];

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <FlaskConical className="w-6 h-6 text-purple-600" />
            Research Command Centre (RMS)
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Aggregated institutional research portfolio from RMS — study lifecycle, ethics review, grant funding, and publication tracking.
          </p>
        </div>
        <a
          href="http://localhost:3011"
          className="btn btn-primary text-xs bg-purple-600 hover:bg-purple-700"
        >
          Open RMS Application <ArrowUpRight className="w-3.5 h-3.5" />
        </a>
      </div>

      {/* Research Studies List */}
      <div className="glass-panel">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Governed Studies ({research.length})</h3>
          <span className="text-xs text-gray-500">Live RMS synchronization</span>
        </div>

        {research.length === 0 ? (
          <p className="text-sm text-gray-500 py-12 text-center">
            No research studies found in RMS or RMS service is unreachable.
          </p>
        ) : (
          <div className="divide-y divide-gray-100">
            {research.map((r: any) => (
              <div
                key={r.id}
                onClick={() => openObjectContext('research', r.id, { title: r.name })}
                className="py-4 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:bg-purple-50/40 p-3 rounded-xl cursor-pointer transition-colors group"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-base font-semibold text-gray-900 group-hover:text-purple-700 transition-colors">
                      {r.name}
                    </span>
                    <span className="text-xs font-mono px-2 py-0.5 rounded bg-purple-50 text-purple-700">
                      {r.code || r.id}
                    </span>
                  </div>
                  <p className="text-xs text-gray-500 mt-1 line-clamp-2">
                    {r.abstract || r.description || 'Governed research study under StatGate RMS'}
                  </p>
                  <div className="flex items-center gap-4 text-xs text-gray-500 mt-2">
                    <span>Stage: <strong className="text-purple-800">{r.stage || 'Proposal'}</strong></span>
                    {r.pi && <span>PI: <strong className="text-gray-800">{r.pi}</strong></span>}
                  </div>
                </div>

                <div className="flex items-center gap-4">
                  <span className="badge badge-medium text-xs">
                    {r.ethics_status || r.stage || 'In Review'}
                  </span>
                  <div className="p-2 text-purple-600 group-hover:bg-purple-100 rounded-lg transition-colors flex items-center gap-1 text-xs font-semibold">
                    <span>Context</span>
                    <ChevronRight className="w-4 h-4" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
