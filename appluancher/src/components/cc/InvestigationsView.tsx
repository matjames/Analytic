import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { Brain, ChevronRight } from 'lucide-react';
import { EnterpriseIntelligencePanel } from '@components/EnterpriseIntelligencePanel';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const InvestigationsView: React.FC<{ user: User }> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [investigations, setInvestigations] = useState<any[]>([]);
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

    fetch(`${API_BASE}/api/ai/v1/investigations`, { headers })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((res) => {
        if (cancelled) return;
        setInvestigations(res?.investigations || res || []);
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [user.id, user.role]);

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <Brain className="w-6 h-6 text-indigo-600" />
            Governed AI Investigations & Evidence Synthesis
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Human-initiated analytical investigations grounded in authorized enterprise records with complete source lineage.
          </p>
        </div>
      </div>

      {/* Embedded Governed AI Panel */}
      <EnterpriseIntelligencePanel user={user} />

      {/* Past Investigations List */}
      <div className="glass-panel">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">Recorded Investigations</h3>
          <span className="text-xs text-gray-400">
            {Array.isArray(investigations) ? investigations.length : 0} items
          </span>
        </div>

        {loading ? (
          <div className="space-y-2 py-4">
            <div className="h-10 bg-gray-100 rounded animate-pulse" />
            <div className="h-10 bg-gray-100 rounded animate-pulse" />
          </div>
        ) : !Array.isArray(investigations) || investigations.length === 0 ? (
          <p className="text-sm text-gray-500 py-8 text-center">
            No investigations recorded yet. Use the panel above to analyze evidence and record an investigation.
          </p>
        ) : (
          <div className="divide-y divide-gray-100">
            {investigations.map((inv: any) => (
              <div
                key={inv.id}
                onClick={() => openObjectContext('investigation', inv.id, { title: inv.title || inv.question })}
                className="py-3 flex items-start justify-between gap-4 p-2 hover:bg-indigo-50/50 rounded-lg cursor-pointer transition-colors group"
              >
                <div>
                  <div className="text-sm font-semibold text-gray-900 group-hover:text-indigo-800 transition-colors">
                    {inv.title || inv.question}
                  </div>
                  <div className="text-xs text-gray-500 mt-1">
                    Assignee: {inv.assignee || user.name} • Status: {inv.status || 'Active'}
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className="badge badge-medium text-[10px]">
                    {inv.status || 'Investigating'}
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
