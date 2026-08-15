import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { Sparkles, ShieldCheck, CheckCircle2 } from 'lucide-react';
import { EnterpriseIntelligencePanel } from '@components/EnterpriseIntelligencePanel';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const MyBriefingView: React.FC<{ user: User }> = ({ user }) => {
  const [briefing, setBriefing] = useState<any[]>([]);
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

    fetch(`${API_BASE}/api/ai/v1/briefings`, { headers })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((res) => {
        if (cancelled) return;
        setBriefing(res?.findings || []);
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [user.id, user.role]);

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
          <Sparkles className="w-6 h-6 text-blue-600" />
          AI Institutional Briefing
        </h2>
        <p className="text-sm text-gray-500 mt-1">
          Evidence-grounded daily intelligence summary curated specifically for your role, responsibilities, and assigned objects.
        </p>
      </div>

      {/* Briefing Card */}
      <div className="glass-panel border-blue-200 bg-gradient-to-br from-blue-50/70 via-white to-white">
        <div className="flex items-center justify-between mb-4">
          <div className="text-xs font-bold uppercase tracking-widest text-blue-700">
            Daily Intelligence Briefing
          </div>
          <span className="text-xs px-2.5 py-0.5 rounded-full bg-blue-100 text-blue-800 font-semibold flex items-center gap-1">
            <ShieldCheck className="w-3.5 h-3.5" /> Governed AI
          </span>
        </div>

        {loading ? (
          <div className="space-y-3 animate-pulse py-4">
            <div className="h-4 bg-blue-100 rounded w-3/4" />
            <div className="h-4 bg-blue-100 rounded w-1/2" />
            <div className="h-4 bg-blue-100 rounded w-5/6" />
          </div>
        ) : briefing.length === 0 ? (
          <div className="py-6 text-center text-sm text-gray-600">
            <CheckCircle2 className="w-6 h-6 text-green-500 mx-auto mb-2" />
            <p className="font-medium">All monitored operations within standard parameters.</p>
            <p className="text-xs text-gray-400">No urgent risks or unhandled deviations flagged.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {briefing.map((item, idx) => (
              <div
                key={idx}
                className="p-3.5 rounded-lg border border-blue-100 bg-white/80 flex items-start justify-between gap-4"
              >
                <div>
                  <div className="text-xs font-semibold uppercase tracking-wider text-blue-600">
                    {item.kind || 'Finding'} • Confidence: {item.confidence || 'Grounded'}
                  </div>
                  <p className="text-sm text-gray-900 font-medium mt-1">{item.statement}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Embedded Deep Query Panel */}
      <EnterpriseIntelligencePanel user={user} />
    </div>
  );
};
