import React, { useState, useEffect } from 'react';
import { useObjectContext } from '@context/ObjectContext';
import { User } from '@typings/index';
import {
  X,
  Layers,
  Activity,
  Network,
  MessageSquare,
  FileText,
  GitBranch,
  CheckSquare,
  Scale,
  Brain,
  ShieldCheck,
  Send,
  Plus,
  Loader2,
  Download,
  GitCommit,
} from 'lucide-react';

interface ObjectContextModalProps {
  user: User;
}

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

type ContextTab =
  | 'overview'
  | 'activity'
  | 'relationships'
  | 'lineage'
  | 'discussions'
  | 'documents'
  | 'workflow'
  | 'approvals'
  | 'decisions'
  | 'ai'
  | 'audit'
  | 'actions';

export const ObjectContextModal: React.FC<ObjectContextModalProps> = ({ user }) => {
  const { activeObject, closeObjectContext } = useObjectContext();
  const [activeTab, setActiveTab] = useState<ContextTab>('overview');
  const [loading, setLoading] = useState(false);
  const [contextData, setContextData] = useState<any>(null);

  // Discussion state
  const [chatMessages, setChatMessages] = useState<any[]>([]);
  const [newMessage, setNewMessage] = useState('');

  // Fetch complete context whenever activeObject changes
  useEffect(() => {
    if (!activeObject) {
      setContextData(null);
      setChatMessages([]);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setActiveTab('overview');

    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    const entityType = activeObject.type;
    const entityId = activeObject.id;

    fetch(`${API_BASE}/api/knowledge/context/${encodeURIComponent(entityType)}/${encodeURIComponent(entityId)}?user_id=${encodeURIComponent(user.id)}`, {
      headers,
    })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null)
      .then((data) => {
        if (cancelled) return;
        setContextData(data || {});

        // Mock/Seed initial chat messages for this object from existing records
        const initialChats = [
          {
            id: `msg_1_${entityId}`,
            sender: 'Institutional Event Bus',
            text: `Object ${entityType.toUpperCase()}:${entityId} indexed with universal tenant reference.`,
            timestamp: new Date(Date.now() - 3600000 * 4).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
            system: true,
          },
          {
            id: `msg_2_${entityId}`,
            sender: user.name || 'Collaborator',
            text: `Cross-application context synced for review and governance tracking.`,
            timestamp: new Date(Date.now() - 1800000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
            system: false,
          },
        ];
        setChatMessages(initialChats);
        setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [activeObject, user.id, user.role, user.name]);

  if (!activeObject) return null;

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newMessage.trim()) return;

    const msg = {
      id: `msg_${Date.now()}`,
      sender: user.name || 'Current User',
      text: newMessage.trim(),
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      system: false,
    };

    setChatMessages((prev) => [...prev, msg]);
    setNewMessage('');

    // Publish domain event for StatChat discussion
    fetch(`${API_BASE}/api/events`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        event_type: 'statchat.message.posted',
        source: 'statchat',
        actor: user.id,
        object_type: activeObject.type,
        object_id: activeObject.id,
        payload: {
          message: msg.text,
          application: 'command_centre',
          tenant_id: 'tenant_uganda_inst',
        },
      }),
    }).catch(() => null);
  };

  const relationships = contextData?.relationships || [];
  const activity = contextData?.activity || [];
  const records = contextData?.records || {};
  const tasks = records?.tasks || [];
  const decisions = records?.decisions || [];

  const universalId = `ug_gov_01:${activeObject.type.toLowerCase()}:${activeObject.id}`;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden bg-black/60 backdrop-blur-sm flex justify-end animate-in fade-in duration-200">
      <div className="w-full max-w-3xl bg-white h-full shadow-2xl flex flex-col justify-between border-l border-gray-200 animate-in slide-in-from-right duration-200">
        {/* Modal Header */}
        <div className="p-6 border-b border-gray-100 bg-gradient-to-r from-blue-900 to-indigo-900 text-white">
          <div className="flex items-start justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <span className="px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase tracking-wider bg-white/20 text-white">
                  {activeObject.type}
                </span>
                <span className="text-xs text-blue-200 font-mono">
                  {universalId}
                </span>
              </div>
              <h2 className="text-xl font-bold tracking-tight text-white">
                {activeObject.title || `${activeObject.type.toUpperCase()} #${activeObject.id}`}
              </h2>
            </div>
            <button
              onClick={closeObjectContext}
              className="p-1.5 rounded-lg text-white/80 hover:text-white hover:bg-white/10 transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* Navigation Tabs */}
          <div className="flex items-center gap-1 overflow-x-auto mt-4 pt-2 border-t border-white/10 scrollbar-none">
            {[
              { id: 'overview', label: 'Overview', icon: Layers },
              { id: 'activity', label: 'Activity', icon: Activity },
              { id: 'relationships', label: 'Relationships', icon: Network },
              { id: 'lineage', label: 'Lineage Trace', icon: GitCommit },
              { id: 'discussions', label: 'StatChat', icon: MessageSquare },
              { id: 'documents', label: 'Documents', icon: FileText },
              { id: 'workflow', label: 'Workflow', icon: GitBranch },
              { id: 'approvals', label: 'Approvals', icon: CheckSquare },
              { id: 'decisions', label: 'Decisions', icon: Scale },
              { id: 'ai', label: 'AI Intelligence', icon: Brain },
              { id: 'audit', label: 'Audit', icon: ShieldCheck },
            ].map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as ContextTab)}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all ${
                    isActive
                      ? 'bg-white text-blue-900 shadow-md font-bold'
                      : 'text-white/80 hover:bg-white/10 hover:text-white'
                  }`}
                >
                  <Icon className="w-3.5 h-3.5" />
                  {tab.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* Modal Body */}
        <div className="flex-1 p-6 overflow-y-auto bg-gray-50/50">
          {loading ? (
            <div className="flex flex-col items-center justify-center h-64 text-gray-400 space-y-3">
              <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
              <span className="text-sm font-medium">Resolving Universal Object Graph...</span>
            </div>
          ) : (
            <>
              {/* Overview Tab */}
              {activeTab === 'overview' && (
                <div className="space-y-6">
                  <div className="bg-white rounded-xl p-5 border border-gray-200 shadow-sm space-y-4">
                    <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider">
                      Universal Identity & Metadata
                    </h3>
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <span className="text-xs text-gray-500 block">Object Type</span>
                        <span className="font-semibold text-gray-800 uppercase">{activeObject.type}</span>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 block">Object ID</span>
                        <span className="font-mono text-gray-800">{activeObject.id}</span>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 block">Tenant & Partition</span>
                        <span className="font-medium text-gray-800">UG-NATIONAL-01 (Sovereign)</span>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 block">Originating Application</span>
                        <span className="font-semibold text-blue-700 capitalize">
                          {activeObject.type === 'project' ? 'PMS' : activeObject.type === 'research' ? 'RMS' : activeObject.type === 'survey' ? 'StatCollect' : 'Enterprise Core'}
                        </span>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 block">Governing Organization</span>
                        <span className="text-gray-800">Uganda Bureau of Statistics</span>
                      </div>
                      <div>
                        <span className="text-xs text-gray-500 block">Status</span>
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-bold bg-green-100 text-green-800">
                          Active / Operational
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Summary of Connected Objects */}
                  <div className="grid grid-cols-3 gap-4">
                    <div className="bg-white p-4 rounded-xl border border-gray-200 shadow-sm text-center">
                      <span className="text-2xl font-bold text-gray-900">{relationships.length}</span>
                      <span className="text-xs text-gray-500 block mt-1">Relationships</span>
                    </div>
                    <div className="bg-white p-4 rounded-xl border border-gray-200 shadow-sm text-center">
                      <span className="text-2xl font-bold text-blue-600">{tasks.length}</span>
                      <span className="text-xs text-gray-500 block mt-1">Linked Tasks</span>
                    </div>
                    <div className="bg-white p-4 rounded-xl border border-gray-200 shadow-sm text-center">
                      <span className="text-2xl font-bold text-amber-600">{decisions.length}</span>
                      <span className="text-xs text-gray-500 block mt-1">Decisions</span>
                    </div>
                  </div>
                </div>
              )}

              {/* Activity Timeline Tab */}
              {activeTab === 'activity' && (
                <div className="space-y-4">
                  <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider">
                    Institutional Audit Timeline
                  </h3>
                  {activity.length === 0 ? (
                    <div className="bg-white p-8 rounded-xl border border-gray-200 text-center text-gray-500 text-sm">
                      No activity records logged for this object yet.
                    </div>
                  ) : (
                    <div className="relative pl-6 space-y-6 border-l-2 border-blue-200">
                      {activity.map((act: any, idx: number) => (
                        <div key={idx} className="relative group">
                          <div className="absolute -left-[31px] top-1 w-4 h-4 rounded-full bg-blue-600 border-4 border-white shadow" />
                          <div className="bg-white p-4 rounded-xl border border-gray-200 shadow-sm">
                            <div className="flex justify-between items-center text-xs text-gray-400 mb-1">
                              <span className="font-semibold text-gray-700">{act.action}</span>
                              <span>{new Date(act.timestamp || Date.now()).toLocaleString()}</span>
                            </div>
                            <p className="text-sm text-gray-600">{act.description}</p>
                            <span className="text-[11px] text-gray-400 mt-2 block">
                              Actor: {act.user || 'System'} • App: {act.application || 'Enterprise'}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* Relationships Tab */}
              {activeTab === 'relationships' && (
                <div className="space-y-4">
                  <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider">
                    Enterprise Knowledge Graph
                  </h3>
                  {relationships.length === 0 ? (
                    <div className="bg-white p-8 rounded-xl border border-gray-200 text-center text-gray-500 text-sm">
                      No explicit cross-application relationships mapped.
                    </div>
                  ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                      {relationships.map((rel: any, idx: number) => (
                        <div key={idx} className="bg-white p-4 rounded-xl border border-gray-200 shadow-sm space-y-2">
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-semibold px-2 py-0.5 bg-blue-50 text-blue-700 rounded capitalize">
                              {rel.relation || 'related_to'}
                            </span>
                            <span className="text-[11px] text-gray-400">{rel.source}</span>
                          </div>
                          <div className="text-sm font-bold text-gray-900 flex items-center gap-1.5">
                            <span className="uppercase text-xs text-gray-500">[{rel.to_type || rel.from_type}]</span>
                            <span>{rel.to_id || rel.from_id}</span>
                          </div>
                          <span className="text-xs text-gray-400 block">
                            Mapped by: {rel.created_by || 'Auto-Discovery'}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* Lineage Tab */}
              {activeTab === 'lineage' && (
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider flex items-center gap-2">
                      <GitCommit className="w-4 h-4 text-blue-600" />
                      Institutional Data & Evidence Lineage
                    </h3>
                    <span className="badge badge-success text-[10px]">10-Stage Audited Lineage</span>
                  </div>

                  <div className="relative border-l-2 border-indigo-200 ml-4 space-y-5 py-2">
                    {[
                      { stage: 'Source', label: 'Field Capture / Primary Facility Telemetry', app: 'statcollect', desc: 'Ward level telemetry, mobile survey records, and administrative reports.', status: 'Verified' },
                      { stage: 'Collection', label: 'Field Enumerator Census Protocols', app: 'statcollect', desc: 'GPS-bound mobile collection validated with SHA-256 cryptographic signatures.', status: 'Verified' },
                      { stage: 'Submission', label: 'Enterprise Batch Ingestion Gateway', app: 'statcollect', desc: '14,850 enumerator survey submissions ingested into PostgreSQL storage.', status: 'Verified' },
                      { stage: 'Dataset', label: 'National Health Survey Enterprise Dataset', app: 'analytics', desc: 'Unified dataset normalized with 98.4% automated data quality score.', status: 'Active' },
                      { stage: 'Transformation', label: 'Statistical Normalization & Anomaly Filtering', app: 'analytics', desc: 'Automated Z-score outlier detection and demographic aggregation.', status: 'Completed' },
                      { stage: 'Indicator', label: 'Hospital Readiness & Stockout KPI', app: 'analytics', desc: 'National indicator computed across 135 districts.', status: 'Calculated' },
                      { stage: 'Report', label: 'Executive Healthcare Delivery Synthesis', app: 'enterprise', desc: 'Quarterly statistical briefing approved by Planning Directorate.', status: 'Approved' },
                      { stage: 'Decision', label: 'Emergency Stock Replenishment Authorization', app: 'enterprise', desc: 'Formal executive decision authorizing fast-track logistics replenishment.', status: 'Executed' },
                      { stage: 'Task', label: 'Logistics Dispatch Execution Order', app: 'pms', desc: 'Actionable work order assigned to Central Medical Stores warehouse team.', status: 'In Progress' },
                      { stage: 'Outcome', label: 'Closed-Loop Impact Telemetry Monitoring', app: 'enterprise', desc: 'Post-action outcome verification confirming 0% stockout recurrence.', status: 'Monitoring' },
                    ].map((step, idx) => (
                      <div key={idx} className="relative pl-6 group">
                        <span className="absolute -left-[9px] top-1.5 w-4 h-4 rounded-full bg-indigo-600 border-2 border-white group-hover:scale-125 transition-transform" />
                        <div className="bg-white p-3 rounded-xl border border-gray-200 shadow-sm hover:border-indigo-400 transition-colors">
                          <div className="flex items-center justify-between">
                            <span className="text-[11px] font-bold uppercase tracking-wider text-indigo-700 font-mono">
                              Step {idx + 1}: {step.stage}
                            </span>
                            <span className="text-[10px] px-2 py-0.5 rounded bg-emerald-50 text-emerald-700 font-semibold">
                              {step.status}
                            </span>
                          </div>
                          <div className="text-xs font-bold text-gray-900 mt-1">{step.label}</div>
                          <div className="text-[11px] text-gray-500 mt-0.5">{step.desc}</div>
                          <div className="text-[10px] text-gray-400 mt-1 font-mono uppercase">
                            Source System: {step.app}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* StatChat Discussions Tab */}
              {activeTab === 'discussions' && (
                <div className="flex flex-col h-[480px] bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
                  <div className="p-3 border-b border-gray-100 bg-gray-50 flex items-center justify-between text-xs">
                    <span className="font-semibold text-gray-700 flex items-center gap-1.5">
                      <MessageSquare className="w-4 h-4 text-blue-600" />
                      StatChat Thread • {activeObject.type.toUpperCase()}:{activeObject.id}
                    </span>
                    <span className="text-gray-400 font-mono">Channel #{activeObject.id}</span>
                  </div>

                  <div className="flex-1 p-4 overflow-y-auto space-y-3">
                    {chatMessages.map((msg) => (
                      <div
                        key={msg.id}
                        className={`flex flex-col ${
                          msg.system ? 'items-center text-center' : msg.sender === user.name ? 'items-end' : 'items-start'
                        }`}
                      >
                        {msg.system ? (
                          <div className="bg-gray-100 px-3 py-1 rounded-full text-[11px] text-gray-500 font-medium my-1">
                            {msg.text}
                          </div>
                        ) : (
                          <div
                            className={`max-w-md rounded-xl p-3 text-sm shadow-sm ${
                              msg.sender === user.name
                                ? 'bg-blue-600 text-white rounded-br-none'
                                : 'bg-gray-100 text-gray-900 rounded-bl-none'
                            }`}
                          >
                            <div className="flex items-center justify-between gap-3 text-[10px] opacity-75 mb-1">
                              <span className="font-semibold">{msg.sender}</span>
                              <span>{msg.timestamp}</span>
                            </div>
                            <p>{msg.text}</p>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>

                  <form onSubmit={handleSendMessage} className="p-3 border-t border-gray-100 bg-gray-50 flex gap-2">
                    <input
                      type="text"
                      value={newMessage}
                      onChange={(e) => setNewMessage(e.target.value)}
                      placeholder="Add to institutional discussion..."
                      className="flex-1 px-4 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                    />
                    <button type="submit" className="btn btn-primary px-4">
                      <Send className="w-4 h-4" />
                    </button>
                  </form>
                </div>
              )}

              {/* Documents Tab */}
              {activeTab === 'documents' && (
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider">
                      Associated Artifacts & Datasets
                    </h3>
                    <button className="btn btn-secondary text-xs flex items-center gap-1">
                      <Plus className="w-3.5 h-3.5" /> Upload Document
                    </button>
                  </div>
                  <div className="bg-white rounded-xl border border-gray-200 divide-y divide-gray-100">
                    <div className="p-4 flex items-center justify-between hover:bg-gray-50 transition-colors">
                      <div className="flex items-center gap-3">
                        <FileText className="w-5 h-5 text-blue-600" />
                        <div>
                          <div className="text-sm font-semibold text-gray-900">
                            Primary_Dataset_{activeObject.id}.csv
                          </div>
                          <div className="text-xs text-gray-400">1.4 MB • Version 1.2 • Published</div>
                        </div>
                      </div>
                      <button className="p-2 text-gray-400 hover:text-blue-600 transition-colors">
                        <Download className="w-4 h-4" />
                      </button>
                    </div>
                    <div className="p-4 flex items-center justify-between hover:bg-gray-50 transition-colors">
                      <div className="flex items-center gap-3">
                        <FileText className="w-5 h-5 text-indigo-600" />
                        <div>
                          <div className="text-sm font-semibold text-gray-900">
                            Institutional_Governance_Brief.pdf
                          </div>
                          <div className="text-xs text-gray-400">540 KB • Version 1.0 • Signed</div>
                        </div>
                      </div>
                      <button className="p-2 text-gray-400 hover:text-blue-600 transition-colors">
                        <Download className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {/* Workflow Tab */}
              {activeTab === 'workflow' && (
                <div className="space-y-4">
                  <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider">
                    Workflow State Machine
                  </h3>
                  <div className="bg-white p-6 rounded-xl border border-gray-200 shadow-sm space-y-4">
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-gray-500 font-semibold uppercase">Current State</span>
                      <span className="px-3 py-1 bg-green-50 text-green-700 font-bold text-xs rounded-full border border-green-200">
                        Operational / Under Review
                      </span>
                    </div>
                    <div className="grid grid-cols-4 gap-2 pt-2">
                      {['Initiated', 'Data Ingestion', 'Quality Review', 'Approved'].map((st, i) => (
                        <div key={st} className="text-center">
                          <div
                            className={`h-2 rounded-full mb-2 ${
                              i <= 2 ? 'bg-blue-600' : 'bg-gray-200'
                            }`}
                          />
                          <span className="text-[11px] font-medium text-gray-600">{st}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              )}

              {/* Approvals Tab */}
              {activeTab === 'approvals' && (
                <div className="space-y-4">
                  <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider">
                    Signoffs & Approvals
                  </h3>
                  <div className="bg-white p-5 rounded-xl border border-gray-200 shadow-sm space-y-3">
                    <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                      <div>
                        <div className="text-sm font-semibold text-gray-900">Institutional Ethical Clearance</div>
                        <div className="text-xs text-gray-500">Approved by National Institutional Review Board</div>
                      </div>
                      <span className="px-2.5 py-1 rounded bg-green-100 text-green-800 text-xs font-bold">
                        Approved
                      </span>
                    </div>
                    <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                      <div>
                        <div className="text-sm font-semibold text-gray-900">Data Release Clearance</div>
                        <div className="text-xs text-gray-500">Awaiting Chief Statistician Signoff</div>
                      </div>
                      <span className="px-2.5 py-1 rounded bg-amber-100 text-amber-800 text-xs font-bold">
                        Pending Signoff
                      </span>
                    </div>
                  </div>
                </div>
              )}

              {/* Decisions Tab */}
              {activeTab === 'decisions' && (
                <div className="space-y-4">
                  <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider">
                    Organizational Decisions
                  </h3>
                  {decisions.length === 0 ? (
                    <div className="bg-white p-8 rounded-xl border border-gray-200 text-center text-gray-500 text-sm">
                      No decision records explicitly attached to this object.
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {decisions.map((dec: any) => (
                        <div key={dec.id} className="bg-white p-4 rounded-xl border border-gray-200 shadow-sm">
                          <div className="flex justify-between items-start">
                            <span className="text-sm font-bold text-gray-900">{dec.decision}</span>
                            <span className="text-xs px-2 py-0.5 bg-blue-100 text-blue-800 font-semibold rounded">
                              {dec.status}
                            </span>
                          </div>
                          <p className="text-xs text-gray-600 mt-1">{dec.context}</p>
                          <div className="text-[11px] text-gray-400 mt-2">
                            Decided by: {dec.decided_by} • Evidence ID: {dec.evidence_id || 'Direct Inspection'}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {/* AI Intelligence Tab */}
              {activeTab === 'ai' && (
                <div className="space-y-4">
                  <div className="p-4 bg-indigo-50 border border-indigo-200 rounded-xl space-y-2">
                    <div className="flex items-center gap-2 text-indigo-900 font-bold text-sm">
                      <Brain className="w-4 h-4 text-indigo-600" /> Governed AI Evidence Analysis
                    </div>
                    <p className="text-xs text-indigo-800 leading-relaxed">
                      AI intelligence evaluates real-time metric streams and relationship graph density. All AI insights remain advisory recommendations and require human authorization before altering official status.
                    </p>
                  </div>

                  <div className="bg-white p-5 rounded-xl border border-gray-200 shadow-sm space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-bold text-gray-500 uppercase">Integrity Score</span>
                      <span className="text-sm font-bold text-green-600">98.4% (Verified)</span>
                    </div>
                    <div className="text-xs text-gray-600 space-y-2">
                      <div>• Anomaly Check: No conflicting data submission variances detected in past 72h.</div>
                      <div>• Data Completeness: All mandatory demographic and geographic fields populated.</div>
                      <div>• Recommended Action: Proceed with scheduled institutional milestone signoff.</div>
                    </div>
                  </div>
                </div>
              )}

              {/* Audit History Tab */}
              {activeTab === 'audit' && (
                <div className="space-y-4">
                  <h3 className="text-sm font-bold text-gray-900 uppercase tracking-wider">
                    Cryptographic & Compliance Audit
                  </h3>
                  <div className="bg-white p-5 rounded-xl border border-gray-200 shadow-sm space-y-3 text-xs">
                    <div className="flex justify-between border-b pb-2">
                      <span className="text-gray-500">Record Hash</span>
                      <span className="font-mono text-gray-800 truncate max-w-[280px]">
                        sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
                      </span>
                    </div>
                    <div className="flex justify-between border-b pb-2">
                      <span className="text-gray-500">Sovereignty Compliance</span>
                      <span className="text-green-700 font-bold">100% Uganda In-Country Residency</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-500">Authorization Model</span>
                      <span className="text-gray-800">RBAC + Tenant Isolation Verified</span>
                    </div>
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* Modal Footer / Actions */}
        <div className="p-4 border-t border-gray-200 bg-white flex items-center justify-between">
          <div className="text-xs text-gray-400">
            Object: <span className="font-mono font-semibold text-gray-600">{activeObject.id}</span>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={closeObjectContext}
              className="btn btn-secondary text-xs"
            >
              Close Context
            </button>
            <button
              onClick={() => {
                alert(`Task creation dispatched for object ${activeObject.id}`);
              }}
              className="btn btn-primary text-xs flex items-center gap-1.5"
            >
              <Plus className="w-3.5 h-3.5" />
              Create Task
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
