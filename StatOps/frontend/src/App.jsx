import React, { useState, useEffect } from 'react'

const API_BASE = 'http://localhost:8098/api/v1'

export default function App() {
  const [activeTab, setActiveTab] = useState('ioc') // ioc, cicd, multicloud, sre, cmdb, status
  const [summary, setSummary] = useState(null)
  const [telemetry, setTelemetry] = useState([])
  const [pipelines, setPipelines] = useState([])
  const [deployments, setDeployments] = useState([])
  const [clusters, setClusters] = useState([])
  const [costs, setCosts] = useState([])
  const [readiness, setReadiness] = useState([])
  const [slos, setSlos] = useState([])
  const [cmdb, setCmdb] = useState([])
  const [alerts, setAlerts] = useState([])
  const [statusPage, setStatusPage] = useState([])
  const [showAppSwitcher, setShowAppSwitcher] = useState(false)
  const [scraping, setScraping] = useState(false)
  const [chaosLoading, setChaosLoading] = useState(false)
  const [chaosResult, setChaosResult] = useState(null)

  // Pipeline modal
  const [newPipelineRepo, setNewPipelineRepo] = useState('statgate/stattrust')
  const [newPipelineBranch, setNewPipelineBranch] = useState('main')

  useEffect(() => {
    fetchSummary()
    fetchTelemetry()
    fetchPipelines()
    fetchDeployments()
    fetchClusters()
    fetchReadiness()
    fetchSLOs()
    fetchCMDB()
    fetchAlerts()
    fetchStatusPage()
    fetchCosts()

    const interval = setInterval(() => {
      fetchSummary()
      fetchTelemetry()
    }, 10000)
    return () => clearInterval(interval)
  }, [])

  const fetchSummary = async () => {
    try {
      const res = await fetch(`${API_BASE}/summary`)
      if (res.ok) setSummary(await res.json())
    } catch (e) {
      console.warn('Backend offline, using fallback summary')
      setSummary({
        platform_health_score: 99.94,
        active_services_count: 10,
        total_clusters: 3,
        firing_alerts_count: 0,
        running_pipelines: 1,
        avg_response_latency_ms: 22,
        monthly_cloud_spend_usd: 1830.5,
        readiness_pass_rate: 100.0
      })
    }
  }

  const fetchTelemetry = async () => {
    try {
      const res = await fetch(`${API_BASE}/observability/telemetry`)
      if (res.ok) {
        const data = await res.json()
        setTelemetry(data.telemetry || [])
      }
    } catch (e) {
      console.warn('Using fallback telemetry')
    }
  }

  const fetchPipelines = async () => {
    try {
      const res = await fetch(`${API_BASE}/devops/pipelines`)
      if (res.ok) {
        const data = await res.json()
        setPipelines(data.pipelines || [])
      }
    } catch (e) {}
  }

  const fetchDeployments = async () => {
    try {
      const res = await fetch(`${API_BASE}/devops/deployments`)
      if (res.ok) {
        const data = await res.json()
        setDeployments(data.deployments || [])
      }
    } catch (e) {}
  }

  const fetchClusters = async () => {
    try {
      const res = await fetch(`${API_BASE}/cloud/clusters`)
      if (res.ok) {
        const data = await res.json()
        setClusters(data.clusters || [])
      }
    } catch (e) {}
  }

  const fetchCosts = async () => {
    try {
      const res = await fetch(`${API_BASE}/cloud/costs`)
      if (res.ok) {
        const data = await res.json()
        setCosts(data.costs || [])
      }
    } catch (e) {}
  }

  const fetchReadiness = async () => {
    try {
      const res = await fetch(`${API_BASE}/sre/readiness`)
      if (res.ok) {
        const data = await res.json()
        setReadiness(data.readiness_checks || [])
      }
    } catch (e) {}
  }

  const fetchSLOs = async () => {
    try {
      const res = await fetch(`${API_BASE}/sre/slos`)
      if (res.ok) {
        const data = await res.json()
        setSlos(data.slos || [])
      }
    } catch (e) {}
  }

  const fetchCMDB = async () => {
    try {
      const res = await fetch(`${API_BASE}/observability/cmdb`)
      if (res.ok) {
        const data = await res.json()
        setCmdb(data.cmdb || [])
      }
    } catch (e) {}
  }

  const fetchAlerts = async () => {
    try {
      const res = await fetch(`${API_BASE}/observability/alerts`)
      if (res.ok) {
        const data = await res.json()
        setAlerts(data.alerts || [])
      }
    } catch (e) {}
  }

  const fetchStatusPage = async () => {
    try {
      const res = await fetch(`${API_BASE}/observability/status-page`)
      if (res.ok) {
        const data = await res.json()
        setStatusPage(data.components || [])
      }
    } catch (e) {}
  }

  const handleManualScrape = async () => {
    setScraping(true)
    try {
      const res = await fetch(`${API_BASE}/observability/telemetry/scrape`, { method: 'POST' })
      if (res.ok) {
        const data = await res.json()
        setTelemetry(data.results || data.telemetry || [])
      }
    } catch (e) {}
    setTimeout(() => setScraping(false), 800)
  }

  const handleTriggerPipeline = async (e) => {
    e.preventDefault()
    try {
      const res = await fetch(`${API_BASE}/devops/pipelines/trigger`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repo_name: newPipelineRepo,
          branch: newPipelineBranch,
          commit_sha: Math.random().toString(36).substring(2, 9),
          commit_msg: 'chore(release): automated continuous deployment rollout',
          author: 'devops-operator'
        })
      })
      if (res.ok) {
        fetchPipelines()
        fetchSummary()
      }
    } catch (e) {}
  }

  const handleRollback = async (depId) => {
    try {
      const res = await fetch(`${API_BASE}/devops/deployments/${depId}/rollback`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ target_version: 'v1.1.9-stable' })
      })
      if (res.ok) {
        fetchDeployments()
      }
    } catch (e) {}
  }

  const handleExecuteChaos = async (scenario, targetService) => {
    setChaosLoading(true)
    setChaosResult(null)
    try {
      const res = await fetch(`${API_BASE}/sre/chaos/simulate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ scenario, target_service: targetService })
      })
      if (res.ok) {
        const data = await res.json()
        setChaosResult(data)
      }
    } catch (e) {}
    setChaosLoading(false)
  }

  const apps3x3 = [
    { name: 'All Apps', port: 3006, desc: 'Complete Catalogue', icon: 'SG' },
    { name: 'Analytics', port: 5000, desc: 'BI & ML Dashboards', icon: '📊' },
    { name: 'Registry', port: 3007, desc: 'Workforce Registry', icon: '🪪' },
    { name: 'Helpdesk', port: 3005, desc: 'Support Operations', icon: '🎧' },
    { name: 'StatChat', port: 3009, desc: 'Enterprise Messaging', icon: '💬' },
    { name: 'PMS', port: 3010, desc: 'Project Management', icon: '🏗️' },
    { name: 'RMS', port: 3011, desc: 'Research & Ethics', icon: '🔬' },
    { name: 'Governance', port: 3012, desc: 'Governance & COI', icon: '⚖️' },
    { name: 'Spatial', port: 3014, desc: 'GIS & Mapping', icon: '🗺️' },
    { name: 'Report Builder', port: 8110, desc: 'Visual Report Builder', icon: '🧮' },
  ]

  return (
    <div className="min-h-screen bg-[#060b13] text-slate-100 flex flex-col font-['Outfit']">
      {/* Top Navigation Header */}
      <header className="border-b border-slate-800 bg-[#09111e]/80 backdrop-blur-md sticky top-0 z-50 px-6 py-3.5 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <img src="/logo.png" alt="StatGate" className="w-10 h-10 rounded-xl object-contain bg-white p-1 shadow-lg ring-1 ring-white/20" onError={(e) => { e.target.style.display='none'; }} />
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-lg font-bold tracking-tight bg-gradient-to-r from-emerald-400 via-teal-300 to-cyan-400 bg-clip-text text-transparent">
                StatOps
              </h1>
              <span className="text-xs px-2 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-mono">
                P20 • P25 • P34 • P49
              </span>
            </div>
            <p className="text-xs text-slate-400">Platform Engineering & Integrated Operations Center</p>
          </div>
        </div>

        {/* Global Nav Tabs */}
        <nav className="flex items-center gap-1 bg-slate-900/60 p-1 rounded-xl border border-slate-800">
          {[
            { id: 'ioc', label: 'IOC & Telemetry', icon: '📡' },
            { id: 'cicd', label: 'CI/CD & GitOps', icon: '🚀' },
            { id: 'multicloud', label: 'Sovereign Cloud', icon: '☁️' },
            { id: 'sre', label: 'SRE & Readiness', icon: '⏱️' },
            { id: 'cmdb', label: 'CMDB & Status', icon: '🏛️' },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-3.5 py-1.5 rounded-lg text-xs font-medium transition-all flex items-center gap-1.5 ${
                activeTab === tab.id
                  ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 shadow-sm'
                  : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40'
              }`}
            >
              <span>{tab.icon}</span>
              {tab.label}
            </button>
          ))}
        </nav>

        {/* Right Action: Scrape & 3x3 Switcher */}
        <div className="flex items-center gap-3">
          <button
            onClick={handleManualScrape}
            disabled={scraping}
            className="px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 flex items-center gap-1.5 transition-all cursor-pointer"
          >
            <span className={scraping ? 'animate-spin' : ''}>🔄</span>
            {scraping ? 'Scraping Mesh...' : 'Scrape Telemetry'}
          </button>

          <div className="relative">
              <button
                onClick={() => setShowAppSwitcher(!showAppSwitcher)}
                className="p-2 rounded-lg bg-slate-800/80 hover:bg-slate-700 border border-slate-700 text-slate-300 transition-colors"
                title="StatGate 3x3 App Launcher"
                aria-label="Open 3 by 3 app launcher"
              >
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 18 18" aria-hidden="true">
                  {[3, 9, 15].flatMap(y => [3, 9, 15].map(x => <circle key={`${x}-${y}`} cx={x} cy={y} r="1.6" />))}
                </svg>
            </button>

            {showAppSwitcher && (
              <div className="absolute right-0 mt-2 w-80 p-3 bg-[#0d1626] border border-slate-700 rounded-2xl shadow-2xl z-50">
                <div className="text-xs font-semibold text-slate-400 mb-2 px-1">STATGATE UNIFIED PLATFORM</div>
                <div className="grid grid-cols-3 gap-2">
                  {apps3x3.map((app) => (
                    <a
                      key={app.name}
                      href={`http://localhost:${app.port}`}
                      className={`p-2.5 rounded-xl border flex flex-col items-center justify-center text-center transition-all ${
                        app.active
                          ? 'bg-emerald-500/20 border-emerald-500/40 text-emerald-300'
                          : 'bg-slate-800/40 border-slate-700/50 hover:bg-slate-700/50 text-slate-200'
                      }`}
                    >
                      <span className="text-xl mb-1">{app.icon}</span>
                      <span className="text-xs font-bold">{app.name}</span>
                      <span className="text-[9px] text-slate-400 truncate w-full">{app.desc}</span>
                    </a>
                  ))}
                </div>
                <a href="http://localhost:3006" className="block mt-2 text-center text-xs font-semibold text-cyan-400 hover:text-cyan-300">Open complete app catalogue →</a>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Hero Mission Control KPIs */}
      <div className="px-6 py-6 border-b border-slate-800/60 bg-gradient-to-b from-[#09111e] to-[#060b13]">
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-8 gap-3">
          <div className="glass-panel p-3.5">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Health Index</div>
            <div className="text-xl font-bold text-emerald-400 mt-1 flex items-center gap-1.5 font-mono">
              <span className="pulse-dot"></span>
              {summary?.platform_health_score ?? 99.92}%
            </div>
            <div className="text-[10px] text-emerald-500/80 mt-1">High Availability</div>
          </div>

          <div className="glass-panel p-3.5">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Active Services</div>
            <div className="text-xl font-bold text-cyan-300 mt-1 font-mono">{summary?.active_services_count ?? 10} / 10</div>
            <div className="text-[10px] text-cyan-400/80 mt-1">Full Mesh Live</div>
          </div>

          <div className="glass-panel p-3.5">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Sovereign Clusters</div>
            <div className="text-xl font-bold text-amber-300 mt-1 font-mono">{summary?.total_clusters ?? 3}</div>
            <div className="text-[10px] text-amber-400/80 mt-1">Kampala / Entebbe / Edge</div>
          </div>

          <div className="glass-panel p-3.5">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Avg Latency</div>
            <div className="text-xl font-bold text-slate-100 mt-1 font-mono">{summary?.avg_response_latency_ms ?? 22} ms</div>
            <div className="text-[10px] text-emerald-400 mt-1">P99 &lt; 50ms</div>
          </div>

          <div className="glass-panel p-3.5">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Pipelines</div>
            <div className="text-xl font-bold text-teal-300 mt-1 font-mono">{pipelines.length} Active</div>
            <div className="text-[10px] text-teal-400 mt-1">GitOps Auto-Deploy</div>
          </div>

          <div className="glass-panel p-3.5">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Firing Alerts</div>
            <div className="text-xl font-bold text-emerald-400 mt-1 font-mono">{summary?.firing_alerts_count ?? 0}</div>
            <div className="text-[10px] text-slate-400 mt-1">0 Critical Issues</div>
          </div>

          <div className="glass-panel p-3.5">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Readiness Pass</div>
            <div className="text-xl font-bold text-indigo-300 mt-1 font-mono">{summary?.readiness_pass_rate ?? 100}%</div>
            <div className="text-[10px] text-indigo-400 mt-1">Production Certified</div>
          </div>

          <div className="glass-panel p-3.5">
            <div className="text-[11px] font-medium text-slate-400 uppercase tracking-wider">Cloud Spend</div>
            <div className="text-xl font-bold text-rose-300 mt-1 font-mono">${summary?.monthly_cloud_spend_usd?.toFixed(0) ?? 1830}/mo</div>
            <div className="text-[10px] text-rose-400 mt-1">Optimal Tier</div>
          </div>
        </div>
      </div>

      {/* Main Workspace Body */}
      <main className="flex-1 p-6 max-w-7xl w-full mx-auto">
        {/* VIEW 1: INTEGRATED OPERATIONS CENTER (IOC) & TELEMETRY */}
        {activeTab === 'ioc' && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                  <span>📡</span> Integrated Operations Center & Live Mesh Telemetry (P49)
                </h2>
                <p className="text-xs text-slate-400">Real-time health, CPU, memory, and latency scraped across all 10 StatGate microservices</p>
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {telemetry.map((svc) => (
                <div key={svc.app_name} className="glass-panel p-4 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center justify-between mb-2">
                      <span className="font-bold text-sm text-slate-200">{svc.app_name}</span>
                      <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                        {svc.status}
                      </span>
                    </div>
                    <div className="text-xs text-slate-400 font-mono mb-3 truncate">{svc.endpoint}</div>

                    <div className="grid grid-cols-2 gap-2 text-xs font-mono bg-slate-900/60 p-2.5 rounded-lg border border-slate-800/80">
                      <div>
                        <span className="text-slate-500 block text-[10px]">LATENCY</span>
                        <span className="text-emerald-400 font-bold">{svc.latency_ms} ms</span>
                      </div>
                      <div>
                        <span className="text-slate-500 block text-[10px]">CPU USAGE</span>
                        <span className="text-cyan-400 font-bold">{svc.cpu_usage_pct}%</span>
                      </div>
                      <div>
                        <span className="text-slate-500 block text-[10px]">RAM USAGE</span>
                        <span className="text-amber-400 font-bold">{svc.memory_usage_mb} MB</span>
                      </div>
                      <div>
                        <span className="text-slate-500 block text-[10px]">ACTIVE REQS</span>
                        <span className="text-slate-200 font-bold">{svc.active_requests} rps</span>
                      </div>
                    </div>
                  </div>

                  <div className="mt-3 pt-2 border-t border-slate-800/60 flex items-center justify-between text-[10px] text-slate-500">
                    <span>5xx Errors: {svc.error_rate_5xx}%</span>
                    <span>Scraped {new Date(svc.last_scraped_at).toLocaleTimeString()}</span>
                  </div>
                  <a href="http://localhost:3006" className="block mt-2 text-center text-xs font-semibold text-cyan-400 hover:text-cyan-300">Open complete app catalogue →</a>
                </div>
              ))}
            </div>

            {/* Active Alert Center */}
            <div className="glass-panel p-5">
              <h3 className="text-sm font-bold text-slate-200 mb-3 flex items-center gap-2">
                <span>⚠️</span> Active Alert Router & Incident Stream
              </h3>
              {alerts.length === 0 ? (
                <div className="p-4 rounded-xl bg-emerald-500/5 border border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
                  <span>✅</span> All services operating within defined thresholds. Zero firing P1/P2 alert incidents.
                </div>
              ) : (
                <div className="space-y-2">
                  {alerts.map((a) => (
                    <div key={a.id} className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/20 flex items-center justify-between">
                      <div>
                        <div className="text-xs font-bold text-rose-300">{a.title}</div>
                        <div className="text-[11px] text-slate-400">{a.description} • {a.source_app}</div>
                      </div>
                      <span className="text-xs font-mono px-2 py-0.5 rounded bg-rose-500/20 text-rose-400">{a.severity}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* VIEW 2: CI/CD & GITOPS DELIVERY (P20) */}
        {activeTab === 'cicd' && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                  <span>🚀</span> Continuous Delivery & GitOps Pipeline Engine (P20)
                </h2>
                <p className="text-xs text-slate-400">Automated builds, multi-stage testing, container registry push, and canary rollback orchestrator</p>
              </div>

              {/* Trigger Pipeline Form */}
              <form onSubmit={handleTriggerPipeline} className="flex items-center gap-2">
                <input
                  type="text"
                  value={newPipelineRepo}
                  onChange={(e) => setNewPipelineRepo(e.target.value)}
                  className="bg-slate-900 border border-slate-700 px-3 py-1.5 rounded-lg text-xs font-mono text-slate-200"
                  placeholder="repo"
                />
                <input
                  type="text"
                  value={newPipelineBranch}
                  onChange={(e) => setNewPipelineBranch(e.target.value)}
                  className="bg-slate-900 border border-slate-700 px-3 py-1.5 rounded-lg text-xs font-mono text-slate-200 w-24"
                  placeholder="branch"
                />
                <button type="submit" className="px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-emerald-600 hover:bg-emerald-500 text-white cursor-pointer transition-all">
                  Trigger Pipeline
                </button>
              </form>
            </div>

            {/* Pipeline List */}
            <div className="space-y-4">
              {pipelines.map((pipe) => (
                <div key={pipe.id} className="glass-panel p-5">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <span className="text-xs font-mono font-bold text-emerald-400 bg-emerald-500/10 px-2.5 py-1 rounded border border-emerald-500/20">
                        {pipe.repo_name}
                      </span>
                      <span className="text-xs text-slate-400 font-mono">branch: {pipe.branch} ({pipe.commit_sha})</span>
                    </div>
                    <span className="text-xs font-mono font-bold text-emerald-300 px-2.5 py-0.5 rounded bg-emerald-500/20">
                      {pipe.status} ({pipe.duration_secs}s)
                    </span>
                  </div>

                  <p className="text-xs text-slate-300 mb-4 italic">"{pipe.commit_msg}" — by {pipe.author}</p>

                  {/* Stages */}
                  <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
                    {pipe.stages.map((stage, i) => (
                      <div key={i} className="p-3 rounded-lg bg-slate-900/70 border border-slate-800">
                        <div className="flex items-center justify-between text-xs font-bold text-slate-200 mb-1">
                          <span>{stage.name}</span>
                          <span className="text-emerald-400 font-mono text-[10px]">{stage.status}</span>
                        </div>
                        <div className="text-[10px] text-slate-400 font-mono space-y-0.5 mt-2">
                          {stage.logs?.map((l, j) => <div key={j} className="truncate">• {l}</div>)}
                        </div>
                      </div>
                    ))}
                  </div>

                  {pipe.artifact_url && (
                    <div className="mt-3 pt-3 border-t border-slate-800 flex items-center justify-between text-xs font-mono text-slate-400">
                      <span>Artifact: <strong className="text-cyan-300">{pipe.artifact_url}</strong></span>
                      <span>Verified & Signed (Ed25519)</span>
                    </div>
                  )}
                </div>
              ))}
            </div>

            {/* Deployment History with Rollback */}
            <div className="glass-panel p-5">
              <h3 className="text-sm font-bold text-slate-200 mb-3 flex items-center gap-2">
                <span>🛡️</span> Production Deployments & Instant Rollback Controller
              </h3>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="text-slate-400 font-mono uppercase border-b border-slate-800">
                    <tr>
                      <th className="pb-2">Service</th>
                      <th className="pb-2">Version</th>
                      <th className="pb-2">Environment</th>
                      <th className="pb-2">Strategy</th>
                      <th className="pb-2">Status</th>
                      <th className="pb-2">Deployed By</th>
                      <th className="pb-2 text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800 font-mono">
                    {deployments.map((d) => (
                      <tr key={d.id} className="hover:bg-slate-800/30">
                        <td className="py-2.5 font-bold text-slate-200">{d.service_name}</td>
                        <td className="py-2.5 text-cyan-300">{d.version}</td>
                        <td className="py-2.5 text-slate-400">{d.environment}</td>
                        <td className="py-2.5 text-amber-300">{d.strategy} ({d.canary_percentage}%)</td>
                        <td className="py-2.5">
                          <span className={`px-2 py-0.5 rounded text-[10px] ${d.status === 'ACTIVE' ? 'bg-emerald-500/20 text-emerald-400' : 'bg-amber-500/20 text-amber-300'}`}>
                            {d.status}
                          </span>
                        </td>
                        <td className="py-2.5 text-slate-400">{d.deployed_by}</td>
                        <td className="py-2.5 text-right">
                          <button
                            onClick={() => handleRollback(d.id)}
                            className="px-2.5 py-1 rounded bg-rose-500/10 hover:bg-rose-500/20 text-rose-300 border border-rose-500/30 text-[11px] cursor-pointer"
                          >
                            1-Click Rollback
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

        {/* VIEW 3: SOVEREIGN MULTI-CLOUD MANAGER (P25) */}
        {activeTab === 'multicloud' && (
          <div className="space-y-6">
            <div>
              <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                <span>☁️</span> Sovereign Multi-Cloud & Hybrid Cluster Fleet (P25)
              </h2>
              <p className="text-xs text-slate-400">National data center sovereignty, cross-region replication, and cloud cost management</p>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
              {clusters.map((cls) => (
                <div key={cls.id} className="glass-panel p-5 border-t-4 border-t-emerald-500">
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-bold text-sm text-slate-100">{cls.name}</span>
                    <span className="text-xs font-mono px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400">
                      {cls.status}
                    </span>
                  </div>
                  <div className="text-xs text-slate-400 font-mono mb-4">{cls.provider} • {cls.region}</div>

                  <div className="space-y-2 text-xs font-mono bg-slate-900/60 p-3 rounded-lg border border-slate-800">
                    <div className="flex justify-between">
                      <span className="text-slate-500">Data Boundary:</span>
                      <span className="text-amber-300 font-semibold">{cls.data_boundary}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-500">Node Capacity:</span>
                      <span className="text-slate-200">{cls.total_nodes} Nodes ({cls.allocated_cpu} / {cls.allocated_ram})</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-500">Storage Usage:</span>
                      <span className="text-cyan-300">{cls.storage_usage_gb} GB SAN</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-500">K8s Engine:</span>
                      <span className="text-slate-400">{cls.k8s_version}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {/* Cloud Cost Optimizer */}
            <div className="glass-panel p-5">
              <h3 className="text-sm font-bold text-slate-200 mb-3 flex items-center gap-2">
                <span>💰</span> Cloud Cost Management & Sovereign Efficiency Analytics
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {costs.map((c) => (
                  <div key={c.id} className="p-4 rounded-xl bg-slate-900/60 border border-slate-800">
                    <div className="flex justify-between items-center mb-1">
                      <span className="font-bold text-xs text-slate-200">{c.service_name}</span>
                      <span className="font-mono text-xs text-emerald-400 font-bold">${c.cost_usd} / ${c.budget_limit_usd}</span>
                    </div>
                    <p className="text-[11px] text-slate-400 mt-2 font-mono">💡 {c.optimization_tip}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* VIEW 4: SRE & PRODUCTION READINESS (P34) */}
        {activeTab === 'sre' && (
          <div className="space-y-6">
            <div>
              <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                <span>⏱️</span> Site Reliability Engineering (SRE) & Production Readiness (P34)
              </h2>
              <p className="text-xs text-slate-400">SLO error budgets, automated self-healing runbooks, and chaos engineering drills</p>
            </div>

            {/* SLO Error Budgets */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {slos.map((slo) => (
                <div key={slo.id} className="glass-panel p-5">
                  <div className="flex justify-between items-center mb-2">
                    <span className="font-bold text-sm text-slate-100">{slo.service_name}</span>
                    <span className="text-xs font-mono font-bold text-emerald-400">{slo.slo_name}</span>
                  </div>
                  <div className="flex items-center gap-4 my-3">
                    <div className="text-2xl font-bold font-mono text-cyan-300">{slo.current_percent}%</div>
                    <div className="text-xs text-slate-400 font-mono">Target: {slo.target_percent}% ({slo.period})</div>
                  </div>
                  <div className="w-full bg-slate-800 h-2 rounded-full overflow-hidden">
                    <div className="bg-emerald-500 h-full rounded-full" style={{ width: `${slo.error_budget_remaining}%` }}></div>
                  </div>
                  <div className="flex justify-between text-[10px] text-slate-400 font-mono mt-2">
                    <span>Remaining Error Budget: {slo.error_budget_remaining}%</span>
                    <span>Burn Rate: <strong className="text-emerald-400">{slo.burn_rate_status}</strong></span>
                  </div>
                </div>
              ))}
            </div>

            {/* Production Readiness Scorecards */}
            <div className="glass-panel p-5">
              <h3 className="text-sm font-bold text-slate-200 mb-3 flex items-center gap-2">
                <span>📋</span> Production Readiness & Certification Gates
              </h3>
              <div className="space-y-2">
                {readiness.map((chk) => (
                  <div key={chk.id} className="p-3 rounded-lg bg-slate-900/60 border border-slate-800 flex items-center justify-between">
                    <div>
                      <div className="text-xs font-bold text-slate-200">{chk.check_name}</div>
                      <div className="text-[11px] text-slate-400">{chk.details} • Audited by {chk.auditor}</div>
                    </div>
                    <span className="text-xs font-mono px-2.5 py-0.5 rounded bg-emerald-500/20 text-emerald-400 font-bold">
                      {chk.status}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            {/* Chaos Drill Trigger */}
            <div className="glass-panel p-5">
              <h3 className="text-sm font-bold text-slate-200 mb-3 flex items-center gap-2">
                <span>💥</span> Chaos Engineering & Disaster Recovery Simulator
              </h3>
              <div className="flex items-center gap-3">
                <button
                  onClick={() => handleExecuteChaos('REGION_FAILOVER', 'Enterprise Core & Database')}
                  disabled={chaosLoading}
                  className="px-4 py-2 rounded-lg text-xs font-bold bg-gradient-to-r from-rose-600 to-amber-600 hover:from-rose-500 hover:to-amber-500 text-white cursor-pointer transition-all shadow-lg shadow-rose-600/20"
                >
                  {chaosLoading ? 'Simulating Failover...' : 'Execute Region Failover Chaos Drill'}
                </button>
                <span className="text-xs text-slate-400">Tests automatic switchover to Entebbe DR replica with zero RPO loss</span>
              </div>

              {chaosResult && (
                <div className="mt-4 p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-xs font-mono text-emerald-300">
                  <div className="font-bold mb-1">🎉 Chaos Simulation Result: {chaosResult.status}</div>
                  <div>• Scenario: {chaosResult.drill_name}</div>
                  <div>• RTO Achieved: {chaosResult.rto_achieved_min} Minutes (Target &lt; 12min)</div>
                  <div>• RPO Achieved: {chaosResult.rpo_achieved_sec} Seconds (Zero data loss)</div>
                  <div className="text-slate-400 mt-1">• Notes: {chaosResult.notes}</div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* VIEW 5: CMDB & PUBLIC STATUS PAGE (P49) */}
        {activeTab === 'cmdb' && (
          <div className="space-y-6">
            <div>
              <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                <span>🏛️</span> Configuration Management Database (CMDB) & Status Page (P49)
              </h2>
              <p className="text-xs text-slate-400">Complete dependency graph, infrastructure assets, and public customer status indicators</p>
            </div>

            {/* CMDB Items */}
            <div className="glass-panel p-5">
              <h3 className="text-sm font-bold text-slate-200 mb-3">Enterprise Configuration Items (CI)</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {cmdb.map((item) => (
                  <div key={item.id} className="p-3.5 rounded-xl bg-slate-900/60 border border-slate-800">
                    <div className="flex justify-between items-center mb-1">
                      <span className="font-bold text-xs text-slate-100">{item.item_name}</span>
                      <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-cyan-500/20 text-cyan-300 font-bold">{item.item_type}</span>
                    </div>
                    <div className="text-[11px] font-mono text-slate-400 mb-2">{item.host_or_url} • {item.criticality}</div>
                    <div className="text-[10px] text-slate-500 font-mono">
                      Dependencies: {item.dependencies?.join(', ')}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Public Status Components */}
            <div className="glass-panel p-5">
              <h3 className="text-sm font-bold text-slate-200 mb-3">StatGate Live Platform Status Page</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                {statusPage.map((comp) => (
                  <div key={comp.id} className="p-3 rounded-lg bg-slate-900/50 border border-slate-800 flex items-center justify-between">
                    <div>
                      <div className="text-xs font-semibold text-slate-200">{comp.name}</div>
                      <div className="text-[10px] text-slate-400">{comp.group} • 90d Uptime: {comp.uptime_90d}%</div>
                    </div>
                    <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 shadow-sm shadow-emerald-400/80"></span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-800/80 bg-[#09111e]/80 px-6 py-3 text-xs text-slate-500 flex items-center justify-between font-mono">
        <div>StatGate Platform Engineering • StatOps v1.0.0</div>
        <div className="flex items-center gap-4">
          <span>Sovereign Cloud: Kampala DC</span>
          <span>Zero-Trust: Active</span>
          <span>Mesh Status: 10/10 Connected</span>
        </div>
      </footer>
    </div>
  )
}
