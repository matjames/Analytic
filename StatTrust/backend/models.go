package main

import "time"

// ==========================================
// P19: SECOPS, CYBERSECURITY & PRIVACY
// ==========================================

// SecurityIncident represents a cybersecurity or operational security event.
type SecurityIncident struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Severity       string                 `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	Status         string                 `json:"status"`   // OPEN, INVESTIGATING, CONTAINED, RESOLVED, CLOSED
	ThreatCategory string                 `json:"threat_category"`
	SourceApp      string                 `json:"source_app"`
	SourceIP       string                 `json:"source_ip"`
	AffectedAsset  string                 `json:"affected_asset"`
	AssignedTo     string                 `json:"assigned_to"`
	Details        map[string]interface{} `json:"details"`
	RemediationLog []string               `json:"remediation_log"`
	CreatedAt      time.Time              `json:"created_at"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
}

// SIEMEvent represents raw or normalized security log telemetry.
type SIEMEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	SourceApp   string                 `json:"source_app"`
	ActorID     string                 `json:"actor_id"`
	SourceIP    string                 `json:"source_ip"`
	Action      string                 `json:"action"`
	Outcome     string                 `json:"outcome"` // SUCCESS, FAILURE, BLOCKED, FLAGGED
	RiskScore   int                    `json:"risk_score"`
	Details     map[string]interface{} `json:"details"`
}

// DLPScanRequest defines the data submitted for real-time DLP inspection.
type DLPScanRequest struct {
	SourceApp   string                 `json:"source_app"`
	DataType    string                 `json:"data_type"` // TEXT, JSON, FILE_METADATA, SQL
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata"`
	Actor       string                 `json:"actor"`
	Redact      bool                   `json:"redact"`
}

// DLPScanResult summarizes findings from the data leakage prevention engine.
type DLPScanResult struct {
	Safe            bool     `json:"safe"`
	ViolationsFound int      `json:"violations_found"`
	MatchedRules    []string `json:"matched_rules"`
	RedactedContent string   `json:"redacted_content,omitempty"`
	RiskLevel       string   `json:"risk_level"` // NONE, LOW, MEDIUM, HIGH, CRITICAL
	ActionTaken     string   `json:"action_taken"` // ALLOWED, REDACTED, BLOCKED, FLAGGED
}

// PrivacyConsent records subject consent for GDPR & DPPA compliance.
type PrivacyConsent struct {
	ID          string    `json:"id"`
	SubjectID   string    `json:"subject_id"`
	SubjectName string    `json:"subject_name"`
	DataScope   string    `json:"data_scope"` // HEALTH, GEOLOCATION, BIOMETRIC, FINANCIAL, RESEARCH
	Purpose     string    `json:"purpose"`
	Status      string    `json:"status"` // GRANTED, REVOKED, EXPIRED
	GrantedAt   time.Time `json:"granted_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// BCMBackupVerification holds automated disaster recovery audit records.
type BCMBackupVerification struct {
	ID             string    `json:"id"`
	ServiceName    string    `json:"service_name"`
	TargetDatabase string    `json:"target_database"`
	BackupChecksum string    `json:"backup_checksum"`
	Status         string    `json:"status"` // VERIFIED, CORRUPTED, IN_PROGRESS
	VerifiedAt     time.Time `json:"verified_at"`
	RPOHours       float64   `json:"rpo_hours"`
	RTOEstimated   string    `json:"rto_estimated"`
}

// SecurityHealthSummary provides overall security metrics.
type SecurityHealthSummary struct {
	SystemStatus        string  `json:"system_status"` // OPTIMAL, ELEVATED_RISK, INCIDENT_ACTIVE
	ActiveIncidents     int     `json:"active_incidents"`
	CriticalIncidents   int     `json:"critical_incidents"`
	DLPScansToday       int     `json:"dlp_scans_today"`
	BlockedThreatsToday int     `json:"blocked_threats_today"`
	LedgerHeight        int64   `json:"ledger_height"`
	ActiveCredentials   int     `json:"active_credentials"`
	VerifiedArtifacts   int     `json:"verified_artifacts"`
	GlobalTrustScore    float64 `json:"global_trust_score"`
}

// ==========================================
// P28: DIGITAL TRUST, BLOCKCHAIN & PROVENANCE
// ==========================================

// VerifiableCredential represents a W3C-compliant cryptographic credential.
type VerifiableCredential struct {
	ID                string                 `json:"id"`
	CredentialID      string                 `json:"credential_id"`
	Type              []string               `json:"type"`
	IssuerDID         string                 `json:"issuer_did"`
	HolderDID         string                 `json:"holder_did"`
	HolderName        string                 `json:"holder_name"`
	IssuanceDate      time.Time              `json:"issuance_date"`
	ExpirationDate    *time.Time             `json:"expiration_date,omitempty"`
	CredentialSubject map[string]interface{} `json:"credential_subject"`
	Proof             CredentialProof        `json:"proof"`
	Status            string                 `json:"status"` // ACTIVE, SUSPENDED, REVOKED
}

// CredentialProof holds cryptographic proof details.
type CredentialProof struct {
	Type               string    `json:"type"` // Ed25519Signature2020, RsaSignature2018
	Created            time.Time `json:"created"`
	VerificationMethod string    `json:"verification_method"`
	ProofPurpose       string    `json:"proof_purpose"`
	ProofValue         string    `json:"proof_value"` // Hex / Base64 signature
}

// DigitalSignature represents a cryptographic document/record signature.
type DigitalSignature struct {
	ID            string    `json:"id"`
	ArtifactID    string    `json:"artifact_id"`
	ArtifactType  string    `json:"artifact_type"` // DATASET, REPORT, ETHICS_APPROVAL, MILESTONE, SPATIAL_LAYER
	SourceApp     string    `json:"source_app"`
	SignerDID     string    `json:"signer_did"`
	SignerName    string    `json:"signer_name"`
	SignerRole    string    `json:"signer_role"`
	Sha256Hash    string    `json:"sha256_hash"`
	SignatureHex  string    `json:"signature_hex"`
	PublicKeyCert string    `json:"public_key_cert"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"` // VALID, INVALID, REVOKED
}

// AuditLedgerBlock represents an immutable cryptographically linked ledger block.
type AuditLedgerBlock struct {
	Index        int64                  `json:"index"`
	PrevHash     string                 `json:"prev_hash"`
	RecordHash   string                 `json:"record_hash"`
	MerkleRoot   string                 `json:"merkle_root"`
	EventType    string                 `json:"event_type"`
	SourceApp    string                 `json:"source_app"`
	ActorID      string                 `json:"actor_id"`
	Payload      map[string]interface{} `json:"payload"`
	Timestamp    time.Time              `json:"timestamp"`
	Nonce        int64                  `json:"nonce"`
}

// ArtifactProvenance maintains the strict chain-of-custody for enterprise assets.
type ArtifactProvenance struct {
	ID             string          `json:"id"`
	ArtifactID     string          `json:"artifact_id"`
	ArtifactName   string          `json:"artifact_name"`
	ArtifactType   string          `json:"artifact_type"` // RESEARCH_REPORT, CLINICAL_DATASET, SURVEY_EXTRACT, GIS_BOUNDARY, GRANT_BUDGET
	OriginApp      string          `json:"origin_app"`
	CurrentOwner   string          `json:"current_owner"`
	Sha256Checksum string          `json:"sha256_checksum"`
	TSATimestamp   time.Time       `json:"tsa_timestamp"`
	CustodyChain   []CustodyEvent  `json:"custody_chain"`
	IntegrityState string          `json:"integrity_state"` // VERIFIED, TAMPERED, UNVERIFIED
	LedgerIndex    int64           `json:"ledger_index"`
}

// CustodyEvent tracks a single handoff, mutation or review in the provenance chain.
type CustodyEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"` // CREATED, HASHED, SIGNED, TRANSFERRED, REVIEWED, PUBLISHED
	App        string    `json:"app"`
	Note       string    `json:"note"`
	RecordHash string    `json:"record_hash"`
}

// TrustRegistryEntry lists registered authorities, issuers, and verification endpoints.
type TrustRegistryEntry struct {
	DID               string    `json:"did"`
	OrganizationName  string    `json:"organization_name"`
	TrustLevel        string    `json:"trust_level"` // ROOT_AUTHORITY, ACCREDITED_PARTNER, VERIFIED_MINISTRY, RESEARCH_INSTITUTION
	PublicKeys        []string  `json:"public_keys"`
	AuthorizedScopes  []string  `json:"authorized_scopes"`
	Status            string    `json:"status"` // ACTIVE, SUSPENDED, REVOKED
	RegisteredAt      time.Time `json:"registered_at"`
}

// InterAppStatus represents the connectivity and security posture of peer applications.
type InterAppStatus struct {
	AppName        string    `json:"app_name"`
	Port           int       `json:"port"`
	ServiceURL     string    `json:"service_url"`
	Status         string    `json:"status"` // CONNECTED, UNREACHABLE, WARNING
	LastChecked    time.Time `json:"last_checked"`
	TrustFederated bool      `json:"trust_federated"`
	EventsIngested int       `json:"events_ingested"`
	PendingScans   int       `json:"pending_scans"`
}

// ==========================================
// P19: ENCRYPTION KEY MANAGEMENT (Key Rotation)
// ==========================================

// EncryptionKeyStatus enumerates lifecycle states.
// ACTIVE → ROTATING → RETIRED → COMPROMISED
type EncryptionKey struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Algorithm     string    `json:"algorithm"`  // AES-256-GCM, ChaCha20-Poly1305, RSA-4096, Ed25519
	Purpose       string    `json:"purpose"`    // DATA_ENCRYPTION, SIGNING, HMAC, TLS, KEY_WRAP
	Status        string    `json:"status"`     // ACTIVE, ROTATING, RETIRED, COMPROMISED
	KeyRef        string    `json:"key_ref"`    // opaque vault reference — never the raw key
	RotationCycle string    `json:"rotation_cycle"` // DAILY, WEEKLY, MONTHLY, QUARTERLY
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	RotatedAt     *time.Time `json:"rotated_at,omitempty"`
	RotatedBy     string    `json:"rotated_by"`
	PreviousKeyID string    `json:"previous_key_id,omitempty"`
	LedgerIndex   int64     `json:"ledger_index"`
}

// ==========================================
// P28: PKI INFRASTRUCTURE
// ==========================================

// PKICertificate represents a certificate issued by the StatGate internal CA.
type PKICertificate struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	CommonName      string    `json:"common_name"`
	Organization    string    `json:"organization"`
	OrganizationUnit string   `json:"organizational_unit"`
	Country         string    `json:"country"`
	SubjectDID      string    `json:"subject_did"`
	CertificatePEM  string    `json:"certificate_pem"`   // X.509 PEM block
	PublicKeyHex    string    `json:"public_key_hex"`
	SerialNumber    string    `json:"serial_number"`
	IssuedBy        string    `json:"issued_by"`         // CA DID
	NotBefore       time.Time `json:"not_before"`
	NotAfter        time.Time `json:"not_after"`
	KeyUsage        []string  `json:"key_usage"`  // DIGITAL_SIGNATURE, KEY_ENCIPHERMENT, DATA_ENCIPHERMENT
	Status          string    `json:"status"`     // VALID, REVOKED, EXPIRED
	RevocationReason string   `json:"revocation_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	LedgerIndex     int64     `json:"ledger_index"`
}

// CSRRequest is the inbound Certificate Signing Request payload.
type CSRRequest struct {
	CommonName       string   `json:"common_name" binding:"required"`
	Organization     string   `json:"organization" binding:"required"`
	OrganizationUnit string   `json:"organizational_unit"`
	Country          string   `json:"country"`
	SubjectDID       string   `json:"subject_did" binding:"required"`
	KeyUsage         []string `json:"key_usage"`
	ValidityDays     int      `json:"validity_days"` // default 365
}

// ==========================================
// P28: TIMESTAMP AUTHORITY (RFC 3161)
// ==========================================

// TimestampRecord is a cryptographically verifiable timestamp token.
type TimestampRecord struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	ArtifactID      string    `json:"artifact_id"`
	ArtifactType    string    `json:"artifact_type"`
	Sha256Hash      string    `json:"sha256_hash"`  // hash of the artifact at stamp time
	IssuedAt        time.Time `json:"issued_at"`
	TSASignatureHex string    `json:"tsa_signature_hex"` // TSA Ed25519 token
	TSAPublicKey    string    `json:"tsa_public_key"`
	PolicyOID       string    `json:"policy_oid"`   // e.g. 1.2.3.4.1.1
	Status          string    `json:"status"`       // VALID, REVOKED
	LedgerIndex     int64     `json:"ledger_index"`
}
