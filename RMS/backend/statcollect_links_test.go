package main

import "testing"

func TestStatCollectEventValueReadsTopLevelAndMetadata(t *testing.T) {
	payload := map[string]interface{}{
		"meta": map[string]interface{}{"_msh_research_id": "research-from-meta"},
	}
	if got := statCollectEventValue("", payload, "research_id", "_msh_research_id"); got != "research-from-meta" {
		t.Fatalf("research id = %q", got)
	}
	if got := statCollectEventValue("explicit", payload, "research_id"); got != "explicit" {
		t.Fatalf("explicit research id = %q", got)
	}
}

func TestStatCollectEventValueDoesNotInferWorkspace(t *testing.T) {
	payload := map[string]interface{}{"meta": map[string]interface{}{"research_id": "research-1"}}
	if got := statCollectEventValue("", payload, "workspace_id", "_msh_workspace_id"); got != "" {
		t.Fatalf("workspace id = %q, want empty", got)
	}
}
