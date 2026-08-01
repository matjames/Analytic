package semantic

import (
	"fmt"
	"sync"
)

type Indicator struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenant_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Formula     string   `json:"formula"` // e.g. "SUM(val) / COUNT(val)" or SQL snippet
	Unit        string   `json:"unit"`
	Aliases     []string `json:"aliases"`
}

type Registry struct {
	mu         sync.RWMutex
	indicators map[string]Indicator
	aliases    map[string]string
}

func NewRegistry() *Registry {
	r := &Registry{
		indicators: make(map[string]Indicator),
		aliases:    make(map[string]string),
	}
	// Seed standard metrics & SDG indicators
	r.Register(Indicator{
		ID:          "ind-001",
		TenantID:    "tenant-alpha",
		Name:        "Avg Response Latency",
		Description: "Calculates average API response latency in ms",
		Formula:     "AVG(latency_ms)",
		Unit:        "ms",
		Aliases:     []string{"latency", "ping", "delay"},
	})
	r.Register(Indicator{
		ID:          "ind-002",
		TenantID:    "tenant-alpha",
		Name:        "Ingestion Throughput Rate",
		Description: "Total record count processed per minute",
		Formula:     "COUNT(event_id) / 60.0",
		Unit:        "records/sec",
		Aliases:     []string{"throughput", "speed", "ingestion_rate"},
	})
	r.Register(Indicator{
		ID:          "sdg-3-health",
		TenantID:    "*",
		Name:        "SDG 3: Global Health Metric",
		Description: "Standardized SDG 3 Health Indicator across datasets",
		Formula:     "AVG(Confirmed) / NULLIF(AVG(Recovered), 0)",
		Unit:        "Ratio",
		Aliases:     []string{"health", "covid", "cases", "mortality"},
	})
	return r
}

func (r *Registry) Register(ind Indicator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.indicators[ind.ID] = ind
	for _, alias := range ind.Aliases {
		r.aliases[alias] = ind.ID
	}
}

func (r *Registry) ResolveAlias(term string) (Indicator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if id, exists := r.aliases[term]; exists {
		ind, ok := r.indicators[id]
		return ind, ok
	}

	for _, ind := range r.indicators {
		for _, alias := range ind.Aliases {
			if alias == term {
				return ind, true
			}
		}
	}
	return Indicator{}, false
}

func (r *Registry) Get(id string) (Indicator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ind, ok := r.indicators[id]
	if !ok {
		return Indicator{}, fmt.Errorf("indicator %s not found", id)
	}
	return ind, nil
}

func (r *Registry) ListByTenant(tenantID string) []Indicator {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []Indicator
	for _, ind := range r.indicators {
		if ind.TenantID == tenantID || ind.TenantID == "*" {
			list = append(list, ind)
		}
	}
	return list
}
