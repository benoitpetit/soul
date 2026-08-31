// Package soul is the public API for the SOUL identity sub-system.
// It exposes only what external modules (e.g., MIRA) need to embed SOUL.
package soul

import (
	"database/sql"

	internalapp "github.com/benoitpetit/soul/internal/app"
	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	internalmcp "github.com/benoitpetit/soul/internal/interfaces/mcp"
	"github.com/benoitpetit/soul/internal/usecases/interactors"
)

// Application is the public alias for the SOUL application.
type Application = internalapp.SoulApplication

// Controller is the public alias for the SOUL MCP controller.
type Controller = internalmcp.Controller

// Config is the public alias for the SOUL configuration.
type Config = internalapp.SoulConfig

// DefaultConfig returns the default SOUL configuration.
func DefaultConfig() *Config {
	return internalapp.DefaultConfig()
}

// NewApplicationWithDB creates a SOUL Application reusing an existing *sql.DB.
// The database connection is NOT closed by Application.Close() — the caller retains ownership.
// Used when SOUL is embedded inside MIRA and shares its SQLite connection.
// Uses default configuration. For custom config, use NewApplicationWithDBAndConfig.
func NewApplicationWithDB(db *sql.DB) (*Application, error) {
	return internalapp.NewSoulApplicationWithDB(db)
}

// NewApplicationWithDBAndConfig creates a SOUL Application reusing an existing *sql.DB
// with a custom configuration.
// The database connection is NOT closed by Application.Close() — the caller retains ownership.
// If config is nil, default configuration is used.
func NewApplicationWithDBAndConfig(db *sql.DB, config *Config) (*Application, error) {
	return internalapp.NewSoulApplicationWithDBAndConfig(db, config)
}

// NewController creates a SOUL MCP controller that wraps the given Application.
func NewController(a *Application) *Controller {
	return internalmcp.NewController(a)
}

// Public type aliases for external consumers (e.g., Miracloud SaaS)

// SoulCaptureRequest is a public alias.
type SoulCaptureRequest = valueobjects.SoulCaptureRequest

// SoulQuery is a public alias.
type SoulQuery = valueobjects.SoulQuery

// IdentityContextPrompt is a public alias.
type IdentityContextPrompt = valueobjects.IdentityContextPrompt

// IdentityDriftReport is a public alias.
type IdentityDriftReport = valueobjects.IdentityDriftReport

// IdentityPatch is a public alias.
type IdentityPatch = valueobjects.IdentityPatch

// UpdateResult is a public alias.
type UpdateResult = interactors.UpdateResult

// IdentitySnapshot is a public alias.
type IdentitySnapshot = entities.IdentitySnapshot

// MergeStrategy is a public alias.
type MergeStrategy = valueobjects.MergeStrategy

// Merge strategy constants.
const (
	MergePreserveDominant = valueobjects.MergePreserveDominant
	MergeWeightedAverage  = valueobjects.MergeWeightedAverage
	MergeLatestWins       = valueobjects.MergeLatestWins
	MergeSynthesize       = valueobjects.MergeSynthesize
)
