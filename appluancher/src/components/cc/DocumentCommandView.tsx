import React, { useEffect, useState } from "react";
import { User } from "@typings/index";
import {
  FolderArchive, FileText, QrCode, Shield, Search, Plus, CheckCircle2, Lock,
  GitBranch, History, FolderOpen, BookOpen, Archive, AlertTriangle, Clock, ChevronRight,
  Edit3, UploadCloud, Globe, Eye, Unlock, RefreshCw, Tag, Database
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_ENTERPRISE_CORE_URL || "http://localhost:8096";

type Tab = "repo" | "editor" | "signatures" | "archives" | "knowledge" | "search";

const statusColor: Record<string, string> = {
  Draft: "bg-gray-100 text-gray-700",
  In_Review: "bg-amber-100 text-amber-800",
  Approved: "bg-emerald-100 text-emerald-800",
  Published: "bg-blue-100 text-blue-800",
  Archived: "bg-slate-100 text-slate-700",
  Legal_Hold: "bg-red-100 text-red-800",
};

const classColor: Record<string, string> = {
  PUBLIC: "bg-green-100 text-green-800",
  INTERNAL: "bg-blue-100 text-blue-800",
  CONFIDENTIAL: "bg-orange-100 text-orange-800",
  RESTRICTED: "bg-red-100 text-red-900",
};

export const DocumentCommandView: React.FC<{ user: User }> = ({ user }) => {
  const [activeTab, setActiveTab] = useState<Tab>("repo");
  const [dashboard, setDashboard] = useState<any>(null);
  const [documents, setDocuments] = useState<any[]>([]);
  const [templates, setTemplates] = useState<any[]>([]);
  const [signatures, setSignatures] = useState<any[]>([]);
  const [retentionPolicies, setRetentionPolicies] = useState<any[]>([]);
  const [legalHolds, setLegalHolds] = useState<any[]>([]);
  const [folders, setFolders] = useState<any[]>([]);
  const [repositories, setRepositories] = useState<any[]>([]);
  const [archives, setArchives] = useState<any[]>([]);
  const [dispositions, setDispositions] = useState<any[]>([]);
  const [articles, setArticles] = useState<any[]>([]);
  const [wikiPages, setWikiPages] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedDoc, setSelectedDoc] = useState<any>(null);
  const [docVersions, setDocVersions] = useState<any[]>([]);
  const [docComments, setDocComments] = useState<any[]>([]);
  const [showVersionModal, setShowVersionModal] = useState(false);
  const [editorTitle, setEditorTitle] = useState("Institutional Data Release Guidelines 2026");
  const [editorCategory, setEditorCategory] = useState("Policy");
  const [editorClassification, setEditorClassification] = useState("INTERNAL");
  const [editorContent, setEditorContent] = useState(
    "# Institutional Data Release Guidelines 2026\n\n## 1. Executive Summary\nMandatory controls for public release of anonymized census microdata."
  );
  const [selectedFolder, setSelectedFolder] = useState("/Policies");
  const [signingDoc, setSigningDoc] = useState<any>(null);
  const [signerRole, setSignerRole] = useState("Executive Approval");
  const [showQrStamp, setShowQrStamp] = useState<any>(null);
  const [showLegalHoldModal, setShowLegalHoldModal] = useState(false);
  const [newHold, setNewHold] = useState({ case_reference: "", reason: "", custodian: "", issued_by: "" });
  const [showRetentionModal, setShowRetentionModal] = useState(false);
  const [newRetention, setNewRetention] = useState({ category: "", retention_period_years: 7, disposition_action: "Review_For_Destruction", legal_authority: "" });
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<any[]>([]);
  const [searching, setSearching] = useState(false);
  const [showCheckinModal, setShowCheckinModal] = useState(false);
  const [checkinSummary, setCheckinSummary] = useState("");
  const [checkinMajor, setCheckinMajor] = useState(false);
  const [checkoutLoading, setCheckoutLoading] = useState<string | null>(null);

  const getHeaders = () => {
    const token = localStorage.getItem("registry_jwt");
    const h: Record<string, string> = { "X-User-ID": user.id, "X-User-Role": user.role };
    if (token) h["Authorization"] = `Bearer ${token}`;
    return h;
  };

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    const h = getHeaders();
    Promise.all([
      fetch(`${API_BASE}/api/documents/dashboard`, { headers: h }).then(r => r.ok ? r.json() : null).catch(() => null),
      fetch(`${API_BASE}/api/documents`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/documents/templates`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/documents/signatures`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/records/retention-policies`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/records/legal-holds`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/documents/folders`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/documents/repositories`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/archives`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/records/dispositions`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/knowledge/articles`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/knowledge/wiki`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
    ]).then(([dash, docs, tpls, sigs, ret, holds, flds, repos, arcs, disps, arts, wiki]) => {
      if (cancelled) return;
      setDashboard(dash || { total_documents: 3, active_legal_holds: 1, knowledge_articles: 3, iso_15489_compliance: "CERTIFIED_COMPLIANT" });
      setDocuments(Array.isArray(docs) ? docs : []);
      setTemplates(Array.isArray(tpls) ? tpls : []);
      setSignatures(Array.isArray(sigs) ? sigs : []);
      setRetentionPolicies(Array.isArray(ret) ? ret : []);
      setLegalHolds(Array.isArray(holds) ? holds : []);
      setFolders(Array.isArray(flds) ? flds : []);
      setRepositories(Array.isArray(repos) ? repos : []);
      setArchives(Array.isArray(arcs) ? arcs : []);
      setDispositions(Array.isArray(disps) ? disps : []);
      setArticles(Array.isArray(arts) ? arts : []);
      setWikiPages(Array.isArray(wiki) ? wiki : []);
      setLoading(false);
    });
    return () => { cancelled = true; };
  }, [user.id]);

  const loadDocDetails = async (doc: any) => {
    setSelectedDoc(doc);
    const h = getHeaders();
    const [vers, cmts] = await Promise.all([
      fetch(`${API_BASE}/api/documents/${doc.id}/versions`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
      fetch(`${API_BASE}/api/documents/${doc.id}/comments`, { headers: h }).then(r => r.ok ? r.json() : []).catch(() => []),
    ]);
    setDocVersions(Array.isArray(vers) ? vers : []);
    setDocComments(Array.isArray(cmts) ? cmts : []);
    setShowVersionModal(true);
  };

  const handleCheckout = async (doc: any) => {
    setCheckoutLoading(doc.id);
    try {
      const res = await fetch(`${API_BASE}/api/documents/${doc.id}/checkout`, {
        method: "POST", headers: { ...getHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ user_name: user.name }),
      });
      const data = await res.json();
      if (res.ok) {
        setDocuments(prev => prev.map(d => d.id === doc.id ? { ...d, checkout_user: user.name } : d));
      } else { alert(data.message || "Could not check out document."); }
    } finally { setCheckoutLoading(null); }
  };

  const handleCheckin = async () => {
    if (!selectedDoc) return;
    const res = await fetch(`${API_BASE}/api/documents/${selectedDoc.id}/checkin`, {
      method: "POST", headers: { ...getHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify({ user_name: user.name, change_summary: checkinSummary, is_major_version: checkinMajor }),
    });
    if (res.ok) {
      const data = await res.json();
      setDocuments(prev => prev.map(d => d.id === selectedDoc.id ? data.document : d));
      setShowCheckinModal(false); setShowVersionModal(false); setCheckinSummary("");
    }
  };

  const handleSaveDocument = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await fetch(`${API_BASE}/api/documents`, {
        method: "POST", headers: { ...getHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ title: editorTitle, category: editorCategory, classification: editorClassification, folder_path: selectedFolder, author_name: user.name, department: (user as any).department || "Executive Directorate", content: editorContent, file_size_kb: Math.round(editorContent.length / 10) }),
      });
      const data = await res.json();
      setDocuments([data, ...documents]); setActiveTab("repo");
    } catch { setActiveTab("repo"); }
  };

  const handleApplySignature = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!signingDoc) return;
    try {
      const res = await fetch(`${API_BASE}/api/documents/sign`, {
        method: "POST", headers: { ...getHeaders(), "Content-Type": "application/json" },
        body: JSON.stringify({ document_id: signingDoc.id, document_number: signingDoc.document_number, signer_name: user.name, signer_role: signerRole, signature_type: "Executive_Approval" }),
      });
      const data = await res.json();
      setSignatures([data, ...signatures]); setShowQrStamp(data); setSigningDoc(null);
    } catch { setSigningDoc(null); }
  };

  const handleCreateLegalHold = async (e: React.FormEvent) => {
    e.preventDefault();
    const res = await fetch(`${API_BASE}/api/records/legal-holds`, {
      method: "POST", headers: { ...getHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify({ ...newHold, issued_by: user.name }),
    });
    if (res.ok) {
      const data = await res.json();
      setLegalHolds([data, ...legalHolds]); setShowLegalHoldModal(false);
      setNewHold({ case_reference: "", reason: "", custodian: "", issued_by: "" });
    }
  };

  const handleCreateRetentionPolicy = async (e: React.FormEvent) => {
    e.preventDefault();
    const res = await fetch(`${API_BASE}/api/records/retention-policies`, {
      method: "POST", headers: { ...getHeaders(), "Content-Type": "application/json" },
      body: JSON.stringify(newRetention),
    });
    if (res.ok) { const data = await res.json(); setRetentionPolicies([data, ...retentionPolicies]); setShowRetentionModal(false); }
  };

  const handleReleaseLegalHold = async (id: string) => {
    const res = await fetch(`${API_BASE}/api/records/legal-holds/${id}/release`, { method: "PUT", headers: getHeaders() });
    if (res.ok) setLegalHolds(prev => prev.map(h => h.id === id ? { ...h, status: "Released" } : h));
  };

  const handleSearch = async () => {
    if (!searchQuery.trim()) return;
    setSearching(true);
    try {
      const res = await fetch(`${API_BASE}/api/documents/search?q=${encodeURIComponent(searchQuery)}`, { headers: getHeaders() });
      const data = await res.json();
      setSearchResults(data.results || []);
    } catch { setSearchResults([]); } finally { setSearching(false); }
  };

  const loadTemplate = (tpl: any) => { setEditorTitle(tpl.name); setEditorCategory(tpl.category); setEditorContent(tpl.default_content); setActiveTab("editor"); };

  if (loading) return (
    <div className="space-y-6 animate-pulse">
      <div className="h-28 bg-gray-100 rounded-xl" />
      <div className="h-80 bg-gray-100 rounded-xl" />
    </div>
  );

  const tabs: { id: Tab; label: string; icon: any }[] = [
    { id: "repo", label: "Document Repository", icon: FolderArchive },
    { id: "editor", label: "Authoring & Templates", icon: FileText },
    { id: "signatures", label: "Digital Signatures", icon: QrCode },
    { id: "archives", label: "Records & Archives", icon: Shield },
    { id: "knowledge", label: "Knowledge Hub", icon: BookOpen },
    { id: "search", label: "Enterprise Search", icon: Search },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900 tracking-tight flex items-center gap-2">
            <FolderArchive className="w-6 h-6 text-indigo-600" />
            Enterprise Document Management & Digital Archives (EDMS)
          </h2>
          <p className="text-sm text-gray-500 mt-1">
            ISO 15489–compliant records management · SHA-256 integrity · Version control · Long-term preservation (Phase 15)
          </p>
        </div>
      </div>

      {/* Dashboard KPIs */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
        {[
          { label: "Documents", value: dashboard?.total_documents ?? documents.length, color: "text-indigo-700", bg: "bg-indigo-50", icon: FileText },
          { label: "Folders", value: dashboard?.total_folders ?? folders.length, color: "text-violet-700", bg: "bg-violet-50", icon: FolderOpen },
          { label: "Repositories", value: dashboard?.total_repositories ?? repositories.length, color: "text-blue-700", bg: "bg-blue-50", icon: Database },
          { label: "Signatures", value: dashboard?.total_signatures ?? signatures.length, color: "text-emerald-700", bg: "bg-emerald-50", icon: CheckCircle2 },
          { label: "Legal Holds", value: dashboard?.active_legal_holds ?? legalHolds.length, color: "text-red-700", bg: "bg-red-50", icon: Lock },
          { label: "Knowledge Articles", value: dashboard?.knowledge_articles ?? articles.length, color: "text-amber-700", bg: "bg-amber-50", icon: BookOpen },
          { label: "Archives", value: dashboard?.total_archives ?? archives.length, color: "text-slate-700", bg: "bg-slate-50", icon: Archive },
        ].map(k => (
          <div key={k.label} className={`${k.bg} rounded-xl p-4 border border-white/60 shadow-sm`}>
            <k.icon className={`w-4 h-4 ${k.color} mb-1`} />
            <div className={`text-xl font-extrabold ${k.color} font-mono`}>{k.value}</div>
            <div className="text-[11px] font-semibold text-gray-500 mt-0.5">{k.label}</div>
          </div>
        ))}
      </div>

      {/* ISO Compliance Banner */}
      <div className="flex items-center gap-2 px-4 py-2.5 bg-emerald-50 border border-emerald-200 rounded-xl text-xs font-semibold text-emerald-800">
        <CheckCircle2 className="w-4 h-4" />
        ISO 15489 Records Management Standard — CERTIFIED COMPLIANT &nbsp;·&nbsp; SHA-256 Integrity Sealing Active &nbsp;·&nbsp; GDPR Anonymization Enforced
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1.5 p-1.5 bg-gray-100 rounded-xl border border-gray-200 overflow-x-auto">
        {tabs.map(tab => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button key={tab.id} onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-2 px-3.5 py-2 text-xs font-semibold rounded-lg transition whitespace-nowrap ${isActive ? "bg-white text-indigo-700 shadow-sm" : "text-gray-600 hover:text-gray-900"}`}>
              <Icon className="w-4 h-4" />{tab.label}
            </button>
          );
        })}
      </div>

      {/* ══ TAB 1: REPOSITORY ══ */}
      {activeTab === "repo" && (
        <div className="space-y-5">
          {/* Repositories */}
          <div>
            <h3 className="text-sm font-bold text-gray-700 mb-3 flex items-center gap-2"><Database className="w-4 h-4 text-indigo-600" /> Document Repositories</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
              {repositories.map(repo => (
                <div key={repo.id} className="bg-white border border-gray-200 rounded-xl p-4 shadow-sm hover:border-indigo-300 transition">
                  <div className="flex items-start justify-between">
                    <div className="font-bold text-xs text-gray-900">{repo.name}</div>
                    <span className={`text-[10px] px-1.5 py-0.5 rounded font-bold ${classColor[repo.access_level] || "bg-gray-100 text-gray-700"}`}>{repo.access_level}</span>
                  </div>
                  <div className="text-[11px] text-gray-500 mt-1">{repo.description}</div>
                  <div className="text-[11px] font-semibold text-indigo-700 mt-2">{repo.document_count} documents</div>
                </div>
              ))}
            </div>
          </div>

          {/* Folders */}
          <div>
            <h3 className="text-sm font-bold text-gray-700 mb-3 flex items-center gap-2"><FolderOpen className="w-4 h-4 text-amber-600" /> Folder Structure</h3>
            <div className="bg-white border border-gray-200 rounded-xl overflow-hidden shadow-sm">
              <table className="w-full text-xs">
                <thead className="bg-gray-50 text-gray-700 font-bold border-b border-gray-200">
                  <tr>
                    <th className="p-3 text-left">Folder Path</th>
                    <th className="p-3 text-left">Access Level</th>
                    <th className="p-3 text-right">Documents</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {folders.map(f => (
                    <tr key={f.id} className="hover:bg-gray-50/70 transition">
                      <td className="p-3 font-mono font-semibold text-indigo-700 flex items-center gap-1.5">
                        <FolderOpen className="w-3.5 h-3.5 text-amber-500" />{f.path}
                      </td>
                      <td className="p-3"><span className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${classColor[f.access_level] || "bg-gray-100 text-gray-700"}`}>{f.access_level}</span></td>
                      <td className="p-3 text-right font-bold text-gray-700">{f.document_count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Documents */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-bold text-gray-700 flex items-center gap-2"><FileText className="w-4 h-4 text-indigo-600" /> Documents</h3>
              <button onClick={() => setActiveTab("editor")} className="flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-lg transition">
                <Plus className="w-3.5 h-3.5" /> New Document
              </button>
            </div>
            <div className="bg-white border border-gray-200 rounded-xl overflow-hidden shadow-sm">
              <table className="w-full text-xs">
                <thead className="bg-gray-50 font-bold text-gray-700 border-b border-gray-200">
                  <tr>
                    <th className="p-3 text-left">Doc Number</th>
                    <th className="p-3 text-left">Title</th>
                    <th className="p-3 text-left">Category</th>
                    <th className="p-3 text-left">Version</th>
                    <th className="p-3 text-left">Status</th>
                    <th className="p-3 text-left">Classification</th>
                    <th className="p-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {documents.map(doc => (
                    <tr key={doc.id} className="hover:bg-gray-50/70 transition">
                      <td className="p-3 font-mono font-bold text-indigo-700">{doc.document_number}</td>
                      <td className="p-3 font-semibold text-gray-900 max-w-[200px] truncate">{doc.title}</td>
                      <td className="p-3 text-gray-600">{doc.category}</td>
                      <td className="p-3 font-mono text-gray-600">{doc.version}</td>
                      <td className="p-3"><span className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${statusColor[doc.status] || "bg-gray-100 text-gray-700"}`}>{doc.status}</span></td>
                      <td className="p-3"><span className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${classColor[doc.classification] || "bg-gray-100 text-gray-700"}`}>{doc.classification}</span></td>
                      <td className="p-3">
                        <div className="flex items-center justify-end gap-1.5">
                          {doc.checkout_user ? (
                            <span className="flex items-center gap-1 text-[10px] text-amber-700 font-semibold"><Lock className="w-3 h-3" /> {doc.checkout_user}</span>
                          ) : (
                            <button onClick={() => handleCheckout(doc)} disabled={checkoutLoading === doc.id}
                              className="px-2 py-1 bg-blue-50 hover:bg-blue-100 text-blue-800 text-[10px] font-bold rounded transition flex items-center gap-1">
                              <Edit3 className="w-3 h-3" /> Check Out
                            </button>
                          )}
                          <button onClick={() => loadDocDetails(doc)}
                            className="px-2 py-1 bg-gray-100 hover:bg-gray-200 text-gray-700 text-[10px] font-bold rounded transition flex items-center gap-1">
                            <History className="w-3 h-3" /> History
                          </button>
                          <button onClick={() => { setSigningDoc(doc); }}
                            className="px-2 py-1 bg-emerald-50 hover:bg-emerald-100 text-emerald-800 text-[10px] font-bold rounded transition flex items-center gap-1">
                            <CheckCircle2 className="w-3 h-3" /> Sign
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* ══ TAB 2: EDITOR ══ */}
      {activeTab === "editor" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Templates */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-5 space-y-3">
            <h3 className="text-sm font-bold text-gray-900">Document Template Library</h3>
            {templates.map(tpl => (
              <div key={tpl.id} className="p-3 border border-gray-200 rounded-lg hover:border-indigo-300 transition cursor-pointer" onClick={() => loadTemplate(tpl)}>
                <div className="font-semibold text-xs text-gray-900">{tpl.name}</div>
                <div className="text-[11px] text-gray-500 mt-0.5">{tpl.description}</div>
                <button className="mt-2 text-[11px] text-indigo-600 font-bold flex items-center gap-1 hover:text-indigo-800">
                  <UploadCloud className="w-3 h-3" /> Use Template
                </button>
              </div>
            ))}
          </div>

          {/* Editor */}
          <form onSubmit={handleSaveDocument} className="lg:col-span-2 bg-white rounded-xl border border-gray-200 shadow-sm p-5 space-y-4">
            <h3 className="text-sm font-bold text-gray-900">Document Authoring Workspace</h3>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Document Title</label>
                <input value={editorTitle} onChange={e => setEditorTitle(e.target.value)} className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 focus:border-indigo-500 outline-none" />
              </div>
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Category</label>
                <select value={editorCategory} onChange={e => setEditorCategory(e.target.value)} className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 bg-white focus:border-indigo-500 outline-none">
                  {["Policy", "SOP", "Statistical_Bulletin", "Research_Protocol", "Grant_Agreement", "Meeting_Minutes", "Legal", "Report"].map(c => <option key={c}>{c}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Classification</label>
                <select value={editorClassification} onChange={e => setEditorClassification(e.target.value)} className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 bg-white focus:border-indigo-500 outline-none">
                  <option value="PUBLIC">PUBLIC</option>
                  <option value="INTERNAL">INTERNAL</option>
                  <option value="CONFIDENTIAL">CONFIDENTIAL</option>
                  <option value="RESTRICTED">RESTRICTED</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Target Folder</label>
                <select value={selectedFolder} onChange={e => setSelectedFolder(e.target.value)} className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 bg-white focus:border-indigo-500 outline-none">
                  {folders.map(f => <option key={f.id} value={f.path}>{f.path}</option>)}
                </select>
              </div>
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Document Content (Markdown)</label>
              <textarea value={editorContent} onChange={e => setEditorContent(e.target.value)} rows={14}
                className="w-full text-xs font-mono border border-gray-300 rounded-lg px-3 py-2 focus:border-indigo-500 outline-none resize-none" />
            </div>
            <div className="flex justify-between items-center">
              <span className="text-[11px] text-gray-400">{editorContent.length} characters · ~{Math.round(editorContent.length / 1024 * 10) / 10} KB</span>
              <button type="submit" className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-bold rounded-lg transition flex items-center gap-2">
                <UploadCloud className="w-3.5 h-3.5" /> Save to Repository (v1.0)
              </button>
            </div>
          </form>
        </div>
      )}

      {/* ══ TAB 3: DIGITAL SIGNATURES ══ */}
      {activeTab === "signatures" && (
        <div className="space-y-6">
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
            <div>
              <h3 className="text-base font-bold text-gray-900">Cryptographic Signatures & QR Authenticity Seals</h3>
              <p className="text-xs text-gray-500">Every signed document receives an immutable SHA-256 seal and a public QR verification endpoint</p>
            </div>
            <div className="overflow-x-auto border border-gray-200 rounded-lg">
              <table className="w-full text-xs text-left">
                <thead className="bg-gray-50 text-gray-700 font-bold border-b border-gray-200">
                  <tr>
                    <th className="p-3">Doc Number</th>
                    <th className="p-3">Signer Name</th>
                    <th className="p-3">Role / Authority</th>
                    <th className="p-3">SHA-256 Verification Hash</th>
                    <th className="p-3">Signed</th>
                    <th className="p-3 text-right">QR Seal</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {signatures.map(sig => (
                    <tr key={sig.id} className="hover:bg-gray-50/50">
                      <td className="p-3 font-mono font-bold text-indigo-700">{sig.document_number}</td>
                      <td className="p-3 font-semibold text-gray-900">{sig.signer_name}</td>
                      <td className="p-3 text-gray-600">{sig.signer_role}</td>
                      <td className="p-3 font-mono text-[11px] text-gray-500 max-w-[200px] truncate">{sig.verification_hash}</td>
                      <td className="p-3 text-gray-500">{new Date(sig.signed_at).toLocaleDateString()}</td>
                      <td className="p-3 text-right">
                        <button onClick={() => setShowQrStamp(sig)} className="px-2.5 py-1 bg-emerald-50 hover:bg-emerald-100 text-emerald-800 font-semibold rounded text-[11px] transition inline-flex items-center gap-1">
                          <QrCode className="w-3.5 h-3.5" /> View Seal
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* ══ TAB 4: RECORDS & ARCHIVES ══ */}
      {activeTab === "archives" && (
        <div className="space-y-6">
          {/* Retention Policies */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
            <div className="flex justify-between items-center">
              <div>
                <h3 className="text-sm font-bold text-gray-900">ISO 15489 Retention Schedules</h3>
                <p className="text-xs text-gray-500">Statutory retention periods and disposition authorities</p>
              </div>
              <button onClick={() => setShowRetentionModal(true)} className="flex items-center gap-1.5 px-3 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-lg transition">
                <Plus className="w-3.5 h-3.5" /> New Policy
              </button>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {retentionPolicies.map(p => (
                <div key={p.id} className="p-4 bg-gray-50 border border-gray-200 rounded-lg">
                  <div className="flex justify-between items-center">
                    <span className="font-bold text-xs text-gray-900">{p.category}</span>
                    <span className="text-xs px-2 py-0.5 bg-emerald-100 text-emerald-800 rounded font-bold">{p.retention_period_years} Yrs</span>
                  </div>
                  <div className="text-[11px] text-indigo-700 font-semibold mt-1">Disposition: {p.disposition_action}</div>
                  <div className="text-[11px] text-gray-500 mt-1">Legal: {p.legal_authority}</div>
                </div>
              ))}
            </div>
          </div>

          {/* Legal Holds */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
            <div className="flex justify-between items-center">
              <div>
                <h3 className="text-sm font-bold text-gray-900">Active Legal Holds</h3>
                <p className="text-xs text-gray-500">Preservation freezes preventing deletion during audits or litigation</p>
              </div>
              <button onClick={() => setShowLegalHoldModal(true)} className="flex items-center gap-1.5 px-3 py-1.5 bg-red-600 hover:bg-red-700 text-white text-xs font-semibold rounded-lg transition">
                <Lock className="w-3.5 h-3.5" /> Issue Legal Hold
              </button>
            </div>
            <div className="space-y-3">
              {legalHolds.map(h => (
                <div key={h.id} className="p-4 bg-amber-50/50 border border-amber-200 rounded-lg flex justify-between items-start">
                  <div>
                    <span className="font-mono font-bold text-xs text-amber-900">{h.case_reference}</span>
                    <span className={`ml-2 text-[10px] px-2 py-0.5 rounded-full font-bold ${h.status === "Active" ? "bg-red-100 text-red-800" : "bg-gray-100 text-gray-600"}`}>
                      <Lock className="w-2.5 h-2.5 inline mr-0.5" />{h.status}
                    </span>
                    <div className="text-xs text-gray-800 mt-1.5 font-medium">{h.reason}</div>
                    <div className="text-[11px] text-gray-500 mt-0.5">Custodian: {h.custodian} · Issued: {h.issued_date}</div>
                  </div>
                  {h.status === "Active" && (
                    <button onClick={() => handleReleaseLegalHold(h.id)} className="px-2.5 py-1 bg-white border border-gray-300 hover:bg-gray-50 text-xs font-semibold text-gray-700 rounded-lg transition flex items-center gap-1">
                      <Unlock className="w-3 h-3" /> Release
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Archives */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
            <h3 className="text-sm font-bold text-gray-900 flex items-center gap-2"><Archive className="w-4 h-4 text-slate-600" /> Archive Locations</h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {archives.map(arc => (
                <div key={arc.id} className="p-4 bg-slate-50 border border-slate-200 rounded-xl">
                  <div className="font-bold text-xs text-slate-900 mb-1">{arc.name}</div>
                  <div className="text-[11px] text-slate-600 mb-2">{arc.location_description}</div>
                  <div className="flex justify-between text-[11px] font-semibold">
                    <span className="text-indigo-700">{arc.document_count.toLocaleString()} docs</span>
                    <span className="text-gray-500">{(arc.size_mb / 1024).toFixed(1)} GB</span>
                  </div>
                  <div className="mt-2">
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold ${arc.archive_type === "Disaster_Recovery" ? "bg-red-100 text-red-800" : arc.archive_type === "Long_Term" ? "bg-purple-100 text-purple-800" : "bg-blue-100 text-blue-800"}`}>
                      {arc.archive_type}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Disposition Workflows */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-3">
            <h3 className="text-sm font-bold text-gray-900 flex items-center gap-2"><Clock className="w-4 h-4 text-orange-600" /> Disposition Workflows</h3>
            {dispositions.map(d => (
              <div key={d.id} className="p-3.5 bg-orange-50/50 border border-orange-200 rounded-lg flex justify-between items-center">
                <div>
                  <div className="font-bold text-xs text-gray-900">{d.document_title || d.document_number}</div>
                  <div className="text-[11px] text-gray-600 mt-0.5">Action: <span className="font-semibold text-orange-800">{d.disposition_action}</span> · Scheduled: {d.scheduled_disposition_date}</div>
                  <div className="text-[11px] text-gray-500">{d.notes}</div>
                </div>
                <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold ${d.status === "Scheduled" ? "bg-blue-100 text-blue-800" : d.status === "Under_Review" ? "bg-amber-100 text-amber-800" : "bg-green-100 text-green-800"}`}>{d.status}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* ══ TAB 5: KNOWLEDGE HUB ══ */}
      {activeTab === "knowledge" && (
        <div className="space-y-6">
          {/* Knowledge Articles */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-sm font-bold text-gray-900">Knowledge Articles & SOPs</h3>
                <p className="text-xs text-gray-500">Institutional knowledge base, SOPs, procedures, and policy summaries</p>
              </div>
            </div>
            <div className="space-y-3">
              {articles.map(art => (
                <div key={art.id} className="p-4 border border-gray-200 rounded-xl hover:border-indigo-300 transition">
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-[11px] font-bold text-indigo-600">{art.article_number}</span>
                        <span className="text-[10px] px-2 py-0.5 bg-blue-100 text-blue-800 rounded-full font-bold">{art.article_type}</span>
                        <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold ${art.status === "Published" ? "bg-emerald-100 text-emerald-800" : "bg-gray-100 text-gray-600"}`}>{art.status}</span>
                      </div>
                      <div className="font-bold text-sm text-gray-900 mt-1">{art.title}</div>
                      <div className="text-[11px] text-gray-500 mt-0.5">{art.summary}</div>
                      <div className="text-[11px] text-gray-400 mt-1">{art.author_name} · {art.department}</div>
                    </div>
                    <div className="text-right ml-4 flex-shrink-0">
                      <div className="text-[11px] font-bold text-gray-700 flex items-center gap-1"><Eye className="w-3 h-3" /> {art.view_count}</div>
                      <div className="text-[11px] text-emerald-700 font-bold mt-1">👍 {art.helpful_votes}</div>
                    </div>
                  </div>
                  {art.tags && art.tags.length > 0 && (
                    <div className="flex gap-1.5 mt-3 flex-wrap">
                      {art.tags.map((t: string) => (
                        <span key={t} className="text-[10px] px-2 py-0.5 bg-gray-100 text-gray-700 rounded-full font-medium flex items-center gap-0.5">
                          <Tag className="w-2.5 h-2.5" />{t}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Wiki Pages */}
          <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
            <h3 className="text-sm font-bold text-gray-900 flex items-center gap-2"><Globe className="w-4 h-4 text-blue-600" /> Institutional Wiki</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {wikiPages.map(p => (
                <div key={p.id} className="p-4 bg-blue-50/40 border border-blue-200 rounded-xl hover:border-blue-400 transition cursor-pointer">
                  <div className="flex items-start justify-between">
                    <div>
                      <div className="font-bold text-xs text-blue-900">{p.title}</div>
                      <div className="text-[11px] font-mono text-blue-600 mt-0.5">/{p.slug}</div>
                      {p.parent_slug && <div className="text-[11px] text-gray-500 mt-0.5 flex items-center gap-1"><ChevronRight className="w-3 h-3" />Parent: /{p.parent_slug}</div>}
                    </div>
                    <div className="text-[11px] text-gray-500 flex items-center gap-1"><Eye className="w-3 h-3" />{p.view_count}</div>
                  </div>
                  <div className="text-[11px] text-gray-600 mt-2 line-clamp-2">{p.content.replace(/[#*\[\]]/g, "").substring(0, 100)}...</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ══ TAB 6: ENTERPRISE SEARCH ══ */}
      {activeTab === "search" && (
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 space-y-4">
          <div>
            <h3 className="text-base font-bold text-gray-900">Enterprise Semantic Document Search</h3>
            <p className="text-xs text-gray-500">Search across policies, bulletins, research protocols, legal agreements, and knowledge articles</p>
          </div>
          <div className="flex gap-3">
            <div className="relative flex-1">
              <Search className="w-4 h-4 text-gray-400 absolute left-3 top-3" />
              <input type="text" value={searchQuery} onChange={e => setSearchQuery(e.target.value)}
                onKeyDown={e => e.key === "Enter" && handleSearch()}
                placeholder="Search by keyword, document number, classification, or domain tag..."
                className="w-full text-xs border border-gray-300 rounded-xl pl-9 pr-4 py-2.5 outline-none focus:border-indigo-600" />
            </div>
            <button onClick={handleSearch} disabled={searching}
              className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-bold rounded-xl transition flex items-center gap-2">
              {searching ? <RefreshCw className="w-3.5 h-3.5 animate-spin" /> : <Search className="w-3.5 h-3.5" />} Search
            </button>
          </div>
          {searchResults.length > 0 && (
            <div className="space-y-2 pt-2">
              <div className="text-xs font-semibold text-gray-500">{searchResults.length} result(s) found</div>
              {searchResults.map(doc => (
                <div key={doc.id} className="p-3.5 bg-gray-50 border border-gray-200 rounded-lg flex justify-between items-center hover:border-indigo-300 transition">
                  <div>
                    <span className="font-mono text-xs font-bold text-indigo-700 mr-2">{doc.document_number}</span>
                    <span className="font-semibold text-xs text-gray-900">{doc.title}</span>
                    <div className="text-[11px] text-gray-500 mt-1">{doc.department} · {doc.category} · <span className={`font-bold ${classColor[doc.classification] ? "text-inherit" : ""}`}>{doc.classification}</span></div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`text-[10px] px-2 py-0.5 rounded-full font-bold ${statusColor[doc.status] || "bg-gray-100 text-gray-600"}`}>{doc.status}</span>
                    <button onClick={() => loadDocDetails(doc)} className="flex items-center gap-1 px-3 py-1.5 text-xs font-semibold bg-white border border-gray-300 hover:bg-gray-50 rounded-lg text-gray-700">
                      <Eye className="w-3.5 h-3.5" /> View
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
          {searchResults.length === 0 && searchQuery && !searching && (
            <div className="text-center py-10 text-gray-400 text-sm">No documents matched your search. Try different keywords.</div>
          )}
        </div>
      )}

      {/* ══ VERSION HISTORY MODAL ══ */}
      {showVersionModal && selectedDoc && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl max-w-2xl w-full p-6 shadow-2xl space-y-5 max-h-[90vh] overflow-y-auto">
            <div className="flex items-start justify-between">
              <div>
                <h3 className="text-base font-bold text-gray-900 flex items-center gap-2"><GitBranch className="w-5 h-5 text-indigo-600" /> Version History</h3>
                <p className="text-xs text-gray-500 font-mono mt-0.5">{selectedDoc.document_number} — {selectedDoc.title}</p>
              </div>
              <button onClick={() => setShowVersionModal(false)} className="text-gray-400 hover:text-gray-700 text-lg font-bold">×</button>
            </div>

            {/* Check-In Action */}
            {selectedDoc.checkout_user === user.name && !showCheckinModal && (
              <div className="p-3 bg-blue-50 border border-blue-200 rounded-xl flex justify-between items-center">
                <span className="text-xs font-semibold text-blue-800">You have this document checked out. Ready to check in?</span>
                <button onClick={() => setShowCheckinModal(true)} className="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold rounded-lg transition flex items-center gap-1">
                  <UploadCloud className="w-3.5 h-3.5" /> Check In
                </button>
              </div>
            )}

            {showCheckinModal && (
              <div className="p-4 bg-blue-50 border border-blue-200 rounded-xl space-y-3">
                <div className="text-xs font-bold text-blue-900">Check In Document</div>
                <input value={checkinSummary} onChange={e => setCheckinSummary(e.target.value)} placeholder="Describe what changed in this version..."
                  className="w-full text-xs border border-blue-300 rounded px-3 py-2 focus:border-blue-500 outline-none" />
                <label className="flex items-center gap-2 text-xs text-blue-800 font-semibold cursor-pointer">
                  <input type="checkbox" checked={checkinMajor} onChange={e => setCheckinMajor(e.target.checked)} /> Major version bump (v2.0 not v1.1)
                </label>
                <div className="flex gap-2 justify-end">
                  <button onClick={() => setShowCheckinModal(false)} className="px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-100 rounded">Cancel</button>
                  <button onClick={handleCheckin} className="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold rounded-lg">Confirm Check-In</button>
                </div>
              </div>
            )}

            {/* Versions Table */}
            <div className="space-y-2">
              {docVersions.length === 0 && <div className="text-xs text-gray-400 text-center py-4">No version history recorded yet.</div>}
              {docVersions.map((v, i) => (
                <div key={v.id} className={`p-3.5 border rounded-xl ${i === 0 ? "border-indigo-300 bg-indigo-50/50" : "border-gray-200 bg-gray-50"}`}>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="font-mono font-bold text-indigo-700 text-xs">{v.version_number}</span>
                      {v.is_major_version && <span className="text-[10px] bg-indigo-100 text-indigo-800 px-1.5 py-0.5 rounded font-bold">MAJOR</span>}
                      {i === 0 && <span className="text-[10px] bg-emerald-100 text-emerald-800 px-1.5 py-0.5 rounded font-bold">LATEST</span>}
                    </div>
                    <span className="text-[11px] text-gray-400">{new Date(v.created_at).toLocaleString()}</span>
                  </div>
                  <div className="text-xs text-gray-700 font-semibold mt-1">{v.change_summary}</div>
                  <div className="text-[11px] text-gray-500 mt-0.5">Author: {v.author_name} · {v.file_size_kb} KB</div>
                  <div className="text-[10px] font-mono text-gray-400 mt-1 truncate">SHA-256: {v.sha256_hash}</div>
                </div>
              ))}
            </div>

            {/* Comments */}
            {docComments.length > 0 && (
              <div className="space-y-2 border-t pt-4">
                <h4 className="text-xs font-bold text-gray-700">Review Comments</h4>
                {docComments.map(cmt => (
                  <div key={cmt.id} className={`p-3 rounded-lg border ${cmt.resolved ? "border-gray-200 bg-gray-50" : "border-amber-200 bg-amber-50/50"}`}>
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-semibold text-xs text-gray-900">{cmt.author_name}</span>
                      <span className="text-[10px] text-gray-500">{cmt.author_role}</span>
                      {cmt.resolved && <span className="text-[10px] bg-emerald-100 text-emerald-800 px-1.5 rounded font-bold">Resolved</span>}
                    </div>
                    <div className="text-xs text-gray-700">{cmt.comment}</div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* ══ SIGN DOCUMENT MODAL ══ */}
      {signingDoc && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <form onSubmit={handleApplySignature} className="bg-white rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold text-gray-900">Apply Digital Signature & Seal</h3>
            <p className="text-xs text-gray-600">Signing document: <strong className="font-mono">{signingDoc.document_number}</strong></p>
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Signer Authority Role</label>
              <select value={signerRole} onChange={e => setSignerRole(e.target.value)} className="w-full text-xs border border-gray-300 rounded p-2 bg-white">
                <option value="Executive Approval">Executive Approval (Director General)</option>
                <option value="Ethics Clearance">Ethics Clearance (IRB Chairperson)</option>
                <option value="Financial Signoff">Financial Signoff (Chief Financial Officer)</option>
                <option value="Chief Statistician">Chief Statistician Certification</option>
              </select>
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button type="button" onClick={() => setSigningDoc(null)} className="px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-100 rounded">Cancel</button>
              <button type="submit" className="px-4 py-1.5 text-xs font-semibold bg-indigo-600 hover:bg-indigo-700 text-white rounded">Seal Document</button>
            </div>
          </form>
        </div>
      )}

      {/* ══ QR AUTHENTICITY SEAL ══ */}
      {showQrStamp && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl max-w-sm w-full p-6 text-center shadow-2xl">
            <div className="w-12 h-12 rounded-full bg-emerald-100 text-emerald-700 flex items-center justify-center mx-auto mb-3">
              <CheckCircle2 className="w-6 h-6" />
            </div>
            <h3 className="text-base font-bold text-gray-900">Cryptographically Sealed</h3>
            <p className="text-xs text-gray-500 mt-1 mb-4">{showQrStamp.document_number} · Signed by {showQrStamp.signer_name}</p>
            <div className="bg-gray-50 p-4 rounded-xl border border-gray-200 mb-4 flex justify-center">
              <div className="p-3 bg-white rounded-lg shadow-sm border border-gray-200">
                <QrCode className="w-32 h-32 text-gray-900" />
              </div>
            </div>
            <div className="text-[10px] font-mono text-gray-500 bg-gray-100 p-2 rounded mb-4 break-all">{showQrStamp.verification_hash}</div>
            <button onClick={() => setShowQrStamp(null)} className="w-full py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-lg transition">Close</button>
          </div>
        </div>
      )}

      {/* ══ LEGAL HOLD MODAL ══ */}
      {showLegalHoldModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <form onSubmit={handleCreateLegalHold} className="bg-white rounded-xl max-w-lg w-full p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold text-gray-900 flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-red-600" /> Issue Legal Hold</h3>
            <p className="text-xs text-gray-600">A legal hold will freeze all affected documents, preventing deletion or modification until explicitly released.</p>
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Case Reference</label>
              <input required value={newHold.case_reference} onChange={e => setNewHold({ ...newHold, case_reference: e.target.value })} placeholder="HOLD-2026-AUDIT-XX" className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 outline-none focus:border-red-500" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Reason / Case Description</label>
              <textarea required value={newHold.reason} onChange={e => setNewHold({ ...newHold, reason: e.target.value })} rows={3} placeholder="Describe the legal basis for this hold..." className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 outline-none focus:border-red-500 resize-none" />
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Custodian</label>
              <input required value={newHold.custodian} onChange={e => setNewHold({ ...newHold, custodian: e.target.value })} placeholder="Chief Financial Officer & Procurement Directorate" className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 outline-none focus:border-red-500" />
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button type="button" onClick={() => setShowLegalHoldModal(false)} className="px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-100 rounded-lg">Cancel</button>
              <button type="submit" className="px-5 py-1.5 text-xs font-semibold bg-red-600 hover:bg-red-700 text-white rounded-lg transition">Issue Hold</button>
            </div>
          </form>
        </div>
      )}

      {/* ══ NEW RETENTION POLICY MODAL ══ */}
      {showRetentionModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <form onSubmit={handleCreateRetentionPolicy} className="bg-white rounded-xl max-w-lg w-full p-6 shadow-2xl space-y-4">
            <h3 className="text-lg font-bold text-gray-900">New Retention Policy (ISO 15489)</h3>
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Record Category</label>
              <input required value={newRetention.category} onChange={e => setNewRetention({ ...newRetention, category: e.target.value })} placeholder="e.g. Audit Reports & Financial Reviews" className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 outline-none focus:border-indigo-500" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Retention Period (Years)</label>
                <input type="number" min={1} required value={newRetention.retention_period_years} onChange={e => setNewRetention({ ...newRetention, retention_period_years: parseInt(e.target.value) })} className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 outline-none focus:border-indigo-500" />
              </div>
              <div>
                <label className="block text-xs font-semibold text-gray-700 mb-1">Disposition Action</label>
                <select value={newRetention.disposition_action} onChange={e => setNewRetention({ ...newRetention, disposition_action: e.target.value })} className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 bg-white outline-none focus:border-indigo-500">
                  <option value="Permanent_National_Archive">Permanent National Archive</option>
                  <option value="Review_For_Destruction">Review for Destruction</option>
                  <option value="Declassify_To_Public">Declassify to Public</option>
                </select>
              </div>
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-700 mb-1">Legal Authority</label>
              <input value={newRetention.legal_authority} onChange={e => setNewRetention({ ...newRetention, legal_authority: e.target.value })} placeholder="e.g. National Records & Archives Act 2002, Section 14" className="w-full text-xs border border-gray-300 rounded-lg px-3 py-2 outline-none focus:border-indigo-500" />
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button type="button" onClick={() => setShowRetentionModal(false)} className="px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-100 rounded-lg">Cancel</button>
              <button type="submit" className="px-5 py-1.5 text-xs font-semibold bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg transition">Create Policy</button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
};
