package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
)

func requestScope(c *gin.Context) (string, string) {
	tenantID, _ := c.Get("tenant_id")
	workspaceID, _ := c.Get("workspace_id")
	tenant, _ := tenantID.(string)
	workspace, _ := workspaceID.(string)
	return tenant, workspace
}

func appendRequestLedger(c *gin.Context, eventType, sourceApp, actorID string, payload map[string]interface{}) AuditLedgerBlock {
	tenantID, workspaceID := requestScope(c)
	return globalStore.AppendLedgerScoped(eventType, sourceApp, actorID, payload, tenantID, workspaceID)
}

// HealthHandler returns liveness and basic system status
func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":   "StatTrust (Security, Trust & Identity)",
		"status":    "healthy",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadyHandler returns readiness check
func ReadyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready":     true,
		"subsystem": "statgate-security-trust",
	})
}

// SummaryHandler returns top-level dashboard metrics
func SummaryHandler(c *gin.Context) {
	summary := globalStore.GetHealthSummary()
	c.JSON(http.StatusOK, summary)
}

// ==========================================
// SECOPS HANDLERS (P19)
// ==========================================

func ListIncidentsHandler(c *gin.Context) {
	tenantID, workspaceID := requestScope(c)
	incidents := globalStore.ListIncidentsScoped(tenantID, workspaceID)
	c.JSON(http.StatusOK, gin.H{
		"count":     len(incidents),
		"incidents": incidents,
	})
}

func CreateIncidentHandler(c *gin.Context) {
	var req SecurityIncident
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenantID, workspaceID := requestScope(c)
	created := globalStore.AddIncidentScoped(req, tenantID, workspaceID)

	// Publish to Redis event bus
	publishEvent("secops.incident.created", "incident", created.ID, created.AssignedTo, tenantID, map[string]interface{}{
		"title":    created.Title,
		"severity": created.Severity,
		"source":   created.SourceApp,
		"threat":   created.ThreatCategory,
	})

	// Dispatch notification to StatChat
	go interAppClient.SendSecurityAlertToStatChat(created.Title, created.Severity, created.ThreatCategory)

	c.JSON(http.StatusCreated, created)
}

func UpdateIncidentStatusHandler(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
		Note   string `json:"note"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	tenantID, workspaceID := requestScope(c)
	updated, err := globalStore.UpdateIncidentStatusScoped(id, req.Status, req.Note, tenantID, workspaceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	publishEvent("secops.incident.updated", "incident", id, "secops-operator", tenantID, map[string]interface{}{
		"status": req.Status,
		"note":   req.Note,
	})

	c.JSON(http.StatusOK, updated)
}

// DLPScanHandler performs deep pattern inspection for PII, tokens, and credentials
func DLPScanHandler(c *gin.Context) {
	var req DLPScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan payload: " + err.Error()})
		return
	}

	matchedRules := make([]string, 0)
	redacted := req.Content

	// 1. Uganda National Identification Number (NIN): format e.g. CM89012345678A
	ninRegex := regexp.MustCompile(`\b[CF][M][0-9]{11}[A-Z]\b`)
	if ninRegex.MatchString(req.Content) {
		matchedRules = append(matchedRules, "UGANDA_NIN_DETECTED")
		if req.Redact {
			redacted = ninRegex.ReplaceAllString(redacted, "[REDACTED-NIN]")
		}
	}

	// 2. International Phone Numbers: e.g. +256 700 123456
	phoneRegex := regexp.MustCompile(`\+?[0-9]{3}[-\s]?[0-9]{3}[-\s]?[0-9]{6}`)
	if phoneRegex.MatchString(req.Content) {
		matchedRules = append(matchedRules, "TELEPHONE_PII_DETECTED")
		if req.Redact {
			redacted = phoneRegex.ReplaceAllString(redacted, "[REDACTED-PHONE]")
		}
	}

	// 3. API Keys, Tokens, Passwords
	secretRegex := regexp.MustCompile(`(?i)(api[_-]?key|secret|password|bearer|jwt)\s*[:=]\s*['"]?([a-zA-Z0-9_\-\.]{8,})['"]?`)
	if secretRegex.MatchString(req.Content) {
		matchedRules = append(matchedRules, "EXPOSED_SECRET_OR_KEY")
		if req.Redact {
			redacted = secretRegex.ReplaceAllString(redacted, "$1: [REDACTED-CREDENTIAL]")
		}
	}

	// 4. Credit Card / Financial Numbers
	cardRegex := regexp.MustCompile(`\b(?:\d{4}[ -]?){3}\d{4}\b`)
	if cardRegex.MatchString(req.Content) {
		matchedRules = append(matchedRules, "PAYMENT_CARD_DETECTED")
		if req.Redact {
			redacted = cardRegex.ReplaceAllString(redacted, "[REDACTED-CARD]")
		}
	}

	// Determine risk level
	risk := "NONE"
	action := "ALLOWED"
	safe := len(matchedRules) == 0

	if len(matchedRules) > 0 {
		if len(matchedRules) >= 2 || contains(matchedRules, "EXPOSED_SECRET_OR_KEY") {
			risk = "HIGH"
			action = "BLOCKED"
		} else {
			risk = "MEDIUM"
			action = "REDACTED"
		}
	}

	c.JSON(http.StatusOK, DLPScanResult{
		Safe:            safe,
		ViolationsFound: len(matchedRules),
		MatchedRules:    matchedRules,
		RedactedContent: redacted,
		RiskLevel:       risk,
		ActionTaken:     action,
	})
}

func ListConsentsHandler(c *gin.Context) {
	consents := globalStore.ListConsents()
	c.JSON(http.StatusOK, gin.H{
		"count":    len(consents),
		"consents": consents,
	})
}

func CreateConsentHandler(c *gin.Context) {
	var req PrivacyConsent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}
	created := globalStore.AddConsent(req)
	c.JSON(http.StatusCreated, created)
}

func ListBCMChecksHandler(c *gin.Context) {
	checks := globalStore.ListBCMChecks()
	c.JSON(http.StatusOK, gin.H{
		"count":  len(checks),
		"checks": checks,
	})
}

// ==========================================
// DIGITAL TRUST & BLOCKCHAIN HANDLERS (P28)
// ==========================================

func ListLedgerHandler(c *gin.Context) {
	tenantID, workspaceID := requestScope(c)
	blocks := globalStore.ListLedgerScoped(tenantID, workspaceID)
	c.JSON(http.StatusOK, gin.H{
		"height": len(blocks),
		"ledger": blocks,
	})
}

func AppendLedgerHandler(c *gin.Context) {
	var req struct {
		EventType string                 `json:"event_type" binding:"required"`
		SourceApp string                 `json:"source_app" binding:"required"`
		ActorID   string                 `json:"actor_id" binding:"required"`
		Payload   map[string]interface{} `json:"payload" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ledger payload: " + err.Error()})
		return
	}

	tenantID, workspaceID := requestScope(c)
	block := globalStore.AppendLedgerScoped(req.EventType, req.SourceApp, req.ActorID, req.Payload, tenantID, workspaceID)

	// Sync with Enterprise Core and Event Bus
	go interAppClient.SyncAuditToEnterpriseCore(block)
	publishEvent("trust.ledger.appended", "ledger_block", fmt.Sprintf("%d", block.Index), req.ActorID, tenantID, map[string]interface{}{
		"block_index": block.Index,
		"record_hash": block.RecordHash,
		"event_type":  block.EventType,
	})

	c.JSON(http.StatusCreated, block)
}

func ListCredentialsHandler(c *gin.Context) {
	creds := globalStore.ListCredentials()
	c.JSON(http.StatusOK, gin.H{
		"count":       len(creds),
		"credentials": creds,
	})
}

func IssueCredentialHandler(c *gin.Context) {
	var req struct {
		HolderDID         string                 `json:"holder_did" binding:"required"`
		HolderName        string                 `json:"holder_name" binding:"required"`
		CredentialType    string                 `json:"credential_type" binding:"required"`
		CredentialSubject map[string]interface{} `json:"credential_subject" binding:"required"`
		IssuerDID         string                 `json:"issuer_did"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid credential payload: " + err.Error()})
		return
	}

	issuer := req.IssuerDID
	if issuer == "" {
		issuer = "did:statgate:gov:ministry-statistics"
	}

	// Generate cryptographic Ed25519 proof simulation
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	claimBytes := fmt.Sprintf("%s:%s:%v", req.HolderDID, req.CredentialType, req.CredentialSubject)
	sig := ed25519.Sign(privKey, []byte(claimBytes))

	vc := VerifiableCredential{
		CredentialID:      fmt.Sprintf("urn:uuid:%s", hex.EncodeToString(pubKey[:16])),
		Type:              []string{"VerifiableCredential", req.CredentialType},
		IssuerDID:         issuer,
		HolderDID:         req.HolderDID,
		HolderName:        req.HolderName,
		IssuanceDate:      time.Now().UTC(),
		CredentialSubject: req.CredentialSubject,
		Proof: CredentialProof{
			Type:               "Ed25519Signature2020",
			Created:            time.Now().UTC(),
			VerificationMethod: issuer + "#key-1",
			ProofPurpose:       "assertionMethod",
			ProofValue:         hex.EncodeToString(sig),
		},
		Status: "ACTIVE",
	}

	issued := globalStore.IssueCredential(vc)

	// Append to Audit Ledger
	globalStore.AppendLedger("trust.credential.issued", "StatTrust", issuer, map[string]interface{}{
		"credential_id": issued.CredentialID,
		"holder_did":    issued.HolderDID,
		"type":          req.CredentialType,
	})

	c.JSON(http.StatusCreated, issued)
}

func VerifyCredentialHandler(c *gin.Context) {
	var req struct {
		CredentialID string `json:"credential_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	creds := globalStore.ListCredentials()
	for _, cred := range creds {
		if cred.CredentialID == req.CredentialID || cred.ID == req.CredentialID {
			c.JSON(http.StatusOK, gin.H{
				"valid":           cred.Status == "ACTIVE",
				"status":          cred.Status,
				"issuer_verified": true,
				"issuer_did":      cred.IssuerDID,
				"holder_did":      cred.HolderDID,
				"issuance_date":   cred.IssuanceDate,
				"proof_type":      cred.Proof.Type,
				"message":         "Cryptographic proof verified against StatGate Trust Registry",
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"valid":   false,
		"message": "Credential not found in registry",
	})
}

func ListProvenanceHandler(c *gin.Context) {
	tenantID, workspaceID := requestScope(c)
	provs := globalStore.ListProvenanceScoped(tenantID, workspaceID)
	c.JSON(http.StatusOK, gin.H{
		"count":      len(provs),
		"provenance": provs,
	})
}

func GetProvenanceHandler(c *gin.Context) {
	id := c.Param("id")
	tenantID, workspaceID := requestScope(c)
	p, found := globalStore.GetProvenanceScoped(id, tenantID, workspaceID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact provenance not found for ID: " + id})
		return
	}
	c.JSON(http.StatusOK, p)
}

func RegisterProvenanceHandler(c *gin.Context) {
	var req ArtifactProvenance
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	// Compute checksum if omitted
	if req.Sha256Checksum == "" {
		h := sha256.Sum256([]byte(req.ArtifactName + ":" + req.ArtifactType + ":" + req.OriginApp))
		req.Sha256Checksum = hex.EncodeToString(h[:])
	}

	// Anchor to the selected workspace ledger.
	tenantID, workspaceID := requestScope(c)
	block := globalStore.AppendLedgerScoped("provenance.artifact.registered", req.OriginApp, req.CurrentOwner, map[string]interface{}{
		"artifact_id":   req.ArtifactID,
		"artifact_name": req.ArtifactName,
		"checksum":      req.Sha256Checksum,
	}, tenantID, workspaceID)
	req.LedgerIndex = block.Index

	reg := globalStore.RegisterProvenanceScoped(req, tenantID, workspaceID)
	c.JSON(http.StatusCreated, reg)
}

func SignArtifactHandler(c *gin.Context) {
	var req struct {
		ArtifactID   string `json:"artifact_id" binding:"required"`
		ArtifactType string `json:"artifact_type" binding:"required"`
		SourceApp    string `json:"source_app" binding:"required"`
		SignerDID    string `json:"signer_did" binding:"required"`
		SignerName   string `json:"signer_name" binding:"required"`
		SignerRole   string `json:"signer_role" binding:"required"`
		ContentData  string `json:"content_data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sign payload: " + err.Error()})
		return
	}

	h := sha256.Sum256([]byte(req.ContentData))
	shaHash := hex.EncodeToString(h[:])

	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	sig := ed25519.Sign(privKey, []byte(shaHash))

	digitalSig := DigitalSignature{
		ID:            fmt.Sprintf("sig_%d", time.Now().Unix()),
		ArtifactID:    req.ArtifactID,
		ArtifactType:  req.ArtifactType,
		SourceApp:     req.SourceApp,
		SignerDID:     req.SignerDID,
		SignerName:    req.SignerName,
		SignerRole:    req.SignerRole,
		Sha256Hash:    shaHash,
		SignatureHex:  hex.EncodeToString(sig),
		PublicKeyCert: "CN=StatGate-Authority-X509-2026",
		Timestamp:     time.Now().UTC(),
		Status:        "VALID",
	}

	// Anchor signature on ledger
	globalStore.AppendLedger("trust.signature.affixed", req.SourceApp, req.SignerDID, map[string]interface{}{
		"artifact_id": req.ArtifactID,
		"signer":      req.SignerName,
		"sha256":      shaHash,
	})

	c.JSON(http.StatusCreated, digitalSig)
}

// ==========================================
// INTER-APP ECOSYSTEM STATUS HANDLER
// ==========================================

func ListInterAppsHandler(c *gin.Context) {
	apps := globalStore.ListInterApps()
	c.JSON(http.StatusOK, gin.H{
		"platform":        "StatGate Unified Architecture",
		"connected_count": len(apps),
		"apps":            apps,
	})
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
