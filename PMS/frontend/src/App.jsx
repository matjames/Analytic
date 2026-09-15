import React, { useState, useEffect } from 'react';
import './App.css';
import KanbanBoard from './components/KanbanBoard';
import GanttChart from './components/GanttChart';
import BudgetFinance from './components/BudgetFinance';
import RiskManagement from './components/RiskManagement';
import StatChatPanel from './components/StatChatPanel';
import StatGateHeader from './components/StatGateHeader';
import MeetingsTab from './components/MeetingsTab';
import DocumentsTab from './components/DocumentsTab';
import AnalyticsDashboard from './components/AnalyticsDashboard';
import TeamTab from './components/TeamTab';
import EnterpriseHierarchy from './components/EnterpriseHierarchy';
import IssuesTab from './components/IssuesTab';
import CalendarTab from './components/CalendarTab';
import AuditLogTab from './components/AuditLogTab';
import ReportsTab from './components/ReportsTab';
import FundingTab from './components/FundingTab';
import LessonsTab from './components/LessonsTab';
import LogFrameTab from './components/LogFrameTab';
import DonorsTab from './components/DonorsTab';
import PortfolioDashboard from './components/PortfolioDashboard';
import ProjectHealthAssistant from './components/ProjectHealthAssistant';
import ProjectLocationMap from './components/ProjectLocationMap';

// API Base URL - configured for the StatGate ecosystem
const API_BASE = import.meta.env.VITE_PMS_API_URL || 'http://localhost:8091';
const SPATIAL_API_BASE = import.meta.env.VITE_SPATIAL_API_URL || 'http://localhost:8108';

export default function App() {
  const [projects, setProjects] = useState([]);
  const [selectedProjectId, setSelectedProjectId] = useState(null);
  const [workspaceData, setWorkspaceData] = useState(null);
  const [portfolioDashboard, setPortfolioDashboard] = useState(null);
  const [portfolioFilter, setPortfolioFilter] = useState('');
  const [programmeFilter, setProgrammeFilter] = useState('');
  const [portfolioDashboardRefresh, setPortfolioDashboardRefresh] = useState(0);
  const [activeTab, setActiveTab] = useState('overview');
  const [searchQuery, setSearchQuery] = useState('');
  
  // New Project Form state
  const [isCreatingProj, setIsCreatingProj] = useState(false);
  const [projName, setProjName] = useState('');
  const [projCode, setProjCode] = useState('');
  const [projDesc, setProjDesc] = useState('');
  const [projOwner, setProjOwner] = useState('Dr. Sarah Jenkins');
  const [projOrg, setProjOrg] = useState('StatGate Ministry Alliance');
  const [projPortfolio, setProjPortfolio] = useState('Public Health Intelligence');
  const [projProg, setProjProg] = useState('National Surveys Programme');
  const [projBudget, setProjBudget] = useState(250000);
  const [projGeo, setProjGeo] = useState('National Coverage');

  // Load projects
  const loadProjects = () => {
    fetch(`${API_BASE}/api/projects`)
      .then(res => res.json())
      .then(data => setProjects(data))
      .catch(err => console.error("Error fetching projects:", err));
  };

  const loadPortfolioDashboard = () => {
    setPortfolioDashboardRefresh(value => value + 1);
  };

  useEffect(() => {
    const params = new URLSearchParams();
    if (portfolioFilter) params.set('portfolio', portfolioFilter);
    if (programmeFilter) params.set('programme', programmeFilter);
    fetch(`${API_BASE}/api/portfolio-dashboard${params.toString() ? `?${params.toString()}` : ''}`)
      .then(res => {
        if (!res.ok) throw new Error(`Portfolio dashboard request failed: ${res.status}`);
        return res.json();
      })
      .then(data => setPortfolioDashboard(data))
      .catch(err => console.error('Error fetching portfolio dashboard:', err));
  }, [portfolioFilter, programmeFilter, portfolioDashboardRefresh]);

  // Load active workspace details
  const loadWorkspace = (id) => {
    fetch(`${API_BASE}/api/projects/${id}`)
      .then(res => res.json())
      .then(data => setWorkspaceData(data))
      .catch(err => console.error("Error fetching workspace details:", err));
  };

  useEffect(() => {
    loadProjects();
  }, []);

  useEffect(() => {
    if (selectedProjectId) {
      loadWorkspace(selectedProjectId);
    } else {
      setWorkspaceData(null);
    }
  }, [selectedProjectId]);

  const handleCreateProject = (e) => {
    e.preventDefault();
    const payload = {
      name: projName,
      code: projCode,
      description: projDesc,
      owner: projOwner,
      org: projOrg,
      portfolio: projPortfolio,
      programme: projProg,
      startDate: new Date().toISOString().split('T')[0],
      endDate: new Date(Date.now() + 180*24*60*60*1000).toISOString().split('T')[0],
      targetGeo: projGeo,
      budgetTotal: Number(projBudget)
    };

    fetch(`${API_BASE}/api/projects`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
      .then(res => res.json())
      .then(newProj => {
        loadProjects();
        loadPortfolioDashboard();
        setIsCreatingProj(false);
        // Automatically switch to the newly created project workspace
        setSelectedProjectId(newProj.id);
        setActiveTab('overview');
      })
      .catch(err => console.error("Error creating project:", err));
  };

  const handleStageTransition = (targetStage) => {
    fetch(`${API_BASE}/api/projects/${selectedProjectId}/stage`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ stage: targetStage, user: workspaceData?.project?.owner || 'System' })
    })
      .then(res => res.json())
      .then(() => {
        loadProjects();
        loadPortfolioDashboard();
        loadWorkspace(selectedProjectId);
      })
      .catch(err => console.error("Error transitioning stage:", err));
  };

  const handleSendMessage = (chatPayload) => {
    fetch(`${API_BASE}/api/chats`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(chatPayload)
    })
      .then(res => res.json())
      .then(() => {
        loadWorkspace(selectedProjectId);
      })
      .catch(err => console.error("Error sending message:", err));
  };

  // Search filter
  const filteredProjects = projects.filter(p => {
    const query = searchQuery.toLowerCase();
    return (
      p.name.toLowerCase().includes(query) ||
      p.code.toLowerCase().includes(query) ||
      p.programme.toLowerCase().includes(query) ||
      p.owner.toLowerCase().includes(query) ||
      p.org.toLowerCase().includes(query) ||
      p.portfolio.toLowerCase().includes(query) ||
      p.targetGeo.toLowerCase().includes(query)
    );
  });

  // Calculate high-level stats
  const totalAllocated = projects.reduce((acc, curr) => acc + curr.budgetTotal, 0);
  const totalSpent = projects.reduce((acc, curr) => acc + curr.spentTotal, 0);

  const stages = [
    'Concept', 'Proposal', 'Planning', 'Approval', 'Funding',
    'Implementation', 'Monitoring', 'Evaluation', 'Closure', 'Archive'
  ];

  return (
    <div className="app-container">
      {/* Sidebar Navigation */}
      <aside className="sidebar">
        <div className="brand-section">
          <span className="brand-logo">StatGate</span>
          <span className="brand-tag">PMS</span>
        </div>

        <nav className="nav-section">
          <span className="nav-title">Ecosystem Center</span>
          <div className={`nav-item ${selectedProjectId === null ? 'active' : ''}`} onClick={() => setSelectedProjectId(null)}>
            <span>🏛️</span> Global Dashboard
          </div>

          <span className="nav-title">Active Workspaces</span>
          <div className="project-list-sidebar">
            {projects.map(p => (
              <div 
                key={p.id} 
                className={`project-sidebar-item ${selectedProjectId === p.id ? 'active' : ''}`}
                onClick={() => {
                  setSelectedProjectId(p.id);
                  setActiveTab('overview');
                }}
              >
                <span style={{ fontWeight: '700' }}>{p.code}</span>
                <span style={{ fontSize: '11px', color: 'rgba(255,255,255,0.5)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {p.name}
                </span>
              </div>
            ))}
          </div>
        </nav>

        <div style={{ marginTop: 'auto', padding: '8px', borderTop: '1px solid rgba(255,255,255,0.1)', fontSize: '11px', color: 'rgba(255,255,255,0.5)' }}>
          StatGate PMS v2.0.0 — Production
        </div>
      </aside>

      {/* Main Panel Wrapper */}
      <div className="main-wrapper">
        <StatGateHeader
          searchValue={searchQuery}
          onSearchChange={setSearchQuery}
          userName={workspaceData?.project?.owner || 'Sarah Jenkins'}
          userRole={selectedProjectId ? 'Project Manager' : 'Admin'}
        />
        <main className="content-body">
          {/* Create Project Modal Overlay */}
          {isCreatingProj && (
            <div className="modal-overlay">
              <form className="modal-content glass-panel" onSubmit={handleCreateProject}>
                <h2 style={{ fontSize: '20px', fontWeight: '800' }}>Initialize Enterprise Workspace</h2>
                <p style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
                  This workflow automatically provisions standard dashboards, chat channels, initial work breakdown structures, calendars, and risk logs.
                </p>

                <div className="form-row">
                  <div className="form-group">
                    <label>Project Name</label>
                    <input type="text" value={projName} onChange={e => setProjName(e.target.value)} placeholder="e.g. Household Census" required />
                  </div>
                  <div className="form-group">
                    <label>System Code</label>
                    <input type="text" value={projCode} onChange={e => setProjCode(e.target.value)} placeholder="SG-2026-HC" required />
                  </div>
                </div>

                <div className="form-group">
                  <label>Description & Scope</label>
                  <textarea rows={3} value={projDesc} onChange={e => setProjDesc(e.target.value)} placeholder="Enter details..." required />
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Portfolio Area</label>
                    <input type="text" value={projPortfolio} onChange={e => setProjPortfolio(e.target.value)} required />
                  </div>
                  <div className="form-group">
                    <label>Parent Programme</label>
                    <input type="text" value={projProg} onChange={e => setProjProg(e.target.value)} required />
                  </div>
                </div>

                <div className="form-row">
                  <div className="form-group">
                    <label>Total Budget Allocation ($)</label>
                    <input type="number" value={projBudget} onChange={e => setProjBudget(e.target.value)} required />
                  </div>
                  <div className="form-group">
                    <label>Target Geographic Region</label>
                    <input type="text" value={projGeo} onChange={e => setProjGeo(e.target.value)} required />
                  </div>
                </div>

                <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end', marginTop: '12px' }}>
                  <button type="button" className="btn btn-secondary" onClick={() => setIsCreatingProj(false)}>Cancel</button>
                  <button type="submit" className="btn btn-primary">Initialize Workspace</button>
                </div>
              </form>
            </div>
          )}

          {/* MAIN VIEW: GLOBAL DASHBOARD */}
          {selectedProjectId === null ? (
            <div>
              <div className="page-title-row">
                <div>
                  <h1 className="page-title">Enterprise Intelligence & Portfolios</h1>
                  <span className="page-subtitle">StatGate operational heartbeat & project registries</span>
                </div>
                <button className="btn btn-primary" onClick={() => setIsCreatingProj(true)}>+ Initialize Project</button>
              </div>

              {/* Statistics Grid */}
              <PortfolioDashboard
                data={portfolioDashboard}
                portfolio={portfolioFilter}
                programme={programmeFilter}
                onPortfolioChange={value => {
                  setPortfolioFilter(value);
                  setProgrammeFilter('');
                }}
                onProgrammeChange={setProgrammeFilter}
              />

              {/* Statistics Grid */}
              <div className="stat-grid">
                <div className="glass-panel stat-card">
                  <span className="stat-label">Active Portfolios</span>
                  <span className="stat-value">{portfolioDashboard?.summary?.projectCount ?? projects.length}</span>
                  <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Public Health, Economic & Resource</span>
                </div>

                <div className="glass-panel stat-card">
                  <span className="stat-label">Total Managed Budgets</span>
                  <span className="stat-value" style={{ color: 'var(--primary-color)' }}>
                    ${(portfolioDashboard?.summary?.budgetTotal ?? totalAllocated).toLocaleString()}
                  </span>
                  <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>Aggregate program allocations</span>
                </div>

                <div className="glass-panel stat-card">
                  <span className="stat-label">System Progress (Spent)</span>
                  <span className="stat-value" style={{ color: 'var(--success-color)' }}>
                    ${(portfolioDashboard?.summary?.spentTotal ?? totalSpent).toLocaleString()}
                  </span>
                  <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{(((portfolioDashboard?.summary?.spentTotal ?? totalSpent) / (portfolioDashboard?.summary?.budgetTotal ?? totalAllocated)) * 100 || 0).toFixed(1)}% total system utilization</span>
                </div>
              </div>

              {/* Projects Registry Panel */}
              <div className="glass-panel" style={{ padding: '24px' }}>
                <h3 style={{ fontSize: '18px', fontWeight: '800', marginBottom: '20px' }}>Active Projects Registry</h3>
                <div className="projects-grid">
                  {filteredProjects.map(p => (
                    <div key={p.id} className="glass-panel project-panel-card" onClick={() => setSelectedProjectId(p.id)}>
                      <div className="project-header-row">
                        <div>
                          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                            <span style={{ fontWeight: '800', fontSize: '16px', color: 'var(--text-primary)' }}>{p.name}</span>
                            <span className="badge" style={{ background: 'rgba(22,92,146,0.1)' }}>{p.code}</span>
                          </div>
                          <span style={{ fontSize: '12px', color: 'var(--text-secondary)', display: 'block', marginTop: '4px' }}>
                            Programme: <strong>{p.programme}</strong> | PM: <strong>{p.owner}</strong>
                          </span>
                        </div>
                        <span className={`badge badge-${p.stage}`}>{p.stage}</span>
                      </div>

                      <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>{p.description}</p>

                      <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px' }}>
                          <span>Target: {p.targetGeo}</span>
                          <span>Progress: {p.progress}%</span>
                        </div>
                        <div className="project-progress-bar">
                          <div className="project-progress-fill" style={{ width: `${p.progress}%` }}></div>
                        </div>
                      </div>

                      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '16px', borderTop: '1px solid var(--border-light)', paddingTop: '12px', fontSize: '12px', color: 'var(--text-muted)' }}>
                        <span>💰 Budget: <strong>${p.budgetTotal.toLocaleString()}</strong></span>
                        <span>⚠️ Risks: <strong style={{ color: p.risksCount > 0 ? 'var(--accent-warning)' : '' }}>{p.risksCount}</strong></span>
                        <span>🎯 Issues: <strong style={{ color: p.issuesCount > 0 ? 'var(--accent-danger)' : '' }}>{p.issuesCount}</strong></span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            /* DETAILED VIEW: INDIVIDUAL PROJECT WORKSPACE */
            workspaceData && (
              <div>
                {/* Title Section */}
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                  <div>
                    <button className="btn btn-secondary" onClick={() => setSelectedProjectId(null)} style={{ padding: '6px 12px', marginBottom: '12px', fontSize: '11px' }}>
                      ← Back to Portfolios
                    </button>
                    <h1 className="page-title" style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      {workspaceData.project.name}
                      <span className="badge" style={{ background: 'rgba(22,92,146,0.1)' }}>{workspaceData.project.code}</span>
                    </h1>
                    <span style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>
                      Portfolio: <strong>{workspaceData.project.portfolio}</strong> | Owner: <strong>{workspaceData.project.owner}</strong>
                    </span>
                  </div>
                  <span className={`badge badge-${workspaceData.project.stage}`} style={{ padding: '8px 16px', fontSize: '13px' }}>
                    Stage: {workspaceData.project.stage}
                  </span>
                </div>

                {/* Lifecycle Stepper / State Machine */}
                <div className="glass-panel stepper">
                  {stages.map((stg, idx) => {
                    const currentIdx = stages.indexOf(workspaceData.project.stage);
                    const isActive = workspaceData.project.stage === stg;
                    const isCompleted = currentIdx > idx;

                    return (
                      <div 
                        key={stg} 
                        className={`step-node ${isActive ? 'active' : ''} ${isCompleted ? 'completed' : ''}`}
                        onClick={() => handleStageTransition(stg)}
                      >
                        <div className="step-dot">{isCompleted ? '✓' : idx + 1}</div>
                        <span className="step-label">{stg}</span>
                      </div>
                    );
                  })}
                </div>

                {/* Workspace Module Tabs */}
                <div className="workspace-tabs">
                  <div className={`workspace-tab ${activeTab === 'overview' ? 'active' : ''}`} onClick={() => setActiveTab('overview')}>Overview</div>
                  <div className={`workspace-tab ${activeTab === 'hierarchy' ? 'active' : ''}`} onClick={() => setActiveTab('hierarchy')}>Hierarchy</div>
                  <div className={`workspace-tab ${activeTab === 'tasks' ? 'active' : ''}`} onClick={() => setActiveTab('tasks')}>Work Planning</div>
                  <div className={`workspace-tab ${activeTab === 'budgets' ? 'active' : ''}`} onClick={() => setActiveTab('budgets')}>Budget & Finance</div>
                  <div className={`workspace-tab ${activeTab === 'funding' ? 'active' : ''}`} onClick={() => setActiveTab('funding')}>Funding</div>
                  <div className={`workspace-tab ${activeTab === 'risks' ? 'active' : ''}`} onClick={() => setActiveTab('risks')}>Risks</div>
                  <div className={`workspace-tab ${activeTab === 'issues' ? 'active' : ''}`} onClick={() => setActiveTab('issues')}>Issues</div>
                  <div className={`workspace-tab ${activeTab === 'analytics' ? 'active' : ''}`} onClick={() => setActiveTab('analytics')}>Analytics</div>
                  <div className={`workspace-tab ${activeTab === 'team' ? 'active' : ''}`} onClick={() => setActiveTab('team')}>Team</div>
                  <div className={`workspace-tab ${activeTab === 'documents' ? 'active' : ''}`} onClick={() => setActiveTab('documents')}>Documents</div>
                  <div className={`workspace-tab ${activeTab === 'meetings' ? 'active' : ''}`} onClick={() => setActiveTab('meetings')}>Meetings</div>
                  <div className={`workspace-tab ${activeTab === 'calendar' ? 'active' : ''}`} onClick={() => setActiveTab('calendar')}>Calendar</div>
                  <div className={`workspace-tab ${activeTab === 'lessons' ? 'active' : ''}`} onClick={() => setActiveTab('lessons')}>Lessons</div>
                  <div className={`workspace-tab ${activeTab === 'reports' ? 'active' : ''}`} onClick={() => setActiveTab('reports')}>Reports</div>
                  <div className={`workspace-tab ${activeTab === 'audit' ? 'active' : ''}`} onClick={() => setActiveTab('audit')}>Audit Log</div>
                  <div className={`workspace-tab ${activeTab === 'helpdesk' ? 'active' : ''}`} onClick={() => setActiveTab('helpdesk')}>HelpDesk</div>
                  <div className={`workspace-tab ${activeTab === 'chat' ? 'active' : ''}`} onClick={() => setActiveTab('chat')}>StatChat</div>
                  <div className={`workspace-tab ${activeTab === 'surveys' ? 'active' : ''}`} onClick={() => setActiveTab('surveys')}>Surveys</div>
                  <div className={`workspace-tab ${activeTab === 'logframe' ? 'active' : ''}`} onClick={() => setActiveTab('logframe')}>M&amp;E LogFrame</div>
                  <div className={`workspace-tab ${activeTab === 'donors' ? 'active' : ''}`} onClick={() => setActiveTab('donors')}>Donors</div>
                </div>

                {/* Tab content rendering */}
                {activeTab === 'overview' && (
                  <>
                  <ProjectHealthAssistant project={workspaceData.project} apiBase={API_BASE} />
                  <ProjectLocationMap project={workspaceData.project} apiBase={SPATIAL_API_BASE} />
                  <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '24px' }}>
                    {/* Main column */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                      {/* Overview Summary */}
                      <div className="glass-panel" style={{ padding: '24px' }}>
                        <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '12px' }}>Workspace Mission & Bounds</h3>
                        <p style={{ fontSize: '14px', color: 'var(--text-secondary)', lineHeight: '1.6' }}>{workspaceData.project.description}</p>
                        
                        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginTop: '20px', borderTop: '1px solid var(--border-light)', paddingTop: '16px' }}>
                          <div>
                            <span style={{ display: 'block', fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Target Geographic Region</span>
                            <span style={{ fontSize: '14px', fontWeight: '600' }}>🌍 {workspaceData.project.targetGeo}</span>
                          </div>
                          <div>
                            <span style={{ display: 'block', fontSize: '11px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>Project Duration</span>
                            <span style={{ fontSize: '14px', fontWeight: '600' }}>📅 {workspaceData.project.startDate} to {workspaceData.project.endDate}</span>
                          </div>
                        </div>

                        {/* Quick stats */}
                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '12px', marginTop: '20px', borderTop: '1px solid var(--border-light)', paddingTop: '16px' }}>
                          {[
                            { label: 'Tasks', value: workspaceData.tasks?.length || 0, icon: '📋' },
                            { label: 'Team', value: workspaceData.members?.length || 0, icon: '👥' },
                            { label: 'Milestones', value: workspaceData.milestones?.length || 0, icon: '🎯' },
                            { label: 'Deliverables', value: workspaceData.deliverables?.length || 0, icon: '📦' },
                          ].map(stat => (
                            <div key={stat.label} style={{ textAlign: 'center', padding: '12px', background: 'rgba(22,92,146,0.05)', borderRadius: '8px' }}>
                              <div style={{ fontSize: '20px' }}>{stat.icon}</div>
                              <div style={{ fontSize: '20px', fontWeight: '800', color: 'var(--primary-color)' }}>{stat.value}</div>
                              <div style={{ fontSize: '10px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>{stat.label}</div>
                            </div>
                          ))}
                        </div>
                      </div>

                      {/* Work planning summary */}
                      <GanttChart tasks={workspaceData.tasks} />
                    </div>

                    {/* Sidebar components */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                      {/* Team Members list */}
                      <div className="glass-panel" style={{ padding: '20px' }}>
                        <h3 style={{ fontSize: '15px', fontWeight: '750', marginBottom: '12px' }}>Team & Staff Assignments</h3>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                          {workspaceData.members && workspaceData.members.map(member => (
                            <div key={member.id} style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                              <img src={member.avatarUrl} alt="avatar" style={{ width: '28px', height: '28px', borderRadius: '50%', background: 'rgba(22,92,146,0.1)' }} />
                              <div style={{ display: 'flex', flexDirection: 'column' }}>
                                <span style={{ fontSize: '13px', fontWeight: '600' }}>{member.name}</span>
                                <span style={{ fontSize: '11px', color: 'var(--text-muted)' }}>{member.role}</span>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>

                      {/* Milestones */}
                      <div className="glass-panel" style={{ padding: '20px' }}>
                        <h3 style={{ fontSize: '15px', fontWeight: '750', marginBottom: '12px' }}>Key Milestones</h3>
                        {workspaceData.milestones && workspaceData.milestones.length > 0 ? (
                          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                            {workspaceData.milestones.slice(0, 5).map(ms => (
                              <div key={ms.id} style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '12px' }}>
                                <span>{ms.status === 'Completed' ? '✅' : ms.status === 'In Progress' ? '🔄' : '⏳'}</span>
                                <span style={{ flex: 1 }}>{ms.name}</span>
                                <span style={{ color: 'var(--text-muted)', fontSize: '10px' }}>{ms.dueDate}</span>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>No milestones defined yet.</div>
                        )}
                      </div>

                      {/* Helpdesk incidents ticker */}
                      <div className="glass-panel" style={{ padding: '20px' }}>
                        <h3 style={{ fontSize: '15px', fontWeight: '750', marginBottom: '12px' }}>Helpdesk Technical Tickets</h3>
                        {workspaceData.helpdesk && workspaceData.helpdesk.length === 0 ? (
                          <div style={{ fontSize: '12px', color: 'var(--text-muted)' }}>No open incidents linked to this project.</div>
                        ) : (
                          workspaceData.helpdesk && workspaceData.helpdesk.map(ticket => (
                            <div key={ticket.id} style={{ borderLeft: '3px solid var(--accent-danger)', paddingLeft: '10px', display: 'flex', flexDirection: 'column', gap: '4px' }}>
                              <span style={{ fontSize: '12px', fontWeight: '700' }}>{ticket.title}</span>
                              <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{ticket.description}</span>
                              <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>Status: {ticket.status} | Priority: {ticket.priority}</span>
                            </div>
                          ))
                        )}
                      </div>
                    </div>
                  </div>
                  </>
                )}

                {activeTab === 'hierarchy' && (
                  <EnterpriseHierarchy 
                    components={workspaceData.components}
                    activities={workspaceData.activities}
                    deliverables={workspaceData.deliverables}
                    milestones={workspaceData.milestones}
                    projectId={selectedProjectId}
                    onRefresh={() => loadWorkspace(selectedProjectId)}
                  />
                )}

                {activeTab === 'tasks' && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '28px' }}>
                    <KanbanBoard 
                      tasks={workspaceData.tasks} 
                      onTaskUpdate={() => loadWorkspace(selectedProjectId)} 
                      projectId={selectedProjectId}
                    />
                    <GanttChart tasks={workspaceData.tasks} />
                  </div>
                )}

                {activeTab === 'budgets' && (
                  <BudgetFinance 
                    budgetTotal={workspaceData.project.budgetTotal} 
                    spentTotal={workspaceData.project.spentTotal} 
                    budgetLines={workspaceData.budgets}
                    costCentres={workspaceData.costCentres}
                    budgetRevisions={workspaceData.budgetRevisions}
                    procurementRefs={workspaceData.procurementRefs}
                  />
                )}

                {activeTab === 'funding' && (
                  <FundingTab 
                    fundingSources={workspaceData.fundingSources}
                    projectId={selectedProjectId}
                    onRefresh={() => loadWorkspace(selectedProjectId)}
                  />
                )}

                {activeTab === 'risks' && (
                  <RiskManagement 
                    risks={workspaceData.risks} 
                    onRiskAdded={() => loadWorkspace(selectedProjectId)} 
                    projectId={selectedProjectId}
                  />
                )}

                {activeTab === 'issues' && (
                  <IssuesTab 
                    issues={workspaceData.issues}
                    assumptions={workspaceData.assumptions}
                    correctiveActions={workspaceData.correctiveActions}
                    projectId={selectedProjectId}
                    onRefresh={() => loadWorkspace(selectedProjectId)}
                  />
                )}

                {activeTab === 'analytics' && (
                  <AnalyticsDashboard
                    project={workspaceData.project}
                    tasks={workspaceData.tasks}
                    risks={workspaceData.risks}
                    surveys={workspaceData.surveys}
                    members={workspaceData.members}
                    milestones={workspaceData.milestones}
                  />
                )}

                {activeTab === 'team' && (
                  <TeamTab members={workspaceData.members} />
                )}

                {activeTab === 'documents' && (
                  <DocumentsTab documents={workspaceData.documents} />
                )}

                {activeTab === 'meetings' && (
                  <MeetingsTab meetings={workspaceData.meetings} />
                )}

                {activeTab === 'calendar' && (
                  <CalendarTab 
                    events={workspaceData.calendarEvents}
                    meetings={workspaceData.meetings}
                    milestones={workspaceData.milestones}
                  />
                )}

                {activeTab === 'lessons' && (
                  <LessonsTab 
                    lessons={workspaceData.lessons}
                    projectId={selectedProjectId}
                    onRefresh={() => loadWorkspace(selectedProjectId)}
                  />
                )}

                {activeTab === 'reports' && (
                  <ReportsTab 
                    reports={workspaceData.reports}
                    projectId={selectedProjectId}
                    onRefresh={() => loadWorkspace(selectedProjectId)}
                  />
                )}

                {activeTab === 'audit' && (
                  <AuditLogTab logs={workspaceData.auditLogs} />
                )}

                {activeTab === 'helpdesk' && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <div>
                        <h2 style={{ fontSize: '20px', fontWeight: '800', marginBottom: '4px' }}>🎫 HelpDesk Tickets</h2>
                        <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>Support incidents, technical issues, escalations, and resolution history linked to this project</p>
                      </div>
                      <a href="http://localhost:3005" target="_blank" rel="noopener noreferrer" className="btn btn-primary" style={{ fontSize: '12px', padding: '8px 16px' }}>
                        Open HelpDesk ↗
                      </a>
                    </div>
                    {/* Summary stats */}
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
                      {[
                        { label: 'Open Tickets', value: (workspaceData.helpdesk || []).filter(t => t.status !== 'Resolved').length, icon: '🔴', color: 'var(--accent-danger)' },
                        { label: 'In Progress', value: (workspaceData.helpdesk || []).filter(t => t.status === 'In Progress').length, icon: '🟡', color: 'var(--accent-warning)' },
                        { label: 'Resolved', value: (workspaceData.helpdesk || []).filter(t => t.status === 'Resolved').length, icon: '🟢', color: 'var(--accent-success)' },
                        { label: 'Total', value: (workspaceData.helpdesk || []).length, icon: '📋', color: 'var(--primary-color)' },
                      ].map(stat => (
                        <div key={stat.label} className="glass-panel" style={{ padding: '16px', display: 'flex', alignItems: 'center', gap: '12px' }}>
                          <span style={{ fontSize: '24px' }}>{stat.icon}</span>
                          <div>
                            <div style={{ fontSize: '24px', fontWeight: '800', color: stat.color }}>{stat.value}</div>
                            <div style={{ fontSize: '10px', color: 'var(--text-muted)', textTransform: 'uppercase' }}>{stat.label}</div>
                          </div>
                        </div>
                      ))}
                    </div>
                    {/* Ticket list */}
                    {workspaceData.helpdesk && workspaceData.helpdesk.length > 0 ? (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                        {workspaceData.helpdesk.map(ticket => (
                          <div key={ticket.id} className="glass-panel" style={{ padding: '16px', display: 'flex', gap: '14px', alignItems: 'flex-start' }}>
                            <div style={{ width: '4px', alignSelf: 'stretch', borderRadius: '4px', background: ticket.priority === 'High' ? 'var(--accent-danger)' : ticket.priority === 'Medium' ? 'var(--accent-warning)' : 'var(--accent-success)', flexShrink: 0 }} />
                            <div style={{ flex: 1 }}>
                              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '4px' }}>
                                <span style={{ fontSize: '13px', fontWeight: '700' }}>{ticket.title}</span>
                                <div style={{ display: 'flex', gap: '6px' }}>
                                  <span className="badge" style={{ fontSize: '10px', background: ticket.priority === 'High' ? 'rgba(239,68,68,0.15)' : 'rgba(245,158,11,0.15)', color: ticket.priority === 'High' ? '#f87171' : '#fbbf24' }}>{ticket.priority}</span>
                                  <span className="badge" style={{ fontSize: '10px', background: 'rgba(22,92,146,0.1)', color: 'var(--primary-color)' }}>{ticket.status}</span>
                                </div>
                              </div>
                              <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>{ticket.description}</div>
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="glass-panel" style={{ padding: '40px', textAlign: 'center', color: 'var(--text-muted)' }}>
                        <div style={{ fontSize: '36px', marginBottom: '10px' }}>✅</div>
                        <div style={{ fontSize: '15px', fontWeight: '700' }}>No open incidents</div>
                        <div style={{ fontSize: '12px', marginTop: '4px' }}>All technical support issues for this project are resolved.</div>
                      </div>
                    )}
                  </div>
                )}

                {activeTab === 'chat' && (
                  <StatChatPanel 
                    chats={workspaceData.chats} 
                    onSendMessage={handleSendMessage} 
                    projectId={selectedProjectId}
                  />
                )}

                {activeTab === 'surveys' && (
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px' }}>
                    {/* StatCollect Forms */}
                    <div className="glass-panel" style={{ padding: '24px' }}>
                      <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>StatCollect Survey Instruments</h3>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                        {workspaceData.surveys && workspaceData.surveys.map(survey => (
                          <div key={survey.id} style={{ display: 'flex', flexDirection: 'column', gap: '8px', borderBottom: '1px solid var(--border-light)', paddingBottom: '16px' }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                              <span style={{ fontSize: '14px', fontWeight: '700' }}>{survey.name}</span>
                              <span className="badge" style={{ background: 'rgba(22,92,146,0.1)', color: 'var(--primary-color)' }}>{survey.status}</span>
                            </div>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px', color: 'var(--text-secondary)' }}>
                              <span>Target Sample: {survey.targetSample}</span>
                              <span>Submissions: {survey.submissions}</span>
                            </div>
                            <div className="project-progress-bar" style={{ height: '4px' }}>
                              <div className="project-progress-fill" style={{ width: `${survey.progress}%` }} />
                            </div>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '11px', color: 'var(--text-muted)' }}>
                              <span>{survey.progress.toFixed(1)}% of Target Sample verified</span>
                              <a href="http://localhost:5174" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--primary-color)', textDecoration: 'none' }}>Open in StatCollect →</a>
                            </div>
                          </div>
                        ))}
                      </div>
                      <div style={{ marginTop: '16px', borderTop: '1px solid var(--border-light)', paddingTop: '12px' }}>
                        <a href="http://localhost:5174" target="_blank" rel="noopener noreferrer" className="btn btn-secondary" style={{ fontSize: '11px', display: 'block', textAlign: 'center' }}>
                          Manage All Surveys in StatCollect ↗
                        </a>
                      </div>
                    </div>

                    {/* Approved Datasets */}
                    <div className="glass-panel" style={{ padding: '24px' }}>
                      <h3 style={{ fontSize: '16px', fontWeight: '750', marginBottom: '16px' }}>Approved Datasets & Field Status</h3>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                        {workspaceData.surveys && workspaceData.surveys.map(survey => {
                          const approvedPct = survey.progress;
                          return (
                            <div key={survey.id + '-ds'} style={{ padding: '14px', background: 'rgba(16,185,129,0.05)', borderRadius: '8px', border: '1px solid rgba(16,185,129,0.1)' }}>
                              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
                                <span style={{ fontSize: '13px', fontWeight: '700' }}>{survey.name} — Dataset</span>
                                <span className="badge" style={{ background: 'rgba(16,185,129,0.1)', color: 'var(--accent-success)', fontSize: '10px' }}>
                                  {approvedPct >= 100 ? 'Approved' : approvedPct >= 50 ? 'Partial' : 'Pending'}
                                </span>
                              </div>
                              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '8px', fontSize: '11px', color: 'var(--text-secondary)' }}>
                                <span>📥 {survey.submissions} records</span>
                                <span>✅ {Math.round(survey.submissions * (approvedPct / 100))} approved</span>
                                <span>📊 {approvedPct.toFixed(0)}% complete</span>
                              </div>
                            </div>
                          );
                        })}
                        {(!workspaceData.surveys || workspaceData.surveys.length === 0) && (
                          <div style={{ fontSize: '12px', color: 'var(--text-muted)', padding: '20px', textAlign: 'center' }}>No survey datasets available yet.</div>
                        )}
                      </div>
                    </div>
                  </div>
                )}

                {activeTab === 'logframe' && (
                  <LogFrameTab projectId={selectedProjectId} apiBase={API_BASE} />
                )}

                {activeTab === 'donors' && (
                  <DonorsTab apiBase={API_BASE} />
                )}
              </div>
            )
          )}
        </main>
      </div>
    </div>
  );
}
