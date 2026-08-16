package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/matjames/statgate-lib/events"
	"statdata-backend/internal/models"
	"statdata-backend/internal/pipeline"
	"statdata-backend/internal/search"
	"statdata-backend/internal/store"
)

// EventWorker consumes events from statgate:events and triggers automated platform actions
type EventWorker struct {
	bus      *events.EventBus
	store    store.Store
	pipeline *pipeline.PipelineEngine
	lineage  *pipeline.LineageTracker
	indexer  *search.UniversalIndexer
	nodeID   string
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewEventWorker creates a background event processor
func NewEventWorker(
	bus *events.EventBus,
	s store.Store,
	pe *pipeline.PipelineEngine,
	lt *pipeline.LineageTracker,
	idx *search.UniversalIndexer,
	nodeID string,
) *EventWorker {
	return &EventWorker{
		bus:      bus,
		store:    s,
		pipeline: pe,
		lineage:  lt,
		indexer:  idx,
		nodeID:   nodeID,
	}
}

// StartEventListener starts listening to statgate:events pub/sub stream
func (ew *EventWorker) StartEventListener(ctx context.Context) {
	workerCtx, cancel := context.WithCancel(ctx)
	ew.cancel = cancel

	if ew.bus == nil {
		log.Println("[EventWorker] EventBus is nil (mock mode), event listening skipped.")
		return
	}

	ew.wg.Add(1)
	go func() {
		defer ew.wg.Done()
		log.Println("[EventWorker] Started listening on Redis channel 'statgate:events'...")

		handler := func(ctx context.Context, evt events.EnterpriseEvent) error {
			ew.processDomainEvent(ctx, evt)
			return nil
		}

		_ = ew.bus.Subscribe(workerCtx, handler)
		<-workerCtx.Done()
		log.Println("[EventWorker] Event listener stopped.")
	}()
}

// Stop gracefully terminates the event worker
func (ew *EventWorker) Stop() {
	if ew.cancel != nil {
		ew.cancel()
	}
	ew.wg.Wait()
}

func (ew *EventWorker) processDomainEvent(ctx context.Context, evt events.EnterpriseEvent) {
	log.Printf("[EventWorker] Received event '%s' from source '%s' (Tenant: %s)", evt.EventType, evt.Source, evt.TenantID)

	tenantID := evt.TenantID
	if tenantID == "" {
		tenantID = "default"
	}

	switch evt.EventType {
	case "cdc.table.mutation", "cdc.event":
		// Handle CDC record
		var cdc models.CDCEvent
		payloadBytes, _ := json.Marshal(evt.Payload)
		_ = json.Unmarshal(payloadBytes, &cdc)
		cdc.TenantID = tenantID
		_ = ew.pipeline.ProcessCDCEvent(ctx, &cdc)

	case "dataset.created", "dataset.updated":
		// Auto-index into universal search and link
		var ds models.Dataset
		payloadBytes, _ := json.Marshal(evt.Payload)
		if err := json.Unmarshal(payloadBytes, &ds); err == nil && ds.ID != "" {
			_ = ew.indexer.IndexDataset(ctx, &ds)
			// Register object link
			_ = ew.store.CreateObjectLink(ctx, &models.ObjectLink{
				SourceType:   "DATASET",
				SourceID:     ds.ID,
				TargetType:   "DOMAIN",
				TargetID:     ds.Domain,
				RelationType: "BELONGS_TO_DOMAIN",
				TenantID:     tenantID,
				Metadata: map[string]interface{}{
					"auto_linked": true,
					"event_type":  evt.EventType,
				},
				CreatedAt: time.Now().UTC(),
			})
		}

	case "model.registered":
		// Auto-index ML Model into search
		var mod models.RegisteredModel
		payloadBytes, _ := json.Marshal(evt.Payload)
		if err := json.Unmarshal(payloadBytes, &mod); err == nil && mod.ID != "" {
			_ = ew.indexer.IndexModel(ctx, &mod)
			_ = ew.store.CreateObjectLink(ctx, &models.ObjectLink{
				SourceType:   "ML_MODEL",
				SourceID:     mod.ID,
				TargetType:   "DOMAIN",
				TargetID:     mod.Domain,
				RelationType: "SERVES_DOMAIN",
				TenantID:     tenantID,
				CreatedAt:    time.Now().UTC(),
			})
		}

	case "pipeline.completed":
		// Automatically update lineage edge
		if pipeID, ok := evt.Payload["pipeline_id"].(string); ok && pipeID != "" {
			if pipe, err := ew.store.GetPipelineByID(ctx, pipeID); err == nil {
				runID, _ := evt.Payload["run_id"].(string)
				_ = ew.lineage.RecordPipelineExecutionLineage(ctx, pipe, runID)
			}
		}

	default:
		// Generic auto-index
		if title, ok := evt.Payload["title"].(string); ok && title != "" {
			resID := fmt.Sprintf("%v", evt.Payload["id"])
			content, _ := evt.Payload["content"].(string)
			domain, _ := evt.Payload["domain"].(string)
			_ = ew.indexer.IndexGeneric(ctx, resID, evt.EventType, title, content, domain, "INTERNAL", tenantID, []string{evt.EventType})
		}
	}
}
