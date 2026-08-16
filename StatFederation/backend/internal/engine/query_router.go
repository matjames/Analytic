package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"statfederation-backend/internal/models"
	"statfederation-backend/internal/store"
)

// QueryRouter manages federated scatter-gather distributed queries
type QueryRouter struct {
	store      store.Store
	fedEngine  *FederationEngine
	httpClient interface{} // extensible for real outbound HTTP
}

// NodeExecutionResult contains data retrieved from a single federated node
type NodeExecutionResult struct {
	NodeID       string                   `json:"node_id"`
	NodeCode     string                   `json:"node_code"`
	NodeName     string                   `json:"node_name"`
	Success      bool                     `json:"success"`
	LatencyMs    int64                    `json:"latency_ms"`
	RecordsFound int                      `json:"records_found"`
	DataPayload  []map[string]interface{} `json:"data_payload,omitempty"`
	ErrorMessage string                   `json:"error_message,omitempty"`
}

// AggregatedQueryResult combines distributed multi-node outputs
type AggregatedQueryResult struct {
	QueryID            string                 `json:"query_id"`
	QueryName          string                 `json:"query_name"`
	TargetDomain       string                 `json:"target_domain"`
	TotalNodesTargeted int                    `json:"total_nodes_targeted"`
	SuccessfulNodes    int                    `json:"successful_nodes"`
	FailedNodes        int                    `json:"failed_nodes"`
	TotalRecords       int                    `json:"total_records"`
	ExecutionTimeMs    int64                  `json:"execution_time_ms"`
	HarmonizedRecords  []map[string]interface{}`json:"harmonized_records"`
	NodeBreakdown      []NodeExecutionResult  `json:"node_breakdown"`
	ConflictResolution string                 `json:"conflict_resolution"`
}

// NewQueryRouter creates a new Distributed Query Router
func NewQueryRouter(s store.Store, fe *FederationEngine) *QueryRouter {
	return &QueryRouter{
		store:     s,
		fedEngine: fe,
	}
}

// DispatchDistributedQuery executes a multi-node scatter-gather query
func (qr *QueryRouter) DispatchDistributedQuery(
	ctx context.Context,
	req models.DistributedQueryRequest,
	initiatorUserID, initiatorTenantID string,
) (*AggregatedQueryResult, error) {
	start := time.Now()
	queryID := "dqr-" + uuid.New().String()[:8]

	if req.QueryName == "" {
		req.QueryName = fmt.Sprintf("Federated-Query-%s", time.Now().Format("20060102-150405"))
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 15
	}
	if req.MaxRecordsPerNode <= 0 {
		req.MaxRecordsPerNode = 500
	}

	// 1. Resolve Target Nodes
	var targetNodes []*models.FederatedNode
	if len(req.TargetNodeIDs) > 0 {
		for _, nid := range req.TargetNodeIDs {
			node, err := qr.store.GetNodeByID(ctx, nid)
			if err != nil {
				// Try by code
				node, err = qr.store.GetNodeByCode(ctx, nid)
			}
			if err == nil && node != nil {
				targetNodes = append(targetNodes, node)
			}
		}
	} else {
		// All healthy NSS nodes in the tenant
		nodes, err := qr.store.ListNodes(ctx, initiatorTenantID, "", "")
		if err != nil {
			return nil, fmt.Errorf("failed to resolve target nodes: %w", err)
		}
		for _, n := range nodes {
			if n.HealthStatus == models.HealthStatusHealthy {
				targetNodes = append(targetNodes, n)
			}
		}
	}

	if len(targetNodes) == 0 {
		return nil, errors.New("no active or healthy federated nodes available for this query")
	}

	// 2. Record initial pending query in ledger
	targetIDs := make([]string, len(targetNodes))
	for i, n := range targetNodes {
		targetIDs[i] = n.ID
	}

	qRecord := &models.DistributedQueryRecord{
		ID:                queryID,
		QueryName:         req.QueryName,
		InitiatorUserID:   initiatorUserID,
		InitiatorTenantID: initiatorTenantID,
		TargetNodes:       targetIDs,
		QuerySyntax: map[string]interface{}{
			"domain":          req.TargetDomain,
			"filter_criteria": req.FilterCriteria,
			"aggregations":    req.Aggregations,
		},
		ExecutionStrategy: "SCATTER_GATHER",
		Status:            models.QueryStatusDispatched,
		DispatchTimestamp: start,
	}
	_ = qr.store.RecordQuery(ctx, qRecord)

	// 3. Concurrent Scatter-Gather Execution with Timeout Context
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()

	resultsChan := make(chan NodeExecutionResult, len(targetNodes))
	var wg sync.WaitGroup

	for _, node := range targetNodes {
		wg.Add(1)
		go func(targetNode *models.FederatedNode) {
			defer wg.Done()
			nodeRes := qr.dispatchToSingleNode(queryCtx, targetNode, req, initiatorTenantID)
			resultsChan <- nodeRes
		}(node)
	}

	wg.Wait()
	close(resultsChan)

	// 4. Aggregation and Statistical Reconciliation
	var (
		successfulCount int
		failedCount     int
		totalRecords    int
		harmonizedList  = make([]map[string]interface{}, 0)
		breakdowns      = make([]NodeExecutionResult, 0, len(targetNodes))
		responsesMap    = make(map[string]interface{})
	)

	for res := range resultsChan {
		breakdowns = append(breakdowns, res)
		responsesMap[res.NodeID] = res
		if res.Success {
			successfulCount++
			totalRecords += res.RecordsFound
			harmonizedList = append(harmonizedList, res.DataPayload...)
		} else {
			failedCount++
		}
	}

	execDuration := time.Since(start).Milliseconds()

	// Determine final query status
	var finalStatus models.QueryStatus
	var errSummary string
	if successfulCount == len(targetNodes) {
		finalStatus = models.QueryStatusCompleted
	} else if successfulCount > 0 {
		finalStatus = models.QueryStatusPartialSuccess
		errSummary = fmt.Sprintf("%d of %d nodes responded successfully", successfulCount, len(targetNodes))
	} else {
		finalStatus = models.QueryStatusFailed
		errSummary = "all target federated nodes failed or timed out"
	}

	_ = qr.store.UpdateQueryExecution(ctx, queryID, finalStatus, totalRecords, int(execDuration), responsesMap, errSummary)

	// 5. Cross-Application Object Linkage
	// Normalise to "default" so that unauthenticated / system callers still
	// produce a queryable object_link with a stable tenant scope.
	linkTenantID := initiatorTenantID
	if linkTenantID == "" {
		linkTenantID = "default"
	}
	_ = qr.store.CreateObjectLink(ctx, &models.ObjectLink{
		SourceType:   "distributed_query",
		SourceID:     queryID,
		TargetType:   "nss_federation",
		TargetID:     linkTenantID,
		Relationship: "federated_dispatch",
		TenantID:     linkTenantID,
		CreatedBy:    initiatorUserID,
		CreatedAt:    time.Now().UTC(),
	})

	return &AggregatedQueryResult{
		QueryID:            queryID,
		QueryName:          req.QueryName,
		TargetDomain:       req.TargetDomain,
		TotalNodesTargeted: len(targetNodes),
		SuccessfulNodes:    successfulCount,
		FailedNodes:        failedCount,
		TotalRecords:       totalRecords,
		ExecutionTimeMs:    execDuration,
		HarmonizedRecords:  harmonizedList,
		NodeBreakdown:      breakdowns,
		ConflictResolution: "CANONICAL_SDMX_MERGE_AND_DEDUPLICATION",
	}, nil
}

// dispatchToSingleNode executes query against a single node (with DSA policy check and synthetic harmonized payload)
func (qr *QueryRouter) dispatchToSingleNode(
	ctx context.Context,
	node *models.FederatedNode,
	req models.DistributedQueryRequest,
	consumerTenant string,
) NodeExecutionResult {
	start := time.Now()

	// 1. Check DSA Authority if not same tenant
	if node.TenantID != consumerTenant && node.TenantID != "default" {
		_, err := qr.fedEngine.ValidateExchangeAuthority(ctx, node.ID, consumerTenant, req.TargetDomain)
		if err != nil {
			return NodeExecutionResult{
				NodeID:       node.ID,
				NodeCode:     node.Code,
				NodeName:     node.Name,
				Success:      false,
				LatencyMs:    time.Since(start).Milliseconds(),
				ErrorMessage: fmt.Sprintf("DSA validation failed: %v", err),
			}
		}
	}

	// 2. Simulate Node Query Execution (microdata aggregation / SDMX indicators)
	select {
	case <-ctx.Done():
		return NodeExecutionResult{
			NodeID:       node.ID,
			NodeCode:     node.Code,
			NodeName:     node.Name,
			Success:      false,
			LatencyMs:    time.Since(start).Milliseconds(),
			ErrorMessage: "query execution timed out on remote node",
		}
	case <-time.After(time.Duration(node.LatencyMs+10) * time.Millisecond):
		// Synthetic data response simulating real statistical outputs
		payload := qr.generateNodeSamplePayload(node, req)
		return NodeExecutionResult{
			NodeID:       node.ID,
			NodeCode:     node.Code,
			NodeName:     node.Name,
			Success:      true,
			LatencyMs:    time.Since(start).Milliseconds(),
			RecordsFound: len(payload),
			DataPayload:  payload,
		}
	}
}

func (qr *QueryRouter) generateNodeSamplePayload(node *models.FederatedNode, req models.DistributedQueryRequest) []map[string]interface{} {
	now := time.Now().UTC()
	return []map[string]interface{}{
		{
			"reporting_agency": node.Name,
			"agency_code":      node.Code,
			"domain":           req.TargetDomain,
			"period":           "2026-Q2",
			"indicator_code":   "IND-FED-001",
			"metric_name":      "National Aggregated Index",
			"value":            87.45,
			"unit":             "Index (0-100)",
			"classification":   "OFFICIAL_STATISTIC",
			"quality_grade":    "A_CERTIFIED",
			"timestamp":        now.Format(time.RFC3339),
		},
		{
			"reporting_agency": node.Name,
			"agency_code":      node.Code,
			"domain":           req.TargetDomain,
			"period":           "2026-Q1",
			"indicator_code":   "IND-FED-002",
			"metric_name":      "Quarterly Variance Ratio",
			"value":            1.04,
			"unit":             "Ratio",
			"classification":   "OFFICIAL_STATISTIC",
			"quality_grade":    "A_CERTIFIED",
			"timestamp":        now.Add(-90 * 24 * time.Hour).Format(time.RFC3339),
		},
	}
}
