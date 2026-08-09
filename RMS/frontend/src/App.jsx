import React, { useState, useEffect } from 'react';
import './App.css';

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

export default function App() {
  const [currentView, setCurrentView] = useState('dashboard');
  const [selectedResearchId, setSelectedResearchId] = useState(null);
  const [researchList, setResearchList] = useState([]);
  const [dashboardData, setDashboardData] = useState(null);
  const [workspaceData, setWorkspaceData] = useState(null);
  const [activeTab, setActiveTab] = useState('overview');
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
        body: JSON.stringify({ stage, user: 'Admin User' })
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

  return (
    <div className="app-container">
      {/* Sidebar */}
      <aside className="sidebar">
        <div className="brand-section">
          <span className="brand-logo">StatGate RMS</span>
        </div>
        <nav className="sidebar-nav">
          <button
            className={`nav-item ${currentView === 'dashboard' ? 'active' : ''}`}
            onClick={() => setCurrentView('dashboard')}
          >
            Dashboard
          </button>
          <button
            className={`nav-item ${currentView === 'research-list' ? 'active' : ''}`}
            onClick={() => setCurrentView('research-list')}
          >
            Research Portfolio
          </button>
          {selectedResearchId && (
            <button
              className={`nav-item ${currentView === 'workspace' ? 'active' : ''}`}
              onClick={() => setCurrentView('workspace')}
            >
              Active Workspace
            </button>
          )}
        </nav>
      </aside>

      {/* Main Content Area */}
      <main className="main-content">
        {/* Top Header Bar */}
        <header className="top-bar">
          <h1 className="page-title">
            {currentView === 'dashboard' && 'Research Dashboard'}
            {currentView === 'research-list' && 'Research Studies Portfolio'}
            {currentView === 'workspace' && `Workspace: ${workspaceData?.research?.name || ''}`}
          </h1>
          <div className="top-bar-actions">
            <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
              + Start Research Study
            </button>
          </div>
        </header>

        {/* Dashboard View */}
        {currentView === 'dashboard' && dashboardData && (
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
        )}

        {/* Research List View */}
        {currentView === 'research-list' && (
          <div style={{ padding: '32px' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', background: '#fff', borderRadius: '8px', overflow: 'hidden' }}>
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
                {researchList.map(r => (
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
                ))}
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
              </div>
            </div>

            <div className="workspace-tabs">
              {[
                { key: 'overview', label: 'Overview' },
                { key: 'team', label: 'Team' },
                { key: 'proposals', label: 'Proposals' },
                { key: 'ethics', label: 'Ethics' },
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
            </div>
          </div>
        )}
      </main>

      {/* Create Research Modal */}
      {showCreateModal && (
        <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, background: 'rgba(0,0,0,0.5)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000 }}>
          <div className="glass-panel" style={{ background: '#fff', width: '500px', padding: '32px' }}>
            <h3>Start New Research Study</h3>
            <form onSubmit={handleCreateResearch} style={{ marginTop: '20px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
              <div>
                <label style={{ display: 'block', marginBottom: '6px', fontSize: '14px', fontWeight: '600' }}>Study Code</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. malaria-eval-2026"
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                  value={newResearch.code}
                  onChange={e => setNewResearch({ ...newResearch, code: e.target.value })}
                />
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: '6px', fontSize: '14px', fontWeight: '600' }}>Study Title</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Randomized Malaria Vaccine Impact Assessment"
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
                  value={newResearch.name}
                  onChange={e => setNewResearch({ ...newResearch, name: e.target.value })}
                />
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: '6px', fontSize: '14px', fontWeight: '600' }}>Description</label>
                <textarea
                  placeholder="Brief summary of research scope"
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)', minHeight: '80px' }}
                  value={newResearch.description}
                  onChange={e => setNewResearch({ ...newResearch, description: e.target.value })}
                />
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: '6px', fontSize: '14px', fontWeight: '600' }}>Principal Investigator</label>
                <input
                  type="text"
                  placeholder="e.g. Dr. Catherine Mitchell"
                  style={{ width: '100%', padding: '8px', borderRadius: '4px', border: '1px solid var(--border-light)' }}
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