package fieldops

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"statiot-backend/internal/models"
	"statiot-backend/internal/store"
)

type SyncEngine struct {
	store     store.Store
	publisher EventPublisher
}

type EventPublisher interface {
	PublishFieldSubmission(ctx context.Context, sub *models.MobileSubmission) error
	PublishSyncCompleted(ctx context.Context, workerID, deviceUUID string, recordsSynced int) error
	PublishConflictDetected(ctx context.Context, conflict *models.ConflictRecord) error
}

func NewSyncEngine(st store.Store, pub EventPublisher) *SyncEngine {
	return &SyncEngine{
		store:     st,
		publisher: pub,
	}
}

// HandlePushSync processes batch submissions, tasks, and breadcrumbs uploaded from offline mobile clients.
func (s *SyncEngine) HandlePushSync(ctx context.Context, req *models.SyncPushRequest) (*models.SyncPushResponse, error) {
	resp := &models.SyncPushResponse{
		TransactionID:         uuid.New().String(),
		ServerSyncTime:        time.Now().UTC(),
		AcknowledgedClientIDs: make([]string, 0),
		ConflictIDs:           make([]string, 0),
	}

	// 1. Process Form Submissions with Idempotency and Conflict Detection
	for _, sub := range req.Submissions {
		sCopy := sub
		if sCopy.WorkerID == "" {
			sCopy.WorkerID = req.WorkerID
		}
		if sCopy.TenantID == "" {
			sCopy.TenantID = "default"
		}

		// Check if submission already exists (Idempotency)
		existing, err := s.store.GetSubmissionByClientID(ctx, sCopy.ClientSubmissionID)
		if err == nil && existing != nil {
			// Already received previously; check if new version creates a conflict
			if sCopy.Version > existing.Version {
				// Resolve using Last-Write-Wins (LWW) or Field-Level Merge
				conflict := &models.ConflictRecord{
					ID:                 uuid.New().String(),
					EntityType:         "SUBMISSION",
					EntityID:           sCopy.ClientSubmissionID,
					ClientVersion:      sCopy.Version,
					ServerVersion:      existing.Version,
					ClientPayload:      sCopy.DataPayload,
					ServerPayload:      existing.DataPayload,
					ResolutionStrategy: models.StrategyLastWriteWins,
					ResolvedPayload:    sCopy.DataPayload,
					Status:             "RESOLVED_AUTO",
					TenantID:           sCopy.TenantID,
					CreatedAt:          time.Now().UTC(),
				}
				_ = s.store.RecordConflict(ctx, conflict)
				resp.ConflictsCount++
				resp.ConflictIDs = append(resp.ConflictIDs, conflict.ID)

				if s.publisher != nil {
					_ = s.publisher.PublishConflictDetected(ctx, conflict)
				}

				existing.DataPayload = sCopy.DataPayload
				existing.Version = sCopy.Version
				existing.SyncStatus = "MERGED"
				_ = s.store.InsertMobileSubmission(ctx, existing)
			}
			resp.AcknowledgedClientIDs = append(resp.AcknowledgedClientIDs, sCopy.ClientSubmissionID)
			resp.CommittedCount++
			continue
		}

		// Fresh Submission Insertion
		sCopy.SyncStatus = "COMMITTED"
		if err := s.store.InsertMobileSubmission(ctx, &sCopy); err != nil {
			log.Printf("[SyncEngine] Failed to commit submission %s: %v", sCopy.ClientSubmissionID, err)
			continue
		}

		resp.AcknowledgedClientIDs = append(resp.AcknowledgedClientIDs, sCopy.ClientSubmissionID)
		resp.CommittedCount++

		if s.publisher != nil {
			_ = s.publisher.PublishFieldSubmission(ctx, &sCopy)
		}
	}

	// 2. Process GPS Breadcrumbs
	if len(req.Breadcrumbs) > 0 {
		for i := range req.Breadcrumbs {
			if req.Breadcrumbs[i].WorkerID == "" {
				req.Breadcrumbs[i].WorkerID = req.WorkerID
			}
			if req.Breadcrumbs[i].DeviceUUID == "" {
				req.Breadcrumbs[i].DeviceUUID = req.DeviceUUID
			}
		}
		_ = s.store.InsertGPSBreadcrumbs(ctx, req.Breadcrumbs)
		resp.CommittedCount += len(req.Breadcrumbs)
	}

	// 3. Process Visit Tasks Updates
	for _, task := range req.UpdatedTasks {
		tCopy := task
		_ = s.store.UpdateVisitTask(ctx, &tCopy)
		resp.CommittedCount++
	}

	// 4. Record Sync Transaction Log
	_ = s.store.RecordSyncTransaction(ctx, resp, req)

	// 5. Publish Event Bus Notification
	if s.publisher != nil {
		_ = s.publisher.PublishSyncCompleted(ctx, req.WorkerID, req.DeviceUUID, resp.CommittedCount)
	}

	log.Printf("[SyncEngine] Push sync complete for worker %s (device %s): %d committed, %d conflicts",
		req.WorkerID, req.DeviceUUID, resp.CommittedCount, resp.ConflictsCount)
	return resp, nil
}

// HandlePullSync delivers updated forms, assigned visits, and tasks to mobile clients since last sync.
func (s *SyncEngine) HandlePullSync(ctx context.Context, req *models.SyncPullRequest) (*models.SyncPullResponse, error) {
	forms, err := s.store.ListFormDefinitions(ctx, "default")
	if err != nil {
		forms = []models.MobileFormDefinition{}
	}

	visits, err := s.store.ListVisitsByWorker(ctx, req.WorkerID)
	if err != nil {
		visits = []models.FieldVisit{}
	}

	var allTasks []models.VisitTask
	for _, v := range visits {
		tasks, _ := s.store.ListTasksByVisit(ctx, v.ID)
		allTasks = append(allTasks, tasks...)
	}

	resp := &models.SyncPullResponse{
		Forms:          forms,
		AssignedVisits: visits,
		Tasks:          allTasks,
		ServerTime:     time.Now().UTC(),
	}

	return resp, nil
}

// ResolveConflict applies a chosen strategy (LWW, Field-Level Merge, Manual) to resolve divergent state.
func (s *SyncEngine) ResolveConflict(ctx context.Context, conflictID string, strategy models.ConflictResolutionStrategy, resolvedBy string) (*models.ConflictRecord, error) {
	c, err := s.store.GetConflict(ctx, conflictID)
	if err != nil {
		return nil, fmt.Errorf("conflict not found: %w", err)
	}

	var finalPayload map[string]interface{}

	switch strategy {
	case models.StrategyLastWriteWins:
		// Client payload overrides server payload
		finalPayload = c.ClientPayload
	case models.StrategyFieldLevelMerge:
		// Merge top-level keys: server baseline overlaid with client updates
		finalPayload = make(map[string]interface{})
		for k, v := range c.ServerPayload {
			finalPayload[k] = v
		}
		for k, v := range c.ClientPayload {
			finalPayload[k] = v
		}
	case models.StrategyManualReview:
		// Keep pending manual resolution
		return c, nil
	default:
		finalPayload = c.ClientPayload
	}

	if err := s.store.ResolveConflict(ctx, conflictID, finalPayload, strategy, resolvedBy); err != nil {
		return nil, err
	}

	c.ResolvedPayload = finalPayload
	c.ResolutionStrategy = strategy
	c.Status = "RESOLVED_MANUAL"
	c.ResolvedBy = &resolvedBy
	now := time.Now().UTC()
	c.ResolvedAt = &now

	return c, nil
}
