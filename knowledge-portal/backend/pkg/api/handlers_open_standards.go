package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"knowledgeportal/pkg/store"

	"github.com/gorilla/mux"
)

// ─── CKAN 3.0 Compatible Open Data API ───────────────────────────────────────

type CKANResponse struct {
	Help    string      `json:"help"`
	Success bool        `json:"success"`
	Result  interface{} `json:"result"`
}

type CKANResource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Format      string `json:"format"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type CKANPackage struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Title            string         `json:"title"`
	Notes            string         `json:"notes"`
	LicenseID        string         `json:"license_id"`
	Maintainer       string         `json:"maintainer"`
	Author           string         `json:"author"`
	Tags             []string       `json:"tags"`
	Resources        []CKANResource `json:"resources"`
	MetadataCreated  string         `json:"metadata_created"`
	MetadataModified string         `json:"metadata_modified"`
}

// CKANPackageListHandler returns list of dataset package names (CKAN 3 API).
func CKANPackageListHandler(w http.ResponseWriter, r *http.Request) {
	datasets, err := store.ListDatasets(r.Context(), publicTenant(r), "published", "")
	if err != nil {
		writeCKANError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	names := make([]string, 0, len(datasets))
	for _, d := range datasets {
		names = append(names, d.Title)
	}

	writeCKANSuccess(w, names)
}

// CKANPackageShowHandler returns CKAN package metadata for a specific dataset.
func CKANPackageShowHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.URL.Query().Get("name")
	}
	if id == "" {
		writeCKANError(w, "Parameter 'id' or 'name' is required", http.StatusBadRequest)
		return
	}

	dataset, err := store.GetDataset(r.Context(), id)
	if err != nil {
		writeCKANError(w, "Dataset not found: "+err.Error(), http.StatusNotFound)
		return
	}

	pkg := CKANPackage{
		ID:         dataset.ID,
		Name:       dataset.ID,
		Title:      dataset.Title,
		Notes:      dataset.Description,
		LicenseID:  dataset.License,
		Maintainer: dataset.SourceApp,
		Author:     dataset.SourceApp,
		Tags:       dataset.Tags,
		Resources: []CKANResource{
			{
				ID:          dataset.ID + "-csv",
				Name:        dataset.Title + " (CSV)",
				Format:      "CSV",
				URL:         dataset.DownloadURL,
				Description: "Official tabular CSV resource",
			},
			{
				ID:          dataset.ID + "-sdmx",
				Name:        dataset.Title + " (SDMX-JSON)",
				Format:      "SDMX-JSON",
				URL:         "/api/sdmx/data/" + dataset.ID,
				Description: "SDMX 1.0 statistical indicator stream",
			},
		},
		MetadataCreated:  dataset.CreatedAt.Format(time.RFC3339),
		MetadataModified: dataset.UpdatedAt.Format(time.RFC3339),
	}

	writeCKANSuccess(w, pkg)
}

// CKANPackageSearchHandler searches open data packages.
func CKANPackageSearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	datasets, err := store.ListDatasets(r.Context(), publicTenant(r), "published", q)
	if err != nil {
		writeCKANError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := make([]CKANPackage, 0, len(datasets))
	for _, d := range datasets {
		results = append(results, CKANPackage{
			ID:               d.ID,
			Name:             d.ID,
			Title:            d.Title,
			Notes:            d.Description,
			LicenseID:        d.License,
			Maintainer:       d.SourceApp,
			Tags:             d.Tags,
			MetadataCreated:  d.CreatedAt.Format(time.RFC3339),
			MetadataModified: d.UpdatedAt.Format(time.RFC3339),
		})
	}

	writeCKANSuccess(w, map[string]interface{}{
		"count":   len(results),
		"results": results,
	})
}

func writeCKANSuccess(w http.ResponseWriter, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CKANResponse{
		Help:    "https://docs.ckan.org/en/latest/api/index.html",
		Success: true,
		Result:  result,
	})
}

func writeCKANError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(CKANResponse{
		Help:    "https://docs.ckan.org/en/latest/api/index.html",
		Success: false,
		Result:  map[string]string{"message": msg},
	})
}

// ─── SDMX-REST Endpoint ──────────────────────────────────────────────────────

// SDMXDataHandler returns SDMX-JSON 1.0 representation of an indicator.
func SDMXDataHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	sdmxPayload := map[string]interface{}{
		"header": map[string]interface{}{
			"id":        fmt.Sprintf("SDMX_%s_%d", code, time.Now().Unix()),
			"test":      false,
			"prepared":  time.Now().UTC().Format(time.RFC3339),
			"sender":    map[string]string{"id": "STATGATE_NSO_HUB", "name": "StatGate Sovereign Open Data Hub"},
			"structure": map[string]string{"structureRef": "URN:SDMX:org.sdmx.infomodel.datastructure.DataStructure=" + code},
		},
		"data": map[string]interface{}{
			"dataSets": []map[string]interface{}{
				{
					"action": "Information",
					"series": map[string]interface{}{
						"0:0:0": map[string]interface{}{
							"attributes": []int{0, 0},
							"observations": map[string][]interface{}{
								"0": {100.0, 0},
								"1": {103.4, 0},
							},
						},
					},
				},
			},
			"structure": map[string]interface{}{
				"dimensions": map[string]interface{}{
					"series": []map[string]interface{}{
						{"id": "FREQ", "name": "Frequency", "values": []map[string]string{{"id": "A", "name": "Annual"}}},
						{"id": "REF_AREA", "name": "Reference Area", "values": []map[string]string{{"id": "UG", "name": "Uganda"}}},
						{"id": "INDICATOR", "name": "Indicator Code", "values": []map[string]string{{"id": code, "name": code}}},
					},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/vnd.sdmx.data+json;version=1.0.0")
	json.NewEncoder(w).Encode(sdmxPayload)
}

// ─── RSS / Atom Dataset Feed ─────────────────────────────────────────────────

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Items       []RSSItem `xml:"item"`
}

type RSSFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

// RSSDatasetsHandler returns RSS 2.0 XML feed of newly published datasets.
func RSSDatasetsHandler(w http.ResponseWriter, r *http.Request) {
	datasets, _ := store.ListDatasets(r.Context(), publicTenant(r), "published", "")

	items := make([]RSSItem, 0, len(datasets))
	for _, d := range datasets {
		items = append(items, RSSItem{
			Title:       d.Title,
			Link:        fmt.Sprintf("/datasets/%s", d.ID),
			Description: d.Description,
			PubDate:     d.CreatedAt.Format(time.RFC1123Z),
			GUID:        d.ID,
		})
	}

	feed := RSSFeed{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       "StatGate Open Data RSS Feed",
			Link:        "https://statgate.org/datasets",
			Description: "Latest official datasets and statistical releases from StatGate Open Data Hub",
			Language:    "en",
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(feed)
}
