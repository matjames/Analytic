package science

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

// NotebookRunner executes interactive scientific computing cells
type NotebookRunner struct {
	store store.Store
}

// NewNotebookRunner creates a new notebook execution service
func NewNotebookRunner(s store.Store) *NotebookRunner {
	return &NotebookRunner{store: s}
}

// ExecuteCell processes a single notebook cell and captures outputs
func (nr *NotebookRunner) ExecuteCell(ctx context.Context, sessionID, cellID, source, language, tenantID string) (*models.NotebookCell, error) {
	nb, err := nr.store.GetNotebookSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("notebook session not found: %w", err)
	}

	start := time.Now()
	var targetCell *models.NotebookCell
	for i := range nb.Cells {
		if nb.Cells[i].ID == cellID {
			targetCell = &nb.Cells[i]
			break
		}
	}

	if targetCell == nil {
		if cellID == "" {
			cellID = "cell-" + uuid.New().String()[:8]
		}
		newCell := models.NotebookCell{
			ID:       cellID,
			CellType: "CODE",
			Source:   source,
		}
		nb.Cells = append(nb.Cells, newCell)
		targetCell = &nb.Cells[len(nb.Cells)-1]
	}

	targetCell.Source = source
	targetCell.Status = "RUNNING"
	targetCell.ExecutionCount++

	// Execute / Simulate code execution
	output, richOut, err := nr.simulateCodeExecution(source, language)
	targetCell.ExecutionTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		targetCell.Status = "ERROR"
		targetCell.Output = fmt.Sprintf("Error: %v", err)
	} else {
		targetCell.Status = "SUCCESS"
		targetCell.Output = output
		targetCell.RichOutput = richOut
	}

	nb.KernelState = "IDLE"
	_ = nr.store.UpdateNotebookSession(ctx, nb)

	return targetCell, nil
}

func (nr *NotebookRunner) simulateCodeExecution(source, language string) (string, map[string]interface{}, error) {
	s := strings.TrimSpace(source)
	rich := make(map[string]interface{})

	if strings.Contains(s, "import pandas") || strings.Contains(s, "import numpy") || strings.Contains(s, "library(") {
		return "Environment packages successfully loaded into kernel memory.", rich, nil
	}

	if strings.Contains(s, "plot(") || strings.Contains(s, "plt.show()") || strings.Contains(s, "ggplot(") {
		rich["mime_type"] = "image/png"
		rich["chart_type"] = "histogram_density"
		rich["chart_data"] = map[string]interface{}{
			"bins":   []int{10, 20, 30, 40, 50, 60, 70},
			"counts": []int{120, 450, 890, 1200, 780, 340, 95},
		}
		return "<Figure size 640x480 with 1 Axes>", rich, nil
	}

	if strings.Contains(s, "head()") || strings.Contains(s, "summary(") || strings.Contains(s, "SELECT") {
		rich["mime_type"] = "application/json"
		rich["table"] = []map[string]interface{}{
			{"district": "Central", "population": 1250000, "growth_rate": 0.024},
			{"district": "Northern", "population": 840000, "growth_rate": 0.018},
			{"district": "Western", "population": 960000, "growth_rate": 0.021},
		}
		return "DataFrame: 3 rows × 3 columns", rich, nil
	}

	return fmt.Sprintf("Executed successfully in %s runtime.", language), rich, nil
}
