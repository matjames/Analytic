package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"statchat/pkg/model"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Client struct {
	conn         *websocket.Conn
	conversation string
}

var db *sql.DB
var clients = map[*Client]bool{}
var clientsMutex sync.Mutex

func Init(dsn string) error {
	if dsn == "" {
		return errors.New("database DSN must be provided")
	}

	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	ctx := context.Background()
	if err = ensureSchema(ctx); err != nil {
		return err
	}
	if err = ensureCollaborationSchema(ctx); err != nil {
		return err
	}
	if err = seedDefaults(ctx); err != nil {
		return err
	}
	if err = seedCollaborationDefaults(ctx); err != nil {
		return err
	}
	return seedKnowledgeDefaults(ctx)
}

func ensureSchema(ctx context.Context) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  organization_id TEXT NOT NULL,
  roles JSONB NOT NULL,
  avatar_url TEXT,
  presence TEXT
);

CREATE TABLE IF NOT EXISTS channels (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS conversations (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  channel_id TEXT,
  member_ids JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL REFERENCES conversations(id),
  channel_id TEXT,
  sender TEXT NOT NULL,
  text TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ,
  parent_message_id TEXT,
  thread_root_id TEXT,
  status TEXT NOT NULL DEFAULT 'active'
);

ALTER TABLE messages ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS parent_message_id TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS thread_root_id TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

CREATE TABLE IF NOT EXISTS message_reactions (
  id TEXT PRIMARY KEY,
  message_id TEXT NOT NULL REFERENCES messages(id),
  user_id TEXT NOT NULL,
  emoji TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  UNIQUE (message_id, user_id, emoji)
);

CREATE TABLE IF NOT EXISTS message_attachments (
  id TEXT PRIMARY KEY,
  message_id TEXT NOT NULL REFERENCES messages(id),
  file_name TEXT NOT NULL,
  file_type TEXT NOT NULL,
  url TEXT NOT NULL,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS pinned_messages (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL REFERENCES conversations(id),
  message_id TEXT NOT NULL REFERENCES messages(id),
  pinned_by TEXT NOT NULL,
  pinned_at TIMESTAMPTZ NOT NULL
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS presence TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS about TEXT;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS channel_id TEXT;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS member_ids JSONB;
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS category TEXT;

CREATE TABLE IF NOT EXISTS user_settings (
  user_id TEXT PRIMARY KEY REFERENCES users(id),
  theme TEXT NOT NULL DEFAULT 'light',
  accent_color TEXT NOT NULL DEFAULT '#0b5fff',
  font_size TEXT NOT NULL DEFAULT 'medium',
  enter_to_send BOOLEAN NOT NULL DEFAULT true,
  language TEXT NOT NULL DEFAULT 'English',
  last_seen TEXT NOT NULL DEFAULT 'everyone',
  profile_photo TEXT NOT NULL DEFAULT 'contacts',
  read_receipts BOOLEAN NOT NULL DEFAULT true,
  typing_indicator BOOLEAN NOT NULL DEFAULT true,
  voice_notes BOOLEAN NOT NULL DEFAULT true,
  read_by_default BOOLEAN NOT NULL DEFAULT false,
  auto_download TEXT NOT NULL DEFAULT 'never',
  notif_messages BOOLEAN NOT NULL DEFAULT true,
  notif_groups BOOLEAN NOT NULL DEFAULT true,
  notif_mentions BOOLEAN NOT NULL DEFAULT true,
  notif_meetings BOOLEAN NOT NULL DEFAULT true,
  notif_sound BOOLEAN NOT NULL DEFAULT true,
  notif_preview BOOLEAN NOT NULL DEFAULT false,
  download_images TEXT NOT NULL DEFAULT 'wifi',
  download_videos TEXT NOT NULL DEFAULT 'wifi',
  download_documents TEXT NOT NULL DEFAULT 'wifi'
);
`)
	return err
}

func seedDefaults(ctx context.Context) error {
	// Fix legacy member IDs in existing conversations (user-1 -> user-001)
	fixLegacyMemberIDs(ctx)

	// Only seed if the users table is empty — never wipe existing data
	var userCount int
	err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&userCount)
	if err != nil {
		return err
	}
	if userCount > 0 {
		// Ensure group conversations exist even if users were already seeded
		ensureGroupConversations(ctx)
		return nil
	}

	firstNames := []string{"Amina", "Jonas", "Claire", "Rahul", "Maya", "Ethan", "Sofia", "Noah", "Priya", "Lucas", "Hana", "Dmitri", "Leila", "Marco", "Sara", "Aria", "Omar", "Yara", "Felix", "Nina"}
	lastNames := []string{"Patel", "Gonzalez", "Wang", "Kumar", "Smith", "Johnson", "Chen", "Davis", "Nguyen", "Martinez", "Ali", "Bakker", "Santos", "Mills", "Park", "Hoffman", "Sengupta", "Elbaz", "Khan", "Stevens"}
	departments := []string{"Ministry of Data", "Ministry of Finance", "Digital Services", "Research Operations", "Enterprise Architecture", "Analytics Lab", "Security Desk", "Policy Office", "AI Initiative", "Program Delivery"}
	roleSets := [][]string{{"member"}, {"researcher"}, {"lead"}, {"manager"}, {"analyst"}, {"engineer"}, {"coordinator"}, {"advisor"}, {"strategist"}, {"director"}}

	userIDs := make([]string, 0, 100)
	for i := 1; i <= 100; i++ {
		id := fmt.Sprintf("user-%03d", i)
		first := firstNames[(i-1)%len(firstNames)]
		last := lastNames[(i-1)%len(lastNames)]
		name := fmt.Sprintf("%s %s", first, last)
		email := fmt.Sprintf("%s.%s@statchat.local", strings.ToLower(first), strings.ToLower(last))
		organization := departments[(i-1)%len(departments)]
		rolesJSON, _ := json.Marshal(roleSets[(i-1)%len(roleSets)])
		avatarURL := fmt.Sprintf("https://api.dicebear.com/6.x/initials/svg?seed=%s", id)
		presence := "offline"
		if i%3 != 0 {
			presence = "online"
		}

		_, err := db.ExecContext(ctx, `INSERT INTO users (id, name, email, organization_id, roles, avatar_url, presence) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`, id, name, email, organization, rolesJSON, avatarURL, presence)
		if err != nil {
			return err
		}
		userIDs = append(userIDs, id)
	}

	allMembersJSON, _ := json.Marshal(userIDs)
	researchMembersJSON, _ := json.Marshal(userIDs[:60])
	adaMembersJSON, _ := json.Marshal([]string{"user-001", "user-002"})
	guestMembersJSON, _ := json.Marshal([]string{"user-001", "user-003", "user-004"})
	teamMembersJSON, _ := json.Marshal(userIDs[:15])

	channels := []struct {
		id   string
		name string
	}{
		{"general", "General"},
		{"research", "Research"},
		{"development", "Development"},
	}
	for _, channel := range channels {
		_, err := db.ExecContext(ctx, `INSERT INTO channels (id, name) VALUES ($1, $2) ON CONFLICT DO NOTHING`, channel.id, channel.name)
		if err != nil {
			return err
		}
	}

	conversations := []struct {
		id        string
		name      string
		typeValue model.ConversationType
		channelID interface{}
		memberIDs []byte
	}{
		{"general", "#general", model.ConversationTypeChannel, "general", allMembersJSON},
		{"research", "#research", model.ConversationTypeChannel, "research", researchMembersJSON},
		{"dm-user-2", "Ada Project", model.ConversationTypeDirect, nil, adaMembersJSON},
		{"project-101", "StatGate Ops", model.ConversationTypeChannel, "development", guestMembersJSON},
		{"group-statistics", "Statistics Team", model.ConversationTypeGroup, nil, teamMembersJSON},
		{"group-development", "AI Team", model.ConversationTypeGroup, nil, teamMembersJSON},
	}

	for _, conv := range conversations {
		_, err := db.ExecContext(ctx, `INSERT INTO conversations (id, name, type, channel_id, member_ids) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, conv.id, conv.name, conv.typeValue, conv.channelID, conv.memberIDs)
		if err != nil {
			return err
		}
	}

	messageSets := []struct {
		conversationID string
		channelID      string
		entries        []struct {
			senderIndex int
			text        string
			attach      bool
		}
	}{
		{
			conversationID: "general",
			channelID:      "general",
			entries: []struct {
				senderIndex int
				text        string
				attach      bool
			}{
				{senderIndex: 1, text: "Welcome to StatChat — your enterprise collaboration hub.", attach: false},
				{senderIndex: 2, text: "We have new datasets ready for review in #research.", attach: true},
				{senderIndex: 5, text: "Please complete the monthly KPI report by EOD.", attach: false},
				{senderIndex: 10, text: "Yesterday's deploy was successful 🎉", attach: false},
			},
		},
		{
			conversationID: "research",
			channelID:      "research",
			entries: []struct {
				senderIndex int
				text        string
				attach      bool
			}{
				{senderIndex: 3, text: "The new dataset shows promising accuracy improvements.", attach: true},
				{senderIndex: 8, text: "Let's schedule a review session for the model drift report.", attach: false},
				{senderIndex: 12, text: "I uploaded the latest CSV to the workspace.", attach: true},
				{senderIndex: 1, text: "Can someone validate the anomaly detection logic?", attach: false},
			},
		},
		{
			conversationID: "dm-user-2",
			channelID:      "",
			entries: []struct {
				senderIndex int
				text        string
				attach      bool
			}{
				{senderIndex: 2, text: "Hey, are you available for a quick sync on Ada Project?", attach: false},
				{senderIndex: 1, text: "Yes, let's connect in 10 minutes.", attach: false},
				{senderIndex: 2, text: "I have a group draft for the executive summary.", attach: true},
			},
		},
	}

	for _, set := range messageSets {
		timestamp := time.Now().UTC().Add(-4 * time.Hour)
		for _, entry := range set.entries {
			messageID := uuid.NewString()
			sender := fmt.Sprintf("%s %s", firstNames[(entry.senderIndex-1)%len(firstNames)], lastNames[(entry.senderIndex-1)%len(lastNames)])
			_, err := db.ExecContext(ctx, `INSERT INTO messages (id, conversation_id, channel_id, sender, text, created_at, status) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`, messageID, set.conversationID, sql.NullString{String: set.channelID, Valid: set.channelID != ""}, sender, entry.text, timestamp, "active")
			if err != nil {
				return err
			}
			if entry.attach {
				_, err := db.ExecContext(ctx, `INSERT INTO message_attachments (id, message_id, file_name, file_type, url, metadata, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`, uuid.NewString(), messageID, "research-dataset.csv", "text/csv", "https://statchat.local/files/research-dataset.csv", json.RawMessage(`{"size":"2.1MB","project":"StatGate"}`), timestamp)
				if err != nil {
					return err
				}
			}
			timestamp = timestamp.Add(18 * time.Minute)
		}
	}

	return nil
}

var groupTemplates = []model.GroupCategory{
	{
		ID: "platform", Name: "Platform-Wide", Icon: "🛡️", Description: "Default groups in every StatGate installation",
		Groups: []model.Group{
			{ID: "announcements", Name: "📢 Announcements", Category: "platform"},
			{ID: "general-discussion", Name: "💬 General Discussion", Category: "platform"},
			{ID: "help-support", Name: "🆘 Help & Support", Category: "platform"},
			{ID: "knowledge-base", Name: "📚 Knowledge Base", Category: "platform"},
			{ID: "platform-updates", Name: "📢 Platform Updates", Category: "platform"},
			{ID: "system-alerts", Name: "🚨 System Alerts", Category: "platform"},
			{ID: "it-support", Name: "🛠 IT Support", Category: "platform"},
			{ID: "suggestions", Name: "💡 Suggestions & Feedback", Category: "platform"},
			{ID: "social-community", Name: "🎉 Social & Community", Category: "platform"},
		},
	},
	{
		ID: "leadership", Name: "Executive Leadership", Icon: "👑", Description: "Executive and leadership coordination groups",
		Groups: []model.Group{
			{ID: "executive-board", Name: "Executive Board", Category: "leadership"},
			{ID: "board-directors", Name: "Board of Directors", Category: "leadership"},
			{ID: "ceo", Name: "Chief Executive Officer", Category: "leadership"},
			{ID: "executive-management", Name: "Executive Management", Category: "leadership"},
			{ID: "senior-management", Name: "Senior Management", Category: "leadership"},
			{ID: "strategy-planning", Name: "Strategy & Planning", Category: "leadership"},
			{ID: "policy-committee", Name: "Policy Committee", Category: "leadership"},
			{ID: "decision-support", Name: "Decision Support Committee", Category: "leadership"},
		},
	},
	{
		ID: "administration", Name: "Administration", Icon: "🏛️", Description: "System and organizational administration",
		Groups: []model.Group{
			{ID: "sysadmins", Name: "System Administrators", Category: "administration"},
			{ID: "platform-admins", Name: "Platform Administrators", Category: "administration"},
			{ID: "org-admins", Name: "Organization Administrators", Category: "administration"},
			{ID: "dept-admins", Name: "Department Administrators", Category: "administration"},
			{ID: "db-admins", Name: "Database Administrators", Category: "administration"},
			{ID: "security-admins", Name: "Security Administrators", Category: "administration"},
			{ID: "infrastructure", Name: "Infrastructure Team", Category: "administration"},
			{ID: "devops", Name: "DevOps Team", Category: "administration"},
		},
	},
	{
		ID: "data-statistics", Name: "Data & Statistics", Icon: "📊", Description: "Statistical and data professionals",
		Groups: []model.Group{
			{ID: "statisticians", Name: "Statisticians", Category: "data-statistics"},
			{ID: "senior-statisticians", Name: "Senior Statisticians", Category: "data-statistics"},
			{ID: "biostatisticians", Name: "Biostatisticians", Category: "data-statistics"},
			{ID: "data-analysts", Name: "Data Analysts", Category: "data-statistics"},
			{ID: "data-scientists", Name: "Data Scientists", Category: "data-statistics"},
			{ID: "data-engineers", Name: "Data Engineers", Category: "data-statistics"},
			{ID: "data-architects", Name: "Data Architects", Category: "data-statistics"},
			{ID: "metadata-team", Name: "Metadata Team", Category: "data-statistics"},
			{ID: "data-quality", Name: "Data Quality Team", Category: "data-statistics"},
			{ID: "bi-team", Name: "Business Intelligence Team", Category: "data-statistics"},
			{ID: "survey-team", Name: "Survey Team", Category: "data-statistics"},
		},
	},
	{
		ID: "ai", Name: "Artificial Intelligence", Icon: "🤖", Description: "AI and machine learning teams",
		Groups: []model.Group{
			{ID: "ai-engineers", Name: "AI Engineers", Category: "ai"},
			{ID: "ml-engineers", Name: "Machine Learning Engineers", Category: "ai"},
			{ID: "deep-learning", Name: "Deep Learning Team", Category: "ai"},
			{ID: "ai-researchers", Name: "AI Researchers", Category: "ai"},
			{ID: "prompt-engineering", Name: "Prompt Engineering Team", Category: "ai"},
			{ID: "model-development", Name: "Model Development", Category: "ai"},
			{ID: "model-evaluation", Name: "Model Evaluation", Category: "ai"},
			{ID: "ai-governance", Name: "AI Governance", Category: "ai"},
		},
	},
	{
		ID: "research", Name: "Research", Icon: "🔬", Description: "Research collaboration groups",
		Groups: []model.Group{
			{ID: "researchers", Name: "Researchers", Category: "research"},
			{ID: "principal-investigators", Name: "Principal Investigators", Category: "research"},
			{ID: "research-coordinators", Name: "Research Coordinators", Category: "research"},
			{ID: "research-assistants", Name: "Research Assistants", Category: "research"},
			{ID: "research-fellows", Name: "Research Fellows", Category: "research"},
			{ID: "research-supervisors", Name: "Research Supervisors", Category: "research"},
			{ID: "ethics-committee", Name: "Research Ethics Committee", Category: "research"},
			{ID: "publications-committee", Name: "Publications Committee", Category: "research"},
			{ID: "research-data-managers", Name: "Research Data Managers", Category: "research"},
		},
	},
	{
		ID: "public-health", Name: "Public Health", Icon: "🏥", Description: "Public health and surveillance",
		Groups: []model.Group{
			{ID: "epidemiologists", Name: "Epidemiologists", Category: "public-health"},
			{ID: "disease-surveillance", Name: "Disease Surveillance", Category: "public-health"},
			{ID: "health-officers", Name: "Public Health Officers", Category: "public-health"},
			{ID: "health-info-officers", Name: "Health Information Officers", Category: "public-health"},
			{ID: "health-informatics", Name: "Health Informatics", Category: "public-health"},
			{ID: "health-planning", Name: "Health Planning", Category: "public-health"},
			{ID: "outbreak-response", Name: "Outbreak Response", Category: "public-health"},
			{ID: "emergency-response", Name: "Emergency Response Team", Category: "public-health"},
			{ID: "community-health", Name: "Community Health", Category: "public-health"},
		},
	},
	{
		ID: "laboratory", Name: "Laboratory", Icon: "🧪", Description: "Laboratory and sample management",
		Groups: []model.Group{
			{ID: "lab-scientists", Name: "Laboratory Scientists", Category: "laboratory"},
			{ID: "lab-managers", Name: "Laboratory Managers", Category: "laboratory"},
			{ID: "sample-management", Name: "Sample Management", Category: "laboratory"},
			{ID: "qa-laboratory", Name: "Quality Assurance Laboratory", Category: "laboratory"},
			{ID: "molecular-biology", Name: "Molecular Biology", Category: "laboratory"},
			{ID: "microbiology", Name: "Microbiology", Category: "laboratory"},
			{ID: "virology", Name: "Virology", Category: "laboratory"},
		},
	},
	{
		ID: "gis", Name: "GIS & Mapping", Icon: "🗺️", Description: "Geospatial analysis and mapping",
		Groups: []model.Group{
			{ID: "gis-specialists", Name: "GIS Specialists", Category: "gis"},
			{ID: "remote-sensing", Name: "Remote Sensing", Category: "gis"},
			{ID: "spatial-analytics", Name: "Spatial Analytics", Category: "gis"},
			{ID: "mapping-team", Name: "Mapping Team", Category: "gis"},
			{ID: "environmental-monitoring", Name: "Environmental Monitoring", Category: "gis"},
		},
	},
	{
		ID: "monitoring-evaluation", Name: "Monitoring & Evaluation", Icon: "📈", Description: "M&E and performance monitoring",
		Groups: []model.Group{
			{ID: "me-officers", Name: "M&E Officers", Category: "monitoring-evaluation"},
			{ID: "performance-monitoring", Name: "Performance Monitoring", Category: "monitoring-evaluation"},
			{ID: "evaluation-team", Name: "Evaluation Team", Category: "monitoring-evaluation"},
			{ID: "indicator-management", Name: "Indicator Management", Category: "monitoring-evaluation"},
			{ID: "quality-improvement", Name: "Quality Improvement", Category: "monitoring-evaluation"},
		},
	},
	{
		ID: "software", Name: "Software Development", Icon: "💻", Description: "Engineering and product teams",
		Groups: []model.Group{
			{ID: "frontend-devs", Name: "Frontend Developers", Category: "software"},
			{ID: "backend-devs", Name: "Backend Developers", Category: "software"},
			{ID: "fullstack-devs", Name: "Full Stack Developers", Category: "software"},
			{ID: "mobile-devs", Name: "Mobile Developers", Category: "software"},
			{ID: "ui-ux", Name: "UI/UX Designers", Category: "software"},
			{ID: "qa-engineers", Name: "QA Engineers", Category: "software"},
			{ID: "software-testers", Name: "Software Testers", Category: "software"},
			{ID: "product-owners", Name: "Product Owners", Category: "software"},
			{ID: "business-analysts", Name: "Business Analysts", Category: "software"},
			{ID: "technical-writers", Name: "Technical Writers", Category: "software"},
		},
	},
	{
		ID: "finance", Name: "Finance", Icon: "💰", Description: "Financial management and accounting",
		Groups: []model.Group{
			{ID: "finance-dept", Name: "Finance Department", Category: "finance"},
			{ID: "accountants", Name: "Accountants", Category: "finance"},
			{ID: "payroll", Name: "Payroll", Category: "finance"},
			{ID: "budget-management", Name: "Budget Management", Category: "finance"},
			{ID: "procurement", Name: "Procurement", Category: "finance"},
			{ID: "internal-audit", Name: "Internal Audit", Category: "finance"},
		},
	},
	{
		ID: "hr", Name: "Human Resources", Icon: "🧑‍💼", Description: "People operations and staffing",
		Groups: []model.Group{
			{ID: "human-resources", Name: "Human Resources", Category: "hr"},
			{ID: "recruitment", Name: "Recruitment", Category: "hr"},
			{ID: "training-capacity", Name: "Training & Capacity Building", Category: "hr"},
			{ID: "staff-welfare", Name: "Staff Welfare", Category: "hr"},
			{ID: "performance-management", Name: "Performance Management", Category: "hr"},
		},
	},
	{
		ID: "operations", Name: "Operations", Icon: "⚙️", Description: "Logistics and field operations",
		Groups: []model.Group{
			{ID: "operations", Name: "Operations", Category: "operations"},
			{ID: "logistics", Name: "Logistics", Category: "operations"},
			{ID: "transport", Name: "Transport", Category: "operations"},
			{ID: "field-operations", Name: "Field Operations", Category: "operations"},
			{ID: "asset-management", Name: "Asset Management", Category: "operations"},
			{ID: "facilities-management", Name: "Facilities Management", Category: "operations"},
		},
	},
	{
		ID: "legal", Name: "Legal & Compliance", Icon: "⚖️", Description: "Legal, compliance and risk",
		Groups: []model.Group{
			{ID: "legal-affairs", Name: "Legal Affairs", Category: "legal"},
			{ID: "compliance", Name: "Compliance", Category: "legal"},
			{ID: "risk-management", Name: "Risk Management", Category: "legal"},
			{ID: "internal-audit-legal", Name: "Internal Audit", Category: "legal"},
			{ID: "data-protection", Name: "Data Protection", Category: "legal"},
		},
	},
	{
		ID: "communication", Name: "Communication", Icon: "📣", Description: "Public relations and marketing",
		Groups: []model.Group{
			{ID: "communications", Name: "Communications", Category: "communication"},
			{ID: "public-relations", Name: "Public Relations", Category: "communication"},
			{ID: "marketing", Name: "Marketing", Category: "communication"},
			{ID: "media-team", Name: "Media Team", Category: "communication"},
			{ID: "content-development", Name: "Content Development", Category: "communication"},
		},
	},
	{
		ID: "education", Name: "Education & Training", Icon: "🎓", Description: "Learning and capacity building",
		Groups: []model.Group{
			{ID: "training-team", Name: "Training Team", Category: "education"},
			{ID: "instructors", Name: "Instructors", Category: "education"},
			{ID: "students", Name: "Students", Category: "education"},
			{ID: "mentors", Name: "Mentors", Category: "education"},
		},
	},
	{
		ID: "projects", Name: "Project Groups", Icon: "📂", Description: "Auto-created for every project",
		Groups: []model.Group{
			{ID: "project-team", Name: "Project Team", Category: "projects"},
			{ID: "project-management", Name: "Project Management", Category: "projects"},
			{ID: "project-stakeholders", Name: "Project Stakeholders", Category: "projects"},
			{ID: "project-finance", Name: "Project Finance", Category: "projects"},
			{ID: "project-technical", Name: "Project Technical Team", Category: "projects"},
			{ID: "project-monitoring", Name: "Project Monitoring", Category: "projects"},
		},
	},
	{
		ID: "research-groups", Name: "Research Groups", Icon: "🔬", Description: "Auto-created for research studies",
		Groups: []model.Group{
			{ID: "research-discussion", Name: "Research Discussion", Category: "research-groups"},
			{ID: "research-analysis", Name: "Research Analysis", Category: "research-groups"},
			{ID: "research-publications", Name: "Research Publications", Category: "research-groups"},
			{ID: "research-supervisors", Name: "Research Supervisors", Category: "research-groups"},
			{ID: "research-collection", Name: "Research Data Collection", Category: "research-groups"},
			{ID: "research-review", Name: "Research Review", Category: "research-groups"},
		},
	},
	{
		ID: "organization", Name: "Organization Groups", Icon: "🏢", Description: "Auto-created for each organization",
		Groups: []model.Group{
			{ID: "org-general", Name: "Organization General", Category: "organization"},
			{ID: "org-management", Name: "Organization Management", Category: "organization"},
			{ID: "org-finance", Name: "Organization Finance", Category: "organization"},
			{ID: "org-hr", Name: "Organization HR", Category: "organization"},
			{ID: "org-it", Name: "Organization IT", Category: "organization"},
			{ID: "org-research", Name: "Organization Research", Category: "organization"},
			{ID: "org-statistics", Name: "Organization Statistics", Category: "organization"},
			{ID: "org-projects", Name: "Organization Projects", Category: "organization"},
		},
	},
	{
		ID: "department", Name: "Department Groups", Icon: "🏛️", Description: "Auto-created for each department",
		Groups: []model.Group{
			{ID: "dept-general", Name: "Department General", Category: "department"},
			{ID: "dept-announcements", Name: "Department Announcements", Category: "department"},
			{ID: "dept-meetings", Name: "Department Meetings", Category: "department"},
			{ID: "dept-tasks", Name: "Department Tasks", Category: "department"},
			{ID: "dept-files", Name: "Department Files", Category: "department"},
		},
	},
	{
		ID: "regional", Name: "District & Regional", Icon: "📍", Description: "Health systems and regional coordination",
		Groups: []model.Group{
			{ID: "hmis-team", Name: "National HMIS Team", Category: "regional"},
			{ID: "regional-coordinators", Name: "Regional Coordinators", Category: "regional"},
			{ID: "district-biostatisticians", Name: "District Biostatisticians", Category: "regional"},
			{ID: "district-health-officers", Name: "District Health Officers", Category: "regional"},
			{ID: "facility-in-charges", Name: "Facility In-Charges", Category: "regional"},
			{ID: "health-info-assistants", Name: "Health Information Assistants", Category: "regional"},
			{ID: "regional-surveillance", Name: "Regional Surveillance Team", Category: "regional"},
		},
	},
	{
		ID: "communities", Name: "Communities of Practice", Icon: "🧠", Description: "Cross-organizational knowledge sharing",
		Groups: []model.Group{
			{ID: "stats-community", Name: "Statistics Community", Category: "communities"},
			{ID: "research-community", Name: "Research Community", Category: "communities"},
			{ID: "public-health-community", Name: "Public Health Community", Category: "communities"},
			{ID: "ai-community", Name: "AI Community", Category: "communities"},
			{ID: "gis-community", Name: "GIS Community", Category: "communities"},
			{ID: "data-eng-community", Name: "Data Engineering Community", Category: "communities"},
			{ID: "me-community", Name: "M&E Community", Category: "communities"},
			{ID: "health-info-community", Name: "Health Informatics Community", Category: "communities"},
		},
	},
	{
		ID: "events", Name: "Temporary Events", Icon: "🎪", Description: "Auto-created for events, auto-archived after",
		Groups: []model.Group{
			{ID: "conferences", Name: "Conferences", Category: "events"},
			{ID: "workshops", Name: "Workshops", Category: "events"},
			{ID: "trainings", Name: "Trainings", Category: "events"},
			{ID: "meetings", Name: "Meetings", Category: "events"},
			{ID: "field-activities", Name: "Field Activities", Category: "events"},
			{ID: "hackathons", Name: "Hackathons", Category: "events"},
		},
	},
	{
		ID: "private", Name: "Private Groups", Icon: "🔒", Description: "Private collaboration spaces",
		Groups: []model.Group{
			{ID: "small-teams", Name: "Small Teams", Category: "private"},
			{ID: "confidential-projects", Name: "Confidential Projects", Category: "private"},
			{ID: "executive-discussions", Name: "Executive Discussions", Category: "private"},
			{ID: "research-collabs", Name: "Research Collaborations", Category: "private"},
		},
	},
}

func GetGroupTemplates() []model.GroupCategory {
	return groupTemplates
}

func ensureGroupConversations(ctx context.Context) error {
	var userCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	userIDs := make([]string, 0, userCount)
	rows, err := db.QueryContext(ctx, `SELECT id FROM users ORDER BY id LIMIT 15`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		userIDs = append(userIDs, id)
	}
	if len(userIDs) == 0 {
		return nil
	}
	teamMembersJSON, _ := json.Marshal(userIDs)

	for _, cat := range groupTemplates {
		for _, g := range cat.Groups {
			_, err := db.ExecContext(ctx, `INSERT INTO conversations (id, name, type, channel_id, member_ids, category) VALUES ($1, $2, $3, NULL, $4, $5) ON CONFLICT (id) DO UPDATE SET category = EXCLUDED.category`, "group-"+g.ID, g.Name, model.ConversationTypeGroup, teamMembersJSON, cat.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func fixLegacyMemberIDs(ctx context.Context) error {
	rows, err := db.QueryContext(ctx, `SELECT id, member_ids FROM conversations`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type convRow struct {
		id        string
		memberIDs []byte
	}
	var rowsToFix []convRow
	for rows.Next() {
		var c convRow
		if err := rows.Scan(&c.id, &c.memberIDs); err != nil {
			return err
		}
		var ids []string
		if err := json.Unmarshal(c.memberIDs, &ids); err != nil {
			continue
		}
		changed := false
		for i, id := range ids {
			if id == "user-1" {
				ids[i] = "user-001"
				changed = true
			} else if id == "user-2" {
				ids[i] = "user-002"
				changed = true
			} else if id == "user-3" {
				ids[i] = "user-003"
				changed = true
			} else if id == "user-4" {
				ids[i] = "user-004"
				changed = true
			}
		}
		if changed {
			rowsToFix = append(rowsToFix, convRow{id: c.id, memberIDs: mustJSON(ids)})
		}
	}
	for _, c := range rowsToFix {
		if _, err := db.ExecContext(ctx, `UPDATE conversations SET member_ids = $1 WHERE id = $2`, c.memberIDs, c.id); err != nil {
			return err
		}
	}
	return nil
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func resetSeedData(ctx context.Context) error {
	// No longer used — seeding is conditional on empty tables only.
	// Kept for backwards compatibility but does nothing.
	return nil
}

func NewClient(conn *websocket.Conn, conversation string) *Client {
	return &Client{conn: conn, conversation: conversation}
}

func (c *Client) SetConversation(conversation string) {
	c.conversation = conversation
}

func RegisterClient(client *Client) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	clients[client] = true
}

func UnregisterClient(client *Client) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	delete(clients, client)
}

func GetCurrentUser() (model.User, error) {
	return GetUserByID("user-001")
}

func GetUserByID(userID string) (model.User, error) {
	var user model.User
	var rolesJSON []byte
	var avatarURL sql.NullString
	var about sql.NullString
	var presence sql.NullString
	err := db.QueryRowContext(context.Background(), `SELECT id, name, email, organization_id, roles, avatar_url, about, presence FROM users WHERE id = $1`, userID).Scan(&user.ID, &user.Name, &user.Email, &user.OrganizationID, &rolesJSON, &avatarURL, &about, &presence)
	if err != nil {
		return user, err
	}
	if err = json.Unmarshal(rolesJSON, &user.Roles); err != nil {
		return user, err
	}
	user.AvatarURL = avatarURL.String
	user.About = about.String
	user.Presence = presence.String
	return user, nil
}

func UpdateUserProfile(userID string, name string, about string, avatarURL string) error {
	// Always update all three fields — empty avatarURL means "remove photo"
	_, err := db.ExecContext(context.Background(), `UPDATE users SET name = $1, about = $2, avatar_url = $3 WHERE id = $4`, name, about, avatarURL, userID)
	return err
}

func GetUserSettings(userID string) (model.UserSettings, error) {
	var s model.UserSettings
	err := db.QueryRowContext(context.Background(), `
SELECT user_id, theme, accent_color, font_size, enter_to_send, language, last_seen, profile_photo,
       read_receipts, typing_indicator, voice_notes, read_by_default, auto_download,
       notif_messages, notif_groups, notif_mentions, notif_meetings, notif_sound, notif_preview,
       download_images, download_videos, download_documents
FROM user_settings WHERE user_id = $1`, userID).Scan(
		&s.UserID, &s.Theme, &s.AccentColor, &s.FontSize, &s.EnterToSend, &s.Language, &s.LastSeen, &s.ProfilePhoto,
		&s.ReadReceipts, &s.TypingIndicator, &s.VoiceNotes, &s.ReadByDefault, &s.AutoDownload,
		&s.NotifMessages, &s.NotifGroups, &s.NotifMentions, &s.NotifMeetings, &s.NotifSound, &s.NotifPreview,
		&s.DownloadImages, &s.DownloadVideos, &s.DownloadDocuments,
	)
	if err == sql.ErrNoRows {
		return s, nil
	}
	return s, err
}

func UpsertUserSettings(s model.UserSettings) error {
	_, err := db.ExecContext(context.Background(), `
INSERT INTO user_settings (
  user_id, theme, accent_color, font_size, enter_to_send, language, last_seen, profile_photo,
  read_receipts, typing_indicator, voice_notes, read_by_default, auto_download,
  notif_messages, notif_groups, notif_mentions, notif_meetings, notif_sound, notif_preview,
  download_images, download_videos, download_documents
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
ON CONFLICT (user_id) DO UPDATE SET
  theme = EXCLUDED.theme, accent_color = EXCLUDED.accent_color, font_size = EXCLUDED.font_size,
  enter_to_send = EXCLUDED.enter_to_send, language = EXCLUDED.language, last_seen = EXCLUDED.last_seen,
  profile_photo = EXCLUDED.profile_photo, read_receipts = EXCLUDED.read_receipts,
  typing_indicator = EXCLUDED.typing_indicator, voice_notes = EXCLUDED.voice_notes,
  read_by_default = EXCLUDED.read_by_default, auto_download = EXCLUDED.auto_download,
  notif_messages = EXCLUDED.notif_messages, notif_groups = EXCLUDED.notif_groups,
  notif_mentions = EXCLUDED.notif_mentions, notif_meetings = EXCLUDED.notif_meetings,
  notif_sound = EXCLUDED.notif_sound, notif_preview = EXCLUDED.notif_preview,
  download_images = EXCLUDED.download_images, download_videos = EXCLUDED.download_videos,
  download_documents = EXCLUDED.download_documents`,
		s.UserID, s.Theme, s.AccentColor, s.FontSize, s.EnterToSend, s.Language, s.LastSeen, s.ProfilePhoto,
		s.ReadReceipts, s.TypingIndicator, s.VoiceNotes, s.ReadByDefault, s.AutoDownload,
		s.NotifMessages, s.NotifGroups, s.NotifMentions, s.NotifMeetings, s.NotifSound, s.NotifPreview,
		s.DownloadImages, s.DownloadVideos, s.DownloadDocuments,
	)
	return err
}

func GetAllUsers() ([]model.User, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, name, email, organization_id, roles, avatar_url, about, presence FROM users ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []model.User{}
	for rows.Next() {
		var user model.User
		var rolesJSON []byte
		var avatarURL sql.NullString
		var about sql.NullString
		var presence sql.NullString
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.OrganizationID, &rolesJSON, &avatarURL, &about, &presence); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rolesJSON, &user.Roles); err != nil {
			return nil, err
		}
		user.AvatarURL = avatarURL.String
		user.About = about.String
		user.Presence = presence.String
		users = append(users, user)
	}
	return users, rows.Err()
}

func CreateGroupConversation(groupID string, name string, memberIDs []string) (model.Conversation, error) {
	memberJSON, _ := json.Marshal(memberIDs)
	convID := fmt.Sprintf("group-%s", groupID)

	var conv model.Conversation
	var memberIDsJSON []byte
	var channelID sql.NullString
	var convType string
	err := db.QueryRowContext(context.Background(), `SELECT id, name, type, channel_id, member_ids FROM conversations WHERE id = $1`, convID).Scan(&conv.ID, &conv.Name, &convType, &channelID, &memberIDsJSON)
	if err == nil {
		conv.Type = model.ConversationType(convType)
		conv.ChannelID = channelID.String
		json.Unmarshal(memberIDsJSON, &conv.MemberIDs)
		return conv, nil
	}

	_, err = db.ExecContext(context.Background(), `INSERT INTO conversations (id, name, type, channel_id, member_ids) VALUES ($1, $2, $3, NULL, $4) ON CONFLICT DO NOTHING`, convID, name, model.ConversationTypeGroup, memberJSON)
	if err != nil {
		return conv, err
	}

	conv = model.Conversation{
		ID:        convID,
		Name:      name,
		Type:      model.ConversationTypeGroup,
		MemberIDs: memberIDs,
	}
	return conv, nil
}

func CreateDirectConversation(user1ID string, user2ID string, name string) (model.Conversation, error) {
	convID := fmt.Sprintf("dm-%s-%s", user1ID, user2ID)
	memberIDs, _ := json.Marshal([]string{user1ID, user2ID})

	// Try to find existing conversation first
	var conv model.Conversation
	var memberIDsJSON []byte
	var channelID sql.NullString
	var convType string
	err := db.QueryRowContext(context.Background(), `SELECT id, name, type, channel_id, member_ids FROM conversations WHERE id = $1`, convID).Scan(&conv.ID, &conv.Name, &convType, &channelID, &memberIDsJSON)
	if err == nil {
		conv.Type = model.ConversationType(convType)
		conv.ChannelID = channelID.String
		json.Unmarshal(memberIDsJSON, &conv.MemberIDs)
		return conv, nil
	}

	// Create new one
	_, err = db.ExecContext(context.Background(), `INSERT INTO conversations (id, name, type, channel_id, member_ids) VALUES ($1, $2, $3, NULL, $4) ON CONFLICT DO NOTHING`, convID, name, model.ConversationTypeDirect, memberIDs)
	if err != nil {
		return conv, err
	}

	conv = model.Conversation{
		ID:        convID,
		Name:      name,
		Type:      model.ConversationTypeDirect,
		MemberIDs: []string{user1ID, user2ID},
	}
	return conv, nil
}

func GetChannels() ([]model.Channel, error) {
	rows, err := db.QueryContext(context.Background(), `SELECT id, name FROM channels ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := []model.Channel{}
	for rows.Next() {
		var channel model.Channel
		if err := rows.Scan(&channel.ID, &channel.Name); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, rows.Err()
}

func GetConversations() ([]model.Conversation, error) {
	rows, err := db.QueryContext(context.Background(), `
SELECT id, name, type, channel_id, member_ids, COALESCE(category, ''), COALESCE(latest_preview, ''), latest_message_at, COALESCE(attachment_count, 0)
FROM (
  SELECT c.id, c.name, c.type, c.channel_id, c.member_ids, c.category,
         m.text AS latest_preview,
         m.created_at AS latest_message_at,
         COALESCE(a.attachment_count, 0) AS attachment_count
  FROM conversations c
  LEFT JOIN LATERAL (
    SELECT id, text, created_at
    FROM messages
    WHERE conversation_id = c.id AND status != 'deleted'
    ORDER BY created_at DESC
    LIMIT 1
  ) m ON true
  LEFT JOIN LATERAL (
    SELECT COUNT(1) AS attachment_count
    FROM message_attachments a
    WHERE a.message_id = m.id
  ) a ON true
) q
ORDER BY name
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := []model.Conversation{}
	for rows.Next() {
		var conversation model.Conversation
		var memberIDsJSON []byte
		var channelID sql.NullString
		var convType string
		var category string
		var latestPreview string
		var latestMessageAt sql.NullTime
		var attachmentCount int
		if err := rows.Scan(&conversation.ID, &conversation.Name, &convType, &channelID, &memberIDsJSON, &category, &latestPreview, &latestMessageAt, &attachmentCount); err != nil {
			return nil, err
		}
		conversation.Type = model.ConversationType(convType)
		conversation.ChannelID = channelID.String
		conversation.Category = category
		conversation.LatestPreview = latestPreview
		conversation.AttachmentCount = attachmentCount
		if latestMessageAt.Valid {
			conversation.LatestMessageAt = latestMessageAt.Time
		}
		if err := json.Unmarshal(memberIDsJSON, &conversation.MemberIDs); err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}

func GetMessages(conversationID string) ([]model.Message, error) {
	var rows *sql.Rows
	var err error

	if conversationID == "" {
		rows, err = db.QueryContext(context.Background(), `SELECT id, conversation_id, channel_id, sender, text, created_at, updated_at, deleted_at, parent_message_id, thread_root_id, status FROM messages WHERE status != 'deleted' ORDER BY created_at`)
	} else {
		rows, err = db.QueryContext(context.Background(), `SELECT id, conversation_id, channel_id, sender, text, created_at, updated_at, deleted_at, parent_message_id, thread_root_id, status FROM messages WHERE conversation_id = $1 AND status != 'deleted' ORDER BY created_at`, conversationID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []model.Message{}
	for rows.Next() {
		var msg model.Message
		var channelID sql.NullString
		var updatedAt sql.NullTime
		var deletedAt sql.NullTime
		var parentID sql.NullString
		var threadRootID sql.NullString
		if err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&channelID,
			&msg.Sender,
			&msg.Text,
			&msg.CreatedAt,
			&updatedAt,
			&deletedAt,
			&parentID,
			&threadRootID,
			&msg.Status,
		); err != nil {
			return nil, err
		}
		msg.ChannelID = channelID.String
		if updatedAt.Valid {
			msg.UpdatedAt = updatedAt.Time
		}
		if deletedAt.Valid {
			msg.DeletedAt = deletedAt.Time
		}
		msg.ParentMessageID = parentID.String
		msg.ThreadRootID = threadRootID.String
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func StoreMessage(message model.Message) error {
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now().UTC()
	}
	if message.Status == "" {
		message.Status = "active"
	}
	_, err := db.ExecContext(context.Background(), `INSERT INTO messages (id, conversation_id, channel_id, sender, text, created_at, updated_at, deleted_at, parent_message_id, thread_root_id, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		message.ID,
		message.ConversationID,
		message.ChannelID,
		message.Sender,
		message.Text,
		message.CreatedAt,
		nullTime(message.UpdatedAt),
		nullTime(message.DeletedAt),
		nullString(message.ParentMessageID),
		nullString(message.ThreadRootID),
		message.Status,
	)
	return err
}

func UpdateMessage(message model.Message) error {
	message.UpdatedAt = time.Now().UTC()
	_, err := db.ExecContext(context.Background(), `UPDATE messages SET text = $1, updated_at = $2, parent_message_id = $3, thread_root_id = $4 WHERE id = $5`, message.Text, message.UpdatedAt, nullString(message.ParentMessageID), nullString(message.ThreadRootID), message.ID)
	return err
}

func SoftDeleteMessage(messageID string) error {
	_, err := db.ExecContext(context.Background(), `UPDATE messages SET status = 'deleted', deleted_at = $1 WHERE id = $2`, time.Now().UTC(), messageID)
	return err
}

func GetMessageByID(messageID string) (model.Message, error) {
	var msg model.Message
	var channelID sql.NullString
	var updatedAt sql.NullTime
	var deletedAt sql.NullTime
	var parentID sql.NullString
	var threadRootID sql.NullString
	var status sql.NullString

	err := db.QueryRowContext(context.Background(), `SELECT id, conversation_id, channel_id, sender, text, created_at, updated_at, deleted_at, parent_message_id, thread_root_id, status FROM messages WHERE id = $1`, messageID).Scan(
		&msg.ID,
		&msg.ConversationID,
		&channelID,
		&msg.Sender,
		&msg.Text,
		&msg.CreatedAt,
		&updatedAt,
		&deletedAt,
		&parentID,
		&threadRootID,
		&status,
	)
	if err != nil {
		return msg, err
	}
	msg.ChannelID = channelID.String
	if updatedAt.Valid {
		msg.UpdatedAt = updatedAt.Time
	}
	if deletedAt.Valid {
		msg.DeletedAt = deletedAt.Time
	}
	msg.ParentMessageID = parentID.String
	msg.ThreadRootID = threadRootID.String
	msg.Status = status.String
	return msg, nil
}

func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func BroadcastMessage(message model.Message) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	for client := range clients {
		if client.conversation != message.ConversationID {
			continue
		}
		if err := client.conn.WriteJSON(message); err != nil {
			client.conn.Close()
			delete(clients, client)
		}
	}
}
