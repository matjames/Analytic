package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

type doiProviderError struct {
	Code int
	Err  error
}

func (e *doiProviderError) Error() string { return e.Err.Error() }

func registerDOIWithProvider(ctx context.Context, record DOIRecord, provider string) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(getEnv("RMS_DOI_PROVIDER", "local")))
	}
	switch provider {
	case "local":
		return record.DOI, nil
	case "datacite":
		return registerDOIWithDataCite(ctx, record)
	case "crossref":
		return registerDOIWithCrossref(ctx, record)
	default:
		return "", &doiProviderError{Code: http.StatusBadRequest, Err: fmt.Errorf("unsupported DOI provider %q", provider)}
	}
}

type crossrefDOIBatch struct {
	XMLName        xml.Name     `xml:"doi_batch"`
	Version        string       `xml:"version,attr"`
	Namespace      string       `xml:"xmlns,attr"`
	XSI            string       `xml:"xmlns:xsi,attr"`
	SchemaLocation string       `xml:"xsi:schemaLocation,attr"`
	Head           crossrefHead `xml:"head"`
	Body           crossrefBody `xml:"body"`
}

type crossrefHead struct {
	BatchID    string            `xml:"doi_batch_id"`
	Timestamp  string            `xml:"timestamp"`
	Depositor  crossrefDepositor `xml:"depositor"`
	Registrant string            `xml:"registrant"`
}

type crossrefDepositor struct {
	Name  string `xml:"depositor_name"`
	Email string `xml:"email_address"`
}

type crossrefBody struct {
	Journal crossrefJournal `xml:"journal"`
}

type crossrefJournal struct {
	Metadata crossrefJournalMetadata `xml:"journal_metadata"`
	Article  crossrefJournalArticle  `xml:"journal_article"`
}

type crossrefJournalMetadata struct {
	Title string `xml:"full_title"`
}

type crossrefJournalArticle struct {
	PublicationType string                  `xml:"publication_type,attr"`
	Titles          crossrefTitles          `xml:"titles"`
	Contributors    *crossrefContributors   `xml:"contributors,omitempty"`
	PublicationDate crossrefPublicationDate `xml:"publication_date"`
	DOIData         crossrefDOIData         `xml:"doi_data"`
}

type crossrefTitles struct {
	Title string `xml:"title"`
}

type crossrefContributors struct {
	Authors []crossrefPersonName `xml:"person_name"`
}

type crossrefPersonName struct {
	Role     string `xml:"contributor_role,attr"`
	Sequence string `xml:"sequence,attr"`
	Given    string `xml:"given_name,omitempty"`
	Surname  string `xml:"surname"`
}

type crossrefPublicationDate struct {
	MediaType string `xml:"media_type,attr"`
	Year      int    `xml:"year"`
}

type crossrefDOIData struct {
	DOI      string `xml:"doi"`
	Resource string `xml:"resource"`
}

func registerDOIWithCrossref(ctx context.Context, record DOIRecord) (string, error) {
	username := strings.TrimSpace(getEnv("RMS_CROSSREF_USERNAME", ""))
	password := getEnv("RMS_CROSSREF_PASSWORD", "")
	depositorEmail := strings.TrimSpace(getEnv("RMS_CROSSREF_DEPOSITOR_EMAIL", ""))
	if username == "" || password == "" || depositorEmail == "" {
		return "", &doiProviderError{Code: http.StatusServiceUnavailable, Err: fmt.Errorf("Crossref credentials and depositor email are not configured")}
	}

	batchID := fmt.Sprintf("statgate-%s-%d", crossrefSafeID(record.ID, record.DOI), time.Now().UnixNano())
	deposit, err := buildCrossrefDeposit(record, batchID)
	if err != nil {
		return "", err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{
		"operation":    getEnv("RMS_CROSSREF_OPERATION", "doQueryUpload"),
		"login_id":     username,
		"login_passwd": password,
	} {
		if err := writer.WriteField(key, value); err != nil {
			return "", err
		}
	}
	fileHeader := make(textproto.MIMEHeader)
	fileHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="fname"; filename="%s.xml"`, batchID))
	fileHeader.Set("Content-Type", "application/xml")
	part, err := writer.CreatePart(fileHeader)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(deposit); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	endpoint := getEnv("RMS_CROSSREF_DEPOSIT_URL", "https://doi.crossref.org/servlet/deposit")
	requestCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/xml, text/plain")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "StatGate-RMS DOI deposit")
	resp, err := (&http.Client{Timeout: 22 * time.Second}).Do(req)
	if err != nil {
		return "", &doiProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("Crossref request failed: %w", err)}
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &doiProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("Crossref returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))}
	}

	var diagnostic struct {
		Status       string `xml:"status,attr"`
		BatchID      string `xml:"batch_id"`
		SubmissionID string `xml:"submission_id"`
	}
	if err := xml.Unmarshal(responseBody, &diagnostic); err != nil {
		return "", &doiProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("Crossref returned invalid XML: %w", err)}
	}
	status := strings.ToLower(strings.TrimSpace(diagnostic.Status))
	if status != "completed" && status != "queued" {
		return "", &doiProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("Crossref deposit was not accepted (status %q)", diagnostic.Status)}
	}
	return firstNonEmpty(diagnostic.SubmissionID, diagnostic.BatchID, batchID), nil
}

func buildCrossrefDeposit(record DOIRecord, batchID string) ([]byte, error) {
	authors := make([]crossrefPersonName, 0, len(record.Authors))
	for _, author := range record.Authors {
		name := strings.TrimSpace(author)
		if name == "" {
			continue
		}
		given, surname := crossrefAuthorName(name)
		sequence := "additional"
		if len(authors) == 0 {
			sequence = "first"
		}
		authors = append(authors, crossrefPersonName{
			Role:     "author",
			Sequence: sequence,
			Given:    given,
			Surname:  surname,
		})
	}
	var contributors *crossrefContributors
	if len(authors) > 0 {
		contributors = &crossrefContributors{Authors: authors}
	}
	year := record.Year
	if year <= 0 {
		year = time.Now().Year()
	}
	batch := crossrefDOIBatch{
		Version:        "4.4.0",
		Namespace:      "http://www.crossref.org/schema/4.4.0",
		XSI:            "http://www.w3.org/2001/XMLSchema-instance",
		SchemaLocation: "http://www.crossref.org/schema/4.4.0 https://www.crossref.org/schemas/crossref4.4.0.xsd",
		Head: crossrefHead{
			BatchID:    batchID,
			Timestamp:  fmt.Sprintf("%d", time.Now().Unix()),
			Depositor:  crossrefDepositor{Name: getEnv("RMS_CROSSREF_DEPOSITOR_NAME", "StatGate"), Email: getEnv("RMS_CROSSREF_DEPOSITOR_EMAIL", "")},
			Registrant: getEnv("RMS_CROSSREF_REGISTRANT", "StatGate"),
		},
		Body: crossrefBody{Journal: crossrefJournal{
			Metadata: crossrefJournalMetadata{Title: firstNonEmpty(record.Publisher, "StatGate Research Output")},
			Article: crossrefJournalArticle{
				PublicationType: "full_text",
				Titles:          crossrefTitles{Title: record.Title},
				Contributors:    contributors,
				PublicationDate: crossrefPublicationDate{MediaType: "online", Year: year},
				DOIData:         crossrefDOIData{DOI: record.DOI, Resource: firstNonEmpty(record.URL, "https://doi.org/"+record.DOI)},
			},
		}},
	}
	return xml.MarshalIndent(batch, "", "  ")
}

func crossrefAuthorName(author string) (string, string) {
	parts := strings.SplitN(author, ",", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]), strings.TrimSpace(parts[0])
	}
	return "", strings.TrimSpace(author)
}

func crossrefSafeID(values ...string) string {
	value := firstNonEmpty(values...)
	var safe strings.Builder
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
			safe.WriteRune(character)
		case character >= 'A' && character <= 'Z':
			safe.WriteRune(character)
		case character >= '0' && character <= '9':
			safe.WriteRune(character)
		case character == '-' || character == '_' || character == '.':
			safe.WriteRune(character)
		default:
			safe.WriteByte('-')
		}
	}
	if safe.Len() == 0 {
		return "record"
	}
	return safe.String()
}

func registerDOIWithDataCite(ctx context.Context, record DOIRecord) (string, error) {
	username := strings.TrimSpace(getEnv("RMS_DATACITE_USERNAME", ""))
	password := getEnv("RMS_DATACITE_PASSWORD", "")
	token := strings.TrimSpace(getEnv("RMS_DATACITE_TOKEN", ""))
	if token == "" && (username == "" || password == "") {
		return "", &doiProviderError{Code: http.StatusServiceUnavailable, Err: fmt.Errorf("DataCite credentials are not configured")}
	}

	creators := make([]map[string]string, 0, len(record.Authors))
	for _, author := range record.Authors {
		if author = strings.TrimSpace(author); author != "" {
			creators = append(creators, map[string]string{"name": author})
		}
	}
	if len(creators) == 0 {
		creators = append(creators, map[string]string{"name": "StatGate Research Team"})
	}
	resourceType := "Text"
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "dois",
			"attributes": map[string]interface{}{
				"doi":             record.DOI,
				"event":           "publish",
				"creators":        creators,
				"titles":          []map[string]string{{"title": record.Title}},
				"publisher":       record.Publisher,
				"publicationYear": record.Year,
				"types":           map[string]string{"resourceTypeGeneral": resourceType},
				"url":             firstNonEmpty(record.URL, "https://doi.org/"+record.DOI),
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimRight(getEnv("RMS_DATACITE_API_URL", "https://api.datacite.org/dois"), "/")
	requestCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		basic := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req.Header.Set("Authorization", "Basic "+basic)
	}
	resp, err := (&http.Client{Timeout: 22 * time.Second}).Do(req)
	if err != nil {
		return "", &doiProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("DataCite request failed: %w", err)}
	}
	defer resp.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &doiProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("DataCite returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))}
	}
	var response struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", &doiProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("DataCite returned invalid JSON: %w", err)}
	}
	if strings.TrimSpace(response.Data.ID) == "" {
		return "", &doiProviderError{Code: http.StatusBadGateway, Err: fmt.Errorf("DataCite response did not include an external identifier")}
	}
	return response.Data.ID, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func providerErrorCode(err error) int {
	if providerErr, ok := err.(*doiProviderError); ok {
		return providerErr.Code
	}
	return http.StatusBadGateway
}

func providerErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
