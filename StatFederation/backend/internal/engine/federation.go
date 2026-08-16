package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

// FederationEngine manages NSS node lifecycle, heartbeats, and data exchange boundaries
type FederationEngine struct {
	store store.Store
	mu    sync.RWMutex
}

// NewFederationEngine initializes the NSS engine
func NewFederationEngine(s store.Store) *FederationEngine {
	return &FederationEngine{
		store: s,
	}
}

// RegisterNSSNode validates and persists a new participating statistical node
func (fe *FederationEngine) RegisterNSSNode(ctx context.Context, node *models.FederatedNode) error {
	if node.Name == "" || node.Code == "" || node.EndpointURL == "" {
		return errors.New("node name, code, and endpoint_url are required")
	}
	if node.NodeType == "" {
		node.NodeType = models.NodeTypeNSSAgency
	}
	if node.HealthStatus == "" {
		node.HealthStatus = models.HealthStatusHealthy
	}
	if node.TrustLevel == "" {
		node.TrustLevel = models.TrustLevelTier2DomesticAgency
	}
	if len(node.Protocols) == 0 {
		node.Protocols = []string{"StatGate-JSON", "SDMX-REST"}
	}
	if len(node.Capabilities) == 0 {
		node.Capabilities = []string{"AGGREGATE_QUERY"}
	}

	return fe.store.RegisterNode(ctx, node)
}

// Heartbeat checks and updates real-time node availability and latency
func (fe *FederationEngine) ProcessHeartbeat(ctx context.Context, nodeID string, status models.HealthStatus, latencyMs int) error {
	if nodeID == "" {
		return errors.New("node ID is required")
	}
	return fe.store.UpdateNodeStatus(ctx, nodeID, status, latencyMs)
}

// ValidateExchangeAuthority checks if a consumer node is authorized under an active DSA
func (fe *FederationEngine) ValidateExchangeAuthority(ctx context.Context, providerNodeID, consumerNodeID, domain string) (*models.DataSharingAgreement, error) {
	dsa, err := fe.store.FindActiveDSA(ctx, providerNodeID, consumerNodeID, domain)
	if err != nil {
		return nil, fmt.Errorf("exchange authorization failed: %w", err)
	}

	// Check rate limit and daily quota
	if dsa.DailyQuota > 0 && dsa.CurrentDailyUsage >= dsa.DailyQuota {
		return nil, fmt.Errorf("daily query quota (%d) exceeded for DSA %s", dsa.DailyQuota, dsa.DSANumber)
	}

	return dsa, nil
}

// ApproveDSA transitions a data sharing agreement to active status
func (fe *FederationEngine) ApproveDSA(ctx context.Context, dsaID, approvedBy string) error {
	dsa, err := fe.store.GetDSAByID(ctx, dsaID)
	if err != nil {
		return err
	}
	if dsa.Status == models.DSAStatusRevoked || dsa.Status == models.DSAStatusExpired {
		return fmt.Errorf("cannot approve DSA in %s state", dsa.Status)
	}
	return fe.store.UpdateDSAStatus(ctx, dsaID, models.DSAStatusActive, approvedBy)
}

// RevokeDSA invalidates a data sharing agreement
func (fe *FederationEngine) RevokeDSA(ctx context.Context, dsaID, revokedBy string) error {
	return fe.store.UpdateDSAStatus(ctx, dsaID, models.DSAStatusRevoked, revokedBy)
}

// CheckNodeHealth probes all registered nodes and updates their status
func (fe *FederationEngine) ProbeAllNodes(ctx context.Context, tenantID string) ([]*models.FederatedNode, error) {
	nodes, err := fe.store.ListNodes(ctx, tenantID, "", "")
	if err != nil {
		return nil, err
	}

	for _, n := range nodes {
		// Mocked probe for simulation / unit tests
		status := models.HealthStatusHealthy
		latency := 25
		if n.Jurisdiction == "GLOBAL" {
			latency = 180
		}
		_ = fe.store.UpdateNodeStatus(ctx, n.ID, status, latency)
		log.Printf("[FederationEngine:Probe] Node %s (%s) healthy, latency: %dms", n.Name, n.Code, latency)
	}

	return fe.store.ListNodes(ctx, tenantID, "", "")
}
