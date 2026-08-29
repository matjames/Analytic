import React, { useState, useEffect } from 'react';

// Lightweight built-in SVG icons
const ShieldIcon = ({ className = "w-5 h-5", style = {} }) => (
  <svg className={className} style={{ width: '20px', height: '20px', ...style }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
  </svg>
);

const LockIcon = ({ style = {} }) => (
  <svg style={{ width: '18px', height: '18px', ...style }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
  </svg>
);

const ActivityIcon = ({ style = {} }) => (
  <svg style={{ width: '18px', height: '18px', ...style }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
  </svg>
);

const AwardIcon = ({ style = {} }) => (
  <svg style={{ width: '18px', height: '18px', ...style }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
  </svg>
);

const FileCheckIcon = ({ style = {} }) => (
  <svg style={{ width: '18px', height: '18px', ...style }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
  </svg>
);

const AlertTriangleIcon = ({ style = {} }) => (
  <svg style={{ width: '18px', height: '18px', ...style }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
  </svg>
);

const GridIcon = ({ style = {} }) => (
  <svg style={{ width: '18px', height: '18px', ...style }} fill="currentColor" viewBox="0 0 18 18" aria-hidden="true">
    {[3, 9, 15].flatMap(y => [3, 9, 15].map(x => <circle key={`${x}-${y}`} cx={x} cy={y} r="1.6" />))}
  </svg>
);

const LayersIcon = ({ style = {} }) => (
  <svg style={{ width: '18px', height: '18px', ...style }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
  </svg>
);

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8094/api/v1';

export default function App() {
  const [activeTab, setActiveTab] = useState('overview');
  const [summary, setSummary] = useState(null);
  const [incidents, setIncidents] = useState([]);
  const [ledger, setLedger] = useState([]);
  const [credentials, setCredentials] = useState([]);
  const [provenance, setProvenance] = useState([]);
  const [interApps, setInterApps] = useState([]);
  const [consents, setConsents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showAppLauncher, setShowAppLauncher] = useState(false);

  // DLP Scanner state
  const [dlpInput, setDlpInput] = useState('Participant CM89012345678A submitted field response with api_key = "sk_live_uganda_data_secret" and phone +256700123456');
  const [dlpRedact, setDlpRedact] = useState(true);
  const [dlpResult, setDlpResult] = useState(null);
  const [dlpScanning, setDlpScanning] = useState(false);

  // New Credential state
  const [newCred, setNewCred] = useState({
    holder_did: 'did:statgate:user:kato.ivan',
    holder_name: 'Ivan Kato',
    credential_type: 'FieldDataOfficerCredential',
    subject_role: 'Senior Enumerator',
    subject_domain: 'Health & Demographic Surveys'
  });

  // New Incident state
  const [newInc, setNewInc] = useState({
    title: '',
    severity: 'HIGH',
    threat_category: 'Unauthorized Data Access',
    source_app: 'StatCollect',
    affected_asset: 'survey_db_staging'
  });

  const fetchData = async () => {
    try {
      setLoading(true);
      const [sumRes, incRes, ledRes, credRes, provRes, appsRes, conRes] = await Promise.allSettled([
        fetch(`${API_BASE}/summary`).then(r => r.json()),
        fetch(`${API_BASE}/secops/incidents`).then(r => r.json()),
        fetch(`${API_BASE}/trust/ledger/records`).then(r => r.json()),
        fetch(`${API_BASE}/trust/credentials`).then(r => r.json()),
        fetch(`${API_BASE}/trust/provenance`).then(r => r.json()),
        fetch(`${API_BASE}/interop/apps`).then(r => r.json()),
        fetch(`${API_BASE}/secops/privacy/consents`).then(r => r.json())
      ]);

      if (sumRes.status === 'fulfilled') setSummary(sumRes.value);
      if (incRes.status === 'fulfilled') setIncidents(incRes.value.incidents || []);
      if (ledRes.status === 'fulfilled') setLedger(ledRes.value.ledger || []);
      if (credRes.status === 'fulfilled') setCredentials(credRes.value.credentials || []);
      if (provRes.status === 'fulfilled') setProvenance(provRes.value.provenance || []);
      if (appsRes.status === 'fulfilled') setInterApps(appsRes.value.apps || []);
      if (conRes.status === 'fulfilled') setConsents(conRes.value.consents || []);
    } catch (err) {
      console.error('Error fetching data:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleRunDLPScan = async () => {
    setDlpScanning(true);
    try {
      const res = await fetch(`${API_BASE}/secops/dlp/scan`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          source_app: 'Interactive Scanner',
          data_type: 'TEXT',
          content: dlpInput,
          redact: dlpRedact,
          actor: 'secops-operator'
        })
      });
      const data = await res.json();
      setDlpResult(data);
    } catch (e) {
      console.error('DLP Scan failed', e);
    } finally {
      setDlpScanning(false);
    }
  };

  const handleIssueCredential = async (e) => {
    e.preventDefault();
    try {
      await fetch(`${API_BASE}/trust/credentials/issue`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          holder_did: newCred.holder_did,
          holder_name: newCred.holder_name,
          credential_type: newCred.credential_type,
          credential_subject: {
            role: newCred.subject_role,
            domain: newCred.subject_domain,
            authorized_by: 'StatGate Trust Authority'
          }
        })
      });
      fetchData();
    } catch (err) {
      console.error('Failed to issue credential', err);
    }
  };

  const handleCreateIncident = async (e) => {
    e.preventDefault();
    try {
      await fetch(`${API_BASE}/secops/incidents`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: newInc.title,
          severity: newInc.severity,
          status: 'OPEN',
          threat_category: newInc.threat_category,
          source_app: newInc.source_app,
          affected_asset: newInc.affected_asset,
          assigned_to: 'SecOps On-Call Lead'
        })
      });
      setNewInc({ title: '', severity: 'HIGH', threat_category: 'Unauthorized Data Access', source_app: 'StatCollect', affected_asset: 'survey_db_staging' });
      fetchData();
    } catch (err) {
      console.error('Failed to create incident', err);
    }
  };

  const handleContainIncident = async (id) => {
    try {
      await fetch(`${API_BASE}/secops/incidents/${id}/status`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          status: 'CONTAINED',
          note: 'Operator triggered automated quarantine via Zero Trust Gateway'
        })
      });
      fetchData();
    } catch (err) {
      console.error('Failed to contain incident', err);
    }
  };

  const statGateApps = [
    { name: 'App Launcher', port: 3006, url: 'http://localhost:3006', icon: '🚀', desc: 'Unified Platform Portal' },
    { name: 'StatGate Analytics', port: 5000, url: 'http://localhost:5000', icon: '📊', desc: 'Statistical Intelligence' },
    { name: 'Field Registry', port: 3007, url: 'http://localhost:3007', icon: '🏛️', desc: 'Operations & Facilities' },
    { name: 'Helpdesk', port: 3005, url: 'http://localhost:3005', icon: '🎫', desc: 'Support & Ticketing' },
    { name: 'StatChat', port: 3009, url: 'http://localhost:3009', icon: '💬', desc: 'Real-Time Collaboration' },
    { name: 'PMS', port: 3010, url: 'http://localhost:3010', icon: '🏗️', desc: 'Project Portfolio' },
    { name: 'RMS', port: 3011, url: 'http://localhost:3011', icon: '🔬', desc: 'Research & Ethics' },
    { name: 'StatGovernance', port: 3012, url: 'http://localhost:3012', icon: '⚖️', desc: 'Audit & Compliance' },
    { name: 'StatSpatial', port: 3014, url: 'http://localhost:3014', icon: '🗺️', desc: 'Geospatial Intelligence' },
{ name: 'Report Builder', port: 8110, url: 'http://localhost:8110/builder.html', icon: '🧮', desc: 'Visual YAML Reports & Maps' },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', minHeight: '100vh', background: '#090d16' }}>
      {/* Top Header */}
      <header style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '14px 28px',
        borderBottom: '1px solid rgba(255, 255, 255, 0.08)',
        background: 'rgba(15, 23, 42, 0.85)',
        backdropFilter: 'blur(12px)',
        position: 'sticky',
        top: 0,
        zIndex: 50
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
          <img
            src="/logo.png"
            alt="StatGate"
            style={{
              width: '38px',
              height: '38px',
              borderRadius: '10px',
              objectFit: 'contain',
              background: '#ffffff',
              padding: '3px',
              boxShadow: '0 0 16px rgba(6, 182, 212, 0.25)'
            }}
            onError={(e) => { e.target.style.display = 'none'; }}
          />
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span style={{ fontSize: '18px', fontWeight: 800, letterSpacing: '-0.5px', color: '#f3f4f6' }}>StatTrust</span>
              <span className="badge badge-cyan">App 3</span>
              <span className="badge badge-emerald">Phase 19 + 28</span>
            </div>
            <p style={{ fontSize: '11px', color: '#9ca3af' }}>Security Operations, Blockchain & Digital Trust Platform</p>
          </div>
        </div>

        {/* Top Status & 3x3 App Launcher */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            background: 'rgba(16, 185, 129, 0.1)',
            border: '1px solid rgba(16, 185, 129, 0.25)',
            padding: '6px 14px',
            borderRadius: '9999px'
          }}>
            <div style={{ width: '8px', height: '8px', borderRadius: '50%', background: '#10b981', boxShadow: '0 0 8px #10b981' }} className="glow-live" />
            <span style={{ fontSize: '12px', fontWeight: 600, color: '#34d399' }}>Zero Trust Active</span>
          </div>

          <button
            onClick={() => setShowAppLauncher(!showAppLauncher)}
            style={{
              background: 'rgba(255, 255, 255, 0.05)',
              border: '1px solid rgba(255, 255, 255, 0.1)',
              borderRadius: '8px',
              padding: '8px 12px',
              color: '#f3f4f6',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
              fontSize: '12px',
              fontWeight: 600
            }}
          >
            <GridIcon style={{ width: '16px', height: '16px' }} />
            <span>App Switcher</span>
          </button>
        </div>
      </header>

      {/* 3x3 App Launcher Modal */}
      {showAppLauncher && (
        <div style={{
          position: 'fixed',
          top: '70px',
          right: '28px',
          width: '380px',
          background: '#0f172a',
          border: '1px solid rgba(56, 189, 248, 0.3)',
          borderRadius: '16px',
          boxShadow: '0 20px 40px rgba(0,0,0,0.6)',
          zIndex: 100,
          padding: '16px'
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '14px' }}>
            <span style={{ fontSize: '13px', fontWeight: 700, color: '#e2e8f0', textTransform: 'uppercase', letterSpacing: '0.5px' }}>StatGate Unified Ecosystem</span>
            <span style={{ fontSize: '11px', color: '#94a3b8' }}>9 Connected Apps</span>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '10px' }}>
            {statGateApps.map(app => (
              <a
                key={app.name}
                href={app.url}
                target="_blank"
                rel="noreferrer"
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  textAlign: 'center',
                  gap: '10px',
                  padding: '10px',
                  borderRadius: '10px',
                  background: 'rgba(255,255,255,0.03)',
                  border: '1px solid rgba(255,255,255,0.06)',
                  textDecoration: 'none',
                  color: '#e2e8f0',
                  transition: 'all 0.2s'
                }}
              >
                <span style={{ fontSize: '20px' }}>{app.icon}</span>
                <div>
                  <div style={{ fontSize: '12px', fontWeight: 700, color: '#f8fafc' }}>{app.name}</div>
                  <div style={{ fontSize: '10px', color: '#94a3b8' }}>Port :{app.port}</div>
                </div>
              </a>
            ))}
          </div>
        </div>
      )}

      {/* Main Navigation Tabs */}
      <nav style={{
        display: 'flex',
        gap: '8px',
        padding: '8px 28px',
        background: '#0b1120',
        borderBottom: '1px solid rgba(255, 255, 255, 0.05)'
      }}>
        {[
          { id: 'overview', label: 'Command Center', icon: ActivityIcon },
          { id: 'soc', label: '🛡️ SOC & Incidents (P19)', icon: ShieldIcon },
          { id: 'dlp', label: '🔍 DLP & Privacy Shield (P19)', icon: LockIcon },
          { id: 'credentials', label: '🪪 Verifiable Credentials (P28)', icon: AwardIcon },
          { id: 'ledger', label: '⛓️ Immutable Ledger & Provenance (P28)', icon: FileCheckIcon },
          { id: 'mesh', label: '🌐 Inter-App Mesh (10 Apps)', icon: LayersIcon }
        ].map(tab => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                padding: '8px 16px',
                borderRadius: '8px',
                border: 'none',
                background: isActive ? 'rgba(6, 182, 212, 0.15)' : 'transparent',
                color: isActive ? '#38bdf8' : '#9ca3af',
                fontSize: '13px',
                fontWeight: isActive ? 700 : 500,
                cursor: 'pointer',
                transition: 'all 0.2s',
                borderBottom: isActive ? '2px solid #06b6d4' : '2px solid transparent'
              }}
            >
              <Icon style={{ width: '15px', height: '15px' }} />
              {tab.label}
            </button>
          );
        })}
      </nav>

      {/* Content Container */}
      <main style={{ flex: 1, padding: '28px', maxWidth: '1440px', width: '100%', margin: '0 auto' }}>
        {/* TAB 1: OVERVIEW */}
        {activeTab === 'overview' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
            {/* Top Stat Cards */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
              <div className="glass-panel" style={{ padding: '20px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', color: '#9ca3af', fontSize: '12px' }}>
                  <span>SYSTEM TRUST INDEX</span>
                  <AwardIcon style={{ color: '#38bdf8' }} />
                </div>
                <div style={{ fontSize: '28px', fontWeight: 800, color: '#f3f4f6', marginTop: '8px' }}>
                  {summary ? `${summary.global_trust_score}%` : '98.7%'}
                </div>
                <span className="badge badge-emerald" style={{ marginTop: '6px' }}>ISO 27001 & W3C Verified</span>
              </div>

              <div className="glass-panel" style={{ padding: '20px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', color: '#9ca3af', fontSize: '12px' }}>
                  <span>ACTIVE INCIDENTS</span>
                  <AlertTriangleIcon style={{ color: '#f59e0b' }} />
                </div>
                <div style={{ fontSize: '28px', fontWeight: 800, color: '#f3f4f6', marginTop: '8px' }}>
                  {summary ? summary.active_incidents : incidents.length}
                </div>
                <span className="badge badge-amber" style={{ marginTop: '6px' }}>0 Critical Breaches</span>
              </div>

              <div className="glass-panel" style={{ padding: '20px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', color: '#9ca3af', fontSize: '12px' }}>
                  <span>MERKLE LEDGER HEIGHT</span>
                  <FileCheckIcon style={{ color: '#10b981' }} />
                </div>
                <div style={{ fontSize: '28px', fontWeight: 800, color: '#f3f4f6', marginTop: '8px' }}>
                  {summary ? `#${summary.ledger_height}` : `#${ledger.length}`}
                </div>
                <span className="badge badge-indigo" style={{ marginTop: '6px' }}>SHA-256 Merkle Chain</span>
              </div>

              <div className="glass-panel" style={{ padding: '20px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', color: '#9ca3af', fontSize: '12px' }}>
                  <span>DLP SCANS TODAY</span>
                  <LockIcon style={{ color: '#6366f1' }} />
                </div>
                <div style={{ fontSize: '28px', fontWeight: 800, color: '#f3f4f6', marginTop: '8px' }}>
                  {summary ? summary.dlp_scans_today : 184}
                </div>
                <span className="badge badge-cyan" style={{ marginTop: '6px' }}>12 PII Threats Blocked</span>
              </div>
            </div>

            {/* Middle 2-Column: Live Threat Operations & Blockchain Ledger Feed */}
            <div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: '20px' }}>
              {/* SOC Incidents Summary */}
              <div className="glass-panel" style={{ padding: '22px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                  <div>
                    <h3 style={{ fontSize: '16px', fontWeight: 700, color: '#f8fafc' }}>Active Security Incidents (P19)</h3>
                    <p style={{ fontSize: '12px', color: '#94a3b8' }}>Real-time threat detection from connected apps</p>
                  </div>
                  <button className="btn-secondary" onClick={() => setActiveTab('soc')}>View All Incidents</button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  {incidents.slice(0, 3).map(inc => (
                    <div key={inc.id} style={{
                      background: 'rgba(255,255,255,0.02)',
                      border: '1px solid rgba(255,255,255,0.06)',
                      borderRadius: '10px',
                      padding: '14px',
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center'
                    }}>
                      <div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <span style={{ fontSize: '13px', fontWeight: 700, color: '#f1f5f9' }}>{inc.title}</span>
                          <span className={`badge ${inc.severity === 'HIGH' ? 'badge-rose' : 'badge-amber'}`}>{inc.severity}</span>
                        </div>
                        <p style={{ fontSize: '11px', color: '#94a3b8', marginTop: '4px' }}>
                          Source: <strong style={{ color: '#38bdf8' }}>{inc.source_app}</strong> • Category: {inc.threat_category}
                        </p>
                      </div>
                      <span className={`badge ${inc.status === 'CONTAINED' ? 'badge-emerald' : 'badge-amber'}`}>{inc.status}</span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Immutable Ledger Live Stream */}
              <div className="glass-panel" style={{ padding: '22px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                  <div>
                    <h3 style={{ fontSize: '16px', fontWeight: 700, color: '#f8fafc' }}>Immutable Audit Ledger (P28)</h3>
                    <p style={{ fontSize: '12px', color: '#94a3b8' }}>Cryptographic block anchoring across ecosystem</p>
                  </div>
                  <button className="btn-secondary" onClick={() => setActiveTab('ledger')}>Explore Blocks</button>
                </div>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                  {ledger.map(block => (
                    <div key={block.index} style={{
                      background: 'rgba(255,255,255,0.02)',
                      border: '1px solid rgba(255,255,255,0.06)',
                      borderRadius: '8px',
                      padding: '10px 14px',
                      fontFamily: 'var(--font-mono)'
                    }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px', color: '#38bdf8' }}>
                        <span>BLOCK #{block.index}</span>
                        <span style={{ color: '#a78bfa' }}>{block.source_app}</span>
                      </div>
                      <div style={{ fontSize: '11px', color: '#cbd5e1', marginTop: '4px', fontWeight: 600 }}>{block.event_type}</div>
                      <div style={{ fontSize: '10px', color: '#64748b', marginTop: '2px', wordBreak: 'break-all' }}>
                        Hash: {block.record_hash.slice(0, 32)}...
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* TAB 2: SOC & INCIDENTS */}
        {activeTab === 'soc' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <h2 style={{ fontSize: '20px', fontWeight: 800, color: '#f8fafc' }}>Security Operations Center (SOC)</h2>
                <p style={{ fontSize: '13px', color: '#94a3b8' }}>Phase 19: Real-time incident triage, threat hunting, and containment playbooks</p>
              </div>
              <button className="btn-primary" onClick={() => setActiveTab('dlp')}>Launch Interactive DLP Scanner</button>
            </div>

            {/* Create Incident Form */}
            <div className="glass-panel" style={{ padding: '20px' }}>
              <h3 style={{ fontSize: '14px', fontWeight: 700, color: '#e2e8f0', marginBottom: '14px' }}>Dispatch New Security Incident</h3>
              <form onSubmit={handleCreateIncident} style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr 1fr auto', gap: '12px', alignItems: 'end' }}>
                <div>
                  <label style={{ fontSize: '11px', color: '#94a3b8', display: 'block', marginBottom: '4px' }}>Incident Title</label>
                  <input
                    type="text"
                    className="input-field"
                    placeholder="e.g. Unusual API burst on Field Survey endpoint"
                    value={newInc.title}
                    onChange={e => setNewInc({ ...newInc, title: e.target.value })}
                    required
                  />
                </div>
                <div>
                  <label style={{ fontSize: '11px', color: '#94a3b8', display: 'block', marginBottom: '4px' }}>Severity</label>
                  <select
                    className="input-field"
                    value={newInc.severity}
                    onChange={e => setNewInc({ ...newInc, severity: e.target.value })}
                  >
                    <option value="LOW">LOW</option>
                    <option value="MEDIUM">MEDIUM</option>
                    <option value="HIGH">HIGH</option>
                    <option value="CRITICAL">CRITICAL</option>
                  </select>
                </div>
                <div>
                  <label style={{ fontSize: '11px', color: '#94a3b8', display: 'block', marginBottom: '4px' }}>Source App</label>
                  <select
                    className="input-field"
                    value={newInc.source_app}
                    onChange={e => setNewInc({ ...newInc, source_app: e.target.value })}
                  >
                    <option value="StatCollect">StatCollect</option>
                    <option value="StatSpatial">StatSpatial</option>
                    <option value="PMS">PMS</option>
                    <option value="RMS">RMS</option>
                    <option value="StatChat">StatChat</option>
                    <option value="StatGovernance">StatGovernance</option>
                  </select>
                </div>
                <div>
                  <label style={{ fontSize: '11px', color: '#94a3b8', display: 'block', marginBottom: '4px' }}>Affected Asset</label>
                  <input
                    type="text"
                    className="input-field"
                    value={newInc.affected_asset}
                    onChange={e => setNewInc({ ...newInc, affected_asset: e.target.value })}
                  />
                </div>
                <button type="submit" className="btn-primary" style={{ height: '40px' }}>Dispatch Alert</button>
              </form>
            </div>

            {/* Incidents List */}
            <div className="glass-panel" style={{ padding: '20px' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '14px' }}>
                <h3 style={{ fontSize: '15px', fontWeight: 700, color: '#f8fafc' }}>Incident Register ({incidents.length})</h3>
                <span className="badge badge-cyan">Automated SIEM & Event Bus Link</span>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                {incidents.map(inc => (
                  <div key={inc.id} style={{
                    background: 'rgba(15, 23, 42, 0.6)',
                    border: '1px solid rgba(255, 255, 255, 0.08)',
                    borderRadius: '12px',
                    padding: '16px'
                  }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                          <span style={{ fontSize: '15px', fontWeight: 700, color: '#f8fafc' }}>{inc.title}</span>
                          <span className={`badge ${inc.severity === 'CRITICAL' || inc.severity === 'HIGH' ? 'badge-rose' : 'badge-amber'}`}>{inc.severity}</span>
                          <span className="badge badge-indigo">{inc.source_app}</span>
                        </div>
                        <p style={{ fontSize: '12px', color: '#94a3b8', marginTop: '6px' }}>
                          ID: <strong style={{ color: '#cbd5e1' }}>{inc.id}</strong> • Asset: <code style={{ color: '#38bdf8' }}>{inc.affected_asset}</code> • Threat: {inc.threat_category}
                        </p>
                      </div>

                      <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                        <span className={`badge ${inc.status === 'RESOLVED' ? 'badge-emerald' : inc.status === 'CONTAINED' ? 'badge-cyan' : 'badge-amber'}`}>{inc.status}</span>
                        {inc.status === 'OPEN' && (
                          <button
                            className="btn-primary"
                            style={{ fontSize: '11px', padding: '6px 12px' }}
                            onClick={() => handleContainIncident(inc.id)}
                          >
                            Quarantine / Contain
                          </button>
                        )}
                      </div>
                    </div>

                    {inc.remediation_log && inc.remediation_log.length > 0 && (
                      <div style={{ marginTop: '12px', padding: '10px', background: 'rgba(0,0,0,0.3)', borderRadius: '8px', fontSize: '11px', color: '#94a3b8' }}>
                        <strong style={{ color: '#cbd5e1' }}>Remediation Timeline:</strong>
                        <ul style={{ paddingLeft: '18px', marginTop: '4px' }}>
                          {inc.remediation_log.map((log, i) => (
                            <li key={i}>{log}</li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* TAB 3: DLP & PRIVACY SHIELD */}
        {activeTab === 'dlp' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div>
              <h2 style={{ fontSize: '20px', fontWeight: 800, color: '#f8fafc' }}>Data Loss Prevention & Privacy Shield</h2>
              <p style={{ fontSize: '13px', color: '#94a3b8' }}>Phase 19: Deep packet & payload inspection for PII, confidential tokens, and GDPR/DPPA compliance</p>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: '20px' }}>
              {/* Interactive DLP Scanner */}
              <div className="glass-panel" style={{ padding: '22px' }}>
                <h3 style={{ fontSize: '15px', fontWeight: 700, color: '#f8fafc', marginBottom: '8px' }}>Interactive Cross-App DLP Engine</h3>
                <p style={{ fontSize: '12px', color: '#94a3b8', marginBottom: '14px' }}>Test real-time payload sanitization against Uganda NIN, telephone numbers, and exposed API keys.</p>

                <textarea
                  className="input-field"
                  rows={6}
                  style={{ fontFamily: 'var(--font-mono)', fontSize: '12px' }}
                  value={dlpInput}
                  onChange={e => setDlpInput(e.target.value)}
                />

                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '14px' }}>
                  <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', color: '#cbd5e1', cursor: 'pointer' }}>
                    <input
                      type="checkbox"
                      checked={dlpRedact}
                      onChange={e => setDlpRedact(e.target.checked)}
                    />
                    Apply Automatic PII Masking / Token Redaction
                  </label>

                  <button className="btn-primary" onClick={handleRunDLPScan} disabled={dlpScanning}>
                    {dlpScanning ? 'Scanning...' : 'Execute DLP Scan'}
                  </button>
                </div>

                {dlpResult && (
                  <div style={{ marginTop: '18px', padding: '14px', background: 'rgba(0,0,0,0.4)', borderRadius: '10px', border: '1px solid rgba(255,255,255,0.08)' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
                      <span style={{ fontSize: '13px', fontWeight: 700, color: dlpResult.safe ? '#34d399' : '#fb7185' }}>
                        {dlpResult.safe ? '✓ Payload Safe: 0 Violations Found' : `⚠️ Risk: ${dlpResult.RiskLevel || dlpResult.risk_level} (${dlpResult.violations_found} Violations)`}
                      </span>
                      <span className={`badge ${dlpResult.action_taken === 'BLOCKED' ? 'badge-rose' : 'badge-amber'}`}>Action: {dlpResult.action_taken}</span>
                    </div>

                    {dlpResult.matched_rules && dlpResult.matched_rules.length > 0 && (
                      <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap', marginBottom: '10px' }}>
                        {dlpResult.matched_rules.map(rule => (
                          <span key={rule} className="badge badge-rose">{rule}</span>
                        ))}
                      </div>
                    )}

                    {dlpResult.redacted_content && (
                      <div>
                        <div style={{ fontSize: '11px', color: '#94a3b8', marginBottom: '4px' }}>Sanitized Output:</div>
                        <div style={{ background: '#090d16', padding: '10px', borderRadius: '6px', fontFamily: 'var(--font-mono)', fontSize: '11px', color: '#38bdf8', wordBreak: 'break-all' }}>
                          {dlpResult.redacted_content}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* Privacy Consents Register */}
              <div className="glass-panel" style={{ padding: '22px' }}>
                <h3 style={{ fontSize: '15px', fontWeight: 700, color: '#f8fafc', marginBottom: '8px' }}>Privacy & Consent Ledger</h3>
                <p style={{ fontSize: '12px', color: '#94a3b8', marginBottom: '14px' }}>Uganda DPPA & GDPR individual subject permissions</p>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                  {consents.map(c => (
                    <div key={c.id} style={{
                      background: 'rgba(255,255,255,0.02)',
                      border: '1px solid rgba(255,255,255,0.06)',
                      borderRadius: '8px',
                      padding: '12px'
                    }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span style={{ fontSize: '13px', fontWeight: 700, color: '#f1f5f9' }}>{c.subject_name || c.subject_id}</span>
                        <span className="badge badge-emerald">{c.status}</span>
                      </div>
                      <p style={{ fontSize: '11px', color: '#94a3b8', marginTop: '4px' }}>Purpose: {c.purpose}</p>
                      <span className="badge badge-indigo" style={{ marginTop: '6px' }}>Scope: {c.data_scope}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* TAB 4: VERIFIABLE CREDENTIALS */}
        {activeTab === 'credentials' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <h2 style={{ fontSize: '20px', fontWeight: 800, color: '#f8fafc' }}>Verifiable Credentials & Decentralized Identity</h2>
                <p style={{ fontSize: '13px', color: '#94a3b8' }}>Phase 28: Cryptographic W3C verifiable credentials with Ed25519 digital signatures</p>
              </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.4fr', gap: '20px' }}>
              {/* Issue Credential Form */}
              <div className="glass-panel" style={{ padding: '22px' }}>
                <h3 style={{ fontSize: '15px', fontWeight: 700, color: '#f8fafc', marginBottom: '14px' }}>Issue W3C Verifiable Credential</h3>
                <form onSubmit={handleIssueCredential} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                  <div>
                    <label style={{ fontSize: '11px', color: '#94a3b8', display: 'block', marginBottom: '4px' }}>Holder DID</label>
                    <input
                      type="text"
                      className="input-field"
                      value={newCred.holder_did}
                      onChange={e => setNewCred({ ...newCred, holder_did: e.target.value })}
                      required
                    />
                  </div>
                  <div>
                    <label style={{ fontSize: '11px', color: '#94a3b8', display: 'block', marginBottom: '4px' }}>Holder Name</label>
                    <input
                      type="text"
                      className="input-field"
                      value={newCred.holder_name}
                      onChange={e => setNewCred({ ...newCred, holder_name: e.target.value })}
                      required
                    />
                  </div>
                  <div>
                    <label style={{ fontSize: '11px', color: '#94a3b8', display: 'block', marginBottom: '4px' }}>Credential Type</label>
                    <input
                      type="text"
                      className="input-field"
                      value={newCred.credential_type}
                      onChange={e => setNewCred({ ...newCred, credential_type: e.target.value })}
                      required
                    />
                  </div>
                  <div>
                    <label style={{ fontSize: '11px', color: '#94a3b8', display: 'block', marginBottom: '4px' }}>Assigned Role</label>
                    <input
                      type="text"
                      className="input-field"
                      value={newCred.subject_role}
                      onChange={e => setNewCred({ ...newCred, subject_role: e.target.value })}
                    />
                  </div>
                  <button type="submit" className="btn-primary" style={{ marginTop: '8px' }}>Cryptographically Issue & Sign</button>
                </form>
              </div>

              {/* Active Credentials Wallet */}
              <div className="glass-panel" style={{ padding: '22px' }}>
                <h3 style={{ fontSize: '15px', fontWeight: 700, color: '#f8fafc', marginBottom: '14px' }}>Trust Registry & Credential Wallet ({credentials.length})</h3>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                  {credentials.map(cred => (
                    <div key={cred.id} style={{
                      background: 'rgba(15, 23, 42, 0.7)',
                      border: '1px solid rgba(56, 189, 248, 0.2)',
                      borderRadius: '12px',
                      padding: '16px'
                    }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                          <span style={{ fontSize: '14px', fontWeight: 700, color: '#f8fafc' }}>{cred.holder_name}</span>
                          <span className="badge badge-cyan" style={{ marginLeft: '8px' }}>{cred.type ? cred.type[1] || cred.type[0] : 'VerifiableCredential'}</span>
                        </div>
                        <span className="badge badge-emerald">{cred.status}</span>
                      </div>

                      <p style={{ fontSize: '11px', color: '#94a3b8', marginTop: '6px', fontFamily: 'var(--font-mono)' }}>
                        DID: {cred.holder_did}
                      </p>

                      <div style={{ marginTop: '10px', padding: '10px', background: 'rgba(0,0,0,0.3)', borderRadius: '8px', fontSize: '11px' }}>
                        <div style={{ color: '#38bdf8' }}>Issuer: {cred.issuer_did}</div>
                        <div style={{ color: '#a78bfa', marginTop: '2px' }}>Proof Type: {cred.proof?.type} ({cred.proof?.proof_purpose})</div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* TAB 5: IMMUTABLE LEDGER & PROVENANCE */}
        {activeTab === 'ledger' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div>
              <h2 style={{ fontSize: '20px', fontWeight: 800, color: '#f8fafc' }}>Immutable Ledger & Research Provenance</h2>
              <p style={{ fontSize: '13px', color: '#94a3b8' }}>Phase 28: Cryptographic chain-of-custody for research outputs, survey datasets, and project milestones</p>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: '20px' }}>
              {/* Merkle Ledger Explorer */}
              <div className="glass-panel" style={{ padding: '22px' }}>
                <h3 style={{ fontSize: '15px', fontWeight: 700, color: '#f8fafc', marginBottom: '14px' }}>Cryptographic Block Chain (Height: {ledger.length})</h3>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                  {ledger.map((block) => (
                    <div key={block.index} style={{
                      background: 'rgba(15, 23, 42, 0.7)',
                      border: '1px solid rgba(99, 102, 241, 0.3)',
                      borderRadius: '10px',
                      padding: '14px',
                      fontFamily: 'var(--font-mono)'
                    }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span style={{ fontSize: '13px', fontWeight: 700, color: '#818cf8' }}>BLOCK #{block.index}</span>
                        <span className="badge badge-indigo">{block.source_app}</span>
                      </div>

                      <div style={{ fontSize: '12px', fontWeight: 600, color: '#f1f5f9', marginTop: '6px' }}>{block.event_type}</div>
                      <div style={{ fontSize: '11px', color: '#94a3b8', marginTop: '4px' }}>Actor: {block.actor_id}</div>

                      <div style={{ marginTop: '8px', padding: '8px', background: '#090d16', borderRadius: '6px', fontSize: '10px', color: '#64748b' }}>
                        <div>Prev: {block.prev_hash}</div>
                        <div style={{ color: '#34d399', marginTop: '2px' }}>Hash: {block.record_hash}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Artifact Provenance Chain */}
              <div className="glass-panel" style={{ padding: '22px' }}>
                <h3 style={{ fontSize: '15px', fontWeight: 700, color: '#f8fafc', marginBottom: '14px' }}>Cross-App Artifact Provenance ({provenance.length})</h3>

                <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
                  {provenance.map(prov => (
                    <div key={prov.id} style={{
                      background: 'rgba(15, 23, 42, 0.7)',
                      border: '1px solid rgba(16, 185, 129, 0.3)',
                      borderRadius: '12px',
                      padding: '16px'
                    }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span style={{ fontSize: '14px', fontWeight: 700, color: '#f8fafc' }}>{prov.artifact_name}</span>
                        <span className="badge badge-emerald">{prov.integrity_state}</span>
                      </div>

                      <p style={{ fontSize: '11px', color: '#94a3b8', marginTop: '6px' }}>
                        Origin: <strong style={{ color: '#38bdf8' }}>{prov.origin_app}</strong> • ID: {prov.artifact_id}
                      </p>

                      <div style={{ marginTop: '10px', padding: '10px', background: '#090d16', borderRadius: '8px', fontSize: '11px' }}>
                        <div style={{ color: '#64748b', fontSize: '10px', wordBreak: 'break-all' }}>SHA256: {prov.sha256_checksum}</div>
                        <div style={{ color: '#a78bfa', marginTop: '6px', fontWeight: 600 }}>Custody Chain ({prov.custody_chain?.length || 0} Steps):</div>
                        <ul style={{ paddingLeft: '16px', marginTop: '4px', color: '#94a3b8' }}>
                          {prov.custody_chain?.map((step, i) => (
                            <li key={i}>{step.action} by {step.actor} ({step.app})</li>
                          ))}
                        </ul>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* TAB 6: INTER-APP MESH */}
        {activeTab === 'mesh' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
            <div>
              <h2 style={{ fontSize: '20px', fontWeight: 800, color: '#f8fafc' }}>Multi-App Mesh & Communication Topology</h2>
              <p style={{ fontSize: '13px', color: '#94a3b8' }}>Bidirectional event stream, DLP inspection, and cross-application trust federation across all 10 StatGate systems</p>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '16px' }}>
              {interApps.map(app => (
                <div key={app.app_name} className="glass-panel" style={{ padding: '18px' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      <div style={{ width: '10px', height: '10px', borderRadius: '50%', background: '#10b981', boxShadow: '0 0 8px #10b981' }} />
                      <span style={{ fontSize: '15px', fontWeight: 700, color: '#f8fafc' }}>{app.app_name}</span>
                    </div>
                    <span className="badge badge-cyan">Port :{app.port}</span>
                  </div>

                  <p style={{ fontSize: '12px', color: '#94a3b8', marginTop: '8px', fontFamily: 'var(--font-mono)' }}>
                    Endpoint: {app.service_url}
                  </p>

                  <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '14px', paddingTop: '10px', borderTop: '1px solid rgba(255,255,255,0.06)', fontSize: '12px' }}>
                    <span style={{ color: '#38bdf8' }}>Events Ingested: <strong>{app.events_ingested}</strong></span>
                    <span className="badge badge-emerald">Trust Federated</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
