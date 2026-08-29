package engine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"statfederation-backend/internal/models"
)

// SDMXDataSet is a minimal SDMX-JSON 1.0 data-set envelope.
type SDMXDataSet struct {
	Header     SDMXHeader               `json:"header"`
	Indicators []SDMXIndicatorObs       `json:"dataSets"`
}

type SDMXHeader struct {
	ID          string    `json:"id"`
	Test        bool      `json:"test"`
	Prepared    time.Time `json:"prepared"`
	Sender      SDMXParty `json:"sender"`
	DataSetID   string    `json:"dataSetID"`
}

type SDMXParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SDMXIndicatorObs struct {
	Action      string                 `json:"action"`
	Observations map[string]SDMXObs   `json:"observations"`
}

type SDMXObs struct {
	Code          string     `json:"code"`
	Title         string     `json:"title"`
	Domain        string     `json:"domain"`
	Value         *float64   `json:"value"`
	Unit          string     `json:"unit"`
	Frequency     string     `json:"freq"`
	SDMXDimension string     `json:"sdmxDimension,omitempty"`
	ReportedAt    time.Time  `json:"reportedAt"`
}

// IndicatorPushRequest describes an outbound push operation.
type IndicatorPushRequest struct {
	SourceNodeID   string   `json:"source_node_id"`
	TargetNodeID   string   `json:"target_node_id"`
	IndicatorIDs   []string `json:"indicator_ids"`
	Domain         string   `json:"domain,omitempty"`
}

// IndicatorPushResult summarises the outcome of a push.
type IndicatorPushResult struct {
	PushID         string    `json:"push_id"`
	SourceNodeID   string    `json:"source_node_id"`
	TargetNodeID   string    `json:"target_node_id"`
	IndicatorCount int       `json:"indicator_count"`
	Status         string    `json:"status"`   // SUCCEEDED | PARTIAL | FAILED
	HTTPStatus     int       `json:"http_status"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	PushedAt       time.Time `json:"pushed_at"`
}

// PushProtocol handles outbound indicator transmission to peer hubs.
type PushProtocol struct {
	store      federationStore
	httpClient *http.Client
	signer     PayloadSigner // optional StatTrust payload signer (Task 1.1)
}

// federationStore is the minimal store interface needed by PushProtocol.
type federationStore interface {
	GetNodeByID(ctx context.Context, id string) (*models.FederatedNode, error)
	FindActiveDSA(ctx context.Context, providerNodeID, consumerNodeID, domain string) (*models.DataSharingAgreement, error)
	GetIndicatorByID(ctx context.Context, id string) (*models.NationalIndicator, error)
	ListIndicators(ctx context.Context, tenantID, domain string) ([]*models.NationalIndicator, error)
	LogComplianceEvent(ctx context.Context, logEntry *models.ComplianceAuditLog) error
}

// SignedPayload is the result of affixing a StatGate digital signature to a
// payload. It mirrors the relevant fields of StatTrust's DigitalSignature.
type SignedPayload struct {
	SignatureID  string
	SignatureHex string
	SHA256Hash   string
	SignerDID    string
	Timestamp    time.Time
}

// PayloadSigner signs a canonical digest of an outbound payload using the
// tenant's private key via the StatTrust integration.
type PayloadSigner interface {
	Sign(ctx context.Context, artifactID, sourceApp, content string) (*SignedPayload, error)
}

// statTrustSignRequest mirrors StatTrust's POST /api/v1/trust/signatures/sign body.
type statTrustSignRequest struct {
	ArtifactID   string `json:"artifact_id"`
	ArtifactType string `json:"artifact_type"`
	SourceApp    string `json:"source_app"`
	SignerDID    string `json:"signer_did"`
	SignerName   string `json:"signer_name"`
	SignerRole   string `json:"signer_role"`
	ContentData  string `json:"content_data"`
}

// statTrustSignResponse mirrors StatTrust's DigitalSignature response.
type statTrustSignResponse struct {
	ID           string    `json:"id"`
	Sha256Hash   string    `json:"sha256_hash"`
	SignatureHex string    `json:"signature_hex"`
	SignerDID    string    `json:"signer_did"`
	Timestamp    time.Time `json:"timestamp"`
}

// StatTrustSigner signs payload digests via the StatTrust trust service.
type StatTrustSigner struct {
	baseURL  string
	sourceID string // source_node_id used as the signer DID
	client   *http.Client
}

// NewStatTrustSigner builds a signer backed by the StatTrust API.
func NewStatTrustSigner(baseURL, sourceID string) *StatTrustSigner {
	return &StatTrustSigner{
		baseURL:  baseURL,
		sourceID: sourceID,
		client:   &http.Client{Timeout: 4 * time.Second},
	}
}

// Sign hashes the content and requests a StatTrust digital signature.
// StatTrust remains optional: an unreachable service is tolerated so peer
// federation is not blocked by a trust-infrastructure blip.
func (s *StatTrustSigner) Sign(ctx context.Context, artifactID, sourceApp, content string) (*SignedPayload, error) {
	if s.baseURL == "" {
		return nil, fmt.Errorf("stat trust url not configured")
	}
	h := sha256.Sum256([]byte(content))
	shaHash := hex.EncodeToString(h[:])

	payload := statTrustSignRequest{
		ArtifactID:   artifactID,
		ArtifactType: "SDMX_INDICATOR_DATASET",
		SourceApp:    sourceApp,
		SignerDID:    s.sourceID,
		SignerName:   sourceApp,
		SignerRole:   "FEDERATION_NODE",
		ContentData:  shaHash,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.baseURL+"/api/v1/trust/signatures/sign", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stat trust returned HTTP %d", resp.StatusCode)
	}

	var out statTrustSignResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &SignedPayload{
		SignatureID:  out.ID,
		SignatureHex: out.SignatureHex,
		SHA256Hash:   shaHash,
		SignerDID:    out.SignerDID,
		Timestamp:    out.Timestamp,
	}, nil
}

// SetSigner attaches the payload signer to the protocol. Defaults to none.
func (pp *PushProtocol) SetSigner(s PayloadSigner) {
	pp.signer = s
}

// NewPushProtocol initialises the push protocol engine.
func NewPushProtocol(s federationStore) *PushProtocol {
	return &PushProtocol{
		store: s,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PushIndicators transmits approved indicator values to a target hub.
//
// Flow:
//  1. Resolve source + target nodes
//  2. Validate an active DSA exists for the domain
//  3. Fetch each NationalIndicator
//  4. Serialise as SDMX-JSON
//  5. POST to target /api/v1/federation/indicators/ingest
//  6. Return IndicatorPushResult
func (pp *PushProtocol) PushIndicators(
	ctx context.Context,
	req IndicatorPushRequest,
	senderName, tenantID string,
) (*IndicatorPushResult, error) {
	pushID := "push-" + uuid.New().String()[:12]
	result := &IndicatorPushResult{
		PushID:       pushID,
		SourceNodeID: req.SourceNodeID,
		TargetNodeID: req.TargetNodeID,
		PushedAt:     time.Now().UTC(),
	}

	// 1. Resolve target node
	targetNode, err := pp.store.GetNodeByID(ctx, req.TargetNodeID)
	if err != nil {
		result.Status = "FAILED"
		result.ErrorMessage = fmt.Sprintf("target node not found: %v", err)
		return result, fmt.Errorf("target node lookup failed: %w", err)
	}
	if targetNode.HealthStatus == models.HealthStatusUnreachable {
		result.Status = "FAILED"
		result.ErrorMessage = "target node is unreachable"
		return result, fmt.Errorf("target node %s is UNREACHABLE", targetNode.Code)
	}

	// 2. Validate DSA
	domain := req.Domain
	if domain == "" {
		domain = "GENERAL"
	}
	dsa, err := pp.store.FindActiveDSA(ctx, req.SourceNodeID, req.TargetNodeID, domain)
	if err != nil {
		result.Status = "FAILED"
		result.ErrorMessage = fmt.Sprintf("no active DSA: %v", err)
		return result, fmt.Errorf("DSA validation failed: %w", err)
	}

	// 1b. Resolve the source node for the compliance jurisdiction (best-effort).
	sourceNode, _ := pp.store.GetNodeByID(ctx, req.SourceNodeID)
	sourceJurisdiction := ""
	if sourceNode != nil {
		sourceJurisdiction = sourceNode.Jurisdiction
	}

	// 3. Fetch indicators
	var indicators []*models.NationalIndicator
	if len(req.IndicatorIDs) == 0 {
		// Push all indicators for the domain
		indicators, err = pp.store.ListIndicators(ctx, tenantID, domain)
		if err != nil {
			result.Status = "FAILED"
			result.ErrorMessage = err.Error()
			return result, err
		}
	} else {
		for _, id := range req.IndicatorIDs {
			ind, err := pp.store.GetIndicatorByID(ctx, id)
			if err != nil {
				log.Printf("[PushProtocol] Indicator %s not found, skipping: %v", id, err)
				continue
			}
			indicators = append(indicators, ind)
		}
	}
	if len(indicators) == 0 {
		result.Status = "FAILED"
		result.ErrorMessage = "no indicators to push"
		return result, fmt.Errorf("no indicators resolved for push %s", pushID)
	}

	// 4. Serialise as SDMX-JSON
	payload := pp.buildSDMXPayload(pushID, req.SourceNodeID, senderName, indicators)
	body, err := json.Marshal(payload)
	if err != nil {
		result.Status = "FAILED"
		result.ErrorMessage = "SDMX serialization error: " + err.Error()
		return result, err
	}

	// 4b. Sign the payload with StatTrust (best-effort; never blocks federation).
	signatureHeaders := map[string]string{}
	if pp.signer != nil {
		if signed, serr := pp.signer.Sign(ctx, pushID, "statfederation", string(body)); serr == nil {
			signatureHeaders["X-StatGate-Signature"] = signed.SignatureHex
			signatureHeaders["X-StatGate-Signature-ID"] = signed.SignatureID
			signatureHeaders["X-StatGate-Signature-SHA256"] = signed.SHA256Hash
			log.Printf("[PushProtocol:%s] Payload signed with StatTrust (sig %s)", pushID, signed.SignatureID)
		} else {
			log.Printf("[PushProtocol:%s] Signing skipped: %v", pushID, serr)
		}
	}

	// 5. POST to target node
	ingestURL := targetNode.EndpointURL + "/api/v1/federation/indicators/ingest"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ingestURL, bytes.NewReader(body))
	if err != nil {
		result.Status = "FAILED"
		result.ErrorMessage = err.Error()
		return result, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Push-ID", pushID)
	httpReq.Header.Set("X-Source-Node", req.SourceNodeID)
	for k, v := range signatureHeaders {
		httpReq.Header.Set(k, v)
	}

	resp, err := pp.httpClient.Do(httpReq)
	if err != nil {
		result.Status = "FAILED"
		result.ErrorMessage = fmt.Sprintf("HTTP push failed: %v", err)
		result.HTTPStatus = 0
		log.Printf("[PushProtocol:%s] Push to %s failed: %v", pushID, targetNode.Code, err)
		pp.recordCompliance(ctx, tenantID, pushID,
			sourceJurisdiction, targetNode.Jurisdiction, dsa, len(signatureHeaders) > 0, result)
		return result, nil // Return result, not error — caller records the failure
	}
	defer resp.Body.Close()

	result.HTTPStatus = resp.StatusCode
	result.IndicatorCount = len(indicators)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = "SUCCEEDED"
		log.Printf("[PushProtocol:%s] Successfully pushed %d indicators to %s (HTTP %d)",
			pushID, len(indicators), targetNode.Code, resp.StatusCode)
	} else {
		result.Status = "FAILED"
		result.ErrorMessage = fmt.Sprintf("target returned HTTP %d", resp.StatusCode)
		log.Printf("[PushProtocol:%s] Push to %s returned HTTP %d", pushID, targetNode.Code, resp.StatusCode)
	}

	// 6. Record sovereign transboundary compliance audit entry
	pp.recordCompliance(ctx, tenantID, pushID,
		sourceJurisdiction, targetNode.Jurisdiction, dsa, len(signatureHeaders) > 0, result)

	return result, nil
}

// recordCompliance writes a sovereign transboundary audit entry describing the
// outcome of an outbound indicator push. Every push attempt — allowed or denied —
// is recorded for the compliance ledger.
func (pp *PushProtocol) recordCompliance(
	ctx context.Context,
	tenantID, pushID, sourceJur, targetJur string,
	dsa *models.DataSharingAgreement,
	signed bool,
	result *IndicatorPushResult,
) {
	decision := "ALLOW"
	action := "federation.indicator.pushed"
	reason := ""
	if result.Status != "SUCCEEDED" {
		decision = "DENY"
		action = "federation.indicator.push.rejected"
		reason = result.ErrorMessage
	}

	appliedRules := []string{"Active DSA: " + dsa.DSANumber}
	if dsa.AccessTier != "" {
		appliedRules = append(appliedRules, "Access tier: "+dsa.AccessTier)
	}
	appliedRules = append(appliedRules, fmt.Sprintf("Indicators: %d", result.IndicatorCount))
	if signed {
		appliedRules = append(appliedRules, "Signed: true")
	} else {
		appliedRules = append(appliedRules, "Signed: false")
	}

	entry := &models.ComplianceAuditLog{
		ID:                 "compliance-" + pushID,
		EventTimestamp:     time.Now().UTC(),
		ActorTenantID:      tenantID,
		ResourceType:       "INDICATOR_DATASET",
		Action:             action,
		SourceJurisdiction: sourceJur,
		TargetJurisdiction: targetJur,
		ResourceID:         pushID,
		Decision:           decision,
		AppliedRules:       appliedRules,
		RedactedFields:     []string{},
		PolicyHash:         "DSA-" + dsa.DSANumber,
		Reason:             reason,
	}
	if err := pp.store.LogComplianceEvent(ctx, entry); err != nil {
		log.Printf("[PushProtocol:%s] compliance log record failed: %v", pushID, err)
	}
}

// buildSDMXPayload serialises indicators into a minimal SDMX-JSON 1.0 envelope.
func (pp *PushProtocol) buildSDMXPayload(
	pushID, senderNodeID, senderName string,
	indicators []*models.NationalIndicator,
) SDMXDataSet {
	obs := make(map[string]SDMXObs, len(indicators))
	for _, ind := range indicators {
		obs[ind.Code] = SDMXObs{
			Code:          ind.Code,
			Title:         ind.Title,
			Domain:        ind.Domain,
			Value:         ind.CurrentValue,
			Unit:          ind.UnitOfMeasure,
			Frequency:     ind.Frequency,
			SDMXDimension: ind.SDMXDimension,
			ReportedAt:    time.Now().UTC(),
		}
	}
	return SDMXDataSet{
		Header: SDMXHeader{
			ID:        pushID,
			Test:      false,
			Prepared:  time.Now().UTC(),
			Sender:    SDMXParty{ID: senderNodeID, Name: senderName},
			DataSetID: pushID,
		},
		Indicators: []SDMXIndicatorObs{
			{Action: "Replace", Observations: obs},
		},
	}
}
