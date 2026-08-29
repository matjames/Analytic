import React, { useState, useEffect } from 'react';
import './App.css';
import StatGateHeader from './components/StatGateHeader';

const API_URL = import.meta.env.VITE_RMS_API_URL || '';

// Tab Components
import OverviewTab from './components/OverviewTab';
import ProposalsTab from './components/ProposalsTab';
import EthicsTab from './components/EthicsTab';
import GrantsTab from './components/GrantsTab';
import LiteratureTab from './components/LiteratureTab';
import DatasetsTab from './components/DatasetsTab';
import PublicationsTab from './components/PublicationsTab';
import TasksTab from './components/TasksTab';
import MembersTab from './components/MembersTab';
import MeetingsTab from './components/MeetingsTab';
import RisksTab from './components/RisksTab';
import IssuesTab from './components/IssuesTab';
import DocumentsTab from './components/DocumentsTab';
import SurveysTab from './components/SurveysTab';
import ReportsTab from './components/ReportsTab';
import CalendarTab from './components/CalendarTab';
import ChatTab from './components/ChatTab';
import IRBCommitteesTab from './components/IRBCommitteesTab';
import DOITab from './components/DOITab';
import OpenScienceTab from './components/OpenScienceTab';

export default function App() {
  const [currentView, setCurrentView] = useState('dashboard');
  const [selectedResearchId, setSelectedResearchId] = useState(null);
  const [researchList, setResearchList] = useState([]);
  const [dashboardData, setDashboardData] = useState(null);
  const [workspaceData, setWorkspaceData] = useState(null);
  const [activeTab, setActiveTab] = useState('overview');
  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newResearch, setNewResearch] = useState({
    code: '',
    name: '',
    description: '',
    principalInvestigator: '',
    owner: 'Admin User',
    budgetTotal: 0,
    tags: []
  });

  useEffect(() => {
    fetchDashboard();
    fetchResearchList();
  }, []);

  const fetchDashboard = async () => {
    try {
      const res = await fetch(`${API_URL}/api/dashboard`);
      if (res.ok) {
        const data = await res.json();
        setDashboardData(data);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const fetchResearchList = async () => {
    try {
      const res = await fetch(`${API_URL}/api/research`);
      if (res.ok) {
        const data = await res.json();
        setResearchList(data);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const fetchWorkspace = async (id) => {
    try {
      const res = await fetch(`${API_URL}/api/research/${id}`);
      if (res.ok) {
        const data = await res.json();
        setWorkspaceData(data);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleSelectResearch = (id) => {
    setSelectedResearchId(id);
    fetchWorkspace(id);
    setCurrentView('workspace');
    setActiveTab('overview');
  };

  const handleCreateResearch = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${API_URL}/api/research`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newResearch)
      });
      if (res.ok) {
        const created = await res.json();
        setShowCreateModal(false);
        setNewResearch({
          code: '',
          name: '',
          description: '',
          principalInvestigator: '',
          owner: 'Admin User',
          budgetTotal: 0,
          tags: []
        });
        fetchResearchList();
        fetchDashboard();
        if (created?.id) {
          handleSelectResearch(created.id);
        }
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleStageTransition = async (stage) => {
    try {
      const res = await fetch(`${API_URL}/api/research/${selectedResearchId}/stage`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ stage, user: workspaceData?.research?.principalInvestigator || 'Admin User' })
      });
      if (res.ok) {
        fetchWorkspace(selectedResearchId);
        fetchDashboard();
      }
    } catch (e) {
      console.error(e);
    }
  };

  const refreshWorkspace = () => {
    if (selectedResearchId) {
      fetchWorkspace(selectedResearchId);
      fetchResearchList();
      fetchDashboard();
    }
  };

  // Filter research list based on top search bar query
  const filteredResearch = researchList.filter(r => {
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    return (
      (r.name && r.name.toLowerCase().includes(q)) ||
      (r.code && r.code.toLowerCase().includes(q)) ||
      (r.principalInvestigator && r.principalInvestigator.toLowerCase().includes(q)) ||
      (r.description && r.description.toLowerCase().includes(q)) ||
      (r.stage && r.stage.toLowerCase().includes(q)) ||
      (r.tags && r.tags.some(t => t.toLowerCase().includes(q)))
    );
  });

  return (
    <div className="app-container">
      {/* Sidebar Navigation */}
      <aside className="sidebar">
        <div className="brand-section">
          <span className="brand-logo">StatGate</span>
          <span className="brand-tag">RMS</span>
        </div>

        <nav className="nav-section">
          <span className="nav-title">Ecosystem Center</span>
          <div
            className={`nav-item ${currentView === 'dashboard' ? 'active' : ''}`}
            onClick={() => {
              setCurrentView('dashboard');
              setSelectedResearchId(null);
            }}
          >
            <span>🏛️</span> Global Dashboard
          </div>
          <div
            className={`nav-item ${currentView === 'research-list' ? 'active' : ''}`}
            onClick={() => {
              setCurrentView('research-list');
              setSelectedResearchId(null);
            }}
          >
            <span>🔬</span> Research Portfolio
          </div>

          <span className="nav-title">Active Studies</span>
          <div className="project-list-sidebar">
            {researchList.map(r => (
              <div
                key={r.id}
                className={`project-sidebar-item ${selectedResearchId === r.id && currentView === 'workspace' ? 'active' : ''}`}
                onClick={() => handleSelectResearch(r.id)}
              >
                <span style={{ fontWeight: '700' }}>{r.code}</span>
                <span style={{ fontSize: '11px', color: 'rgba(255,255,255,0.6)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {r.name}
                </span>
              </div>
            ))}
          </div>
        </nav>

        <div style={{ marginTop: 'auto', padding: '8px', borderTop: '1px solid rgba(255,255,255,0.1)', fontSize: '11px', color: 'rgba(255,255,255,0.5)' }}>
          StatGate RMS v2.0.0 — Production
        </div>
      </aside>

      {/* Main Panel Wrapper */}
      <div className="main-wrapper">
        <StatGateHeader
          searchValue={searchQuery}
          onSearchChange={setSearchQuery}
          userName={workspaceData?.research?.principalInvestigator || 'Dr. Catherine Mitchell'}
          userRole={selectedResearchId ? 'Principal Investigator' : 'Admin'}
        />

        <main className="content-body">
          {/* Top Title Action Row */}
          <div className="page-title-row">
            <div>
              <h1 className="page-title">
                {currentView === 'dashboard' && '🔬 Research Intelligence Dashboard'}
                {currentView === 'research-list' && '📚 Research Studies Portfolio'}
                {currentView === 'workspace' && `Workspace: ${workspaceData?.research?.name || ''}`}
              </h1>
            </div>
            <div>
              <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
                + Start Research Study
              </button>
            </div>
          </div>

          {/* Dashboard View */}
          {currentView === 'dashboard' && dashboardData && (
            <div>
              <div className="dashboard-grid">
                <div className="glass-panel stat-card">
                  <span className="stat-val">{dashboardData.totalResearch}</span>
                  <span className="stat-label">Total Research Projects</span>
                </div>
                <div className="glass-panel stat-card">
                  <span className="stat-val">{dashboardData.activeStudies}</span>
                  <span className="stat-label">Active Studies</span>
                </div>
                <div className="glass-panel stat-card">
                  <span className="stat-val">{dashboardData.proposalsPending}</span>
                  <span className="stat-label">Proposals Pending Approval</span>
                </div>
                <div className="glass-panel stat-card">
                  <span className="stat-val">{dashboardData.ethicsPending}</span>
                  <span className="stat-label">Ethics Applications Pending</span>
                </div>
                <div className="glass-panel stat-card">
                  <span className="stat-val">{dashboardData.totalPublications}</span>
                  <span className="stat-label">Total Publications</span>
                </div>
                <div className="glass-panel stat-card">
                  <span className="stat-val">{dashboardData.totalDatasets}</span>
                  <span className="stat-label">Total Datasets</span>
                </div>
                <div className="glass-panel stat-card">
                  <span className="stat-val">${dashboardData.totalBudget.toLocaleString()}</span>
                  <span className="stat-label">Total Allocated Funding</span>
                </div>
                <div className="glass-panel stat-card">
                  <span className="stat-val">{dashboardData.avgProgress.toFixed(1)}%</span>
                  <span className="stat-label">Average Lifecycle Progress</span>
                </div>
              </div>

              <div style={{ marginTop: '24px' }}>
                <h3 style={{ marginBottom: '16px', fontSize: '18px', fontWeight: '700' }}>Recent Research Studies</h3>
                <table style={{ width: '100%', borderCollapse: 'collapse', background: '#fff', borderRadius: '8px', overflow: 'hidden', boxShadow: 'var(--shadow-premium)' }}>
                  <thead>
                    <tr style={{ background: 'var(--primary-dark)', color: '#fff', textAlign: 'left' }}>
                      <th style={{ padding: '14px 16px' }}>Code</th>
                      <th style={{ padding: '14px 16px' }}>Study Title</th>
                      <th style={{ padding: '14px 16px' }}>Principal Investigator</th>
                      <th style={{ padding: '14px 16px' }}>Lifecycle Stage</th>
                      <th style={{ padding: '14px 16px' }}>Progress</th>
                      <th style={{ padding: '14px 16px' }}>Action</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredResearch.slice(0, 5).map(r => (
                      <tr key={r.id} style={{ borderBottom: '1px solid var(--border-light)' }}>
                        <td style={{ padding: '14px 16px', fontWeight: 'bold' }}>{r.code}</td>
                        <td style={{ padding: '14px 16px' }}>{r.name}</td>
                        <td style={{ padding: '14px 16px' }}>{r.principalInvestigator}</td>
                        <td style={{ padding: '14px 16px' }}>
                          <span style={{ padding: '3px 8px', borderRadius: '4px', background: '#e8f0fe', color: '#1a73e8', fontSize: '12px', fontWeight: '600' }}>
                            {r.stage}
                          </span>
                        </td>
                        <td style={{ padding: '14px 16px' }}>{r.progress.toFixed(0)}%</td>
                        <td style={{ padding: '14px 16px' }}>
                          <button className="btn btn-secondary" onClick={() => handleSelectResearch(r.id)}>
                            Open Workspace
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Research List View */}
          {currentView === 'research-list' && (
            <div className="glass-panel" style={{ padding: '0', overflow: 'hidden' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', background: '#fff' }}>
                <thead>
                  <tr style={{ background: 'var(--primary-dark)', color: '#fff', textAlign: 'left' }}>
                    <th style={{ padding: '16px' }}>Code</th>
                    <th style={{ padding: '16px' }}>Study Title</th>
                    <th style={{ padding: '16px' }}>Principal Investigator</th>
                    <th style={{ padding: '16px' }}>Lifecycle Stage</th>
                    <th style={{ padding: '16px' }}>Progress</th>
                    <th style={{ padding: '16px' }}>Action</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredResearch.length === 0 ? (
                    <tr>
                      <td colSpan="6" style={{ padding: '32px', textAlign: 'center', color: 'var(--text-secondary)' }}>
                        No research studies matching "{searchQuery}".
                      </td>
                    </tr>
                  ) : (
                    filteredResearch.map(r => (
                      <tr key={r.id} style={{ borderBottom: '1px solid var(--border-light)' }}>
                        <td style={{ padding: '16px', fontWeight: 'bold' }}>{r.code}</td>
                        <td style={{ padding: '16px' }}>{r.name}</td>
                        <td style={{ padding: '16px' }}>{r.principalInvestigator}</td>
                        <td style={{ padding: '16px' }}>
                          <span style={{ padding: '4px 8px', borderRadius: '4px', background: '#e8f0fe', color: '#1a73e8', fontSize: '12px', fontWeight: '600' }}>
                            {r.stage}
                          </span>
                        </td>
                        <td style={{ padding: '16px' }}>{r.progress.toFixed(0)}%</td>
                        <td style={{ padding: '16px' }}>
                          <button className="btn btn-secondary" onClick={() => handleSelectResearch(r.id)}>
                            Open Workspace
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}

          {/* Workspace View */}
          {currentView === 'workspace' && workspaceData && (
            <div>
              <div className="workspace-header">
                <h2>{workspaceData.research.name}</h2>
                <p style={{ marginTop: '8px', opacity: 0.9 }}>{workspaceData.research.description}</p>
                <div style={{ marginTop: '16px', display: 'flex', gap: '24px', fontSize: '14px' }}>
                  <span><strong>PI:</strong> {workspaceData.research.principalInvestigator}</span>
                  <span><strong>Stage:</strong> {workspaceData.research.stage}</span>
                  <span><strong>Progress:</strong> {workspaceData.research.progress.toFixed(0)}%</span>
                  <span><strong>Allocated Budget:</strong> ${workspaceData.research.budgetTotal.toLocaleString()}</span>
                </div>
              </div>

              <div className="workspace-tabs">
                {[
                  { key: 'overview', label: 'Overview' },
                  { key: 'team', label: 'Team' },
                  { key: 'proposals', label: 'Proposals' },
                  { key: 'ethics', label: 'Ethics & IRB' },
                  { key: 'grants', label: 'Funding' },
                  { key: 'literature', label: 'Literature' },
                  { key: 'datasets', label: 'Datasets' },
                  { key: 'publications', label: 'Publications' },
                  { key: 'tasks', label: 'Tasks' },
                  { key: 'meetings', label: 'Meetings' },
                  { key: 'risks', label: 'Risks' },
                  { key: 'issues', label: 'Issues' },
                  { key: 'documents', label: 'Documents' },
                  { key: 'surveys', label: 'Surveys' },
                  { key: 'reports', label: 'Reports' },
                  { key: 'calendar', label: 'Calendar' },
                  { key: 'chat', label: 'StatChat' },
                  { key: 'irb', label: 'IRB Committees' },
                  { key: 'doi', label: 'DOI Registry' },
                  { key: 'open-science', label: 'Open Science' },
                ].map(tab => (
                  <button
                    key={tab.key}
                    className={`tab-btn ${activeTab === tab.key ? 'active' : ''}`}
                    onClick={() => setActiveTab(tab.key)}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>

              <div className="tab-content">
                {activeTab === 'overview' && (
                  <OverviewTab workspaceData={workspaceData} onStageTransition={handleStageTransition} />
                )}
                {activeTab === 'team' && (
                  <MembersTab members={workspaceData.members} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'proposals' && (
                  <ProposalsTab proposals={workspaceData.proposals} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'ethics' && (
                  <EthicsTab ethics={workspaceData.ethics} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'grants' && (
                  <GrantsTab grants={workspaceData.grants} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'literature' && (
                  <LiteratureTab literature={workspaceData.literature} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'datasets' && (
                  <DatasetsTab datasets={workspaceData.datasets} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'publications' && (
                  <PublicationsTab publications={workspaceData.publications} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'tasks' && (
                  <TasksTab tasks={workspaceData.tasks} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'meetings' && (
                  <MeetingsTab meetings={workspaceData.meetings} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'risks' && (
                  <RisksTab risks={workspaceData.risks} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'issues' && (
                  <IssuesTab issues={workspaceData.issues} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'documents' && (
                  <DocumentsTab documents={workspaceData.documents} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'surveys' && (
                  <SurveysTab surveys={workspaceData.surveys} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'reports' && (
                  <ReportsTab reports={workspaceData.reports} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'calendar' && (
                  <CalendarTab events={workspaceData.calendarEvents} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'chat' && (
                  <ChatTab messages={workspaceData.chatMessages} researchId={selectedResearchId} onRefresh={refreshWorkspace} />
                )}
                {activeTab === 'irb' && (
                  <IRBCommitteesTab apiBase={API_URL} />
                )}
                {activeTab === 'doi' && (
                  <DOITab apiBase={API_URL} />
                )}
                {activeTab === 'open-science' && (
                  <OpenScienceTab apiBase={API_URL} />
                )}
              </div>
            </div>
          )}
        </main>
      </div>

      {/* Create Research Modal */}
      {showCreateModal && (
        <div className="modal-overlay">
          <div className="modal-content glass-panel" style={{ borderRadius: '12px' }}>
            <h3 style={{ fontSize: '20px', fontWeight: '800' }}>Initialize Research Study</h3>
            <p style={{ fontSize: '12px', color: 'var(--text-secondary)', marginTop: '-10px' }}>
              Provisions research protocols, ethics review trackers, funding logs, data catalogues, and Open Science DOIs.
            </p>
            <form onSubmit={handleCreateResearch} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <div className="form-group">
                <label>Study System Code</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. malaria-eval-2026"
                  value={newResearch.code}
                  onChange={e => setNewResearch({ ...newResearch, code: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label>Study Title</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Randomized Malaria Vaccine Impact Assessment"
                  value={newResearch.name}
                  onChange={e => setNewResearch({ ...newResearch, name: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label>Scope &amp; Description</label>
                <textarea
                  placeholder="Brief summary of research scope"
                  style={{ minHeight: '80px' }}
                  value={newResearch.description}
                  onChange={e => setNewResearch({ ...newResearch, description: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label>Principal Investigator</label>
                <input
                  type="text"
                  placeholder="e.g. Dr. Catherine Mitchell"
                  value={newResearch.principalInvestigator}
                  onChange={e => setNewResearch({ ...newResearch, principalInvestigator: e.target.value })}
                />
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px', marginTop: '10px' }}>
                <button type="button" className="btn btn-secondary" onClick={() => setShowCreateModal(false)}>Cancel</button>
                <button type="submit" className="btn btn-primary">Create Study</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
