package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

type MemStore struct {
	mu                 sync.RWMutex
	Policies           []Policy
	PolicyVersions     []PolicyVersion
	SOPs               []SOP
	Regulations        []Regulation
	Obligations        []ComplianceObligation
	Assessments        []ComplianceAssessment
	Risks              []Risk
	RiskTreatments     []RiskTreatment
	Controls           []Control
	ControlTests       []ControlTest
	Audits             []Audit
	Findings           []AuditFinding
	CorrectiveActions  []CorrectiveAction
	Committees         []Committee
	Meetings           []Meeting
	Decisions          []GovernanceDecision
	Evidence           []EvidenceRecord
	DataGovernance     []DataGovernanceRecord
	PrivacyAssessments []PrivacyAssessment
	Delegations        []Delegation
}

var memStore = initMemStore()

func initMemStore() *MemStore {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	nextMonth := now.AddDate(0, 1, 0)
	nextYear := now.AddDate(1, 0, 0)

	s := &MemStore{}

	// 1. Initial Policies
	s.Policies = []Policy{
		{
			ID:                 "pol-001",
			PolicyNumber:       "POL-DATA-2026-01",
			Title:              "National Statistical Data Governance Policy",
			Summary:            "Defines data classification, quality standards, stewardship, access controls, and retention for all statistical assets.",
			Content:            "# National Statistical Data Governance Policy\n\n## 1. Purpose\nTo establish institutional control, data integrity, and ethical stewardship over all statistical datasets.\n\n## 2. Scope\nApplies to all analytical hubs, surveys, registries, and field collections within StatGate.\n\n## 3. Data Classification\n- Public Open Data\n- Internal Operational\n- Confidential Survey Data\n- Restricted Identifiable Microdata",
			Category:           "Data Governance",
			Scope:              "Organization-Wide",
			Classification:     "Internal",
			Version:            "1.0",
			Status:             "Active",
			Owner:              "Dr. Sarah Nabatanzi",
			OwnerID:            "usr-002",
			Department:         "Data Management & Standards",
			EffectiveDate:      "2026-01-01",
			ReviewDate:         nextYear.Format("2006-01-02"),
			TenantID:           "tenant-alpha",
			RelatedRegulations: []string{"reg-001"},
			RelatedRisks:       []string{"risk-001"},
			RelatedControls:    []string{"ctl-001"},
			ApprovedBy:         "Management Board",
			ApprovedAt:         yesterday.Format("2006-01-02 15:04:05"),
			PublishedAt:        yesterday.Format("2006-01-02 15:04:05"),
			CreatedTime:        yesterday,
			UpdatedTime:        yesterday,
		},
		{
			ID:                 "pol-002",
			PolicyNumber:       "POL-SEC-2026-02",
			Title:              "Information Security & Access Control Policy",
			Summary:            "Mandates RBAC, MFA, cryptographic storage, and API key management across all internal and field systems.",
			Content:            "# Information Security Policy\n\n## 1. Mandatory Safeguards\nAll microservices must enforce JWT authorization and TLS encryption.",
			Category:           "Security",
			Scope:              "Organization-Wide",
			Classification:     "Confidential",
			Version:            "1.2",
			Status:             "Approval",
			Owner:              "Alex Tumusiime",
			OwnerID:            "usr-003",
			Department:         "ICT & Infrastructure",
			EffectiveDate:      "2026-02-01",
			ReviewDate:         nextYear.Format("2006-01-02"),
			TenantID:           "tenant-alpha",
			ApprovedBy:         "Executive Committee",
			CreatedTime:        yesterday,
			UpdatedTime:        now,
		},
		{
			ID:                 "pol-003",
			PolicyNumber:       "POL-ETH-2026-03",
			Title:              "Research & Survey Ethics Governance Policy",
			Summary:            "Establishes ethical approval protocols, informed consent rules, and data minimization for all field statistical research.",
			Content:            "# Research Ethics Policy\n\nInformed consent is required prior to survey enumeration.",
			Category:           "Ethics & Research",
			Scope:              "Organization-Wide",
			Classification:     "Public",
			Version:            "2.0",
			Status:             "Active",
			Owner:              "Prof. Emmanuel Kato",
			Department:         "Research Directorate",
			EffectiveDate:      "2025-06-01",
			ReviewDate:         nextMonth.Format("2006-01-02"),
			TenantID:           "tenant-alpha",
			CreatedTime:        yesterday,
			UpdatedTime:        yesterday,
		},
	}

	// 2. Initial SOPs
	s.SOPs = []SOP{
		{
			ID:               "sop-001",
			SOPNumber:        "SOP-DATA-01",
			Title:            "Microdata Anonymisation and De-identification Procedure",
			Process:          "Data Release",
			Purpose:          "Ensure statistical microdata protects respondent identity prior to research dissemination.",
			Scope:            "National Census and Survey Microdata",
			Responsibilities: "Data Steward performs k-anonymity validation; Governance Officer approves disclosure.",
			ProcedureSteps: []SOPStep{
				{StepNumber: 1, Title: "Identifier Scrubbing", Description: "Strip direct identifiers (Names, Phone numbers, GPS offsets)", Role: "Data Engineer", Evidence: "Scrubbed dataset checksum"},
				{StepNumber: 2, Title: "K-Anonymity & L-Diversity Check", Description: "Run 3-sigma privacy risk test across quasi-identifiers", Role: "Privacy Analyst", Evidence: "Validation Test Report"},
				{StepNumber: 3, Title: "Governance Approval", Description: "Review de-identification certificate and grant signed token", Role: "Governance Officer", Evidence: "Signed Decision Record"},
			},
			RequiredEvidence: []string{"Privacy Assessment Report", "Disclosure Review Certificate"},
			RelatedPolicyID:  "pol-001",
			ReviewFrequency:  "Semi-Annual",
			Version:          "1.0",
			Status:           "Active",
			Owner:            "Dr. Sarah Nabatanzi",
			Department:       "Data Management & Standards",
			TenantID:         "tenant-alpha",
			CreatedTime:      yesterday,
			UpdatedTime:      yesterday,
		},
	}

	// 3. Regulations & Compliance Obligations
	s.Regulations = []Regulation{
		{
			ID:                  "reg-001",
			Code:                "DPA-2019",
			Name:                "Data Protection and Privacy Act",
			RegulatoryAuthority: "National Data Protection Office",
			Jurisdiction:        "National",
			Category:            "Data Protection",
			Description:         "Mandates lawful data processing, subject consent, storage limitation, and cross-border safeguard mechanisms.",
			EffectiveDate:       "2019-03-01",
			TenantID:            "tenant-alpha",
			CreatedTime:         yesterday,
			UpdatedTime:         yesterday,
		},
		{
			ID:                  "reg-002",
			Code:                "STAT-ACT-1998",
			Name:                "National Statistics Act",
			RegulatoryAuthority: "National Bureau of Statistics",
			Jurisdiction:        "National",
			Category:            "Statistical Governance",
			Description:         "Mandates confidentiality of respondent information and professional independence of official statistics.",
			EffectiveDate:       "1998-10-01",
			TenantID:            "tenant-alpha",
			CreatedTime:         yesterday,
			UpdatedTime:         yesterday,
		},
	}

	s.Obligations = []ComplianceObligation{
		{
			ID:                  "obl-001",
			ObligationCode:      "OBL-DPA-S7",
			RegulationID:        "reg-001",
			Requirement:         "Annual Data Protection Impact Assessment (DPIA) for high-risk data processing systems.",
			ApplicableScope:     "All Statistical Datasets & Survey Registries",
			ResponsibleDept:     "Information Security & Privacy",
			ResponsiblePerson:   "David Okello",
			Frequency:           "Annual",
			ComplianceStatus:    "Compliant",
			EvidenceRequirement: "Certified DPIA Assessment Report signed by DPO",
			ViolationRisk:       "High",
			Deadline:            nextYear.Format("2006-01-02"),
			TenantID:            "tenant-alpha",
			CreatedTime:         yesterday,
			UpdatedTime:         yesterday,
		},
		{
			ID:                  "obl-002",
			ObligationCode:      "OBL-STAT-S19",
			RegulationID:        "reg-002",
			Requirement:         "Mandatory statistical confidentiality oath executed by all field enumerators prior to data capture.",
			ApplicableScope:     "Field Operations & Registry Staff",
			ResponsibleDept:     "Field Operations",
			ResponsiblePerson:   "James Mukasa",
			Frequency:           "Continuous",
			ComplianceStatus:    "Compliant",
			EvidenceRequirement: "Signed Confidentiality Registers in Evidence Vault",
			ViolationRisk:       "Critical",
			Deadline:            nextMonth.Format("2006-01-02"),
			TenantID:            "tenant-alpha",
			CreatedTime:         yesterday,
			UpdatedTime:         yesterday,
		},
	}

	// 4. Institutional Risks
	s.Risks = []Risk{
		{
			ID:                "risk-001",
			RiskCode:          "RSK-OPS-2026-04",
			Title:             "Field Data Collection Synchronisation Latency",
			Description:       "Network disruptions in rural districts could delay real-time survey aggregation into the central analytical repository.",
			Category:          "Operational",
			Owner:             "James Mukasa",
			Department:        "Field Operations",
			Probability:       3,
			Impact:            3,
			InherentRiskScore: 9,
			InherentRiskLevel: "Medium",
			ResidualRiskScore: 4,
			ResidualRiskLevel: "Low",
			TreatmentStrategy: "Mitigate",
			MitigationActions: "Implement SQLite offline-first sync cache in StatCollect with automated checksum reconciliation.",
			Status:            "Active",
			ReviewDate:        nextMonth.Format("2006-01-02"),
			TenantID:          "tenant-alpha",
			CreatedTime:       yesterday,
			UpdatedTime:       now,
		},
		{
			ID:                "risk-002",
			RiskCode:          "RSK-SEC-2026-01",
			Title:             "Unauthorized Microdata Exfiltration Vulnerability",
			Description:       "Potential leakage of identifiable household microdata through insecure third-party analytics integrations.",
			Category:          "Security",
			Owner:             "Alex Tumusiime",
			Department:        "ICT & Infrastructure",
			Probability:       2,
			Impact:            5,
			InherentRiskScore: 10,
			InherentRiskLevel: "High",
			ResidualRiskScore: 4,
			ResidualRiskLevel: "Low",
			TreatmentStrategy: "Mitigate",
			MitigationActions: "Enforce mandatory token-scoped ABAC export limits and SHA256 audit logging.",
			Status:            "Active",
			ReviewDate:        nextMonth.Format("2006-01-02"),
			TenantID:          "tenant-alpha",
			CreatedTime:       yesterday,
			UpdatedTime:       now,
		},
		{
			ID:                "risk-003",
			RiskCode:          "RSK-COMP-2026-02",
			Title:             "Survey Consent Documentation Gaps",
			Description:       "Missing signed digital consent for longitudinal health survey participants.",
			Category:          "Compliance",
			Owner:             "David Okello",
			Department:        "Information Security & Privacy",
			Probability:       4,
			Impact:            4,
			InherentRiskScore: 16,
			InherentRiskLevel: "Critical",
			ResidualRiskScore: 8,
			ResidualRiskLevel: "Medium",
			TreatmentStrategy: "Mitigate",
			MitigationActions: "Integrate biometric/digital signature capture step into ODK collect master workflow.",
			Status:            "Active",
			ReviewDate:        nextMonth.Format("2006-01-02"),
			TenantID:          "tenant-alpha",
			CreatedTime:       yesterday,
			UpdatedTime:       now,
		},
	}

	// 5. Controls
	s.Controls = []Control{
		{
			ID:                  "ctl-001",
			ControlCode:         "CTL-SEC-001",
			Name:                "Automated JWT & RBAC Token Authorization",
			Description:         "Validates cryptographic signatures and role claims on every incoming REST/RPC request across StatGate services.",
			Objective:           "Prevent unauthorized access and enforce tenant isolation.",
			Owner:               "Alex Tumusiime",
			Department:          "ICT & Infrastructure",
			ControlType:         "Preventive",
			Frequency:           "Continuous",
			TestMethod:          "Automated Probe",
			EvidenceRequirement: "Gateway HTTP 401/403 security telemetry logs",
			Effectiveness:       "Effective",
			RelatedRiskID:       "risk-002",
			RelatedPolicyID:     "pol-002",
			LastTested:          yesterday.Format("2006-01-02 15:04:05"),
			NextTestDate:        nextMonth.Format("2006-01-02"),
			TenantID:            "tenant-alpha",
			CreatedTime:         yesterday,
			UpdatedTime:         yesterday,
		},
		{
			ID:                  "ctl-002",
			ControlCode:         "CTL-DATA-002",
			Name:                "Automated 3-Sigma Anomaly & Quality Gate",
			Description:         "Scans statistical data payloads on ingestion to block records with extreme variance or corrupt schemas.",
			Objective:           "Guarantee dataset integrity and prevent dirty data propagation.",
			Owner:               "Dr. Sarah Nabatanzi",
			Department:          "Data Management & Standards",
			ControlType:         "Detective",
			Frequency:           "Daily",
			TestMethod:          "Automated Lakehouse Assertion",
			EvidenceRequirement: "Ingestion Validation Reports",
			Effectiveness:       "Effective",
			RelatedRiskID:       "risk-001",
			RelatedPolicyID:     "pol-001",
			LastTested:          now.Format("2006-01-02 15:04:05"),
			NextTestDate:        nextMonth.Format("2006-01-02"),
			TenantID:            "tenant-alpha",
			CreatedTime:         yesterday,
			UpdatedTime:         now,
		},
	}

	// 6. Audits & Findings
	s.Audits = []Audit{
		{
			ID:          "aud-001",
			AuditCode:   "AUD-2026-Q1-SEC",
			Title:       "Q1 Institutional Information Security & Privacy Audit",
			AuditType:   "Internal Audit",
			Scope:       "StatGate Backend Lakehouse, Redis Event Bus, and Evidence Vault Storage",
			Objectives:  "Verify compliance with Data Protection Act 2019 and ISO 27001 control effectiveness.",
			LeadAuditor: "David Okello",
			Auditors:    []string{"David Okello", "Grace Akello"},
			StartDate:   yesterday.Format("2006-01-02"),
			EndDate:     nextMonth.Format("2006-01-02"),
			Status:      "In Progress",
			TenantID:    "tenant-alpha",
			CreatedTime: yesterday,
			UpdatedTime: now,
		},
	}

	s.Findings = []AuditFinding{
		{
			ID:             "fnd-001",
			FindingCode:    "FND-2026-001",
			AuditID:        "aud-001",
			Title:          "Evidence File Checksum Hash Missing on Legacy Uploads",
			Description:    "14 historical PDF audit reports in evidence vault lack SHA256 integrity verification metadata.",
			Severity:       "Medium",
			Source:         "Internal Audit",
			Owner:          "Alex Tumusiime",
			Department:     "ICT & Infrastructure",
			Recommendation: "Execute batch retroactive SHA256 calculation script across historical evidence storage.",
			DueDate:        nextMonth.Format("2006-01-02"),
			Status:         "In Remediation",
			TenantID:       "tenant-alpha",
			CreatedTime:    yesterday,
			UpdatedTime:    now,
		},
	}

	// 7. Committees & Meetings
	s.Committees = []Committee{
		{
			ID:               "com-001",
			Name:             "Institutional Data Governance Board",
			Code:             "IDGB",
			Mandate:          "Oversees statistical data standards, compliance obligations, control effectiveness, and privacy impact determinations.",
			Chairperson:      "Prof. Emmanuel Kato",
			Secretary:        "Dr. Sarah Nabatanzi",
			MeetingFrequency: "Monthly",
			TenantID:         "tenant-alpha",
			CreatedTime:      yesterday,
			UpdatedTime:      yesterday,
		},
		{
			ID:               "com-002",
			Name:             "Institutional Risk & Compliance Committee",
			Code:             "IRCC",
			Mandate:          "Evaluates institutional risk heatmaps, approves treatment plans, and monitors statutory compliance.",
			Chairperson:      "Dr. Sarah Nabatanzi",
			Secretary:        "David Okello",
			MeetingFrequency: "Quarterly",
			TenantID:         "tenant-alpha",
			CreatedTime:      yesterday,
			UpdatedTime:      yesterday,
		},
	}

	// 8. Decisions
	s.Decisions = []GovernanceDecision{
		{
			ID:             "dec-001",
			DecisionCode:   "DEC-2026-01",
			CommitteeID:    "com-001",
			Title:          "Approval of National Statistical Data Governance Policy v1.0",
			Context:        "Formulation of standard governance protocol across all StatGate ecosystem modules.",
			FinalDecision:  "Approved and ratified without amendment for immediate platform publication.",
			DecisionMaker:  "Prof. Emmanuel Kato (Chairperson)",
			EffectiveDate:  yesterday.Format("2006-01-02"),
			Status:         "Approved",
			TenantID:       "tenant-alpha",
			CreatedTime:    yesterday,
		},
	}

	// 9. Evidence
	hash := sha256.Sum256([]byte("POL-DATA-2026-01 Signed Executive Board Charter"))
	s.Evidence = []EvidenceRecord{
		{
			ID:                "evi-001",
			Title:             "Signed Board Charter - National Data Governance Policy",
			Description:       "Scanned executive approval memorandum with signatures of Board Members.",
			Classification:    "Confidential",
			SourceApplication: "Document",
			FileName:          "Signed_Data_Governance_Policy_Charter_2026.pdf",
			MimeType:          "application/pdf",
			ChecksumSHA256:    hex.EncodeToString(hash[:]),
			UploadedBy:        "Dr. Sarah Nabatanzi",
			RelatedEntityType: "Policy",
			RelatedEntityID:   "pol-001",
			TenantID:          "tenant-alpha",
			CreatedTime:       yesterday,
		},
	}

	// 10. Data Governance Records
	s.DataGovernance = []DataGovernanceRecord{
		{
			ID:                "dg-001",
			DatasetName:       "National Household Socio-Economic Survey 2025/2026",
			DatasetSource:     "StatCollect Field Census",
			DataOwner:         "National Statistical Bureau",
			DataSteward:       "Dr. Sarah Nabatanzi",
			Classification:    "Confidential",
			SensitivityLevel:  "High",
			RetentionPeriod:   "10 Years",
			AccessRules:       "Restricted to verified research teams with approved DPIA protocol.",
			LineageReference:  "StatCollect -> Go Lakehouse -> Microdata Anonymizer -> Analytics Hub",
			QualityThreshold:  95.0,
			SharingAgreements: "Academic Research Data Access Agreement 2026",
			TenantID:          "tenant-alpha",
			CreatedTime:       yesterday,
			UpdatedTime:       yesterday,
		},
	}

	// 11. Privacy Assessments
	s.PrivacyAssessments = []PrivacyAssessment{
		{
			ID:                  "dpia-001",
			ActivityName:        "National Household Geospatial Mapping & Socio-Demographic Census",
			DataController:      "StatGate Institutional Platform",
			LawfulBasis:         "Statutory Legal Obligation (National Statistics Act)",
			PersonalDataTypes:   []string{"GPS Coordinates", "Household Head Names", "Demographics", "Income Brackets"},
			RetentionRules:      "De-identified after 3 years; archived for 10 years in secure vault.",
			CrossBorderTransfer: false,
			DPIAStatus:          "Approved",
			BreachRecordsCount:  0,
			TenantID:            "tenant-alpha",
			CreatedTime:         yesterday,
			UpdatedTime:         yesterday,
		},
	}

	// 12. Delegations
	s.Delegations = []Delegation{
		{
			ID:            "del-001",
			DelegatorID:   "usr-002",
			DelegatorName: "Dr. Sarah Nabatanzi",
			DelegateID:    "usr-005",
			DelegateName:  "David Okello",
			RoleScope:     "Acting Policy Reviewer",
			ScopeDetails:  "Authority to sign off compliance reviews for Data Management SOPs during Q1 field leave.",
			StartDate:     yesterday,
			EndDate:       nextMonth,
			Reason:        "Field Operations Leadership Mission",
			Status:        "Active",
			TenantID:      "tenant-alpha",
			CreatedTime:   yesterday,
			UpdatedTime:   yesterday,
		},
	}

	return s
}

// ─── MemStore Query & Mutation Methods ────────────────────────────

func (s *MemStore) GetDashboard(tenantID string) GovernanceDashboardKPIs {
	s.mu.RLock()
	defer s.mu.RUnlock()

	kpis := GovernanceDashboardKPIs{
		RiskHeatmapMatrix: make(map[string]int),
	}

	now := time.Now()

	// Policies
	for _, p := range s.Policies {
		if p.TenantID != tenantID {
			continue
		}
		if p.Status == "Active" || p.Status == "Published" {
			kpis.PoliciesActive++
		}
		if p.Status == "Under Review" || p.Status == "Under Revision" {
			kpis.PoliciesUnderReview++
		}
		if p.Status == "Approval" {
			kpis.PoliciesAwaitingApproval++
		}
		if p.ReviewDate != "" {
			t, err := time.Parse("2006-01-02", p.ReviewDate)
			if err == nil && t.Before(now) && p.Status == "Active" {
				kpis.PoliciesExpired++
			}
		}
	}

	// Compliance
	var totalObligations int
	for _, o := range s.Obligations {
		if o.TenantID != tenantID {
			continue
		}
		totalObligations++
		if o.ComplianceStatus == "Compliant" {
			kpis.ComplianceCompliant++
		} else if o.ComplianceStatus == "Partially Compliant" {
			kpis.CompliancePartially++
		} else if o.ComplianceStatus == "Non-Compliant" {
			kpis.ComplianceNonCompliant++
		}
		if o.Deadline != "" {
			t, err := time.Parse("2006-01-02", o.Deadline)
			if err == nil && t.Before(now) && o.ComplianceStatus != "Compliant" {
				kpis.ComplianceOverdue++
			}
		}
	}
	if totalObligations > 0 {
		kpis.ComplianceRate = float64(kpis.ComplianceCompliant) / float64(totalObligations) * 100.0
	} else {
		kpis.ComplianceRate = 100.0
	}

	// Risks
	for _, r := range s.Risks {
		if r.TenantID != tenantID || r.Status == "Closed" {
			continue
		}
		if r.InherentRiskLevel == "Critical" {
			kpis.RisksCritical++
		} else if r.InherentRiskLevel == "High" {
			kpis.RisksHigh++
		} else if r.InherentRiskLevel == "Medium" {
			kpis.RisksMedium++
		} else {
			kpis.RisksLow++
		}
		key := fmt.Sprintf("%d_%d", r.Probability, r.Impact)
		kpis.RiskHeatmapMatrix[key]++
	}

	// Controls
	for _, c := range s.Controls {
		if c.TenantID != tenantID {
			continue
		}
		if c.Effectiveness == "Effective" {
			kpis.ControlsEffective++
		} else if c.Effectiveness == "Partially Effective" {
			kpis.ControlsNeedsReview++
		} else if c.Effectiveness == "Ineffective" {
			kpis.ControlsFailed++
		} else {
			kpis.ControlsNotTested++
		}
	}

	// Audits & Findings
	for _, a := range s.Audits {
		if a.TenantID != tenantID {
			continue
		}
		if a.Status == "Planned" || a.Status == "Scheduled" {
			kpis.AuditsPlanned++
		} else if a.Status == "In Progress" {
			kpis.AuditsInProgress++
		}
	}

	for _, f := range s.Findings {
		if f.TenantID != tenantID {
			continue
		}
		if f.Status == "Open" || f.Status == "In Remediation" {
			kpis.FindingsOpen++
			if f.DueDate != "" {
				t, err := time.Parse("2006-01-02", f.DueDate)
				if err == nil && t.Before(now) {
					kpis.FindingsOverdue++
				}
			}
		}
	}

	kpis.RecentDecisionsCount = len(s.Decisions)
	kpis.PendingApprovalsCount = kpis.PoliciesAwaitingApproval

	return kpis
}

func (s *MemStore) GetPolicies(tenantID, status, category string) []Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []Policy
	for _, p := range s.Policies {
		if p.TenantID != tenantID {
			continue
		}
		if status != "" && !strings.EqualFold(p.Status, status) {
			continue
		}
		if category != "" && !strings.EqualFold(p.Category, category) {
			continue
		}
		res = append(res, p)
	}
	return res
}

func (s *MemStore) CreatePolicy(p Policy) Policy {
	s.mu.Lock()
	defer s.mu.Unlock()

	p.CreatedTime = time.Now()
	p.UpdatedTime = time.Now()
	s.Policies = append([]Policy{p}, s.Policies...)
	return p
}

func (s *MemStore) GetPolicyByID(id, tenantID string) (Policy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.Policies {
		if p.ID == id && p.TenantID == tenantID {
			return p, true
		}
	}
	return Policy{}, false
}

func (s *MemStore) UpdatePolicy(p Policy) (Policy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.Policies {
		if existing.ID == p.ID && existing.TenantID == p.TenantID {
			p.CreatedTime = existing.CreatedTime
			p.UpdatedTime = time.Now()
			s.Policies[i] = p
			return p, true
		}
	}
	return Policy{}, false
}

func (s *MemStore) ApprovePolicy(id, approver, tenantID string) (Policy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.Policies {
		if existing.ID == id && existing.TenantID == tenantID {
			s.Policies[i].Status = "Approved"
			s.Policies[i].ApprovedBy = approver
			s.Policies[i].ApprovedAt = time.Now().Format("2006-01-02 15:04:05")
			s.Policies[i].UpdatedTime = time.Now()
			return s.Policies[i], true
		}
	}
	return Policy{}, false
}

func (s *MemStore) PublishPolicy(id, tenantID string) (Policy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.Policies {
		if existing.ID == id && existing.TenantID == tenantID {
			s.Policies[i].Status = "Active"
			s.Policies[i].PublishedAt = time.Now().Format("2006-01-02 15:04:05")
			s.Policies[i].UpdatedTime = time.Now()
			return s.Policies[i], true
		}
	}
	return Policy{}, false
}

func (s *MemStore) GetPolicyVersions(policyID string) []PolicyVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []PolicyVersion
	for _, v := range s.PolicyVersions {
		if v.PolicyID == policyID {
			res = append(res, v)
		}
	}
	return res
}

func (s *MemStore) GetSOPs(tenantID string) []SOP {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []SOP
	for _, sop := range s.SOPs {
		if sop.TenantID == tenantID {
			res = append(res, sop)
		}
	}
	return res
}

func (s *MemStore) CreateSOP(sop SOP) SOP {
	s.mu.Lock()
	defer s.mu.Unlock()

	sop.CreatedTime = time.Now()
	sop.UpdatedTime = time.Now()
	s.SOPs = append([]SOP{sop}, s.SOPs...)
	return sop
}

func (s *MemStore) GetRegulations(tenantID string) []Regulation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []Regulation
	for _, reg := range s.Regulations {
		if reg.TenantID == tenantID {
			res = append(res, reg)
		}
	}
	return res
}

func (s *MemStore) CreateRegulation(reg Regulation) Regulation {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg.CreatedTime = time.Now()
	reg.UpdatedTime = time.Now()
	s.Regulations = append([]Regulation{reg}, s.Regulations...)
	return reg
}

func (s *MemStore) GetObligations(tenantID, regID, status string) []ComplianceObligation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []ComplianceObligation
	for _, o := range s.Obligations {
		if o.TenantID != tenantID {
			continue
		}
		if regID != "" && o.RegulationID != regID {
			continue
		}
		if status != "" && !strings.EqualFold(o.ComplianceStatus, status) {
			continue
		}
		res = append(res, o)
	}
	return res
}

func (s *MemStore) CreateObligation(o ComplianceObligation) ComplianceObligation {
	s.mu.Lock()
	defer s.mu.Unlock()

	o.CreatedTime = time.Now()
	o.UpdatedTime = time.Now()
	s.Obligations = append([]ComplianceObligation{o}, s.Obligations...)
	return o
}

func (s *MemStore) GetAssessments(tenantID string) []ComplianceAssessment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []ComplianceAssessment
	for _, a := range s.Assessments {
		if a.TenantID == tenantID {
			res = append(res, a)
		}
	}
	return res
}

func (s *MemStore) CreateAssessment(a ComplianceAssessment) ComplianceAssessment {
	s.mu.Lock()
	defer s.mu.Unlock()

	a.CreatedTime = time.Now()
	a.UpdatedTime = time.Now()
	s.Assessments = append([]ComplianceAssessment{a}, s.Assessments...)
	return a
}

func (s *MemStore) GetRisks(tenantID, category, level, status string) []Risk {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []Risk
	for _, r := range s.Risks {
		if r.TenantID != tenantID {
			continue
		}
		if category != "" && !strings.EqualFold(r.Category, category) {
			continue
		}
		if level != "" && !strings.EqualFold(r.InherentRiskLevel, level) {
			continue
		}
		if status != "" && !strings.EqualFold(r.Status, status) {
			continue
		}
		res = append(res, r)
	}
	return res
}

func (s *MemStore) CreateRisk(r Risk) Risk {
	s.mu.Lock()
	defer s.mu.Unlock()

	r.CreatedTime = time.Now()
	r.UpdatedTime = time.Now()
	s.Risks = append([]Risk{r}, s.Risks...)
	return r
}

func (s *MemStore) EscalateRisk(id, strategy, notes, tenantID string) (Risk, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.Risks {
		if r.ID == id && r.TenantID == tenantID {
			s.Risks[i].TreatmentStrategy = strategy
			s.Risks[i].MitigationActions += " [Escalated: " + notes + "]"
			s.Risks[i].UpdatedTime = time.Now()
			return s.Risks[i], true
		}
	}
	return Risk{}, false
}

func (s *MemStore) GetControls(tenantID, cType, effect string) []Control {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []Control
	for _, c := range s.Controls {
		if c.TenantID != tenantID {
			continue
		}
		if cType != "" && !strings.EqualFold(c.ControlType, cType) {
			continue
		}
		if effect != "" && !strings.EqualFold(c.Effectiveness, effect) {
			continue
		}
		res = append(res, c)
	}
	return res
}

func (s *MemStore) CreateControl(c Control) Control {
	s.mu.Lock()
	defer s.mu.Unlock()

	c.CreatedTime = time.Now()
	c.UpdatedTime = time.Now()
	s.Controls = append([]Control{c}, s.Controls...)
	return c
}

func (s *MemStore) TestControl(id, tester, result, effect, findings, tenantID string) (Control, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.Controls {
		if c.ID == id && c.TenantID == tenantID {
			s.Controls[i].Effectiveness = effect
			s.Controls[i].LastTested = time.Now().Format("2006-01-02 15:04:05")
			s.Controls[i].UpdatedTime = time.Now()
			return s.Controls[i], true
		}
	}
	return Control{}, false
}

func (s *MemStore) GetAudits(tenantID, status string) []Audit {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []Audit
	for _, a := range s.Audits {
		if a.TenantID != tenantID {
			continue
		}
		if status != "" && !strings.EqualFold(a.Status, status) {
			continue
		}
		res = append(res, a)
	}
	return res
}

func (s *MemStore) CreateAudit(a Audit) Audit {
	s.mu.Lock()
	defer s.mu.Unlock()

	a.CreatedTime = time.Now()
	a.UpdatedTime = time.Now()
	s.Audits = append([]Audit{a}, s.Audits...)
	return a
}

func (s *MemStore) GetFindings(tenantID, severity, status string) []AuditFinding {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []AuditFinding
	for _, f := range s.Findings {
		if f.TenantID != tenantID {
			continue
		}
		if severity != "" && !strings.EqualFold(f.Severity, severity) {
			continue
		}
		if status != "" && !strings.EqualFold(f.Status, status) {
			continue
		}
		res = append(res, f)
	}
	return res
}

func (s *MemStore) CreateFinding(f AuditFinding) AuditFinding {
	s.mu.Lock()
	defer s.mu.Unlock()

	f.CreatedTime = time.Now()
	f.UpdatedTime = time.Now()
	s.Findings = append([]AuditFinding{f}, s.Findings...)
	return f
}

func (s *MemStore) CloseFinding(id, evidence, verifier, tenantID string) (AuditFinding, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, f := range s.Findings {
		if f.ID == id && f.TenantID == tenantID {
			s.Findings[i].Status = "Closed"
			s.Findings[i].ClosureEvidence = evidence
			s.Findings[i].ClosedBy = verifier
			s.Findings[i].ClosedAt = time.Now().Format("2006-01-02")
			s.Findings[i].UpdatedTime = time.Now()
			return s.Findings[i], true
		}
	}
	return AuditFinding{}, false
}

func (s *MemStore) GetCorrectiveActions(tenantID, status string) []CorrectiveAction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []CorrectiveAction
	for _, ca := range s.CorrectiveActions {
		if ca.TenantID != tenantID {
			continue
		}
		if status != "" && !strings.EqualFold(ca.Status, status) {
			continue
		}
		res = append(res, ca)
	}
	return res
}

func (s *MemStore) CreateCorrectiveAction(ca CorrectiveAction) CorrectiveAction {
	s.mu.Lock()
	defer s.mu.Unlock()

	ca.CreatedTime = time.Now()
	s.CorrectiveActions = append([]CorrectiveAction{ca}, s.CorrectiveActions...)
	return ca
}

func (s *MemStore) GetCommittees(tenantID string) []Committee {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []Committee
	for _, c := range s.Committees {
		if c.TenantID == tenantID {
			res = append(res, c)
		}
	}
	return res
}

func (s *MemStore) CreateCommittee(c Committee) Committee {
	s.mu.Lock()
	defer s.mu.Unlock()

	c.CreatedTime = time.Now()
	c.UpdatedTime = time.Now()
	s.Committees = append([]Committee{c}, s.Committees...)
	return c
}

func (s *MemStore) GetMeetings(tenantID, committeeID string) []Meeting {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []Meeting
	for _, m := range s.Meetings {
		if m.TenantID != tenantID {
			continue
		}
		if committeeID != "" && m.CommitteeID != committeeID {
			continue
		}
		res = append(res, m)
	}
	return res
}

func (s *MemStore) CreateMeeting(m Meeting) Meeting {
	s.mu.Lock()
	defer s.mu.Unlock()

	m.CreatedTime = time.Now()
	s.Meetings = append([]Meeting{m}, s.Meetings...)
	return m
}

func (s *MemStore) GetDecisions(tenantID, committeeID string) []GovernanceDecision {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []GovernanceDecision
	for _, d := range s.Decisions {
		if d.TenantID != tenantID {
			continue
		}
		if committeeID != "" && d.CommitteeID != committeeID {
			continue
		}
		res = append(res, d)
	}
	return res
}

func (s *MemStore) CreateDecision(d GovernanceDecision) GovernanceDecision {
	s.mu.Lock()
	defer s.mu.Unlock()

	d.CreatedTime = time.Now()
	s.Decisions = append([]GovernanceDecision{d}, s.Decisions...)
	return d
}

func (s *MemStore) GetEvidence(tenantID, entityType, entityID string) []EvidenceRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []EvidenceRecord
	for _, e := range s.Evidence {
		if e.TenantID != tenantID {
			continue
		}
		if entityType != "" && !strings.EqualFold(e.RelatedEntityType, entityType) {
			continue
		}
		if entityID != "" && e.RelatedEntityID != entityID {
			continue
		}
		res = append(res, e)
	}
	return res
}

func (s *MemStore) CreateEvidence(e EvidenceRecord) EvidenceRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	e.CreatedTime = time.Now()
	s.Evidence = append([]EvidenceRecord{e}, s.Evidence...)
	return e
}

func (s *MemStore) GetDataGovernance(tenantID, class string) []DataGovernanceRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []DataGovernanceRecord
	for _, dg := range s.DataGovernance {
		if dg.TenantID != tenantID {
			continue
		}
		if class != "" && !strings.EqualFold(dg.Classification, class) {
			continue
		}
		res = append(res, dg)
	}
	return res
}

func (s *MemStore) CreateDataGovernance(dg DataGovernanceRecord) DataGovernanceRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	dg.CreatedTime = time.Now()
	dg.UpdatedTime = time.Now()
	s.DataGovernance = append([]DataGovernanceRecord{dg}, s.DataGovernance...)
	return dg
}

func (s *MemStore) GetPrivacyAssessments(tenantID string) []PrivacyAssessment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []PrivacyAssessment
	for _, pa := range s.PrivacyAssessments {
		if pa.TenantID == tenantID {
			res = append(res, pa)
		}
	}
	return res
}

func (s *MemStore) GetDelegations(tenantID, status string) []Delegation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []Delegation
	for _, d := range s.Delegations {
		if d.TenantID != tenantID {
			continue
		}
		if status != "" && !strings.EqualFold(d.Status, status) {
			continue
		}
		res = append(res, d)
	}
	return res
}

func (s *MemStore) CreateDelegation(d Delegation) Delegation {
	s.mu.Lock()
	defer s.mu.Unlock()

	d.CreatedTime = time.Now()
	d.UpdatedTime = time.Now()
	s.Delegations = append([]Delegation{d}, s.Delegations...)
	return d
}

func (s *MemStore) Search(q, tenantID string) []SearchItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []SearchItem
	lower := strings.ToLower(q)

	for _, p := range s.Policies {
		if p.TenantID == tenantID && (strings.Contains(strings.ToLower(p.Title), lower) || strings.Contains(strings.ToLower(p.Summary), lower)) {
			results = append(results, SearchItem{
				Type:        "policy",
				ID:          p.ID,
				Title:       p.Title,
				Description: p.Summary,
				Source:      "StatGovernance",
				URL:         "http://localhost:3012/?tab=policies&id=" + p.ID,
				Score:       90,
			})
		}
	}

	for _, r := range s.Risks {
		if r.TenantID == tenantID && (strings.Contains(strings.ToLower(r.Title), lower) || strings.Contains(strings.ToLower(r.Description), lower)) {
			results = append(results, SearchItem{
				Type:        "risk",
				ID:          r.ID,
				Title:       r.Title,
				Description: r.Description,
				Source:      "StatGovernance",
				URL:         "http://localhost:3012/?tab=risks&id=" + r.ID,
				Score:       85,
			})
		}
	}

	for _, c := range s.Controls {
		if c.TenantID == tenantID && (strings.Contains(strings.ToLower(c.Name), lower) || strings.Contains(strings.ToLower(c.Description), lower)) {
			results = append(results, SearchItem{
				Type:        "control",
				ID:          c.ID,
				Title:       c.Name,
				Description: c.Description,
				Source:      "StatGovernance",
				URL:         "http://localhost:3012/?tab=controls&id=" + c.ID,
				Score:       80,
			})
		}
	}

	return results
}
