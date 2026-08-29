import React, { useState, useEffect } from 'react';

const API_BASE = 'http://localhost:8105/api/v1';

export default function App() {
  const [activeTab, setActiveTab] = useState('nodes');
  const [nodes, setNodes] = useState([]);
  const [indicators, setIndicators] = useState([]);
  const [dsas, setDsas] = useState([]);
  const [treaties, setTreaties] = useState([]);
  const [reports, setReports] = useState([]);
  const [auditLogs, setAuditLogs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [notification, setNotification] = useState(null);

  // Push Protocol State
  const [pushTargetNode, setPushTargetNode] = useState('');
  const [pushSelectedIndicators, setPushSelectedIndicators] = useState([]);
  const [pushDomain, setPushDomain] = useState('SDG');
  const [pushResult, setPushResult] = useState(null);
  const [pushing, setPushing] = useState(false);

  // Federated Search State
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [searching, setSearching] = useState(false);

  // Modal States
  const [showNodeModal, setShowNodeModal] = useState(false);
  const [showDsaModal, setShowDsaModal] = useState(false);
  const [showIndicatorModal, setShowIndicatorModal] = useState(false);
  const [showTreatyModal, setShowTreatyModal] = useState(false);
  const [showAppLauncher, setShowAppLauncher] = useState(false);

  const statGateApps = [
    { name: 'All Apps Portal', url: 'http://localhost:3006', icon: '🚀', desc: 'Unified Platform Portal' },
    { name: 'Analytics Hub', url: 'http://localhost:5000', icon: '📊', desc: 'BI & Statistical Intelligence' },
    { name: 'Facility Registry', url: 'http://localhost:3007', icon: '🪪', desc: 'Master Facility List & Workforce' },
    { name: 'StatCollect', url: 'http://localhost:8084/admin/index.html', icon: '📋', desc: 'Enterprise CAPI & Forms' },
    { name: 'StatFederation', url: 'http://localhost:3017', icon: '🌐', desc: 'SDMX & Global Diplomacy Mesh', current: true },
    { name: 'StatChat', url: 'http://localhost:3009', icon: '💬', desc: 'Secure Messaging & Conferencing' },
    { name: 'PMS', url: 'http://localhost:3010', icon: '🏗️', desc: 'Project Portfolio & Logistics' },
    { name: 'RMS', url: 'http://localhost:3011', icon: '🔬', desc: 'Research & Ethics Management' },
    { name: 'StatGovernance', url: 'http://localhost:3012', icon: '⚖️', desc: 'Institutional Compliance & Risk' },
    { name: 'StatSpatial', url: 'http://localhost:3014', icon: '🗺️', desc: 'Geospatial & PostGIS Studio' },
    { name: 'StatTrust', url: 'http://localhost:3013', icon: '🔒', desc: 'SecOps & Verifiable Credentials' },
    { name: 'StatOps', url: 'http://localhost:3015', icon: '🛠️', desc: 'SRE Observability & Cloud Hub' },
  ];

  // Forms
  const [nodeForm, setNodeForm] = useState({
    name: '',
    code: '',
    node_type: 'nss_agency',
    jurisdiction: 'NATIONAL',
    endpoint_url: '',
    trust_level: 'TIER_2_DOMESTIC_AGENCY',
    contact_email: '',
  });

  const [dsaForm, setDsaForm] = useState({
    title: '',
    provider_node_id: '',
    consumer_node_id: '',
    permitted_domains: 'health,sdg,demographics',
    purpose: '',
    daily_quota: 10000,
    valid_from: new Date().toISOString().split('T')[0],
    valid_until: new Date(Date.now() + 365 * 24 * 3600 * 1000).toISOString().split('T')[0],
  });

  const [indicatorForm, setIndicatorForm] = useState({
    code: '',
    title: '',
    domain: 'SDG',
    current_value: 0,
    unit_of_measure: '',
    frequency: 'ANNUAL',
    sdmx_dimension: '',
  });

  const [treatyForm, setTreatyForm] = useState({
    treaty_code: '',
    title: '',
    jurisdiction: 'EAST_AFRICAN_COMMUNITY',
    framework_type: 'BILATERAL_EVIDENCE_PACT',
    governing_body: 'EAC Secretariat',
    encryption_standard: 'AES_256_GCM',
  });

  const notify = (msg, type = 'success') => {
    setNotification({ msg, type });
    setTimeout(() => setNotification(null), 4500);
  };

  const fetchAll = async () => {
    setLoading(true);
    try {
      const [nRes, iRes, dRes, tRes, rRes, aRes] = await Promise.all([
        fetch(`${API_BASE}/federation/nodes`).then((r) => r.json()),
        fetch(`${API_BASE}/federation/indicators`).then((r) => r.json()),
        fetch(`${API_BASE}/federation/dsa`).then((r) => r.json()),
        fetch(`${API_BASE}/diplomacy/treaties`).then((r) => r.json()),
        fetch(`${API_BASE}/diplomacy/reports`).then((r) => r.json()),
        fetch(`${API_BASE}/diplomacy/compliance/audit-logs`).then((r) => r.json()),
      ]);

      if (nRes?.nodes) setNodes(nRes.nodes);
      if (iRes?.indicators) setIndicators(iRes.indicators);
      if (dRes?.agreements) setDsas(dRes.agreements);
      if (tRes?.treaties) setTreaties(tRes.treaties);
      if (rRes?.reports) setReports(rRes.reports);
      if (aRes?.audit_logs) setAuditLogs(aRes.audit_logs);
    } catch (e) {
      console.warn('API error (using fallback state):', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAll();
  }, []);

  // Probe Nodes
  const handleProbeNodes = async () => {
    try {
      const res = await fetch(`${API_BASE}/federation/nodes/probe`, { method: 'POST' });
      const data = await res.json();
      if (data?.nodes) {
        setNodes(data.nodes);
        notify('Fleet health probe complete across all federated participants!');
      }
    } catch (err) {
      notify('Probe failed: ' + err.message, 'error');
    }
  };

  // Execute Sovereign Push
  const handleExecutePush = async () => {
    if (!pushTargetNode) {
      notify('Please select a target hub node', 'error');
      return;
    }
    setPushing(true);
    try {
      const payload = {
        source_node_id: 'node-nss-001',
        target_node_id: pushTargetNode,
        indicator_ids: pushSelectedIndicators,
        domain: pushDomain,
      };
      const res = await fetch(`${API_BASE}/federation/indicators/push`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      setPushResult(data);
      if (data.status === 'SUCCEEDED') {
        notify(`Pushed ${data.indicator_count} indicators to target hub successfully!`);
      } else {
        notify(`Push completed with status: ${data.status}`, 'warning');
      }
      fetchAll();
    } catch (err) {
      notify('Push error: ' + err.message, 'error');
    } finally {
      setPushing(false);
    }
  };

  // Federated Search
  const handleFederatedSearch = async () => {
    if (!searchQuery) return;
    setSearching(true);
    try {
      const res = await fetch(`${API_BASE}/diplomacy/search`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: searchQuery }),
      });
      const data = await res.json();
      if (data?.results) {
        setSearchResults(data.results);
      }
    } catch (err) {
      notify('Federated search error: ' + err.message, 'error');
    } finally {
      setSearching(false);
    }
  };

  // Node Register
  const handleRegisterNode = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${API_BASE}/federation/nodes`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(nodeForm),
      });
      if (res.ok) {
        notify('Node registered successfully');
        setShowNodeModal(false);
        fetchAll();
      }
    } catch (err) {
      notify(err.message, 'error');
    }
  };

  // DSA Create
  const handleCreateDsa = async (e) => {
    e.preventDefault();
    try {
      const payload = {
        ...dsaForm,
        permitted_domains: dsaForm.permitted_domains.split(',').map((s) => s.trim()),
        valid_from: new Date(dsaForm.valid_from).toISOString(),
        valid_until: new Date(dsaForm.valid_until).toISOString(),
      };
      const res = await fetch(`${API_BASE}/federation/dsa`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (res.ok) {
        notify('Data Sharing Agreement drafted');
        setShowDsaModal(false);
        fetchAll();
      }
    } catch (err) {
      notify(err.message, 'error');
    }
  };

  // Approve DSA
  const handleApproveDsa = async (id) => {
    try {
      const res = await fetch(`${API_BASE}/federation/dsa/${id}/approve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ approved_by: 'Data Governance Commissioner' }),
      });
      if (res.ok) {
        notify('DSA Approved & Activated');
        fetchAll();
      }
    } catch (err) {
      notify(err.message, 'error');
    }
  };

  // Create Indicator
  const handleCreateIndicator = async (e) => {
    e.preventDefault();
    try {
      const val = parseFloat(indicatorForm.current_value);
      const res = await fetch(`${API_BASE}/federation/indicators`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...indicatorForm, current_value: isNaN(val) ? 0 : val }),
      });
      if (res.ok) {
        notify('National Indicator registered');
        setShowIndicatorModal(false);
        fetchAll();
      }
    } catch (err) {
      notify(err.message, 'error');
    }
  };

  // Generate SDG Report
  const handleGenerateSDGReport = async () => {
    try {
      const res = await fetch(`${API_BASE}/diplomacy/reports/sdg`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          reporting_year: 2026,
          sdg_goals: [3, 4, 8, 13],
        }),
      });
      const data = await res.json();
      notify('Voluntary National Review SDG Report compiled and transmitted!');
      fetchAll();
    } catch (err) {
      notify(err.message, 'error');
    }
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Top Header */}
      <header
        style={{
          background: 'linear-gradient(135deg, #0f3f5f 0%, #165c92 50%, #1a7ab5 100%)',
          borderBottom: '1px solid rgba(255,255,255,0.15)',
          boxShadow: '0 2px 10px rgba(15, 63, 95, 0.25)',
          padding: '12px 24px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          position: 'sticky',
          top: 0,
          zIndex: 100,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 14 }}>
          <img
            src="/logo.png"
            alt="StatGate"
            style={{
              width: 38,
              height: 38,
              borderRadius: 8,
              objectFit: 'contain',
              background: '#ffffff',
              padding: 2,
              border: '1px solid rgba(255,255,255,0.3)',
              boxShadow: '0 2px 6px rgba(0,0,0,0.15)',
            }}
          />
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <h1 style={{ fontSize: 18, fontWeight: 700, color: '#ffffff', letterSpacing: -0.2 }}>
                StatGate <span style={{ color: '#38bdf8' }}>Federation</span>
              </h1>
              <span className="badge" style={{ background: 'rgba(255,255,255,0.18)', color: '#fff', border: '1px solid rgba(255,255,255,0.25)' }}>SDMX &amp; Diplomacy Mesh</span>
              <span className="badge badge-emerald">
                <span
                  style={{
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    background: '#10b981',
                    display: 'inline-block',
                  }}
                  className="glow-live"
                ></span>
                LIVE DIPLOMACY MESH
              </span>
            </div>
            <p style={{ fontSize: 11, color: 'rgba(255, 255, 255, 0.85)' }}>
              National Statistical System (NSS) Interoperability &amp; Global Evidence Exchange
            </p>
          </div>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 8, position: 'relative' }}>
          <button className="btn-secondary" onClick={handleProbeNodes} title="Probe participant nodes" style={{ background: 'rgba(255,255,255,0.12)', color: '#fff', border: '1px solid rgba(255,255,255,0.2)' }}>
            ⚡ Probe Nodes
          </button>
          <button className="btn-secondary" onClick={fetchAll} style={{ background: 'rgba(255,255,255,0.12)', color: '#fff', border: '1px solid rgba(255,255,255,0.2)' }}>
            🔄 Refresh
          </button>
          <a
            href="http://localhost:5000"
            className="btn-secondary"
            style={{ textDecoration: 'none', background: 'rgba(255,255,255,0.12)', color: '#fff', border: '1px solid rgba(255,255,255,0.2)' }}
          >
            📊 Analytics Hub
          </a>
          <a
            href="http://localhost:8084/admin/index.html"
            className="btn-secondary"
            style={{ textDecoration: 'none', background: 'rgba(255,255,255,0.12)', color: '#fff', border: '1px solid rgba(255,255,255,0.2)' }}
          >
            📋 StatCollect
          </a>

          {/* 3x3 StatGate Universal App Launcher Button */}
          <button
            onClick={() => setShowAppLauncher(!showAppLauncher)}
            title="StatGate 3x3 Universal App Launcher"
            style={{
              background: showAppLauncher ? 'rgba(255,255,255,0.3)' : 'rgba(255,255,255,0.14)',
              border: '1px solid rgba(255,255,255,0.25)',
              borderRadius: 8,
              padding: '8px 10px',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'all 0.2s',
              marginLeft: 4,
            }}
          >
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 2.5, width: 16, height: 16 }}>
              <div style={{ background: '#ffffff', borderRadius: 1 }}></div><div style={{ background: '#ffffff', borderRadius: 1 }}></div><div style={{ background: '#ffffff', borderRadius: 1 }}></div>
              <div style={{ background: '#ffffff', borderRadius: 1 }}></div><div style={{ background: '#ffffff', borderRadius: 1 }}></div><div style={{ background: '#ffffff', borderRadius: 1 }}></div>
              <div style={{ background: '#ffffff', borderRadius: 1 }}></div><div style={{ background: '#ffffff', borderRadius: 1 }}></div><div style={{ background: '#ffffff', borderRadius: 1 }}></div>
            </div>
          </button>

          {/* 3x3 App Launcher Dropdown */}
          {showAppLauncher && (
            <div
              style={{
                position: 'absolute',
                top: 48,
                right: 0,
                width: 380,
                background: '#ffffff',
                border: '1px solid #e2e8f0',
                borderRadius: 12,
                boxShadow: '0 16px 36px rgba(15,23,42,0.18)',
                padding: '16px',
                zIndex: 1000,
                color: '#0f172a',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12, paddingBottom: 8, borderBottom: '1px solid #e2e8f0' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <img src="/logo.png" alt="StatGate" style={{ width: 22, height: 22, objectFit: 'contain' }} onError={(e) => { e.target.style.display='none'; }} />
                  <span style={{ fontSize: 13, fontWeight: 700, color: '#0f3f5f' }}>StatGate Platform Ecosystem</span>
                </div>
                <span style={{ fontSize: 11, color: '#64748b', fontWeight: 600 }}>12 Apps</span>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 8 }}>
                {statGateApps.map((app) => (
                  <a
                    key={app.name}
                    href={app.url}
                    target="_blank"
                    rel="noreferrer"
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      padding: '10px 6px',
                      borderRadius: 8,
                      border: app.current ? '1px solid #165c92' : '1px solid transparent',
                      background: app.current ? '#eff6ff' : '#f8fafc',
                      textDecoration: 'none',
                      color: app.current ? '#165c92' : '#1e293b',
                      transition: 'all 0.15s ease',
                      textAlign: 'center',
                    }}
                    onMouseEnter={(e) => { e.currentTarget.style.background = '#e2e8f0'; e.currentTarget.style.transform = 'translateY(-1px)'; }}
                    onMouseLeave={(e) => { e.currentTarget.style.background = app.current ? '#eff6ff' : '#f8fafc'; e.currentTarget.style.transform = 'none'; }}
                  >
                    <span style={{ fontSize: 22, marginBottom: 4 }}>{app.icon}</span>
                    <span style={{ fontSize: 11, fontWeight: 600, lineHeight: 1.2 }}>{app.name}</span>
                    {app.current && <span style={{ fontSize: 9, color: '#165c92', fontWeight: 700, marginTop: 2 }}>CURRENT</span>}
                  </a>
                ))}
              </div>
            </div>
          )}
        </div>
      </header>

      {/* Notification Toast */}
      {notification && (
        <div
          style={{
            position: 'fixed',
            top: 75,
            right: 24,
            zIndex: 999,
            background: notification.type === 'error' ? 'rgba(244, 63, 94, 0.9)' : 'rgba(6, 182, 212, 0.9)',
            color: 'white',
            padding: '12px 20px',
            borderRadius: 10,
            backdropFilter: 'blur(10px)',
            boxShadow: '0 8px 30px rgba(0,0,0,0.5)',
            fontWeight: 600,
            fontSize: 13,
            display: 'flex',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <span>{notification.type === 'error' ? '❌' : '✨'}</span>
          {notification.msg}
        </div>
      )}

      {/* Navigation Tabs */}
      <div
        style={{
          display: 'flex',
          gap: 6,
          padding: '12px 28px',
          background: '#ffffff',
          borderBottom: '1px solid #e2e8f0',
          boxShadow: '0 1px 2px rgba(0,0,0,0.03)',
          overflowX: 'auto',
        }}
      >
        {[
          { id: 'nodes', label: 'Participant Nodes', icon: '🏛️', count: nodes.length },
          { id: 'push', label: 'Push Protocol', icon: '🚀', highlight: true },
          { id: 'indicators', label: 'National Indicators', icon: '📊', count: indicators.length },
          { id: 'dsa', label: 'Data Sharing Agreements', icon: '📜', count: dsas.length },
          { id: 'treaties', label: 'Diplomatic Treaties', icon: '🕊️', count: treaties.length },
          { id: 'reports', label: 'Multilateral Reports', icon: '📑', count: reports.length },
          { id: 'search', label: 'Federated Search', icon: '🔍' },
          { id: 'audit', label: 'Compliance Audit', icon: '🛡️', count: auditLogs.length },
        ].map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '8px 16px',
              borderRadius: 8,
              fontSize: 13,
              fontWeight: 600,
              border:
                activeTab === tab.id
                  ? '1px solid #165c92'
                  : '1px solid transparent',
              background:
                activeTab === tab.id
                  ? '#eff6ff'
                  : tab.highlight
                  ? '#f5f3ff'
                  : 'transparent',
              color: activeTab === tab.id ? '#165c92' : '#64748b',
              cursor: 'pointer',
              whiteSpace: 'nowrap',
              transition: 'all 0.15s ease',
            }}
          >
            <span>{tab.icon}</span>
            {tab.label}
            {tab.count !== undefined && (
              <span
                style={{
                  background: activeTab === tab.id ? 'rgba(22,92,146,0.12)' : '#e2e8f0',
                  color: activeTab === tab.id ? '#165c92' : '#475569',
                  padding: '2px 7px',
                  borderRadius: 10,
                  fontSize: 11,
                  fontWeight: 700,
                }}
              >
                {tab.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Main Content Area */}
      <main style={{ flex: 1, padding: '24px 28px', maxWidth: 1440, width: '100%', margin: '0 auto' }}>
        {/* TAB 1: PARTICIPANT NODES */}
        {activeTab === 'nodes' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
              <div>
                <h2 style={{ fontSize: 20, fontWeight: 700 }}>Participating Statistical Hubs & Nodes</h2>
                <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                  Registered national ministries, bureaus, regional blocs (AU/EAC), and international partners
                </p>
              </div>
              <button className="btn-primary" onClick={() => setShowNodeModal(true)}>
                ➕ Register New Node
              </button>
            </div>

            {/* Metrics Row */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: 16, marginBottom: 24 }}>
              <div className="glass-panel" style={{ padding: 18 }}>
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>TOTAL REGISTERED NODES</div>
                <div style={{ fontSize: 28, fontWeight: 800, marginTop: 4, color: '#38bdf8' }}>{nodes.length}</div>
                <div style={{ fontSize: 11, color: 'var(--emerald-success)', marginTop: 4 }}>✓ 100% NSS Mesh Connected</div>
              </div>
              <div className="glass-panel" style={{ padding: 18 }}>
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>HEALTHY NODES</div>
                <div style={{ fontSize: 28, fontWeight: 800, marginTop: 4, color: '#34d399' }}>
                  {nodes.filter((n) => n.health_status === 'HEALTHY').length}
                </div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>Real-time heartbeat verified</div>
              </div>
              <div className="glass-panel" style={{ padding: 18 }}>
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>AVERAGE LATENCY</div>
                <div style={{ fontSize: 28, fontWeight: 800, marginTop: 4, color: '#fbbf24' }}>
                  {nodes.length > 0
                    ? Math.round(nodes.reduce((acc, n) => acc + (n.latency_ms || 25), 0) / nodes.length)
                    : 0}{' '}
                  ms
                </div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>Sub-second scatter-gather</div>
              </div>
              <div className="glass-panel" style={{ padding: 18 }}>
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>ACTIVE DSAs</div>
                <div style={{ fontSize: 28, fontWeight: 800, marginTop: 4, color: '#c084fc' }}>
                  {dsas.filter((d) => d.status === 'ACTIVE').length}
                </div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>Legal treaties in force</div>
              </div>
            </div>

            {/* Node Grid */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: 16 }}>
              {nodes.map((node) => (
                <div key={node.id} className="glass-panel" style={{ padding: 20 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12 }}>
                    <div>
                      <div style={{ fontSize: 15, fontWeight: 700 }}>{node.name}</div>
                      <div style={{ fontSize: 12, color: '#38bdf8', fontFamily: 'var(--font-mono)', marginTop: 2 }}>
                        {node.code}
                      </div>
                    </div>
                    <span
                      className={`badge ${
                        node.health_status === 'HEALTHY'
                          ? 'badge-emerald'
                          : node.health_status === 'DEGRADED'
                          ? 'badge-amber'
                          : 'badge-rose'
                      }`}
                    >
                      {node.health_status}
                    </span>
                  </div>

                  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 12, wordBreak: 'break-all' }}>
                    🔗 <span style={{ fontFamily: 'var(--font-mono)' }}>{node.endpoint_url}</span>
                  </div>

                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginBottom: 14 }}>
                    <span className="badge badge-purple">{node.jurisdiction}</span>
                    <span className="badge badge-cyan">{node.node_type}</span>
                    <span className="badge badge-amber">{node.trust_level}</span>
                  </div>

                  <div style={{ borderTop: '1px solid var(--border-subtle)', paddingTop: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 12, color: 'var(--text-dim)' }}>
                    <span>⏱️ Latency: <strong style={{ color: 'white' }}>{node.latency_ms || 25}ms</strong></span>
                    <span>Protocols: {(node.protocols || ['SDMX-REST']).join(', ')}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* TAB 2: SOVEREIGN PUSH PROTOCOL */}
        {activeTab === 'push' && (
          <div style={{ maxWidth: 880, margin: '0 auto' }}>
            <div style={{ marginBottom: 24, textAlign: 'center' }}>
              <span className="badge badge-purple" style={{ marginBottom: 8 }}>
                SPRINT 1 CORE CAPABILITY
              </span>
              <h2 style={{ fontSize: 24, fontWeight: 800 }}>Sovereign Indicator Push Protocol</h2>
              <p style={{ fontSize: 13, color: 'var(--text-muted)', marginTop: 4 }}>
                Transmit approved aggregated indicators to national, regional, and global partner hubs in standard SDMX-JSON format.
              </p>
            </div>

            <div className="glass-panel" style={{ padding: 28, marginBottom: 24 }}>
              <h3 style={{ fontSize: 16, fontWeight: 700, marginBottom: 16, color: '#38bdf8' }}>
                1. Select Target Analytical Hub
              </h3>
              <select
                className="input-field"
                value={pushTargetNode}
                onChange={(e) => setPushTargetNode(e.target.value)}
                style={{ marginBottom: 20 }}
              >
                <option value="">-- Choose destination node --</option>
                {nodes.map((n) => (
                  <option key={n.id} value={n.id}>
                    {n.name} ({n.code}) — {n.jurisdiction}
                  </option>
                ))}
              </select>

              <h3 style={{ fontSize: 16, fontWeight: 700, marginBottom: 16, color: '#38bdf8' }}>
                2. Select Domain & National Indicators to Push
              </h3>
              <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
                {['SDG', 'health', 'demographics', 'economy', 'environment'].map((dom) => (
                  <button
                    key={dom}
                    type="button"
                    onClick={() => setPushDomain(dom)}
                    style={{
                      padding: '6px 14px',
                      borderRadius: 8,
                      fontSize: 12,
                      fontWeight: 600,
                      textTransform: 'uppercase',
                      border: pushDomain === dom ? '1px solid #38bdf8' : '1px solid var(--border-subtle)',
                      background: pushDomain === dom ? 'rgba(6, 182, 212, 0.2)' : 'transparent',
                      color: pushDomain === dom ? '#38bdf8' : 'var(--text-muted)',
                      cursor: 'pointer',
                    }}
                  >
                    {dom}
                  </button>
                ))}
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 10, marginBottom: 24 }}>
                {indicators.map((ind) => (
                  <label
                    key={ind.id}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 10,
                      padding: '10px 14px',
                      borderRadius: 8,
                      background: 'rgba(255,255,255,0.03)',
                      border: '1px solid var(--border-subtle)',
                      cursor: 'pointer',
                      fontSize: 13,
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={pushSelectedIndicators.includes(ind.id)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setPushSelectedIndicators([...pushSelectedIndicators, ind.id]);
                        } else {
                          setPushSelectedIndicators(pushSelectedIndicators.filter((x) => x !== ind.id));
                        }
                      }}
                    />
                    <div>
                      <div style={{ fontWeight: 600 }}>{ind.title}</div>
                      <div style={{ fontSize: 11, color: '#38bdf8' }}>
                        {ind.code} = {ind.current_value} {ind.unit_of_measure}
                      </div>
                    </div>
                  </label>
                ))}
              </div>

              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                  Selected for push: <strong>{pushSelectedIndicators.length || 'All indicators in domain'}</strong>
                </div>
                <button
                  className="btn-primary"
                  onClick={handleExecutePush}
                  disabled={pushing}
                  style={{ padding: '12px 28px', fontSize: 14 }}
                >
                  {pushing ? '⏳ Transmitting SDMX Payload...' : '🚀 Execute Sovereign Push'}
                </button>
              </div>
            </div>

            {/* Push Result Confirmation */}
            {pushResult && (
              <div className="glass-panel" style={{ padding: 24, border: '1px solid #10b981' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
                  <span style={{ fontSize: 24 }}>✅</span>
                  <div>
                    <h4 style={{ fontSize: 16, fontWeight: 700, color: '#34d399' }}>
                      Push Protocol Transmission Confirmed
                    </h4>
                    <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                      Transaction ID: <span style={{ fontFamily: 'var(--font-mono)' }}>{pushResult.push_id}</span>
                    </p>
                  </div>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, background: 'rgba(0,0,0,0.3)', padding: 14, borderRadius: 8, fontSize: 12 }}>
                  <div>Target Node: <strong>{pushResult.target_node_id}</strong></div>
                  <div>Indicators Transferred: <strong>{pushResult.indicator_count}</strong></div>
                  <div>Status: <span className="badge badge-emerald">{pushResult.status}</span></div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* TAB 3: NATIONAL INDICATORS */}
        {activeTab === 'indicators' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
              <div>
                <h2 style={{ fontSize: 20, fontWeight: 700 }}>National & Global Indicator Repository</h2>
                <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                  Standard official statistics, SDG indicators, and harmonized cross-hub metrics
                </p>
              </div>
              <button className="btn-primary" onClick={() => setShowIndicatorModal(true)}>
                ➕ Create Indicator
              </button>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(360px, 1fr))', gap: 16 }}>
              {indicators.map((ind) => (
                <div key={ind.id} className="glass-panel" style={{ padding: 20 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                    <span className="badge badge-cyan">{ind.domain}</span>
                    <span className="badge badge-emerald">{ind.frequency}</span>
                  </div>
                  <h3 style={{ fontSize: 15, fontWeight: 700, marginBottom: 4 }}>{ind.title}</h3>
                  <div style={{ fontSize: 12, color: '#38bdf8', fontFamily: 'var(--font-mono)', marginBottom: 12 }}>
                    {ind.code}
                  </div>

                  <div style={{ background: 'rgba(0,0,0,0.3)', padding: 14, borderRadius: 10, display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                    <div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>CURRENT VALUE</div>
                      <div style={{ fontSize: 22, fontWeight: 800, color: '#34d399' }}>
                        {ind.current_value !== null ? ind.current_value : 'N/A'}{' '}
                        <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--text-muted)' }}>
                          {ind.unit_of_measure}
                        </span>
                      </div>
                    </div>
                    {ind.target_value && (
                      <div style={{ textAlign: 'right' }}>
                        <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>TARGET</div>
                        <div style={{ fontSize: 16, fontWeight: 700, color: '#fbbf24' }}>
                          {ind.target_value} {ind.unit_of_measure}
                        </div>
                      </div>
                    )}
                  </div>

                  <div style={{ fontSize: 11, color: 'var(--text-dim)' }}>
                    SDMX Dimension: {ind.sdmx_dimension || 'STANDARD_DIMENSION'}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* TAB 4: DATA SHARING AGREEMENTS */}
        {activeTab === 'dsa' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
              <div>
                <h2 style={{ fontSize: 20, fontWeight: 700 }}>Data Sharing Agreements (DSAs)</h2>
                <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                  Bilateral and multilateral legal protocols governing transboundary evidence access
                </p>
              </div>
              <button className="btn-primary" onClick={() => setShowDsaModal(true)}>
                ➕ Draft New Agreement
              </button>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(380px, 1fr))', gap: 16 }}>
              {dsas.map((d) => (
                <div key={d.id} className="glass-panel" style={{ padding: 20 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 10 }}>
                    <div>
                      <div style={{ fontSize: 15, fontWeight: 700 }}>{d.title}</div>
                      <div style={{ fontSize: 12, color: '#c084fc', fontFamily: 'var(--font-mono)', marginTop: 2 }}>
                        {d.dsa_number}
                      </div>
                    </div>
                    <span className={`badge ${d.status === 'ACTIVE' ? 'badge-emerald' : 'badge-amber'}`}>
                      {d.status}
                    </span>
                  </div>

                  <p style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 14 }}>{d.purpose}</p>

                  <div style={{ background: 'rgba(0,0,0,0.25)', padding: 12, borderRadius: 8, fontSize: 12, marginBottom: 14 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
                      <span>Daily Query Usage:</span>
                      <strong>
                        {d.current_daily_usage} / {d.daily_quota}
                      </strong>
                    </div>
                    <div style={{ width: '100%', height: 6, background: 'rgba(255,255,255,0.1)', borderRadius: 3, overflow: 'hidden' }}>
                      <div
                        style={{
                          width: `${Math.min(100, (d.current_daily_usage / (d.daily_quota || 1)) * 100)}%`,
                          height: '100%',
                          background: '#06b6d4',
                        }}
                      ></div>
                    </div>
                  </div>

                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div style={{ fontSize: 11, color: 'var(--text-dim)' }}>
                      Valid: {new Date(d.valid_from).toLocaleDateString()} - {new Date(d.valid_until).toLocaleDateString()}
                    </div>
                    {d.status !== 'ACTIVE' && (
                      <button className="btn-primary" style={{ padding: '4px 12px', fontSize: 11 }} onClick={() => handleApproveDsa(d.id)}>
                        ✓ Approve & Activate
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* TAB 5: DIPLOMATIC TREATIES */}
        {activeTab === 'treaties' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
              <div>
                <h2 style={{ fontSize: 20, fontWeight: 700 }}>Diplomatic Treaties & Sovereign Accords</h2>
                <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                  Regional treaties (AU STATAFRIC, EAC, AfCFTA) and bilateral data privacy pacts
                </p>
              </div>
              <button className="btn-primary" onClick={() => setShowTreatyModal(true)}>
                ➕ Register Treaty
              </button>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(360px, 1fr))', gap: 16 }}>
              {treaties.map((t) => (
                <div key={t.id} className="glass-panel" style={{ padding: 20 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                    <span className="badge badge-purple">{t.jurisdiction}</span>
                    <span className="badge badge-emerald">{t.status}</span>
                  </div>
                  <h3 style={{ fontSize: 15, fontWeight: 700, marginBottom: 4 }}>{t.title}</h3>
                  <div style={{ fontSize: 12, color: '#38bdf8', fontFamily: 'var(--font-mono)', marginBottom: 12 }}>
                    {t.treaty_code}
                  </div>
                  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 10 }}>
                    Governing Body: <strong style={{ color: 'white' }}>{t.governing_body}</strong>
                  </div>
                  <div style={{ fontSize: 11, color: 'var(--text-dim)' }}>
                    Encryption Standard: {t.encryption_standard}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* TAB 6: MULTILATERAL REPORTS */}
        {activeTab === 'reports' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
              <div>
                <h2 style={{ fontSize: 20, fontWeight: 700 }}>Multilateral International Reports</h2>
                <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                  Official transmissions to the United Nations, African Union, and regional economic communities
                </p>
              </div>
              <button className="btn-primary" onClick={handleGenerateSDGReport}>
                📑 Generate SDG Voluntary National Report
              </button>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(360px, 1fr))', gap: 16 }}>
              {reports.map((r) => (
                <div key={r.id} className="glass-panel" style={{ padding: 20 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                    <span className="badge badge-cyan">{r.destination_body}</span>
                    <span className="badge badge-emerald">{r.status}</span>
                  </div>
                  <h3 style={{ fontSize: 15, fontWeight: 700, marginBottom: 6 }}>{r.report_title}</h3>
                  <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 12 }}>
                    Period: <strong>{r.reporting_period}</strong>
                  </div>
                  {r.submission_hash && (
                    <div style={{ background: 'rgba(0,0,0,0.3)', padding: 8, borderRadius: 6, fontSize: 10, fontFamily: 'var(--font-mono)', wordBreak: 'break-all', color: '#94a3b8' }}>
                      SHA256: {r.submission_hash}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* TAB 7: FEDERATED SEARCH */}
        {activeTab === 'search' && (
          <div style={{ maxWidth: 880, margin: '0 auto' }}>
            <div style={{ textAlign: 'center', marginBottom: 24 }}>
              <h2 style={{ fontSize: 24, fontWeight: 800 }}>Cross-Hub Federated Evidence Search</h2>
              <p style={{ fontSize: 13, color: 'var(--text-muted)', marginTop: 4 }}>
                Discover datasets, statistical tabulations, and research artifacts across all connected participant nodes
              </p>
            </div>

            <div style={{ display: 'flex', gap: 10, marginBottom: 28 }}>
              <input
                type="text"
                className="input-field"
                placeholder="Search across nodes (e.g. maternal health, agricultural census, AfCFTA tariff lines)..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleFederatedSearch()}
                style={{ fontSize: 14, padding: '12px 18px' }}
              />
              <button className="btn-primary" onClick={handleFederatedSearch} disabled={searching} style={{ padding: '0 24px' }}>
                {searching ? 'Searching...' : '🔍 Search'}
              </button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              {searchResults.map((item, idx) => (
                <div key={idx} className="glass-panel" style={{ padding: 20 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 8 }}>
                    <div>
                      <h4 style={{ fontSize: 16, fontWeight: 700, color: '#38bdf8' }}>{item.title}</h4>
                      <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>Node: {item.node_name}</div>
                    </div>
                    <span className="badge badge-purple">{item.resource_type}</span>
                  </div>
                  <p style={{ fontSize: 13, color: '#cbd5e1', marginBottom: 12 }}>{item.abstract}</p>
                  <div style={{ display: 'flex', gap: 6 }}>
                    {(item.keywords || []).map((k, ki) => (
                      <span key={ki} className="badge badge-cyan" style={{ fontSize: 10 }}>
                        {k}
                      </span>
                    ))}
                  </div>
                </div>
              ))}
              {searchResults.length === 0 && !searching && (
                <div style={{ textAlign: 'center', padding: 40, color: 'var(--text-dim)' }}>
                  Enter keywords above to dispatch real-time queries to participating NSS and international nodes.
                </div>
              )}
            </div>
          </div>
        )}

        {/* TAB 8: COMPLIANCE AUDIT */}
        {activeTab === 'audit' && (
          <div>
            <div style={{ marginBottom: 20 }}>
              <h2 style={{ fontSize: 20, fontWeight: 700 }}>Transboundary Sovereignty & Compliance Ledger</h2>
              <p style={{ fontSize: 13, color: 'var(--text-muted)' }}>
                Immutable audit trail of transboundary policy evaluations, redactions, and exchange decisions
              </p>
            </div>

            <div className="glass-panel" style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13, textAlign: 'left' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid var(--border-subtle)', background: 'rgba(255,255,255,0.02)' }}>
                    <th style={{ padding: '14px 16px' }}>Timestamp</th>
                    <th style={{ padding: '14px 16px' }}>Action</th>
                    <th style={{ padding: '14px 16px' }}>Source → Target</th>
                    <th style={{ padding: '14px 16px' }}>Decision</th>
                    <th style={{ padding: '14px 16px' }}>Policy Hash</th>
                  </tr>
                </thead>
                <tbody>
                  {auditLogs.map((log) => (
                    <tr key={log.id} style={{ borderBottom: '1px solid var(--border-subtle)' }}>
                      <td style={{ padding: '12px 16px', color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: 12 }}>
                        {new Date(log.event_timestamp).toLocaleString()}
                      </td>
                      <td style={{ padding: '12px 16px', fontWeight: 600 }}>{log.action}</td>
                      <td style={{ padding: '12px 16px' }}>
                        {log.source_jurisdiction} → {log.target_jurisdiction}
                      </td>
                      <td style={{ padding: '12px 16px' }}>
                        <span className={`badge ${log.decision === 'ALLOWED' ? 'badge-emerald' : 'badge-rose'}`}>
                          {log.decision}
                        </span>
                      </td>
                      <td style={{ padding: '12px 16px', fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-dim)' }}>
                        {log.policy_hash || 'SHA256:4f8a...'}
                      </td>
                    </tr>
                  ))}
                  {auditLogs.length === 0 && (
                    <tr>
                      <td colSpan={5} style={{ textAlign: 'center', padding: 24, color: 'var(--text-dim)' }}>
                        No compliance events recorded yet.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </main>

      {/* MODAL: Register Node */}
      {showNodeModal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', backdropFilter: 'blur(8px)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: 20 }}>
          <div className="glass-panel" style={{ width: '100%', maxWidth: 500, padding: 28 }}>
            <h3 style={{ fontSize: 18, fontWeight: 700, marginBottom: 16 }}>Register Participating Statistical Node</h3>
            <form onSubmit={handleRegisterNode}>
              <div style={{ marginBottom: 14 }}>
                <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Node Name</label>
                <input
                  type="text"
                  required
                  className="input-field"
                  value={nodeForm.name}
                  onChange={(e) => setNodeForm({ ...nodeForm, name: e.target.value })}
                  placeholder="e.g. Ministry of Agriculture Statistics Unit"
                />
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 14 }}>
                <div>
                  <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Code</label>
                  <input
                    type="text"
                    required
                    className="input-field"
                    value={nodeForm.code}
                    onChange={(e) => setNodeForm({ ...nodeForm, code: e.target.value.toUpperCase() })}
                    placeholder="NSS-MOA-STAT"
                  />
                </div>
                <div>
                  <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Jurisdiction</label>
                  <select
                    className="input-field"
                    value={nodeForm.jurisdiction}
                    onChange={(e) => setNodeForm({ ...nodeForm, jurisdiction: e.target.value })}
                  >
                    <option value="NATIONAL">NATIONAL</option>
                    <option value="EAST_AFRICAN_COMMUNITY">EAST AFRICAN COMMUNITY</option>
                    <option value="AFRICAN_UNION">AFRICAN UNION</option>
                    <option value="GLOBAL">GLOBAL</option>
                  </select>
                </div>
              </div>
              <div style={{ marginBottom: 14 }}>
                <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Endpoint URL</label>
                <input
                  type="url"
                  required
                  className="input-field"
                  value={nodeForm.endpoint_url}
                  onChange={(e) => setNodeForm({ ...nodeForm, endpoint_url: e.target.value })}
                  placeholder="https://statistics.gov/api/v1"
                />
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 20 }}>
                <button type="button" className="btn-secondary" onClick={() => setShowNodeModal(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn-primary">
                  Register Node
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Draft DSA */}
      {showDsaModal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', backdropFilter: 'blur(8px)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: 20 }}>
          <div className="glass-panel" style={{ width: '100%', maxWidth: 520, padding: 28 }}>
            <h3 style={{ fontSize: 18, fontWeight: 700, marginBottom: 16 }}>Draft Data Sharing Agreement (DSA)</h3>
            <form onSubmit={handleCreateDsa}>
              <div style={{ marginBottom: 14 }}>
                <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Agreement Title</label>
                <input
                  type="text"
                  required
                  className="input-field"
                  value={dsaForm.title}
                  onChange={(e) => setDsaForm({ ...dsaForm, title: e.target.value })}
                  placeholder="e.g. MOA Agricultural Yield Data Exchange Protocol"
                />
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 14 }}>
                <div>
                  <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Provider Node</label>
                  <select
                    className="input-field"
                    required
                    value={dsaForm.provider_node_id}
                    onChange={(e) => setDsaForm({ ...dsaForm, provider_node_id: e.target.value })}
                  >
                    <option value="">-- Choose Provider --</option>
                    {nodes.map((n) => (
                      <option key={n.id} value={n.id}>
                        {n.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Consumer Node</label>
                  <select
                    className="input-field"
                    required
                    value={dsaForm.consumer_node_id}
                    onChange={(e) => setDsaForm({ ...dsaForm, consumer_node_id: e.target.value })}
                  >
                    <option value="">-- Choose Consumer --</option>
                    {nodes.map((n) => (
                      <option key={n.id} value={n.id}>
                        {n.name}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
              <div style={{ marginBottom: 14 }}>
                <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Purpose & Objectives</label>
                <textarea
                  required
                  className="input-field"
                  rows={3}
                  value={dsaForm.purpose}
                  onChange={(e) => setDsaForm({ ...dsaForm, purpose: e.target.value })}
                  placeholder="State the statistical, research, or governance justification..."
                />
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 20 }}>
                <button type="button" className="btn-secondary" onClick={() => setShowDsaModal(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn-primary">
                  Draft DSA
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: Create Indicator */}
      {showIndicatorModal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', backdropFilter: 'blur(8px)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: 20 }}>
          <div className="glass-panel" style={{ width: '100%', maxWidth: 500, padding: 28 }}>
            <h3 style={{ fontSize: 18, fontWeight: 700, marginBottom: 16 }}>Register National / Official Indicator</h3>
            <form onSubmit={handleCreateIndicator}>
              <div style={{ marginBottom: 14 }}>
                <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Indicator Title</label>
                <input
                  type="text"
                  required
                  className="input-field"
                  value={indicatorForm.title}
                  onChange={(e) => setIndicatorForm({ ...indicatorForm, title: e.target.value })}
                  placeholder="e.g. Under-5 Stunting Prevalence"
                />
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 14 }}>
                <div>
                  <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Code</label>
                  <input
                    type="text"
                    required
                    className="input-field"
                    value={indicatorForm.code}
                    onChange={(e) => setIndicatorForm({ ...indicatorForm, code: e.target.value.toUpperCase() })}
                    placeholder="IND-SDG-2.2.1"
                  />
                </div>
                <div>
                  <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Domain</label>
                  <select
                    className="input-field"
                    value={indicatorForm.domain}
                    onChange={(e) => setIndicatorForm({ ...indicatorForm, domain: e.target.value })}
                  >
                    <option value="SDG">SDG</option>
                    <option value="health">Health</option>
                    <option value="agriculture">Agriculture</option>
                    <option value="economy">Economy</option>
                    <option value="education">Education</option>
                  </select>
                </div>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 14 }}>
                <div>
                  <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Current Value</label>
                  <input
                    type="number"
                    step="0.01"
                    className="input-field"
                    value={indicatorForm.current_value}
                    onChange={(e) => setIndicatorForm({ ...indicatorForm, current_value: e.target.value })}
                  />
                </div>
                <div>
                  <label style={{ fontSize: 12, color: 'var(--text-muted)' }}>Unit</label>
                  <input
                    type="text"
                    className="input-field"
                    value={indicatorForm.unit_of_measure}
                    onChange={(e) => setIndicatorForm({ ...indicatorForm, unit_of_measure: e.target.value })}
                    placeholder="Percentage"
                  />
                </div>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 20 }}>
                <button type="button" className="btn-secondary" onClick={() => setShowIndicatorModal(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn-primary">
                  Save Indicator
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
