// Package ports définit les interfaces repository et services de SOUL
// Architecture hexagonale : les usecases dépendent de ces abstractions,
// pas des implémentations concrètes.
package ports

import (
	"context"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/google/uuid"
)

// IdentityRepository définit l'interface pour le stockage des snapshots d'identité
type IdentityRepository interface {
	// StoreIdentity sauvegarde un snapshot d'identité
	StoreIdentity(ctx context.Context, identity *entities.IdentitySnapshot) error
	
	// GetLatestIdentity récupère le snapshot le plus récent pour un agent
	GetLatestIdentity(ctx context.Context, agentID string) (*entities.IdentitySnapshot, error)
	
	// GetIdentityByID récupère un snapshot par son ID
	GetIdentityByID(ctx context.Context, id uuid.UUID) (*entities.IdentitySnapshot, error)
	
	// GetIdentityHistory récupère l'historique des versions pour un agent
	GetIdentityHistory(ctx context.Context, agentID string, limit int) ([]*entities.IdentitySnapshot, error)
	
	// GetIdentityAtVersion récupère un snapshot spécifique par version
	GetIdentityAtVersion(ctx context.Context, agentID string, version int) (*entities.IdentitySnapshot, error)
	
	// DeleteIdentity supprime un snapshot
	DeleteIdentity(ctx context.Context, id uuid.UUID) error
	
	// ListAgents liste tous les agents ayant une identité stockée
	ListAgents(ctx context.Context) ([]string, error)
	
	// GetIdentityLineage récupère la lignée (parent → enfant) d'un snapshot
	GetIdentityLineage(ctx context.Context, snapshotID uuid.UUID) (*IdentityLineage, error)
}

// IdentityLineage représente la lignée évolutive d'une identité
type IdentityLineage struct {
	Root       *entities.IdentitySnapshot   `json:"root"`
	Snapshots  []*entities.IdentitySnapshot `json:"snapshots"` // Ordre chronologique
	Depth      int                          `json:"depth"`     // Nombre de générations
}

// TraitRepository définit l'interface pour le stockage des traits de personnalité
type TraitRepository interface {
	// StoreTrait sauvegarde un trait
	StoreTrait(ctx context.Context, trait *entities.PersonalityTrait) error
	
	// GetTraitByName récupère un trait par nom pour un agent
	GetTraitByName(ctx context.Context, agentID, name string) (*entities.PersonalityTrait, error)

	// GetTraitsByNames récupère plusieurs traits par noms en une seule requête (batch)
	GetTraitsByNames(ctx context.Context, agentID string, names []string) ([]*entities.PersonalityTrait, error)
	
	// GetAllTraits récupère tous les traits d'un agent
	GetAllTraits(ctx context.Context, agentID string) ([]*entities.PersonalityTrait, error)
	
	// GetTraitsByCategory récupère les traits par catégorie
	GetTraitsByCategory(ctx context.Context, agentID string, category entities.TraitCategory) ([]*entities.PersonalityTrait, error)
	
	// UpdateTrait met à jour un trait existant (fusion avec nouvelles observations)
	UpdateTrait(ctx context.Context, trait *entities.PersonalityTrait) error

	// UpsertTraits met à jour ou insère plusieurs traits en une seule transaction (batch)
	UpsertTraits(ctx context.Context, agentID string, traits []*entities.PersonalityTrait) error
	
	// GetWellEstablishedTraits récupère les traits bien établis (confiance élevée)
	GetWellEstablishedTraits(ctx context.Context, agentID string, minConfidence float64) ([]*entities.PersonalityTrait, error)
	
	// DeleteTrait supprime un trait
	DeleteTrait(ctx context.Context, id uuid.UUID) error
}

// TraitObservationRepository définit l'interface pour le stockage des observations brutes
type TraitObservationRepository interface {
	// StoreObservation sauvegarde une observation brute
	StoreObservation(ctx context.Context, obs *entities.TraitObservation) error
	
	// GetObservationsForTrait récupère les observations pour un trait spécifique
	GetObservationsForTrait(ctx context.Context, agentID, traitName string, limit int) ([]*entities.TraitObservation, error)
	
	// GetRecentObservations récupère les observations récentes
	GetRecentObservations(ctx context.Context, agentID string, since time.Time) ([]*entities.TraitObservation, error)
	
	// GetObservationsBySource récupère les observations par source
	GetObservationsBySource(ctx context.Context, agentID string, sourceType valueobjects.SourceType) ([]*entities.TraitObservation, error)
	
	// DeleteOldObservations supprime les observations anciennes (maintenance)
	DeleteOldObservations(ctx context.Context, agentID string, before time.Time) (int, error)
}

// EvolutionRepository définit l'interface pour le stockage de l'évolution identitaire
type EvolutionRepository interface {
	// RecordDiff enregistre un diff entre deux snapshots
	RecordDiff(ctx context.Context, diff *entities.IdentityDiff) error
	
	// GetDiffsForAgent récupère l'historique des diffs
	GetDiffsForAgent(ctx context.Context, agentID string, limit int) ([]*entities.IdentityDiff, error)
	
	// GetLatestDiff récupère le dernier diff
	GetLatestDiff(ctx context.Context, agentID string) (*entities.IdentityDiff, error)
	
	// GetDriftReport génère un rapport de dérive
	GetDriftReport(ctx context.Context, agentID string, windowSize int) (*valueobjects.IdentityDriftReport, error)
}

// ModelSwapRepository définit l'interface pour le stockage des changements de modèle
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
	GetMiraMemories(ctx context.Context, agentID, query string, limit int) ([]MiraMemoryReference, error)
	
	// LinkIdentityToMemory crée un lien entre une identité SOUL et une mémoire MIRA
	LinkIdentityToMemory(ctx context.Context, identityID, memoryID uuid.UUID) error
	
	// GetLinkedMemories récupère les mémoires MIRA liées à une identité
	GetLinkedMemories(ctx context.Context, identityID uuid.UUID) ([]MiraMemoryReference, error)
	
	// NotifyMiraOfIdentityChange notifie MIRA d'un changement d'identité
	NotifyMiraOfIdentityChange(ctx context.Context, agentID string, changeType string) error
}

// MiraMemoryReference représente une référence à une mémoire MIRA
type MiraMemoryReference struct {
	MemoryID    uuid.UUID `json:"memory_id"`
	Content     string    `json:"content"`      // Contenu T1 (fingerprint)
	MemoryType  string    `json:"memory_type"`  // Type de mémoire MIRA
	Relevance   float64   `json:"relevance"`    // Score de pertinence
	Timestamp   time.Time `json:"timestamp"`
	Wing        string    `json:"wing"`
	Room        *string   `json:"room"`
}

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
}

// SoulTx définit une transaction SOUL
type SoulTx interface {
	// Commit valide la transaction
	Commit() error
	// Rollback annule la transaction
	Rollback() error
}
