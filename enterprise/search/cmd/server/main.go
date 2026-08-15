package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// SearchResult represents a unified search result from any application
type SearchResult struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Source      string `json:"source"`
	URL         string `json:"url"`
	Meta        string `json:"meta,omitempty"`
	Score       int    `json:"score"`
}

// ServiceConfig defines the endpoints for each application
type ServiceConfig struct {
	Name    string
	BaseURL string
	Search  string
	Timeout time.Duration
}

func main() {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load(".env")

	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3006",
			"http://localhost:3010",
			"http://localhost:3011",
			"http://localhost:3009",
			"http://localhost:3005",
			"http://localhost:3007",
			"http://localhost:5000",
			"http://host.docker.internal:3006",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "statgate-enterprise-search"})
	})

	// Unified search endpoint
	r.GET("/api/search", handleUnifiedSearch)

	// Enterprise dashboard endpoints
	r.GET("/api/dashboard/executive", handleExecutiveDashboard)
	r.GET("/api/dashboard/projects", handleProjectsDashboard)
	r.GET("/api/dashboard/research", handleResearchDashboard)
	r.GET("/api/dashboard/operations", handleOperationsDashboard)

	// Cross-application activity timeline
	r.GET("/api/activity", handleActivityTimeline)

	// Enterprise notifications aggregation
	r.GET("/api/notifications", handleNotifications)

	port := getEnv("ENTERPRISE_SEARCH_PORT", "8095")
	log.Printf("Starting StatGate Enterprise Search Service on :%s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getServiceConfigs returns the list of all application services to search
func getServiceConfigs() []ServiceConfig {
	return []ServiceConfig{
		{
			Name:    "pms",
			BaseURL: getEnv("PMS_API_URL", "http://localhost:8091"),
			Search:  "/api/search",
			Timeout: 3 * time.Second,
		},
		{
			Name:    "rms",
			BaseURL: getEnv("RMS_API_URL", "http://localhost:8092"),
			Search:  "/api/search",
			Timeout: 3 * time.Second,
		},
		{
			Name:    "registry",
			BaseURL: getEnv("REGISTRY_API_URL", "http://localhost:9090"),
			Search:  "/api/search",
			Timeout: 3 * time.Second,
		},
		{
			Name:    "statchat",
			BaseURL: getEnv("STATCHAT_API_URL", "http://localhost:4000"),
			Search:  "/v1/search",
			Timeout: 3 * time.Second,
		},
		{
			Name:    "helpdesk",
			BaseURL: getEnv("HELPDESK_API_URL", "http://localhost:5006"),
			Search:  "/api/search",
			Timeout: 3 * time.Second,
		},
		{
			Name:    "statgovernance",
			BaseURL: getEnv("STATGOVERNANCE_API_URL", "http://localhost:8093"),
			Search:  "/api/search",
			Timeout: 3 * time.Second,
		},
		{
			Name:    "enterprise_core",
			BaseURL: getEnv("ENTERPRISE_CORE_URL", "http://localhost:8096"),
			Search:  "/api/knowledge/search",
			Timeout: 3 * time.Second,
		},
	}
}

func handleUnifiedSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(400, gin.H{"error": "Search query required"})
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	services := getServiceConfigs()
	results := make([]SearchResult, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, svc := range services {
		wg.Add(1)
		go func(svc ServiceConfig) {
			defer wg.Done()
			svcResults := searchService(svc, query, limit)
			mu.Lock()
			results = append(results, svcResults...)
			mu.Unlock()
		}(svc)
	}

	wg.Wait()

	// Sort by score (higher first)
	// Simple sort - results from more relevant sources first
	// In production, this would use proper relevance scoring
	sortResults(results)

	// Limit total results
	if len(results) > limit*len(services) {
		results = results[:limit*len(services)]
	}

	c.JSON(200, gin.H{
		"query":   query,
		"count":   len(results),
		"results": results,
	})
}

func searchService(svc ServiceConfig, query string, limit int) []SearchResult {
	client := &http.Client{Timeout: svc.Timeout}

	searchURL := fmt.Sprintf("%s%s?q=%s&limit=%d",
		svc.BaseURL, svc.Search, url.QueryEscape(query), limit)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil
	}

	// Forward auth token if present
	if token := os.Getenv("STATGATE_REGISTRY_JWT_SECRET"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Search %s failed: %v", svc.Name, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	// Try to parse as array of results
	var rawResults []map[string]interface{}
	if err := json.Unmarshal(body, &rawResults); err != nil {
		return nil
	}

	results := make([]SearchResult, 0, len(rawResults))
	for _, raw := range rawResults {
		result := SearchResult{
			Source: svc.Name,
		}
		if v, ok := raw["type"].(string); ok {
			result.Type = v
		}
		if v, ok := raw["id"].(string); ok {
			result.ID = v
		}
		if v, ok := raw["title"].(string); ok {
			result.Title = v
		} else if v, ok := raw["name"].(string); ok {
			result.Title = v
		}
		if v, ok := raw["description"].(string); ok {
			result.Description = v
		} else if v, ok := raw["desc"].(string); ok {
			result.Description = v
		}
		if v, ok := raw["meta"].(string); ok {
			result.Meta = v
		}
		result.URL = buildResultURL(svc.Name, result.Type, result.ID)
		result.Score = calculateScore(result, query)
		results = append(results, result)
	}

	return results
}

func buildResultURL(source, resultType, id string) string {
	switch source {
	case "pms":
		return fmt.Sprintf("%s/projects/%s", getEnv("PMS_UI_URL", "http://localhost:3010"), id)
	case "rms":
		return fmt.Sprintf("%s/research/%s", getEnv("RMS_UI_URL", "http://localhost:3011"), id)
	case "registry":
		return fmt.Sprintf("%s/facilities/%s", getEnv("REGISTRY_UI_URL", "http://localhost:3007"), id)
	case "statchat":
		return fmt.Sprintf("%s/chats/%s", getEnv("STATCHAT_UI_URL", "http://localhost:3009"), id)
	case "helpdesk":
		return fmt.Sprintf("%s/tickets/%s", getEnv("HELPDESK_UI_URL", "http://localhost:3005"), id)
	default:
		return ""
	}
}

func calculateScore(result SearchResult, query string) int {
	score := 0
	lowerQuery := strings.ToLower(query)

	if strings.Contains(strings.ToLower(result.Title), lowerQuery) {
		score += 10
	}
	if strings.Contains(strings.ToLower(result.Description), lowerQuery) {
		score += 5
	}
	if strings.Contains(strings.ToLower(result.Meta), lowerQuery) {
		score += 3
	}
	if strings.EqualFold(result.Title, query) {
		score += 20
	}

	return score
}

func sortResults(results []SearchResult) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// ─── Enterprise Dashboards ───────────────────────────────────────

func handleExecutiveDashboard(c *gin.Context) {
	// Aggregate data from all services
	type DashboardData struct {
		Projects  map[string]interface{} `json:"projects"`
		Research  map[string]interface{} `json:"research"`
		Registry  map[string]interface{} `json:"registry"`
		Helpdesk  map[string]interface{} `json:"helpdesk"`
		StatChat  map[string]interface{} `json:"statchat"`
		Timestamp string                 `json:"timestamp"`
	}

	data := DashboardData{
		Projects:  fetchJSON(getEnv("PMS_API_URL", "http://localhost:8091") + "/api/dashboard"),
		Research:  fetchJSON(getEnv("RMS_API_URL", "http://localhost:8092") + "/api/dashboard"),
		Registry:  fetchJSON(getEnv("REGISTRY_API_URL", "http://localhost:9090") + "/api/dashboard"),
		Helpdesk:  fetchJSON(getEnv("HELPDESK_API_URL", "http://localhost:5006") + "/api/dashboard"),
		StatChat:  fetchJSON(getEnv("STATCHAT_API_URL", "http://localhost:4000") + "/api/dashboard"),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(200, data)
}

func handleProjectsDashboard(c *gin.Context) {
	c.JSON(200, fetchJSON(getEnv("PMS_API_URL", "http://localhost:8091")+"/api/dashboard"))
}

func handleResearchDashboard(c *gin.Context) {
	c.JSON(200, fetchJSON(getEnv("RMS_API_URL", "http://localhost:8092")+"/api/dashboard"))
}

func handleOperationsDashboard(c *gin.Context) {
	// Aggregate registry + helpdesk
	c.JSON(200, gin.H{
		"registry": fetchJSON(getEnv("REGISTRY_API_URL", "http://localhost:9090") + "/api/dashboard"),
		"helpdesk": fetchJSON(getEnv("HELPDESK_API_URL", "http://localhost:5006") + "/api/dashboard"),
	})
}

func fetchJSON(url string) map[string]interface{} {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return data
}

// ─── Activity Timeline ───────────────────────────────────────────

func handleActivityTimeline(c *gin.Context) {
	// Aggregate recent activity from all services
	type ActivityItem struct {
		Source    string `json:"source"`
		Type      string `json:"type"`
		Title     string `json:"title"`
		Timestamp string `json:"timestamp"`
		User      string `json:"user"`
	}

	// Fetch from PMS audit logs
	pmsActivity := fetchActivity(getEnv("PMS_API_URL", "http://localhost:8091")+"/api/activity", "pms")
	rmsActivity := fetchActivity(getEnv("RMS_API_URL", "http://localhost:8092")+"/api/activity", "rms")

	all := append(pmsActivity, rmsActivity...)

	// Sort by timestamp descending
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			tsI, _ := all[i]["timestamp"].(string)
			tsJ, _ := all[j]["timestamp"].(string)
			if tsJ > tsI {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	if len(all) > 50 {
		all = all[:50]
	}

	c.JSON(200, all)
}

func fetchActivity(url, source string) []map[string]interface{} {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var items []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil
	}

	for _, item := range items {
		item["source"] = source
	}
	return items
}

// ─── Notifications ───────────────────────────────────────────────

func handleNotifications(c *gin.Context) {
	// Aggregate notifications from StatChat (which already receives events from all apps)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(getEnv("STATCHAT_API_URL", "http://localhost:4000") + "/v1/notifications")
	if err != nil {
		c.JSON(200, gin.H{"notifications": []interface{}{}, "error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var notifications []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&notifications); err != nil {
		c.JSON(200, gin.H{"notifications": []interface{}{}})
		return
	}

	c.JSON(200, gin.H{"notifications": notifications})
}
