//go:build integration

// Package main — end-to-end integration tests.
//
// These tests exercise the full request pipeline against a real PostgreSQL
// instance seeded by `make test-db` (see database/testdb/README.md). They are
// excluded from the default `go test ./server` run by the `integration` build
// tag so the fast suite stays fast.
//
// Running:
//
//	make test-db                                          # seed the DB once
//	cp .env.localdb .env                                  # point server at it
//	CI_INTEGRATION_DB=1 go test -tags=integration ./server
//
// What these tests prove that unit tests can't:
//   - handler → YAML load → goroutine orchestration → SQL build → pgx bind
//     → real Postgres → transformer → JSON encoding is wired correctly
//   - filter placeholder ordering (applyFiltersToSQL before substituteFilters)
//     survives against real query results
//   - middleware chain (request-id, auth-off, metrics, recovery) is routable
package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// The fixed target report for all E2E tests: a small, stable surveillance
// report whose queries hit report.mv_communicable_diseases_district (1,008
// seeded rows in the local test DB).
const e2eReportID = "Surveillance/acute-diarrhea-summary-report"

// setupIntegrationServer wires production handlers at the root path.
// See setupIntegrationServerWithBase for details.
func setupIntegrationServer(t *testing.T) *httptest.Server {
	return setupIntegrationServerWithBase(t, "")
}

// setupIntegrationServerWithBase wires a subset of production handlers into a
// real httptest.Server backed by the test database, rooted at the given
// prefix (empty for default). Closes automatically when the test ends.
//
// Preconditions:
//   - DB is initialized (TestMain in main_test.go does this when
//     CI_INTEGRATION_DB=1 is set, reading .env).
//   - CWD is the project root on return (we chdir up from server/).
func setupIntegrationServerWithBase(t *testing.T, base string) *httptest.Server {
	t.Helper()

	if DB == nil {
		t.Fatalf("DB not initialized. Run with:\n  CI_INTEGRATION_DB=1 go test -tags=integration ./server\n" +
			"Ensure .env points at a seeded test DB (see database/testdb/README.md).")
	}

	// go test CWD is the package dir (server/). Report YAMLs live in ../configs/
	// and GetReportByID joins "configs" relative to CWD, so move up one level
	// for the duration of the test.
	t.Chdir("..")

	// Force auth off for tests even if the real .env has AUTH_MODE=on.
	// InitAuth reads from env, so t.Setenv is enough before calling it.
	t.Setenv("AUTH_MODE", "off")
	InitAuth()

	// MetricsMiddleware assumes InitMetrics has run; main_test.go does not
	// call it, so initialize here. Safe to call multiple times.
	InitMetrics()

	// The basePath package var is read by GetReportHandler to strip the
	// prefix from incoming request paths. Save & restore so tests are isolated.
	savedBase := basePath
	basePath = base
	t.Cleanup(func() { basePath = savedBase })

	mux := http.NewServeMux()
	mux.HandleFunc(base+"/api/reports", WithRequestID(AuthMiddleware(ListReportsHandler)))
	mux.HandleFunc(base+"/api/report/", WithRequestID(AuthMiddleware(
		MetricsMiddleware(GetReportHandler, extractReportIDFromRequest))))
	mux.HandleFunc(base+"/api/filters/districts", WithRequestID(AuthMiddleware(GetDistrictsHandler)))

	srv := httptest.NewServer(RecoveryMiddleware(mux))
	t.Cleanup(srv.Close)
	return srv
}

// fetchReport issues GET url and returns the decoded report body. Fails the
// test on non-200 or decode errors — callers can focus on assertions.
func fetchReport(t *testing.T, u string) decodedReport {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", u, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status = %d, body = %s", u, resp.StatusCode, string(body))
	}

	var report decodedReport
	if err := json.Unmarshal(body, &report); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, string(body))
	}
	return report
}

// resetCacheState clears the global component cache and its counters so
// hit/miss assertions start from a known zero baseline.
func resetCacheState(t *testing.T) {
	t.Helper()
	componentCache.Clear()
	componentCache.hits.Store(0)
	componentCache.misses.Store(0)
}

// decodedReport is the subset of the Report JSON we assert against. We decode
// into a local shape rather than the Report struct so the test survives
// additive changes (new fields) to the production type.
type decodedReport struct {
	Title       string `json:"title"`
	GeneratedAt string `json:"generated_at"`
	Sections    []struct {
		ID         string `json:"id"`
		Components []struct {
			Type  string          `json:"type"`
			Data  json.RawMessage `json:"data"`
			Error string          `json:"error,omitempty"`
		} `json:"components"`
	} `json:"sections"`
}

// TestE2E_ReportPipeline_Structured exercises the full request lifecycle for
// configs/Surveillance/acute-diarrhea-summary-report.yaml. It is deliberately
// chosen for breadth: 5 sections, 6 components across all three
// component-execution paths.
//
//	section 0 (table)              structured query → table transformer
//	section 1 (table)              structured query with calculate formulas
//	section 2 (bar_line × 2)       multi-dataset; periodLimit expansion
//	section 3 (choropleth)         faceted path — processFacetedComponent
//	section 4 (table_advanced)     raw SQL with {{district_filter}}
//	                               {{year_filter}} {{month_filter}} —
//	                               exercises the filter-ordering footgun
//	                               flagged in CLAUDE.md
//
// Success means every layer between HTTP and Postgres is wired correctly.
// Failures here that unit tests would miss include: broken route registration,
// filter placeholder ordering regressions, pgx bind mismatches, goroutine
// result-slot corruption, transformer errors on real row shapes, and the
// faceted-choropleth and raw-SQL choropleth paths diverging from their tests.
func TestE2E_ReportPipeline_Structured(t *testing.T) {
	srv := setupIntegrationServer(t)

	// year=2024 is the newest year in the seed; handler auto-fills month from
	// the data-aware default query against the primary table.
	report := fetchReport(t, srv.URL+"/api/report/"+e2eReportID+"?year=2024")

	if report.GeneratedAt == "" {
		t.Error("generated_at is empty — report orchestration did not complete")
	}
	if len(report.Sections) != 5 {
		t.Fatalf("expected 5 sections, got %d", len(report.Sections))
	}

	// Every component should produce a result without an error. A single
	// failure here probably means a wiring break (filter ordering, transformer
	// shape, goroutine result slot) — which is exactly what this test exists
	// to catch.
	totalComponents := 0
	for i, sec := range report.Sections {
		for j, c := range sec.Components {
			totalComponents++
			if c.Error != "" {
				t.Errorf("section %d component %d (%s): pipeline error: %s", i, j, c.Type, c.Error)
			}
			if len(c.Data) == 0 {
				t.Errorf("section %d component %d (%s): empty data field", i, j, c.Type)
			}
		}
	}
	if totalComponents != 6 {
		// 5 sections, but section 2 has two bar_line components.
		t.Errorf("expected 6 total components, got %d", totalComponents)
	}

	// --- Section 0: table from structured query ---
	tbl := report.Sections[0].Components[0]
	if tbl.Type != "table" {
		t.Errorf("section 0: type = %q, want table", tbl.Type)
	}
	var tableData struct {
		Headers []string        `json:"headers"`
		Rows    [][]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(tbl.Data, &tableData); err != nil {
		t.Fatalf("decode table data: %v\nraw: %s", err, string(tbl.Data))
	}
	if len(tableData.Headers) == 0 {
		t.Error("section 0: table headers empty — selectColumns/transformer regression")
	}
	if len(tableData.Rows) == 0 {
		t.Error("section 0: table returned 0 rows for year=2024 (seed coverage gap?)")
	}

	// --- Section 3: faceted choropleth ---
	chor := report.Sections[3].Components[0]
	if chor.Type != "choropleth" {
		t.Errorf("section 3: type = %q, want choropleth", chor.Type)
	}

	// --- Section 4: raw-SQL table_advanced ---
	// This is the highest-value assertion: if applyFiltersToSQL doesn't run
	// before substituteFilters, the {{district_filter}}, {{year_filter}}, and
	// {{month_filter}} placeholders get stripped — the query then runs with
	// no WHERE constraints, returns rows but *not the filtered ones*. That
	// specific failure mode is exercised end-to-end in TestE2E_FilterBinding.
	adv := report.Sections[4].Components[0]
	if adv.Type != "table_advanced" {
		t.Errorf("section 4: type = %q, want table_advanced", adv.Type)
	}
}

// TestE2E_FilterBinding proves that the district filter actually reaches
// Postgres as a bound parameter — not silently dropped by the
// applyFiltersToSQL/substituteFilters ordering footgun.
//
// The target is section 4 (table_advanced), whose raw SQL groups by district
// and has {{district_filter}} as the only district constraint. If filter
// binding works, rows contain only the requested district. If the
// placeholder is stripped, every seeded district comes back.
func TestE2E_FilterBinding(t *testing.T) {
	srv := setupIntegrationServer(t)

	// Bukedea District has non-zero "Acute" sums in the 2024 seed, so the
	// HAVING SUM > 0 clause in the advanced SQL will let the row through.
	const district = "Bukedea District"
	u := srv.URL + "/api/report/" + e2eReportID +
		"?year=2024&month=6&district=" + url.QueryEscape(district)

	report := fetchReport(t, u)
	if len(report.Sections) < 5 {
		t.Fatalf("expected at least 5 sections, got %d", len(report.Sections))
	}
	adv := report.Sections[4].Components[0]
	if adv.Type != "table_advanced" {
		t.Fatalf("section 4: type = %q, want table_advanced", adv.Type)
	}
	if adv.Error != "" {
		t.Fatalf("table_advanced pipeline error: %s", adv.Error)
	}

	var tbl struct {
		Headers []string        `json:"headers"`
		Rows    [][]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(adv.Data, &tbl); err != nil {
		t.Fatalf("decode table_advanced data: %v\nraw: %s", err, string(adv.Data))
	}

	districtCol := -1
	for i, h := range tbl.Headers {
		if h == "district" {
			districtCol = i
			break
		}
	}
	if districtCol == -1 {
		t.Fatalf("district column missing from headers: %v", tbl.Headers)
	}
	if len(tbl.Rows) == 0 {
		t.Fatal("no rows for filtered district — either the filter dropped all " +
			"data or seed values are zero; pick a different district")
	}

	// Every returned row must be for the requested district. More than one
	// distinct district => filter was stripped and the query ran unfiltered.
	for i, row := range tbl.Rows {
		if got, ok := row[districtCol].(string); !ok || got != district {
			t.Errorf("row %d district = %v, want %q — filter binding is broken "+
				"(applyFiltersToSQL/substituteFilters ordering regression?)",
				i, row[districtCol], district)
		}
	}
}

// TestE2E_CacheHitMiss verifies the component cache actually short-circuits
// repeat requests. Two identical GETs must produce exactly one set of DB
// queries; counters prove it.
func TestE2E_CacheHitMiss(t *testing.T) {
	srv := setupIntegrationServer(t)
	resetCacheState(t)

	// Pin every filter so no data-aware defaults muddy the cache key across
	// the two requests.
	u := srv.URL + "/api/report/" + e2eReportID + "?year=2024&month=6"

	// First request — everything is a cache miss.
	_ = fetchReport(t, u)
	miss1 := componentCache.misses.Load()
	hit1 := componentCache.hits.Load()
	if miss1 == 0 {
		t.Fatal("first request produced no cache misses — cache may already have been populated")
	}
	if hit1 != 0 {
		t.Errorf("first request produced %d hits, want 0", hit1)
	}

	// Second request — same URL, every component should hit the cache.
	_ = fetchReport(t, u)
	miss2 := componentCache.misses.Load()
	hit2 := componentCache.hits.Load()

	if got := miss2 - miss1; got != 0 {
		t.Errorf("second request added %d new misses, want 0 (cache was bypassed)", got)
	}
	if got := hit2 - hit1; got != miss1 {
		t.Errorf("second request produced %d hits, want %d (one per component)", got, miss1)
	}
}

// TestE2E_BasePath confirms the BASE_PATH subpath wiring — routes registered
// at /x/api/... are reachable and strip the prefix correctly for report ID
// resolution, while root-level paths 404.
func TestE2E_BasePath(t *testing.T) {
	const base = "/x"
	srv := setupIntegrationServerWithBase(t, base)

	// Prefixed path works.
	report := fetchReport(t, srv.URL+base+"/api/report/"+e2eReportID+"?year=2024&month=6")
	if len(report.Sections) == 0 {
		t.Fatal("prefixed request returned 0 sections")
	}

	// Unprefixed path must 404 — if it succeeds, the basePath extraction in
	// GetReportHandler isn't honoring the configured prefix.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
		srv.URL+"/api/report/"+e2eReportID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unprefixed GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unprefixed GET: status = %d, want 404", resp.StatusCode)
	}
}
