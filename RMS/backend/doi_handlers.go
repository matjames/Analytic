package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func loadDOIRecordForWorkspace(id, workspaceID string) (DOIRecord, error) {
	var record DOIRecord
	var authors string
	var lastAttempt sql.NullTime
	err := DB.QueryRow(`SELECT d.id, d.research_id, COALESCE(d.publication_id,''), d.doi, d.title, d.authors, d.year,
		d.publisher, COALESCE(d.url,''), d.status, COALESCE(d.provider,'local'), COALESCE(d.provider_status,'Not Submitted'),
		COALESCE(d.external_id,''), COALESCE(d.registration_attempts,0), COALESCE(d.last_registration_error,''), d.last_attempt_at, d.created_time
		FROM rms.doi_records d JOIN rms.research_projects p ON p.id=d.research_id
		WHERE d.id=$1 AND (p.workspace_id = NULLIF($2,'') OR NULLIF($2,'') IS NULL)`, id, workspaceID).Scan(
		&record.ID, &record.ResearchID, &record.PublicationID, &record.DOI, &record.Title, &authors, &record.Year,
		&record.Publisher, &record.URL, &record.Status, &record.Provider, &record.ProviderStatus, &record.ExternalID,
		&record.RegistrationAttempts, &record.LastRegistrationError, &lastAttempt, &record.CreatedTime)
	if err != nil {
		return record, err
	}
	record.Authors = splitAuthors(authors)
	if lastAttempt.Valid {
		record.LastAttemptAt = &lastAttempt.Time
	}
	return record, nil
}

func dbRegisterDOIRecord(c *gin.Context) {
	record, err := loadDOIRecordForWorkspace(c.Param("id"), workspaceIDContext(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DOI record not found"})
		return
	}
	var request struct {
		Provider string `json:"provider"`
	}
	if c.Request.ContentLength > 0 {
		if err := json.NewDecoder(c.Request.Body).Decode(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	provider := strings.ToLower(strings.TrimSpace(request.Provider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(record.Provider))
	}
	if provider == "" {
		provider = "local"
	}
	if record.ProviderStatus == "Registered" && strings.EqualFold(record.Provider, provider) {
		c.JSON(http.StatusOK, record)
		return
	}

	attempt := record.RegistrationAttempts + 1
	now := time.Now().UTC()
	if _, err := DB.Exec(`UPDATE rms.doi_records SET provider=$1, provider_status='Pending', registration_attempts=$2,
		last_registration_error=NULL, last_attempt_at=$3 WHERE id=$4 AND EXISTS (
		SELECT 1 FROM rms.research_projects p WHERE p.id=rms.doi_records.research_id AND (p.workspace_id = NULLIF($5,'') OR NULLIF($5,'') IS NULL))`,
		provider, attempt, now, record.ID, workspaceIDContext(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not start DOI registration"})
		return
	}

	externalID, registrationErr := registerDOIWithProvider(c.Request.Context(), record, provider)
	if registrationErr != nil {
		message := providerErrorMessage(registrationErr)
		_, _ = DB.Exec(`UPDATE rms.doi_records SET provider=$1, provider_status='Failed', last_registration_error=$2,
			last_attempt_at=$3 WHERE id=$4`, provider, message, now, record.ID)
		record.Provider, record.ProviderStatus, record.RegistrationAttempts = provider, "Failed", attempt
		record.LastRegistrationError, record.LastAttemptAt = message, &now
		c.JSON(providerErrorCode(registrationErr), gin.H{
			"error": message, "doi": record.DOI, "provider": provider,
			"providerStatus": record.ProviderStatus, "registrationAttempts": attempt,
		})
		return
	}

	if _, err := DB.Exec(`UPDATE rms.doi_records SET provider=$1, provider_status='Registered', external_id=$2,
		last_registration_error=NULL, last_attempt_at=$3 WHERE id=$4`, provider, externalID, now, record.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record DOI registration"})
		return
	}
	record.Provider, record.ProviderStatus, record.ExternalID, record.RegistrationAttempts = provider, "Registered", externalID, attempt
	record.LastRegistrationError, record.LastAttemptAt = "", &now
	c.JSON(http.StatusOK, record)
}
