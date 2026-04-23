// Package embedder implements the SoulEmbedder port.
// Converts identity snapshots and traits into 13-dimensional float32 vectors
// using the IdentityDimensionVector representation, enabling vector similarity
// searches (e.g. integration with MIRA's HNSW index).
package embedder

import (
	"context"
	"fmt"
	"math"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

const (
	// Dimension is the size of every identity embedding vector (13D).
	Dimension = 13

	// ModelHash identifies this embedding approach for cache / invalidation.
	ModelHash = "soul-13d-v1"
)

// SoulEmbedderService implements ports.SoulEmbedder.
// It encodes identities using the deterministic IdentityDimensionVector mapping
// and, when a SoulStorage is provided, supports linear-scan similarity search.
type SoulEmbedderService struct {
	storage ports.SoulStorage // optional; used only by FindSimilarIdentities
}

// NewSoulEmbedderService creates a new embedder.
// storage may be nil; passing nil disables FindSimilarIdentities.
func NewSoulEmbedderService(storage ports.SoulStorage) *SoulEmbedderService {
	return &SoulEmbedderService{storage: storage}
}

// EncodeIdentity encodes a snapshot as a normalised 13D float32 vector.
func (e *SoulEmbedderService) EncodeIdentity(ctx context.Context, identity *entities.IdentitySnapshot) ([]float32, error) {
	if identity == nil {
		return nil, fmt.Errorf("identity snapshot is nil")
	}
	vec := entities.FromIdentitySnapshot(identity)
	return toFloat32(normalize(vec.ToSlice())), nil
}

// EncodeTrait encodes a personality trait as a 13D float32 vector.
// Each trait category activates a subset of the identity dimensions.
func (e *SoulEmbedderService) EncodeTrait(ctx context.Context, trait *entities.PersonalityTrait) ([]float32, error) {
	if trait == nil {
		return nil, fmt.Errorf("trait is nil")
	}

	// Dimensions (index → IdentityDimensionVector field):
	//  0 openness  1 conscientiousness  2 extraversion  3 agreeableness
	//  4 emotional_stability  5 voice_formality  6 voice_humor
	//  7 voice_empathy  8 technical_depth  9 directness
	// 10 helpfulness  11 curiosity  12 creativity
	raw := make([]float64, Dimension)
	w := trait.Intensity * trait.Confidence // weighted activation value

	switch trait.Category {
	case entities.TraitCognitive:
		raw[0] = w  // openness
		raw[1] = w  // conscientiousness
		raw[11] = w // curiosity
	case entities.TraitEmotional:
		raw[3] = w // agreeableness
		raw[4] = w // emotional_stability
		raw[7] = w // voice_empathy
	case entities.TraitSocial:
		raw[2] = w // extraversion
		raw[3] = w // agreeableness
		raw[6] = w // voice_humor
	case entities.TraitEpistemic:
		raw[0] = w  // openness
		raw[1] = w  // conscientiousness
		raw[11] = w // curiosity
	case entities.TraitExpressive:
		raw[0] = w  // openness
		raw[2] = w  // extraversion
		raw[6] = w  // voice_humor
		raw[12] = w // creativity
	case entities.TraitEthical:
		raw[1] = w  // conscientiousness
		raw[3] = w  // agreeableness
		raw[10] = w // helpfulness
	default:
		// Distribute the activation uniformly across all dimensions.
		share := w / float64(Dimension)
		for i := range raw {
			raw[i] = share
		}
	}

	return toFloat32(normalize(raw)), nil
}

// FindSimilarIdentities performs a linear-scan cosine similarity search over
// the latest snapshot of every known agent.
// Returns up to limit snapshots sorted by descending similarity.
func (e *SoulEmbedderService) FindSimilarIdentities(ctx context.Context, vector []float32, limit int) ([]*entities.IdentitySnapshot, error) {
	if e.storage == nil {
		return nil, fmt.Errorf("no storage configured: cannot perform similarity search")
	}
	if len(vector) != Dimension {
		return nil, fmt.Errorf("vector dimension mismatch: got %d, expected %d", len(vector), Dimension)
	}
	if limit <= 0 {
		limit = 10
	}

	agents, err := e.storage.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	type candidate struct {
		snap *entities.IdentitySnapshot
		sim  float64
	}

	queryVec := toFloat64(vector)
	candidates := make([]candidate, 0, len(agents))

	for _, agentID := range agents {
		snap, err := e.storage.GetLatestIdentity(ctx, agentID)
		if err != nil || snap == nil {
			continue
		}
		enc, err := e.EncodeIdentity(ctx, snap)
		if err != nil {
			continue
		}
		sim := cosineSimilarity(queryVec, toFloat64(enc))
		candidates = append(candidates, candidate{snap, sim})
	}

	// Insertion-sort descending by similarity (candidate sets are small in practice).
	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].sim > candidates[j-1].sim; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}

	result := make([]*entities.IdentitySnapshot, 0, limit)
	for i, c := range candidates {
		if i >= limit {
			break
		}
		result = append(result, c.snap)
	}
	return result, nil
}

// ModelHash returns the embedding model identifier.
func (e *SoulEmbedderService) ModelHash() string { return ModelHash }

// Dimension returns the vector length.
func (e *SoulEmbedderService) Dimension() int { return Dimension }

// --- internal helpers ---

// normalize returns the L2-normalised copy of v.
func normalize(v []float64) []float64 {
	norm := 0.0
	for _, x := range v {
		norm += x * x
	}
	if norm == 0 {
		return v
	}
	norm = math.Sqrt(norm)
	result := make([]float64, len(v))
	for i, x := range v {
		result[i] = x / norm
	}
	return result
}

func toFloat32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(x)
	}
	return out
}

func toFloat64(v []float32) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = float64(x)
	}
	return out
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}
	dot, normA, normB := 0.0, 0.0, 0.0
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
