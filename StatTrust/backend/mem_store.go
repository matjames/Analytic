package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type MemStore struct {
	mu              sync.RWMutex
	incidents       map[string]SecurityIncident
	siemEvents      []SIEMEvent
	consents        map[string]PrivacyConsent
	bcmChecks       []BCMBackupVerification
	credentials     map[string]VerifiableCredential
	signatures      map[string]DigitalSignature
	ledger          []AuditLedgerBlock
	provenance      map[string]ArtifactProvenance
	trustEntries    map[string]TrustRegistryEntry
	interApps       map[string]InterAppStatus
	encryptionKeys  map[string]EncryptionKey
	pkiCerts        map[string]PKICertificate
	timestamps      map[string]TimestampRecord
}

var globalStore *MemStore

func NewMemStore() *MemStore {
	store := &MemStore{
		incidents:      make(map[string]SecurityIncident),
		siemEvents:     make([]SIEMEvent, 0),
		consents:       make(map[string]PrivacyConsent),
		bcmChecks:      make([]BCMBackupVerification, 0),
		credentials:    make(map[string]VerifiableCredential),
		signatures:     make(map[string]DigitalSignature),
		ledger:         make([]AuditLedgerBlock, 0),
		provenance:     make(map[string]ArtifactProvenance),
		trustEntries:   make(map[string]TrustRegistryEntry),
		interApps:      make(map[string]InterAppStatus),
		encryptionKeys: make(map[string]EncryptionKey),
		pkiCerts:       make(map[string]PKICertificate),
		timestamps:     make(map[string]TimestampRecord),
	}
	store.seedInitialData()
	return store
}

func (s *MemStore) seedInitialData() {
	now := time.Now().UTC()

	// 1. Seed Inter-App Network Configuration
	apps := []InterAppStatus{
		{AppName: "StatGate Analytics", Port: 5000, ServiceURL: "http://statgate-analytics:5000", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 1420, PendingScans: 0},
		{AppName: "PMS (Projects)", Port: 8091, ServiceURL: "http://statgate-pms-api:8080", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 852, PendingScans: 0},
		{AppName: "RMS (Research)", Port: 8092, ServiceURL: "http://statgate-rms-api:8080", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 630, PendingScans: 0},
		{AppName: "StatGovernance", Port: 8093, ServiceURL: "http://statgate-governance-api:8080", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 419, PendingScans: 0},
		{AppName: "StatSpatial (GIS)", Port: 4200, ServiceURL: "http://statspatial:4200", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 310, PendingScans: 0},
		{AppName: "StatChat", Port: 4000, ServiceURL: "http://statchat-backend:4000", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 2540, PendingScans: 0},
		{AppName: "Field Registry", Port: 9090, ServiceURL: "http://statgate-registry-api:9090", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 1105, PendingScans: 0},
		{AppName: "Operations Helpdesk", Port: 5006, ServiceURL: "http://statgate-helpdesk-api:5000", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 490, PendingScans: 0},
		{AppName: "Enterprise Core", Port: 8096, ServiceURL: "http://statgate-enterprise-core:8096", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 3820, PendingScans: 0},
		{AppName: "Integration Hub", Port: 8097, ServiceURL: "http://statgate-integration-hub:8097", Status: "CONNECTED", LastChecked: now, TrustFederated: true, EventsIngested: 215, PendingScans: 0},
	}
	for _, app := range apps {
		s.interApps[app.AppName] = app
	}

	// 2. Seed Trust Registry
	trustAuthorities := []TrustRegistryEntry{
		{
			DID:              "did:statgate:gov:ministry-statistics",
			OrganizationName: "National Statistics Authority (StatGate Root)",
			TrustLevel:       "ROOT_AUTHORITY",
			PublicKeys:       []string{"04a8b7f8c12...ed25519pub"},
			AuthorizedScopes: []string{"*"},
			Status:           "ACTIVE",
			RegisteredAt:     now.Add(-90 * 24 * time.Hour),
		},
		{
			DID:              "did:statgate:inst:makerere-research",
			OrganizationName: "Makerere Health & Demographic Institute",
			TrustLevel:       "RESEARCH_INSTITUTION",
			PublicKeys:       []string{"04c3d2e1f98...ed25519pub"},
			AuthorizedScopes: []string{"rms.publish", "dataset.sign", "ethics.certify"},
			Status:           "ACTIVE",
			RegisteredAt:     now.Add(-60 * 24 * time.Hour),
		},
		{
			DID:              "did:statgate:partner:who-africa",
			OrganizationName: "WHO Regional Evidence Office",
			TrustLevel:       "ACCREDITED_PARTNER",
			PublicKeys:       []string{"04f9e8d7c65...ed25519pub"},
			AuthorizedScopes: []string{"surveillance.audit", "indicator.verify"},
			Status:           "ACTIVE",
			RegisteredAt:     now.Add(-30 * 24 * time.Hour),
		},
	}
	for _, ta := range trustAuthorities {
		s.trustEntries[ta.DID] = ta
	}

	// 3. Seed Genesis & Chained Immutable Audit Ledger
	genesisHash := "0000000000000000000000000000000000000000000000000000000000000000"
	b1Hash := calculateRecordHash(1, genesisHash, "system.genesis", "system", map[string]interface{}{"event": "StatGate Trust Ledger Initialized"})
	b1 := AuditLedgerBlock{
		Index:      1,
		PrevHash:   genesisHash,
		RecordHash: b1Hash,
		MerkleRoot: b1Hash,
		EventType:  "system.genesis",
		SourceApp:  "StatTrust",
		ActorID:    "did:statgate:gov:ministry-statistics",
		Payload:    map[string]interface{}{"msg": "Genesis block established for multi-app audit trail"},
		Timestamp:  now.Add(-48 * time.Hour),
		Nonce:      1024,
	}

	b2Hash := calculateRecordHash(2, b1Hash, "research.publication.notarized", "RMS", map[string]interface{}{"paper_id": "RES-2026-084"})
	b2 := AuditLedgerBlock{
		Index:      2,
		PrevHash:   b1Hash,
		RecordHash: b2Hash,
		MerkleRoot: b2Hash,
		EventType:  "research.publication.notarized",
		SourceApp:  "RMS",
		ActorID:    "did:statgate:inst:makerere-research",
		Payload:    map[string]interface{}{"paper_id": "RES-2026-084", "title": "National Malaria Intervention Cohort 2026", "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		Timestamp:  now.Add(-24 * time.Hour),
		Nonce:      2048,
	}

	b3Hash := calculateRecordHash(3, b2Hash, "project.milestone.signed", "PMS", map[string]interface{}{"project_id": "PRJ-901"})
	b3 := AuditLedgerBlock{
		Index:      3,
		PrevHash:   b2Hash,
		RecordHash: b3Hash,
		MerkleRoot: b3Hash,
		EventType:  "project.milestone.signed",
		SourceApp:  "PMS",
		ActorID:    "admin-uganda-hq",
		Payload:    map[string]interface{}{"project_id": "PRJ-901", "milestone": "Phase 2 Field Enumeration Signed Off"},
		Timestamp:  now.Add(-6 * time.Hour),
		Nonce:      4096,
	}
	s.ledger = append(s.ledger, b1, b2, b3)

	// 4. Seed Artifact Provenance
	p1 := ArtifactProvenance{
		ID:             "prov_res_084",
		ArtifactID:     "RES-2026-084",
		ArtifactName:   "Uganda Longitudinal Malaria Cohort Report 2026",
		ArtifactType:   "RESEARCH_REPORT",
		OriginApp:      "RMS",
		CurrentOwner:   "Dr. Grace Nakato (Lead Epidemiologist)",
		Sha256Checksum: "7d865e959b2466918c9863afca942d0fb89d7c9ac0c99bafc3749504ded97730",
		TSATimestamp:   now.Add(-24 * time.Hour),
		IntegrityState: "VERIFIED",
		LedgerIndex:    2,
		CustodyChain: []CustodyEvent{
			{Timestamp: now.Add(-36 * time.Hour), Actor: "RMS Lead Investigator", Action: "CREATED", App: "RMS", Note: "Draft finalized with statistical appendices", RecordHash: "0a1b2c..."},
			{Timestamp: now.Add(-30 * time.Hour), Actor: "StatTrust DLP Scanner", Action: "SCANNED", App: "StatTrust", Note: "0 PII violations identified; dataset de-identified", RecordHash: "1b2c3d..."},
			{Timestamp: now.Add(-24 * time.Hour), Actor: "Makerere Institutional Review Board", Action: "SIGNED", App: "RMS", Note: "Cryptographic X.509 signature appended", RecordHash: "2c3d4e..."},
			{Timestamp: now.Add(-24 * time.Hour), Actor: "StatTrust RFC3161 TSA", Action: "TIMESTAMPED", App: "StatTrust", Note: "Anchored to block #2 on audit ledger", RecordHash: b2Hash},
		},
	}
	s.provenance[p1.ArtifactID] = p1

	// 5. Seed Verifiable Credentials
	vc1 := VerifiableCredential{
		ID:           "vc_rec_001",
		CredentialID: "urn:uuid:6a91c89f-897d-41fb-9923-3b1a8d052671",
		Type:         []string{"VerifiableCredential", "ResearchAccreditationCredential"},
		IssuerDID:    "did:statgate:gov:ministry-statistics",
		HolderDID:    "did:statgate:user:nakato.grace",
		HolderName:   "Dr. Grace Nakato",
		IssuanceDate: now.Add(-30 * 24 * time.Hour),
		Status:       "ACTIVE",
		CredentialSubject: map[string]interface{}{
			"role":            "Principal Investigator",
			"institution":     "Makerere University",
			"clearance_level": "LEVEL_3_CONFIDENTIAL_HEALTH",
			"valid_domains":   []string{"RMS", "StatSpatial", "StatCollect"},
		},
		Proof: CredentialProof{
			Type:               "Ed25519Signature2020",
			Created:            now.Add(-30 * 24 * time.Hour),
			VerificationMethod: "did:statgate:gov:ministry-statistics#key-1",
			ProofPurpose:       "assertionMethod",
			ProofValue:         "ed25519_sig_9f8e7d6c5b4a3...verified",
		},
	}
	s.credentials[vc1.ID] = vc1

	// 6. Seed Security Incidents (P19)
	inc1 := SecurityIncident{
		ID:             "INC-2026-001",
		Title:          "Unauthorized Geofence Query Burst Detected",
		Severity:       "MEDIUM",
		Status:         "CONTAINED",
		ThreatCategory: "API Abuse & Geospatial Enumeration",
		SourceApp:      "StatSpatial",
		SourceIP:       "197.239.4.18",
		AffectedAsset:  "/api/spatial/layers/boundary/confidential",
		AssignedTo:     "SecOps Team Alpha",
		Details: map[string]interface{}{
			"query_count": 840,
			"time_window": "60s",
			"action_taken": "Dynamic rate limit enforced via Zero Trust Gateway",
		},
		RemediationLog: []string{
			"Threat detected by StatTrust Anomaly Engine",
			"IP 197.239.4.18 throttled for 1 hour",
			"Notified SecOps incident channel on StatChat",
		},
		CreatedAt: now.Add(-3 * time.Hour),
	}

	inc2 := SecurityIncident{
		ID:             "INC-2026-002",
		Title:          "DLP Intercept: Sensitive PII in Field Survey Export",
		Severity:       "HIGH",
		Status:         "RESOLVED",
		ThreatCategory: "Data Loss Prevention (DLP)",
		SourceApp:      "StatCollect",
		SourceIP:       "10.0.4.52",
		AffectedAsset:  "export_survey_batch_409.csv",
		AssignedTo:     "Compliance Officer",
		Details: map[string]interface{}{
			"rule_matched": "UGANDA_NATIONAL_ID_EXPOSURE",
			"fields_found": []string{"nin_number", "phone_primary"},
			"redaction":    "Automated masking applied (NIN-***-48)",
		},
		RemediationLog: []string{
			"File intercepted at StatTrust Gateway",
			"Applied token-level redaction for 340 participant rows",
			"Dispatched compliance audit token to Enterprise Core",
		},
		CreatedAt:  now.Add(-12 * time.Hour),
		ResolvedAt: &now,
	}
	s.incidents[inc1.ID] = inc1
	s.incidents[inc2.ID] = inc2

	// 7. Seed Privacy Consents & BCM checks
	c1 := PrivacyConsent{
		ID:          "pc_sub_9921",
		SubjectID:   "subj_ug_kam_0081",
		SubjectName: "Anonymous Participant 0081",
		DataScope:   "HEALTH",
		Purpose:     "Longitudinal Malaria Biomarker Study (RMS-2026)",
		Status:      "GRANTED",
		GrantedAt:   now.Add(-40 * 24 * time.Hour),
		ExpiresAt:   now.Add(325 * 24 * time.Hour),
	}
	s.consents[c1.ID] = c1

	bcm1 := BCMBackupVerification{
		ID:             "bcm_chk_01",
		ServiceName:    "PostgreSQL Unified Cluster",
		TargetDatabase: "statgate_ml_staging, statgovernance, statchat, pms, rms",
		BackupChecksum: "sha256:4d8a1c90ef2456b...verified",
		Status:         "VERIFIED",
		VerifiedAt:     now.Add(-2 * time.Hour),
		RPOHours:       0.5,
		RTOEstimated:   "12 Minutes",
	}
	s.bcmChecks = append(s.bcmChecks, bcm1)

	// 8. Seed Encryption Keys (P19)
	k1 := EncryptionKey{
		ID:            "key_sec_master_01",
		TenantID:      "tenant-alpha",
		Algorithm:     "AES-256-GCM",
		Purpose:       "DATA_ENCRYPTION",
		Status:        "ACTIVE",
		KeyRef:        "vault://statgate/keys/tenant-alpha/k_master_01",
		RotationCycle: "MONTHLY",
		CreatedAt:     now.Add(-30 * 24 * time.Hour),
		ExpiresAt:     now.Add(60 * 24 * time.Hour),
		RotatedBy:     "secops-lead@statgate.internal",
		LedgerIndex:   1,
	}
	s.encryptionKeys[k1.ID] = k1

	// 9. Seed PKI Certificates (P28)
	cert1 := PKICertificate{
		ID:               "cert_statgate_root_01",
		TenantID:         "tenant-alpha",
		CommonName:       "StatGate Platform Root CA",
		Organization:     "National Evidence Authority",
		OrganizationUnit: "Digital Trust Division",
		Country:          "UG",
		SubjectDID:       "did:statgate:ca:root",
		CertificatePEM:   "-----BEGIN CERTIFICATE-----\nCN=StatGate Platform Root CA,O=National Evidence Authority,C=UG\n-----END CERTIFICATE-----",
		PublicKeyHex:     "3f8a49c0d12e9871abf340982751eacb92019482751029384756102938475610",
		SerialNumber:     "SNGCA-20260816001",
		IssuedBy:         "did:statgate:ca:root",
		NotBefore:        now.Add(-180 * 24 * time.Hour),
		NotAfter:         now.Add(730 * 24 * time.Hour),
		KeyUsage:         []string{"DIGITAL_SIGNATURE", "KEY_ENCIPHERMENT", "CERTIFICATE_SIGNING"},
		Status:           "VALID",
		CreatedAt:        now.Add(-180 * 24 * time.Hour),
		LedgerIndex:      2,
	}
	s.pkiCerts[cert1.ID] = cert1

	// 10. Seed Trust Registry Entries (P28)
	tr1 := TrustRegistryEntry{
		DID:              "did:statgate:gov:ministry-statistics",
		OrganizationName: "National Bureau of Statistics",
		TrustLevel:       "ROOT_AUTHORITY",
		PublicKeys:       []string{"ed25519:7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b"},
		AuthorizedScopes: []string{"OFFICIAL_STATISTICS", "CREDENTIAL_ISSUER", "CENSUS_PROVENANCE"},
		Status:           "ACTIVE",
		RegisteredAt:     now.Add(-365 * 24 * time.Hour),
	}
	tr2 := TrustRegistryEntry{
		DID:              "did:statgate:health:vector-research-council",
		OrganizationName: "Uganda Vector Control Council",
		TrustLevel:       "ACCREDITED_PARTNER",
		PublicKeys:       []string{"ed25519:1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b"},
		AuthorizedScopes: []string{"CLINICAL_TRIAL_VERIFICATION", "ETHICS_CERTIFICATION"},
		Status:           "ACTIVE",
		RegisteredAt:     now.Add(-120 * 24 * time.Hour),
	}
	s.trustEntries[tr1.DID] = tr1
	s.trustEntries[tr2.DID] = tr2
}

func calculateRecordHash(index int64, prevHash, eventType, sourceApp string, payload map[string]interface{}) string {
	raw := fmt.Sprintf("%d:%s:%s:%s:%v", index, prevHash, eventType, sourceApp, payload)
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// Getters and Mutators with thread safety

func (s *MemStore) GetHealthSummary() SecurityHealthSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeInc := 0
	critInc := 0
	for _, inc := range s.incidents {
		if inc.Status == "OPEN" || inc.Status == "INVESTIGATING" || inc.Status == "CONTAINED" {
			activeInc++
			if inc.Severity == "CRITICAL" {
				critInc++
			}
		}
	}

	status := "OPTIMAL"
	if critInc > 0 {
		status = "INCIDENT_ACTIVE"
	} else if activeInc > 0 {
		status = "ELEVATED_RISK"
	}

	return SecurityHealthSummary{
		SystemStatus:        status,
		ActiveIncidents:     activeInc,
		CriticalIncidents:   critInc,
		DLPScansToday:       184,
		BlockedThreatsToday: 12,
		LedgerHeight:        int64(len(s.ledger)),
		ActiveCredentials:   len(s.credentials),
		VerifiedArtifacts:   len(s.provenance),
		GlobalTrustScore:    98.7,
	}
}

func (s *MemStore) ListIncidents() []SecurityIncident {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]SecurityIncident, 0, len(s.incidents))
	for _, v := range s.incidents {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) AddIncident(inc SecurityIncident) SecurityIncident {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inc.ID == "" {
		inc.ID = fmt.Sprintf("INC-%d", time.Now().Unix())
	}
	if inc.CreatedAt.IsZero() {
		inc.CreatedAt = time.Now().UTC()
	}
	s.incidents[inc.ID] = inc
	return inc
}

func (s *MemStore) UpdateIncidentStatus(id, status, note string) (*SecurityIncident, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inc, exists := s.incidents[id]
	if !exists {
		return nil, fmt.Errorf("incident not found: %s", id)
	}
	inc.Status = status
	if note != "" {
		inc.RemediationLog = append(inc.RemediationLog, fmt.Sprintf("[%s] %s", time.Now().UTC().Format(time.RFC3339), note))
	}
	if status == "RESOLVED" || status == "CLOSED" {
		now := time.Now().UTC()
		inc.ResolvedAt = &now
	}
	s.incidents[id] = inc
	return &inc, nil
}

func (s *MemStore) ListLedger() []AuditLedgerBlock {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ledger
}

func (s *MemStore) AppendLedger(eventType, sourceApp, actorID string, payload map[string]interface{}) AuditLedgerBlock {
	s.mu.Lock()
	defer s.mu.Unlock()

	prevHash := "0000000000000000000000000000000000000000000000000000000000000000"
	index := int64(len(s.ledger) + 1)
	if len(s.ledger) > 0 {
		prevHash = s.ledger[len(s.ledger)-1].RecordHash
	}

	recHash := calculateRecordHash(index, prevHash, eventType, sourceApp, payload)
	block := AuditLedgerBlock{
		Index:      index,
		PrevHash:   prevHash,
		RecordHash: recHash,
		MerkleRoot: recHash,
		EventType:  eventType,
		SourceApp:  sourceApp,
		ActorID:    actorID,
		Payload:    payload,
		Timestamp:  time.Now().UTC(),
		Nonce:      time.Now().UnixNano() % 10000,
	}

	s.ledger = append(s.ledger, block)
	return block
}

func (s *MemStore) ListCredentials() []VerifiableCredential {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]VerifiableCredential, 0, len(s.credentials))
	for _, v := range s.credentials {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) IssueCredential(vc VerifiableCredential) VerifiableCredential {
	s.mu.Lock()
	defer s.mu.Unlock()
	if vc.ID == "" {
		vc.ID = fmt.Sprintf("vc_%d", time.Now().Unix())
	}
	if vc.IssuanceDate.IsZero() {
		vc.IssuanceDate = time.Now().UTC()
	}
	vc.Status = "ACTIVE"
	s.credentials[vc.ID] = vc
	return vc
}

func (s *MemStore) ListProvenance() []ArtifactProvenance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]ArtifactProvenance, 0, len(s.provenance))
	for _, v := range s.provenance {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) GetProvenance(artifactID string) (*ArtifactProvenance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.provenance[artifactID]
	return &p, ok
}

func (s *MemStore) RegisterProvenance(p ArtifactProvenance) ArtifactProvenance {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = fmt.Sprintf("prov_%d", time.Now().Unix())
	}
	if p.TSATimestamp.IsZero() {
		p.TSATimestamp = time.Now().UTC()
	}
	p.IntegrityState = "VERIFIED"
	s.provenance[p.ArtifactID] = p
	return p
}

func (s *MemStore) ListInterApps() []InterAppStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]InterAppStatus, 0, len(s.interApps))
	for _, v := range s.interApps {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) ListConsents() []PrivacyConsent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]PrivacyConsent, 0, len(s.consents))
	for _, v := range s.consents {
		res = append(res, v)
	}
	return res
}

func (s *MemStore) AddConsent(c PrivacyConsent) PrivacyConsent {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = fmt.Sprintf("pc_%d", time.Now().Unix())
	}
	if c.GrantedAt.IsZero() {
		c.GrantedAt = time.Now().UTC()
	}
	s.consents[c.ID] = c
	return c
}

func (s *MemStore) ListBCMChecks() []BCMBackupVerification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bcmChecks
}

// ==========================================
// ENCRYPTION KEY ROTATION
// ==========================================

func (s *MemStore) ListEncryptionKeys(tenantID string) []EncryptionKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]EncryptionKey, 0)
	for _, k := range s.encryptionKeys {
		if tenantID == "" || k.TenantID == tenantID {
			res = append(res, k)
		}
	}
	return res
}

func (s *MemStore) GetEncryptionKey(id string) (EncryptionKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.encryptionKeys[id]
	return k, ok
}

func (s *MemStore) AddEncryptionKey(k EncryptionKey) EncryptionKey {
	s.mu.Lock()
	defer s.mu.Unlock()
	if k.ID == "" {
		k.ID = fmt.Sprintf("key_%d", time.Now().UnixNano())
	}
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now().UTC()
	}
	s.encryptionKeys[k.ID] = k
	return k
}

// RotateEncryptionKey marks the old key RETIRING, creates a successor, and links them.
func (s *MemStore) RotateEncryptionKey(oldID, rotatedBy string) (EncryptionKey, EncryptionKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, ok := s.encryptionKeys[oldID]
	if !ok {
		return EncryptionKey{}, EncryptionKey{}, fmt.Errorf("key %s not found", oldID)
	}
	now := time.Now().UTC()
	old.Status = "RETIRING"
	old.RotatedAt = &now
	old.RotatedBy = rotatedBy
	s.encryptionKeys[oldID] = old

	newKey := EncryptionKey{
		ID:            fmt.Sprintf("key_%d", now.UnixNano()),
		TenantID:      old.TenantID,
		Algorithm:     old.Algorithm,
		Purpose:       old.Purpose,
		Status:        "ACTIVE",
		KeyRef:        fmt.Sprintf("vault://statgate/keys/%s/%s", old.TenantID, fmt.Sprintf("key_%d", now.UnixNano())),
		RotationCycle: old.RotationCycle,
		CreatedAt:     now,
		ExpiresAt:     now.Add(90 * 24 * time.Hour),
		RotatedBy:     rotatedBy,
		PreviousKeyID: old.ID,
	}
	s.encryptionKeys[newKey.ID] = newKey
	return old, newKey, nil
}

// ==========================================
// PKI CERTIFICATE AUTHORITY
// ==========================================

func (s *MemStore) ListCertificates(tenantID string) []PKICertificate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]PKICertificate, 0)
	for _, c := range s.pkiCerts {
		if tenantID == "" || c.TenantID == tenantID {
			res = append(res, c)
		}
	}
	return res
}

func (s *MemStore) AddCertificate(c PKICertificate) PKICertificate {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.ID == "" {
		c.ID = fmt.Sprintf("cert_%d", time.Now().UnixNano())
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	s.pkiCerts[c.ID] = c
	return c
}

func (s *MemStore) RevokeCertificate(id, reason string) (PKICertificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.pkiCerts[id]
	if !ok {
		return PKICertificate{}, fmt.Errorf("certificate %s not found", id)
	}
	c.Status = "REVOKED"
	c.RevocationReason = reason
	s.pkiCerts[id] = c
	return c, nil
}

// ==========================================
// TIMESTAMP AUTHORITY
// ==========================================

func (s *MemStore) AddTimestamp(ts TimestampRecord) TimestampRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ts.ID == "" {
		ts.ID = fmt.Sprintf("tsa_%d", time.Now().UnixNano())
	}
	if ts.IssuedAt.IsZero() {
		ts.IssuedAt = time.Now().UTC()
	}
	s.timestamps[ts.ID] = ts
	return ts
}

func (s *MemStore) GetTimestamp(id string) (TimestampRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ts, ok := s.timestamps[id]
	return ts, ok
}

// ==========================================
// TRUST REGISTRY
// ==========================================

func (s *MemStore) ListTrustRegistry() []TrustRegistryEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]TrustRegistryEntry, 0, len(s.trustEntries))
	for _, entry := range s.trustEntries {
		res = append(res, entry)
	}
	return res
}

func (s *MemStore) GetTrustEntry(did string) (TrustRegistryEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.trustEntries[did]
	return entry, ok
}

func (s *MemStore) UpsertTrustEntry(entry TrustRegistryEntry) TrustRegistryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trustEntries[entry.DID] = entry
	return entry
}


