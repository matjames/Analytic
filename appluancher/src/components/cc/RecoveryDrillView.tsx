import React, { useEffect, useState, useCallback } from 'react';
import {
  Play,
  CheckCircle2,
  RefreshCw,
  FileCheck,
  Flame,
} from 'lucide-react';

const ENTERPRISE_CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) ||
  'http://localhost:8096';

interface DrillStep {
  id: string;
  step_number: number;
  name: string;
  description: string;
  action_type: string;
  status: string;
  duration_ms: number;
  output: string;
  error_message?: string;
}

interface RecoveryDrill {
  id: string;
  title: string;
  service_id: string;
  scenario: string;
  mode: string;
  status: string;
  rto_target_sec: number;
  rpo_target_sec: number;
  measured_rto_sec: number;
  measured_rpo_sec: number;
  drill_result: string;
  started_at?: string;
  completed_at?: string;
  conducted_by: string;
  evidence_id?: string;
  summary: string;
  steps?: DrillStep[];
}

export const RecoveryDrillView: React.FC = () => {
  const [drills, setDrills] = useState<RecoveryDrill[]>([]);
  const [selectedDrill, setSelectedDrill] = useState<RecoveryDrill | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [executing, setExecuting] = useState(false);

  const fetchDrills = useCallback(async () => {
    setRefreshing(true);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      const res = await fetch(`${ENTERPRISE_CORE}/api/recovery/drills`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setDrills(data.drills ?? []);
        if (data.drills && data.drills.length > 0 && !selectedDrill) {
          setSelectedDrill(data.drills[0]);
        }
      }
    } catch (e) {
      console.error('Failed to fetch drills:', e);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [selectedDrill]);

  useEffect(() => {
    fetchDrills();
  }, [fetchDrills]);

  const handleStartDrill = async (drillId: string) => {
    setExecuting(true);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      const res = await fetch(`${ENTERPRISE_CORE}/api/recovery/drills/${drillId}/start`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        if (data.drill) {
          setSelectedDrill(data.drill);
        }
        await fetchDrills();
      }
    } catch (e) {
      console.error('Failed to start recovery drill:', e);
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
            <span className="px-2.5 py-0.5 text-xs font-bold bg-amber-100 text-amber-800 rounded-full">
              Controlled Failure Injection
            </span>
            <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
              <Flame className="w-6 h-6 text-amber-600" />
              Disaster Recovery Drill Engine
            </h2>
          </div>
          <p className="text-sm text-gray-500 mt-1">
            Simulate broker outages, database failovers, and service degradation to continuously measure and verify RTO compliance.
          </p>
        </div>
        <button
          onClick={fetchDrills}
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
          Loading recovery drills…
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Drill Catalog */}
          <div className="lg:col-span-1 bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden flex flex-col h-[620px]">
            <div className="px-4 py-3 bg-gray-50 border-b border-gray-100 font-bold text-xs text-gray-700 uppercase tracking-wider">
              Standard Scenarios ({drills.length})
            </div>
            <div className="overflow-y-auto divide-y divide-gray-100 flex-1">
              {drills.map((d) => {
                const isSelected = selectedDrill?.id === d.id;
                const resultColor =
                  d.drill_result === 'SUCCESS'
                    ? 'text-emerald-700 bg-emerald-50 border-emerald-200'
                    : d.drill_result === 'BREACHED'
                    ? 'text-amber-700 bg-amber-50 border-amber-200'
                    : 'text-gray-700 bg-gray-50 border-gray-200';

                return (
                  <div
                    key={d.id}
                    onClick={() => setSelectedDrill(d)}
                    className={`p-4 cursor-pointer transition-colors ${
                      isSelected ? 'bg-amber-50/70 border-l-4 border-amber-600' : 'hover:bg-gray-50'
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-[10px] font-mono text-gray-500">{d.scenario}</span>
                      <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${resultColor}`}>
                        {d.drill_result}
                      </span>
                    </div>
                    <div className="font-semibold text-sm text-gray-900 mt-1">{d.title}</div>
                    <div className="text-xs text-gray-500 mt-0.5">Target: {d.service_id}</div>
                    <div className="flex items-center justify-between text-[11px] text-gray-400 mt-2 font-mono">
                      <span>RTO Target: {d.rto_target_sec}s</span>
                      <span className="text-emerald-600 font-bold">Measured: {d.measured_rto_sec}s</span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Drill Detail & Execution Checklist */}
          <div className="lg:col-span-2 bg-white rounded-xl border border-gray-200 shadow-sm p-6 flex flex-col justify-between">
            {selectedDrill ? (
              <div className="space-y-6">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <span className="text-xs font-mono text-amber-700 font-bold bg-amber-50 px-2 py-0.5 rounded">
                      {selectedDrill.id}
                    </span>
                    <h3 className="text-lg font-bold text-gray-900 mt-1">{selectedDrill.title}</h3>
                    <div className="text-xs text-gray-500 mt-1">
                      Mode: <strong className="text-gray-800">{selectedDrill.mode}</strong> • Target: {selectedDrill.service_id}
                    </div>
                  </div>
                  <button
                    disabled={executing}
                    onClick={() => handleStartDrill(selectedDrill.id)}
                    className="flex items-center gap-2 px-4 py-2 bg-amber-600 text-white rounded-lg text-xs font-bold hover:bg-amber-700 disabled:opacity-40 shadow-sm transition-colors"
                  >
                    <Play className={`w-3.5 h-3.5 ${executing ? 'animate-spin' : ''}`} />
                    {executing ? 'Simulating…' : 'Execute Drill'}
                  </button>
                </div>

                {/* RTO vs Measured Benchmark Card */}
                <div className="bg-gradient-to-r from-slate-900 to-indigo-950 rounded-xl p-4 text-white">
                  <div className="grid grid-cols-3 gap-4 text-center">
                    <div>
                      <div className="text-[10px] font-bold uppercase text-indigo-300">RTO Target</div>
                      <div className="text-xl font-black mt-0.5">{selectedDrill.rto_target_sec}s</div>
                    </div>
                    <div>
                      <div className="text-[10px] font-bold uppercase text-indigo-300">Measured RTO</div>
                      <div className="text-xl font-black mt-0.5 text-emerald-400">{selectedDrill.measured_rto_sec}s</div>
                    </div>
                    <div>
                      <div className="text-[10px] font-bold uppercase text-indigo-300">Result Status</div>
                      <div className="text-sm font-bold mt-1 text-emerald-300">{selectedDrill.drill_result}</div>
                    </div>
                  </div>
                </div>

                {/* Step Verification Checklist */}
                <div>
                  <div className="text-xs font-bold text-gray-700 uppercase tracking-wide mb-3 flex items-center gap-2">
                    <CheckCircle2 className="w-4 h-4 text-emerald-600" />
                    Automated Multi-Step Verification Checklist
                  </div>
                  <div className="space-y-2">
                    {selectedDrill.steps && selectedDrill.steps.length > 0 ? (
                      selectedDrill.steps.map((step) => (
                        <div
                          key={step.id}
                          className="flex items-start gap-3 p-3 rounded-lg border border-gray-100 bg-gray-50/70 text-xs"
                        >
                          <div className="w-5 h-5 rounded-full bg-emerald-100 text-emerald-700 font-bold flex items-center justify-center text-[10px] flex-shrink-0 mt-0.5">
                            {step.step_number}
                          </div>
                          <div className="flex-1">
                            <div className="flex items-center justify-between font-semibold text-gray-900">
                              <span>{step.name}</span>
                              <span className="font-mono text-[10px] text-gray-400">{step.duration_ms}ms</span>
                            </div>
                            <div className="text-gray-500 text-[11px] mt-0.5">{step.output || step.description}</div>
                          </div>
                        </div>
                      ))
                    ) : (
                      <div className="text-xs text-gray-400">No steps defined.</div>
                    )}
                  </div>
                </div>

                {/* Evidence Certification Tag */}
                {selectedDrill.evidence_id && (
                  <div className="p-3 bg-emerald-50 rounded-lg border border-emerald-200 flex items-center justify-between text-xs">
                    <div className="flex items-center gap-2 text-emerald-800">
                      <FileCheck className="w-4 h-4 text-emerald-600" />
                      <span>Certified Evidence Token: <strong className="font-mono">{selectedDrill.evidence_id}</strong></span>
                    </div>
                    <span className="text-[10px] font-bold uppercase text-emerald-600 bg-white px-2 py-0.5 rounded border border-emerald-200">
                      Immutable
                    </span>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex items-center justify-center h-64 text-xs text-gray-400">Select a drill scenario to view details.</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
