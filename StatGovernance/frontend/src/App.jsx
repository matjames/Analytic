import React, { useState, useEffect, useCallback } from 'react';
import OverviewTab from './components/OverviewTab';
import PoliciesTab from './components/PoliciesTab';
import SOPsTab from './components/SOPsTab';
import ComplianceTab from './components/ComplianceTab';
import RisksTab from './components/RisksTab';
import ControlsTab from './components/ControlsTab';
import AuditsTab from './components/AuditsTab';
import FindingsTab from './components/FindingsTab';
import CommitteesTab from './components/CommitteesTab';
import DecisionsTab from './components/DecisionsTab';
import DataGovernanceTab from './components/DataGovernanceTab';
import EvidenceTab from './components/EvidenceTab';
import DelegationsTab from './components/DelegationsTab';
import AIGovernancePanel from './components/AIGovernancePanel';
import WhistleblowerTab from './components/WhistleblowerTab';
import ConflictOfInterestTab from './components/ConflictOfInterestTab';
import AdminConfigTab from './components/AdminConfigTab';

export default function App() {
  const [activeTab, setActiveTab] = useState('overview');
  const [kpis, setKpis] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [showSearchModal, setShowSearchModal] = useState(false);
  const [userRole, setUserRole] = useState('Governance Officer');
  const [tenantID, setTenantID] = useState('tenant-alpha');
  const [branding, setBranding] = useState({
    display_name: 'StatGate',
    logo_url: '',
    primary_color: '#0f3f5f',
    secondary_color: '#082236',
  });

  // Backend API URL
  const apiBase = import.meta.env.VITE_GOVERNANCE_API_URL || 'http://localhost:8093';
  const registryAPI = import.meta.env.VITE_REGISTRY_API_URL || 'http://localhost:9090/api';

  // Single-sign-on bootstrap: a Registry-issued token can be handed over by the
  // launcher via ?statgate_token (canonical) or ?registry_token (legacy alias).
  // We persist it, strip it from the address bar immediately, and NEVER fabricate
  // a demo token — a missing token means signed out (the fail-closed backend would
  // reject an unauthenticated session anyway).
  const token = (() => {
    if (typeof window === 'undefined') return '';
    const sp = new URLSearchParams(window.location.search);
    const handed = sp.get('statgate_token') || sp.get('registry_token');
    if (handed) {
      localStorage.setItem('registry_jwt', handed);
      const u = new URL(window.location.href);
      u.searchParams.delete('statgate_token');
      u.searchParams.delete('registry_token');
      window.history.replaceState({}, document.title, `${u.pathname}${u.search}${u.hash}`);
    }
    return (handed || localStorage.getItem('registry_jwt') || '').trim();
  })();

  // Sync tab with URL
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const tabParam = params.get('tab');
    if (tabParam) {
      setActiveTab(tabParam);
    }
  }, []);

  const changeTab = (tab) => {
    setActiveTab(tab);
    const url = new URL(window.location.href);
    url.searchParams.set('tab', tab);
    window.history.pushState({}, '', url);
  };

  const fetchDashboardData = useCallback(async () => {
    try {
      const res = await fetch(`${apiBase}/api/dashboard`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setKpis(data);
      }
    } catch (err) {
      console.error('Error loading governance dashboard telemetry:', err);
    }
  }, [apiBase, token]);

  useEffect(() => {
    fetchDashboardData();
    const interval = setInterval(fetchDashboardData, 30000); // 30s live telemetry refresh
    return () => clearInterval(interval);
  }, [fetchDashboardData]);

  useEffect(() => {
    if (!token) return;
    fetch(`${registryAPI}/organisation/branding`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (!data) return;
        setBranding((current) => ({
          display_name: data.display_name || current.display_name,
          logo_url: data.logo_url || '',
          primary_color: data.primary_color || current.primary_color,
          secondary_color: data.secondary_color || current.secondary_color,
        }));
      })
      .catch(() => {});
  }, [registryAPI, token]);

  const handleGlobalSearch = async (e) => {
    e.preventDefault();
    if (!searchQuery.trim()) return;

    try {
      const res = await fetch(`${apiBase}/api/search?q=${encodeURIComponent(searchQuery)}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        setSearchResults(await res.json());
        setShowSearchModal(true);
      }
    } catch (err) {
      console.error('Error during global search:', err);
    }
  };

  return (
    <div
      className="app-container"
      style={{ '--bg-sidebar': `linear-gradient(180deg, ${branding.primary_color} 0%, ${branding.secondary_color} 100%)` }}
    >
      {/* Sidebar */}
      <aside className="sidebar">
        <div className="brand-section">
          <img src={branding.logo_url || '/logo.png'} alt={`${branding.display_name} logo`} className="brand-icon" onError={(e) => { e.target.style.display='none'; }} style={{ objectFit:'contain', borderRadius:5, background:'rgba(255,255,255,0.1)', padding:2 }} />
          <div className="brand-info">
            <div className="brand-title">{branding.display_name}</div>
            <div className="brand-subtitle">Institutional Control</div>
          </div>
        </div>

        <div className="sidebar-category">Executive & Control</div>
        <div className={`nav-item ${activeTab === 'overview' ? 'active' : ''}`} onClick={() => changeTab('overview')}>
          <span className="nav-icon">📊</span>
          <span>Governance Overview</span>
        </div>
        <div className={`nav-item ${activeTab === 'ai-advisory' ? 'active' : ''}`} onClick={() => changeTab('ai-advisory')}>
          <span className="nav-icon">🤖</span>
          <span>AI Intelligence</span>
          <span className="nav-badge" style={{ background: '#0284c7' }}>Phase VII</span>
        </div>

        <div className="sidebar-category">Governance Framework</div>
        <div className={`nav-item ${activeTab === 'policies' ? 'active' : ''}`} onClick={() => changeTab('policies')}>
          <span className="nav-icon">📜</span>
          <span>Policy Management</span>
          {kpis?.policies_awaiting_approval > 0 && (
            <span className="nav-badge">{kpis.policies_awaiting_approval}</span>
          )}
        </div>
        <div className={`nav-item ${activeTab === 'sops' ? 'active' : ''}`} onClick={() => changeTab('sops')}>
          <span className="nav-icon">📑</span>
          <span>Procedures & SOPs</span>
        </div>
        <div className={`nav-item ${activeTab === 'compliance' ? 'active' : ''}`} onClick={() => changeTab('compliance')}>
          <span className="nav-icon">⚖️</span>
          <span>Compliance & Regulations</span>
        </div>

        <div className="sidebar-category">Risk & Assurance</div>
        <div className={`nav-item ${activeTab === 'risks' ? 'active' : ''}`} onClick={() => changeTab('risks')}>
          <span className="nav-icon">🎯</span>
          <span>Risk Register & Matrix</span>
          {kpis?.risks_critical > 0 && (
            <span className="nav-badge">{kpis.risks_critical}</span>
          )}
        </div>
        <div className={`nav-item ${activeTab === 'controls' ? 'active' : ''}`} onClick={() => changeTab('controls')}>
          <span className="nav-icon">🛡️</span>
          <span>Control Library & Testing</span>
        </div>
        <div className={`nav-item ${activeTab === 'audits' ? 'active' : ''}`} onClick={() => changeTab('audits')}>
          <span className="nav-icon">📋</span>
          <span>Audit Engagements</span>
        </div>
        <div className={`nav-item ${activeTab === 'findings' ? 'active' : ''}`} onClick={() => changeTab('findings')}>
          <span className="nav-icon">🔍</span>
          <span>Findings & CAPA</span>
          {kpis?.findings_open > 0 && (
            <span className="nav-badge">{kpis.findings_open}</span>
          )}
        </div>

        <div className="sidebar-category">Governance Bodies & Assets</div>
        <div className={`nav-item ${activeTab === 'committees' ? 'active' : ''}`} onClick={() => changeTab('committees')}>
          <span className="nav-icon">🏛️</span>
          <span>Committees & Meetings</span>
        </div>
        <div className={`nav-item ${activeTab === 'decisions' ? 'active' : ''}`} onClick={() => changeTab('decisions')}>
          <span className="nav-icon">✍️</span>
          <span>Decision Records</span>
        </div>
        <div className={`nav-item ${activeTab === 'data-gov' ? 'active' : ''}`} onClick={() => changeTab('data-gov')}>
          <span className="nav-icon">🗄️</span>
          <span>Data Governance & Privacy</span>
        </div>
        <div className={`nav-item ${activeTab === 'evidence' ? 'active' : ''}`} onClick={() => changeTab('evidence')}>
          <span className="nav-icon">📦</span>
          <span>Evidence Vault</span>
        </div>
        <div className={`nav-item ${activeTab === 'delegations' ? 'active' : ''}`} onClick={() => changeTab('delegations')}>
          <span className="nav-icon">👥</span>
          <span>Delegations of Authority</span>
        </div>

        <div className="sidebar-category">Accountability &amp; Config</div>
        <div className={`nav-item ${activeTab === 'whistleblower' ? 'active' : ''}`} onClick={() => changeTab('whistleblower')}>
          <span className="nav-icon">🔒</span>
          <span>Whistleblower Reports</span>
        </div>
        <div className={`nav-item ${activeTab === 'coi' ? 'active' : ''}`} onClick={() => changeTab('coi')}>
          <span className="nav-icon">⚖️</span>
          <span>Conflict of Interest</span>
        </div>
        <div className={`nav-item ${activeTab === 'admin-config' ? 'active' : ''}`} onClick={() => changeTab('admin-config')}>
          <span className="nav-icon">⚙️</span>
          <span>System Configuration</span>
        </div>

        <div className="sidebar-footer">
          <a href="http://localhost:3006" className="ecosystem-link">
            <span>🚀</span>
            <span>App Launcher</span>
          </a>
          <a href="http://localhost:3009" className="ecosystem-link">
            <span>💬</span>
            <span>StatChat Channels</span>
          </a>
          <a href="http://localhost:5000" className="ecosystem-link">
            <span>📊</span>
            <span>Analytics Hub</span>
          </a>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="main-wrapper">
        {/* Top Navigation Bar */}
        <header className="top-header">
          <div className="header-left">
            <h1 className="page-headline">Institutional Governance Platform</h1>
            <span className="tenant-badge">{tenantID}</span>
          </div>

          <div className="header-right">
            {/* Global Search Box */}
            <form onSubmit={handleGlobalSearch} className="search-input-box">
              <span>🔍</span>
              <input
                placeholder="Search policies, risks, controls, audits..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
              />
            </form>

            {/* Role Switcher for RBAC simulation */}
            <select
              style={{
                padding: '6px 10px',
                borderRadius: 8,
                border: '1px solid var(--border-light)',
                background: '#f8fafc',
                fontSize: 12,
                fontWeight: 600,
                color: 'var(--text-dark)'
              }}
              value={userRole}
              onChange={e => setUserRole(e.target.value)}
            >
              <option value="Governance Officer">Governance Officer</option>
              <option value="Risk Manager">Risk Manager</option>
              <option value="Compliance Officer">Compliance Officer</option>
              <option value="Chief Auditor">Chief Auditor</option>
              <option value="Executive Board">Executive Board</option>
              <option value="System Administrator">System Administrator</option>
            </select>

            {/* User Profile */}
            <div className="user-profile">
              <div className="user-avatar">
                {userRole.substring(0, 2).toUpperCase()}
              </div>
              <div className="user-meta">
                <span className="user-name">Dr. Sarah Nabatanzi</span>
                <span className="user-role">{userRole}</span>
              </div>
            </div>
          </div>
        </header>

        {/* Dynamic Content Body */}
        <main className="content-body">
          {activeTab === 'overview' && (
            <OverviewTab
              kpis={kpis}
              onNavigate={changeTab}
              onOpenNewRisk={() => changeTab('risks')}
              onOpenNewPolicy={() => changeTab('policies')}
            />
          )}

          {activeTab === 'ai-advisory' && (
            <AIGovernancePanel apiBase={apiBase} token={token} />
          )}

          {activeTab === 'policies' && (
            <PoliciesTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'sops' && (
            <SOPsTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'compliance' && (
            <ComplianceTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'risks' && (
            <RisksTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'controls' && (
            <ControlsTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'audits' && (
            <AuditsTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'findings' && (
            <FindingsTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'committees' && (
            <CommitteesTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'decisions' && (
            <DecisionsTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'data-gov' && (
            <DataGovernanceTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'evidence' && (
            <EvidenceTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'delegations' && (
            <DelegationsTab apiBase={apiBase} token={token} onRefreshDashboard={fetchDashboardData} />
          )}

          {activeTab === 'whistleblower' && (
            <WhistleblowerTab apiBase={apiBase} />
          )}

          {activeTab === 'coi' && (
            <ConflictOfInterestTab apiBase={apiBase} />
          )}

          {activeTab === 'admin-config' && (
            <AdminConfigTab apiBase={apiBase} />
          )}
        </main>
      </div>

      {/* Global Search Results Modal */}
      {showSearchModal && (
        <div className="modal-overlay">
          <div className="modal-card">
            <div className="modal-header">
              <h3 style={{ fontSize: 16, fontWeight: 700 }}>Search Results for "{searchQuery}"</h3>
              <button className="btn btn-secondary btn-sm" onClick={() => setShowSearchModal(false)}>✕</button>
            </div>
            <div className="modal-body">
              {searchResults.length === 0 ? (
                <div style={{ color: 'var(--text-muted)', textAlign: 'center', padding: 20 }}>
                  No governance records matching query.
                </div>
              ) : (
                searchResults.map((item, idx) => (
                  <div
                    key={idx}
                    onClick={() => {
                      setShowSearchModal(false);
                      if (item.type === 'policy') changeTab('policies');
                      else if (item.type === 'risk') changeTab('risks');
                      else if (item.type === 'control') changeTab('controls');
                      else if (item.type === 'finding') changeTab('findings');
                    }}
                    style={{
                      padding: 12,
                      borderBottom: '1px solid var(--border-light)',
                      cursor: 'pointer',
                      borderRadius: 6
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                      <span style={{ fontWeight: 700, fontSize: 13.5 }}>{item.title}</span>
                      <span className="badge badge-draft">{item.type}</span>
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{item.description}</div>
                  </div>
                ))
              )}
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowSearchModal(false)}>Close</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
