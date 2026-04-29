// Package ports defines public ports for SOUL external consumers.
package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MiraMemoryProvider abstracts access to MIRA factual memories.
// Implemented by MIRA (or a bridge), consumed by SOUL.
type MiraMemoryProvider interface {
	// GetMiraMemories retrieves factual memories relevant to the given context.
	GetMiraMemories(ctx context.Context, agentID, query string, limit int) ([]MiraMemoryReference, error)

	// GetLinkedMemories retrieves MIRA memories linked to a SOUL identity.
	GetLinkedMemories(ctx context.Context, identityID uuid.UUID) ([]MiraMemoryReference, error)

	// LinkIdentityToMemory creates a link between a SOUL identity and a MIRA memory.
	LinkIdentityToMemory(ctx context.Context, identityID, memoryID uuid.UUID) error

	// NotifyMiraOfIdentityChange notifies MIRA that the agent's identity has changed.
	NotifyMiraOfIdentityChange(ctx context.Context, agentID string, changeType string) error
}

// MiraMemoryReference represents a reference to a MIRA memory.
type MiraMemoryReference struct {
	MemoryID   uuid.UUID `json:"memory_id"`
	Content    string    `json:"content"`
	MemoryType string    `json:"memory_type"`
	Relevance  float64   `json:"relevance"`
	Timestamp  time.Time `json:"timestamp"`
	Wing       string    `json:"wing"`
	Room       *string   `json:"room"`
}
