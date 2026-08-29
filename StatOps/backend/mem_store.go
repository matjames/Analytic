package main

import (
	"fmt"
	"sync"
	"time"
)

type MemStore struct {
	mu          sync.RWMutex
	pipelines   map[string]PipelineRun
	deployments map[string]DeploymentRecord
	releases    map[string]ReleaseTrain
	configs     map[string]EnvironmentConfig
	clusters    map[string]CloudCluster
	costs       []CloudCostRecord
	readiness   map[string]ProductionReadinessCheck
	slos        map[string]SLOBudget
	runbooks    []RunbookExecution
	drDrills    []DisasterRecoveryDrill
	telemetry   map[string]ServiceTelemetry
	logs        []CentralizedLog
	traces      []DistributedTraceSpan
	cmdb        map[string]CMDBItem
	alerts      map[string]AlertIncident
	statusComps map[string]PublicStatusComponent
}

var globalStore *MemStore

func NewMemStore() *MemStore {
	store := &MemStore{
		pipelines:   make(map[string]PipelineRun),
		deployments: make(map[string]DeploymentRecord),
		releases:    make(map[string]ReleaseTrain),
		configs:     make(map[string]EnvironmentConfig),
		clusters:    make(map[string]CloudCluster),
		costs:       make([]CloudCostRecord, 0),
		readiness:   make(map[string]ProductionReadinessCheck),
		slos:        make(map[string]SLOBudget),
		runbooks:    make([]RunbookExecution, 0),
		drDrills:    make([]DisasterRecoveryDrill, 0),
		telemetry:   make(map[string]ServiceTelemetry),
		logs:        make([]CentralizedLog, 0),
		traces:      make([]DistributedTraceSpan, 0),
		cmdb:        make(map[string]CMDBItem),
		alerts:      make(map[string]AlertIncident),
		statusComps: make(map[string]PublicStatusComponent),
	}
	store.seedInitialData()
	return store
}

func (s *MemStore) seedInitialData() {
	now := time.Now().UTC()

	// 1. Seed Telemetry for 10 Platform Peer Services
	services := []ServiceTelemetry{
		{AppName: "StatGate App Launcher", Port: 3006, Endpoint: "http://applauncher:3006", Status: "UP", LatencyMs: 14, CPUUsagePct: 12.4, MemoryUsageMB: 210, UptimeSecs: 864000, ActiveRequests: 48, ErrorRate5xx: 0.00, LastScrapedAt: now},
		{AppName: "Enterprise Core", Port: 8096, Endpoint: "http://statgate-enterprise-core:8096", Status: "UP", LatencyMs: 22, CPUUsagePct: 28.1, MemoryUsageMB: 480, UptimeSecs: 864000, ActiveRequests: 135, ErrorRate5xx: 0.01, LastScrapedAt: now},
		{AppName: "StatTrust (Security & Trust)", Port: 8094, Endpoint: "http://statgate-trust-api:8080", Status: "UP", LatencyMs: 18, CPUUsagePct: 15.6, MemoryUsageMB: 195, UptimeSecs: 864000, ActiveRequests: 62, ErrorRate5xx: 0.00, LastScrapedAt: now},
		{AppName: "PMS (Project Management)", Port: 8091, Endpoint: "http://statgate-pms-api:8080", Status: "UP", LatencyMs: 25, CPUUsagePct: 19.8, MemoryUsageMB: 310, UptimeSecs: 864000, ActiveRequests: 84, ErrorRate5xx: 0.00, LastScrapedAt: now},
		{AppName: "RMS (Research Management)", Port: 8092, Endpoint: "http://statgate-rms-api:8080", Status: "UP", LatencyMs: 20, CPUUsagePct: 18.2, MemoryUsageMB: 290, UptimeSecs: 864000, ActiveRequests: 55, ErrorRate5xx: 0.00, LastScrapedAt: now},
		{AppName: "StatGovernance", Port: 8093, Endpoint: "http://statgate-governance-api:8080", Status: "UP", LatencyMs: 16, CPUUsagePct: 11.3, MemoryUsageMB: 180, UptimeSecs: 864000, ActiveRequests: 28, ErrorRate5xx: 0.00, LastScrapedAt: now},
		{AppName: "StatSpatial (GIS Engine)", Port: 4200, Endpoint: "http://statspatial:4200", Status: "UP", LatencyMs: 38, CPUUsagePct: 34.5, MemoryUsageMB: 650, UptimeSecs: 864000, ActiveRequests: 92, ErrorRate5xx: 0.02, LastScrapedAt: now},
		{AppName: "StatChat (Collaboration)", Port: 4000, Endpoint: "http://statchat-backend:4000", Status: "UP", LatencyMs: 12, CPUUsagePct: 22.0, MemoryUsageMB: 340, UptimeSecs: 864000, ActiveRequests: 210, ErrorRate5xx: 0.00, LastScrapedAt: now},
		{AppName: "Field Registry", Port: 9090, Endpoint: "http://statgate-registry-api:9090", Status: "UP", LatencyMs: 19, CPUUsagePct: 14.8, MemoryUsageMB: 220, UptimeSecs: 864000, ActiveRequests: 76, ErrorRate5xx: 0.00, LastScrapedAt: now},
		{AppName: "Analytics Core", Port: 8082, Endpoint: "http://statgate-backend:8082", Status: "UP", LatencyMs: 45, CPUUsagePct: 41.2, MemoryUsageMB: 820, UptimeSecs: 864000, ActiveRequests: 115, ErrorRate5xx: 0.01, LastScrapedAt: now},
	}
	for _, svc := range services {
		s.telemetry[svc.AppName] = svc
	}

	// 2. Seed Multi-Cloud & Sovereign Clusters (P25)
	c1 := CloudCluster{
		ID: "cls-gov-kampala-01", Name: "NITA-U National Data Center Primary", Provider: "NITA_U_GOVCLOUD", Region: "kampala-central",
		Status: "HEALTHY", DataBoundary: "NATIONAL_SOVEREIGN", TotalNodes: 12, AllocatedCPU: "48 vCPU", AllocatedRAM: "192 GB",
		StorageUsageGB: 2450.5, K8sVersion: "v1.30.2", LastHeartbeat: now,
	}
	c2 := CloudCluster{
		ID: "cls-gov-entebbe-dr", Name: "Entebbe Secondary DR Vault Cluster", Provider: "ON_PREM_DC", Region: "entebbe-dr",
		Status: "HEALTHY", DataBoundary: "NATIONAL_SOVEREIGN", TotalNodes: 8, AllocatedCPU: "32 vCPU", AllocatedRAM: "128 GB",
		StorageUsageGB: 1890.0, K8sVersion: "v1.30.2", LastHeartbeat: now,
	}
	c3 := CloudCluster{
		ID: "cls-cloud-af-south-1", Name: "AWS Africa (Cape Town) Edge Replica", Provider: "AWS_AFRICA_SOUTH", Region: "af-south-1",
		Status: "HEALTHY", DataBoundary: "REGIONAL_RESTRICTED", TotalNodes: 6, AllocatedCPU: "24 vCPU", AllocatedRAM: "96 GB",
		StorageUsageGB: 980.2, K8sVersion: "v1.30.2", LastHeartbeat: now,
	}
	s.clusters[c1.ID] = c1
	s.clusters[c2.ID] = c2
	s.clusters[c3.ID] = c3

	// 3. Seed CI/CD Pipelines (P20)
	p1 := PipelineRun{
		ID: "pipe-2026-0816-01", TenantID: "tenant-alpha", RepoName: "statgate/stattrust", Branch: "main",
		CommitSHA: "9f8a12d", CommitMsg: "feat: add RFC 3161 Timestamp Authority and PKI CA endpoints", Author: "matjames",
		Status: "SUCCESS", StartedAt: now.Add(-35 * time.Minute), FinishedAt: &now, DurationSecs: 142,
		ArtifactURL: "registry.statgate.gov.ug/stattrust:v1.2.0",
		Stages: []PipelineStage{
			{Name: "Lint & CodeSec Scan", Status: "SUCCESS", DurationSecs: 18, Logs: []string{"gofmt checked", "govulncheck clean", "semgrep 0 findings"}},
			{Name: "Unit & Integration Tests", Status: "SUCCESS", DurationSecs: 42, Logs: []string{"=== RUN TestHealthEndpoint", "PASS: 9/9 passed"}},
			{Name: "Container Build", Status: "SUCCESS", DurationSecs: 55, Logs: []string{"Docker multi-stage build complete", "image size: 28.4MB"}},
			{Name: "Artifact Registry Push", Status: "SUCCESS", DurationSecs: 27, Logs: []string{"Pushed sha256:7bc89d to Sovereign Registry"}},
		},
	}
	s.pipelines[p1.ID] = p1

	// 4. Seed Deployments (P20)
	d1 := DeploymentRecord{
		ID: "dep-2026-0816-001", TenantID: "tenant-alpha", ServiceName: "StatTrust", Version: "v1.2.0",
		Environment: "PRODUCTION_SOVEREIGN", Strategy: "CANARY", TargetCluster: "cls-gov-kampala-01",
		Status: "ACTIVE", CanaryPercentage: 100, DeployedBy: "sre-lead@statgate.gov.ug", DeployedAt: now.Add(-25 * time.Minute),
	}
	s.deployments[d1.ID] = d1

	// 5. Seed SLO Budgets (P34)
	slo1 := SLOBudget{
		ID: "slo-api-avail", ServiceName: "StatGate Unified Gateway", SLOName: "Platform Availability",
		TargetPercent: 99.95, CurrentPercent: 99.98, ErrorBudgetRemaining: 74.2, BurnRateStatus: "NORMAL",
		Period: "30d", LastCalculated: now,
	}
	slo2 := SLOBudget{
		ID: "slo-spatial-lat", ServiceName: "StatSpatial GIS Engine", SLOName: "P95 Query Latency (<150ms)",
		TargetPercent: 99.0, CurrentPercent: 99.4, ErrorBudgetRemaining: 68.0, BurnRateStatus: "NORMAL",
		Period: "30d", LastCalculated: now,
	}
	s.slos[slo1.ID] = slo1
	s.slos[slo2.ID] = slo2

	// 6. Seed Production Readiness Checks (P34)
	checks := []ProductionReadinessCheck{
		{ID: "chk-01", Category: "SECURITY", CheckName: "Zero-Trust JWT & Tenant Isolation Enforced", Status: "PASSED", Details: "All v1 routes validate token signature & X-Tenant-ID", Auditor: "SecOps Lead", LastAuditAt: now.Add(-2 * time.Hour)},
		{ID: "chk-02", Category: "INFRASTRUCTURE", CheckName: "Multi-Cloud Sovereign HA Replication Active", Status: "PASSED", Details: "PostgreSQL streaming replication to Entebbe DR verified", Auditor: "Infrastructure Lead", LastAuditAt: now.Add(-4 * time.Hour)},
		{ID: "chk-03", Category: "PERFORMANCE", CheckName: "P99 Response Latency Under 200ms Benchmark", Status: "PASSED", Details: "Load tested at 2,500 concurrent rps — max P99 was 142ms", Auditor: "Performance SRE", LastAuditAt: now.Add(-6 * time.Hour)},
		{ID: "chk-04", Category: "BACKUP_DR", CheckName: "Automated Nightly Backup & Checksum Verification", Status: "PASSED", Details: "RPO verified at 15min; RTO under 12 minutes", Auditor: "BCM Coordinator", LastAuditAt: now.Add(-12 * time.Hour)},
		{ID: "chk-05", Category: "DOCUMENTATION", CheckName: "Production Runbooks & Disaster Drill Playbooks", Status: "PASSED", Details: "12 automated runbooks tested with rollback paths", Auditor: "Platform Architect", LastAuditAt: now.Add(-24 * time.Hour)},
	}
	for _, chk := range checks {
		s.readiness[chk.ID] = chk
	}

	// 7. Seed CMDB Items (P49)
	items := []CMDBItem{
		{ID: "ci-db-cluster", ItemName: "PostgreSQL Sovereign Cluster", ItemType: "DATABASE", Environment: "PRODUCTION_SOVEREIGN", HostOrURL: "postgres:5432", OwnerTeam: "Data Ops", Criticality: "TIER_0_MISSION_CRITICAL", Dependencies: []string{"NITA-U SAN Storage"}, Status: "OPERATIONAL", LastUpdated: now},
		{ID: "ci-redis-bus", ItemName: "Redis Enterprise Event Mesh", ItemType: "REDIS_CLUSTER", Environment: "PRODUCTION_SOVEREIGN", HostOrURL: "redis:6379", OwnerTeam: "Platform Core", Criticality: "TIER_0_MISSION_CRITICAL", Dependencies: []string{"statgate-network"}, Status: "OPERATIONAL", LastUpdated: now},
		{ID: "ci-svc-core", ItemName: "StatGate Enterprise Core API", ItemType: "MICROSERVICE", Environment: "PRODUCTION_SOVEREIGN", HostOrURL: "statgate-enterprise-core:8096", OwnerTeam: "Enterprise Team", Criticality: "TIER_0_MISSION_CRITICAL", Dependencies: []string{"ci-db-cluster", "ci-redis-bus"}, Status: "OPERATIONAL", LastUpdated: now},
		{ID: "ci-svc-trust", ItemName: "StatTrust Security & Digital Trust API", ItemType: "MICROSERVICE", Environment: "PRODUCTION_SOVEREIGN", HostOrURL: "statgate-trust-api:8080", OwnerTeam: "SecOps / SRE", Criticality: "TIER_0_MISSION_CRITICAL", Dependencies: []string{"ci-db-cluster", "ci-redis-bus"}, Status: "OPERATIONAL", LastUpdated: now},
	}
	for _, ci := range items {
		s.cmdb[ci.ID] = ci
	}

	// 8. Seed Public Status Components (P49)
	statusList := []PublicStatusComponent{
		{ID: "comp-auth", Name: "Identity & Access Gateway", Group: "Core Services", Status: "OPERATIONAL", Uptime90d: 99.99, UpdatedAt: now},
		{ID: "comp-trust", Name: "Digital Trust & Merkle Ledger", Group: "Trust & Governance", Status: "OPERATIONAL", Uptime90d: 99.98, UpdatedAt: now},
		{ID: "comp-pms", Name: "Project Management System (PMS)", Group: "Core Services", Status: "OPERATIONAL", Uptime90d: 99.95, UpdatedAt: now},
		{ID: "comp-rms", Name: "Research Management System (RMS)", Group: "Core Services", Status: "OPERATIONAL", Uptime90d: 99.96, UpdatedAt: now},
		{ID: "comp-spatial", Name: "StatSpatial GIS Mapping Engine", Group: "Intelligence & GIS", Status: "OPERATIONAL", Uptime90d: 99.92, UpdatedAt: now},
		{ID: "comp-collect", Name: "Field Data Collection API", Group: "Data Collection", Status: "OPERATIONAL", Uptime90d: 99.97, UpdatedAt: now},
	}
	for _, comp := range statusList {
		s.statusComps[comp.ID] = comp
	}

	// 9. Seed Cloud Costs (P25)
	s.costs = []CloudCostRecord{
		{ID: "cost-01", MonthYear: "2026-08", Provider: "NITA_U_GOVCLOUD", ServiceName: "National Sovereign Cloud Host", CostUSD: 1450.00, BudgetLimitUSD: 2000.00, OptimizationTip: "Reserved instance quota optimal"},
		{ID: "cost-02", MonthYear: "2026-08", Provider: "AWS_AFRICA_SOUTH", ServiceName: "Edge CDN & S3 Backup Vault", CostUSD: 380.50, BudgetLimitUSD: 600.00, OptimizationTip: "Lifecycle policy archiving backups >90d to Glacier Deep Archive saved $120/mo"},
	}
}

// Getters and Mutators

func (s *MemStore) GetOverviewSummary() OperationsOverviewSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	firingAlerts := 0
	for _, a := range s.alerts {
		if a.Status == "FIRING" {
			firingAlerts++
		}
	}

	runningPipes := 0
	for _, p := range s.pipelines {
		if p.Status == "RUNNING" {
			runningPipes++
		}
	}

	totalLat := 0
	count := 0
	for _, t := range s.telemetry {
		totalLat += t.LatencyMs
		count++
	}
	avgLat := 20
	if count > 0 {
		avgLat = totalLat / count
	}

	passedReadiness := 0
	for _, r := range s.readiness {
		if r.Status == "PASSED" {
			passedReadiness++
		}
	}
	readinessRate := 100.0
	if len(s.readiness) > 0 {
		readinessRate = (float64(passedReadiness) / float64(len(s.readiness))) * 100.0
	}

	totalSpend := 0.0
	for _, c := range s.costs {
		totalSpend += c.CostUSD
	}

	return OperationsOverviewSummary{
		PlatformHealthScore:  99.92,
		ActiveServicesCount:  len(s.telemetry),
		TotalClusters:        len(s.clusters),
		FiringAlertsCount:    firingAlerts,
		RunningPipelines:     runningPipes,
		AvgResponseLatencyMs: avgLat,
		MonthlyCloudSpendUSD: totalSpend,
		ReadinessPassRate:    readinessRate,
	}
}

func (s *MemStore) ListTelemetry() []ServiceTelemetry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]ServiceTelemetry, 0, len(s.telemetry))
	for _, v := range s.telemetry {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) UpdateTelemetry(t ServiceTelemetry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t.LastScrapedAt = time.Now().UTC()
	s.telemetry[t.AppName] = t
}

func (s *MemStore) ListClusters() []CloudCluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]CloudCluster, 0, len(s.clusters))
	for _, v := range s.clusters {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) ListPipelines(tenantID, workspaceID string) []PipelineRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]PipelineRun, 0, len(s.pipelines))
	for _, v := range s.pipelines {
		if (tenantID == "" || v.TenantID == tenantID) && (workspaceID == "" || v.WorkspaceID == workspaceID) {
			res = append(res, v)
		}
	}
	return res
}

func (s *MemStore) TriggerPipeline(req PipelineRun) PipelineRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if req.ID == "" {
		req.ID = fmt.Sprintf("pipe-%s-%d", req.RepoName, now.Unix())
	}
	req.StartedAt = now
	req.Status = "RUNNING"
	req.Stages = []PipelineStage{
		{Name: "Lint & CodeSec Scan", Status: "SUCCESS", DurationSecs: 12, Logs: []string{"Zero critical security findings"}},
		{Name: "Unit & Integration Tests", Status: "SUCCESS", DurationSecs: 34, Logs: []string{"All unit test suites PASSED"}},
		{Name: "Container Build", Status: "RUNNING", Logs: []string{"Compiling multi-tenant image"}},
	}
	s.pipelines[req.ID] = req
	return req
}

func (s *MemStore) ListDeployments(tenantID, workspaceID string) []DeploymentRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]DeploymentRecord, 0, len(s.deployments))
	for _, v := range s.deployments {
		if (tenantID == "" || v.TenantID == tenantID) && (workspaceID == "" || v.WorkspaceID == workspaceID) {
			res = append(res, v)
		}
	}
	return res
}

func (s *MemStore) CreateDeployment(d DeploymentRecord) DeploymentRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d.ID == "" {
		d.ID = fmt.Sprintf("dep-%d", time.Now().Unix())
	}
	if d.DeployedAt.IsZero() {
		d.DeployedAt = time.Now().UTC()
	}
	d.Status = "ACTIVE"
	s.deployments[d.ID] = d
	return d
}

func (s *MemStore) RollbackDeployment(id, targetVersion, actor string) (DeploymentRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dep, ok := s.deployments[id]
	if !ok {
		return DeploymentRecord{}, fmt.Errorf("deployment %s not found", id)
	}
	dep.Status = "ROLLED_BACK"
	dep.RollbackVersion = targetVersion
	dep.DeployedBy = actor
	s.deployments[id] = dep
	return dep, nil
}

func (s *MemStore) ListSLOs() []SLOBudget {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]SLOBudget, 0, len(s.slos))
	for _, v := range s.slos {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) ListReadinessChecks() []ProductionReadinessCheck {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]ProductionReadinessCheck, 0, len(s.readiness))
	for _, v := range s.readiness {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) ExecuteRunbook(r RunbookExecution) RunbookExecution {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if r.ID == "" {
		r.ID = fmt.Sprintf("runbook-exec-%d", now.Unix())
	}
	r.ExecutedAt = now
	r.Status = "SUCCESS"
	r.StepsExecuted = []string{
		"Verified cluster node telemetry & health probes",
		"Applied automated remediation script",
		"Verified service recovery across load balancer targets",
	}
	r.OutputLogs = fmt.Sprintf("Runbook %s completed in 1.4s with 0 errors.", r.RunbookName)
	s.runbooks = append(s.runbooks, r)
	return r
}

func (s *MemStore) ListCMDB() []CMDBItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]CMDBItem, 0, len(s.cmdb))
	for _, v := range s.cmdb {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) ListAlerts() []AlertIncident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]AlertIncident, 0, len(s.alerts))
	for _, v := range s.alerts {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) CreateAlert(a AlertIncident) AlertIncident {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.ID == "" {
		a.ID = fmt.Sprintf("alert-%d", time.Now().Unix())
	}
	if a.TriggeredAt.IsZero() {
		a.TriggeredAt = time.Now().UTC()
	}
	a.Status = "FIRING"
	s.alerts[a.ID] = a
	return a
}

func (s *MemStore) ListPublicStatus() []PublicStatusComponent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]PublicStatusComponent, 0, len(s.statusComps))
	for _, v := range s.statusComps {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) ListCosts() []CloudCostRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.costs
}
