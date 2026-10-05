package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegisterDOIWithCrossref(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if got := r.FormValue("operation"); got != "doQueryUpload" {
			t.Errorf("operation = %q, want doQueryUpload", got)
		}
		if got := r.FormValue("login_id"); got != "crossref-user" {
			t.Errorf("login_id = %q, want crossref-user", got)
		}
		if got := r.FormValue("login_passwd"); got != "crossref-password" {
			t.Errorf("login_passwd = %q, want crossref-password", got)
		}
		file, header, err := r.FormFile("fname")
		if err != nil {
			t.Fatalf("deposit file: %v", err)
		}
		defer file.Close()
		if !strings.HasSuffix(header.Filename, ".xml") {
			t.Errorf("filename = %q, want XML suffix", header.Filename)
		}
		deposit, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read deposit: %v", err)
		}
		for _, expected := range []string{"doi_batch", "10.1234/example", "A research title", "Doe", "Jane"} {
			if !strings.Contains(string(deposit), expected) {
				t.Errorf("deposit does not contain %q: %s", expected, string(deposit))
			}
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<doi_batch_diagnostic status="completed"><submission_id>submission-123</submission_id><batch_id>batch-123</batch_id></doi_batch_diagnostic>`)
	}))
	defer server.Close()

	t.Setenv("RMS_CROSSREF_USERNAME", "crossref-user")
	t.Setenv("RMS_CROSSREF_PASSWORD", "crossref-password")
	t.Setenv("RMS_CROSSREF_DEPOSITOR_EMAIL", "doi@example.org")
	t.Setenv("RMS_CROSSREF_DEPOSIT_URL", server.URL)
	t.Setenv("RMS_CROSSREF_DEPOSITOR_NAME", "StatGate Depositor")

	externalID, err := registerDOIWithCrossref(context.Background(), DOIRecord{
		ID:        "record-1",
		DOI:       "10.1234/example",
		Title:     "A research title",
		Authors:   []string{"Doe, Jane"},
		Year:      2026,
		Publisher: "StatGate Journal",
		URL:       "https://example.org/research/record-1",
	})
	if err != nil {
		t.Fatalf("register DOI: %v", err)
	}
	if externalID != "submission-123" {
		t.Fatalf("external ID = %q, want submission-123", externalID)
	}
}

func TestRegisterDOIWithCrossrefRequiresConfiguration(t *testing.T) {
	t.Setenv("RMS_CROSSREF_USERNAME", "")
	t.Setenv("RMS_CROSSREF_PASSWORD", "")
	t.Setenv("RMS_CROSSREF_DEPOSITOR_EMAIL", "")

	_, err := registerDOIWithCrossref(context.Background(), DOIRecord{DOI: "10.1234/example"})
	if err == nil {
		t.Fatal("expected missing configuration error")
	}
	if got := providerErrorCode(err); got != http.StatusServiceUnavailable {
		t.Fatalf("error code = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestBuildCrossrefDepositXML(t *testing.T) {
	deposit, err := buildCrossrefDeposit(DOIRecord{
		ID:      "record-2",
		DOI:     "10.1234/xml",
		Title:   "XML title",
		Authors: []string{"Doe, Jane", "Smith"},
		Year:    2025,
	}, "batch-2")
	if err != nil {
		t.Fatalf("build deposit: %v", err)
	}
	contents := string(deposit)
	for _, expected := range []string{
		`version="4.4.0"`,
		`doi_batch_id>batch-2`,
		`given_name>Jane`,
		`surname>Doe`,
		`surname>Smith`,
		`doi>10.1234/xml`,
	} {
		if !strings.Contains(contents, expected) {
			t.Errorf("XML does not contain %q: %s", expected, contents)
		}
	}
}
