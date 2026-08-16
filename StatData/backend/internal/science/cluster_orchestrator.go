package science

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

// ClusterOrchestrator schedules jobs and manages node resources
type ClusterOrchestrator struct {
	store store.Store
}

// NewClusterOrchestrator creates a compute orchestrator
func NewClusterOrchestrator(s store.Store) *ClusterOrchestrator {
	return &ClusterOrchestrator{store: s}
}

// SubmitJob schedules a scientific computation job on an available cluster node
func (co *ClusterOrchestrator) SubmitJob(ctx context.Context, job *models.ComputeJob) (*models.ComputeJob, error) {
	if job.ID == "" {
		job.ID = "cjob-" + uuid.New().String()[:8]
	}

	nodes, err := co.store.ListComputeNodes(ctx, job.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check cluster nodes: %w", err)
	}

	var candidateNode *models.ComputeNode
	for _, n := range nodes {
		if n.Status == "READY" && (n.TotalCPUs-n.AllocCPUs) >= job.RequiredCPUs && (n.TotalRAMGB-n.AllocRAMGB) >= job.RequiredRAMGB {
			candidateNode = n
			break
		}
	}

	if candidateNode == nil && len(nodes) > 0 {
		// Fallback to first active node
		candidateNode = nodes[0]
	}

	if candidateNode != nil {
		job.AssignedNode = candidateNode.Hostname
		job.Status = "RUNNING"
		candidateNode.AllocCPUs += job.RequiredCPUs
		candidateNode.AllocRAMGB += job.RequiredRAMGB
		candidateNode.AllocGPUs += job.RequiredGPUs
		_ = co.store.RegisterComputeNode(ctx, candidateNode)
	} else {
		job.Status = "QUEUED"
	}

	if err := co.store.CreateComputeJob(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}

// CompleteJob marks computation finished and releases cluster resources
func (co *ClusterOrchestrator) CompleteJob(ctx context.Context, jobID string, output map[string]interface{}) (*models.ComputeJob, error) {
	job, err := co.store.GetComputeJobByID(ctx, jobID)
	if err != nil {
		return nil, errors.New("compute job not found")
	}

	now := time.Now().UTC()
	job.Status = "COMPLETED"
	job.CompletedAt = &now
	job.OutputData = output

	if err := co.store.UpdateComputeJob(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}
