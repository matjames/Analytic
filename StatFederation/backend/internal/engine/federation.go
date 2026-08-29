package engine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

// FederationEngine manages NSS node lifecycle, heartbeats, and data exchange boundaries
type FederationEngine struct {
	store      store.Store
	mu         sync.RWMutex
	httpClient *http.Client
}

// NewFederationEngine initializes the NSS engine
func NewFederationEngine(s store.Store) *FederationEngine {
	return &FederationEngine{
		store: s,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// StartProbeLoop runs a background goroutine that probes all registered nodes
// every 60 seconds. Call this once after initialisation.
func (fe *FederationEngine) StartProbeLoop(ctx context.Context, tenantID string) {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, err := fe.ProbeAllNodes(ctx, tenantID)
				if err != nil {
					log.Printf("[FederationEngine:ProbeLoop] error: %v", err)
				}
			case <-ctx.Done():
				log.Println("[FederationEngine:ProbeLoop] stopping")
				return
			}
		}
	}()
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

// ProbeAllNodes performs real HTTP GET /health calls on every registered node,
// measures latency, and updates stored health status accordingly.
func (fe *FederationEngine) ProbeAllNodes(ctx context.Context, tenantID string) ([]*models.FederatedNode, error) {
	nodes, err := fe.store.ListNodes(ctx, tenantID, "", "")
	if err != nil {
		return nil, err
	}

	for _, n := range nodes {
		if n.EndpointURL == "" {
			continue
		}
		status, latency := fe.probeNode(ctx, n.EndpointURL)
		_ = fe.store.UpdateNodeStatus(ctx, n.ID, status, latency)
		log.Printf("[FederationEngine:Probe] Node %s (%s) → %s, latency: %dms",
			n.Name, n.Code, status, latency)
	}

	return fe.store.ListNodes(ctx, tenantID, "", "")
}

// probeNode does a single HTTP GET /health and returns status + latency in ms.
func (fe *FederationEngine) probeNode(ctx context.Context, endpointURL string) (models.HealthStatus, int) {
	start := time.Now()
	healthURL := endpointURL + "/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return models.HealthStatusUnreachable, 0
	}

	resp, err := fe.httpClient.Do(req)
	latencyMs := int(time.Since(start).Milliseconds())
	if err != nil {
		return models.HealthStatusUnreachable, latencyMs
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		if latencyMs > 2000 {
			return models.HealthStatusDegraded, latencyMs
		}
		return models.HealthStatusHealthy, latencyMs
	case resp.StatusCode == http.StatusServiceUnavailable:
		return models.HealthStatusDegraded, latencyMs
	default:
		return models.HealthStatusUnreachable, latencyMs
	}
}
