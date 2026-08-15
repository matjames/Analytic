import React, { useEffect, useState, useCallback } from 'react';
import {
  AlertOctagon,
  Clock,
  Filter,
  RefreshCw,
  SlidersHorizontal,
} from 'lucide-react';

const ENTERPRISE_CORE =
  (typeof process !== 'undefined' && process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL) ||
  'http://localhost:8096';

interface IncidentEvent {
  id: number;
  from_status: string;
  to_status: string;
  actor: string;
  reason: string;
  created_at: string;
}

interface IncidentAction {
  id: string;
  action_type: string;
  description: string;
  status: string;
  executed_by: string;
  created_at: string;
}

interface Incident {
  id: string;
  title: string;
  severity: string;
  status: string;
  service_id: string;
  service_name: string;
  detection_source: string;
  root_cause_summary: string;
  detected_at: string;
  resolved_at?: string;
  evidence_id?: string;
  events?: IncidentEvent[];
  actions?: IncidentAction[];
}

export const IncidentManagementView: React.FC = () => {
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedIncident, setSelectedIncident] = useState<Incident | null>(null);
  const [severityFilter, setSeverityFilter] = useState<string>('ALL');
  const [statusFilter, setStatusFilter] = useState<string>('ALL');
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  const fetchIncidents = useCallback(async () => {
    setRefreshing(true);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      const res = await fetch(`${ENTERPRISE_CORE}/api/incidents`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setIncidents(data.incidents ?? []);
        if (data.incidents && data.incidents.length > 0 && !selectedIncident) {
          setSelectedIncident(data.incidents[0]);
        }
      }
    } catch (e) {
      console.error('Failed to fetch incidents:', e);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [selectedIncident]);

  useEffect(() => {
    fetchIncidents();
  }, [fetchIncidents]);

  const handleTransition = async (action: 'acknowledge' | 'mitigate' | 'resolve' | 'close') => {
    if (!selectedIncident) return;
    setActionLoading(true);
    try {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') ?? '' : '';
      const res = await fetch(`${ENTERPRISE_CORE}/api/incidents/${selectedIncident.id}/${action}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ reason: `Operator executed ${action} transition` }),
      });

      if (res.ok) {
        await fetchIncidents();
        const updatedRes = await fetch(`${ENTERPRISE_CORE}/api/incidents/${selectedIncident.id}`, {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (updatedRes.ok) {
          setSelectedIncident(await updatedRes.json());
        }
      }
    } catch (e) {
      console.error(`Failed to transition incident: ${e}`);
    } finally {
      setActionLoading(false);
    }
  };

  const filtered = incidents.filter((inc) => {
    if (severityFilter !== 'ALL' && inc.severity !== severityFilter) return false;
    if (statusFilter !== 'ALL' && inc.status !== statusFilter) return false;
    return true;
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <span className="px-2.5 py-0.5 text-xs font-bold bg-rose-100 text-rose-800 rounded-full">
              Autonomous Triage
            </span>
            <h2 className="text-xl font-bold text-gray-900 flex items-center gap-2">
              <AlertOctagon className="w-6 h-6 text-rose-600" />
              Incident Management & Containment
            </h2>
          </div>
          <p className="text-sm text-gray-500 mt-1">
            Formal 8-stage state machine (DETECTED → CLOSED), automated root-cause detection, and auditable evidence logging.
          </p>
        </div>
        <button
          onClick={fetchIncidents}
          disabled={refreshing}
          className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors shadow-sm"
        >
          <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {/* Filters Bar */}
      <div className="flex flex-wrap items-center gap-3 bg-white p-3 rounded-xl border border-gray-200 shadow-sm text-xs">
        <div className="flex items-center gap-1.5 text-gray-500 font-semibold uppercase">
          <Filter className="w-3.5 h-3.5" /> Filter Severity:
        </div>
        {['ALL', 'CRITICAL', 'HIGH', 'WARNING', 'INFO'].map((sev) => (
          <button
            key={sev}
            onClick={() => setSeverityFilter(sev)}
            className={`px-2.5 py-1 rounded-md transition-colors ${
              severityFilter === sev
                ? 'bg-rose-50 text-rose-700 font-bold border border-rose-200'
                : 'text-gray-600 hover:bg-gray-100'
            }`}
          >
            {sev}
          </button>
        ))}

        <div className="h-4 w-px bg-gray-200 mx-2" />

        <div className="flex items-center gap-1.5 text-gray-500 font-semibold uppercase">
          Status:
        </div>
        {['ALL', 'DETECTED', 'ACKNOWLEDGED', 'MITIGATING', 'RESOLVED', 'CLOSED'].map((st) => (
          <button
            key={st}
            onClick={() => setStatusFilter(st)}
            className={`px-2.5 py-1 rounded-md transition-colors ${
              statusFilter === st
                ? 'bg-indigo-50 text-indigo-700 font-bold border border-indigo-200'
                : 'text-gray-600 hover:bg-gray-100'
            }`}
          >
            {st}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center justify-center h-64 text-gray-400">
          <RefreshCw className="w-6 h-6 animate-spin mr-2" />
          Loading institutional incident records…
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Incident List */}
          <div className="lg:col-span-1 bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden flex flex-col h-[600px]">
            <div className="px-4 py-3 bg-gray-50 border-b border-gray-100 font-bold text-xs text-gray-700 uppercase tracking-wider">
              Active Incidents ({filtered.length})
            </div>
            <div className="overflow-y-auto divide-y divide-gray-100 flex-1">
              {filtered.length === 0 ? (
                <div className="p-8 text-center text-xs text-gray-400">No matching incidents found.</div>
              ) : (
                filtered.map((inc) => {
                  const isSelected = selectedIncident?.id === inc.id;
                  const sevColor =
                    inc.severity === 'CRITICAL'
                      ? 'bg-rose-50 text-rose-700 border-rose-200'
                      : inc.severity === 'HIGH'
                      ? 'bg-amber-50 text-amber-700 border-amber-200'
                      : 'bg-blue-50 text-blue-700 border-blue-200';

                  return (
                    <div
                      key={inc.id}
                      onClick={() => setSelectedIncident(inc)}
                      className={`p-4 cursor-pointer transition-colors ${
                        isSelected ? 'bg-indigo-50/70 border-l-4 border-indigo-600' : 'hover:bg-gray-50'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${sevColor}`}>
                          {inc.severity}
                        </span>
                        <span className="text-[11px] font-mono text-gray-400">{inc.status}</span>
                      </div>
                      <div className="font-semibold text-sm text-gray-900 mt-1 line-clamp-1">{inc.title}</div>
                      <div className="text-xs text-gray-500 mt-0.5">{inc.service_name || inc.service_id}</div>
                      <div className="text-[10px] text-gray-400 mt-2 flex items-center gap-1">
                        <Clock className="w-3 h-3" />
                        {new Date(inc.detected_at).toLocaleTimeString()}
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>

          {/* Incident Detail & State Machine Controls */}
          <div className="lg:col-span-2 bg-white rounded-xl border border-gray-200 shadow-sm p-6 flex flex-col justify-between">
            {selectedIncident ? (
              <div className="space-y-6">
                {/* Header & Status */}
                <div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-mono text-indigo-600 font-bold bg-indigo-50 px-2 py-0.5 rounded">
                      {selectedIncident.id}
                    </span>
                    <span className="px-2.5 py-0.5 rounded-full text-xs font-bold bg-gray-100 text-gray-800 border border-gray-200">
                      Status: {selectedIncident.status}
                    </span>
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 mt-2">{selectedIncident.title}</h3>
                  <div className="text-xs text-gray-500 mt-1">
                    Affected Service: <strong className="text-gray-800">{selectedIncident.service_id}</strong> • Source: {selectedIncident.detection_source}
                  </div>
                </div>

                {/* Root Cause Summary */}
                <div className="bg-gray-50 rounded-xl p-4 border border-gray-100">
                  <div className="text-xs font-bold text-gray-700 uppercase tracking-wide">Diagnosis & Root Cause</div>
                  <p className="text-xs text-gray-600 mt-1 leading-relaxed">{selectedIncident.root_cause_summary || 'Analysis in progress by autonomous monitor.'}</p>
                </div>

                {/* State Machine Transition Actions */}
                <div className="bg-indigo-50/50 rounded-xl p-4 border border-indigo-100">
                  <div className="text-xs font-bold text-indigo-900 uppercase tracking-wide mb-3 flex items-center gap-1.5">
                    <SlidersHorizontal className="w-3.5 h-3.5 text-indigo-600" />
                    Lifecycle State Transitions
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <button
                      disabled={actionLoading || selectedIncident.status !== 'DETECTED'}
                      onClick={() => handleTransition('acknowledge')}
                      className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm"
                    >
                      Acknowledge
                    </button>
                    <button
                      disabled={actionLoading || (selectedIncident.status !== 'ACKNOWLEDGED' && selectedIncident.status !== 'TRIAGED')}
                      onClick={() => handleTransition('mitigate')}
                      className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-amber-600 text-white hover:bg-amber-700 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm"
                    >
                      Initiate Mitigation
                    </button>
                    <button
                      disabled={actionLoading || (selectedIncident.status !== 'MITIGATING' && selectedIncident.status !== 'RECOVERING')}
                      onClick={() => handleTransition('resolve')}
                      className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-emerald-600 text-white hover:bg-emerald-700 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm"
                    >
                      Mark Resolved
                    </button>
                    <button
                      disabled={actionLoading || selectedIncident.status !== 'RESOLVED'}
                      onClick={() => handleTransition('close')}
                      className="px-3 py-1.5 text-xs font-semibold rounded-lg bg-gray-800 text-white hover:bg-gray-900 disabled:opacity-40 disabled:cursor-not-allowed shadow-sm"
                    >
                      Close & Certify
                    </button>
                  </div>
                </div>

                {/* Incident Event Timeline */}
                <div>
                  <div className="text-xs font-bold text-gray-700 uppercase tracking-wide mb-3">Audit Lifecycle Timeline</div>
                  <div className="space-y-2">
                    {selectedIncident.events && selectedIncident.events.length > 0 ? (
                      selectedIncident.events.map((ev, i) => (
                        <div key={i} className="flex items-start gap-3 text-xs bg-gray-50/70 p-2.5 rounded-lg border border-gray-100">
                          <div className="w-2 h-2 rounded-full bg-indigo-500 mt-1 flex-shrink-0" />
                          <div className="flex-1">
                            <div className="flex items-center justify-between">
                              <span className="font-bold text-gray-800">
                                {ev.from_status ? `${ev.from_status} → ${ev.to_status}` : `State: ${ev.to_status}`}
                              </span>
                              <span className="text-[10px] text-gray-400 font-mono">
                                {new Date(ev.created_at).toLocaleTimeString()}
                              </span>
                            </div>
                            <div className="text-gray-600 text-[11px] mt-0.5">{ev.reason}</div>
                            <div className="text-gray-400 text-[10px]">Actor: {ev.actor}</div>
                          </div>
                        </div>
                      ))
                    ) : (
                      <div className="text-xs text-gray-400">No transition events recorded.</div>
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-64 text-xs text-gray-400">Select an incident to view details.</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};
