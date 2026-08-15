import React, { useEffect, useState } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import { ShieldCheck, ShieldAlert, CheckCircle2, ArrowUpRight, Scale } from 'lucide-react';

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

export const GovernanceView: React.FC<{ user: User }> = ({ user }) => {
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

    fetch(`${API_BASE}/api/command-centre/governance?user_id=${encodeURIComponent(user.id)}`, {
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
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="h-64 bg-gray-100 rounded-xl" />
          <div className="h-64 bg-gray-100 rounded-xl" />
        </div>
      </div>
    );
  }

  const risks = data?.risks || [];
  const controls = data?.controls || [];
  const findings = data?.findings || [];

  return (
    <div className="space-y-8">
      {/* View Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <ShieldCheck className="w-6 h-6 text-indigo-600" />
            Institutional Governance & Compliance (StatGovernance)
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Enterprise policies, compliance obligations, internal controls library, audit findings, and risk registers.
          </p>
        </div>
        <a
          href="http://localhost:3012"
          className="btn btn-primary text-xs bg-indigo-600 hover:bg-indigo-700"
        >
          Open StatGovernance <ArrowUpRight className="w-3.5 h-3.5" />
        </a>
      </div>

      {/* Grid: Risks, Controls, Audit Findings */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Risks */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-base font-semibold text-gray-900 flex items-center gap-2">
              <ShieldAlert className="w-4 h-4 text-amber-500" />
              Risk Register ({risks.length})
            </h3>
          </div>

          {risks.length === 0 ? (
            <p className="text-xs text-gray-500 py-6 text-center">No active risks registered.</p>
          ) : (
            <div className="space-y-2.5">
              {risks.slice(0, 5).map((r: any) => (
                <div
                  key={r.id}
                  onClick={() => openObjectContext('risk', r.id, { title: r.title || r.name })}
                  className="p-2.5 rounded border border-gray-200 hover:bg-amber-50/50 cursor-pointer transition-colors"
                >
                  <div className="text-xs font-semibold text-gray-900 truncate">{r.title || r.name}</div>
                  <div className="flex items-center justify-between text-[11px] text-gray-500 mt-1">
                    <span>{r.category || 'Institutional'}</span>
                    <span className="badge badge-medium text-[10px]">{r.severity || 'Assessed'}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Controls */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-base font-semibold text-gray-900 flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-green-600" />
              Controls Library ({controls.length})
            </h3>
          </div>

          {controls.length === 0 ? (
            <p className="text-xs text-gray-500 py-6 text-center">No controls registered.</p>
          ) : (
            <div className="space-y-2.5">
              {controls.slice(0, 5).map((c: any) => (
                <div
                  key={c.id}
                  onClick={() => openObjectContext('control', c.id, { title: c.title || c.name })}
                  className="p-2.5 rounded border border-gray-200 hover:bg-green-50/50 cursor-pointer transition-colors"
                >
                  <div className="text-xs font-semibold text-gray-900 truncate">{c.title || c.name}</div>
                  <div className="flex items-center justify-between text-[11px] text-gray-500 mt-1">
                    <span>{c.control_type || 'Internal Control'}</span>
                    <span className="badge badge-success text-[10px]">{c.status || 'Active'}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Audit Findings */}
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-base font-semibold text-gray-900 flex items-center gap-2">
              <Scale className="w-4 h-4 text-purple-600" />
              Audit Findings ({findings.length})
            </h3>
          </div>

          {findings.length === 0 ? (
            <p className="text-xs text-gray-500 py-6 text-center">No outstanding audit findings.</p>
          ) : (
            <div className="space-y-2.5">
              {findings.slice(0, 5).map((f: any) => (
                <div
                  key={f.id}
                  onClick={() => openObjectContext('finding', f.id, { title: f.title })}
                  className="p-2.5 rounded border border-red-100 bg-red-50/40 hover:bg-red-50 cursor-pointer transition-colors"
                >
                  <div className="text-xs font-semibold text-gray-900 truncate">{f.title}</div>
                  <div className="flex items-center justify-between text-[11px] text-gray-500 mt-1">
                    <span>{f.status || 'Open'}</span>
                    <span className="badge badge-critical text-[10px]">{f.severity || 'Finding'}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
