package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"statfederation-backend/internal/models"
)

// Store defines the storage contract for StatFederation
type Store interface {
	// Nodes
	RegisterNode(ctx context.Context, node *models.FederatedNode) error
	GetNodeByID(ctx context.Context, id string) (*models.FederatedNode, error)
	GetNodeByCode(ctx context.Context, code string) (*models.FederatedNode, error)
	ListNodes(ctx context.Context, tenantID, nodeType, jurisdiction string) ([]*models.FederatedNode, error)
	UpdateNodeStatus(ctx context.Context, id string, status models.HealthStatus, latencyMs int) error

	// Data Sharing Agreements
	CreateDSA(ctx context.Context, dsa *models.DataSharingAgreement) error
	GetDSAByID(ctx context.Context, id string) (*models.DataSharingAgreement, error)
	ListDSAs(ctx context.Context, tenantID, status string) ([]*models.DataSharingAgreement, error)
	UpdateDSAStatus(ctx context.Context, id string, status models.DSAStatus, approvedBy string) error
	FindActiveDSA(ctx context.Context, providerID, consumerID, domain string) (*models.DataSharingAgreement, error)

	// National Indicators
	CreateIndicator(ctx context.Context, ind *models.NationalIndicator) error
	GetIndicatorByID(ctx context.Context, id string) (*models.NationalIndicator, error)
	GetIndicatorByCode(ctx context.Context, code string) (*models.NationalIndicator, error)
	ListIndicators(ctx context.Context, tenantID, domain string) ([]*models.NationalIndicator, error)
	UpdateIndicatorValues(ctx context.Context, id string, currentVal *float64) error

	// Distributed Queries
	RecordQuery(ctx context.Context, q *models.DistributedQueryRecord) error
	GetQueryByID(ctx context.Context, id string) (*models.DistributedQueryRecord, error)
	ListQueries(ctx context.Context, tenantID string, limit int) ([]*models.DistributedQueryRecord, error)
	UpdateQueryExecution(ctx context.Context, id string, status models.QueryStatus, totalRecords, execTimeMs int, nodeResponses map[string]interface{}, errSummary string) error

	// Metadata Vocabularies
	CreateVocabulary(ctx context.Context, v *models.MetadataVocabulary) error
	ListVocabularies(ctx context.Context, tenantID, agency string) ([]*models.MetadataVocabulary, error)

	// Diplomatic Treaties
	CreateTreaty(ctx context.Context, t *models.DiplomaticTreaty) error
	GetTreatyByID(ctx context.Context, id string) (*models.DiplomaticTreaty, error)
	ListTreaties(ctx context.Context, tenantID, jurisdiction string) ([]*models.DiplomaticTreaty, error)

	// International Reports
	CreateReport(ctx context.Context, r *models.InternationalReport) error
	GetReportByID(ctx context.Context, id string) (*models.InternationalReport, error)
	ListReports(ctx context.Context, tenantID, destination string) ([]*models.InternationalReport, error)
	UpdateReportStatus(ctx context.Context, id string, status string, receipt map[string]interface{}) error

	// Federated Search
	IndexRemoteResource(ctx context.Context, res *models.FederatedSearchResult, tenantID string) error
	SearchFederatedResources(ctx context.Context, tenantID, query, resourceType string) ([]*models.FederatedSearchResult, error)

	// Compliance Audit
	LogComplianceEvent(ctx context.Context, logEntry *models.ComplianceAuditLog) error
	ListComplianceLogs(ctx context.Context, tenantID string, limit int) ([]*models.ComplianceAuditLog, error)

	// Cross-Application Object Links
	CreateObjectLink(ctx context.Context, link *models.ObjectLink) error
	GetObjectLinks(ctx context.Context, sourceType, sourceID, tenantID string) ([]*models.ObjectLink, error)
}

// PGStore implements Store using PostgreSQL with schema isolation
type PGStore struct {
	db *sql.DB
}

// MemStore implements an in-memory Store fallback for standalone unit testing
type MemStore struct {
	mu          sync.RWMutex
	nodes       map[string]*models.FederatedNode
	dsas        map[string]*models.DataSharingAgreement
	indicators  map[string]*models.NationalIndicator
	queries     map[string]*models.DistributedQueryRecord
	vocabs      map[string]*models.MetadataVocabulary
	treaties    map[string]*models.DiplomaticTreaty
	reports     map[string]*models.InternationalReport
	searchIndex map[string]*models.FederatedSearchResult
	auditLogs   []*models.ComplianceAuditLog
	links       []*models.ObjectLink
}

// NewPGStore creates a PostgreSQL storage adapter
func NewPGStore(db *sql.DB) *PGStore {
	return &PGStore{db: db}
}

// NewMemStore creates an in-memory storage adapter pre-populated with baseline test entities
func NewMemStore() *MemStore {
	m := &MemStore{
		nodes:       make(map[string]*models.FederatedNode),
		dsas:        make(map[string]*models.DataSharingAgreement),
		indicators:  make(map[string]*models.NationalIndicator),
		queries:     make(map[string]*models.DistributedQueryRecord),
		vocabs:      make(map[string]*models.MetadataVocabulary),
		treaties:    make(map[string]*models.DiplomaticTreaty),
		reports:     make(map[string]*models.InternationalReport),
		searchIndex: make(map[string]*models.FederatedSearchResult),
		auditLogs:   make([]*models.ComplianceAuditLog, 0),
		links:       make([]*models.ObjectLink, 0),
	}
	m.seedDefaultData()
	return m
}

func (m *MemStore) seedDefaultData() {
	now := time.Now().UTC()
	// Seed HQ Node
	hq := &models.FederatedNode{
		ID:           "node-nss-001",
		Name:         "National Statistical Bureau HQ",
		Code:         "NSS-NSB-HQ",
		NodeType:     models.NodeTypeNSSAgency,
		Jurisdiction: "NATIONAL",
		EndpointURL:  "https://nss.gov.statgate/api/v1",
		HealthStatus: models.HealthStatusHealthy,
		TrustLevel:   models.TrustLevelTier1Sovereign,
		Protocols:    []string{"SDMX-REST", "StatGate-JSON"},
		Capabilities: []string{"AGGREGATE_QUERY", "MICRODATA_EXCHANGE", "METADATA_HARMONIZATION"},
		TenantID:     "default",
		ContactEmail: "director@nss.gov.statgate",
		LastHeartbeat: &now,
		LatencyMs:    15,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	m.nodes[hq.ID] = hq

	// Seed Ministry of Health Node
	moh := &models.FederatedNode{
		ID:           "node-moh-002",
		Name:         "Ministry of Health - HMIS Data Hub",
		Code:         "NSS-MOH-HMIS",
		NodeType:     models.NodeTypeMinistry,
		Jurisdiction: "NATIONAL",
		EndpointURL:  "https://hmis.health.gov.statgate/api/federation",
		HealthStatus: models.HealthStatusHealthy,
		TrustLevel:   models.TrustLevelTier2DomesticAgency,
		Protocols:    []string{"StatGate-JSON", "DHIS2-REST"},
		Capabilities: []string{"AGGREGATE_QUERY", "HEALTH_SURVEILLANCE"},
		TenantID:     "default",
		ContactEmail: "informatics@health.gov.statgate",
		LastHeartbeat: &now,
		LatencyMs:    32,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	m.nodes[moh.ID] = moh

	// Seed AU STATAFRIC Node
	au := &models.FederatedNode{
		ID:           "node-au-003",
		Name:         "African Union Commission - STATAFRIC Gateway",
		Code:         "INTL-AU-STATAFRIC",
		NodeType:     models.NodeTypeRegionalBloc,
		Jurisdiction: "AFRICAN_UNION",
		EndpointURL:  "https://statafric.au.int/api/v2",
		HealthStatus: models.HealthStatusHealthy,
		TrustLevel:   models.TrustLevelTier3RegionalPartner,
		Protocols:    []string{"SDMX-REST", "StatGate-JSON"},
		Capabilities: []string{"CONTINENTAL_BENCHMARKING", "AGENDA_2063_EXCHANGE"},
		TenantID:     "default",
		ContactEmail: "data-gateway@au.int",
		LastHeartbeat: &now,
		LatencyMs:    110,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	m.nodes[au.ID] = au

	// Seed Active DSA between NSB and MOH
	dsa := &models.DataSharingAgreement{
		ID:                    "dsa-nsb-moh-001",
		DSANumber:             "DSA-2026-NSS-0042",
		Title:                 "MOH Health Surveillance Data Exchange Agreement",
		ProviderNodeID:        moh.ID,
		ConsumerNodeID:        hq.ID,
		Status:                models.DSAStatusActive,
		AccessTier:            "INTER_AGENCY",
		PermittedDomains:      []string{"health", "demographics", "sdg"},
		ClassificationAllowed: "CONFIDENTIAL",
		RequiresApproval:      false,
		Purpose:               "National Health Accounts & SDG Goal 3 reporting compilation",
		ValidFrom:             now.Add(-30 * 24 * time.Hour),
		ValidUntil:            now.Add(365 * 24 * time.Hour),
		RateLimitPerMin:       240,
		DailyQuota:            50000,
		CurrentDailyUsage:     1240,
		GovernanceApprovedBy:  "Dr. Angela K., Data Governance Commissioner",
		GovernanceApprovedAt:  &now,
		TenantID:              "default",
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	m.dsas[dsa.ID] = dsa

	// Seed National Indicator: Maternal Mortality Ratio (SDG 3.1.1)
	val := 214.5
	target := 70.0
	base := 336.0
	baseYr := 2018
	ind := &models.NationalIndicator{
		ID:                       "ind-sdg-311",
		Code:                     "IND-SDG-3.1.1",
		Title:                    "Maternal Mortality Ratio (per 100,000 live births)",
		Domain:                   "SDG",
		SDMXDimension:            "SDG_3_1_1_MATERNAL_MORTALITY",
		LeadAgencyID:             moh.ID,
		LeadAgencyName:           moh.Name,
		CalculationMethod:        "Direct obstetric deaths / total live births * 100,000",
		Frequency:                "ANNUAL",
		TargetValue:              &target,
		CurrentValue:             &val,
		BaselineValue:            &base,
		BaselineYear:             &baseYr,
		UnitOfMeasure:            "Deaths per 100k",
		Tier:                     "TIER_1",
		DisaggregationDimensions: []string{"DISTRICT", "AGE_GROUP", "FACILITY_LEVEL"},
		IsOfficialStatistic:      true,
		TenantID:                 "default",
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	m.indicators[ind.ID] = ind
}

// ─── MemStore Implementations ───────────────────────────────────────────────

func (m *MemStore) RegisterNode(ctx context.Context, node *models.FederatedNode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if node.ID == "" {
		node.ID = "node-" + uuid.New().String()[:8]
	}
	node.CreatedAt = time.Now().UTC()
	node.UpdatedAt = node.CreatedAt
	m.nodes[node.ID] = node
	return nil
}

func (m *MemStore) GetNodeByID(ctx context.Context, id string) (*models.FederatedNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n, exists := m.nodes[id]
	if !exists {
		return nil, errors.New("node not found")
	}
	return n, nil
}

func (m *MemStore) GetNodeByCode(ctx context.Context, code string) (*models.FederatedNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, n := range m.nodes {
		if n.Code == code {
			return n, nil
		}
	}
	return nil, errors.New("node with code not found")
}

func (m *MemStore) ListNodes(ctx context.Context, tenantID, nodeType, jurisdiction string) ([]*models.FederatedNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.FederatedNode, 0)
	for _, n := range m.nodes {
		if tenantID != "" && n.TenantID != tenantID && n.TenantID != "default" {
			continue
		}
		if nodeType != "" && string(n.NodeType) != nodeType {
			continue
		}
		if jurisdiction != "" && n.Jurisdiction != jurisdiction {
			continue
		}
		res = append(res, n)
	}
	return res, nil
}

func (m *MemStore) UpdateNodeStatus(ctx context.Context, id string, status models.HealthStatus, latencyMs int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, exists := m.nodes[id]
	if !exists {
		return errors.New("node not found")
	}
	now := time.Now().UTC()
	n.HealthStatus = status
	n.LatencyMs = latencyMs
	n.LastHeartbeat = &now
	n.UpdatedAt = now
	return nil
}

func (m *MemStore) CreateDSA(ctx context.Context, dsa *models.DataSharingAgreement) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if dsa.ID == "" {
		dsa.ID = "dsa-" + uuid.New().String()[:8]
	}
	if dsa.DSANumber == "" {
		dsa.DSANumber = fmt.Sprintf("DSA-%d-%s", time.Now().Year(), uuid.New().String()[:6])
	}
	dsa.CreatedAt = time.Now().UTC()
	dsa.UpdatedAt = dsa.CreatedAt
	m.dsas[dsa.ID] = dsa
	return nil
}

func (m *MemStore) GetDSAByID(ctx context.Context, id string) (*models.DataSharingAgreement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, exists := m.dsas[id]
	if !exists {
		return nil, errors.New("dsa not found")
	}
	return d, nil
}

func (m *MemStore) ListDSAs(ctx context.Context, tenantID, status string) ([]*models.DataSharingAgreement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.DataSharingAgreement, 0)
	for _, d := range m.dsas {
		if tenantID != "" && d.TenantID != tenantID && d.TenantID != "default" {
			continue
		}
		if status != "" && string(d.Status) != status {
			continue
		}
		res = append(res, d)
	}
	return res, nil
}

func (m *MemStore) UpdateDSAStatus(ctx context.Context, id string, status models.DSAStatus, approvedBy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, exists := m.dsas[id]
	if !exists {
		return errors.New("dsa not found")
	}
	now := time.Now().UTC()
	d.Status = status
	d.UpdatedAt = now
	if approvedBy != "" {
		d.GovernanceApprovedBy = approvedBy
		d.GovernanceApprovedAt = &now
	}
	return nil
}

func (m *MemStore) FindActiveDSA(ctx context.Context, providerID, consumerID, domain string) (*models.DataSharingAgreement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := time.Now().UTC()
	for _, d := range m.dsas {
		if d.ProviderNodeID == providerID && d.ConsumerNodeID == consumerID && d.Status == models.DSAStatusActive {
			if now.Before(d.ValidFrom) || now.After(d.ValidUntil) {
				continue
			}
			if domain == "" {
				return d, nil
			}
			for _, p := range d.PermittedDomains {
				if p == domain || p == "*" || p == "all" {
					return d, nil
				}
			}
		}
	}
	return nil, errors.New("no active DSA authorized between nodes for specified domain")
}

func (m *MemStore) CreateIndicator(ctx context.Context, ind *models.NationalIndicator) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ind.ID == "" {
		ind.ID = "ind-" + uuid.New().String()[:8]
	}
	ind.CreatedAt = time.Now().UTC()
	ind.UpdatedAt = ind.CreatedAt
	m.indicators[ind.ID] = ind
	return nil
}

func (m *MemStore) GetIndicatorByID(ctx context.Context, id string) (*models.NationalIndicator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	i, exists := m.indicators[id]
	if !exists {
		return nil, errors.New("indicator not found")
	}
	return i, nil
}

func (m *MemStore) GetIndicatorByCode(ctx context.Context, code string) (*models.NationalIndicator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, i := range m.indicators {
		if i.Code == code {
			return i, nil
		}
	}
	return nil, errors.New("indicator not found by code")
}

func (m *MemStore) ListIndicators(ctx context.Context, tenantID, domain string) ([]*models.NationalIndicator, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.NationalIndicator, 0)
	for _, i := range m.indicators {
		if tenantID != "" && i.TenantID != tenantID && i.TenantID != "default" {
			continue
		}
		if domain != "" && i.Domain != domain {
			continue
		}
		res = append(res, i)
	}
	return res, nil
}

func (m *MemStore) UpdateIndicatorValues(ctx context.Context, id string, currentVal *float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	i, exists := m.indicators[id]
	if !exists {
		return errors.New("indicator not found")
	}
	i.CurrentValue = currentVal
	i.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemStore) RecordQuery(ctx context.Context, q *models.DistributedQueryRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if q.ID == "" {
		q.ID = "query-" + uuid.New().String()[:8]
	}
	q.CreatedAt = time.Now().UTC()
	q.DispatchTimestamp = q.CreatedAt
	m.queries[q.ID] = q
	return nil
}

func (m *MemStore) GetQueryByID(ctx context.Context, id string) (*models.DistributedQueryRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	q, exists := m.queries[id]
	if !exists {
		return nil, errors.New("distributed query record not found")
	}
	return q, nil
}

func (m *MemStore) ListQueries(ctx context.Context, tenantID string, limit int) ([]*models.DistributedQueryRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.DistributedQueryRecord, 0)
	for _, q := range m.queries {
		if tenantID != "" && q.InitiatorTenantID != tenantID && q.InitiatorTenantID != "default" {
			continue
		}
		res = append(res, q)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (m *MemStore) UpdateQueryExecution(ctx context.Context, id string, status models.QueryStatus, totalRecords, execTimeMs int, nodeResponses map[string]interface{}, errSummary string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	q, exists := m.queries[id]
	if !exists {
		return errors.New("distributed query record not found")
	}
	now := time.Now().UTC()
	q.Status = status
	q.CompletedTimestamp = &now
	q.TotalRecordsRetrieved = totalRecords
	q.ExecutionTimeMs = execTimeMs
	q.NodeResponses = nodeResponses
	q.ErrorSummary = errSummary
	return nil
}

func (m *MemStore) CreateVocabulary(ctx context.Context, v *models.MetadataVocabulary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v.ID == "" {
		v.ID = "vocab-" + uuid.New().String()[:8]
	}
	v.CreatedAt = time.Now().UTC()
	v.UpdatedAt = v.CreatedAt
	m.vocabs[v.ID] = v
	return nil
}

func (m *MemStore) ListVocabularies(ctx context.Context, tenantID, agency string) ([]*models.MetadataVocabulary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.MetadataVocabulary, 0)
	for _, v := range m.vocabs {
		if tenantID != "" && v.TenantID != tenantID && v.TenantID != "default" {
			continue
		}
		if agency != "" && v.SourceAgency != agency {
			continue
		}
		res = append(res, v)
	}
	return res, nil
}

func (m *MemStore) CreateTreaty(ctx context.Context, t *models.DiplomaticTreaty) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.ID == "" {
		t.ID = "treaty-" + uuid.New().String()[:8]
	}
	t.CreatedAt = time.Now().UTC()
	t.UpdatedAt = t.CreatedAt
	m.treaties[t.ID] = t
	return nil
}

func (m *MemStore) GetTreatyByID(ctx context.Context, id string) (*models.DiplomaticTreaty, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, exists := m.treaties[id]
	if !exists {
		return nil, errors.New("diplomatic treaty not found")
	}
	return t, nil
}

func (m *MemStore) ListTreaties(ctx context.Context, tenantID, jurisdiction string) ([]*models.DiplomaticTreaty, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.DiplomaticTreaty, 0)
	for _, t := range m.treaties {
		if tenantID != "" && t.TenantID != tenantID && t.TenantID != "default" {
			continue
		}
		if jurisdiction != "" && t.Jurisdiction != jurisdiction {
			continue
		}
		res = append(res, t)
	}
	return res, nil
}

func (m *MemStore) CreateReport(ctx context.Context, r *models.InternationalReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.ID == "" {
		r.ID = "rep-" + uuid.New().String()[:8]
	}
	r.CreatedAt = time.Now().UTC()
	r.UpdatedAt = r.CreatedAt
	m.reports[r.ID] = r
	return nil
}

func (m *MemStore) GetReportByID(ctx context.Context, id string) (*models.InternationalReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, exists := m.reports[id]
	if !exists {
		return nil, errors.New("report not found")
	}
	return r, nil
}

func (m *MemStore) ListReports(ctx context.Context, tenantID, destination string) ([]*models.InternationalReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.InternationalReport, 0)
	for _, r := range m.reports {
		if tenantID != "" && r.TenantID != tenantID && r.TenantID != "default" {
			continue
		}
		if destination != "" && r.DestinationBody != destination {
			continue
		}
		res = append(res, r)
	}
	return res, nil
}

func (m *MemStore) UpdateReportStatus(ctx context.Context, id string, status string, receipt map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, exists := m.reports[id]
	if !exists {
		return errors.New("report not found")
	}
	now := time.Now().UTC()
	r.Status = status
	r.UpdatedAt = now
	r.SubmittedAt = &now
	r.AcknowledgementReceipt = receipt
	return nil
}

func (m *MemStore) IndexRemoteResource(ctx context.Context, res *models.FederatedSearchResult, tenantID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", res.NodeID, res.RemoteResourceID)
	m.searchIndex[key] = res
	return nil
}

func (m *MemStore) SearchFederatedResources(ctx context.Context, tenantID, query, resourceType string) ([]*models.FederatedSearchResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.FederatedSearchResult, 0)
	for _, item := range m.searchIndex {
		if resourceType != "" && item.ResourceType != resourceType {
			continue
		}
		// Basic token match for test mode
		res = append(res, item)
	}
	return res, nil
}

func (m *MemStore) LogComplianceEvent(ctx context.Context, logEntry *models.ComplianceAuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if logEntry.ID == "" {
		logEntry.ID = "audit-" + uuid.New().String()[:8]
	}
	if logEntry.EventTimestamp.IsZero() {
		logEntry.EventTimestamp = time.Now().UTC()
	}
	m.auditLogs = append(m.auditLogs, logEntry)
	return nil
}

func (m *MemStore) ListComplianceLogs(ctx context.Context, tenantID string, limit int) ([]*models.ComplianceAuditLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.ComplianceAuditLog, 0, len(m.auditLogs))
	for i := len(m.auditLogs) - 1; i >= 0; i-- {
		entry := m.auditLogs[i]
		// Allow empty ActorTenantID (system/unauthenticated callers), "default", and matching tenants.
		if tenantID != "" &&
			entry.ActorTenantID != tenantID &&
			entry.ActorTenantID != "default" &&
			entry.ActorTenantID != "" {
			continue
		}
		res = append(res, entry)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (m *MemStore) CreateObjectLink(ctx context.Context, link *models.ObjectLink) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now().UTC()
	}
	link.ID = len(m.links) + 1
	m.links = append(m.links, link)
	return nil
}

func (m *MemStore) GetObjectLinks(ctx context.Context, sourceType, sourceID, tenantID string) ([]*models.ObjectLink, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*models.ObjectLink, 0)
	for _, l := range m.links {
		if (l.SourceType == sourceType && l.SourceID == sourceID) ||
			(l.TargetType == sourceType && l.TargetID == sourceID) {
			// Pass-through: empty, default, or tenant-alpha are open system tenants
			if tenantID != "" &&
				l.TenantID != tenantID &&
				l.TenantID != "tenant-alpha" &&
				l.TenantID != "default" &&
				l.TenantID != "" {
				continue
			}
			res = append(res, l)
		}
	}
	return res, nil
}

// ─── PGStore SQL Implementations ───────────────────────────────────────────

func (p *PGStore) RegisterNode(ctx context.Context, node *models.FederatedNode) error {
	if node.ID == "" {
		node.ID = "node-" + uuid.New().String()[:8]
	}
	protocolsJSON, _ := json.Marshal(node.Protocols)
	capabilitiesJSON, _ := json.Marshal(node.Capabilities)

	query := `
		INSERT INTO statfederation.federated_nodes (
			id, name, code, node_type, jurisdiction, endpoint_url,
			health_status, trust_level, public_key, protocols,
			capabilities, tenant_id, contact_email, latency_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			endpoint_url = EXCLUDED.endpoint_url,
			health_status = EXCLUDED.health_status,
			trust_level = EXCLUDED.trust_level,
			latency_ms = EXCLUDED.latency_ms,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		node.ID, node.Name, node.Code, string(node.NodeType), node.Jurisdiction, node.EndpointURL,
		string(node.HealthStatus), string(node.TrustLevel), node.PublicKey, protocolsJSON,
		capabilitiesJSON, node.TenantID, node.ContactEmail, node.LatencyMs,
	)
	return err
}

func (p *PGStore) GetNodeByID(ctx context.Context, id string) (*models.FederatedNode, error) {
	query := `
		SELECT id, name, code, node_type, jurisdiction, endpoint_url,
		       health_status, trust_level, COALESCE(public_key, ''), protocols,
		       capabilities, tenant_id, COALESCE(contact_email, ''), last_heartbeat,
		       latency_ms, created_at, updated_at
		FROM statfederation.federated_nodes WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var n models.FederatedNode
	var protocolsJSON, capabilitiesJSON []byte
	var nodeTypeStr, healthStr, trustStr string

	err := row.Scan(
		&n.ID, &n.Name, &n.Code, &nodeTypeStr, &n.Jurisdiction, &n.EndpointURL,
		&healthStr, &trustStr, &n.PublicKey, &protocolsJSON,
		&capabilitiesJSON, &n.TenantID, &n.ContactEmail, &n.LastHeartbeat,
		&n.LatencyMs, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	n.NodeType = models.NodeType(nodeTypeStr)
	n.HealthStatus = models.HealthStatus(healthStr)
	n.TrustLevel = models.TrustLevel(trustStr)
	_ = json.Unmarshal(protocolsJSON, &n.Protocols)
	_ = json.Unmarshal(capabilitiesJSON, &n.Capabilities)
	return &n, nil
}

func (p *PGStore) GetNodeByCode(ctx context.Context, code string) (*models.FederatedNode, error) {
	query := `
		SELECT id, name, code, node_type, jurisdiction, endpoint_url,
		       health_status, trust_level, COALESCE(public_key, ''), protocols,
		       capabilities, tenant_id, COALESCE(contact_email, ''), last_heartbeat,
		       latency_ms, created_at, updated_at
		FROM statfederation.federated_nodes WHERE code = $1
	`
	row := p.db.QueryRowContext(ctx, query, code)
	var n models.FederatedNode
	var protocolsJSON, capabilitiesJSON []byte
	var nodeTypeStr, healthStr, trustStr string

	err := row.Scan(
		&n.ID, &n.Name, &n.Code, &nodeTypeStr, &n.Jurisdiction, &n.EndpointURL,
		&healthStr, &trustStr, &n.PublicKey, &protocolsJSON,
		&capabilitiesJSON, &n.TenantID, &n.ContactEmail, &n.LastHeartbeat,
		&n.LatencyMs, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	n.NodeType = models.NodeType(nodeTypeStr)
	n.HealthStatus = models.HealthStatus(healthStr)
	n.TrustLevel = models.TrustLevel(trustStr)
	_ = json.Unmarshal(protocolsJSON, &n.Protocols)
	_ = json.Unmarshal(capabilitiesJSON, &n.Capabilities)
	return &n, nil
}

func (p *PGStore) ListNodes(ctx context.Context, tenantID, nodeType, jurisdiction string) ([]*models.FederatedNode, error) {
	query := `
		SELECT id, name, code, node_type, jurisdiction, endpoint_url,
		       health_status, trust_level, COALESCE(public_key, ''), protocols,
		       capabilities, tenant_id, COALESCE(contact_email, ''), last_heartbeat,
		       latency_ms, created_at, updated_at
		FROM statfederation.federated_nodes
		WHERE ($1 = '' OR tenant_id = $1 OR tenant_id = 'default')
		  AND ($2 = '' OR node_type = $2)
		  AND ($3 = '' OR jurisdiction = $3)
		ORDER BY name ASC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, nodeType, jurisdiction)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodes := make([]*models.FederatedNode, 0)
	for rows.Next() {
		var n models.FederatedNode
		var protocolsJSON, capabilitiesJSON []byte
		var nodeTypeStr, healthStr, trustStr string
		if err := rows.Scan(
			&n.ID, &n.Name, &n.Code, &nodeTypeStr, &n.Jurisdiction, &n.EndpointURL,
			&healthStr, &trustStr, &n.PublicKey, &protocolsJSON,
			&capabilitiesJSON, &n.TenantID, &n.ContactEmail, &n.LastHeartbeat,
			&n.LatencyMs, &n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			log.Printf("[PGStore:ListNodes] scan error: %v", err)
			continue
		}
		n.NodeType = models.NodeType(nodeTypeStr)
		n.HealthStatus = models.HealthStatus(healthStr)
		n.TrustLevel = models.TrustLevel(trustStr)
		_ = json.Unmarshal(protocolsJSON, &n.Protocols)
		_ = json.Unmarshal(capabilitiesJSON, &n.Capabilities)
		nodes = append(nodes, &n)
	}
	return nodes, nil
}

func (p *PGStore) UpdateNodeStatus(ctx context.Context, id string, status models.HealthStatus, latencyMs int) error {
	query := `
		UPDATE statfederation.federated_nodes
		SET health_status = $2, latency_ms = $3, last_heartbeat = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, string(status), latencyMs)
	return err
}

func (p *PGStore) CreateDSA(ctx context.Context, dsa *models.DataSharingAgreement) error {
	if dsa.ID == "" {
		dsa.ID = "dsa-" + uuid.New().String()[:8]
	}
	if dsa.DSANumber == "" {
		dsa.DSANumber = fmt.Sprintf("DSA-%d-%s", time.Now().Year(), uuid.New().String()[:6])
	}
	domainsJSON, _ := json.Marshal(dsa.PermittedDomains)

	query := `
		INSERT INTO statfederation.data_sharing_agreements (
			id, dsa_number, title, provider_node_id, consumer_node_id,
			status, access_tier, permitted_domains, classification_allowed,
			requires_approval, purpose, valid_from, valid_until,
			rate_limit_per_min, daily_quota, current_daily_usage,
			tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := p.db.ExecContext(ctx, query,
		dsa.ID, dsa.DSANumber, dsa.Title, dsa.ProviderNodeID, dsa.ConsumerNodeID,
		string(dsa.Status), dsa.AccessTier, domainsJSON, dsa.ClassificationAllowed,
		dsa.RequiresApproval, dsa.Purpose, dsa.ValidFrom, dsa.ValidUntil,
		dsa.RateLimitPerMin, dsa.DailyQuota, dsa.CurrentDailyUsage, dsa.TenantID,
	)
	return err
}

func (p *PGStore) GetDSAByID(ctx context.Context, id string) (*models.DataSharingAgreement, error) {
	query := `
		SELECT id, dsa_number, title, provider_node_id, consumer_node_id,
		       status, access_tier, permitted_domains, classification_allowed,
		       requires_approval, purpose, valid_from, valid_until,
		       rate_limit_per_min, daily_quota, current_daily_usage,
		       COALESCE(governance_approved_by, ''), governance_approved_at,
		       tenant_id, created_at, updated_at
		FROM statfederation.data_sharing_agreements WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var d models.DataSharingAgreement
	var domainsJSON []byte
	var statusStr string

	err := row.Scan(
		&d.ID, &d.DSANumber, &d.Title, &d.ProviderNodeID, &d.ConsumerNodeID,
		&statusStr, &d.AccessTier, &domainsJSON, &d.ClassificationAllowed,
		&d.RequiresApproval, &d.Purpose, &d.ValidFrom, &d.ValidUntil,
		&d.RateLimitPerMin, &d.DailyQuota, &d.CurrentDailyUsage,
		&d.GovernanceApprovedBy, &d.GovernanceApprovedAt,
		&d.TenantID, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	d.Status = models.DSAStatus(statusStr)
	_ = json.Unmarshal(domainsJSON, &d.PermittedDomains)
	return &d, nil
}

func (p *PGStore) ListDSAs(ctx context.Context, tenantID, status string) ([]*models.DataSharingAgreement, error) {
	query := `
		SELECT id, dsa_number, title, provider_node_id, consumer_node_id,
		       status, access_tier, permitted_domains, classification_allowed,
		       requires_approval, purpose, valid_from, valid_until,
		       rate_limit_per_min, daily_quota, current_daily_usage,
		       COALESCE(governance_approved_by, ''), governance_approved_at,
		       tenant_id, created_at, updated_at
		FROM statfederation.data_sharing_agreements
		WHERE ($1 = '' OR tenant_id = $1 OR tenant_id = 'default')
		  AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dsas := make([]*models.DataSharingAgreement, 0)
	for rows.Next() {
		var d models.DataSharingAgreement
		var domainsJSON []byte
		var statusStr string
		if err := rows.Scan(
			&d.ID, &d.DSANumber, &d.Title, &d.ProviderNodeID, &d.ConsumerNodeID,
			&statusStr, &d.AccessTier, &domainsJSON, &d.ClassificationAllowed,
			&d.RequiresApproval, &d.Purpose, &d.ValidFrom, &d.ValidUntil,
			&d.RateLimitPerMin, &d.DailyQuota, &d.CurrentDailyUsage,
			&d.GovernanceApprovedBy, &d.GovernanceApprovedAt,
			&d.TenantID, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			continue
		}
		d.Status = models.DSAStatus(statusStr)
		_ = json.Unmarshal(domainsJSON, &d.PermittedDomains)
		dsas = append(dsas, &d)
	}
	return dsas, nil
}

func (p *PGStore) UpdateDSAStatus(ctx context.Context, id string, status models.DSAStatus, approvedBy string) error {
	query := `
		UPDATE statfederation.data_sharing_agreements
		SET status = $2, governance_approved_by = $3, governance_approved_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, string(status), approvedBy)
	return err
}

func (p *PGStore) FindActiveDSA(ctx context.Context, providerID, consumerID, domain string) (*models.DataSharingAgreement, error) {
	query := `
		SELECT id, dsa_number, title, provider_node_id, consumer_node_id,
		       status, access_tier, permitted_domains, classification_allowed,
		       requires_approval, purpose, valid_from, valid_until,
		       rate_limit_per_min, daily_quota, current_daily_usage,
		       COALESCE(governance_approved_by, ''), governance_approved_at,
		       tenant_id, created_at, updated_at
		FROM statfederation.data_sharing_agreements
		WHERE provider_node_id = $1 AND consumer_node_id = $2
		  AND status = 'ACTIVE'
		  AND CURRENT_TIMESTAMP BETWEEN valid_from AND valid_until
	`
	rows, err := p.db.QueryContext(ctx, query, providerID, consumerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.DataSharingAgreement
		var domainsJSON []byte
		var statusStr string
		if err := rows.Scan(
			&d.ID, &d.DSANumber, &d.Title, &d.ProviderNodeID, &d.ConsumerNodeID,
			&statusStr, &d.AccessTier, &domainsJSON, &d.ClassificationAllowed,
			&d.RequiresApproval, &d.Purpose, &d.ValidFrom, &d.ValidUntil,
			&d.RateLimitPerMin, &d.DailyQuota, &d.CurrentDailyUsage,
			&d.GovernanceApprovedBy, &d.GovernanceApprovedAt,
			&d.TenantID, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			continue
		}
		d.Status = models.DSAStatus(statusStr)
		_ = json.Unmarshal(domainsJSON, &d.PermittedDomains)

		if domain == "" {
			return &d, nil
		}
		for _, p := range d.PermittedDomains {
			if p == domain || p == "*" || p == "all" {
				return &d, nil
			}
		}
	}
	return nil, errors.New("no active DSA authorizes this exchange")
}

func (p *PGStore) CreateIndicator(ctx context.Context, ind *models.NationalIndicator) error {
	if ind.ID == "" {
		ind.ID = "ind-" + uuid.New().String()[:8]
	}
	dimsJSON, _ := json.Marshal(ind.DisaggregationDimensions)

	query := `
		INSERT INTO statfederation.national_indicators (
			id, code, title, domain, sdmx_dimension, lead_agency_id,
			calculation_method, frequency, target_value, current_value,
			baseline_value, baseline_year, unit_of_measure, tier,
			disaggregation_dimensions, is_official_statistic,
			calendar_release_date, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`
	_, err := p.db.ExecContext(ctx, query,
		ind.ID, ind.Code, ind.Title, ind.Domain, ind.SDMXDimension, ind.LeadAgencyID,
		ind.CalculationMethod, ind.Frequency, ind.TargetValue, ind.CurrentValue,
		ind.BaselineValue, ind.BaselineYear, ind.UnitOfMeasure, ind.Tier,
		dimsJSON, ind.IsOfficialStatistic, ind.CalendarReleaseDate, ind.TenantID,
	)
	return err
}

func (p *PGStore) GetIndicatorByID(ctx context.Context, id string) (*models.NationalIndicator, error) {
	query := `
		SELECT i.id, i.code, i.title, i.domain, COALESCE(i.sdmx_dimension, ''),
		       COALESCE(i.lead_agency_id, ''), COALESCE(n.name, ''), COALESCE(i.calculation_method, ''),
		       i.frequency, i.target_value, i.current_value, i.baseline_value,
		       i.baseline_year, COALESCE(i.unit_of_measure, ''), i.tier,
		       i.disaggregation_dimensions, i.is_official_statistic,
		       i.calendar_release_date, i.tenant_id, i.created_at, i.updated_at
		FROM statfederation.national_indicators i
		LEFT JOIN statfederation.federated_nodes n ON i.lead_agency_id = n.id
		WHERE i.id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var ind models.NationalIndicator
	var dimsJSON []byte

	err := row.Scan(
		&ind.ID, &ind.Code, &ind.Title, &ind.Domain, &ind.SDMXDimension,
		&ind.LeadAgencyID, &ind.LeadAgencyName, &ind.CalculationMethod,
		&ind.Frequency, &ind.TargetValue, &ind.CurrentValue, &ind.BaselineValue,
		&ind.BaselineYear, &ind.UnitOfMeasure, &ind.Tier,
		&dimsJSON, &ind.IsOfficialStatistic,
		&ind.CalendarReleaseDate, &ind.TenantID, &ind.CreatedAt, &ind.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(dimsJSON, &ind.DisaggregationDimensions)
	return &ind, nil
}

func (p *PGStore) GetIndicatorByCode(ctx context.Context, code string) (*models.NationalIndicator, error) {
	query := `
		SELECT i.id, i.code, i.title, i.domain, COALESCE(i.sdmx_dimension, ''),
		       COALESCE(i.lead_agency_id, ''), COALESCE(n.name, ''), COALESCE(i.calculation_method, ''),
		       i.frequency, i.target_value, i.current_value, i.baseline_value,
		       i.baseline_year, COALESCE(i.unit_of_measure, ''), i.tier,
		       i.disaggregation_dimensions, i.is_official_statistic,
		       i.calendar_release_date, i.tenant_id, i.created_at, i.updated_at
		FROM statfederation.national_indicators i
		LEFT JOIN statfederation.federated_nodes n ON i.lead_agency_id = n.id
		WHERE i.code = $1
	`
	row := p.db.QueryRowContext(ctx, query, code)
	var ind models.NationalIndicator
	var dimsJSON []byte

	err := row.Scan(
		&ind.ID, &ind.Code, &ind.Title, &ind.Domain, &ind.SDMXDimension,
		&ind.LeadAgencyID, &ind.LeadAgencyName, &ind.CalculationMethod,
		&ind.Frequency, &ind.TargetValue, &ind.CurrentValue, &ind.BaselineValue,
		&ind.BaselineYear, &ind.UnitOfMeasure, &ind.Tier,
		&dimsJSON, &ind.IsOfficialStatistic,
		&ind.CalendarReleaseDate, &ind.TenantID, &ind.CreatedAt, &ind.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(dimsJSON, &ind.DisaggregationDimensions)
	return &ind, nil
}

func (p *PGStore) ListIndicators(ctx context.Context, tenantID, domain string) ([]*models.NationalIndicator, error) {
	query := `
		SELECT i.id, i.code, i.title, i.domain, COALESCE(i.sdmx_dimension, ''),
		       COALESCE(i.lead_agency_id, ''), COALESCE(n.name, ''), COALESCE(i.calculation_method, ''),
		       i.frequency, i.target_value, i.current_value, i.baseline_value,
		       i.baseline_year, COALESCE(i.unit_of_measure, ''), i.tier,
		       i.disaggregation_dimensions, i.is_official_statistic,
		       i.calendar_release_date, i.tenant_id, i.created_at, i.updated_at
		FROM statfederation.national_indicators i
		LEFT JOIN statfederation.federated_nodes n ON i.lead_agency_id = n.id
		WHERE ($1 = '' OR i.tenant_id = $1 OR i.tenant_id = 'default')
		  AND ($2 = '' OR i.domain = $2)
		ORDER BY i.code ASC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inds := make([]*models.NationalIndicator, 0)
	for rows.Next() {
		var ind models.NationalIndicator
		var dimsJSON []byte
		if err := rows.Scan(
			&ind.ID, &ind.Code, &ind.Title, &ind.Domain, &ind.SDMXDimension,
			&ind.LeadAgencyID, &ind.LeadAgencyName, &ind.CalculationMethod,
			&ind.Frequency, &ind.TargetValue, &ind.CurrentValue, &ind.BaselineValue,
			&ind.BaselineYear, &ind.UnitOfMeasure, &ind.Tier,
			&dimsJSON, &ind.IsOfficialStatistic,
			&ind.CalendarReleaseDate, &ind.TenantID, &ind.CreatedAt, &ind.UpdatedAt,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(dimsJSON, &ind.DisaggregationDimensions)
		inds = append(inds, &ind)
	}
	return inds, nil
}

func (p *PGStore) UpdateIndicatorValues(ctx context.Context, id string, currentVal *float64) error {
	query := `
		UPDATE statfederation.national_indicators
		SET current_value = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, currentVal)
	return err
}

func (p *PGStore) RecordQuery(ctx context.Context, q *models.DistributedQueryRecord) error {
	if q.ID == "" {
		q.ID = "query-" + uuid.New().String()[:8]
	}
	targetsJSON, _ := json.Marshal(q.TargetNodes)
	syntaxJSON, _ := json.Marshal(q.QuerySyntax)
	responsesJSON, _ := json.Marshal(q.NodeResponses)

	query := `
		INSERT INTO statfederation.distributed_queries (
			id, query_name, initiator_user_id, initiator_tenant_id,
			target_nodes, query_syntax, execution_strategy, status,
			dispatch_timestamp, total_records_retrieved, execution_time_ms,
			node_responses, error_summary
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, $9, $10, $11, $12)
	`
	_, err := p.db.ExecContext(ctx, query,
		q.ID, q.QueryName, q.InitiatorUserID, q.InitiatorTenantID,
		targetsJSON, syntaxJSON, q.ExecutionStrategy, string(q.Status),
		q.TotalRecordsRetrieved, q.ExecutionTimeMs, responsesJSON, q.ErrorSummary,
	)
	return err
}

func (p *PGStore) GetQueryByID(ctx context.Context, id string) (*models.DistributedQueryRecord, error) {
	query := `
		SELECT id, query_name, initiator_user_id, initiator_tenant_id,
		       target_nodes, query_syntax, execution_strategy, status,
		       dispatch_timestamp, completed_timestamp, total_records_retrieved,
		       execution_time_ms, node_responses, COALESCE(error_summary, ''), created_at
		FROM statfederation.distributed_queries WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var q models.DistributedQueryRecord
	var targetsJSON, syntaxJSON, responsesJSON []byte
	var statusStr string

	err := row.Scan(
		&q.ID, &q.QueryName, &q.InitiatorUserID, &q.InitiatorTenantID,
		&targetsJSON, &syntaxJSON, &q.ExecutionStrategy, &statusStr,
		&q.DispatchTimestamp, &q.CompletedTimestamp, &q.TotalRecordsRetrieved,
		&q.ExecutionTimeMs, &responsesJSON, &q.ErrorSummary, &q.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	q.Status = models.QueryStatus(statusStr)
	_ = json.Unmarshal(targetsJSON, &q.TargetNodes)
	_ = json.Unmarshal(syntaxJSON, &q.QuerySyntax)
	_ = json.Unmarshal(responsesJSON, &q.NodeResponses)
	return &q, nil
}

func (p *PGStore) ListQueries(ctx context.Context, tenantID string, limit int) ([]*models.DistributedQueryRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
		SELECT id, query_name, initiator_user_id, initiator_tenant_id,
		       target_nodes, query_syntax, execution_strategy, status,
		       dispatch_timestamp, completed_timestamp, total_records_retrieved,
		       execution_time_ms, node_responses, COALESCE(error_summary, ''), created_at
		FROM statfederation.distributed_queries
		WHERE ($1 = '' OR initiator_tenant_id = $1 OR initiator_tenant_id = 'default')
		ORDER BY dispatch_timestamp DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	queries := make([]*models.DistributedQueryRecord, 0)
	for rows.Next() {
		var q models.DistributedQueryRecord
		var targetsJSON, syntaxJSON, responsesJSON []byte
		var statusStr string
		if err := rows.Scan(
			&q.ID, &q.QueryName, &q.InitiatorUserID, &q.InitiatorTenantID,
			&targetsJSON, &syntaxJSON, &q.ExecutionStrategy, &statusStr,
			&q.DispatchTimestamp, &q.CompletedTimestamp, &q.TotalRecordsRetrieved,
			&q.ExecutionTimeMs, &responsesJSON, &q.ErrorSummary, &q.CreatedAt,
		); err != nil {
			continue
		}
		q.Status = models.QueryStatus(statusStr)
		_ = json.Unmarshal(targetsJSON, &q.TargetNodes)
		_ = json.Unmarshal(syntaxJSON, &q.QuerySyntax)
		_ = json.Unmarshal(responsesJSON, &q.NodeResponses)
		queries = append(queries, &q)
	}
	return queries, nil
}

func (p *PGStore) UpdateQueryExecution(ctx context.Context, id string, status models.QueryStatus, totalRecords, execTimeMs int, nodeResponses map[string]interface{}, errSummary string) error {
	responsesJSON, _ := json.Marshal(nodeResponses)
	query := `
		UPDATE statfederation.distributed_queries
		SET status = $2, completed_timestamp = CURRENT_TIMESTAMP,
		    total_records_retrieved = $3, execution_time_ms = $4,
		    node_responses = $5, error_summary = $6
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, string(status), totalRecords, execTimeMs, responsesJSON, errSummary)
	return err
}

func (p *PGStore) CreateVocabulary(ctx context.Context, v *models.MetadataVocabulary) error {
	if v.ID == "" {
		v.ID = "vocab-" + uuid.New().String()[:8]
	}
	rulesJSON, _ := json.Marshal(v.MappingRules)
	query := `
		INSERT INTO statfederation.metadata_vocabularies (
			id, vocabulary_name, standard_framework, source_agency,
			target_canonical_concept, source_concept_term, mapping_rules,
			transformation_expression, status, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := p.db.ExecContext(ctx, query,
		v.ID, v.VocabularyName, v.StandardFramework, v.SourceAgency,
		v.TargetCanonicalConcept, v.SourceConceptTerm, rulesJSON,
		v.TransformationExpression, v.Status, v.TenantID,
	)
	return err
}

func (p *PGStore) ListVocabularies(ctx context.Context, tenantID, agency string) ([]*models.MetadataVocabulary, error) {
	query := `
		SELECT id, vocabulary_name, standard_framework, source_agency,
		       target_canonical_concept, source_concept_term, mapping_rules,
		       COALESCE(transformation_expression, ''), status, tenant_id, created_at, updated_at
		FROM statfederation.metadata_vocabularies
		WHERE ($1 = '' OR tenant_id = $1 OR tenant_id = 'default')
		  AND ($2 = '' OR source_agency = $2)
		ORDER BY vocabulary_name ASC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, agency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vocabs := make([]*models.MetadataVocabulary, 0)
	for rows.Next() {
		var v models.MetadataVocabulary
		var rulesJSON []byte
		if err := rows.Scan(
			&v.ID, &v.VocabularyName, &v.StandardFramework, &v.SourceAgency,
			&v.TargetCanonicalConcept, &v.SourceConceptTerm, &rulesJSON,
			&v.TransformationExpression, &v.Status, &v.TenantID, &v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(rulesJSON, &v.MappingRules)
		vocabs = append(vocabs, &v)
	}
	return vocabs, nil
}

func (p *PGStore) CreateTreaty(ctx context.Context, t *models.DiplomaticTreaty) error {
	if t.ID == "" {
		t.ID = "treaty-" + uuid.New().String()[:8]
	}
	partnersJSON, _ := json.Marshal(t.PartnerStates)
	rulesJSON, _ := json.Marshal(t.ComplianceRules)

	query := `
		INSERT INTO statfederation.diplomatic_treaties (
			id, treaty_code, title, partner_states, jurisdiction,
			framework_type, status, ratification_date, expiry_date,
			governing_body, compliance_rules, data_localization_required,
			encryption_standard, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := p.db.ExecContext(ctx, query,
		t.ID, t.TreatyCode, t.Title, partnersJSON, t.Jurisdiction,
		t.FrameworkType, t.Status, t.RatificationDate, t.ExpiryDate,
		t.GoverningBody, rulesJSON, t.DataLocalizationRequired,
		t.EncryptionStandard, t.TenantID,
	)
	return err
}

func (p *PGStore) GetTreatyByID(ctx context.Context, id string) (*models.DiplomaticTreaty, error) {
	query := `
		SELECT id, treaty_code, title, partner_states, jurisdiction,
		       framework_type, status, ratification_date, expiry_date,
		       governing_body, compliance_rules, data_localization_required,
		       encryption_standard, tenant_id, created_at, updated_at
		FROM statfederation.diplomatic_treaties WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var t models.DiplomaticTreaty
	var partnersJSON, rulesJSON []byte

	err := row.Scan(
		&t.ID, &t.TreatyCode, &t.Title, &partnersJSON, &t.Jurisdiction,
		&t.FrameworkType, &t.Status, &t.RatificationDate, &t.ExpiryDate,
		&t.GoverningBody, &rulesJSON, &t.DataLocalizationRequired,
		&t.EncryptionStandard, &t.TenantID, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(partnersJSON, &t.PartnerStates)
	_ = json.Unmarshal(rulesJSON, &t.ComplianceRules)
	return &t, nil
}

func (p *PGStore) ListTreaties(ctx context.Context, tenantID, jurisdiction string) ([]*models.DiplomaticTreaty, error) {
	query := `
		SELECT id, treaty_code, title, partner_states, jurisdiction,
		       framework_type, status, ratification_date, expiry_date,
		       governing_body, compliance_rules, data_localization_required,
		       encryption_standard, tenant_id, created_at, updated_at
		FROM statfederation.diplomatic_treaties
		WHERE ($1 = '' OR tenant_id = $1 OR tenant_id = 'default')
		  AND ($2 = '' OR jurisdiction = $2)
		ORDER BY title ASC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, jurisdiction)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	treaties := make([]*models.DiplomaticTreaty, 0)
	for rows.Next() {
		var t models.DiplomaticTreaty
		var partnersJSON, rulesJSON []byte
		if err := rows.Scan(
			&t.ID, &t.TreatyCode, &t.Title, &partnersJSON, &t.Jurisdiction,
			&t.FrameworkType, &t.Status, &t.RatificationDate, &t.ExpiryDate,
			&t.GoverningBody, &rulesJSON, &t.DataLocalizationRequired,
			&t.EncryptionStandard, &t.TenantID, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(partnersJSON, &t.PartnerStates)
		_ = json.Unmarshal(rulesJSON, &t.ComplianceRules)
		treaties = append(treaties, &t)
	}
	return treaties, nil
}

func (p *PGStore) CreateReport(ctx context.Context, r *models.InternationalReport) error {
	if r.ID == "" {
		r.ID = "rep-" + uuid.New().String()[:8]
	}
	indsJSON, _ := json.Marshal(r.TransferredIndicators)
	receiptJSON, _ := json.Marshal(r.AcknowledgementReceipt)

	query := `
		INSERT INTO statfederation.international_reports (
			id, report_title, destination_body, reporting_period,
			status, submission_hash, transferred_indicators,
			compliance_passed, compliance_notes, submitted_by,
			submitted_at, acknowledgement_receipt, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := p.db.ExecContext(ctx, query,
		r.ID, r.ReportTitle, r.DestinationBody, r.ReportingPeriod,
		r.Status, r.SubmissionHash, indsJSON,
		r.CompliancePassed, r.ComplianceNotes, r.SubmittedBy,
		r.SubmittedAt, receiptJSON, r.TenantID,
	)
	return err
}

func (p *PGStore) GetReportByID(ctx context.Context, id string) (*models.InternationalReport, error) {
	query := `
		SELECT id, report_title, destination_body, reporting_period,
		       status, COALESCE(submission_hash, ''), transferred_indicators,
		       compliance_passed, COALESCE(compliance_notes, ''), COALESCE(submitted_by, ''),
		       submitted_at, acknowledgement_receipt, tenant_id, created_at, updated_at
		FROM statfederation.international_reports WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)
	var r models.InternationalReport
	var indsJSON, receiptJSON []byte

	err := row.Scan(
		&r.ID, &r.ReportTitle, &r.DestinationBody, &r.ReportingPeriod,
		&r.Status, &r.SubmissionHash, &indsJSON,
		&r.CompliancePassed, &r.ComplianceNotes, &r.SubmittedBy,
		&r.SubmittedAt, &receiptJSON, &r.TenantID, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(indsJSON, &r.TransferredIndicators)
	_ = json.Unmarshal(receiptJSON, &r.AcknowledgementReceipt)
	return &r, nil
}

func (p *PGStore) ListReports(ctx context.Context, tenantID, destination string) ([]*models.InternationalReport, error) {
	query := `
		SELECT id, report_title, destination_body, reporting_period,
		       status, COALESCE(submission_hash, ''), transferred_indicators,
		       compliance_passed, COALESCE(compliance_notes, ''), COALESCE(submitted_by, ''),
		       submitted_at, acknowledgement_receipt, tenant_id, created_at, updated_at
		FROM statfederation.international_reports
		WHERE ($1 = '' OR tenant_id = $1 OR tenant_id = 'default')
		  AND ($2 = '' OR destination_body = $2)
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, destination)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := make([]*models.InternationalReport, 0)
	for rows.Next() {
		var r models.InternationalReport
		var indsJSON, receiptJSON []byte
		if err := rows.Scan(
			&r.ID, &r.ReportTitle, &r.DestinationBody, &r.ReportingPeriod,
			&r.Status, &r.SubmissionHash, &indsJSON,
			&r.CompliancePassed, &r.ComplianceNotes, &r.SubmittedBy,
			&r.SubmittedAt, &receiptJSON, &r.TenantID, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(indsJSON, &r.TransferredIndicators)
		_ = json.Unmarshal(receiptJSON, &r.AcknowledgementReceipt)
		reports = append(reports, &r)
	}
	return reports, nil
}

func (p *PGStore) UpdateReportStatus(ctx context.Context, id string, status string, receipt map[string]interface{}) error {
	receiptJSON, _ := json.Marshal(receipt)
	query := `
		UPDATE statfederation.international_reports
		SET status = $2, submitted_at = CURRENT_TIMESTAMP, acknowledgement_receipt = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := p.db.ExecContext(ctx, query, id, status, receiptJSON)
	return err
}

func (p *PGStore) IndexRemoteResource(ctx context.Context, res *models.FederatedSearchResult, tenantID string) error {
	if res.RemoteResourceID == "" {
		res.RemoteResourceID = uuid.New().String()
	}
	id := fmt.Sprintf("%s:%s", res.NodeID, res.RemoteResourceID)
	kwJSON, _ := json.Marshal(res.Keywords)

	query := `
		INSERT INTO statfederation.federated_search_indices (
			id, node_id, resource_type, remote_resource_id, title,
			abstract, keywords, classification, temporal_coverage,
			spatial_coverage, direct_access_url, tenant_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			abstract = EXCLUDED.abstract,
			keywords = EXCLUDED.keywords,
			cached_at = CURRENT_TIMESTAMP
	`
	_, err := p.db.ExecContext(ctx, query,
		id, res.NodeID, res.ResourceType, res.RemoteResourceID, res.Title,
		res.Abstract, kwJSON, res.Classification, res.TemporalCoverage,
		res.SpatialCoverage, res.DirectAccessURL, tenantID,
	)
	return err
}

func (p *PGStore) SearchFederatedResources(ctx context.Context, tenantID, queryStr, resourceType string) ([]*models.FederatedSearchResult, error) {
	query := `
		SELECT s.node_id, COALESCE(n.name, 'External Partner'), s.resource_type,
		       s.remote_resource_id, s.title, COALESCE(s.abstract, ''),
		       s.keywords, s.classification, COALESCE(s.temporal_coverage, ''),
		       COALESCE(s.spatial_coverage, ''), COALESCE(s.direct_access_url, '')
		FROM statfederation.federated_search_indices s
		LEFT JOIN statfederation.federated_nodes n ON s.node_id = n.id
		WHERE ($1 = '' OR s.tenant_id = $1 OR s.tenant_id = 'default')
		  AND ($2 = '' OR s.resource_type = $2)
		  AND ($3 = '' OR s.title ILIKE '%' || $3 || '%' OR s.abstract ILIKE '%' || $3 || '%')
		LIMIT 50
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, resourceType, queryStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*models.FederatedSearchResult, 0)
	for rows.Next() {
		var item models.FederatedSearchResult
		var kwJSON []byte
		if err := rows.Scan(
			&item.NodeID, &item.NodeName, &item.ResourceType,
			&item.RemoteResourceID, &item.Title, &item.Abstract,
			&kwJSON, &item.Classification, &item.TemporalCoverage,
			&item.SpatialCoverage, &item.DirectAccessURL,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(kwJSON, &item.Keywords)
		item.Score = 0.95
		results = append(results, &item)
	}
	return results, nil
}

func (p *PGStore) LogComplianceEvent(ctx context.Context, logEntry *models.ComplianceAuditLog) error {
	if logEntry.ID == "" {
		logEntry.ID = "audit-" + uuid.New().String()[:8]
	}
	rulesJSON, _ := json.Marshal(logEntry.AppliedRules)
	redactedJSON, _ := json.Marshal(logEntry.RedactedFields)

	query := `
		INSERT INTO statfederation.compliance_audit_logs (
			id, actor_user_id, actor_tenant_id, action,
			source_jurisdiction, target_jurisdiction, resource_type,
			resource_id, decision, applied_rules, redacted_fields,
			policy_hash, reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := p.db.ExecContext(ctx, query,
		logEntry.ID, logEntry.ActorUserID, logEntry.ActorTenantID, logEntry.Action,
		logEntry.SourceJurisdiction, logEntry.TargetJurisdiction, logEntry.ResourceType,
		logEntry.ResourceID, logEntry.Decision, rulesJSON, redactedJSON,
		logEntry.PolicyHash, logEntry.Reason,
	)
	return err
}

func (p *PGStore) ListComplianceLogs(ctx context.Context, tenantID string, limit int) ([]*models.ComplianceAuditLog, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, event_timestamp, actor_user_id, actor_tenant_id, action,
		       source_jurisdiction, target_jurisdiction, resource_type,
		       resource_id, decision, applied_rules, redacted_fields,
		       policy_hash, COALESCE(reason, '')
		FROM statfederation.compliance_audit_logs
		WHERE ($1 = '' OR actor_tenant_id = $1 OR actor_tenant_id = 'default')
		ORDER BY event_timestamp DESC
		LIMIT $2
	`
	rows, err := p.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*models.ComplianceAuditLog, 0)
	for rows.Next() {
		var l models.ComplianceAuditLog
		var rulesJSON, redactedJSON []byte
		if err := rows.Scan(
			&l.ID, &l.EventTimestamp, &l.ActorUserID, &l.ActorTenantID, &l.Action,
			&l.SourceJurisdiction, &l.TargetJurisdiction, &l.ResourceType,
			&l.ResourceID, &l.Decision, &rulesJSON, &redactedJSON,
			&l.PolicyHash, &l.Reason,
		); err != nil {
			continue
		}
		_ = json.Unmarshal(rulesJSON, &l.AppliedRules)
		_ = json.Unmarshal(redactedJSON, &l.RedactedFields)
		logs = append(logs, &l)
	}
	return logs, nil
}

func (p *PGStore) CreateObjectLink(ctx context.Context, link *models.ObjectLink) error {
	query := `
		INSERT INTO object_links (
			source_type, source_id, target_type, target_id,
			relationship, tenant_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (source_type, source_id, target_type, target_id, relationship) DO NOTHING
	`
	_, err := p.db.ExecContext(ctx, query,
		link.SourceType, link.SourceID, link.TargetType, link.TargetID,
		link.Relationship, link.TenantID, link.CreatedBy,
	)
	return err
}

func (p *PGStore) GetObjectLinks(ctx context.Context, sourceType, sourceID, tenantID string) ([]*models.ObjectLink, error) {
	query := `
		SELECT id, source_type, source_id, target_type, target_id,
		       relationship, tenant_id, COALESCE(created_by, ''), created_at
		FROM object_links
		WHERE (source_type = $1 AND source_id = $2)
		   OR (target_type = $1 AND target_id = $2)
		  AND ($3 = '' OR tenant_id = $3 OR tenant_id = 'tenant-alpha')
		ORDER BY created_at DESC
	`
	rows, err := p.db.QueryContext(ctx, query, sourceType, sourceID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]*models.ObjectLink, 0)
	for rows.Next() {
		var l models.ObjectLink
		if err := rows.Scan(
			&l.ID, &l.SourceType, &l.SourceID, &l.TargetType, &l.TargetID,
			&l.Relationship, &l.TenantID, &l.CreatedBy, &l.CreatedAt,
		); err != nil {
			continue
		}
		links = append(links, &l)
	}
	return links, nil
}
