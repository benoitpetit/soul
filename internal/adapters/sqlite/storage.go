// Package sqlite implémente le stockage SOUL via SQLite
// Réutilise le même mécanisme de stockage que MIRA pour une intégration native.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SoulSQLiteStorage implémente ports.SoulStorage avec SQLite
// Partage la même base de données que MIRA (soul_ tables dans le .mira/)
type SoulSQLiteStorage struct {
	db     *sql.DB
	dbPath string
	ownsDB bool // true if this storage owns the *sql.DB and must close it
}

// NewSoulSQLiteStorage crée un nouveau stockage SQLite
func NewSoulSQLiteStorage(dbPath string) (*SoulSQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	storage := &SoulSQLiteStorage{
		db:     db,
		dbPath: dbPath,
		ownsDB: true,
	}
	
	// Initialiser le schéma
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}
	
	return storage, nil
}

// NewSoulSQLiteStorageFromDB crée un stockage SQLite à partir d'une connexion existante.
// La connexion n'est PAS fermée par Close() — l'appelant reste responsable du cycle de vie.
// Utilisé quand SOUL est embarqué dans MIRA et partage sa connexion *sql.DB.
func NewSoulSQLiteStorageFromDB(db *sql.DB) (*SoulSQLiteStorage, error) {
	storage := &SoulSQLiteStorage{
		db:     db,
		dbPath: "",
		ownsDB: false,
	}
	
	// Initialiser le schéma SOUL dans la base existante
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}
	
	return storage, nil
}

// initSchema crée les tables SOUL dans la base SQLite
// Les tables SOUL coexistent avec les tables MIRA dans la même base
func (s *SoulSQLiteStorage) initSchema() error {
	schema := `
	-- Table principale des snapshots d'identité
	CREATE TABLE IF NOT EXISTS soul_identities (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL,
		derived_from_id TEXT,
		personality_traits TEXT NOT NULL, -- JSON
		voice_profile TEXT NOT NULL,      -- JSON
		communication_style TEXT NOT NULL, -- JSON
		behavioral_signature TEXT NOT NULL, -- JSON
		value_system TEXT NOT NULL,        -- JSON
		emotional_tone TEXT NOT NULL,      -- JSON
		behavioral_metrics TEXT,           -- JSON (optional)
		confidence_score REAL NOT NULL DEFAULT 0,
		model_identifier TEXT,
		source_memories_count INTEGER DEFAULT 0,
		linked_mira_memories TEXT,         -- JSON array of UUIDs
		FOREIGN KEY (derived_from_id) REFERENCES soul_identities(id)
	);
	
	-- Index pour recherche rapide par agent
	CREATE INDEX IF NOT EXISTS idx_soul_identities_agent ON soul_identities(agent_id);
	CREATE INDEX IF NOT EXISTS idx_soul_identities_version ON soul_identities(agent_id, version);
	CREATE INDEX IF NOT EXISTS idx_soul_identities_created ON soul_identities(created_at);
	
	-- Table des traits de personnalité
	CREATE TABLE IF NOT EXISTS soul_traits (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		intensity REAL NOT NULL DEFAULT 0.5,
		confidence REAL NOT NULL DEFAULT 0.3,
		evidence_count INTEGER DEFAULT 1,
		first_observed DATETIME NOT NULL,
		last_observed DATETIME NOT NULL,
		last_evidence TEXT,
		contexts TEXT, -- JSON array
		consistency REAL DEFAULT 0.5,
		UNIQUE(agent_id, name)
	);
	
	CREATE INDEX IF NOT EXISTS idx_soul_traits_agent ON soul_traits(agent_id);
	CREATE INDEX IF NOT EXISTS idx_soul_traits_confidence ON soul_traits(agent_id, confidence);
	
	-- Table des observations brutes
	CREATE TABLE IF NOT EXISTS soul_observations (
		id TEXT PRIMARY KEY,
		agent_id TEXT NOT NULL,
		trait_name TEXT NOT NULL,
		category TEXT NOT NULL,
		evidence TEXT,
		context TEXT,
		intensity REAL DEFAULT 0.5,
		observed_at DATETIME NOT NULL,
		source_type TEXT,
		source_memory_id TEXT
	);
	
	CREATE INDEX IF NOT EXISTS idx_soul_observations_agent ON soul_observations(agent_id);
	CREATE INDEX IF NOT EXISTS idx_soul_observations_trait ON soul_observations(agent_id, trait_name);
	CREATE INDEX IF NOT EXISTS idx_soul_observations_time ON soul_observations(observed_at);
	
	-- Table des diffs d'évolution
	CREATE TABLE IF NOT EXISTS soul_diffs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id TEXT NOT NULL,
		from_version INTEGER NOT NULL,
		to_version INTEGER NOT NULL,
		timestamp DATETIME NOT NULL,
		added_traits TEXT,       -- JSON
		removed_traits TEXT,     -- JSON
		strengthened_traits TEXT,-- JSON
		weakened_traits TEXT,    -- JSON
		voice_changes TEXT,      -- JSON array
		style_changes TEXT,      -- JSON array
		value_changes TEXT,      -- JSON array
		overall_drift REAL NOT NULL DEFAULT 0
	);
	
	CREATE INDEX IF NOT EXISTS idx_soul_diffs_agent ON soul_diffs(agent_id);
	CREATE INDEX IF NOT EXISTS idx_soul_diffs_time ON soul_diffs(timestamp);
	
	-- Table des changements de modèle
	CREATE TABLE IF NOT EXISTS soul_model_swaps (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		agent_id TEXT NOT NULL,
		previous_model TEXT NOT NULL,
		new_model TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		identity_preserved BOOLEAN DEFAULT FALSE,
		identity_drift REAL DEFAULT 0,
		reinforcement_applied BOOLEAN DEFAULT FALSE
	);
	
	CREATE INDEX IF NOT EXISTS idx_soul_swaps_agent ON soul_model_swaps(agent_id);
	
	-- Table des liens identité-mémoire MIRA
	CREATE TABLE IF NOT EXISTS soul_mira_links (
		identity_id TEXT NOT NULL,
		memory_id TEXT NOT NULL,
		linked_at DATETIME NOT NULL,
		PRIMARY KEY (identity_id, memory_id)
	);
	
	CREATE INDEX IF NOT EXISTS idx_soul_mira_links_identity ON soul_mira_links(identity_id);
	`
	
	_, err := s.db.Exec(schema)
	return err
}

// Close ferme la connexion à la base, seulement si ce storage en est propriétaire.
func (s *SoulSQLiteStorage) Close() error {
	if s.ownsDB {
		return s.db.Close()
	}
	return nil
}

// --- Implémentation IdentityRepository ---

func (s *SoulSQLiteStorage) StoreIdentity(ctx context.Context, identity *entities.IdentitySnapshot) error {
	traitsJSON, _ := json.Marshal(identity.PersonalityTraits)
	voiceJSON, _ := json.Marshal(identity.VoiceProfile)
	commJSON, _ := json.Marshal(identity.CommunicationStyle)
	behaviorJSON, _ := json.Marshal(identity.BehavioralSignature)
	valuesJSON, _ := json.Marshal(identity.ValueSystem)
	emotionsJSON, _ := json.Marshal(identity.EmotionalTone)
	miraMemoriesJSON, _ := json.Marshal(identity.LinkedMiraMemories)
	behavioralMetricsJSON, _ := json.Marshal(identity.BehavioralMetrics)
	
	var derivedFrom interface{}
	if identity.DerivedFromID != nil {
		derivedFrom = identity.DerivedFromID.String()
	} else {
		derivedFrom = nil
	}
	
	query := `
		INSERT INTO soul_identities 
		(id, agent_id, version, created_at, derived_from_id, personality_traits, 
		 voice_profile, communication_style, behavioral_signature, value_system,
		 emotional_tone, behavioral_metrics, confidence_score, model_identifier, source_memories_count, linked_mira_memories)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		identity.ID.String(), identity.AgentID, identity.Version, identity.CreatedAt,
		derivedFrom, string(traitsJSON), string(voiceJSON), string(commJSON),
		string(behaviorJSON), string(valuesJSON), string(emotionsJSON),
		string(behavioralMetricsJSON),
		identity.ConfidenceScore, identity.ModelIdentifier,
		identity.SourceMemoriesCount, string(miraMemoriesJSON),
	)
	
	return err
}

func (s *SoulSQLiteStorage) GetLatestIdentity(ctx context.Context, agentID string) (*entities.IdentitySnapshot, error) {
	query := `
		SELECT id, agent_id, version, created_at, derived_from_id, personality_traits,
		       voice_profile, communication_style, behavioral_signature, value_system,
		       emotional_tone, behavioral_metrics, confidence_score, model_identifier, source_memories_count, linked_mira_memories
		FROM soul_identities 
		WHERE agent_id = ? 
		ORDER BY version DESC, created_at DESC 
		LIMIT 1
	`
	
	return s.scanIdentity(ctx, query, agentID)
}

func (s *SoulSQLiteStorage) GetIdentityByID(ctx context.Context, id uuid.UUID) (*entities.IdentitySnapshot, error) {
	query := `
		SELECT id, agent_id, version, created_at, derived_from_id, personality_traits,
		       voice_profile, communication_style, behavioral_signature, value_system,
		       emotional_tone, behavioral_metrics, confidence_score, model_identifier, source_memories_count, linked_mira_memories
		FROM soul_identities 
		WHERE id = ?
	`
	
	return s.scanIdentity(ctx, query, id.String())
}

func (s *SoulSQLiteStorage) GetIdentityHistory(ctx context.Context, agentID string, limit int) ([]*entities.IdentitySnapshot, error) {
	query := `
		SELECT id, agent_id, version, created_at, derived_from_id, personality_traits,
		       voice_profile, communication_style, behavioral_signature, value_system,
		       emotional_tone, behavioral_metrics, confidence_score, model_identifier, source_memories_count, linked_mira_memories
		FROM soul_identities 
		WHERE agent_id = ? 
		ORDER BY version DESC
		LIMIT ?
	`
	
	rows, err := s.db.QueryContext(ctx, query, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var identities []*entities.IdentitySnapshot
	for rows.Next() {
		identity, err := s.scanIdentityRow(rows)
		if err != nil {
			return nil, err
		}
		identities = append(identities, identity)
	}
	
	return identities, rows.Err()
}

func (s *SoulSQLiteStorage) GetIdentityAtVersion(ctx context.Context, agentID string, version int) (*entities.IdentitySnapshot, error) {
	query := `
		SELECT id, agent_id, version, created_at, derived_from_id, personality_traits,
		       voice_profile, communication_style, behavioral_signature, value_system,
		       emotional_tone, behavioral_metrics, confidence_score, model_identifier, source_memories_count, linked_mira_memories
		FROM soul_identities 
		WHERE agent_id = ? AND version = ?
	`
	
	return s.scanIdentity(ctx, query, agentID, version)
}

func (s *SoulSQLiteStorage) DeleteIdentity(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM soul_identities WHERE id = ?", id.String())
	return err
}

func (s *SoulSQLiteStorage) ListAgents(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT DISTINCT agent_id FROM soul_identities")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var agents []string
	for rows.Next() {
		var agentID string
		if err := rows.Scan(&agentID); err != nil {
			return nil, err
		}
		agents = append(agents, agentID)
	}
	
	return agents, rows.Err()
}

func (s *SoulSQLiteStorage) GetIdentityLineage(ctx context.Context, snapshotID uuid.UUID) (*ports.IdentityLineage, error) {
	// Récupérer tous les ancêtres
	var snapshots []*entities.IdentitySnapshot
	currentID := snapshotID
	
	for currentID != uuid.Nil {
		snapshot, err := s.GetIdentityByID(ctx, currentID)
		if err != nil {
			break
		}
		snapshots = append([]*entities.IdentitySnapshot{snapshot}, snapshots...)
		
		if snapshot.DerivedFromID != nil {
			currentID = *snapshot.DerivedFromID
		} else {
			break
		}
	}
	
	lineage := &ports.IdentityLineage{
		Snapshots: snapshots,
		Depth:     len(snapshots),
	}
	
	if len(snapshots) > 0 {
		lineage.Root = snapshots[0]
	}
	
	return lineage, nil
}

// --- Implémentation TraitRepository ---

func (s *SoulSQLiteStorage) StoreTrait(ctx context.Context, trait *entities.PersonalityTrait) error {
	contextsJSON, _ := json.Marshal(trait.Contexts)
	
	query := `
		INSERT INTO soul_traits 
		(id, agent_id, name, category, intensity, confidence, evidence_count,
		 first_observed, last_observed, last_evidence, contexts, consistency)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		trait.ID.String(), trait.AgentID, trait.Name, string(trait.Category),
		trait.Intensity, trait.Confidence, trait.EvidenceCount,
		trait.FirstObserved, trait.LastObserved, trait.LastEvidence,
		string(contextsJSON), trait.Consistency,
	)
	
	return err
}

func (s *SoulSQLiteStorage) GetTraitByName(ctx context.Context, agentID, name string) (*entities.PersonalityTrait, error) {
	query := `
		SELECT id, agent_id, name, category, intensity, confidence, evidence_count,
		       first_observed, last_observed, last_evidence, contexts, consistency
		FROM soul_traits 
		WHERE agent_id = ? AND name = ?
	`
	
	return s.scanTrait(ctx, query, agentID, name)
}

func (s *SoulSQLiteStorage) GetTraitsByNames(ctx context.Context, agentID string, names []string) ([]*entities.PersonalityTrait, error) {
	if len(names) == 0 {
		return []*entities.PersonalityTrait{}, nil
	}
	// Build IN clause with placeholders
	placeholders := make([]string, len(names))
	args := make([]interface{}, 0, len(names)+1)
	args = append(args, agentID)
	for i, name := range names {
		placeholders[i] = "?"
		args = append(args, name)
	}
	query := `
		SELECT id, agent_id, name, category, intensity, confidence, evidence_count,
		       first_observed, last_observed, last_evidence, contexts, consistency
		FROM soul_traits 
		WHERE agent_id = ? AND name IN (` + fmt.Sprintf("%s", placeholders) + `)
	`
	// Need to join placeholders
	query = fmt.Sprintf(`
		SELECT id, agent_id, name, category, intensity, confidence, evidence_count,
		       first_observed, last_observed, last_evidence, contexts, consistency
		FROM soul_traits 
		WHERE agent_id = ? AND name IN (%s)
	`, strings.Join(placeholders, ", "))
	return s.scanTraits(ctx, query, args...)
}

func (s *SoulSQLiteStorage) GetAllTraits(ctx context.Context, agentID string) ([]*entities.PersonalityTrait, error) {
	query := `
		SELECT id, agent_id, name, category, intensity, confidence, evidence_count,
		       first_observed, last_observed, last_evidence, contexts, consistency
		FROM soul_traits 
		WHERE agent_id = ? 
		ORDER BY confidence DESC
	`
	
	return s.scanTraits(ctx, query, agentID)
}

func (s *SoulSQLiteStorage) GetTraitsByCategory(ctx context.Context, agentID string, category entities.TraitCategory) ([]*entities.PersonalityTrait, error) {
	query := `
		SELECT id, agent_id, name, category, intensity, confidence, evidence_count,
		       first_observed, last_observed, last_evidence, contexts, consistency
		FROM soul_traits 
		WHERE agent_id = ? AND category = ?
		ORDER BY confidence DESC
	`
	
	return s.scanTraits(ctx, query, agentID, string(category))
}

func (s *SoulSQLiteStorage) UpdateTrait(ctx context.Context, trait *entities.PersonalityTrait) error {
	contextsJSON, _ := json.Marshal(trait.Contexts)
	
	query := `
		UPDATE soul_traits SET
			intensity = ?, confidence = ?, evidence_count = ?,
			last_observed = ?, last_evidence = ?, contexts = ?, consistency = ?
		WHERE id = ?
	`
	
	_, err := s.db.ExecContext(ctx, query,
		trait.Intensity, trait.Confidence, trait.EvidenceCount,
		trait.LastObserved, trait.LastEvidence, string(contextsJSON), trait.Consistency,
		trait.ID.String(),
	)
	
	return err
}

func (s *SoulSQLiteStorage) UpsertTraits(ctx context.Context, agentID string, traits []*entities.PersonalityTrait) error {
	if len(traits) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin upsert transaction: %w", err)
	}
	defer tx.Rollback()

	insertStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO soul_traits 
		(id, agent_id, name, category, intensity, confidence, evidence_count,
		 first_observed, last_observed, last_evidence, contexts, consistency)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			intensity = excluded.intensity,
			confidence = excluded.confidence,
			evidence_count = excluded.evidence_count,
			last_observed = excluded.last_observed,
			last_evidence = excluded.last_evidence,
			contexts = excluded.contexts,
			consistency = excluded.consistency
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare upsert statement: %w", err)
	}
	defer insertStmt.Close()

	for _, trait := range traits {
		contextsJSON, _ := json.Marshal(trait.Contexts)
		_, err := insertStmt.ExecContext(ctx,
			trait.ID.String(), agentID, trait.Name, string(trait.Category),
			trait.Intensity, trait.Confidence, trait.EvidenceCount,
			trait.FirstObserved, trait.LastObserved, trait.LastEvidence,
			string(contextsJSON), trait.Consistency,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert trait %s: %w", trait.Name, err)
		}
	}

	return tx.Commit()
}

func (s *SoulSQLiteStorage) GetWellEstablishedTraits(ctx context.Context, agentID string, minConfidence float64) ([]*entities.PersonalityTrait, error) {
	query := `
		SELECT id, agent_id, name, category, intensity, confidence, evidence_count,
		       first_observed, last_observed, last_evidence, contexts, consistency
		FROM soul_traits 
		WHERE agent_id = ? AND confidence >= ?
		ORDER BY confidence DESC
	`
	
	return s.scanTraits(ctx, query, agentID, minConfidence)
}

func (s *SoulSQLiteStorage) DeleteTrait(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM soul_traits WHERE id = ?", id.String())
	return err
}

// --- Implémentation TraitObservationRepository ---

func (s *SoulSQLiteStorage) StoreObservation(ctx context.Context, obs *entities.TraitObservation) error {
	var memoryID interface{}
	if obs.SourceMemory != uuid.Nil {
		memoryID = obs.SourceMemory.String()
	}
	
	query := `
		INSERT INTO soul_observations 
		(id, agent_id, trait_name, category, evidence, context, intensity, observed_at, source_type, source_memory_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		obs.ID.String(), obs.AgentID, obs.TraitName, string(obs.Category),
		obs.Evidence, obs.Context, obs.Intensity, obs.ObservedAt,
		string(obs.SourceType), memoryID,
	)
	
	return err
}

func (s *SoulSQLiteStorage) GetObservationsForTrait(ctx context.Context, agentID, traitName string, limit int) ([]*entities.TraitObservation, error) {
	query := `
		SELECT id, agent_id, trait_name, category, evidence, context, intensity, observed_at, source_type, source_memory_id
		FROM soul_observations 
		WHERE agent_id = ? AND trait_name = ?
		ORDER BY observed_at DESC
		LIMIT ?
	`
	
	return s.scanObservations(ctx, query, agentID, traitName, limit)
}

func (s *SoulSQLiteStorage) GetRecentObservations(ctx context.Context, agentID string, since time.Time) ([]*entities.TraitObservation, error) {
	query := `
		SELECT id, agent_id, trait_name, category, evidence, context, intensity, observed_at, source_type, source_memory_id
		FROM soul_observations 
		WHERE agent_id = ? AND observed_at > ?
		ORDER BY observed_at DESC
	`
	
	return s.scanObservations(ctx, query, agentID, since)
}

func (s *SoulSQLiteStorage) GetObservationsBySource(ctx context.Context, agentID string, sourceType valueobjects.SourceType) ([]*entities.TraitObservation, error) {
	query := `
		SELECT id, agent_id, trait_name, category, evidence, context, intensity, observed_at, source_type, source_memory_id
		FROM soul_observations 
		WHERE agent_id = ? AND source_type = ?
		ORDER BY observed_at DESC
	`
	
	return s.scanObservations(ctx, query, agentID, string(sourceType))
}

func (s *SoulSQLiteStorage) DeleteOldObservations(ctx context.Context, agentID string, before time.Time) (int, error) {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM soul_observations WHERE agent_id = ? AND observed_at < ?",
		agentID, before,
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// --- Implémentation EvolutionRepository ---

func (s *SoulSQLiteStorage) RecordDiff(ctx context.Context, diff *entities.IdentityDiff) error {
	addedJSON, _ := json.Marshal(diff.AddedTraits)
	removedJSON, _ := json.Marshal(diff.RemovedTraits)
	strengthenedJSON, _ := json.Marshal(diff.StrengthenedTraits)
	weakenedJSON, _ := json.Marshal(diff.WeakenedTraits)
	voiceJSON, _ := json.Marshal(diff.VoiceChanges)
	styleJSON, _ := json.Marshal(diff.StyleChanges)
	valueJSON, _ := json.Marshal(diff.ValueChanges)
	
	query := `
		INSERT INTO soul_diffs 
		(agent_id, from_version, to_version, timestamp, added_traits, removed_traits,
		 strengthened_traits, weakened_traits, voice_changes, style_changes, value_changes, overall_drift)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		diff.AgentID, diff.FromVersion, diff.ToVersion, diff.Timestamp,
		string(addedJSON), string(removedJSON), string(strengthenedJSON), string(weakenedJSON),
		string(voiceJSON), string(styleJSON), string(valueJSON), diff.OverallDrift,
	)
	
	return err
}

func (s *SoulSQLiteStorage) GetDiffsForAgent(ctx context.Context, agentID string, limit int) ([]*entities.IdentityDiff, error) {
	query := `
		SELECT agent_id, from_version, to_version, timestamp, added_traits, removed_traits,
		       strengthened_traits, weakened_traits, voice_changes, style_changes, value_changes, overall_drift
		FROM soul_diffs 
		WHERE agent_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`
	
	rows, err := s.db.QueryContext(ctx, query, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var diffs []*entities.IdentityDiff
	for rows.Next() {
		diff := &entities.IdentityDiff{}
		var addedJSON, removedJSON, strengthenedJSON, weakenedJSON, voiceJSON, styleJSON, valueJSON string
		
		err := rows.Scan(
			&diff.AgentID, &diff.FromVersion, &diff.ToVersion, &diff.Timestamp,
			&addedJSON, &removedJSON, &strengthenedJSON, &weakenedJSON,
			&voiceJSON, &styleJSON, &valueJSON, &diff.OverallDrift,
		)
		if err != nil {
			return nil, err
		}
		
		json.Unmarshal([]byte(addedJSON), &diff.AddedTraits)
		json.Unmarshal([]byte(removedJSON), &diff.RemovedTraits)
		json.Unmarshal([]byte(strengthenedJSON), &diff.StrengthenedTraits)
		json.Unmarshal([]byte(weakenedJSON), &diff.WeakenedTraits)
		json.Unmarshal([]byte(voiceJSON), &diff.VoiceChanges)
		json.Unmarshal([]byte(styleJSON), &diff.StyleChanges)
		json.Unmarshal([]byte(valueJSON), &diff.ValueChanges)
		
		diffs = append(diffs, diff)
	}
	
	return diffs, rows.Err()
}

func (s *SoulSQLiteStorage) GetLatestDiff(ctx context.Context, agentID string) (*entities.IdentityDiff, error) {
	query := `
		SELECT agent_id, from_version, to_version, timestamp, added_traits, removed_traits,
		       strengthened_traits, weakened_traits, voice_changes, style_changes, value_changes, overall_drift
		FROM soul_diffs 
		WHERE agent_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`
	
	rows, err := s.db.QueryContext(ctx, query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	if !rows.Next() {
		return nil, nil
	}
	
	diff := &entities.IdentityDiff{}
	var addedJSON, removedJSON, strengthenedJSON, weakenedJSON, voiceJSON, styleJSON, valueJSON string
	
	err = rows.Scan(
		&diff.AgentID, &diff.FromVersion, &diff.ToVersion, &diff.Timestamp,
		&addedJSON, &removedJSON, &strengthenedJSON, &weakenedJSON,
		&voiceJSON, &styleJSON, &valueJSON, &diff.OverallDrift,
	)
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal([]byte(addedJSON), &diff.AddedTraits)
	json.Unmarshal([]byte(removedJSON), &diff.RemovedTraits)
	json.Unmarshal([]byte(strengthenedJSON), &diff.StrengthenedTraits)
	json.Unmarshal([]byte(weakenedJSON), &diff.WeakenedTraits)
	json.Unmarshal([]byte(voiceJSON), &diff.VoiceChanges)
	json.Unmarshal([]byte(styleJSON), &diff.StyleChanges)
	json.Unmarshal([]byte(valueJSON), &diff.ValueChanges)
	
	return diff, nil
}

func (s *SoulSQLiteStorage) GetDriftReport(ctx context.Context, agentID string, windowSize int) (*valueobjects.IdentityDriftReport, error) {
	diffs, err := s.GetDiffsForAgent(ctx, agentID, windowSize)
	if err != nil {
		return nil, err
	}

	if len(diffs) == 0 {
		return &valueobjects.IdentityDriftReport{
			Timestamp:     time.Now(),
			DriftScore:    0,
			IsSignificant: false,
		}, nil
	}

	// Calculer le score de dérive moyen
	totalDrift := 0.0
	for _, diff := range diffs {
		totalDrift += diff.OverallDrift
	}
	avgDrift := totalDrift / float64(len(diffs))

	report := &valueobjects.IdentityDriftReport{
		Timestamp:       time.Now(),
		PreviousVersion: diffs[len(diffs)-1].FromVersion,
		CurrentVersion:  diffs[0].ToVersion,
		DriftScore:      avgDrift,
		IsSignificant:   avgDrift > 0.3,
		DriftDimensions: make([]valueobjects.DimensionDrift, 0),
		Recommendations: make([]string, 0),
	}

	// Aggregating dimension drift from all diffs
	dimensionDrifts := make(map[string]float64)
	dimensionCounts := make(map[string]int)

	for _, diff := range diffs {
		// Voice changes
		if len(diff.VoiceChanges) > 0 {
			dimensionDrifts["voice_profile"] += diff.OverallDrift
			dimensionCounts["voice_profile"]++
		}
		// Style changes
		if len(diff.StyleChanges) > 0 {
			dimensionDrifts["communication_style"] += diff.OverallDrift
			dimensionCounts["communication_style"]++
		}
		// Value changes
		if len(diff.ValueChanges) > 0 {
			dimensionDrifts["value_system"] += diff.OverallDrift
			dimensionCounts["value_system"]++
		}
		// Trait changes contribute to personality drift
		if len(diff.AddedTraits) > 0 || len(diff.RemovedTraits) > 0 || len(diff.StrengthenedTraits) > 0 || len(diff.WeakenedTraits) > 0 {
			dimensionDrifts["personality_traits"] += diff.OverallDrift
			dimensionCounts["personality_traits"]++
		}
	}

	// Build DimensionDrift entries with averaged values
	for dim, total := range dimensionDrifts {
		count := dimensionCounts[dim]
		avg := total / float64(count)
		report.DriftDimensions = append(report.DriftDimensions, valueobjects.DimensionDrift{
			Dimension:     dim,
			PreviousValue: 0.5,
			CurrentValue:  0.5,
			Change:        avg,
			IsSignificant: avg > 0.3,
		})
	}

	if report.IsSignificant {
		report.Recommendations = append(report.Recommendations,
			"Identity drift detected. Consider reinforcing identity.",
			"Review recent conversations for consistency.",
		)
	}

	return report, nil
}

// --- Implémentation ModelSwapRepository ---

func (s *SoulSQLiteStorage) RecordModelSwap(ctx context.Context, swap *valueobjects.ModelSwapContext) error {
	query := `
		INSERT INTO soul_model_swaps 
		(agent_id, previous_model, new_model, timestamp, identity_preserved, identity_drift, reinforcement_applied)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		swap.AgentID, swap.PreviousModel, swap.NewModel, swap.Timestamp,
		swap.IdentityPreserved, swap.IdentityDrift, swap.ReinforcementApplied,
	)
	
	return err
}

func (s *SoulSQLiteStorage) GetModelSwaps(ctx context.Context, agentID string) ([]*valueobjects.ModelSwapContext, error) {
	query := `
		SELECT agent_id, previous_model, new_model, timestamp, identity_preserved, identity_drift, reinforcement_applied
		FROM soul_model_swaps 
		WHERE agent_id = ?
		ORDER BY timestamp DESC
	`
	
	rows, err := s.db.QueryContext(ctx, query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var swaps []*valueobjects.ModelSwapContext
	for rows.Next() {
		swap := &valueobjects.ModelSwapContext{}
		err := rows.Scan(
			&swap.AgentID, &swap.PreviousModel, &swap.NewModel, &swap.Timestamp,
			&swap.IdentityPreserved, &swap.IdentityDrift, &swap.ReinforcementApplied,
		)
		if err != nil {
			return nil, err
		}
		swaps = append(swaps, swap)
	}
	
	return swaps, rows.Err()
}

func (s *SoulSQLiteStorage) GetLatestModelSwap(ctx context.Context, agentID string) (*valueobjects.ModelSwapContext, error) {
	query := `
		SELECT agent_id, previous_model, new_model, timestamp, identity_preserved, identity_drift, reinforcement_applied
		FROM soul_model_swaps 
		WHERE agent_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`
	
	row := s.db.QueryRowContext(ctx, query, agentID)
	swap := &valueobjects.ModelSwapContext{}
	err := row.Scan(
		&swap.AgentID, &swap.PreviousModel, &swap.NewModel, &swap.Timestamp,
		&swap.IdentityPreserved, &swap.IdentityDrift, &swap.ReinforcementApplied,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return swap, nil
}

// --- Implémentation MiraBridgeRepository ---

func (s *SoulSQLiteStorage) GetMiraMemories(ctx context.Context, agentID, query string, limit int) ([]ports.MiraMemoryReference, error) {
	// Cette méthode interroge les tables de MIRA directement
	// Elle suppose que MIRA et SOUL partagent la même base SQLite
	// Tables MIRA réelles: verbatim, fingerprints (colonne ftype, pas type)
	
	sqlQuery := `
		SELECT v.id, f.data, f.ftype, v.created_at, v.wing, v.room
		FROM verbatim v
		JOIN fingerprints f ON f.verbatim_id = v.id
		WHERE v.content LIKE ? OR f.data LIKE ?
		ORDER BY v.created_at DESC
		LIMIT ?
	`
	
	searchPattern := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, sqlQuery, searchPattern, searchPattern, limit)
	if err != nil {
		// Si les tables MIRA n'existent pas, retourner vide sans erreur
		return []ports.MiraMemoryReference{}, nil
	}
	defer rows.Close()
	
	var memories []ports.MiraMemoryReference
	for rows.Next() {
		mem := ports.MiraMemoryReference{}
		var idStr string
		err := rows.Scan(&idStr, &mem.Content, &mem.MemoryType, &mem.Timestamp, &mem.Wing, &mem.Room)
		if err != nil {
			continue
		}
		mem.MemoryID = uuid.MustParse(idStr)
		mem.Relevance = 0.8 // Score par défaut
		memories = append(memories, mem)
	}
	
	return memories, rows.Err()
}

func (s *SoulSQLiteStorage) LinkIdentityToMemory(ctx context.Context, identityID, memoryID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO soul_mira_links (identity_id, memory_id, linked_at) VALUES (?, ?, ?)",
		identityID.String(), memoryID.String(), time.Now(),
	)
	return err
}

func (s *SoulSQLiteStorage) GetLinkedMemories(ctx context.Context, identityID uuid.UUID) ([]ports.MiraMemoryReference, error) {
	// JOIN avec les tables MIRA réelles pour récupérer contenu + type.
	// Si les tables MIRA n'existent pas encore, on retombe sur la requête basique.
	query := `
		SELECT l.memory_id, COALESCE(v.content, ''), COALESCE(f.ftype, ''), l.linked_at, COALESCE(v.wing, ''), v.room
		FROM soul_mira_links l
		LEFT JOIN verbatim v ON v.id = l.memory_id
		LEFT JOIN fingerprints f ON f.verbatim_id = l.memory_id
		WHERE l.identity_id = ?
	`

	rows, err := s.db.QueryContext(ctx, query, identityID.String())
	if err != nil {
		// Tables MIRA absentes (ex: tests isolés) → requête de secours sans JOIN
		fallback := `
			SELECT memory_id, '' as content, '' as ftype, linked_at, '' as wing, NULL as room
			FROM soul_mira_links
			WHERE identity_id = ?
		`
		rows, err = s.db.QueryContext(ctx, fallback, identityID.String())
		if err != nil {
			return nil, err
		}
	}
	defer rows.Close()

	var memories []ports.MiraMemoryReference
	for rows.Next() {
		mem := ports.MiraMemoryReference{}
		var idStr string
		err := rows.Scan(&idStr, &mem.Content, &mem.MemoryType, &mem.Timestamp, &mem.Wing, &mem.Room)
		if err != nil {
			continue
		}
		mem.MemoryID = uuid.MustParse(idStr)
		memories = append(memories, mem)
	}

	return memories, rows.Err()
}

func (s *SoulSQLiteStorage) NotifyMiraOfIdentityChange(ctx context.Context, agentID string, changeType string) error {
	// Insérer une mémoire dans MIRA pour documenter le changement d'identité
	// Utilise la vraie table MIRA: verbatim (pas mira_verbatims)
	content := fmt.Sprintf("Identity change detected: %s for agent %s at %s", 
		changeType, agentID, time.Now().Format(time.RFC3339))
	
	_, _ = s.db.ExecContext(ctx,
		"INSERT OR IGNORE INTO verbatim (id, content, created_at, wing) VALUES (?, ?, ?, ?)",
		uuid.New().String(), content, time.Now(), "soul_identity",
	)
	
	// Si la table n'existe pas, ignorer l'erreur
	return nil
}

// --- Transaction support ---

type soulTx struct {
	tx *sql.Tx
}

func (s *SoulSQLiteStorage) BeginTx(ctx context.Context) (ports.SoulTx, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &soulTx{tx: tx}, nil
}

func (t *soulTx) Commit() error {
	return t.tx.Commit()
}

func (t *soulTx) Rollback() error {
	return t.tx.Rollback()
}

// --- Helpers de scan ---

func (s *SoulSQLiteStorage) scanIdentity(ctx context.Context, query string, args ...interface{}) (*entities.IdentitySnapshot, error) {
	row := s.db.QueryRowContext(ctx, query, args...)
	return s.scanIdentityRow(row)
}

func (s *SoulSQLiteStorage) scanIdentityRow(row interface{ // *sql.Row ou *sql.Rows
	Scan(dest ...interface{}) error
}) (*entities.IdentitySnapshot, error) {
	identity := &entities.IdentitySnapshot{}
	var idStr, agentID string
	var derivedFromID interface{}
	var traitsJSON, voiceJSON, commJSON, behaviorJSON, valuesJSON, emotionsJSON, behavioralMetricsJSON, miraMemoriesJSON string
	
	err := row.Scan(
		&idStr, &agentID, &identity.Version, &identity.CreatedAt, &derivedFromID,
		&traitsJSON, &voiceJSON, &commJSON, &behaviorJSON, &valuesJSON, &emotionsJSON,
		&behavioralMetricsJSON,
		&identity.ConfidenceScore, &identity.ModelIdentifier, &identity.SourceMemoriesCount, &miraMemoriesJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	identity.ID = uuid.MustParse(idStr)
	identity.AgentID = agentID
	
	if derivedFromID != nil {
		id := uuid.MustParse(derivedFromID.(string))
		identity.DerivedFromID = &id
	}
	
	json.Unmarshal([]byte(traitsJSON), &identity.PersonalityTraits)
	json.Unmarshal([]byte(voiceJSON), &identity.VoiceProfile)
	json.Unmarshal([]byte(commJSON), &identity.CommunicationStyle)
	json.Unmarshal([]byte(behaviorJSON), &identity.BehavioralSignature)
	json.Unmarshal([]byte(valuesJSON), &identity.ValueSystem)
	json.Unmarshal([]byte(emotionsJSON), &identity.EmotionalTone)
	if behavioralMetricsJSON != "" {
		json.Unmarshal([]byte(behavioralMetricsJSON), &identity.BehavioralMetrics)
	}
	json.Unmarshal([]byte(miraMemoriesJSON), &identity.LinkedMiraMemories)
	
	return identity, nil
}

func (s *SoulSQLiteStorage) scanTrait(ctx context.Context, query string, args ...interface{}) (*entities.PersonalityTrait, error) {
	row := s.db.QueryRowContext(ctx, query, args...)
	return s.scanTraitRow(row)
}

func (s *SoulSQLiteStorage) scanTraitRow(row interface{ // *sql.Row ou *sql.Rows
	Scan(dest ...interface{}) error
}) (*entities.PersonalityTrait, error) {
	trait := &entities.PersonalityTrait{}
	var idStr string
	var contextsJSON string
	var categoryStr string
	
	var agentID string
	err := row.Scan(
		&idStr, &agentID, &trait.Name, &categoryStr, &trait.Intensity, &trait.Confidence,
		&trait.EvidenceCount, &trait.FirstObserved, &trait.LastObserved,
		&trait.LastEvidence, &contextsJSON, &trait.Consistency,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	trait.ID = uuid.MustParse(idStr)
	trait.AgentID = agentID
	trait.Category = entities.TraitCategory(categoryStr)
	json.Unmarshal([]byte(contextsJSON), &trait.Contexts)
	
	return trait, nil
}

func (s *SoulSQLiteStorage) scanTraits(ctx context.Context, query string, args ...interface{}) ([]*entities.PersonalityTrait, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var traits []*entities.PersonalityTrait
	for rows.Next() {
		trait := &entities.PersonalityTrait{}
		var idStr string
		var contextsJSON string
		var categoryStr string
		
		var agentID string
		err := rows.Scan(
			&idStr, &agentID, &trait.Name, &categoryStr, &trait.Intensity, &trait.Confidence,
			&trait.EvidenceCount, &trait.FirstObserved, &trait.LastObserved,
			&trait.LastEvidence, &contextsJSON, &trait.Consistency,
		)
		if err != nil {
			return nil, err
		}
		
		trait.ID = uuid.MustParse(idStr)
		trait.AgentID = agentID
		trait.Category = entities.TraitCategory(categoryStr)
		json.Unmarshal([]byte(contextsJSON), &trait.Contexts)
		traits = append(traits, trait)
	}
	
	return traits, rows.Err()
}

func (s *SoulSQLiteStorage) scanObservations(ctx context.Context, query string, args ...interface{}) ([]*entities.TraitObservation, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var observations []*entities.TraitObservation
	for rows.Next() {
		obs := &entities.TraitObservation{}
		var idStr string
		var memoryIDStr interface{}
		
		err := rows.Scan(
			&idStr, &obs.AgentID, &obs.TraitName, &obs.Category,
			&obs.Evidence, &obs.Context, &obs.Intensity, &obs.ObservedAt,
			&obs.SourceType, &memoryIDStr,
		)
		if err != nil {
			return nil, err
		}
		
		obs.ID = uuid.MustParse(idStr)
		if memoryIDStr != nil {
			obs.SourceMemory = uuid.MustParse(memoryIDStr.(string))
		}
		observations = append(observations, obs)
	}
	
	return observations, rows.Err()
}
