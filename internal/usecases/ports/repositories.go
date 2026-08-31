// Package ports defines repository and service interfaces for SOUL.
// Hexagonal architecture: use cases depend on these abstractions,
// not on concrete implementations.
package ports

import (
	"context"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	pkgports "github.com/benoitpetit/soul/pkg/ports"
	"github.com/google/uuid"
)

// IdentityRepository defines the interface for identity snapshot storage
type IdentityRepository interface {
	// StoreIdentity saves an identity snapshot
	StoreIdentity(ctx context.Context, identity *entities.IdentitySnapshot) error
	
	// GetLatestIdentity retrieves the most recent snapshot for an agent
	GetLatestIdentity(ctx context.Context, agentID string) (*entities.IdentitySnapshot, error)
	
	// GetIdentityByID retrieves a snapshot by ID
	GetIdentityByID(ctx context.Context, id uuid.UUID) (*entities.IdentitySnapshot, error)
	
	// GetIdentityHistory retrieves version history for an agent
	GetIdentityHistory(ctx context.Context, agentID string, limit int) ([]*entities.IdentitySnapshot, error)
	
	// GetIdentityAtVersion retrieves a specific snapshot by version
	GetIdentityAtVersion(ctx context.Context, agentID string, version int) (*entities.IdentitySnapshot, error)
	
	// DeleteIdentity supprime un snapshot
	DeleteIdentity(ctx context.Context, id uuid.UUID) error
	
	// ListAgents lists all agents with stored identity
	ListAgents(ctx context.Context) ([]string, error)
	
	// GetIdentityLineage retrieves the lineage (parent -> child) of a snapshot
	GetIdentityLineage(ctx context.Context, snapshotID uuid.UUID) (*IdentityLineage, error)
}

// IdentityLineage represents the evolutionary lineage of an identity
type IdentityLineage struct {
	Root       *entities.IdentitySnapshot   `json:"root"`
	Snapshots  []*entities.IdentitySnapshot `json:"snapshots"` // Ordre chronologique
	Depth      int                          `json:"depth"`     // Number of generations
}

// TraitRepository defines the interface for personality trait storage
type TraitRepository interface {
	// StoreTrait sauvegarde un trait
	StoreTrait(ctx context.Context, trait *entities.PersonalityTrait) error
	
	// GetTraitByName retrieves a trait by name for an agent
	GetTraitByName(ctx context.Context, agentID, name string) (*entities.PersonalityTrait, error)

	// GetTraitsByNames retrieves multiple traits by names in a single query (batch)
	GetTraitsByNames(ctx context.Context, agentID string, names []string) ([]*entities.PersonalityTrait, error)
	
	// GetAllTraits retrieves all traits for an agent
	GetAllTraits(ctx context.Context, agentID string) ([]*entities.PersonalityTrait, error)
	
	// GetTraitsByCategory retrieves traits by category
	GetTraitsByCategory(ctx context.Context, agentID string, category entities.TraitCategory) ([]*entities.PersonalityTrait, error)
	
	// UpdateTrait updates an existing trait (merge with new observations)
	UpdateTrait(ctx context.Context, trait *entities.PersonalityTrait) error

	// UpsertTraits updates or inserts multiple traits in a single transaction (batch)
	UpsertTraits(ctx context.Context, agentID string, traits []*entities.PersonalityTrait) error
	
	// GetWellEstablishedTraits retrieves well-established traits (high confidence)
	GetWellEstablishedTraits(ctx context.Context, agentID string, minConfidence float64) ([]*entities.PersonalityTrait, error)
	
	// DeleteTrait supprime un trait
	DeleteTrait(ctx context.Context, id uuid.UUID) error
}

// TraitObservationRepository defines the interface for raw observation storage
type TraitObservationRepository interface {
	// StoreObservation sauvegarde une observation brute
	StoreObservation(ctx context.Context, obs *entities.TraitObservation) error
	
	// GetObservationsForTrait retrieves observations for a specific trait
	GetObservationsForTrait(ctx context.Context, agentID, traitName string, limit int) ([]*entities.TraitObservation, error)
	
	// GetRecentObservations retrieves recent observations
	GetRecentObservations(ctx context.Context, agentID string, since time.Time) ([]*entities.TraitObservation, error)
	
	// GetObservationsBySource retrieves observations by source
	GetObservationsBySource(ctx context.Context, agentID string, sourceType valueobjects.SourceType) ([]*entities.TraitObservation, error)
	
	// DeleteOldObservations supprime les observations anciennes (maintenance)
	DeleteOldObservations(ctx context.Context, agentID string, before time.Time) (int, error)
}

// EvolutionRepository defines the interface for identity evolution storage
type EvolutionRepository interface {
	// RecordDiff enregistre un diff entre deux snapshots
	RecordDiff(ctx context.Context, diff *entities.IdentityDiff) error
	
	// GetDiffsForAgent retrieves diff history
	GetDiffsForAgent(ctx context.Context, agentID string, limit int) ([]*entities.IdentityDiff, error)
	
	// GetLatestDiff retrieves the latest diff
	GetLatestDiff(ctx context.Context, agentID string) (*entities.IdentityDiff, error)
	
	// GetDriftReport generates a drift report
	GetDriftReport(ctx context.Context, agentID string, windowSize int) (*valueobjects.IdentityDriftReport, error)
}

// ModelSwapRepository defines the interface for model change storage
type ModelSwapRepository interface {
	// RecordModelSwap enregistre un changement de modèle
	RecordModelSwap(ctx context.Context, swap *valueobjects.ModelSwapContext) error
	
	// GetModelSwaps récupère l'historique des changements
	GetModelSwaps(ctx context.Context, agentID string) ([]*valueobjects.ModelSwapContext, error)
	
	// GetLatestModelSwap récupère le dernier changement
	GetLatestModelSwap(ctx context.Context, agentID string) (*valueobjects.ModelSwapContext, error)
}

// MiraBridgeRepository définit l'interface pour la communication avec MIRA
// C'est le pont entre SOUL (identité) et MIRA (mémoire factuelle)
type MiraBridgeRepository interface {
	// GetMiraMemories récupère des mémoires factuelles de MIRA pour un contexte donné
	GetMiraMemories(ctx context.Context, agentID, query string, limit int) ([]pkgports.MiraMemoryReference, error)

	// LinkIdentityToMemory crée un lien entre une identité SOUL et une mémoire MIRA
	LinkIdentityToMemory(ctx context.Context, identityID, memoryID uuid.UUID) error

	// GetLinkedMemories récupère les mémoires MIRA liées à une identité
	GetLinkedMemories(ctx context.Context, identityID uuid.UUID) ([]pkgports.MiraMemoryReference, error)

	// NotifyMiraOfIdentityChange notifie MIRA d'un changement d'identité
	NotifyMiraOfIdentityChange(ctx context.Context, agentID string, changeType string) error
}

// MiraMemoryReference is an alias for the public type.
type MiraMemoryReference = pkgports.MiraMemoryReference

// SoulStorage définit l'interface composite pour toutes les opérations de stockage
// C'est l'interface principale que les adapters implémentent
type SoulStorage interface {
	IdentityRepository
	TraitRepository
	TraitObservationRepository
	EvolutionRepository
	ModelSwapRepository
	MiraBridgeRepository
	
	// Transaction support
	BeginTx(ctx context.Context) (SoulTx, error)
	WithTx(tx SoulTx) (SoulStorage, error)
}

// SoulTx définit une transaction SOUL
type SoulTx interface {
	// Commit valide la transaction
	Commit() error
	// Rollback annule la transaction
	Rollback() error
}
