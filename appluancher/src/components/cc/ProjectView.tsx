import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { FolderGit2, ArrowUpRight, ChevronRight } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const ProjectView: React.FC<{ user: User }> = ({ user }) => {
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

    fetch(`${API_BASE}/api/command-centre/projects?user_id=${encodeURIComponent(user.id)}`, {
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

  const projects = data?.projects || [];

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <FolderGit2 className="w-6 h-6 text-blue-600" />
            Projects Command Centre (PMS)
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Aggregated institutional project portfolio from PMS — milestones, work planning, and deliverable progress.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <a
            href="http://localhost:3010"
            className="btn btn-primary text-xs"
          >
            Open PMS Application <ArrowUpRight className="w-3.5 h-3.5" />
          </a>
        </div>
      </div>

      {/* Projects List */}
      <div className="glass-panel">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Institutional Projects ({projects.length})</h3>
          <span className="text-xs text-gray-500">Live PMS synchronization</span>
        </div>

        {projects.length === 0 ? (
          <p className="text-sm text-gray-500 py-12 text-center">
            No projects found in PMS or PMS service is unreachable.
          </p>
        ) : (
          <div className="divide-y divide-gray-100">
            {projects.map((p: any) => (
              <div
                key={p.id}
                onClick={() => openObjectContext('project', p.id, { title: p.name })}
                className="py-4 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:bg-blue-50/40 p-3 rounded-xl cursor-pointer transition-colors group"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-base font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">
                      {p.name}
                    </span>
                    <span className="text-xs font-mono px-2 py-0.5 rounded bg-gray-100 text-gray-600">
                      {p.code || p.id}
                    </span>
                  </div>
                  <p className="text-xs text-gray-500 mt-1 line-clamp-2">
                    {p.description || 'Project managed under StatGate PMS'}
                  </p>
                  <div className="flex items-center gap-4 text-xs text-gray-500 mt-2">
                    <span>Stage: <strong className="text-gray-800">{p.stage || 'Execution'}</strong></span>
                    {p.owner && <span>Owner: <strong className="text-gray-800">{p.owner}</strong></span>}
                  </div>
                </div>

                <div className="flex items-center gap-6">
                  <div className="w-32">
                    <div className="flex justify-between text-xs text-gray-500 mb-1">
                      <span>Progress</span>
                      <span className="font-semibold text-gray-900">{p.progress || 0}%</span>
                    </div>
                    <div className="w-full bg-gray-200 rounded-full h-2 overflow-hidden">
                      <div
                        className="bg-blue-600 h-2 rounded-full"
                        style={{ width: `${Math.min(p.progress || 0, 100)}%` }}
                      />
                    </div>
                  </div>

                  <div className="p-2 text-blue-600 group-hover:bg-blue-100 rounded-lg transition-colors flex items-center gap-1 text-xs font-semibold">
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
