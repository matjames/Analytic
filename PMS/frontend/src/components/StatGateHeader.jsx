import React, { useState, useEffect, useRef } from 'react';

// All StatGate apps for the switcher overlay — mirrors the platform service
// registry (single source: frontend/config/services.json, served live via the
// launcher). Keep this list in sync with that registry.
const STATGATE_APPS = [
  { id: 'analytics',  name: 'Analytics',           icon: '📊', url: 'http://localhost:5000' },
  { id: 'registry',   name: 'Field Registry',       icon: '🏥', url: 'http://localhost:3007' },
  { id: 'helpdesk',   name: 'Operations Helpdesk',  icon: '🎫', url: 'http://localhost:3005' },
  { id: 'statchat',   name: 'StatChat',             icon: '💬', url: 'http://localhost:3009' },
  { id: 'pms',        name: 'PMS',                  icon: '🏗️', url: 'http://localhost:3010', current: true },
  { id: 'rms',        name: 'RMS',                  icon: '🔬', url: 'http://localhost:3011' },
  { id: 'governance', name: 'StatGovernance',       icon: '🛡️', url: 'http://localhost:3012' },
  { id: 'statcollect',name: 'StatCollect',          icon: '📥', url: 'http://localhost:8080' },
  { id: 'statspatial',name: 'StatSpatial',          icon: '🗺️', url: 'http://localhost:4200' },
  { id: 'enterprise', name: 'Enterprise',           icon: '🧩', url: 'http://localhost:8096' },
  { id: 'jupyter',    name: 'JupyterHub',           icon: '📓', url: 'http://localhost:8000' },
  { id: 'superset',   name: 'Superset BI',          icon: '📈', url: 'http://localhost:8088' },
  { id: 'grafana',    name: 'Grafana',              icon: '📉', url: 'http://localhost:3003' },
  { id: 'mlflow',     name: 'MLflow',               icon: '🤖', url: 'http://localhost:5002' },
];

export default function StatGateHeader({ searchValue, onSearchChange, userName = 'Sarah Jenkins', userRole = 'Admin' }) {
  const [launcherOpen, setLauncherOpen] = useState(false);
  const [notifOpen, setNotifOpen] = useState(false);
  const launcherRef = useRef(null);
  const notifRef = useRef(null);

  // Close overlays on outside click
  useEffect(() => {
    const handler = (e) => {
      if (launcherRef.current && !launcherRef.current.contains(e.target)) setLauncherOpen(false);
      if (notifRef.current && !notifRef.current.contains(e.target)) setNotifOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  // Close on Escape
  useEffect(() => {
    const handler = (e) => { if (e.key === 'Escape') { setLauncherOpen(false); setNotifOpen(false); } };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

  return (
    <header className="statgate-header">
      {/* Left: Brand */}
      <div className="header-brand">
        <div className="header-logo">SG</div>
        <div className="header-brand-text">
          <span className="header-brand-name">StatGate</span>
          <span className="header-brand-sub">Projects Management System</span>
        </div>
      </div>

      {/* Centre: Search */}
      <div className="header-search-wrap">
        <span className="header-search-icon">🔍</span>
        <input
          type="text"
          placeholder="Search workspaces, programmes, codes, tags…"
          value={searchValue}
          onChange={e => onSearchChange(e.target.value)}
          className="header-search-input"
        />
      </div>

      {/* Right: Actions */}
      <div className="header-actions">

        {/* Notification Bell */}
        <div ref={notifRef} style={{ position: 'relative' }}>
          <button
            className="header-icon-btn"
            title="Notifications"
            onClick={() => { setNotifOpen(o => !o); setLauncherOpen(false); }}
            aria-label="Notifications"
          >
            🔔
            <span className="header-badge">3</span>
          </button>
          {notifOpen && (
            <div className="header-dropdown notif-dropdown">
              <div className="dropdown-title">Notifications</div>
              {[
                { icon: '⚠️', text: 'NCD Survey: 2 new risks flagged', time: '5m ago' },
                { icon: '✅', text: 'AGRI Census task approved by Marcus', time: '1h ago' },
                { icon: '💬', text: 'New message in NCD Project chat', time: '2h ago' },
              ].map((n, i) => (
                <div key={i} className="notif-item">
                  <span>{n.icon}</span>
                  <div>
                    <div style={{ fontSize: '12px' }}>{n.text}</div>
                    <div style={{ fontSize: '10px', color: 'var(--text-muted)' }}>{n.time}</div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* App Launcher (9-dot grid) */}
        <div ref={launcherRef} style={{ position: 'relative' }}>
          <button
            className="header-icon-btn app-grid-btn"
            title="App Launcher"
            onClick={() => { setLauncherOpen(o => !o); setNotifOpen(false); }}
            aria-label="App Launcher"
          >
            <div className="app-grid-dots">
              {[...Array(9)].map((_, i) => <div key={i} className="dot" />)}
            </div>
          </button>

          {launcherOpen && (
            <div className="header-dropdown launcher-dropdown">
              <div className="dropdown-title">StatGate Platform</div>
              <div className="launcher-grid">
                {STATGATE_APPS.map(app => (
                  <a
                    key={app.id}
                    href={app.url}
                    target="_self"
                    rel="noopener noreferrer"
                    className={`launcher-app-item ${app.current ? 'launcher-app-current' : ''}`}
                    onClick={() => setLauncherOpen(false)}
                    title={app.name}
                  >
                    <div className="launcher-app-icon">{app.icon}</div>
                    <div className="launcher-app-name">{app.name}</div>
                    {app.current && <div className="launcher-app-badge">Active</div>}
                  </a>
                ))}
              </div>
              <div className="launcher-footer">
                <a href="http://localhost:3006" target="_blank" rel="noopener noreferrer" className="launcher-all-link">
                  View All Applications →
                </a>
              </div>
            </div>
          )}
        </div>

        {/* User Profile */}
        <div className="header-user">
          <img
            src={`https://api.dicebear.com/7.x/adventurer/svg?seed=${userName.replace(' ', '')}`}
            alt="avatar"
            className="header-avatar"
          />
          <div className="header-user-info">
            <span className="header-user-name">{userName}</span>
            <span className="header-user-role">{userRole}</span>
          </div>
        </div>
      </div>
    </header>
  );
}