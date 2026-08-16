package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB(dsn string) error {
	var err error
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		DB = nil
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		DB = nil
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	log.Println("Successfully connected to StatGovernance PostgreSQL database")

	if err := migrateDB(); err != nil {
		_ = DB.Close()
		DB = nil
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func migrateDB() error {
	// Ensure schema exists
	_, err := DB.Exec(`CREATE SCHEMA IF NOT EXISTS statgovernance AUTHORIZATION "StatGovernance"`)
	if err != nil {
		// Try without authorization if role does not exist yet in local testing
		_, err = DB.Exec(`CREATE SCHEMA IF NOT EXISTS statgovernance`)
		if err != nil {
			return err
		}
	}

	// 1. Policies
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.policies (
			id                  VARCHAR(36)   PRIMARY KEY,
			policy_number       VARCHAR(50)   UNIQUE NOT NULL,
			title               VARCHAR(255)  NOT NULL,
			summary             TEXT,
			content             TEXT          NOT NULL,
			category            VARCHAR(100)  NOT NULL,
			scope               VARCHAR(100)  DEFAULT 'Organization-Wide',
			classification      VARCHAR(50)   DEFAULT 'Internal',
			version             VARCHAR(20)   NOT NULL DEFAULT '1.0',
			status              VARCHAR(50)   NOT NULL DEFAULT 'Draft',
			owner               VARCHAR(255)  NOT NULL,
			owner_id            VARCHAR(36),
			department          VARCHAR(100)  NOT NULL,
			effective_date      DATE,
			review_date         DATE,
			expiry_date         DATE,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			org_id              VARCHAR(50),
			related_regulations TEXT[],
			related_risks       TEXT[],
			related_controls    TEXT[],
			related_projects    TEXT[],
			related_research    TEXT[],
			related_datasets    TEXT[],
			approved_by         VARCHAR(255),
			approved_at         TIMESTAMP,
			published_at        TIMESTAMP,
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.policy_versions (
			id                  VARCHAR(36)   PRIMARY KEY,
			policy_id           VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE CASCADE,
			version             VARCHAR(20)   NOT NULL,
			title               VARCHAR(255)  NOT NULL,
			content             TEXT          NOT NULL,
			change_summary      TEXT,
			changed_by          VARCHAR(255)  NOT NULL,
			approved_by         VARCHAR(255),
			status              VARCHAR(50)   NOT NULL,
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 2. SOPs
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.sops (
			id                  VARCHAR(36)   PRIMARY KEY,
			sop_number          VARCHAR(50)   UNIQUE NOT NULL,
			title               VARCHAR(255)  NOT NULL,
			process             VARCHAR(255)  NOT NULL,
			purpose             TEXT          NOT NULL,
			scope               TEXT          NOT NULL,
			responsibilities    TEXT,
			procedure_steps     JSONB         DEFAULT '[]'::jsonb,
			required_evidence   TEXT[],
			related_policy_id   VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE SET NULL,
			related_control_id  VARCHAR(36),
			review_frequency    VARCHAR(50)   DEFAULT 'Annual',
			version             VARCHAR(20)   DEFAULT '1.0',
			status              VARCHAR(50)   DEFAULT 'Active',
			owner               VARCHAR(255)  NOT NULL,
			department          VARCHAR(100)  NOT NULL,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			approved_by         VARCHAR(255),
			approved_at         TIMESTAMP,
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 3. Regulations & Compliance
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.regulations (
			id                  VARCHAR(36)   PRIMARY KEY,
			code                VARCHAR(50)   UNIQUE NOT NULL,
			name                VARCHAR(255)  NOT NULL,
			regulatory_authority VARCHAR(255) NOT NULL,
			jurisdiction        VARCHAR(100)  DEFAULT 'National',
			category            VARCHAR(100)  NOT NULL,
			description         TEXT,
			source_url          TEXT,
			effective_date      DATE,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.compliance_obligations (
			id                  VARCHAR(36)   PRIMARY KEY,
			obligation_code     VARCHAR(50)   UNIQUE NOT NULL,
			regulation_id       VARCHAR(36)   REFERENCES statgovernance.regulations(id) ON DELETE CASCADE,
			requirement         TEXT          NOT NULL,
			applicable_scope    VARCHAR(255),
			responsible_dept    VARCHAR(100)  NOT NULL,
			responsible_person  VARCHAR(255)  NOT NULL,
			frequency           VARCHAR(50)   DEFAULT 'Continuous',
			compliance_status   VARCHAR(50)   NOT NULL DEFAULT 'Compliant',
			evidence_requirement TEXT,
			evidence_location   TEXT,
			violation_risk      VARCHAR(50)   DEFAULT 'High',
			corrective_action   TEXT,
			deadline            DATE,
			review_date         DATE,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.compliance_assessments (
			id                  VARCHAR(36)   PRIMARY KEY,
			assessment_code     VARCHAR(50)   UNIQUE NOT NULL,
			title               VARCHAR(255)  NOT NULL,
			scope               TEXT          NOT NULL,
			assessor            VARCHAR(255)  NOT NULL,
			assessor_id         VARCHAR(36),
			status              VARCHAR(50)   DEFAULT 'In Progress',
			score_percentage    FLOAT         DEFAULT 0.0,
			findings_count      INTEGER       DEFAULT 0,
			recommendations     TEXT,
			start_date          DATE,
			completion_date     DATE,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.assessment_items (
			id                  VARCHAR(36)   PRIMARY KEY,
			assessment_id       VARCHAR(36)   REFERENCES statgovernance.compliance_assessments(id) ON DELETE CASCADE,
			obligation_id       VARCHAR(36)   REFERENCES statgovernance.compliance_obligations(id) ON DELETE CASCADE,
			status              VARCHAR(50)   NOT NULL,
			notes               TEXT,
			evidence_ref        VARCHAR(255),
			findings            TEXT,
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 4. Risks & Treatments
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.risks (
			id                  VARCHAR(36)   PRIMARY KEY,
			risk_code           VARCHAR(50)   UNIQUE NOT NULL,
			title               VARCHAR(255)  NOT NULL,
			description         TEXT          NOT NULL,
			category            VARCHAR(100)  NOT NULL,
			owner               VARCHAR(255)  NOT NULL,
			owner_id            VARCHAR(36),
			organization        VARCHAR(255)  DEFAULT 'StatGate Authority',
			department          VARCHAR(100),
			project_id          VARCHAR(36),
			research_id         VARCHAR(36),
			probability         INTEGER       NOT NULL,
			impact              INTEGER       NOT NULL,
			inherent_risk_score INTEGER       DEFAULT 1,
			inherent_risk_level VARCHAR(20)   NOT NULL,
			treatment_strategy  VARCHAR(50)   DEFAULT 'Mitigate',
			mitigation_actions  TEXT,
			residual_probability INTEGER,
			residual_impact     INTEGER,
			residual_risk_score INTEGER,
			residual_risk_level VARCHAR(20),
			status              VARCHAR(50)   NOT NULL DEFAULT 'Active',
			target_date         DATE,
			review_date         DATE,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.risk_treatments (
			id                  VARCHAR(36)   PRIMARY KEY,
			risk_id             VARCHAR(36)   REFERENCES statgovernance.risks(id) ON DELETE CASCADE,
			title               VARCHAR(255)  NOT NULL,
			description         TEXT,
			action_type         VARCHAR(50)   DEFAULT 'Preventive',
			responsible_person  VARCHAR(255)  NOT NULL,
			due_date            DATE,
			status              VARCHAR(50)   DEFAULT 'Pending',
			enterprise_task_id  VARCHAR(50),
			completion_date     DATE,
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 5. Controls & Tests
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.controls (
			id                  VARCHAR(36)   PRIMARY KEY,
			control_code        VARCHAR(50)   UNIQUE NOT NULL,
			name                VARCHAR(255)  NOT NULL,
			description         TEXT          NOT NULL,
			objective           TEXT          NOT NULL,
			owner               VARCHAR(255)  NOT NULL,
			department          VARCHAR(100)  NOT NULL,
			control_type        VARCHAR(50)   NOT NULL,
			frequency           VARCHAR(50)   NOT NULL DEFAULT 'Monthly',
			test_method         VARCHAR(100)  DEFAULT 'Automated Probe',
			evidence_requirement TEXT,
			effectiveness       VARCHAR(50)   NOT NULL DEFAULT 'Effective',
			related_risk_id     VARCHAR(36)   REFERENCES statgovernance.risks(id) ON DELETE SET NULL,
			related_policy_id   VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE SET NULL,
			last_tested         TIMESTAMP,
			next_test_date      DATE,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.control_tests (
			id                  VARCHAR(36)   PRIMARY KEY,
			control_id          VARCHAR(36)   REFERENCES statgovernance.controls(id) ON DELETE CASCADE,
			tester              VARCHAR(255)  NOT NULL,
			test_date           TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			result              VARCHAR(50)   NOT NULL,
			effectiveness       VARCHAR(50)   NOT NULL,
			findings            TEXT,
			evidence_ref        VARCHAR(255),
			remediation_task_id VARCHAR(50),
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 6. Audits & Findings
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.audits (
			id                  VARCHAR(36)   PRIMARY KEY,
			audit_code          VARCHAR(50)   UNIQUE NOT NULL,
			title               VARCHAR(255)  NOT NULL,
			audit_type          VARCHAR(100)  NOT NULL,
			scope               TEXT          NOT NULL,
			objectives          TEXT          NOT NULL,
			lead_auditor        VARCHAR(255)  NOT NULL,
			auditors            TEXT[],
			auditees            TEXT[],
			department          VARCHAR(100)  NOT NULL,
			start_date          DATE,
			end_date            DATE,
			status              VARCHAR(50)   NOT NULL DEFAULT 'Planned',
			findings_count      INTEGER       DEFAULT 0,
			critical_findings   INTEGER       DEFAULT 0,
			summary             TEXT,
			report_file_id      VARCHAR(50),
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.audit_findings (
			id                  VARCHAR(36)   PRIMARY KEY,
			finding_code        VARCHAR(50)   UNIQUE NOT NULL,
			audit_id            VARCHAR(36)   REFERENCES statgovernance.audits(id) ON DELETE CASCADE,
			title               VARCHAR(255)  NOT NULL,
			description         TEXT          NOT NULL,
			severity            VARCHAR(50)   NOT NULL,
			source              VARCHAR(100)  DEFAULT 'Audit',
			owner               VARCHAR(255)  NOT NULL,
			department          VARCHAR(100)  NOT NULL,
			policy_id           VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE SET NULL,
			control_id          VARCHAR(36)   REFERENCES statgovernance.controls(id) ON DELETE SET NULL,
			risk_id             VARCHAR(36)   REFERENCES statgovernance.risks(id) ON DELETE SET NULL,
			recommendation      TEXT          NOT NULL,
			management_response TEXT,
			due_date            DATE          NOT NULL,
			status              VARCHAR(50)   NOT NULL DEFAULT 'Open',
			enterprise_task_id  VARCHAR(50),
			closure_evidence    TEXT,
			closed_by           VARCHAR(255),
			closed_at           TIMESTAMP,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 7. Corrective Actions (CAPA)
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.corrective_actions (
			id                  VARCHAR(36)   PRIMARY KEY,
			action_code         VARCHAR(50)   UNIQUE NOT NULL,
			title               VARCHAR(255)  NOT NULL,
			description         TEXT          NOT NULL,
			source_type         VARCHAR(50)   NOT NULL,
			source_id           VARCHAR(36)   NOT NULL,
			owner               VARCHAR(255)  NOT NULL,
			priority            VARCHAR(50)   NOT NULL DEFAULT 'High',
			due_date            DATE          NOT NULL,
			enterprise_task_id  VARCHAR(50),
			status              VARCHAR(50)   NOT NULL DEFAULT 'Pending',
			completion_date     DATE,
			verification_notes  TEXT,
			verified_by         VARCHAR(255),
			verified_at         TIMESTAMP,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 8. Committees & Meetings
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.governance_committees (
			id                  VARCHAR(36)   PRIMARY KEY,
			name                VARCHAR(255)  NOT NULL,
			code                VARCHAR(50)   UNIQUE NOT NULL,
			mandate             TEXT          NOT NULL,
			chairperson         VARCHAR(255)  NOT NULL,
			secretary           VARCHAR(255)  NOT NULL,
			meeting_frequency   VARCHAR(50)   DEFAULT 'Monthly',
			statchat_channel_id VARCHAR(100),
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.committee_members (
			id                  VARCHAR(36)   PRIMARY KEY,
			committee_id        VARCHAR(36)   REFERENCES statgovernance.governance_committees(id) ON DELETE CASCADE,
			name                VARCHAR(255)  NOT NULL,
			role                VARCHAR(100)  NOT NULL,
			email               VARCHAR(255)  NOT NULL,
			department          VARCHAR(100),
			joined_date         DATE          DEFAULT CURRENT_DATE,
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.governance_meetings (
			id                  VARCHAR(36)   PRIMARY KEY,
			committee_id        VARCHAR(36)   REFERENCES statgovernance.governance_committees(id) ON DELETE CASCADE,
			title               VARCHAR(255)  NOT NULL,
			meeting_date        TIMESTAMP     NOT NULL,
			location            VARCHAR(255)  DEFAULT 'StatGate Boardroom / StatChat Video',
			calendar_event_id   VARCHAR(50),
			status              VARCHAR(50)   DEFAULT 'Scheduled',
			agenda              TEXT,
			minutes             TEXT,
			attendees           TEXT[],
			decisions_count     INTEGER       DEFAULT 0,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 9. Decisions
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.governance_decisions (
			id                  VARCHAR(36)   PRIMARY KEY,
			decision_code       VARCHAR(50)   UNIQUE NOT NULL,
			title               VARCHAR(255)  NOT NULL,
			context             TEXT          NOT NULL,
			options_considered  JSONB         DEFAULT '[]'::jsonb,
			risks_evaluated     TEXT,
			recommendation      TEXT,
			final_decision      TEXT          NOT NULL,
			decision_maker      VARCHAR(255)  NOT NULL,
			committee_id        VARCHAR(36)   REFERENCES statgovernance.governance_committees(id) ON DELETE SET NULL,
			meeting_id          VARCHAR(36)   REFERENCES statgovernance.governance_meetings(id) ON DELETE SET NULL,
			policy_id           VARCHAR(36)   REFERENCES statgovernance.policies(id) ON DELETE SET NULL,
			status              VARCHAR(50)   DEFAULT 'Approved',
			ai_advisory         BOOLEAN       DEFAULT FALSE,
			human_approved      BOOLEAN       DEFAULT TRUE,
			enterprise_decision_id VARCHAR(50),
			effective_date      DATE          DEFAULT CURRENT_DATE,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 10. Evidence Records
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.evidence_records (
			id                  VARCHAR(36)   PRIMARY KEY,
			title               VARCHAR(255)  NOT NULL,
			description         TEXT,
			category            VARCHAR(100)  NOT NULL,
			source_application  VARCHAR(100)  NOT NULL,
			related_entity_type VARCHAR(50)   NOT NULL,
			related_entity_id   VARCHAR(36)   NOT NULL,
			file_id             VARCHAR(50),
			file_name           VARCHAR(255),
			file_url            TEXT,
			mime_type           VARCHAR(100),
			file_size           BIGINT,
			checksum_sha256     VARCHAR(64),
			classification      VARCHAR(50)   DEFAULT 'Internal',
			version             INTEGER       DEFAULT 1,
			uploaded_by         VARCHAR(255)  NOT NULL,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 11. Data Governance & Privacy
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.data_governance_records (
			id                  VARCHAR(36)   PRIMARY KEY,
			dataset_name        VARCHAR(255)  NOT NULL,
			dataset_source      VARCHAR(100)  NOT NULL,
			data_owner          VARCHAR(255)  NOT NULL,
			data_steward        VARCHAR(255)  NOT NULL,
			classification      VARCHAR(50)   NOT NULL DEFAULT 'Confidential',
			sensitivity_level   VARCHAR(50)   NOT NULL DEFAULT 'Medium',
			retention_period    VARCHAR(50)   NOT NULL DEFAULT '7 Years',
			access_rules        TEXT          NOT NULL,
			lineage_reference   TEXT,
			quality_threshold   FLOAT         DEFAULT 90.0,
			sharing_agreements  TEXT,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.privacy_assessments (
			id                  VARCHAR(36)   PRIMARY KEY,
			activity_name       VARCHAR(255)  NOT NULL,
			data_controller     VARCHAR(255)  NOT NULL,
			lawful_basis        VARCHAR(100)  NOT NULL,
			personal_data_types TEXT[],
			retention_rules     TEXT          NOT NULL,
			cross_border_transfer BOOLEAN     DEFAULT FALSE,
			dpia_status         VARCHAR(50)   DEFAULT 'Approved',
			breach_records_count INTEGER      DEFAULT 0,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 12. Delegations
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.delegations (
			id                  VARCHAR(36)   PRIMARY KEY,
			delegator_id        VARCHAR(36)   NOT NULL,
			delegator_name      VARCHAR(255)  NOT NULL,
			delegate_id         VARCHAR(36)   NOT NULL,
			delegate_name       VARCHAR(255)  NOT NULL,
			role_scope          VARCHAR(100)  NOT NULL,
			scope_details       TEXT          NOT NULL,
			start_date          TIMESTAMP     NOT NULL,
			end_date            TIMESTAMP     NOT NULL,
			reason              TEXT          NOT NULL,
			status              VARCHAR(50)   NOT NULL DEFAULT 'Active',
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// 13. Audit Logs
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.audit_logs (
			id                  VARCHAR(36)   PRIMARY KEY,
			entity_type         VARCHAR(50)   NOT NULL,
			entity_id           VARCHAR(36)   NOT NULL,
			action              VARCHAR(100)  NOT NULL,
			actor               VARCHAR(255)  NOT NULL,
			actor_id            VARCHAR(36),
			previous_state      JSONB,
			new_state           JSONB,
			details             TEXT,
			ip_address          VARCHAR(50),
			correlation_id      VARCHAR(100),
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			timestamp           TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// ─── Phase 13 Extensions: Whistleblower, COI, Feature Flags, System Parameters ───

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.whistleblower_reports (
			id                  VARCHAR(36)   PRIMARY KEY,
			ticket_number       VARCHAR(50)   UNIQUE NOT NULL,
			title               VARCHAR(255)  NOT NULL,
			category            VARCHAR(100)  NOT NULL,
			description         TEXT          NOT NULL,
			evidence_files      TEXT[],
			status              VARCHAR(50)   DEFAULT 'Submitted',
			encrypted_notes     TEXT,
			assigned_to         VARCHAR(255),
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.conflict_declarations (
			id                  VARCHAR(36)   PRIMARY KEY,
			user_id             VARCHAR(36)   NOT NULL,
			user_name           VARCHAR(255)  NOT NULL,
			department          VARCHAR(100),
			declaration_type    VARCHAR(100)  NOT NULL,
			entity_name         VARCHAR(255)  NOT NULL,
			nature_of_interest  VARCHAR(255)  NOT NULL,
			description         TEXT,
			mitigation_plan     TEXT,
			status              VARCHAR(50)   DEFAULT 'Declared',
			reviewed_by         VARCHAR(255),
			reviewed_at         TIMESTAMP,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.feature_flags (
			id                  VARCHAR(36)   PRIMARY KEY,
			key                 VARCHAR(100)  UNIQUE NOT NULL,
			name                VARCHAR(255)  NOT NULL,
			description         TEXT,
			enabled             BOOLEAN       DEFAULT FALSE,
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			module              VARCHAR(100),
			rollout_pct         INTEGER       DEFAULT 100,
			created_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS statgovernance.system_parameters (
			id                  VARCHAR(36)   PRIMARY KEY,
			param_key           VARCHAR(100)  UNIQUE NOT NULL,
			param_value         TEXT          NOT NULL,
			description         TEXT,
			data_type           VARCHAR(50)   DEFAULT 'string',
			category            VARCHAR(100)  DEFAULT 'General',
			tenant_id           VARCHAR(50)   DEFAULT 'tenant-alpha',
			updated_time        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}

	// Seed minimal foundational governance data if database is fresh
	seedFoundationalData()

	return nil
}

func seedFoundationalData() {
	var count int
	_ = DB.QueryRow("SELECT COUNT(*) FROM statgovernance.policies").Scan(&count)
	if count > 0 {
		return
	}

	log.Println("Seeding foundational institutional governance baseline...")

	// 1. Initial Policy
	_, _ = DB.Exec(`
		INSERT INTO statgovernance.policies (
			id, policy_number, title, summary, content, category, scope, classification, version, status,
			owner, owner_id, department, effective_date, review_date, tenant_id, approved_by, approved_at, published_at
		) VALUES (
			'pol-001', 'POL-DATA-2026-01', 'National Statistical Data Governance Policy',
			'Defines data classification, quality standards, stewardship, access controls, and retention for all statistical assets.',
			'# National Statistical Data Governance Policy\n\n## 1. Purpose\nTo establish institutional control, data integrity, and ethical stewardship over all statistical datasets.\n\n## 2. Scope\nApplies to all analytical hubs, surveys, registries, and field collections within StatGate.\n\n## 3. Data Classification\n- Public Open Data\n- Internal Operational\n- Confidential Survey Data\n- Restricted Identifiable Microdata',
			'Data Governance', 'Organization-Wide', 'Internal', '1.0', 'Active',
			'Dr. Sarah Nabatanzi', 'usr-002', 'Data Management & Standards', '2026-01-01', '2027-01-01', 'tenant-alpha', 'Management Board', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)`)

	// 2. Regulation & Compliance Obligation
	_, _ = DB.Exec(`
		INSERT INTO statgovernance.regulations (
			id, code, name, regulatory_authority, jurisdiction, category, description, effective_date, tenant_id
		) VALUES (
			'reg-001', 'DPA-2019', 'Data Protection and Privacy Act', 'National Data Protection Office', 'National',
			'Data Protection', 'Mandates lawful data processing, subject consent, storage limitation, and cross-border safeguard mechanisms.', '2019-03-01', 'tenant-alpha'
		)`)

	_, _ = DB.Exec(`
		INSERT INTO statgovernance.compliance_obligations (
			id, obligation_code, regulation_id, requirement, applicable_scope, responsible_dept, responsible_person,
			frequency, compliance_status, evidence_requirement, violation_risk, deadline, tenant_id
		) VALUES (
			'obl-001', 'OBL-DPA-S7', 'reg-001', 'Annual Data Protection Impact Assessment (DPIA) for high-risk data processing systems.',
			'All Statistical Datasets & Survey Registries', 'Information Security & Privacy', 'David Okello',
			'Annual', 'Compliant', 'Certified DPIA Assessment Report signed by DPO', 'High', '2026-12-31', 'tenant-alpha'
		)`)

	// 3. Initial Risk
	_, _ = DB.Exec(`
		INSERT INTO statgovernance.risks (
			id, risk_code, title, description, category, owner, department, probability, impact, inherent_risk_score,
			inherent_risk_level, treatment_strategy, mitigation_actions, status, review_date, tenant_id
		) VALUES (
			'risk-001', 'RSK-OPS-2026-04', 'Field Data Collection Synchronisation Latency',
			'Network disruptions in rural districts could delay real-time survey aggregation into the central analytical repository.',
			'Operational', 'James Mukasa', 'Field Operations', 3, 3, 9,
			'Medium', 'Mitigate', 'Implement SQLite offline-first sync cache in StatCollect with automated checksum reconciliation.', 'Active', '2026-09-30', 'tenant-alpha'
		)`)

	// 4. Initial Control
	_, _ = DB.Exec(`
		INSERT INTO statgovernance.controls (
			id, control_code, name, description, objective, owner, department, control_type, frequency,
			test_method, effectiveness, related_risk_id, related_policy_id, tenant_id
		) VALUES (
			'ctl-001', 'CTL-SEC-001', 'Automated JWT & RBAC Token Authorization',
			'Validates cryptographic signatures and role claims on every incoming REST/RPC request across StatGate services.',
			'Prevent unauthorized access and enforce tenant isolation.',
			'Alex Tumusiime', 'ICT & Infrastructure', 'Preventive', 'Continuous',
			'Automated Probe', 'Effective', 'risk-001', 'pol-001', 'tenant-alpha'
		)`)

	// 5. Governance Committee
	_, _ = DB.Exec(`
		INSERT INTO statgovernance.governance_committees (
			id, name, code, mandate, chairperson, secretary, meeting_frequency, tenant_id
		) VALUES (
			'com-001', 'Institutional Data Governance Board', 'IDGB',
			'Oversees statistical data standards, compliance obligations, control effectiveness, and privacy impact determinations.',
			'Prof. Emmanuel Kato', 'Dr. Sarah Nabatanzi', 'Monthly', 'tenant-alpha'
		)`)
}
