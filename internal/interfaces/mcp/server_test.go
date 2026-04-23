// Package mcp_test tests the SOUL MCP controller with a real in-memory SQLite database.
package mcp_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/benoitpetit/soul/internal/app"
	mcp "github.com/benoitpetit/soul/internal/interfaces/mcp"
)

// newTestController wires a Controller backed by an in-memory SQLite database.
func newTestController(t *testing.T) *mcp.Controller {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory DB: %v", err)
	}
	a, err := app.NewSoulApplicationWithDB(db)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create SoulApplication: %v", err)
	}
	t.Cleanup(func() {
		a.Close() // no-op on DB (ownsDB=false)
		db.Close()
	})
	return mcp.NewController(a)
}

// captureIdentity is a test helper that performs a soul_capture and fails the test on error.
func captureIdentity(t *testing.T, c *mcp.Controller, agentID string) {
	t.Helper()
	_, err := c.Call(context.Background(), "soul_capture", map[string]interface{}{
		"agent_id":     agentID,
		"conversation": "I prefer clear, direct communication. I value honesty and logical reasoning. I am analytical and methodical.",
		"model_id":     "test-model",
	})
	if err != nil {
		t.Fatalf("capture failed for agent %s: %v", agentID, err)
	}
}

// ============================================================================
// ToolDefinitions
// ============================================================================

var expectedToolNames = []string{
	"soul_capture",
	"soul_recall",
	"soul_drift",
	"soul_swap",
	"soul_status",
	"soul_history",
	"soul_update",
	"soul_patch",
}

func TestToolDefinitions_Count(t *testing.T) {
	c := newTestController(t)
	tools := c.ToolDefinitions()
	if len(tools) != len(expectedToolNames) {
		t.Errorf("expected %d tools, got %d", len(expectedToolNames), len(tools))
	}
}

func TestToolDefinitions_Names(t *testing.T) {
	c := newTestController(t)
	tools := c.ToolDefinitions()
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}

	if len(names) != len(expectedToolNames) {
		t.Fatalf("expected exactly %d unique tool names, got %d", len(expectedToolNames), len(names))
	}

	for _, name := range expectedToolNames {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}

	for found := range names {
		known := false
		for _, expected := range expectedToolNames {
			if found == expected {
				known = true
				break
			}
		}
		if !known {
			t.Errorf("unexpected tool exposed: %s", found)
		}
	}
}

// ============================================================================
// Call — unknown tool
// ============================================================================

func TestCall_UnknownTool(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "unknown_tool", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}
	if !strings.Contains(err.Error(), "unknown soul tool") {
		t.Errorf("expected 'unknown soul tool' error, got: %v", err)
	}
}

// ============================================================================
// soul_capture
// ============================================================================

func TestHandleCapture_MissingAgentID(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_capture", map[string]interface{}{
		"conversation": "Some conversation",
	})
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id in error, got: %v", err)
	}
}

func TestHandleCapture_EmptyAgentID(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_capture", map[string]interface{}{
		"agent_id":     "   ",
		"conversation": "Some conversation",
	})
	if err == nil {
		t.Fatal("expected error for whitespace agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id in error, got: %v", err)
	}
}

func TestHandleCapture_MissingConversation(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_capture", map[string]interface{}{
		"agent_id": "test-agent",
	})
	if err == nil {
		t.Fatal("expected error for missing conversation")
	}
	if !strings.Contains(err.Error(), "conversation") {
		t.Errorf("expected conversation in error, got: %v", err)
	}
}

func TestHandleCapture_Success(t *testing.T) {
	c := newTestController(t)
	res, err := c.Call(context.Background(), "soul_capture", map[string]interface{}{
		"agent_id":     "capture-agent",
		"conversation": "I prefer structured, logical analysis. I value precision and clarity.",
		"model_id":     "gpt-4",
		"session_id":   "session-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected non-empty result")
	}
}

// ============================================================================
// soul_recall
// ============================================================================

func TestHandleRecall_MissingAgentID(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_recall", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id in error, got: %v", err)
	}
}

func TestHandleRecall_AfterCapture(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	captureIdentity(t, c, "recall-agent")

	res, err := c.Call(ctx, "soul_recall", map[string]interface{}{
		"agent_id": "recall-agent",
		"budget":   float64(800),
	})
	if err != nil {
		t.Fatalf("recall failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected non-empty recall result")
	}
}

func TestHandleRecall_BudgetAsInt(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	captureIdentity(t, c, "recall-int-budget-agent")

	// budget passed as int (not float64) — handler must handle both types
	res, err := c.Call(ctx, "soul_recall", map[string]interface{}{
		"agent_id": "recall-int-budget-agent",
		"budget":   500, // int, not float64
	})
	if err != nil {
		t.Fatalf("recall with int budget failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected non-empty recall result")
	}
}

// ============================================================================
// soul_drift
// ============================================================================

func TestHandleDrift_MissingAgentID(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_drift", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id in error, got: %v", err)
	}
}

func TestHandleDrift_AfterCapture(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	captureIdentity(t, c, "drift-agent")

	res, err := c.Call(ctx, "soul_drift", map[string]interface{}{
		"agent_id": "drift-agent",
		"window":   float64(5),
	})
	if err != nil {
		t.Fatalf("drift failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected non-empty drift result")
	}
}

func TestHandleDrift_WindowAsInt(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	captureIdentity(t, c, "drift-int-window-agent")

	res, err := c.Call(ctx, "soul_drift", map[string]interface{}{
		"agent_id": "drift-int-window-agent",
		"window":   10, // int, not float64
	})
	if err != nil {
		t.Fatalf("drift with int window failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected non-empty drift result")
	}
}

// ============================================================================
// soul_swap
// ============================================================================

func TestHandleSwap_MissingAgentID(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_swap", map[string]interface{}{
		"from_model": "gpt-4",
		"to_model":   "claude-3",
	})
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id in error, got: %v", err)
	}
}

func TestHandleSwap_MissingFromModel(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_swap", map[string]interface{}{
		"agent_id": "swap-agent",
		"to_model": "claude-3",
	})
	if err == nil {
		t.Fatal("expected error for missing from_model")
	}
	if !strings.Contains(err.Error(), "from_model") {
		t.Errorf("expected from_model in error, got: %v", err)
	}
}

func TestHandleSwap_MissingToModel(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_swap", map[string]interface{}{
		"agent_id":   "swap-agent",
		"from_model": "gpt-4",
	})
	if err == nil {
		t.Fatal("expected error for missing to_model")
	}
	if !strings.Contains(err.Error(), "to_model") {
		t.Errorf("expected to_model in error, got: %v", err)
	}
}

func TestHandleSwap_EmptyFromModel(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_swap", map[string]interface{}{
		"agent_id":   "swap-agent",
		"from_model": "  ",
		"to_model":   "claude-3",
	})
	if err == nil {
		t.Fatal("expected error for whitespace from_model")
	}
	if !strings.Contains(err.Error(), "from_model") {
		t.Errorf("expected from_model in error, got: %v", err)
	}
}

func TestHandleSwap_AfterCapture(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	captureIdentity(t, c, "swap-live-agent")

	res, err := c.Call(ctx, "soul_swap", map[string]interface{}{
		"agent_id":   "swap-live-agent",
		"from_model": "gpt-4",
		"to_model":   "claude-3-sonnet",
	})
	if err != nil {
		t.Fatalf("swap failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected non-empty swap result")
	}
}

// ============================================================================
// soul_status
// ============================================================================

func TestHandleStatus_MissingAgentID(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_status", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id in error, got: %v", err)
	}
}

func TestHandleStatus_AfterCapture(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	captureIdentity(t, c, "status-agent")

	res, err := c.Call(ctx, "soul_status", map[string]interface{}{
		"agent_id": "status-agent",
	})
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected non-empty status result")
	}
}

// ============================================================================
// soul_history
// ============================================================================

func TestHandleHistory_MissingAgentID(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_history", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id in error, got: %v", err)
	}
}

func TestHandleHistory_Empty(t *testing.T) {
	c := newTestController(t)
	// Agent with no data — should return a valid (empty) history, not an error
	res, err := c.Call(context.Background(), "soul_history", map[string]interface{}{
		"agent_id": "no-data-agent",
		"limit":    float64(5),
	})
	if err != nil {
		t.Fatalf("history for empty agent returned unexpected error: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected a result even for empty history")
	}
}

func TestHandleHistory_AfterCapture(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	captureIdentity(t, c, "history-agent")

	res, err := c.Call(ctx, "soul_history", map[string]interface{}{
		"agent_id": "history-agent",
		"limit":    float64(10),
	})
	if err != nil {
		t.Fatalf("history failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected non-empty history result")
	}
}

func TestHandleHistory_LimitAsInt(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	captureIdentity(t, c, "history-int-limit-agent")

	res, err := c.Call(ctx, "soul_history", map[string]interface{}{
		"agent_id": "history-int-limit-agent",
		"limit":    5, // int, not float64
	})
	if err != nil {
		t.Fatalf("history with int limit failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Error("expected non-empty history result")
	}
}

// ============================================================================
// soul_update
// ============================================================================

func TestHandleUpdate_MissingAgentID(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_update", map[string]interface{}{
		"directive": "be more enthusiastic",
	})
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id in error, got: %v", err)
	}
}

func TestHandleUpdate_MissingDirective(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_update", map[string]interface{}{
		"agent_id": "update-agent",
	})
	if err == nil {
		t.Fatal("expected error for missing directive")
	}
	if !strings.Contains(err.Error(), "directive") {
		t.Errorf("expected directive in error, got: %v", err)
	}
}

func TestHandleUpdate_AfterCapture(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	agentID := "update-live-agent"
	captureIdentity(t, c, agentID)

	res, err := c.Call(ctx, "soul_update", map[string]interface{}{
		"agent_id":  agentID,
		"directive": "be more enthusiastic",
		"reason":    "user request",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Fatal("expected non-empty update result")
	}
	text, ok := res.Content[0].(map[string]interface{})
	if ok {
		if got, _ := text["text"].(string); got != "" && !strings.Contains(got, "SOUL UPDATE") {
			t.Errorf("expected SOUL UPDATE in output, got: %s", got)
		}
	}
}

// ============================================================================
// soul_patch
// ============================================================================

func TestHandlePatch_MissingAgentID(t *testing.T) {
	c := newTestController(t)
	_, err := c.Call(context.Background(), "soul_patch", map[string]interface{}{
		"enthusiasm_level": float64(0.8),
	})
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
	if !strings.Contains(err.Error(), "agent_id") {
		t.Errorf("expected agent_id in error, got: %v", err)
	}
}

func TestHandlePatch_AfterCapture(t *testing.T) {
	c := newTestController(t)
	ctx := context.Background()
	agentID := "patch-live-agent"
	captureIdentity(t, c, agentID)

	res, err := c.Call(ctx, "soul_patch", map[string]interface{}{
		"agent_id":          agentID,
		"enthusiasm_level":  float64(0.85),
		"formality_level":   float64(0.35),
		"uses_emojis":       true,
		"response_length":   "concise",
		"structure_preference": "bulleted",
		"reason":            "style adjustment",
	})
	if err != nil {
		t.Fatalf("patch failed: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Fatal("expected non-empty patch result")
	}
}
