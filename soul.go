// Package soul is the public API for the SOUL identity sub-system.
// It exposes only what external modules (e.g., MIRA) need to embed SOUL.
package soul

import (
	"database/sql"

	internalapp "github.com/benoitpetit/soul/internal/app"
	internalmcp "github.com/benoitpetit/soul/internal/interfaces/mcp"
)

// Application is the public alias for the SOUL application.
type Application = internalapp.SoulApplication

// Controller is the public alias for the SOUL MCP controller.
type Controller = internalmcp.Controller

// NewApplicationWithDB creates a SOUL Application reusing an existing *sql.DB.
// The database connection is NOT closed by Application.Close() — the caller retains ownership.
// Used when SOUL is embedded inside MIRA and shares its SQLite connection.
func NewApplicationWithDB(db *sql.DB) (*Application, error) {
	return internalapp.NewSoulApplicationWithDB(db)
}

// NewController creates a SOUL MCP controller that wraps the given Application.
func NewController(a *Application) *Controller {
	return internalmcp.NewController(a)
}
