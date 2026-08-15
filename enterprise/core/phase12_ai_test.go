package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — TEST SUITE: GOVERNED AI REASONING
//
// Covers mandatory AI tests (directive §30): AI cannot directly mutate records,
// AI recommendation requires human review, rejected recommendation cannot
// execute, AI output is auditable and labelled AI_GENERATED, and the data
// classification gate fails closed.
// ═══════════════════════════════════════════════════════════════════════════════

import "testing"

func genRec(t *testing.T, s *Phase12Store, classification string) *IntelligenceRecommendation {
	t.Helper()
	rec, err := s.GenerateRecommendation("t1", "ai-user", "corr_ai_1", AIInput{
		TenantID: "t1", DataClassification: classification, Context: "test",
	})
	if err != nil {
		t.Fatalf("generate recommendation failed: %v", err)
	}
	return rec
}

// A generated recommendation must enter human review (never auto-execute).
func TestPhase12_AI_RequiresHumanReview(t *testing.T) {
	s := newPhase12Store()
	rec := genRec(t, s, "INTERNAL")
	if rec.Status != AIStatusPendingReview {
		t.Fatalf("recommendation must enter PENDING_REVIEW, got %s", rec.Status)
	}
	// It cannot be executed directly from review without authorization.
	if err := s.ExecuteRecommendation("t1", rec.ID, "operator"); err == nil {
		t.Fatal("PENDING_REVIEW recommendation must not execute without authorization (AI cannot skip human review)")
	}
}

// A rejected recommendation can never execute.
func TestPhase12_AI_RejectedCannotExecute(t *testing.T) {
	s := newPhase12Store()
	rec := genRec(t, s, "INTERNAL")
	mustNoErr(t, s.ReviewRecommendation("t1", rec.ID, "human-reviewer", "insufficient evidence", false), "reject")
	if rec.Status != AIStatusRejected {
		t.Fatalf("expected REJECTED, got %s", rec.Status)
	}
	if err := s.ExecuteRecommendation("t1", rec.ID, "operator"); err == nil {
		t.Fatal("rejected recommendation must not execute")
	}
}

// Execution requires prior authorization by a human (AI never self-executes).
func TestPhase12_AI_HumanAuthorizationBeforeExecution(t *testing.T) {
	s := newPhase12Store()
	rec := genRec(t, s, "INTERNAL")
	mustNoErr(t, s.ReviewRecommendation("t1", rec.ID, "institutional_lead", "approved", true), "authorize")
	if rec.Status != AIStatusAuthorized {
		t.Fatalf("expected AUTHORIZED, got %s", rec.Status)
	}
	mustNoErr(t, s.ExecuteRecommendation("t1", rec.ID, "human-operator"), "execute")
	if rec.Status != AIStatusExecuted {
		t.Fatalf("expected EXECUTED, got %s", rec.Status)
	}
}

// AI cannot directly mutate institutional records (advisory only). There is no
// path where a provider writes to objects/edges/conditions directly.
func TestPhase12_AI_CannotMutateRecordsDirectly(t *testing.T) {
	s := newPhase12Store()
	seedObject(s, "t1", "obj_a", "Project", "pms")
	genRec(t, s, "INTERNAL")

	// The recommendation lifecycle only changes recommendation state; the
	// institutional object registry and graph must remain untouched.
	if len(s.ListObjects("t1", "")) != 1 {
		t.Fatal("AI recommendation must not alter the object registry")
	}
	if len(s.ListEdges("t1")) != 0 {
		t.Fatal("AI recommendation must not create graph edges")
	}
	if len(s.ListRiskEvents("t1")) != 0 {
		t.Fatal("AI recommendation must not mutate risk events")
	}
}

// AI output is auditable and always labelled AI_GENERATED.
func TestPhase12_AI_OutputAuditableAndLabeled(t *testing.T) {
	s := newPhase12Store()
	rec := genRec(t, s, "INTERNAL")
	if rec.AILabel != "AI_GENERATED" {
		t.Fatalf("AI output must be labelled AI_GENERATED, got %q", rec.AILabel)
	}
	audit := s.ListAIAudit("t1")
	if len(audit) == 0 {
		t.Fatal("AI request must be written to ai_audit_log")
	}
	entry := audit[0]
	if entry.RecommendationID == "" || entry.CorrelationID == "" {
		t.Fatal("AI audit entry must be traceable (correlation_id / recommendation_id)")
	}
}

// Data classification gate fails closed: RESTRICTED/SENSITIVE blocked.
func TestPhase12_AI_ClassificationGate(t *testing.T) {
	s := newPhase12Store()
	if _, err := s.GenerateRecommendation("t1", "u", "c", AIInput{TenantID: "t1", DataClassification: "RESTRICTED"}); err == nil {
		t.Fatal("RESTRICTED data must be blocked from AI reasoning (fail closed)")
	}
	if _, err := s.GenerateRecommendation("t1", "u", "c", AIInput{TenantID: "t1", DataClassification: "SENSITIVE"}); err == nil {
		t.Fatal("SENSITIVE data must be blocked from AI reasoning (fail closed)")
	}
}

// Unauthorized execution: the API layer (requireRole admin/institutional_lead)
// guards mutation; the store has no anonymous path. Verify an empty/unknown
// actor cannot transition a recommendation on another tenant.
func TestPhase12_AI_TenantScopedLifecycle(t *testing.T) {
	s := newPhase12Store()
	rec := genRec(t, s, "INTERNAL")
	// Attempting to review under a different tenant must fail.
	if err := s.ReviewRecommendation("other_tenant", rec.ID, "attacker", "ok", true); err == nil {
		t.Fatal("cross-tenant AI recommendation review must be denied (tenant isolation)")
	}
}
