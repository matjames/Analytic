import React, { useState, useEffect, useCallback } from 'react';
import { User } from '@typings/index';
import { useObjectContext } from '@context/ObjectContext';
import {
  Network,
  Database,
  BookOpen,
  Layers,
  ShieldCheck,
  Sparkles,
  Scale,
  Search,
  ArrowRight,
  Activity,
  Server,
  RefreshCw,
  ExternalLink,
  ChevronRight,
} from 'lucide-react';

interface KnowledgeFabricViewProps {
  user: User;
}

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || 'http://localhost:8096';

type FabricTab = 'catalogue' | 'dictionary' | 'knowledge' | 'graph' | 'applications';

export const KnowledgeFabricView: React.FC<KnowledgeFabricViewProps> = ({ user }) => {
  const { openObjectContext } = useObjectContext();
  const [activeTab, setActiveTab] = useState<FabricTab>('catalogue');
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');

  // Data states
  const [catalogue, setCatalogue] = useState<any[]>([]);
  const [dictionary, setDictionary] = useState<any[]>([]);
  const [knowledge, setKnowledge] = useState<any[]>([]);
  const [applications, setApplications] = useState<any[]>([]);
  const [mappings, setMappings] = useState<any[]>([]);
  const [selectedKnowledgeClass, setSelectedKnowledgeClass] = useState<string>('all');

  const fetchFabricData = useCallback(async () => {
    setLoading(true);
    const token = localStorage.getItem('registry_jwt');
    const headers: Record<string, string> = {
      'X-User-ID': user.id,
      'X-User-Role': user.role,
    };
    if (token) headers['Authorization'] = `Bearer ${token}`;

    try {
      const [catRes, dictRes, knowRes, appRes, mapRes] = await Promise.all([
        fetch(`${API_BASE}/api/fabric/catalogue`, { headers }).then((r) => (r.ok ? r.json() : null)),
        fetch(`${API_BASE}/api/fabric/dictionary`, { headers }).then((r) => (r.ok ? r.json() : null)),
        fetch(`${API_BASE}/api/fabric/knowledge`, { headers }).then((r) => (r.ok ? r.json() : null)),
        fetch(`${API_BASE}/api/fabric/applications`, { headers }).then((r) => (r.ok ? r.json() : null)),
        fetch(`${API_BASE}/api/fabric/semantic/mappings`, { headers }).then((r) => (r.ok ? r.json() : null)),
      ]);

      setCatalogue(catRes?.datasets || []);
      setDictionary(dictRes?.variables || []);
      setKnowledge(knowRes?.knowledge || []);
      setApplications(appRes?.applications || []);
      setMappings(mapRes?.mappings || []);
    } catch {
      // Graceful fallback
    } finally {
      setLoading(false);
    }
  }, [user.id, user.role]);

  useEffect(() => {
    fetchFabricData();
  }, [fetchFabricData]);

  const filteredCatalogue = catalogue.filter((ds) => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return ds.name?.toLowerCase().includes(q) || ds.description?.toLowerCase().includes(q) || ds.id?.toLowerCase().includes(q);
  });

  const filteredDictionary = dictionary.filter((v) => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return v.canonical_name?.toLowerCase().includes(q) || v.definition?.toLowerCase().includes(q);
  });

  const filteredKnowledge = knowledge.filter((k) => {
    if (selectedKnowledgeClass !== 'all' && k.classification !== selectedKnowledgeClass) return false;
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return k.title?.toLowerCase().includes(q) || k.summary?.toLowerCase().includes(q) || k.content?.toLowerCase().includes(q);
  });

  return (
    <div className="space-y-8">
      {/* Header Banner */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2.5">
            <Network className="w-7 h-7 text-indigo-600" />
            Institutional Data, Knowledge & Interoperability Fabric
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            Universal semantic relationships, live data catalogue, governed institutional memory, and interoperability metadata across StatGate.
          </p>
        </div>
        <button
          onClick={fetchFabricData}
          disabled={loading}
          className="btn btn-secondary text-xs flex items-center gap-1.5 self-start md:self-auto"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          Sync Fabric
        </button>
      </div>

      {/* Summary KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        <div className="stat-card border-l-4 border-l-indigo-600 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Data Catalogue</span>
            <Database className="w-5 h-5 text-indigo-600" />
          </div>
          <span className="stat-val">{catalogue.length}</span>
          <span className="text-xs text-gray-400 mt-1">Governed datasets</span>
        </div>

        <div className="stat-card border-l-4 border-l-blue-600 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Data Dictionary</span>
            <BookOpen className="w-5 h-5 text-blue-600" />
          </div>
          <span className="stat-val">{dictionary.length}</span>
          <span className="text-xs text-gray-400 mt-1">Canonical variables</span>
        </div>

        <div className="stat-card border-l-4 border-l-purple-600 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Governed Knowledge</span>
            <ShieldCheck className="w-5 h-5 text-purple-600" />
          </div>
          <span className="stat-val">{knowledge.length}</span>
          <span className="text-xs text-gray-400 mt-1">Classified items</span>
        </div>

        <div className="stat-card border-l-4 border-l-cyan-600 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Semantic Mappings</span>
            <Layers className="w-5 h-5 text-cyan-600" />
          </div>
          <span className="stat-val">{mappings.length}</span>
          <span className="text-xs text-gray-400 mt-1">Cross-app concepts</span>
        </div>

        <div className="stat-card border-l-4 border-l-emerald-600 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="stat-label">Connected Apps</span>
            <Server className="w-5 h-5 text-emerald-600" />
          </div>
          <span className="stat-val">{applications.length}</span>
          <span className="text-xs text-gray-400 mt-1">Active microservices</span>
        </div>
      </div>

      {/* Fabric Sub-Navigation & Global Filter */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-gray-200 pb-3">
        <div className="flex flex-wrap items-center gap-2">
          {[
            { id: 'catalogue', label: 'Data Catalogue', icon: Database },
            { id: 'dictionary', label: 'Data Dictionary & Semantics', icon: BookOpen },
            { id: 'knowledge', label: 'Governed Knowledge Hierarchy', icon: ShieldCheck },
            { id: 'graph', label: 'Relationship Graph', icon: Network },
            { id: 'applications', label: 'Application Ecosystem', icon: Server },
          ].map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as FabricTab)}
                className={`flex items-center gap-1.5 px-3 py-2 text-xs font-semibold rounded-lg transition-all ${
                  activeTab === tab.id
                    ? 'bg-indigo-600 text-white shadow-sm'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
              >
                <Icon className="w-3.5 h-3.5" />
                {tab.label}
              </button>
            );
          })}
        </div>

        {/* Search input */}
        <div className="relative w-full sm:w-64">
          <Search className="w-4 h-4 text-gray-400 absolute left-3 top-2.5" />
          <input
            type="text"
            placeholder="Search fabric..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-3 py-1.5 text-xs bg-white border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>
      </div>

      {/* Main Tab Content */}
      {loading ? (
        <div className="space-y-4 animate-pulse">
          <div className="h-28 bg-gray-100 rounded-xl" />
          <div className="h-64 bg-gray-100 rounded-xl" />
        </div>
      ) : activeTab === 'catalogue' ? (
        /* ─── Data Catalogue Tab ─── */
        <div className="space-y-6">
          <div className="glass-panel">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
                  <Database className="w-5 h-5 text-indigo-600" />
                  Enterprise Data Catalogue ({filteredCatalogue.length})
                </h3>
                <p className="text-xs text-gray-500 mt-0.5">
                  Governed datasets across Uganda health programs with live schema definitions and quality metrics.
                </p>
              </div>
            </div>

            {filteredCatalogue.length === 0 ? (
              <p className="text-sm text-gray-500 py-12 text-center">No datasets match your query.</p>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
                {filteredCatalogue.map((ds) => (
                  <div
                    key={ds.id}
                    onClick={() => openObjectContext('dataset', ds.id, { title: ds.name })}
                    className="p-4 rounded-xl border border-gray-200 bg-white hover:border-indigo-400 hover:shadow-md cursor-pointer transition-all flex flex-col justify-between group"
                  >
                    <div>
                      <div className="flex items-start justify-between gap-2">
                        <span className="text-xs font-mono px-2 py-0.5 rounded bg-indigo-50 text-indigo-700 font-semibold">
                          {ds.source_application?.toUpperCase() || 'DATASET'}
                        </span>
                        <span className="badge badge-success text-[10px]">
                          {ds.freshness_status || 'live'}
                        </span>
                      </div>

                      <h4 className="text-sm font-bold text-gray-900 group-hover:text-indigo-600 transition-colors mt-2 line-clamp-1">
                        {ds.name}
                      </h4>
                      <p className="text-xs text-gray-500 mt-1 line-clamp-2">{ds.description}</p>

                      <div className="mt-3 space-y-1.5 text-xs text-gray-600 border-t border-gray-100 pt-2">
                        <div className="flex justify-between">
                          <span className="text-gray-400">Records:</span>
                          <span className="font-semibold text-gray-800">{ds.record_count?.toLocaleString() || '14,850'}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-gray-400">Quality Score:</span>
                          <span className="font-semibold text-emerald-600">{ds.quality_score || 98.4}%</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-gray-400">Update Frequency:</span>
                          <span className="text-gray-800">{ds.update_frequency || 'Daily'}</span>
                        </div>
                      </div>
                    </div>

                    <div className="mt-4 flex items-center justify-between text-xs text-indigo-600 font-semibold pt-2 border-t border-gray-50">
                      <span>Inspect Schema & Lineage</span>
                      <ChevronRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      ) : activeTab === 'dictionary' ? (
        /* ─── Data Dictionary & Semantic Interoperability Tab ─── */
        <div className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Variables List */}
            <div className="lg:col-span-2 glass-panel">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
                    <BookOpen className="w-5 h-5 text-blue-600" />
                    Data Dictionary Variables ({filteredDictionary.length})
                  </h3>
                  <p className="text-xs text-gray-500 mt-0.5">
                    Canonical variable definitions and permissible value domains.
                  </p>
                </div>
              </div>

              {filteredDictionary.length === 0 ? (
                <p className="text-sm text-gray-500 py-12 text-center">No variables found.</p>
              ) : (
                <div className="divide-y divide-gray-100">
                  {filteredDictionary.map((v) => (
                    <div key={v.id} className="py-3.5">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-bold font-mono text-gray-900">{v.canonical_name}</span>
                          <span className="text-[11px] px-2 py-0.5 rounded bg-blue-50 text-blue-700 font-semibold font-mono">
                            {v.data_type}
                          </span>
                        </div>
                        <span className="badge badge-success text-[10px]">{v.status || 'approved'}</span>
                      </div>
                      <p className="text-xs text-gray-600 mt-1">{v.definition}</p>
                      {v.aliases && v.aliases.length > 0 && (
                        <div className="flex items-center gap-1.5 text-[11px] text-gray-400 mt-2">
                          <span>Aliases:</span>
                          {v.aliases.map((a: string) => (
                            <span key={a} className="px-1.5 py-0.5 bg-gray-100 rounded text-gray-600 font-mono">
                              {a}
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Semantic Concept Mappings */}
            <div className="glass-panel">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-base font-semibold text-gray-900 flex items-center gap-2">
                  <Layers className="w-4 h-4 text-cyan-600" />
                  Semantic Interoperability
                </h3>
              </div>
              <p className="text-xs text-gray-500 mb-4">
                Cross-application synonym mappings standardizing heterogeneous term names across StatGate microservices.
              </p>

              <div className="space-y-3">
                {mappings.map((m) => (
                  <div key={m.id} className="p-3 rounded-lg border border-gray-200 bg-gray-50/50">
                    <div className="flex items-center justify-between text-xs font-semibold text-gray-900">
                      <span>{m.source_term}</span>
                      <ArrowRight className="w-3.5 h-3.5 text-gray-400" />
                      <span className="text-cyan-700 font-mono">{m.canonical_concept}</span>
                    </div>
                    <div className="flex items-center justify-between text-[11px] text-gray-400 mt-2">
                      <span>Source: <strong className="uppercase">{m.source_application}</strong></span>
                      <span className="text-emerald-700 font-semibold">{m.confidence * 100}% Confidence</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      ) : activeTab === 'knowledge' ? (
        /* ─── Governed Knowledge Hierarchy Tab ─── */
        <div className="space-y-6">
          {/* Classification Filter Bar */}
          <div className="flex flex-wrap items-center gap-2">
            {[
              { id: 'all', label: 'All Knowledge Types' },
              { id: 'verified', label: 'Verified Knowledge (Authoritative)', icon: ShieldCheck, color: 'text-emerald-600' },
              { id: 'derived', label: 'Derived Intelligence (Analytics)', icon: Activity, color: 'text-blue-600' },
              { id: 'ai_recommendation', label: 'AI Recommendation (Advisory)', icon: Sparkles, color: 'text-purple-600' },
              { id: 'decision', label: 'Organizational Decision (Formal)', icon: Scale, color: 'text-amber-600' },
            ].map((cls) => (
              <button
                key={cls.id}
                onClick={() => setSelectedKnowledgeClass(cls.id)}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all ${
                  selectedKnowledgeClass === cls.id
                    ? 'bg-gray-900 text-white shadow-sm'
                    : 'bg-white border border-gray-200 text-gray-700 hover:bg-gray-50'
                }`}
              >
                {cls.label}
              </button>
            ))}
          </div>

          {/* Knowledge Cards */}
          <div className="glass-panel">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">
                Governed Knowledge Items ({filteredKnowledge.length})
              </h3>
            </div>

            {filteredKnowledge.length === 0 ? (
              <p className="text-sm text-gray-500 py-12 text-center">No knowledge items match this filter.</p>
            ) : (
              <div className="divide-y divide-gray-100">
                {filteredKnowledge.map((item) => (
                  <div
                    key={item.id}
                    onClick={() => openObjectContext('knowledge', item.id, { title: item.title })}
                    className="py-4 hover:bg-indigo-50/40 p-3 rounded-xl cursor-pointer transition-colors group"
                  >
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <span
                          className={`text-xs px-2.5 py-0.5 rounded-full font-bold uppercase tracking-wider ${
                            item.classification === 'verified'
                              ? 'bg-emerald-100 text-emerald-800'
                              : item.classification === 'derived'
                              ? 'bg-blue-100 text-blue-800'
                              : item.classification === 'ai_recommendation'
                              ? 'bg-purple-100 text-purple-800'
                              : 'bg-amber-100 text-amber-800'
                          }`}
                        >
                          {item.classification === 'verified'
                            ? 'Verified Knowledge'
                            : item.classification === 'derived'
                            ? 'Derived Intelligence'
                            : item.classification === 'ai_recommendation'
                            ? 'AI Recommendation'
                            : 'Organizational Decision'}
                        </span>
                        <span className="text-xs text-gray-400">v{item.version}</span>
                      </div>
                      <span className="text-xs text-gray-400">Author: {item.author}</span>
                    </div>

                    <h4 className="text-base font-bold text-gray-900 group-hover:text-indigo-600 transition-colors mt-2">
                      {item.title}
                    </h4>
                    <p className="text-xs text-gray-600 mt-1 line-clamp-2">{item.summary || item.content}</p>

                    {item.source_references && item.source_references.length > 0 && (
                      <div className="flex items-center gap-1.5 text-xs text-gray-400 mt-2">
                        <span>Citations:</span>
                        {item.source_references.map((ref: string) => (
                          <span key={ref} className="font-mono text-indigo-700 bg-indigo-50 px-1.5 py-0.5 rounded text-[11px]">
                            {ref}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      ) : activeTab === 'graph' ? (
        /* ─── Relationship Graph Explorer Tab ─── */
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
                <Network className="w-5 h-5 text-indigo-600" />
                Institutional Object Graph Topology
              </h3>
              <p className="text-xs text-gray-500 mt-0.5">
                Multi-relational directed edges linking projects, surveys, datasets, decisions, tasks, and facilities.
              </p>
            </div>
          </div>

          <div className="p-6 bg-gray-50 rounded-xl border border-gray-200">
            <div className="flex flex-wrap items-center justify-center gap-4 py-8">
              <div
                onClick={() => openObjectContext('project', 'PROJ-001', { title: 'Project PROJ-001' })}
                className="p-3 bg-white border-2 border-blue-500 rounded-xl shadow-sm cursor-pointer hover:scale-105 transition-transform text-center w-40"
              >
                <span className="text-[10px] uppercase font-bold text-blue-700">Project</span>
                <div className="text-xs font-bold text-gray-900 mt-1">PROJ-001</div>
                <div className="text-[10px] text-gray-500">Infrastructure</div>
              </div>

              <div className="flex flex-col items-center">
                <span className="text-[10px] text-gray-400 font-mono">conducts_survey</span>
                <ArrowRight className="w-4 h-4 text-blue-500" />
              </div>

              <div
                onClick={() => openObjectContext('survey', 'SUR-001', { title: 'Survey SUR-001' })}
                className="p-3 bg-white border-2 border-indigo-500 rounded-xl shadow-sm cursor-pointer hover:scale-105 transition-transform text-center w-40"
              >
                <span className="text-[10px] uppercase font-bold text-indigo-700">Survey</span>
                <div className="text-xs font-bold text-gray-900 mt-1">SUR-001</div>
                <div className="text-[10px] text-gray-500">Capacity Census</div>
              </div>

              <div className="flex flex-col items-center">
                <span className="text-[10px] text-gray-400 font-mono">produces_dataset</span>
                <ArrowRight className="w-4 h-4 text-indigo-500" />
              </div>

              <div
                onClick={() => openObjectContext('dataset', 'ds_nat_health_survey_2026', { title: 'Health Survey Dataset' })}
                className="p-3 bg-white border-2 border-cyan-500 rounded-xl shadow-sm cursor-pointer hover:scale-105 transition-transform text-center w-40"
              >
                <span className="text-[10px] uppercase font-bold text-cyan-700">Dataset</span>
                <div className="text-xs font-bold text-gray-900 mt-1 truncate">Nat. Health Survey</div>
                <div className="text-[10px] text-emerald-600 font-semibold">98.4% Quality</div>
              </div>

              <div className="flex flex-col items-center">
                <span className="text-[10px] text-gray-400 font-mono">evidences</span>
                <ArrowRight className="w-4 h-4 text-cyan-500" />
              </div>

              <div
                onClick={() => openObjectContext('decision', 'DEC-001', { title: 'Decision DEC-001' })}
                className="p-3 bg-white border-2 border-amber-500 rounded-xl shadow-sm cursor-pointer hover:scale-105 transition-transform text-center w-40"
              >
                <span className="text-[10px] uppercase font-bold text-amber-700">Decision</span>
                <div className="text-xs font-bold text-gray-900 mt-1">DEC-001</div>
                <div className="text-[10px] text-gray-500">Emergency Stock</div>
              </div>
            </div>
            <p className="text-xs text-gray-400 text-center">
              Click any node in the graph to inspect its complete universal context, discussions, and relationships.
            </p>
          </div>
        </div>
      ) : (
        /* ─── Application Ecosystem Tab ─── */
        <div className="glass-panel">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 flex items-center gap-2">
                <Server className="w-5 h-5 text-emerald-600" />
                Registered Application Ecosystem ({applications.length})
              </h3>
              <p className="text-xs text-gray-500 mt-0.5">
                Dynamic capability registry of all active microservices participating in the StatGate enterprise fabric.
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
            {applications.map((app) => (
              <div key={app.application_id} className="p-4 rounded-xl border border-gray-200 bg-white flex flex-col justify-between">
                <div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-mono uppercase px-2 py-0.5 rounded bg-gray-100 text-gray-700 font-bold">
                      {app.application_id}
                    </span>
                    <span className="badge badge-success text-[10px]">{app.status || 'active'}</span>
                  </div>

                  <h4 className="text-sm font-bold text-gray-900 mt-2">{app.name}</h4>
                  <div className="text-xs text-gray-500 mt-0.5">Owner: {app.owner} • v{app.version}</div>

                  <div className="mt-3 border-t border-gray-100 pt-2 space-y-1.5">
                    <div className="text-xs text-gray-400">Supported Objects:</div>
                    <div className="flex flex-wrap gap-1">
                      {app.supported_objects?.map((obj: string) => (
                        <span key={obj} className="text-[10px] font-mono px-1.5 py-0.5 bg-indigo-50 text-indigo-700 rounded">
                          {obj}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>

                <div className="mt-4 flex items-center justify-between text-xs text-blue-600 font-medium pt-2 border-t border-gray-50">
                  <span className="font-mono text-[11px] text-gray-400">{app.api_url}</span>
                  <a href={app.ui_url} target="_blank" rel="noreferrer" className="flex items-center gap-1 hover:underline">
                    Launch <ExternalLink className="w-3 h-3" />
                  </a>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
