package main

// ═══════════════════════════════════════════════════════════════════════════════
// PHASE XII — INSTITUTIONAL INTELLIGENCE & GOVERNED AI — MODELS
//
// Phase XII is the institutional intelligence layer. Every model here is a
// projection / correlation structure over existing authoritative systems.
// Nothing here is a second copy of an application database.
//
// Security rules enforced by these types:
//   - Every record is tenant-scoped (tenant_id is mandatory, never derived
//     from client headers).
//   - Graph edges carry provenance so AI_GENERATED / INFERRED relationships
//     are never indistinguishable from AUTHORITATIVE ones.
//   - KPI data distinguishes ACTUAL | ESTIMATED | STALE | MISSING | INVALID.
//   - AI output is structured and labelled AI_GENERATED.
// ═══════════════════════════════════════════════════════════════════════════════

// ─── Institutional Condition Levels ───────────────────────────────────────────

type ConditionLevel string

const (
	ConditionOptimal   ConditionLevel = "OPTIMAL"
	ConditionNominal   ConditionLevel = "NOMINAL"
	ConditionAttention ConditionLevel = "ATTENTION"
	ConditionElevated  ConditionLevel = "ELEVATED"
	ConditionCritical  ConditionLevel = "CRITICAL"
	ConditionEmergency ConditionLevel = "EMERGENCY"
	ConditionUnknown   ConditionLevel = "UNKNOWN"
)

// Signal domains — the eight intelligence dimensions of the institution.
const (
	DomainStrategicPerformance  = "STRATEGIC_PERFORMANCE"
	DomainOperationalHealth     = "OPERATIONAL_HEALTH"
	DomainDataQuality           = "DATA_QUALITY"
	DomainReportingCompleteness = "REPORTING_COMPLETENESS"
	DomainFinancial             = "FINANCIAL"
	DomainResearch              = "RESEARCH"
	DomainRisk                  = "RISK"
	DomainSecurity              = "SECURITY"
)

var allSignalDomains = []string{
	DomainStrategicPerformance, DomainOperationalHealth, DomainDataQuality,
	DomainReportingCompleteness, DomainFinancial, DomainResearch, DomainRisk,
	DomainSecurity,
}

func validSignalDomain(d string) bool {
	for _, dom := range allSignalDomains {
		if d == dom {
			return true
		}
	}
	return false
}

// ─── Knowledge Graph Relationship Types (directive §7) ───────────────────────

var validRelationshipTypes = map[string]bool{
	"SUPPORTS":        true,
	"CONTRIBUTES_TO":  true,
	"AFFECTS":         true,
	"DEPENDS_ON":      true,
	"PRODUCED_BY":     true,
	"OWNED_BY":        true,
	"ASSIGNED_TO":     true,
	"FUNDED_BY":       true,
	"GOVERNED_BY":     true,
	"ASSOCIATED_WITH": true,
	"OBSERVED_IN":     true,
	"REFERENCED_BY":   true,
	"ALERTS_ON":       true,
}

func validRelationshipType(t string) bool {
	return validRelationshipTypes[t]
}

// ─── Object Types (directive §6) ─────────────────────────────────────────────

var validObjectTypes = map[string]bool{
	"Person": true, "Organisation": true, "Facility": true, "Project": true,
	"Programme": true, "Dataset": true, "Indicator": true, "ResearchStudy": true,
	"Budget": true, "Transaction": true, "Task": true, "Incident": true,
	"Service": true, "Event": true, "Document": true, "Policy": true,
	"Risk": true, "Objective": true,
}

func validObjectType(t string) bool {
	return validObjectTypes[t]
}

// ─── Provenance ───────────────────────────────────────────────────────────────

type ProvenanceType string

const (
	ProvenanceAuthoritative ProvenanceType = "AUTHORITATIVE"
	ProvenanceDerived       ProvenanceType = "DERIVED"
	ProvenanceInferred      ProvenanceType = "INFERRED"
	ProvenanceAIGenerated   ProvenanceType = "AI_GENERATED"
)

func validProvenance(p ProvenanceType) bool {
	switch p {
	case ProvenanceAuthoritative, ProvenanceDerived, ProvenanceInferred, ProvenanceAIGenerated:
		return true
	}
	return false
}