// Package mcp provides the Model Context Protocol interface adapter for SOUL.
// Implements the same stdio JSON-RPC pattern as MIRA (mark3labs/mcp-go v0.2.0).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/benoitpetit/soul/internal/app"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	mcptypes "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Controller handles MCP tool calls for SOUL.
type Controller struct {
	app *app.SoulApplication
}

// NewController creates a new MCP controller wrapping the SOUL application.
func NewController(a *app.SoulApplication) *Controller {
	return &Controller{app: a}
}

// RegisterTools registers all 8 SOUL MCP tools on the given server.
func (c *Controller) RegisterTools(mcpServer server.MCPServer) {
	tools := c.ToolDefinitions()
	mcpServer.HandleListTools(func(ctx context.Context, cursor *string) (*mcptypes.ListToolsResult, error) {
		return &mcptypes.ListToolsResult{Tools: tools}, nil
	})
	mcpServer.HandleCallTool(func(ctx context.Context, name string, arguments map[string]interface{}) (*mcptypes.CallToolResult, error) {
		return c.Call(ctx, name, arguments)
	})
}

// ToolDefinitions returns the 8 SOUL tool definitions.
// Used for combined registration with another MCP server (e.g., MIRA).
func (c *Controller) ToolDefinitions() []mcptypes.Tool {
	return []mcptypes.Tool{
		{
			Name: "soul_capture",
			Description: `Capture and persist an agent's identity from a conversation.

Analyzes the conversation to extract personality traits, voice profile, communication style,
value system, emotional tone, and behavioral signature. The result is stored as a versioned
identity snapshot linked to the agent.

Parameters:
  - agent_id:    Agent identifier (required)
  - conversation: Raw conversation text to analyze (required)
  - model_id:    Model used for this conversation (optional, default: "unknown")
  - session_id:  Session identifier for grouping (optional)

Example:
  {"agent_id": "mira-agent", "conversation": "...", "model_id": "claude-3-sonnet"}`,
			InputSchema: mcptypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_id":           map[string]string{"type": "string", "description": "Agent identifier"},
					"conversation":       map[string]string{"type": "string", "description": "Raw conversation text to analyze"},
					"model_id":           map[string]string{"type": "string", "description": "Model identifier (optional)"},
					"session_id":         map[string]string{"type": "string", "description": "Session identifier (optional)"},
					"behavioral_metrics": map[string]string{"type": "string", "description": "Optional JSON with pre-computed behavioral metrics"},
				},
			},
		},
		{
			Name: "soul_recall",
			Description: `Retrieve the identity context prompt for an agent, ready for LLM injection.

Composes an identity prompt from the agent's latest snapshot including personality traits,
voice profile, communication style, and value system. The prompt fits within the specified
token budget. Optionally enriched with relevant MIRA memories.

Parameters:
  - agent_id: Agent identifier (required)
  - context:  Current conversation context for relevance scoring (optional)
  - budget:   Max tokens for identity prompt (optional, default: 1000)

Example:
  {"agent_id": "mira-agent", "budget": 800}`,
			InputSchema: mcptypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_id": map[string]string{"type": "string", "description": "Agent identifier"},
					"context":  map[string]string{"type": "string", "description": "Current conversation context (optional)"},
					"budget":   map[string]string{"type": "number", "description": "Token budget (default: 1000)"},
				},
			},
		},
		{
			Name: "soul_drift",
			Description: `Analyze identity drift for an agent over recent versions.

Computes a drift score comparing the current identity snapshot against previous ones.
Returns whether drift is significant and recommendations for reinforcement if needed.

Parameters:
  - agent_id: Agent identifier (required)
  - window:   Number of recent versions to analyze (optional, default: 10)

Example:
  {"agent_id": "mira-agent", "window": 5}`,
			InputSchema: mcptypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_id": map[string]string{"type": "string", "description": "Agent identifier"},
					"window":   map[string]string{"type": "number", "description": "Number of versions to analyze (default: 10)"},
				},
			},
		},
		{
			Name: "soul_swap",
			Description: `Handle a model swap for an agent, preserving identity continuity.

Records the model transition, measures identity drift post-swap, and generates a
reinforcement prompt to inject into the new model's context so it adopts the
established identity of the previous model.

Parameters:
  - agent_id:    Agent identifier (required)
  - from_model:  Previous model identifier (required)
  - to_model:    New model identifier (required)

Example:
  {"agent_id": "mira-agent", "from_model": "gpt-4", "to_model": "claude-3-sonnet"}`,
			InputSchema: mcptypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_id":   map[string]string{"type": "string", "description": "Agent identifier"},
					"from_model": map[string]string{"type": "string", "description": "Previous model identifier"},
					"to_model":   map[string]string{"type": "string", "description": "New model identifier"},
				},
			},
		},
		{
			Name: "soul_status",
			Description: `Get the current identity status and summary for an agent.

Returns a human-readable summary of the agent's identity including:
- Current version and confidence score
- Key personality traits
- Voice profile and communication style
- Recent drift information

Parameters:
  - agent_id: Agent identifier (required)

Example:
  {"agent_id": "mira-agent"}`,
			InputSchema: mcptypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_id": map[string]string{"type": "string", "description": "Agent identifier"},
				},
			},
		},
		{
			Name: "soul_history",
			Description: `Retrieve the identity evolution history for an agent.

Returns a chronological list of identity snapshots with version numbers, confidence
scores, model identifiers, and timestamps. Useful for auditing how an agent's
identity has evolved over time.

Parameters:
  - agent_id: Agent identifier (required)
  - limit:    Number of snapshots to return (optional, default: 10)

Example:
  {"agent_id": "mira-agent", "limit": 5}`,
			InputSchema: mcptypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_id": map[string]string{"type": "string", "description": "Agent identifier"},
					"limit":    map[string]string{"type": "number", "description": "Max snapshots to return (default: 10)"},
				},
			},
		},
		{
			Name: "soul_update",
			Description: `Update an agent's identity from a natural language directive (FR/EN).

Parses the directive heuristically and creates a new versioned identity snapshot
derived from the current one. Never modifies existing snapshots (immutability preserved).

Supported directives include (examples):
  - "réponds avec plus d'enthousiasme" / "be more enthusiastic"
  - "sois plus formel" / "be more formal"
  - "utilise de l'humour" / "use more humor"
  - "réponds de manière concise" / "be concise"
  - "fais un rapport à la fin" / "add a summary at the end"
  - "sois plus technique" / "be more technical"
  - "vulgarise, rends accessible" / "simplify, make it accessible"
  - "sois créatif" / "be creative"
  - "utilise des emojis" / "use emojis"
  - "utilise des listes" / "use bullet lists"
  - "sois positif et encourageant" / "be positive and encouraging"

Parameters:
  - agent_id:  Agent identifier (required)
  - directive: Natural language instruction (required, FR or EN)
  - reason:    Human-readable reason for the change, stored in history (optional)

Example:
  {"agent_id": "mira-agent", "directive": "réponds avec plus d'enthousiasme"}`,
			InputSchema: mcptypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_id":  map[string]string{"type": "string", "description": "Agent identifier"},
					"directive": map[string]string{"type": "string", "description": "Natural language instruction (FR or EN)"},
					"reason":    map[string]string{"type": "string", "description": "Reason for the change (optional, stored in history)"},
				},
			},
		},
		{
			Name: "soul_patch",
			Description: `Apply a structured explicit patch to an agent's identity.

Creates a new versioned identity snapshot with the specified fields overridden.
Only non-null fields are applied; all others are inherited from the current snapshot.
Never modifies existing snapshots (immutability preserved).

Numeric fields accept values between 0.0 and 1.0.

Parameters:
  - agent_id:              Agent identifier (required)
  - reason:                Reason stored in history (optional)
  - enthusiasm_level:      Voice enthusiasm 0=measured, 1=very enthusiastic (optional)
  - formality_level:       Voice formality 0=casual, 1=very formal (optional)
  - humor_level:           Voice humor 0=serious, 1=very humorous (optional)
  - empathy_level:         Voice empathy 0=neutral, 1=very empathetic (optional)
  - technical_depth:       Voice technical depth 0=vulgarizer, 1=very technical (optional)
  - directness_level:      Voice directness 0=diplomatic, 1=very direct (optional)
  - vocabulary_richness:   Voice vocabulary richness 0=simple, 1=very rich (optional)
  - metaphor_usage:        Voice metaphor frequency 0=rare, 1=frequent (optional)
  - uses_emojis:           Enable/disable emoji usage (boolean, optional)
  - uses_markdown:         Enable/disable markdown formatting (boolean, optional)
  - sentence_structure:    "concise"|"elaborate"|"balanced"|"punchy"|"flowing" (optional)
  - explanation_style:     "analogy"|"step_by_step"|"big_picture"|"example_driven"|"socratic" (optional)
  - add_catch_phrases:     Array of recurring phrases to add (optional)
  - add_preferred_closings: Array of closing phrases to add (optional)
  - add_preferred_openings: Array of opening phrases to add (optional)
  - remove_catch_phrases:  Array of recurring phrases to remove (optional)
  - remove_preferred_closings: Array of closing phrases to remove (optional)
  - response_length:       "terse"|"concise"|"moderate"|"detailed"|"exhaustive" (optional)
  - structure_preference:  "freeform"|"bulleted"|"numbered"|"sectioned"|"mixed" (optional)
  - warmth:                Emotional warmth 0=cold, 1=very warm (optional)
  - emotion_enthusiasm:    Emotional enthusiasm 0=neutral, 1=very enthusiastic (optional)
  - playfulness:           Emotional playfulness 0=serious, 1=very playful (optional)
  - seriousness:           Emotional seriousness 0=relaxed, 1=very serious (optional)
  - encouragement_level:   Encouragement toward user 0=neutral, 1=very encouraging (optional)
  - traits:                JSON array of trait changes [{name, category, intensity, confidence, action}] (optional)

Example:
  {"agent_id": "mira-agent", "enthusiasm_level": 0.9, "humor_level": 0.6, "reason": "user request"}`,
			InputSchema: mcptypes.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_id":                 map[string]string{"type": "string", "description": "Agent identifier"},
					"reason":                   map[string]string{"type": "string", "description": "Reason for the change (stored in history)"},
					"enthusiasm_level":         map[string]string{"type": "number", "description": "Voice enthusiasm (0-1)"},
					"formality_level":          map[string]string{"type": "number", "description": "Voice formality (0-1)"},
					"humor_level":              map[string]string{"type": "number", "description": "Voice humor (0-1)"},
					"empathy_level":            map[string]string{"type": "number", "description": "Voice empathy (0-1)"},
					"technical_depth":          map[string]string{"type": "number", "description": "Voice technical depth (0-1)"},
					"directness_level":         map[string]string{"type": "number", "description": "Voice directness (0-1)"},
					"vocabulary_richness":      map[string]string{"type": "number", "description": "Voice vocabulary richness (0-1)"},
					"metaphor_usage":           map[string]string{"type": "number", "description": "Voice metaphor frequency (0-1)"},
					"uses_emojis":              map[string]string{"type": "boolean", "description": "Enable/disable emoji usage"},
					"uses_markdown":            map[string]string{"type": "boolean", "description": "Enable/disable markdown formatting"},
					"sentence_structure":       map[string]string{"type": "string", "description": "concise|elaborate|balanced|punchy|flowing"},
					"explanation_style":        map[string]string{"type": "string", "description": "analogy|step_by_step|big_picture|example_driven|socratic"},
					"add_catch_phrases":        map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Recurring phrases to add"},
					"add_preferred_closings":   map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Closing phrases to add"},
					"add_preferred_openings":   map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Opening phrases to add"},
					"remove_catch_phrases":     map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Recurring phrases to remove"},
					"remove_preferred_closings": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Closing phrases to remove"},
					"response_length":          map[string]string{"type": "string", "description": "terse|concise|moderate|detailed|exhaustive"},
					"structure_preference":     map[string]string{"type": "string", "description": "freeform|bulleted|numbered|sectioned|mixed"},
					"warmth":                   map[string]string{"type": "number", "description": "Emotional warmth (0-1)"},
					"emotion_enthusiasm":       map[string]string{"type": "number", "description": "Emotional enthusiasm (0-1)"},
					"playfulness":              map[string]string{"type": "number", "description": "Emotional playfulness (0-1)"},
					"seriousness":              map[string]string{"type": "number", "description": "Emotional seriousness (0-1)"},
					"encouragement_level":      map[string]string{"type": "number", "description": "Encouragement level (0-1)"},
					"traits":                   map[string]string{"type": "string", "description": "JSON array of trait changes [{name,category,intensity,confidence,action}]"},
				},
			},
		},
	}
}

// Call dispatches a soul_* tool call. Returns an error for unknown tool names.
// Used for combined registration with another MCP server (e.g., MIRA).
func (c *Controller) Call(ctx context.Context, name string, arguments map[string]interface{}) (*mcptypes.CallToolResult, error) {
	switch name {
	case "soul_capture":
		return c.handleCapture(ctx, arguments)
	case "soul_recall":
		return c.handleRecall(ctx, arguments)
	case "soul_drift":
		return c.handleDrift(ctx, arguments)
	case "soul_swap":
		return c.handleSwap(ctx, arguments)
	case "soul_status":
		return c.handleStatus(ctx, arguments)
	case "soul_history":
		return c.handleHistory(ctx, arguments)
	case "soul_update":
		return c.handleUpdate(ctx, arguments)
	case "soul_patch":
		return c.handlePatch(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown soul tool: %s", name)
	}
}

// Serve starts the SOUL MCP server over stdio (same pattern as MIRA).
// Blocks until the transport closes.
func Serve(a *app.SoulApplication) error {
	s := server.NewDefaultServer("soul", "0.1.0")
	ctrl := NewController(a)
	ctrl.RegisterTools(s)
	return server.ServeStdio(s)
}

// --- Tool Handlers ---

func (c *Controller) handleCapture(ctx context.Context, args map[string]interface{}) (*mcptypes.CallToolResult, error) {
	agentID, ok := args["agent_id"].(string)
	if !ok || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	conversation, ok := args["conversation"].(string)
	if !ok || strings.TrimSpace(conversation) == "" {
		return nil, fmt.Errorf("conversation is required")
	}

	modelID := "unknown"
	if m, ok := args["model_id"].(string); ok && m != "" {
		modelID = m
	}

	sessionID := ""
	if s, ok := args["session_id"].(string); ok {
		sessionID = s
	}

	behavioralMetrics := map[string]interface{}{}
	if bm, ok := args["behavioral_metrics"].(string); ok && bm != "" {
		if err := json.Unmarshal([]byte(bm), &behavioralMetrics); err != nil {
			return nil, fmt.Errorf("behavioral_metrics must be valid JSON: %w", err)
		}
	}

	request := &valueobjects.SoulCaptureRequest{
		AgentID:           agentID,
		Conversation:      conversation,
		ModelID:           modelID,
		SessionID:         sessionID,
		Timestamp:         time.Now(),
		BehavioralMetrics: behavioralMetrics,
	}

	snapshot, err := c.app.Capture(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("capture failed: %w", err)
	}

	result := fmt.Sprintf(`Identity captured successfully.
Agent:     %s
Version:   %d
Model:     %s
Confidence: %.1f%%
Traits:    %d captured
Timestamp: %s`,
		snapshot.AgentID,
		snapshot.Version,
		snapshot.ModelIdentifier,
		snapshot.ConfidenceScore*100,
		len(snapshot.PersonalityTraits),
		snapshot.CreatedAt.Format(time.RFC3339),
	)

	return &mcptypes.CallToolResult{
		Content: []mcptypes.Content{mcptypes.TextContent{Type: "text", Text: result}},
	}, nil
}

func (c *Controller) handleRecall(ctx context.Context, args map[string]interface{}) (*mcptypes.CallToolResult, error) {
	agentID, ok := args["agent_id"].(string)
	if !ok || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	budget := 1000
	if b, ok := args["budget"]; ok {
		switch v := b.(type) {
		case float64:
			budget = int(v)
		case int:
			budget = v
		}
	}
	if budget <= 0 {
		budget = 1000
	}

	contextStr := ""
	if ct, ok := args["context"].(string); ok {
		contextStr = ct
	}

	query := &valueobjects.SoulQuery{
		AgentID:          agentID,
		Context:          contextStr,
		BudgetTokens:     budget,
		PrioritizeRecent: true,
	}

	prompt, err := c.app.Recall(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("recall failed: %w", err)
	}

	result := fmt.Sprintf("=== SOUL IDENTITY CONTEXT (%d tokens) ===\n\n%s\n\n=== END SOUL IDENTITY ===",
		prompt.TokenEstimate, prompt.Content)

	return &mcptypes.CallToolResult{
		Content: []mcptypes.Content{mcptypes.TextContent{Type: "text", Text: result}},
	}, nil
}

func (c *Controller) handleDrift(ctx context.Context, args map[string]interface{}) (*mcptypes.CallToolResult, error) {
	agentID, ok := args["agent_id"].(string)
	if !ok || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	window := 10
	if w, ok := args["window"]; ok {
		switch v := w.(type) {
		case float64:
			window = int(v)
		case int:
			window = v
		}
	}
	if window <= 0 {
		window = 10
	}

	report, err := c.app.GetDriftReport(ctx, agentID, window)
	if err != nil {
		return nil, fmt.Errorf("drift check failed: %w", err)
	}

	status := "stable"
	if report.IsSignificant {
		status = "SIGNIFICANT DRIFT DETECTED"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("=== SOUL DRIFT REPORT: %s ===", agentID))
	parts = append(parts, fmt.Sprintf("Status:     %s", status))
	parts = append(parts, fmt.Sprintf("Drift Score: %.3f / 1.0", report.DriftScore))
	parts = append(parts, fmt.Sprintf("Window:     last %d versions", window))

	if report.PreviousVersion > 0 {
		parts = append(parts, fmt.Sprintf("Versions:   v%d -> v%d", report.PreviousVersion, report.CurrentVersion))
	}

	if len(report.DriftDimensions) > 0 {
		parts = append(parts, "\nDrift by Dimension:")
		for _, dim := range report.DriftDimensions {
			marker := ""
			if dim.IsSignificant {
				marker = " [!]"
			}
			parts = append(parts, fmt.Sprintf("  %-20s %.3f%s", dim.Dimension, dim.Change, marker))
		}
	}

	if len(report.Recommendations) > 0 {
		parts = append(parts, "\nRecommendations:")
		for _, rec := range report.Recommendations {
			parts = append(parts, "  - "+rec)
		}
	}

	return &mcptypes.CallToolResult{
		Content: []mcptypes.Content{mcptypes.TextContent{Type: "text", Text: strings.Join(parts, "\n")}},
	}, nil
}

func (c *Controller) handleSwap(ctx context.Context, args map[string]interface{}) (*mcptypes.CallToolResult, error) {
	agentID, ok := args["agent_id"].(string)
	if !ok || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	fromModel, ok := args["from_model"].(string)
	if !ok || strings.TrimSpace(fromModel) == "" {
		return nil, fmt.Errorf("from_model is required")
	}

	toModel, ok := args["to_model"].(string)
	if !ok || strings.TrimSpace(toModel) == "" {
		return nil, fmt.Errorf("to_model is required")
	}

	prompt, err := c.app.HandleModelSwap(ctx, agentID, fromModel, toModel)
	if err != nil {
		return nil, fmt.Errorf("model swap failed: %w", err)
	}

	result := fmt.Sprintf(`=== SOUL MODEL SWAP ===
Agent:      %s
Transition: %s -> %s
Status:     Identity preserved

=== REINFORCEMENT PROMPT (%d tokens) ===

%s

=== END REINFORCEMENT PROMPT ===`,
		agentID, fromModel, toModel, prompt.TokenEstimate, prompt.Content)

	return &mcptypes.CallToolResult{
		Content: []mcptypes.Content{mcptypes.TextContent{Type: "text", Text: result}},
	}, nil
}

func (c *Controller) handleStatus(ctx context.Context, args map[string]interface{}) (*mcptypes.CallToolResult, error) {
	agentID, ok := args["agent_id"].(string)
	if !ok || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	summary, err := c.app.GetIdentitySummary(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity summary: %w", err)
	}

	history, err := c.app.GetIdentityHistory(ctx, agentID, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	driftReport, driftErr := c.app.GetDriftReport(ctx, agentID, 10)
	if driftErr != nil {
		log.Printf("[SOUL] soul_status: drift report unavailable for agent %s: %v", agentID, driftErr)
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("=== SOUL STATUS: %s ===", agentID))
	parts = append(parts, "")
	parts = append(parts, summary)

	if len(history) > 0 {
		parts = append(parts, "\nRecent Snapshots:")
		for _, snap := range history {
			parts = append(parts, fmt.Sprintf("  v%-3d | %.1f%% confidence | %s | model: %s",
				snap.Version, snap.ConfidenceScore*100,
				snap.CreatedAt.Format("2006-01-02 15:04"),
				snap.ModelIdentifier))
		}
	}

	if driftReport != nil {
		driftStatus := "stable"
		if driftReport.IsSignificant {
			driftStatus = "DRIFT ALERT"
		}
		parts = append(parts, fmt.Sprintf("\nDrift: %.3f (%s)", driftReport.DriftScore, driftStatus))
	}

	return &mcptypes.CallToolResult{
		Content: []mcptypes.Content{mcptypes.TextContent{Type: "text", Text: strings.Join(parts, "\n")}},
	}, nil
}

func (c *Controller) handleHistory(ctx context.Context, args map[string]interface{}) (*mcptypes.CallToolResult, error) {
	agentID, ok := args["agent_id"].(string)
	if !ok || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	limit := 10
	if l, ok := args["limit"]; ok {
		switch v := l.(type) {
		case float64:
			limit = int(v)
		case int:
			limit = v
		}
	}
	if limit <= 0 {
		limit = 10
	}

	history, err := c.app.GetIdentityHistory(ctx, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	evolutionSummary, _ := c.app.GetEvolutionSummary(ctx, agentID)

	var parts []string
	parts = append(parts, fmt.Sprintf("=== SOUL HISTORY: %s ===", agentID))
	if evolutionSummary != "" {
		parts = append(parts, "")
		parts = append(parts, evolutionSummary)
	}

	parts = append(parts, fmt.Sprintf("\nSnapshots (%d):", len(history)))
	for i, snap := range history {
		parts = append(parts, fmt.Sprintf("\n[%d] Version %d", i+1, snap.Version))
		parts = append(parts, fmt.Sprintf("    Date:       %s", snap.CreatedAt.Format("2006-01-02 15:04:05")))
		parts = append(parts, fmt.Sprintf("    Model:      %s", snap.ModelIdentifier))
		parts = append(parts, fmt.Sprintf("    Confidence: %.1f%%", snap.ConfidenceScore*100))
		parts = append(parts, fmt.Sprintf("    Traits:     %d", len(snap.PersonalityTraits)))
		if snap.DerivedFromID != nil {
			parts = append(parts, fmt.Sprintf("    Parent:     %s", snap.DerivedFromID.String()))
		}
	}

	if len(history) == 0 {
		parts = append(parts, "  No identity snapshots found for this agent.")
	}

	return &mcptypes.CallToolResult{
		Content: []mcptypes.Content{mcptypes.TextContent{Type: "text", Text: strings.Join(parts, "\n")}},
	}, nil
}

func (c *Controller) handleUpdate(ctx context.Context, args map[string]interface{}) (*mcptypes.CallToolResult, error) {
	agentID, ok := args["agent_id"].(string)
	if !ok || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	directive, ok := args["directive"].(string)
	if !ok || strings.TrimSpace(directive) == "" {
		return nil, fmt.Errorf("directive is required")
	}

	reason := ""
	if r, ok := args["reason"].(string); ok {
		reason = r
	}

	snap, result, err := c.app.UpdateFromDirective(ctx, agentID, directive, reason)
	if err != nil {
		return nil, fmt.Errorf("soul_update failed: %w", err)
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("=== SOUL UPDATE: %s ===", agentID))
	parts = append(parts, fmt.Sprintf("New version:  v%d", snap.Version))
	parts = append(parts, fmt.Sprintf("Confidence:   %.1f%%", snap.ConfidenceScore*100))
	parts = append(parts, fmt.Sprintf("Directive:    %s", directive))
	if len(result.ChangesApplied) > 0 {
		parts = append(parts, fmt.Sprintf("\nChanges applied (%d):", len(result.ChangesApplied)))
		for _, ch := range result.ChangesApplied {
			parts = append(parts, "  • "+ch)
		}
	}
	parts = append(parts, fmt.Sprintf("\nTimestamp: %s", result.Timestamp.Format(time.RFC3339)))

	return &mcptypes.CallToolResult{
		Content: []mcptypes.Content{mcptypes.TextContent{Type: "text", Text: strings.Join(parts, "\n")}},
	}, nil
}

func (c *Controller) handlePatch(ctx context.Context, args map[string]interface{}) (*mcptypes.CallToolResult, error) {
	agentID, ok := args["agent_id"].(string)
	if !ok || strings.TrimSpace(agentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	patch := &valueobjects.IdentityPatch{}

	if r, ok := args["reason"].(string); ok {
		patch.Reason = r
	}

	// Helper to extract float64 pointer
	getFloat := func(key string) *float64 {
		if v, ok := args[key]; ok {
			switch n := v.(type) {
			case float64:
				return &n
			case int:
				f := float64(n)
				return &f
			}
		}
		return nil
	}

	patch.EnthusiasmLevel = getFloat("enthusiasm_level")
	patch.FormalityLevel = getFloat("formality_level")
	patch.HumorLevel = getFloat("humor_level")
	patch.EmpathyLevel = getFloat("empathy_level")
	patch.TechnicalDepth = getFloat("technical_depth")
	patch.DirectnessLevel = getFloat("directness_level")
	patch.VocabularyRichness = getFloat("vocabulary_richness")
	patch.MetaphorUsage = getFloat("metaphor_usage")
	patch.Warmth = getFloat("warmth")
	patch.EmotionEnthusiasm = getFloat("emotion_enthusiasm")
	patch.Playfulness = getFloat("playfulness")
	patch.Seriousness = getFloat("seriousness")
	patch.EncouragementLevel = getFloat("encouragement_level")

	if b, ok := args["uses_emojis"].(bool); ok {
		patch.UsesEmojis = &b
	}
	if b, ok := args["uses_markdown"].(bool); ok {
		patch.UsesMarkdown = &b
	}

	getString := func(key string) *string {
		if s, ok := args[key].(string); ok && s != "" {
			return &s
		}
		return nil
	}
	patch.SentenceStructure = getString("sentence_structure")
	patch.ExplanationStyle = getString("explanation_style")
	patch.ResponseLength = getString("response_length")
	patch.StructurePreference = getString("structure_preference")

	getStringSlice := func(key string) []string {
		v, ok := args[key]
		if !ok {
			return nil
		}
		switch s := v.(type) {
		case []interface{}:
			out := make([]string, 0, len(s))
			for _, item := range s {
				if str, ok := item.(string); ok {
					out = append(out, str)
				}
			}
			return out
		case []string:
			return s
		}
		return nil
	}
	patch.AddCatchPhrases = getStringSlice("add_catch_phrases")
	patch.AddPreferredClosings = getStringSlice("add_preferred_closings")
	patch.AddPreferredOpenings = getStringSlice("add_preferred_openings")
	patch.RemoveCatchPhrases = getStringSlice("remove_catch_phrases")
	patch.RemovePreferredClosings = getStringSlice("remove_preferred_closings")

	// Traits: accept JSON string or []interface{} (already decoded by MCP)
	if tv, ok := args["traits"]; ok {
		var rawTraits []valueobjects.TraitChange
		switch t := tv.(type) {
		case string:
			if err := json.Unmarshal([]byte(t), &rawTraits); err != nil {
				log.Printf("[SOUL] soul_patch: invalid traits JSON: %v", err)
			} else {
				patch.TraitChanges = rawTraits
			}
		case []interface{}:
			if b, err := json.Marshal(t); err == nil {
				_ = json.Unmarshal(b, &rawTraits)
				patch.TraitChanges = rawTraits
			}
		}
	}

	snap, result, err := c.app.PatchIdentity(ctx, agentID, patch)
	if err != nil {
		return nil, fmt.Errorf("soul_patch failed: %w", err)
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("=== SOUL PATCH: %s ===", agentID))
	parts = append(parts, fmt.Sprintf("New version:  v%d", snap.Version))
	parts = append(parts, fmt.Sprintf("Confidence:   %.1f%%", snap.ConfidenceScore*100))
	if len(result.ChangesApplied) > 0 {
		parts = append(parts, fmt.Sprintf("\nChanges applied (%d):", len(result.ChangesApplied)))
		for _, ch := range result.ChangesApplied {
			parts = append(parts, "  • "+ch)
		}
	}
	parts = append(parts, fmt.Sprintf("\nTimestamp: %s", result.Timestamp.Format(time.RFC3339)))

	return &mcptypes.CallToolResult{
		Content: []mcptypes.Content{mcptypes.TextContent{Type: "text", Text: strings.Join(parts, "\n")}},
	}, nil
}
