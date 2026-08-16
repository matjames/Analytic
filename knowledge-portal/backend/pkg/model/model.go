package model

import "time"

// ContentItem represents a publication / article / news / blog / report /
// policy or journal entry managed by the CMS (P17).
type ContentItem struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	OrgID       string     `json:"org_id,omitempty"`
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Summary     string     `json:"summary,omitempty"`
	Body        string     `json:"body,omitempty"`
	Status      string     `json:"status"`
	AuthorID    string     `json:"author_id,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	UpdatedBy   string     `json:"updated_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// PublicDataset represents an open data set exposed on the Open Data Portal
// (P41), with versioning and machine-readable formats.
type PublicDataset struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	Description     string    `json:"description,omitempty"`
	License         string    `json:"license"`
	Format          string    `json:"format"`
	SizeBytes       int64     `json:"size_bytes"`
	DownloadURL     string    `json:"download_url,omitempty"`
	SourceApp       string    `json:"source_app,omitempty"`
	SourceObjType   string    `json:"source_object_type,omitempty"`
	SourceObjID     string    `json:"source_object_id,omitempty"`
	Status          string    `json:"status"`
	Version         string    `json:"version"`
	Tags            []string  `json:"tags,omitempty"`
	UpdatedBy       string    `json:"updated_by,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// RepositoryItem represents a library / repository asset in the National
// Digital Library & Knowledge Repository (P40).
type RepositoryItem struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Kind        string     `json:"kind"` // book|thesis|journal|article|report|working_paper|dataset|code|protocol|multimedia
	Title       string     `json:"title"`
	Authors     []string   `json:"authors,omitempty"`
	DOI         string     `json:"doi,omitempty"`
	ISBN        string     `json:"isbn,omitempty"`
	ISSN        string     `json:"issn,omitempty"`
	Abstract    string     `json:"abstract,omitempty"`
	Status      string     `json:"status"` // draft|in_review|published|archived
	Access      string     `json:"access"` // open|restricted|private
	Rights      string     `json:"rights,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	UpdatedBy   string     `json:"updated_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// PersistentIdentifier records a DOI / ORCID / ISBN / ISSN / handle (P40).
type PersistentIdentifier struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	ItemType  string    `json:"item_type"`
	ItemID    string    `json:"item_id"`
	IDType    string    `json:"id_type"` // doi|orcid|isbn|issn|handle
	IDValue   string    `json:"id_value"`
	Provider  string    `json:"provider,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Subscription is a newsletter / content subscription for the public portal
// and open data portal (P17 / P41).
type Subscription struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Topics    []string  `json:"topics,omitempty"`
	Frequency string    `json:"frequency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// Feedback covers public feedback, data requests, citizen reports and
// Freedom of Information requests (P17 / P41).
type Feedback struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Kind      string    `json:"kind"` // feedback|foi_request|data_request|citizen_report
	Contact   string    `json:"contact,omitempty"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ObjectLink mirrors the shared object_links contract so items published in
// this app stay connected to source objects across the platform.
type ObjectLink struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	TargetType   string    `json:"target_type"`
	TargetID     string    `json:"target_id"`
	Relationship string    `json:"relationship"`
	CreatedAt    time.Time `json:"created_at"`
}

// SearchResult is a combined cross-catalog public search hit (P17 / P47).
type SearchResult struct {
	Type        string `json:"type"` // content|dataset|repository
	ID          string `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary,omitempty"`
	Status      string `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// PortalSummary is the administrative overview of the portal.
type PortalSummary struct {
	ContentCount   int64 `json:"content_count"`
	DatasetCount   int64 `json:"dataset_count"`
	RepositoryCount int64 `json:"repository_count"`
	PublishedCount int64 `json:"published_count"`
	SubscriptionCount int64 `json:"subscription_count"`
	FeedbackCount  int64 `json:"feedback_count"`
}
