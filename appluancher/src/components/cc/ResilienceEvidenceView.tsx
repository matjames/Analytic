import React, { useEffect, useState, useCallback } from 'react';
import {
  ShieldCheck,
  FileCheck,
  CheckCircle2,
  RefreshCw,
  Filter,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';

const ENTERPRISE_CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) ||
  'http://localhost:8096';

interface ResilienceEvidence {
  id: string;
  activity_type: string;
  target_service: string;
  actor: string;
  result_status: string;
  metrics: Record<string, unknown>;
  logs_summary: string;
  checksum_digest: string;
  audit_reference: string;
  verification_hash: string;
  created_at: string;
}

export const ResilienceEvidenceView: React.FC = () => {
  const [evidenceList, setEvidenceList] = useState<ResilienceEvidence[]>([]);
  const [activityFilter, setActivityFilter] = useState<string>('ALL');
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const fetchEvidence = useCallback(async () => {
    setRefreshing(true);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      const res = await fetch(`${ENTERPRISE_CORE}/api/evidence`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setEvidenceList(data.evidence ?? []);
      }
    } catch (e) {
      console.error('Failed to fetch evidence:', e);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchEvidence();
  }, [fetchEvidence]);

  const filtered = evidenceList.filter((e) => {
    if (activityFilter === 'ALL') return true;
    return e.activity_type === activityFilter;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <span className="px-2.5 py-0.5 text-xs font-bold bg-blue-100 text-blue-800 rounded-full">
              Cryptographic Non-Repudiation
            </span>
            <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
              <ShieldCheck className="w-6 h-6 text-blue-600" />
              Institutional Resilience Evidence Vault
            </h2>
          </div>
          <p className="text-sm text-gray-500 mt-1">
            Certified evidence records with immutable SHA-256 verification hashes for audits, DR drills, and restore tests.
          </p>
        </div>
        <button
          onClick={fetchEvidence}
          disabled={refreshing}
          className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors shadow-sm"
        >
          <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {/* Filter Tabs */}
      <div className="flex flex-wrap items-center gap-2 bg-white p-3 rounded-xl border border-gray-200 shadow-sm text-xs font-medium">
        <span className="text-gray-500 font-semibold uppercase flex items-center gap-1.5 mr-2">
          <Filter className="w-3.5 h-3.5" /> Activity Filter:
        </span>
        {['ALL', 'RECOVERY_DRILL', 'BACKUP_RESTORE_TEST', 'INCIDENT_RECOVERY', 'INTEGRITY_AUDIT'].map(
          (act) => (
            <button
              key={act}
              onClick={() => setActivityFilter(act)}
              className={`px-3 py-1.5 rounded-lg transition-colors ${
                activityFilter === act
                  ? 'bg-blue-600 text-white font-bold shadow-sm'
                  : 'text-gray-600 hover:bg-gray-100'
              }`}
            >
              {act.replace(/_/g, ' ')}
            </button>
          )
        )}
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64 text-gray-400">
          <RefreshCw className="w-6 h-6 animate-spin mr-2" />
          Loading certified evidence records…
        </div>
      ) : (
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
          <div className="px-5 py-4 border-b border-gray-100 font-bold text-sm text-gray-800">
            Certified Evidence Records ({filtered.length})
          </div>

          <div className="divide-y divide-gray-100">
            {filtered.length === 0 ? (
              <div className="p-8 text-center text-xs text-gray-400">No evidence records matching filter.</div>
            ) : (
              filtered.map((e) => {
                const isExpanded = expandedId === e.id;
                return (
                  <div key={e.id} className="p-5 hover:bg-gray-50/60 transition-colors">
                    <div
                      className="flex items-start justify-between gap-4 cursor-pointer"
                      onClick={() => setExpandedId(isExpanded ? null : e.id)}
                    >
                      <div className="flex items-start gap-3">
                        <div className="p-2.5 rounded-xl bg-blue-50 text-blue-700 mt-0.5">
                          <FileCheck className="w-5 h-5" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-mono font-bold text-blue-700 bg-blue-50 px-2 py-0.5 rounded">
                              {e.id}
                            </span>
                            <span className="text-xs font-bold px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-700 border border-emerald-200 flex items-center gap-1">
                              <CheckCircle2 className="w-3 h-3" />
                              {e.result_status}
                            </span>
                          </div>
                          <h4 className="font-bold text-sm text-gray-900 mt-1">{e.activity_type.replace(/_/g, ' ')}</h4>
                          <div className="text-xs text-gray-500 mt-0.5">
                            Target Service: <strong className="text-gray-800">{e.target_service}</strong> • Actor: {e.actor}
                          </div>
                        </div>
                      </div>

                      <div className="text-right flex items-center gap-3 flex-shrink-0">
                        <div className="text-xs text-gray-400 font-mono">
                          {new Date(e.created_at).toLocaleDateString()} {new Date(e.created_at).toLocaleTimeString()}
                        </div>
                        {isExpanded ? (
                          <ChevronDown className="w-4 h-4 text-gray-400" />
                        ) : (
                          <ChevronRight className="w-4 h-4 text-gray-400" />
                        )}
                      </div>
                    </div>

                    {/* Expandable Non-Repudiation Details */}
                    {isExpanded && (
                      <div className="mt-4 pt-4 border-t border-gray-100 grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                        <div className="space-y-2">
                          <div className="bg-gray-50 p-3 rounded-lg border border-gray-100">
                            <div className="font-bold text-gray-700">Audit Logs & Summary</div>
                            <div className="text-gray-600 mt-1 leading-relaxed">{e.logs_summary}</div>
                          </div>
                          <div className="bg-gray-50 p-3 rounded-lg border border-gray-100">
                            <div className="font-bold text-gray-700">Metrics Snapshot</div>
                            <pre className="text-[11px] font-mono text-gray-600 mt-1 overflow-x-auto">
                              {JSON.stringify(e.metrics, null, 2)}
                            </pre>
                          </div>
                        </div>

                        <div className="space-y-2">
                          <div className="bg-slate-900 text-white p-3 rounded-lg space-y-2 font-mono text-[11px]">
                            <div>
                              <div className="text-[10px] uppercase font-bold text-indigo-300">
                                SHA-256 Verification Hash
                              </div>
                              <div className="text-emerald-400 truncate mt-0.5">{e.verification_hash}</div>
                            </div>
                            <div>
                              <div className="text-[10px] uppercase font-bold text-indigo-300">
                                Checksum Digest
                              </div>
                              <div className="text-cyan-300 truncate mt-0.5">{e.checksum_digest}</div>
                            </div>
                            <div>
                              <div className="text-[10px] uppercase font-bold text-indigo-300">
                                Immutable Audit Reference
                              </div>
                              <div className="text-gray-300 truncate mt-0.5">{e.audit_reference}</div>
                            </div>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}
    </div>
  );
};
