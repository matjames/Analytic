import React, { useEffect, useState, useCallback } from 'react';
import {
  BookOpen,
  Play,
  AlertTriangle,
  RefreshCw,
  FileCheck,
  ListOrdered,
  RotateCcw,
  PhoneCall,
} from 'lucide-react';

const ENTERPRISE_CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) ||
  'http://localhost:8096';

interface Runbook {
  id: string;
  title: string;
  incident_type: string;
  affected_services: string[];
  detection_method: string;
  immediate_actions: string[];
  recovery_steps: string[];
  validation_checks: string[];
  rollback_procedure: string;
  escalation_path: string;
  evidence_required: string;
  closure_criteria: string;
  version: number;
}

export const RunbookView: React.FC = () => {
  const [runbooks, setRunbooks] = useState<Runbook[]>([]);
  const [selectedRunbook, setSelectedRunbook] = useState<Runbook | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [lastExecutionEvidence, setLastExecutionEvidence] = useState<string | null>(null);

  const fetchRunbooks = useCallback(async () => {
    setRefreshing(true);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      const res = await fetch(`${ENTERPRISE_CORE}/api/runbooks`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setRunbooks(data.runbooks ?? []);
        if (data.runbooks && data.runbooks.length > 0 && !selectedRunbook) {
          setSelectedRunbook(data.runbooks[0]);
        }
      }
    } catch (e) {
      console.error('Failed to fetch runbooks:', e);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [selectedRunbook]);

  useEffect(() => {
    fetchRunbooks();
  }, [fetchRunbooks]);

  const handleExecute = async (runbookId: string) => {
    setExecuting(true);
    setLastExecutionEvidence(null);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      const res = await fetch(`${ENTERPRISE_CORE}/api/runbooks/${runbookId}/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ incident_id: '' }),
      });
      if (res.ok) {
        const data = await res.json();
        setLastExecutionEvidence(data.evidence_id);
      }
    } catch (e) {
      console.error('Execute runbook failed:', e);
    } finally {
      setExecuting(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <span className="px-2.5 py-0.5 text-xs font-bold bg-emerald-100 text-emerald-800 rounded-full">
              Standard Operating Procedures
            </span>
            <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
              <BookOpen className="w-6 h-6 text-emerald-600" />
              Operational Recovery Runbooks
            </h2>
          </div>
          <p className="text-sm text-gray-500 mt-1">
            Machine-readable and human-executable recovery runbooks with automated verification steps and proof tokens.
          </p>
        </div>
        <button
          onClick={fetchRunbooks}
          disabled={refreshing}
          className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors shadow-sm"
        >
          <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64 text-gray-400">
          <RefreshCw className="w-6 h-6 animate-spin mr-2" />
          Loading recovery runbook library…
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Runbook Catalog */}
          <div className="lg:col-span-1 bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden flex flex-col h-[650px]">
            <div className="px-4 py-3 bg-gray-50 border-b border-gray-100 font-bold text-xs text-gray-700 uppercase tracking-wider">
              Runbook Catalog ({runbooks.length})
            </div>
            <div className="overflow-y-auto divide-y divide-gray-100 flex-1">
              {runbooks.map((rb) => {
                const isSelected = selectedRunbook?.id === rb.id;
                return (
                  <div
                    key={rb.id}
                    onClick={() => {
                      setSelectedRunbook(rb);
                      setLastExecutionEvidence(null);
                    }}
                    className={`p-4 cursor-pointer transition-colors ${
                      isSelected ? 'bg-emerald-50/70 border-l-4 border-emerald-600' : 'hover:bg-gray-50'
                    }`}
                  >
                    <div className="flex items-center justify-between text-[10px] font-mono text-gray-500">
                      <span>{rb.incident_type}</span>
                      <span className="font-bold text-emerald-700">v{rb.version}.0</span>
                    </div>
                    <div className="font-semibold text-sm text-gray-900 mt-1">{rb.title}</div>
                    <div className="text-xs text-gray-500 mt-1">
                      Services: {rb.affected_services?.join(', ') || 'all'}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Runbook Detail & Steps */}
          <div className="lg:col-span-2 bg-white rounded-xl border border-gray-200 shadow-sm p-6 flex flex-col justify-between">
            {selectedRunbook ? (
              <div className="space-y-6">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <span className="text-xs font-mono text-emerald-700 font-bold bg-emerald-50 px-2 py-0.5 rounded">
                      {selectedRunbook.id}
                    </span>
                    <h3 className="text-lg font-bold text-gray-900 mt-1">{selectedRunbook.title}</h3>
                    <div className="text-xs text-gray-500 mt-1">
                      Detection: <strong className="text-gray-800">{selectedRunbook.detection_method}</strong>
                    </div>
                  </div>
                  <button
                    disabled={executing}
                    onClick={() => handleExecute(selectedRunbook.id)}
                    className="flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg text-xs font-bold hover:bg-emerald-700 disabled:opacity-40 shadow-sm transition-colors"
                  >
                    <Play className={`w-3.5 h-3.5 ${executing ? 'animate-spin' : ''}`} />
                    {executing ? 'Executing…' : 'Execute Runbook'}
                  </button>
                </div>

                {/* Evidence Banner if executed */}
                {lastExecutionEvidence && (
                  <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-lg flex items-center justify-between text-xs">
                    <div className="flex items-center gap-2 text-emerald-800">
                      <FileCheck className="w-4 h-4 text-emerald-600" />
                      <span>Runbook completed. Certified Evidence: <strong className="font-mono">{lastExecutionEvidence}</strong></span>
                    </div>
                  </div>
                )}

                {/* Immediate Containment Actions */}
                <div className="bg-amber-50/60 rounded-xl p-4 border border-amber-200/70">
                  <div className="text-xs font-bold text-amber-900 uppercase tracking-wide mb-2 flex items-center gap-1.5">
                    <AlertTriangle className="w-3.5 h-3.5 text-amber-700" />
                    Immediate Containment Actions
                  </div>
                  <div className="space-y-1 text-xs text-amber-950 font-medium">
                    {selectedRunbook.immediate_actions.map((act, i) => (
                      <div key={i}>{act}</div>
                    ))}
                  </div>
                </div>

                {/* Recovery Steps */}
                <div>
                  <div className="text-xs font-bold text-gray-700 uppercase tracking-wide mb-3 flex items-center gap-1.5">
                    <ListOrdered className="w-4 h-4 text-emerald-600" />
                    Standard Recovery Steps
                  </div>
                  <div className="space-y-2">
                    {selectedRunbook.recovery_steps.map((step, i) => (
                      <div key={i} className="flex items-start gap-3 p-3 bg-gray-50 rounded-lg border border-gray-100 text-xs">
                        <div className="w-5 h-5 rounded-full bg-emerald-100 text-emerald-800 font-bold flex items-center justify-center text-[10px] flex-shrink-0 mt-0.5">
                          {i + 1}
                        </div>
                        <div className="text-gray-800 font-medium">{step}</div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Rollback & Escalation Footer */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs pt-2 border-t border-gray-100">
                  <div className="bg-gray-50 p-3 rounded-lg">
                    <div className="font-bold text-gray-700 flex items-center gap-1">
                      <RotateCcw className="w-3 h-3 text-rose-500" /> Rollback Procedure
                    </div>
                    <div className="text-gray-500 text-[11px] mt-1">{selectedRunbook.rollback_procedure}</div>
                  </div>
                  <div className="bg-gray-50 p-3 rounded-lg">
                    <div className="font-bold text-gray-700 flex items-center gap-1">
                      <PhoneCall className="w-3 h-3 text-indigo-500" /> Escalation Contact
                    </div>
                    <div className="text-gray-500 text-[11px] mt-1">{selectedRunbook.escalation_path}</div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-64 text-xs text-gray-400">Select a runbook to view procedures.</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
