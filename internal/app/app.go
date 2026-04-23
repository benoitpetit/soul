// Package app implémente le wiring de l'application SOUL
// Connecte tous les composants selon l'architecture hexagonale.
package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/benoitpetit/soul/internal/adapters/composition"
	"github.com/benoitpetit/soul/internal/adapters/drift"
	"github.com/benoitpetit/soul/internal/adapters/extraction"
	modelswaphandler "github.com/benoitpetit/soul/internal/adapters/modelswap"
	"github.com/benoitpetit/soul/internal/adapters/sqlite"
	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/interactors"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// SoulApplication représente l'application SOUL complète
// C'est le point d'entrée central qui orchestre tous les use cases.
type SoulApplication struct {
	// Storage
	Storage ports.SoulStorage
	
	// Services
	Extractor   ports.IdentityExtractor
	Composer    ports.IdentityComposer
	DriftDetector ports.IdentityDriftDetector
	SwapHandler ports.ModelSwapHandler
	Merger      ports.SoulMerger
	
	// Use Cases
	CaptureUC   *interactors.IdentityCaptureUseCase
	RecallUC    *interactors.IdentityRecallUseCase
	DriftUC     *interactors.DriftDetectionUseCase
	SwapUC      *interactors.ModelSwapUseCase
	EvolutionUC *interactors.IdentityEvolutionUseCase
	MergeUC     *interactors.IdentityMergeUseCase
	UpdateUC    *interactors.IdentityUpdateUseCase
}

// SoulConfig configure l'application SOUL
type SoulConfig struct {
	StoragePath      string  `json:"storage_path"`        // Chemin vers la base SQLite (partagée avec MIRA)
	DriftThreshold   float64 `json:"drift_threshold"`     // Seuil de détection de dérive (0-1)
	MaxContextTokens int     `json:"max_context_tokens"`  // Budget de tokens pour le prompt d'identité

	// Extraction configuration
	MinTraitConfidence       float64 `json:"min_trait_confidence"`        // Default: 0.3
	MinObservationsForTrait  int     `json:"min_observations_for_trait"` // Default: 5

	// Drift detection configuration
	DriftWindowSize           int  `json:"drift_window_size"`            // Default: 10
	AutoCheckAfterCapture     bool `json:"auto_check_after_capture"`    // Default: true

	// Model swap configuration
	AutoReinforce             bool    `json:"auto_reinforce"`            // Default: true

	// Evolution tracking
	EvolutionEnabled          bool `json:"evolution_enabled"`           // Default: true
	MaxHistoryVersions        int  `json:"max_history_versions"`         // Default: 100

	// MCP server configuration (for HTTP transport - future use)
	MCPEnabled                bool   `json:"mcp_enabled"`
	MCPHost                   string `json:"mcp_host"`  // Default: "localhost"
	MCPPort                   int    `json:"mcp_port"`  // Default: 8081
}

// DefaultConfig retourne la configuration par défaut
func DefaultConfig() *SoulConfig {
	return &SoulConfig{
		StoragePath:      ".mira/mira.db", // Partage la même base que MIRA
		DriftThreshold:   0.3,
		MaxContextTokens: 1000,

		// Extraction defaults
		MinTraitConfidence:      0.3,
		MinObservationsForTrait: 5,

		// Drift detection defaults
		DriftWindowSize:       10,
		AutoCheckAfterCapture: true,

		// Model swap defaults
		AutoReinforce: true,

		// Evolution defaults
		EvolutionEnabled:   true,
		MaxHistoryVersions: 100,

		// MCP defaults (currently stdio only, HTTP is future)
		MCPEnabled: false,
		MCPHost:    "localhost",
		MCPPort:    8081,
	}
}

// NewSoulApplication crée et configure l'application SOUL
func NewSoulApplication(config *SoulConfig) (*SoulApplication, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	// 1. Storage (SQLite - partagé avec MIRA)
	storage, err := sqlite.NewSoulSQLiteStorage(config.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	
	return wireSoulApplication(storage, config)
}

// NewSoulApplicationWithDB crée l'application SOUL en réutilisant une connexion *sql.DB existante.
// Utilisé quand SOUL est embarqué dans MIRA — la connexion n'est PAS fermée par Close().
func NewSoulApplicationWithDB(db *sql.DB) (*SoulApplication, error) {
	// 1. Storage — réutilise la connexion MIRA (ownsDB = false)
	storage, err := sqlite.NewSoulSQLiteStorageFromDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	
	return wireSoulApplication(storage, DefaultConfig())
}

// wireSoulApplication câble les services et use cases à partir d'un storage déjà initialisé.
func wireSoulApplication(storage *sqlite.SoulSQLiteStorage, config *SoulConfig) (*SoulApplication, error) {
	// Services
	extractor := extraction.NewSoulExtractorService()
	composer := composition.NewSoulComposerService()
	driftDetector := drift.NewSoulDriftDetector(config.DriftThreshold)
	swapHandler := modelswaphandler.NewSoulModelSwapHandler()
	merger := modelswaphandler.NewSoulMergerService()
	
	// Use Cases
	captureUC := interactors.NewIdentityCaptureUseCase(storage, extractor)
	recallUC := interactors.NewIdentityRecallUseCase(storage, composer)
	driftUC := interactors.NewDriftDetectionUseCase(storage, driftDetector, composer)
	swapUC := interactors.NewModelSwapUseCase(storage, swapHandler, composer)
	evolutionUC := interactors.NewIdentityEvolutionUseCase(storage, nil)
	mergeUC := interactors.NewIdentityMergeUseCase(storage, merger)
	updateUC := interactors.NewIdentityUpdateUseCase(storage)

	return &SoulApplication{
		Storage:       storage,
		Extractor:     extractor,
		Composer:      composer,
		DriftDetector: driftDetector,
		SwapHandler:   swapHandler,
		Merger:        merger,
		CaptureUC:     captureUC,
		RecallUC:      recallUC,
		DriftUC:       driftUC,
		SwapUC:        swapUC,
		EvolutionUC:   evolutionUC,
		MergeUC:       mergeUC,
		UpdateUC:      updateUC,
	}, nil
}

// --- API Publique de SOUL ---

// Capture capture l'identité depuis une conversation
func (app *SoulApplication) Capture(ctx context.Context, request *valueobjects.SoulCaptureRequest) (*entities.IdentitySnapshot, error) {
	log.Printf("[SOUL] Capturing identity for agent %s", request.AgentID)
	
	snapshot, err := app.CaptureUC.CaptureFromConversation(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("capture failed: %w", err)
	}
	
	// Tracker l'évolution
	if app.EvolutionUC != nil {
		_, _ = app.EvolutionUC.TrackSnapshot(ctx, snapshot)
	}
	
	log.Printf("[SOUL] Identity captured: version %d (confidence: %.2f)", 
		snapshot.Version, snapshot.ConfidenceScore)
	
	return snapshot, nil
}

// Recall récupère l'identité pour injection dans le contexte LLM
func (app *SoulApplication) Recall(ctx context.Context, query *valueobjects.SoulQuery) (*valueobjects.IdentityContextPrompt, error) {
	log.Printf("[SOUL] Recalling identity for agent %s", query.AgentID)
	
	prompt, err := app.RecallUC.RecallIdentityWithContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("recall failed: %w", err)
	}
	
	log.Printf("[SOUL] Identity recalled: %d tokens", prompt.TokenEstimate)
	
	return prompt, nil
}

// CheckDrift vérifie la dérive identitaire
func (app *SoulApplication) CheckDrift(ctx context.Context, agentID string, currentIdentity *entities.IdentitySnapshot) (*valueobjects.IdentityDriftReport, error) {
	log.Printf("[SOUL] Checking drift for agent %s", agentID)
	
	report, err := app.DriftUC.CheckDrift(ctx, agentID, currentIdentity)
	if err != nil {
		return nil, fmt.Errorf("drift check failed: %w", err)
	}
	
	if report.IsSignificant {
		log.Printf("[SOUL] ALERT: Significant drift detected (score: %.2f)", report.DriftScore)
	}
	
	return report, nil
}

// HandleModelSwap gère un changement de modèle
func (app *SoulApplication) HandleModelSwap(ctx context.Context, agentID, previousModel, newModel string) (*valueobjects.IdentityContextPrompt, error) {
	log.Printf("[SOUL] Handling model swap for agent %s: %s -> %s", agentID, previousModel, newModel)
	
	// 1. Enregistrer le swap
	_, err := app.SwapUC.HandleModelSwap(ctx, agentID, previousModel, newModel)
	if err != nil {
		return nil, fmt.Errorf("model swap handling failed: %w", err)
	}
	
	// 2. Générer le prompt de renforcement
	prompt, err := app.SwapUC.GetReinforcementPrompt(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("reinforcement prompt generation failed: %w", err)
	}
	
	log.Printf("[SOUL] Reinforcement prompt generated: %d tokens", prompt.TokenEstimate)
	
	return prompt, nil
}

// GetIdentitySummary retourne un résumé lisible de l'identité
func (app *SoulApplication) GetIdentitySummary(ctx context.Context, agentID string) (string, error) {
	return app.RecallUC.GetIdentitySummary(ctx, agentID)
}

// GetIdentityHistory retourne l'historique des snapshots
func (app *SoulApplication) GetIdentityHistory(ctx context.Context, agentID string, limit int) ([]*entities.IdentitySnapshot, error) {
	return app.RecallUC.GetIdentityHistory(ctx, agentID, limit)
}

// GetDriftReport retourne le rapport de dérive
func (app *SoulApplication) GetDriftReport(ctx context.Context, agentID string, windowSize int) (*valueobjects.IdentityDriftReport, error) {
	return app.DriftUC.GetDriftReport(ctx, agentID, windowSize)
}

// GetEvolutionSummary retourne un résumé de l'évolution
func (app *SoulApplication) GetEvolutionSummary(ctx context.Context, agentID string) (string, error) {
	return app.EvolutionUC.GetEvolutionSummary(ctx, agentID)
}

// UpdateFromDirective parse une directive en langage naturel et l'applique comme patch d'identité.
func (app *SoulApplication) UpdateFromDirective(ctx context.Context, agentID, directive, reason string) (*entities.IdentitySnapshot, *interactors.UpdateResult, error) {
	log.Printf("[SOUL] UpdateFromDirective for agent %s: %q", agentID, directive)
	snap, result, err := app.UpdateUC.UpdateFromDirective(ctx, agentID, directive, reason)
	if err != nil {
		return nil, nil, fmt.Errorf("update from directive failed: %w", err)
	}
	log.Printf("[SOUL] Update applied: version %d, %d change(s)", snap.Version, len(result.ChangesApplied))
	return snap, result, nil
}

// PatchIdentity applique un patch structuré sur l'identité de l'agent.
func (app *SoulApplication) PatchIdentity(ctx context.Context, agentID string, patch *valueobjects.IdentityPatch) (*entities.IdentitySnapshot, *interactors.UpdateResult, error) {
	log.Printf("[SOUL] PatchIdentity for agent %s", agentID)
	snap, result, err := app.UpdateUC.PatchIdentity(ctx, agentID, patch)
	if err != nil {
		return nil, nil, fmt.Errorf("patch identity failed: %w", err)
	}
	log.Printf("[SOUL] Patch applied: version %d, %d change(s)", snap.Version, len(result.ChangesApplied))
	return snap, result, nil
}

// Close ferme l'application proprement
func (app *SoulApplication) Close() error {
	if app.Storage != nil {
		// Type-assert to concrete storage to access Close
		type closer interface{ Close() error }
		if c, ok := app.Storage.(closer); ok {
			return c.Close()
		}
	}
	return nil
}
