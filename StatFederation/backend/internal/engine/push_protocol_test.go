package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

// fakeSigner records signing calls and returns a deterministic signature.
type fakeSigner struct {
	mu           sync.Mutex
	calls        int
	lastArtifact string
	lastSource   string
	lastContent  string
}

func (f *fakeSigner) Sign(_ context.Context, artifactID, sourceApp, content string) (*SignedPayload, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastArtifact = artifactID
	f.lastSource = sourceApp
	f.lastContent = content
	return &SignedPayload{
		SignatureID:  "sig-test-1",
		SignatureHex: "deadbeefcafe",
		SHA256Hash:   "sha-test",
		SignerDID:    "node-test-src",
		Timestamp:    time.Now().UTC(),
	}, nil
}

func (f *fakeSigner) snapshot() (int, string, string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls, f.lastArtifact, f.lastSource, f.lastContent
}

// setupPushFixture registers a source node, a target node (whose endpoint is the
// provided httptest server), and an active DSA for the "health" domain.
func setupPushFixture(t *testing.T, s *store.MemStore, targetURL string) (string, string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	src := &models.FederatedNode{
		ID:           "node-test-src",
		Name:         "Test Source Hub",
		Code:         "TEST-SRC",
		NodeType:     models.NodeTypeNSSAgency,
		Jurisdiction: "NATIONAL",
		EndpointURL:  "https://source.example/not-used",
		HealthStatus: models.HealthStatusHealthy,
		TrustLevel:   models.TrustLevelTier1Sovereign,
		TenantID:     "default",
	}
	if err := s.RegisterNode(ctx, src); err != nil {
		t.Fatalf("register source node: %v", err)
	}

	dst := &models.FederatedNode{
		ID:           "node-test-dst",
		Name:         "Test Target Hub",
		Code:         "TEST-DST",
		NodeType:     models.NodeTypeRegionalBloc,
		Jurisdiction: "EAST_AFRICA",
		EndpointURL:  targetURL,
		HealthStatus: models.HealthStatusHealthy,
		TrustLevel:   models.TrustLevelTier3RegionalPartner,
		TenantID:     "default",
	}
	if err := s.RegisterNode(ctx, dst); err != nil {
		t.Fatalf("register target node: %v", err)
	}

	dsa := &models.DataSharingAgreement{
		ID:               "dsa-test-001",
		DSANumber:        "DSA-TEST-2026-001",
		Title:            "Test Data Sharing Agreement",
		ProviderNodeID:   src.ID,
		ConsumerNodeID:   dst.ID,
		Status:           models.DSAStatusActive,
		AccessTier:       "INTER_AGENCY",
		PermittedDomains: []string{"health", "demographics", "sdg"},
		Purpose:          "Integration test",
		ValidFrom:        now.Add(-24 * time.Hour),
		ValidUntil:       now.Add(30 * 24 * time.Hour),
		DailyQuota:       100000,
		TenantID:         "default",
	}
	if err := s.CreateDSA(ctx, dsa); err != nil {
		t.Fatalf("create dsa: %v", err)
	}
	return src.ID, dst.ID
}

// receivingTarget returns an httptest server that captures the incoming push
// and responds with the supplied status code.
func receivingTarget(t *testing.T, status int, gotBody *[]byte, gotHeaders *http.Header) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*gotBody = b
		*gotHeaders = r.Header
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	t.Cleanup(ts.Close)
	return ts
}
// TestPushIndicatorsSignedSuccess verifies a full signed, DSA-authorised push:
// SDMX-JSON body, StatTrust signature headers, and an ALLOW compliance entry.
func TestPushIndicatorsSignedSuccess(t *testing.T) {
	mem := store.NewMemStore()
	var gotBody []byte
	var gotHeaders http.Header
	target := receivingTarget(t, http.StatusOK, &gotBody, &gotHeaders)

	srcID, dstID := setupPushFixture(t, mem, target.URL)

	protocol := NewPushProtocol(mem)
	signer := &fakeSigner{}
	protocol.SetSigner(signer)

	result, err := protocol.PushIndicators(context.Background(), IndicatorPushRequest{
		SourceNodeID: srcID,
		TargetNodeID: dstID,
		IndicatorIDs: []string{"ind-sdg-311"},
		Domain:       "health",
	}, "Test Hub (default)", "default")
	if err != nil {
		t.Fatalf("PushIndicators returned error: %v", err)
	}
	if result.Status != "SUCCEEDED" {
		t.Fatalf("expected SUCCEEDED, got %q (%s)", result.Status, result.ErrorMessage)
	}
	if result.IndicatorCount != 1 {
		t.Errorf("expected 1 indicator, got %d", result.IndicatorCount)
	}

	// The signer must have seen the exact marshalled body.
	calls, artifact, sourceApp, content := signer.snapshot()
	if calls != 1 {
		t.Fatalf("expected signer to be called once, got %d", calls)
	}
	if content != string(gotBody) {
		t.Errorf("signed content does not match transmitted body")
	}
	if artifact != result.PushID {
		t.Errorf("expected artifact push id %s, got %s", result.PushID, artifact)
	}
	if sourceApp != "statfederation" {
		t.Errorf("expected source app statfederation, got %q", sourceApp)
	}

	// Signature headers must be present on the outgoing request.
	if h := gotHeaders.Get("X-StatGate-Signature"); h != "deadbeefcafe" {
		t.Errorf("expected X-StatGate-Signature header, got %q", h)
	}
	if h := gotHeaders.Get("X-StatGate-Signature-ID"); h != "sig-test-1" {
		t.Errorf("expected X-StatGate-Signature-ID header, got %q", h)
	}
	if h := gotHeaders.Get("X-Push-ID"); h != result.PushID {
		t.Errorf("expected X-Push-ID %s, got %q", result.PushID, h)
	}

	// The received body must be a valid SDMX dataset containing the indicator.
	var dataset SDMXDataSet
	if err := json.Unmarshal(gotBody, &dataset); err != nil {
		t.Fatalf("received body is not valid SDMX JSON: %v", err)
	}
	if dataset.Header.ID != result.PushID {
		t.Errorf("expected header id %s, got %s", result.PushID, dataset.Header.ID)
	}
	if len(dataset.Indicators) != 1 {
		t.Fatalf("expected 1 dataset block, got %d", len(dataset.Indicators))
	}
	obs, ok := dataset.Indicators[0].Observations["IND-SDG-3.1.1"]
	if !ok {
		t.Fatalf("expected observation for IND-SDG-3.1.1 in body")
	}
	if obs.Value == nil || *obs.Value != 214.5 {
		t.Errorf("expected value 214.5, got %v", obs.Value)
	}

	// Compliance ledger must contain an ALLOW entry for this push.
	logs, err := mem.ListComplianceLogs(context.Background(), "default", 10)
	if err != nil {
		t.Fatalf("list compliance logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected a compliance audit entry")
	}
	entry := logs[0]
	if entry.Decision != "ALLOW" {
		t.Errorf("expected ALLOW decision, got %s", entry.Decision)
	}
	if entry.Action != "federation.indicator.pushed" {
		t.Errorf("expected federation.indicator.pushed action, got %s", entry.Action)
	}
	if entry.ResourceID != result.PushID {
		t.Errorf("expected resource id %s, got %s", result.PushID, entry.ResourceID)
	}
	if entry.SourceJurisdiction != "NATIONAL" || entry.TargetJurisdiction != "EAST_AFRICA" {
		t.Errorf("unexpected jurisdictions: %s -> %s", entry.SourceJurisdiction, entry.TargetJurisdiction)
	}
	if !strings.Contains(strings.Join(entry.AppliedRules, "|"), "Signed: true") {
		t.Errorf("expected Signed: true rule, got %v", entry.AppliedRules)
	}
}
// TestPushIndicatorsDeniedWhenTargetFails verifies that a non-2xx target
// response produces a FAILED result and a DENY compliance entry.
func TestPushIndicatorsDeniedWhenTargetFails(t *testing.T) {
	mem := store.NewMemStore()
	var gotBody []byte
	var gotHeaders http.Header
	target := receivingTarget(t, http.StatusInternalServerError, &gotBody, &gotHeaders)

	srcID, dstID := setupPushFixture(t, mem, target.URL)
	protocol := NewPushProtocol(mem)
	protocol.SetSigner(&fakeSigner{})

	result, err := protocol.PushIndicators(context.Background(), IndicatorPushRequest{
		SourceNodeID: srcID,
		TargetNodeID: dstID,
		IndicatorIDs: []string{"ind-sdg-311"},
		Domain:       "health",
	}, "Test Hub (default)", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "FAILED" {
		t.Fatalf("expected FAILED, got %s", result.Status)
	}
	if result.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("expected HTTP 500, got %d", result.HTTPStatus)
	}

	logs, err := mem.ListComplianceLogs(context.Background(), "default", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected a compliance entry (err=%v)", err)
	}
	if logs[0].Decision != "DENY" {
		t.Errorf("expected DENY decision, got %s", logs[0].Decision)
	}
	if logs[0].Action != "federation.indicator.push.rejected" {
		t.Errorf("expected rejected action, got %s", logs[0].Action)
	}
}

// TestPushIndicatorsWithoutSigner verifies graceful operation when no signer is
// configured: the push still succeeds and is recorded as unsigned.
func TestPushIndicatorsWithoutSigner(t *testing.T) {
	mem := store.NewMemStore()
	var gotBody []byte
	var gotHeaders http.Header
	target := receivingTarget(t, http.StatusOK, &gotBody, &gotHeaders)

	srcID, dstID := setupPushFixture(t, mem, target.URL)
	protocol := NewPushProtocol(mem) // no signer attached

	result, err := protocol.PushIndicators(context.Background(), IndicatorPushRequest{
		SourceNodeID: srcID,
		TargetNodeID: dstID,
		IndicatorIDs: []string{"ind-sdg-311"},
		Domain:       "health",
	}, "Test Hub (default)", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "SUCCEEDED" {
		t.Fatalf("expected SUCCEEDED, got %s", result.Status)
	}
	if gotHeaders.Get("X-StatGate-Signature") != "" {
		t.Error("expected no signature header when no signer is configured")
	}

	logs, err := mem.ListComplianceLogs(context.Background(), "default", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected a compliance entry (err=%v)", err)
	}
	if !strings.Contains(strings.Join(logs[0].AppliedRules, "|"), "Signed: false") {
		t.Errorf("expected Signed: false rule, got %v", logs[0].AppliedRules)
	}
}

// TestStatTrustSignerHTTP verifies the StatTrust signer calls the trust API with
// a SHA-256 digest and parses the DigitalSignature response.
func TestStatTrustSignerHTTP(t *testing.T) {
	var gotPath string
	var gotContent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var req statTrustSignRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotContent = req.ContentData
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "sig-http-001",
			"sha256_hash": "` + req.ContentData + `",
			"signature_hex": "aabbcc",
			"signer_did": "node-test-src",
			"timestamp": "2026-08-25T12:00:00Z",
			"status": "VALID"
		}`))
	}))
	t.Cleanup(ts.Close)

	signer := NewStatTrustSigner(ts.URL, "node-test-src")
	signed, err := signer.Sign(context.Background(), "push-1", "statfederation", `{"hello":"sdmx"}`)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}
	if gotPath != "/api/v1/trust/signatures/sign" {
		t.Errorf("expected trust sign path, got %q", gotPath)
	}
	h := sha256.Sum256([]byte(`{"hello":"sdmx"}`))
	expectedHash := hex.EncodeToString(h[:])
	if gotContent != expectedHash {
		t.Errorf("expected hashed content %s, got %s", expectedHash, gotContent)
	}
	if signed.SignatureHex != "aabbcc" {
		t.Errorf("expected signature aabbcc, got %q", signed.SignatureHex)
	}
	if signed.SignatureID != "sig-http-001" {
		t.Errorf("expected signature id, got %q", signed.SignatureID)
	}
	if signed.SignerDID != "node-test-src" {
		t.Errorf("expected signer did, got %q", signed.SignerDID)
	}
}

// TestStatTrustSignerUnavailable confirms an unreachable trust service yields an
// error that callers absorb (best-effort signing).
func TestStatTrustSignerUnavailable(t *testing.T) {
	signer := NewStatTrustSigner("http://127.0.0.1:1", "node-test-src")
	if _, err := signer.Sign(context.Background(), "push-x", "statfederation", "data"); err == nil {
		t.Fatal("expected an error from an unreachable trust service")
	}
}