import React, { useState, useEffect, useRef } from 'react';

// All StatGate apps for the switcher overlay — mirrors the platform service
// registry (single source: frontend/config/services.json, served live via the launcher).
const STATGATE_APPS = [
  { id: 'launcher', name: 'All Apps Portal', icon: '🚀', url: 'http://localhost:3006' },
  { id: 'analytics', name: 'Analytics Hub', icon: '📊', url: 'http://localhost:5000' },
  { id: 'registry', name: 'Facility Registry', icon: '🪪', url: 'http://localhost:3007' },
  { id: 'helpdesk', name: 'Operations Helpdesk', icon: '🎧', url: 'http://localhost:3005' },
  { id: 'statchat', name: 'StatChat Messenger', icon: '💬', url: 'http://localhost:3009' },
  { id: 'pms', name: 'Projects (PMS)', icon: '🏗️', url: 'http://localhost:3010' },
  { id: 'rms', name: 'Research (RMS)', icon: '🔬', url: 'http://localhost:3011', current: true },
  { id: 'governance', name: 'Governance & Risk', icon: '⚖️', url: 'http://localhost:3012' },
  { id: 'spatial', name: 'Geospatial & GIS', icon: '🗺️', url: 'http://localhost:3014' },
  { id: 'federation', name: 'Diplomacy & SDMX', icon: '🌐', url: 'http://localhost:3017' },
  { id: 'trust', name: 'Trust & SecOps', icon: '🔒', url: 'http://localhost:3013' },
  { id: 'ops', name: 'SRE & Cloud Ops', icon: '🛠️', url: 'http://localhost:3015' },
];

/**
 * Decode JWT token safely without external dependencies
 */
function decodeJwt(token) {
  if (!token) return null;
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payloadBase64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const jsonStr = decodeURIComponent(
      atob(payloadBase64)
        .split('')
        .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    );
    const payload = JSON.parse(jsonStr);
    if (payload && payload.exp && payload.exp * 1000 < Date.now()) {
      return null;
    }
    return payload;
  } catch (e) {
    return null;
  }
}

export default function StatGateHeader({ searchValue, onSearchChange }) {
  const [launcherOpen, setLauncherOpen] = useState(false);
  const [notifOpen, setNotifOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [currentUser, setCurrentUser] = useState(null);
  const [branding, setBranding] = useState({ display_name: 'StatGate', logo_url: '', primary_color: '#165c92', secondary_color: '#0f3f5f' });
  const launcherRef = useRef(null);
  const notifRef = useRef(null);
  const userMenuRef = useRef(null);

  // Sync authenticated identity from shared Registry token
  const syncAuth = () => {
    const token = localStorage.getItem('registry_jwt') || localStorage.getItem('token');
    const payload = decodeJwt(token);
    if (payload) {
      setCurrentUser({
        name: payload.name || payload.email || payload.sub || 'Authenticated Researcher',
        email: payload.email || '',
        role: payload.role || 'Principal Investigator',
        tenantId: payload.tenant_id || payload.tenantId || 'National Research Unit',
        orgId: payload.org_id || '',
      });
    } else {
      setCurrentUser(null);
    }
  };

  useEffect(() => {
    syncAuth();
    const token = localStorage.getItem('registry_jwt') || localStorage.getItem('token');
    const registryAPI = import.meta.env.VITE_REGISTRY_API_URL || 'http://localhost:9090/api';
    fetch(`${registryAPI}/organisation/branding`, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
      .then(response => response.ok ? response.json() : null)
      .then(data => { if (data) setBranding(prev => ({ ...prev, ...data })); })
      .catch(() => {});

    window.addEventListener('storage', syncAuth);
    window.addEventListener('auth-expired', syncAuth);
    return () => {
      window.removeEventListener('storage', syncAuth);
      window.removeEventListener('auth-expired', syncAuth);
    };
  }, []);

  // Close overlays on outside click
  useEffect(() => {
    const handler = (e) => {
      if (launcherRef.current && !launcherRef.current.contains(e.target)) setLauncherOpen(false);
      if (notifRef.current && !notifRef.current.contains(e.target)) setNotifOpen(false);
      if (userMenuRef.current && !userMenuRef.current.contains(e.target)) setUserMenuOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  // Close on Escape
  useEffect(() => {
    const handler = (e) => {
      if (e.key === 'Escape') {
        setLauncherOpen(false);
        setNotifOpen(false);
        setUserMenuOpen(false);
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

  const handleLogout = () => {
    localStorage.removeItem('registry_jwt');
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    window.dispatchEvent(new CustomEvent('auth-expired'));
    window.location.href = 'http://localhost:3006';
  };

  const handleLaunchApp = (appUrl) => {
    const token = localStorage.getItem('registry_jwt');
    const separator = appUrl.includes('?') ? '&' : '?';
    const target = token ? `${appUrl}${separator}statgate_token=${encodeURIComponent(token)}` : appUrl;
    window.location.href = target;
  };

  const displayName = currentUser?.name || 'Dr. Catherine Mitchell';
  const displayRole = currentUser?.role || 'Principal Investigator';

  return (
    <header className="statgate-header" style={{ background: `linear-gradient(135deg, ${branding.primary_color} 0%, ${branding.secondary_color} 100%)` }}>
      {/* Left: Brand */}
      <div className="header-brand">
        {branding.logo_url ? (
          <img className="header-logo" src={branding.logo_url} alt={branding.display_name} />
        ) : (
          <img className="header-logo" src="/logo.png" alt="StatGate" onError={(e) => { e.target.style.display='none'; }} style={{ borderRadius: 6, background: 'rgba(255,255,255,0.12)', padding: 2 }} />
        )}
        <div className="header-brand-text">
          <span className="header-brand-name">{branding.display_name}</span>
          <span className="header-brand-sub">Research Management System</span>
        </div>
      </div>

      {/* Centre: Search */}
      <div className="header-search-wrap">
        <span className="header-search-icon">🔍</span>
        <input
          type="text"
          placeholder="Search research studies, proposals, PI, codes, tags…"
          value={searchValue || ''}
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
            onClick={() => { setNotifOpen(o => !o); setLauncherOpen(false); setUserMenuOpen(false); }}
            aria-label="Notifications"
          >
            🔔
            <span className="header-badge">3</span>
          </button>
          {notifOpen && (
            <div className="header-dropdown notif-dropdown">
              <div className="dropdown-title">Research Notifications</div>
              {[
                { icon: '⚠️', text: 'Malaria Study: IRB Ethics review expiring in 14 days', time: '10m ago' },
                { icon: '✅', text: 'Grant Proposal approved by NIH Review Board', time: '1h ago' },
                { icon: '📄', text: 'Manuscript revision requested by Lancet Global Health', time: '3h ago' },
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
            title="StatGate Interconnected App Launcher"
            onClick={() => { setLauncherOpen(o => !o); setNotifOpen(false); setUserMenuOpen(false); }}
            aria-label="App Launcher"
          >
            <div className="app-grid-dots">
              {[...Array(9)].map((_, i) => <div key={i} className="dot" />)}
            </div>
          </button>

          {launcherOpen && (
            <div className="header-dropdown launcher-dropdown">
              <div className="dropdown-title">StatGate Platform Ecosystem</div>
              <div className="launcher-grid">
                {STATGATE_APPS.map(app => (
                  <div
                    key={app.id}
                    className={`launcher-app-item ${app.current ? 'launcher-app-current' : ''}`}
                    onClick={() => {
                      setLauncherOpen(false);
                      handleLaunchApp(app.url);
                    }}
                    title={app.name}
                  >
                    <div className="launcher-app-icon">{app.icon}</div>
                    <div className="launcher-app-name">{app.name}</div>
                    {app.current && <div className="launcher-app-badge">Active</div>}
                  </div>
                ))}
              </div>
              <div className="launcher-footer">
                <a href="http://localhost:3006" target="_blank" rel="noopener noreferrer" className="launcher-all-link">
                  Open Complete Portal →
                </a>
              </div>
            </div>
          )}
        </div>

        {/* User Profile & Single Sign-On */}
        <div ref={userMenuRef} style={{ position: 'relative' }}>
          <div
            className="header-user"
            onClick={() => { setUserMenuOpen(o => !o); setLauncherOpen(false); setNotifOpen(false); }}
            title="Account & SSO Profile"
          >
            <img
              src={`https://api.dicebear.com/7.x/adventurer/svg?seed=${displayName.replace(/[^a-zA-Z0-9]/g, '')}`}
              alt="avatar"
              className="header-avatar"
            />
            <div className="header-user-info">
              <span className="header-user-name">{displayName}</span>
              <span className="header-user-role">{displayRole}</span>
            </div>
          </div>

          {userMenuOpen && (
            <div className="header-dropdown" style={{ width: 280, right: 0, padding: 16 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, paddingBottom: 12, borderBottom: '1px solid var(--border-light)' }}>
                <img
                  src={`https://api.dicebear.com/7.x/adventurer/svg?seed=${displayName.replace(/[^a-zA-Z0-9]/g, '')}`}
                  alt="avatar"
                  style={{ width: 44, height: 44, borderRadius: '50%', background: 'var(--bg-light)' }}
                />
                <div>
                  <div style={{ fontWeight: 800, fontSize: 14, color: 'var(--text-dark)' }}>{displayName}</div>
                  <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>{currentUser?.email || 'admin@statgate.local'}</div>
                  <div style={{ fontSize: 10, color: 'var(--primary-color)', fontWeight: 700, marginTop: 2 }}>{displayRole}</div>
                </div>
              </div>

              <div style={{ padding: '12px 0', display: 'flex', flexDirection: 'column', gap: 8, fontSize: 12 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', color: 'var(--text-secondary)' }}>
                  <span>Organisation:</span>
                  <strong style={{ color: 'var(--text-dark)' }}>{currentUser?.tenantId || 'National Health Institute'}</strong>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', color: 'var(--text-secondary)' }}>
                  <span>Auth Standard:</span>
                  <strong style={{ color: '#059669' }}>StatGate Registry SSO</strong>
                </div>
              </div>

              <div style={{ paddingTop: 8, borderTop: '1px solid var(--border-light)', display: 'flex', flexDirection: 'column', gap: 6 }}>
                <a
                  href="http://localhost:3006"
                  className="btn btn-secondary"
                  style={{ width: '100%', justifyContent: 'center', fontSize: 12 }}
                >
                  🚀 Switch Application
                </a>
                <button
                  onClick={handleLogout}
                  className="btn"
                  style={{ width: '100%', justifyContent: 'center', fontSize: 12, background: '#fee2e2', color: '#dc2626', border: '1px solid #fca5a5' }}
                >
                  🔒 Sign Out (Ecosystem)
                </button>
              </div>
            </div>
          )}
        </div>

      </div>
    </header>
  );
}
