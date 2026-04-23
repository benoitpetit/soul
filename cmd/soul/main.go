// SOUL - System for Observed Unique Legacy
// Identity preservation extension for LLM agents.
//
// Usage:
//
//	soul capture --agent <id> --conversation <file>    # Capture identity from conversation
//	soul recall --agent <id>                           # Recall identity prompt
//	soul drift --agent <id>                            # Check identity drift
//	soul swap --agent <id> --from <model> --to <model> # Handle model change
//	soul status --agent <id>                           # Show identity status
//	soul history --agent <id>                          # Show evolution history
//
//	# MCP Server mode (standalone or with MIRA)
//	soul mcp                                             # Start MCP server
//
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/benoitpetit/soul/internal/app"
	soulmcp "github.com/benoitpetit/soul/internal/interfaces/mcp"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()

	// Dispatch commands
	switch os.Args[1] {
	case "capture":
		handleCapture(ctx)
	case "recall":
		handleRecall(ctx)
	case "drift":
		handleDrift(ctx)
	case "swap":
		handleSwap(ctx)
	case "status":
		handleStatus(ctx)
	case "history":
		handleHistory(ctx)
	case "mcp":
		handleMCP(ctx)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// loadApp is a helper used by each handler to initialize the application.
func loadApp(configFile, storagePath string) (*app.SoulApplication, error) {
	config, err := app.LoadConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if storagePath != "" {
		config.StoragePath = storagePath
	}
	return app.NewSoulApplication(config)
}

func handleCapture(ctx context.Context) {
	captureCmd := flag.NewFlagSet("capture", flag.ExitOnError)
	agentID := captureCmd.String("agent", "", "Agent ID (required)")
	conversationFile := captureCmd.String("conversation", "", "Path to conversation file (required)")
	modelID := captureCmd.String("model", "unknown", "Model identifier")
	sessionID := captureCmd.String("session", "", "Session ID")
	configFile := captureCmd.String("config", "", "Path to YAML config file")
	captureCmd.StringVar(configFile, "c", "", "Path to YAML config file (short)")
	storagePath := captureCmd.String("storage", "", "Path to SQLite database (shared with MIRA)")
	captureCmd.StringVar(storagePath, "s", "", "Path to SQLite database (short)")

	captureCmd.Parse(os.Args[2:])

	if *agentID == "" || *conversationFile == "" {
		fmt.Fprintf(os.Stderr, "Error: --agent and --conversation are required\n")
		captureCmd.Usage()
		os.Exit(1)
	}

	// Read the conversation
	conversation, err := readFile(*conversationFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read conversation: %v\n", err)
		os.Exit(1)
	}

	soulApp, err := loadApp(*configFile, *storagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SOUL: %v\n", err)
		os.Exit(1)
	}
	defer soulApp.Close()

	// Build the request
	request := &valueobjects.SoulCaptureRequest{
		AgentID:      *agentID,
		Conversation: conversation,
		ModelID:      *modelID,
		SessionID:    *sessionID,
		Timestamp:    time.Now(),
	}

	// Capture
	snapshot, err := soulApp.Capture(ctx, request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Capture failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Identity captured successfully!\n")
	fmt.Printf("  Agent: %s\n", snapshot.AgentID)
	fmt.Printf("  Version: %d\n", snapshot.Version)
	fmt.Printf("  Confidence: %.1f%%\n", snapshot.ConfidenceScore*100)
	fmt.Printf("  Model: %s\n", snapshot.ModelIdentifier)
	fmt.Printf("  Traits captured: %d\n", len(snapshot.PersonalityTraits))
}

func handleRecall(ctx context.Context) {
	recallCmd := flag.NewFlagSet("recall", flag.ExitOnError)
	agentID := recallCmd.String("agent", "", "Agent ID (required)")
	ctxFlag := recallCmd.String("context", "", "Current conversation context")
	budget := recallCmd.Int("budget", 1000, "Token budget for identity prompt")
	configFile := recallCmd.String("config", "", "Path to YAML config file")
	recallCmd.StringVar(configFile, "c", "", "Path to YAML config file (short)")
	storagePath := recallCmd.String("storage", "", "Path to SQLite database (shared with MIRA)")
	recallCmd.StringVar(storagePath, "s", "", "Path to SQLite database (short)")

	recallCmd.Parse(os.Args[2:])

	if *agentID == "" {
		fmt.Fprintf(os.Stderr, "Error: --agent is required\n")
		recallCmd.Usage()
		os.Exit(1)
	}

	soulApp, err := loadApp(*configFile, *storagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SOUL: %v\n", err)
		os.Exit(1)
	}
	defer soulApp.Close()

	query := &valueobjects.SoulQuery{
		AgentID:          *agentID,
		Context:          *ctxFlag,
		BudgetTokens:     *budget,
		PrioritizeRecent: true,
	}

	prompt, err := soulApp.Recall(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Recall failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("--- Identity Context Prompt (%d tokens) ---\n", prompt.TokenEstimate)
	fmt.Println(prompt.Content)
	fmt.Println("--- End of Identity Context Prompt ---")
}

func handleDrift(ctx context.Context) {
	driftCmd := flag.NewFlagSet("drift", flag.ExitOnError)
	agentID := driftCmd.String("agent", "", "Agent ID (required)")
	window := driftCmd.Int("window", 10, "Number of versions to analyze")
	configFile := driftCmd.String("config", "", "Path to YAML config file")
	driftCmd.StringVar(configFile, "c", "", "Path to YAML config file (short)")
	storagePath := driftCmd.String("storage", "", "Path to SQLite database (shared with MIRA)")
	driftCmd.StringVar(storagePath, "s", "", "Path to SQLite database (short)")

	driftCmd.Parse(os.Args[2:])

	if *agentID == "" {
		fmt.Fprintf(os.Stderr, "Error: --agent is required\n")
		driftCmd.Usage()
		os.Exit(1)
	}

	soulApp, err := loadApp(*configFile, *storagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SOUL: %v\n", err)
		os.Exit(1)
	}
	defer soulApp.Close()

	report, err := soulApp.GetDriftReport(ctx, *agentID, *window)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Drift check failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("--- Drift Report for Agent %s ---\n", *agentID)
	fmt.Printf("Overall Drift Score: %.2f\n", report.DriftScore)
	fmt.Printf("Significant: %v\n", report.IsSignificant)

	if len(report.DriftDimensions) > 0 {
		fmt.Printf("\nDrift by Dimension:\n")
		for _, dim := range report.DriftDimensions {
			marker := ""
			if dim.IsSignificant {
				marker = " (*)"
			}
			fmt.Printf("  - %s: %.2f%s\n", dim.Dimension, dim.Change, marker)
		}
	}

	if len(report.Recommendations) > 0 {
		fmt.Printf("\nRecommendations:\n")
		for _, rec := range report.Recommendations {
			fmt.Printf("  - %s\n", rec)
		}
	}
}

func handleSwap(ctx context.Context) {
	swapCmd := flag.NewFlagSet("swap", flag.ExitOnError)
	agentID := swapCmd.String("agent", "", "Agent ID (required)")
	fromModel := swapCmd.String("from", "", "Previous model (required)")
	toModel := swapCmd.String("to", "", "New model (required)")
	configFile := swapCmd.String("config", "", "Path to YAML config file")
	swapCmd.StringVar(configFile, "c", "", "Path to YAML config file (short)")
	storagePath := swapCmd.String("storage", "", "Path to SQLite database (shared with MIRA)")
	swapCmd.StringVar(storagePath, "s", "", "Path to SQLite database (short)")

	swapCmd.Parse(os.Args[2:])

	if *agentID == "" || *fromModel == "" || *toModel == "" {
		fmt.Fprintf(os.Stderr, "Error: --agent, --from, and --to are required\n")
		swapCmd.Usage()
		os.Exit(1)
	}

	soulApp, err := loadApp(*configFile, *storagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SOUL: %v\n", err)
		os.Exit(1)
	}
	defer soulApp.Close()

	prompt, err := soulApp.HandleModelSwap(ctx, *agentID, *fromModel, *toModel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Model swap handling failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Model swap handled!\n")
	fmt.Printf("  Agent: %s\n", *agentID)
	fmt.Printf("  Transition: %s -> %s\n", *fromModel, *toModel)
	fmt.Printf("\n--- Reinforcement Prompt (%d tokens) ---\n", prompt.TokenEstimate)
	fmt.Println(prompt.Content)
}

func handleStatus(ctx context.Context) {
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	agentID := statusCmd.String("agent", "", "Agent ID (required)")
	configFile := statusCmd.String("config", "", "Path to YAML config file")
	statusCmd.StringVar(configFile, "c", "", "Path to YAML config file (short)")
	storagePath := statusCmd.String("storage", "", "Path to SQLite database (shared with MIRA)")
	statusCmd.StringVar(storagePath, "s", "", "Path to SQLite database (short)")

	statusCmd.Parse(os.Args[2:])

	if *agentID == "" {
		fmt.Fprintf(os.Stderr, "Error: --agent is required\n")
		statusCmd.Usage()
		os.Exit(1)
	}

	soulApp, err := loadApp(*configFile, *storagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SOUL: %v\n", err)
		os.Exit(1)
	}
	defer soulApp.Close()

	// Get the summary
	summary, err := soulApp.GetIdentitySummary(ctx, *agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get identity summary: %v\n", err)
		os.Exit(1)
	}

	// Get the history
	history, err := soulApp.GetIdentityHistory(ctx, *agentID, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get history: %v\n", err)
	}

	fmt.Printf("=== SOUL Status for Agent: %s ===\n\n", *agentID)
	fmt.Println(summary)

	if len(history) > 0 {
		fmt.Printf("\n--- Recent Snapshots (%d) ---\n", len(history))
		for _, snap := range history {
			fmt.Printf("  v%d | %.1f%% confidence | %s\n",
				snap.Version, snap.ConfidenceScore*100,
				snap.CreatedAt.Format("2006-01-02 15:04"))
		}
	}

	// Check for drift
	report, _ := soulApp.GetDriftReport(ctx, *agentID, 10)
	if report != nil {
		fmt.Printf("\n--- Drift Status ---\n")
		fmt.Printf("  Score: %.2f | Significant: %v\n", report.DriftScore, report.IsSignificant)
	}
}

func handleHistory(ctx context.Context) {
	historyCmd := flag.NewFlagSet("history", flag.ExitOnError)
	agentID := historyCmd.String("agent", "", "Agent ID (required)")
	limit := historyCmd.Int("limit", 10, "Number of entries")
	configFile := historyCmd.String("config", "", "Path to YAML config file")
	historyCmd.StringVar(configFile, "c", "", "Path to YAML config file (short)")
	storagePath := historyCmd.String("storage", "", "Path to SQLite database (shared with MIRA)")
	historyCmd.StringVar(storagePath, "s", "", "Path to SQLite database (short)")

	historyCmd.Parse(os.Args[2:])

	if *agentID == "" {
		fmt.Fprintf(os.Stderr, "Error: --agent is required\n")
		historyCmd.Usage()
		os.Exit(1)
	}

	soulApp, err := loadApp(*configFile, *storagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SOUL: %v\n", err)
		os.Exit(1)
	}
	defer soulApp.Close()

	// Get evolution summary
	summary, err := soulApp.GetEvolutionSummary(ctx, *agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get evolution summary: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== Evolution History for Agent: %s ===\n\n", *agentID)
	fmt.Println(summary)

	// Get snapshots
	history, err := soulApp.GetIdentityHistory(ctx, *agentID, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get history: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n--- Snapshots (%d) ---\n", len(history))
	for i, snap := range history {
		fmt.Printf("\n[%d] Version %d (confidence: %.1f%%)\n",
			i+1, snap.Version, snap.ConfidenceScore*100)
		fmt.Printf("    Created: %s\n", snap.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Model: %s\n", snap.ModelIdentifier)
		fmt.Printf("    Traits: %d\n", len(snap.PersonalityTraits))
		if snap.DerivedFromID != nil {
			fmt.Printf("    Parent: %s\n", snap.DerivedFromID.String())
		}
	}
}

func handleMCP(ctx context.Context) {
	mcpCmd := flag.NewFlagSet("mcp", flag.ExitOnError)
	configFile := mcpCmd.String("config", "", "Path to YAML config file")
	mcpCmd.StringVar(configFile, "c", "", "Path to YAML config file (short)")
	storagePath := mcpCmd.String("storage", "", "Path to SQLite database (shared with MIRA)")
	mcpCmd.StringVar(storagePath, "s", "", "Path to SQLite database (short)")

	mcpCmd.Parse(os.Args[2:])

	soulApp, err := loadApp(*configFile, *storagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize SOUL: %v\n", err)
		os.Exit(1)
	}
	defer soulApp.Close()

	if err := soulmcp.Serve(soulApp); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

// --- Utilities ---

func printUsage() {
	fmt.Println(`SOUL - System for Observed Unique Legacy
Identity preservation for LLM agents. Can run standalone or embedded in MIRA.

Usage:
  soul <command> [options]

Commands:
  capture   Capture identity from conversation
  recall    Recall identity prompt for LLM context injection
  drift     Check identity drift
  swap      Handle model swap (critical identity preservation)
  status    Show current identity status
  history   Show identity evolution history
  mcp       Start MCP server for integration with MIRA
  help      Show this help

Per-command Options (all commands support):
  --config, -c   Path to YAML config file (default: none)
  --storage, -s  Path to SQLite database (default: .mira/mira.db)

Capture Options:
  --agent        Agent ID (required)
  --conversation Path to conversation file (required)
  --model        Model identifier (default: unknown)
  --session      Session ID

Recall Options:
  --agent        Agent ID (required)
  --context      Current conversation context
  --budget       Token budget (default: 1000)

Drift Options:
  --agent        Agent ID (required)
  --window       Number of versions to analyze (default: 10)

Swap Options:
  --agent        Agent ID (required)
  --from         Previous model (required)
  --to           New model (required)

Examples:
  # Capture identity
  soul capture --agent my-agent --conversation chat.txt --model gpt-4

  # Recall identity for prompt injection
  soul recall --agent my-agent --budget 800

  # Check for identity drift
  soul drift --agent my-agent

  # Handle model swap (critical!)
  soul swap --agent my-agent --from gpt-4 --to claude-3

  # Show identity status
  soul status --agent my-agent

  # Start MCP server (for use with MIRA / Claude Desktop)
  soul mcp --storage /path/to/.mira/mira.db`)
}

func readFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
