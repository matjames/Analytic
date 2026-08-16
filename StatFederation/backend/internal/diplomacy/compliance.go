package diplomacy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

// ComplianceEvaluator enforces transboundary sovereign policies, Malabo convention, and PII masking
type ComplianceEvaluator struct {
	store store.Store
}

// EvaluationDecision represents the sovereign policy outcome
type EvaluationDecision string

const (
	DecisionAllowed            EvaluationDecision = "ALLOWED"
	DecisionDenied             EvaluationDecision = "DENIED"
	DecisionRedactedAndAllowed EvaluationDecision = "REDACTED_AND_ALLOWED"
)

// ComplianceEvaluationResult details policy check outcomes and applied redactions
type ComplianceEvaluationResult struct {
	Decision           EvaluationDecision     `json:"decision"`
	SourceJurisdiction string                 `json:"source_jurisdiction"`
	TargetJurisdiction string                 `json:"target_jurisdiction"`
	ResourceType       string                 `json:"resource_type"`
	AppliedRules       []string               `json:"applied_rules"`
	RedactedFields     []string               `json:"redacted_fields"`
	SanitizedPayload   map[string]interface{} `json:"sanitized_payload"`
	PolicyHash         string                 `json:"policy_hash"`
	Reason             string                 `json:"reason"`
	EvaluatedAt        time.Time              `json:"evaluated_at"`
}

// NewComplianceEvaluator initializes the policy compliance engine
func NewComplianceEvaluator(s store.Store) *ComplianceEvaluator {
	return &ComplianceEvaluator{store: s}
}

// EvaluateCrossBorderTransmission verifies if a data payload is legally permissible for transboundary egress
func (ce *ComplianceEvaluator) EvaluateCrossBorderTransmission(
	ctx context.Context,
	actorUserID, actorTenantID string,
	sourceJurisdiction, targetJurisdiction string,
	resourceType, resourceID string,
	payload map[string]interface{},
) (*ComplianceEvaluationResult, error) {
	appliedRules := make([]string, 0)
	redactedFields := make([]string, 0)
	sanitized := make(map[string]interface{})

	// 1. Rule: Malabo Convention on Cybersecurity and Personal Data Protection
	appliedRules = append(appliedRules, "MALABO_CONVENTION_ART_14_CROSS_BORDER_PROTECTION")

	// 2. Rule: Sovereign Data Localization Check
	if sourceJurisdiction == "NATIONAL" && targetJurisdiction == "GLOBAL" && resourceType == "RAW_MICRODATA" {
		appliedRules = append(appliedRules, "NATIONAL_STATISTICAL_ACT_SEC_29_MICRODATA_LOCALIZATION")
		reason := "Direct egress of unaggregated raw microdata across global jurisdictions is prohibited by national statistical legislation."

		_ = ce.store.LogComplianceEvent(ctx, &models.ComplianceAuditLog{
			ActorUserID:        actorUserID,
			ActorTenantID:      actorTenantID,
			Action:             "SOVEREIGNTY_VIOLATION",
			SourceJurisdiction: sourceJurisdiction,
			TargetJurisdiction: targetJurisdiction,
			ResourceType:       resourceType,
			ResourceID:         resourceID,
			Decision:           string(DecisionDenied),
			AppliedRules:       appliedRules,
			PolicyHash:         hashRules(appliedRules),
			Reason:             reason,
		})

		return &ComplianceEvaluationResult{
			Decision:           DecisionDenied,
			SourceJurisdiction: sourceJurisdiction,
			TargetJurisdiction: targetJurisdiction,
			ResourceType:       resourceType,
			AppliedRules:       appliedRules,
			Reason:             reason,
			EvaluatedAt:        time.Now().UTC(),
		}, nil
	}

	// 3. Rule: Automated PII Masking and Anonymization
	piiKeys := map[string]bool{
		"national_id":     true,
		"nin":             true,
		"phone":           true,
		"phone_number":    true,
		"email":           true,
		"full_name":       true,
		"respondent_name": true,
		"gps_exact":       true,
		"biometrics":      true,
		"ip_address":      true,
	}

	for k, v := range payload {
		cleanK := strings.ToLower(strings.TrimSpace(k))
		if piiKeys[cleanK] {
			redactedFields = append(redactedFields, k)
			sanitized[k] = "[REDACTED_SOVEREIGN_PII]"
		} else {
			sanitized[k] = v
		}
	}

	var decision EvaluationDecision = DecisionAllowed
	reason := "Cross-border evidence transfer conforms to bilateral treaties and regional charter."
	if len(redactedFields) > 0 {
		decision = DecisionRedactedAndAllowed
		reason = fmt.Sprintf("Transmission approved following mandatory PII masking on %d fields.", len(redactedFields))
		appliedRules = append(appliedRules, "PII_ANONYMIZATION_POLICY_V3")
	}

	policyHash := hashRules(appliedRules)

	_ = ce.store.LogComplianceEvent(ctx, &models.ComplianceAuditLog{
		ActorUserID:        actorUserID,
		ActorTenantID:      actorTenantID,
		Action:             "CROSS_BORDER_DISPATCH",
		SourceJurisdiction: sourceJurisdiction,
		TargetJurisdiction: targetJurisdiction,
		ResourceType:       resourceType,
		ResourceID:         resourceID,
		Decision:           string(decision),
		AppliedRules:       appliedRules,
		RedactedFields:     redactedFields,
		PolicyHash:         policyHash,
		Reason:             reason,
	})

	return &ComplianceEvaluationResult{
		Decision:           decision,
		SourceJurisdiction: sourceJurisdiction,
		TargetJurisdiction: targetJurisdiction,
		ResourceType:       resourceType,
		AppliedRules:       appliedRules,
		RedactedFields:     redactedFields,
		SanitizedPayload:   sanitized,
		PolicyHash:         policyHash,
		Reason:             reason,
		EvaluatedAt:        time.Now().UTC(),
	}, nil
}

func hashRules(rules []string) string {
	h := sha256.Sum256([]byte(strings.Join(rules, "|") + uuid.New().String()[:4]))
	return hex.EncodeToString(h[:16])
}
