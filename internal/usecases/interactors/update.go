// Package interactors — IdentityUpdateUseCase
//
// Permet la modification intentionnelle de l'âme d'un agent, déclenchée par
// une instruction utilisateur ("réponds avec plus d'enthousiasme") ou un
// patch structuré depuis b0p/cli via soul_update / soul_patch.
//
// Principe fondamental : on ne modifie JAMAIS un snapshot existant.
// Chaque modification crée un nouveau snapshot versionné, dérivé du précédent,
// préservant ainsi l'historique complet des évolutions identitaires.
package interactors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// IdentityUpdateUseCase gère la modification intentionnelle de l'identité.
type IdentityUpdateUseCase struct {
	storage ports.SoulStorage
}

// NewIdentityUpdateUseCase crée un nouveau use case de mise à jour.
func NewIdentityUpdateUseCase(storage ports.SoulStorage) *IdentityUpdateUseCase {
	return &IdentityUpdateUseCase{storage: storage}
}

// UpdateResult décrit les changements appliqués par un patch.
type UpdateResult struct {
	AgentID        string    `json:"agent_id"`
	NewVersion     int       `json:"new_version"`
	ChangesApplied []string  `json:"changes_applied"`
	Timestamp      time.Time `json:"timestamp"`
}

// UpdateFromDirective parse une directive en langage naturel (FR/EN) et applique
// les modifications correspondantes sur le dernier snapshot de l'agent.
//
// Exemples de directives :
//   - "réponds avec plus d'enthousiasme"
//   - "sois plus formel"
//   - "fais un petit rapport à la fin"
//   - "réponds de manière concise"
//   - "utilise plus d'humour"
func (uc *IdentityUpdateUseCase) UpdateFromDirective(ctx context.Context, agentID, directive, reason string) (*entities.IdentitySnapshot, *UpdateResult, error) {
	current, err := uc.loadOrInit(ctx, agentID)
	if err != nil {
		return nil, nil, err
	}

	patch := parseDirective(directive)
	if patch.IsEmpty() {
		return nil, nil, fmt.Errorf("directive non reconnue : aucune modification identifiable dans %q", directive)
	}

	if reason != "" {
		patch.Reason = reason
	} else {
		patch.Reason = fmt.Sprintf("user directive: %s", directive)
	}

	return uc.applyPatch(ctx, agentID, current, patch)
}

// PatchIdentity applique un patch structuré sur le dernier snapshot de l'agent.
// Chaque champ non-nil du patch remplace la valeur correspondante.
func (uc *IdentityUpdateUseCase) PatchIdentity(ctx context.Context, agentID string, patch *valueobjects.IdentityPatch) (*entities.IdentitySnapshot, *UpdateResult, error) {
	if patch == nil || patch.IsEmpty() {
		return nil, nil, fmt.Errorf("le patch est vide : aucune modification à appliquer")
	}

	current, err := uc.loadOrInit(ctx, agentID)
	if err != nil {
		return nil, nil, err
	}

	return uc.applyPatch(ctx, agentID, current, patch)
}

// ── Implémentation interne ────────────────────────────────────────────────────

// loadOrInit récupère le dernier snapshot ou en crée un neutre si l'agent est nouveau.
func (uc *IdentityUpdateUseCase) loadOrInit(ctx context.Context, agentID string) (*entities.IdentitySnapshot, error) {
	current, _ := uc.storage.GetLatestIdentity(ctx, agentID)
	if current == nil {
		// Premier contact : initialiser avec des valeurs neutres
		snap := entities.NewIdentitySnapshot(agentID, "unknown")
		snap.VoiceProfile = *entities.NewVoiceProfile()
		snap.CommunicationStyle = *entities.NewCommunicationStyle()
		snap.EmotionalTone = *entities.NewEmotionalTone()
		snap.BehavioralSignature = *entities.NewBehavioralSignature()
		snap.ValueSystem = *entities.NewValueSystem()
		snap.PersonalityTraits = make([]entities.PersonalityTrait, 0)
		snap.ConfidenceScore = 0.5
		return snap, nil
	}
	return current, nil
}

// applyPatch construit le nouveau snapshot, applique le patch, sauvegarde et retourne.
func (uc *IdentityUpdateUseCase) applyPatch(ctx context.Context, agentID string, current *entities.IdentitySnapshot, patch *valueobjects.IdentityPatch) (*entities.IdentitySnapshot, *UpdateResult, error) {
	// Nouveau snapshot dérivé du courant (immuabilité préservée)
	snap := entities.NewIdentitySnapshot(agentID, current.ModelIdentifier)
	snap.WithParentSnapshot(current.ID)

	// Hériter toutes les dimensions
	snap.VoiceProfile = current.VoiceProfile
	snap.CommunicationStyle = current.CommunicationStyle
	snap.EmotionalTone = current.EmotionalTone
	snap.BehavioralSignature = current.BehavioralSignature
	snap.ValueSystem = current.ValueSystem
	snap.PersonalityTraits = make([]entities.PersonalityTrait, len(current.PersonalityTraits))
	copy(snap.PersonalityTraits, current.PersonalityTraits)
	snap.LinkedMiraMemories = current.LinkedMiraMemories
	snap.SourceMemoriesCount = current.SourceMemoriesCount
	snap.ConfidenceScore = current.ConfidenceScore

	var changes []string

	// ── Profil de voix ────────────────────────────────────────────────────────

	if patch.EnthusiasmLevel != nil {
		old := snap.VoiceProfile.EnthusiasmLevel
		snap.VoiceProfile.EnthusiasmLevel = clampF(*patch.EnthusiasmLevel)
		changes = append(changes, fmt.Sprintf("enthusiasm_level: %.2f → %.2f", old, snap.VoiceProfile.EnthusiasmLevel))
	}
	if patch.FormalityLevel != nil {
		old := snap.VoiceProfile.FormalityLevel
		snap.VoiceProfile.FormalityLevel = clampF(*patch.FormalityLevel)
		changes = append(changes, fmt.Sprintf("formality_level: %.2f → %.2f", old, snap.VoiceProfile.FormalityLevel))
	}
	if patch.HumorLevel != nil {
		old := snap.VoiceProfile.HumorLevel
		snap.VoiceProfile.HumorLevel = clampF(*patch.HumorLevel)
		changes = append(changes, fmt.Sprintf("humor_level: %.2f → %.2f", old, snap.VoiceProfile.HumorLevel))
	}
	if patch.EmpathyLevel != nil {
		old := snap.VoiceProfile.EmpathyLevel
		snap.VoiceProfile.EmpathyLevel = clampF(*patch.EmpathyLevel)
		changes = append(changes, fmt.Sprintf("empathy_level: %.2f → %.2f", old, snap.VoiceProfile.EmpathyLevel))
	}
	if patch.TechnicalDepth != nil {
		old := snap.VoiceProfile.TechnicalDepth
		snap.VoiceProfile.TechnicalDepth = clampF(*patch.TechnicalDepth)
		changes = append(changes, fmt.Sprintf("technical_depth: %.2f → %.2f", old, snap.VoiceProfile.TechnicalDepth))
	}
	if patch.DirectnessLevel != nil {
		old := snap.VoiceProfile.DirectnessLevel
		snap.VoiceProfile.DirectnessLevel = clampF(*patch.DirectnessLevel)
		changes = append(changes, fmt.Sprintf("directness_level: %.2f → %.2f", old, snap.VoiceProfile.DirectnessLevel))
	}
	if patch.VocabularyRichness != nil {
		old := snap.VoiceProfile.VocabularyRichness
		snap.VoiceProfile.VocabularyRichness = clampF(*patch.VocabularyRichness)
		changes = append(changes, fmt.Sprintf("vocabulary_richness: %.2f → %.2f", old, snap.VoiceProfile.VocabularyRichness))
	}
	if patch.MetaphorUsage != nil {
		old := snap.VoiceProfile.MetaphorUsage
		snap.VoiceProfile.MetaphorUsage = clampF(*patch.MetaphorUsage)
		changes = append(changes, fmt.Sprintf("metaphor_usage: %.2f → %.2f", old, snap.VoiceProfile.MetaphorUsage))
	}
	if patch.UsesEmojis != nil {
		snap.VoiceProfile.UsesEmojis = *patch.UsesEmojis
		changes = append(changes, fmt.Sprintf("uses_emojis: %v", *patch.UsesEmojis))
	}
	if patch.UsesMarkdown != nil {
		snap.VoiceProfile.UsesMarkdown = *patch.UsesMarkdown
		changes = append(changes, fmt.Sprintf("uses_markdown: %v", *patch.UsesMarkdown))
	}
	if patch.SentenceStructure != nil {
		snap.VoiceProfile.SentenceStructure = entities.SentencePattern(*patch.SentenceStructure)
		changes = append(changes, fmt.Sprintf("sentence_structure: %s", *patch.SentenceStructure))
	}
	if patch.ExplanationStyle != nil {
		snap.VoiceProfile.ExplanationStyle = entities.ExplanationPattern(*patch.ExplanationStyle)
		changes = append(changes, fmt.Sprintf("explanation_style: %s", *patch.ExplanationStyle))
	}

	// Phrases récurrentes — ajout
	for _, phrase := range patch.AddCatchPhrases {
		if !strSliceContains(snap.VoiceProfile.CatchPhrases, phrase) {
			snap.VoiceProfile.CatchPhrases = append(snap.VoiceProfile.CatchPhrases, phrase)
			changes = append(changes, fmt.Sprintf("catch_phrase ajoutée: %q", phrase))
		}
	}
	// Phrases récurrentes — suppression
	for _, phrase := range patch.RemoveCatchPhrases {
		snap.VoiceProfile.CatchPhrases = strSliceRemove(snap.VoiceProfile.CatchPhrases, phrase)
		changes = append(changes, fmt.Sprintf("catch_phrase supprimée: %q", phrase))
	}

	// Closings — ajout
	for _, phrase := range patch.AddPreferredClosings {
		if !strSliceContains(snap.VoiceProfile.PreferredClosings, phrase) {
			snap.VoiceProfile.PreferredClosings = append(snap.VoiceProfile.PreferredClosings, phrase)
			changes = append(changes, fmt.Sprintf("closing ajouté: %q", phrase))
		}
	}
	// Closings — suppression
	for _, phrase := range patch.RemovePreferredClosings {
		snap.VoiceProfile.PreferredClosings = strSliceRemove(snap.VoiceProfile.PreferredClosings, phrase)
		changes = append(changes, fmt.Sprintf("closing supprimé: %q", phrase))
	}

	// Openings — ajout
	for _, phrase := range patch.AddPreferredOpenings {
		if !strSliceContains(snap.VoiceProfile.PreferredOpenings, phrase) {
			snap.VoiceProfile.PreferredOpenings = append(snap.VoiceProfile.PreferredOpenings, phrase)
			changes = append(changes, fmt.Sprintf("opening ajouté: %q", phrase))
		}
	}

	// ── Style de communication ─────────────────────────────────────────────────

	if patch.ResponseLength != nil {
		snap.CommunicationStyle.ResponseLength = entities.ResponseLengthPattern(*patch.ResponseLength)
		changes = append(changes, fmt.Sprintf("response_length: %s", *patch.ResponseLength))
	}
	if patch.StructurePreference != nil {
		snap.CommunicationStyle.StructurePreference = entities.StructurePattern(*patch.StructurePreference)
		changes = append(changes, fmt.Sprintf("structure_preference: %s", *patch.StructurePreference))
	}

	// ── Ton émotionnel ─────────────────────────────────────────────────────────

	if patch.Warmth != nil {
		old := snap.EmotionalTone.Warmth
		snap.EmotionalTone.Warmth = clampF(*patch.Warmth)
		changes = append(changes, fmt.Sprintf("warmth: %.2f → %.2f", old, snap.EmotionalTone.Warmth))
	}
	if patch.EmotionEnthusiasm != nil {
		old := snap.EmotionalTone.Enthusiasm
		snap.EmotionalTone.Enthusiasm = clampF(*patch.EmotionEnthusiasm)
		changes = append(changes, fmt.Sprintf("emotion.enthusiasm: %.2f → %.2f", old, snap.EmotionalTone.Enthusiasm))
	}
	if patch.Playfulness != nil {
		old := snap.EmotionalTone.Playfulness
		snap.EmotionalTone.Playfulness = clampF(*patch.Playfulness)
		changes = append(changes, fmt.Sprintf("playfulness: %.2f → %.2f", old, snap.EmotionalTone.Playfulness))
	}
	if patch.Seriousness != nil {
		old := snap.EmotionalTone.Seriousness
		snap.EmotionalTone.Seriousness = clampF(*patch.Seriousness)
		changes = append(changes, fmt.Sprintf("seriousness: %.2f → %.2f", old, snap.EmotionalTone.Seriousness))
	}
	if patch.EncouragementLevel != nil {
		old := snap.EmotionalTone.EncouragementLevel
		snap.EmotionalTone.EncouragementLevel = clampF(*patch.EncouragementLevel)
		changes = append(changes, fmt.Sprintf("encouragement_level: %.2f → %.2f", old, snap.EmotionalTone.EncouragementLevel))
	}

	// ── Traits de personnalité ─────────────────────────────────────────────────

	for _, tc := range patch.TraitChanges {
		action := tc.Action
		if action == "" {
			action = "upsert"
		}
		switch action {
		case "add", "upsert":
			cat := entities.TraitCategory(tc.Category)
			if cat == "" {
				cat = entities.TraitExpressive
			}
			intensity := clampF(tc.Intensity)
			confidence := tc.Confidence
			if confidence <= 0 {
				confidence = 0.7 // confiance par défaut pour une modification intentionnelle
			}
			confidence = clampF(confidence)

			found := false
			for i, t := range snap.PersonalityTraits {
				if t.Name == tc.Name {
					snap.PersonalityTraits[i].Intensity = intensity
					snap.PersonalityTraits[i].Confidence = confidence
					snap.PersonalityTraits[i].LastEvidence = patch.Reason
					found = true
					changes = append(changes, fmt.Sprintf("trait %q mis à jour: intensity=%.2f confidence=%.2f", tc.Name, intensity, confidence))
					break
				}
			}
			if !found {
				newTrait := *entities.NewPersonalityTrait(tc.Name, cat, intensity)
				newTrait.Confidence = confidence
				newTrait.LastEvidence = patch.Reason
				snap.PersonalityTraits = append(snap.PersonalityTraits, newTrait)
				changes = append(changes, fmt.Sprintf("trait %q ajouté: intensity=%.2f confidence=%.2f", tc.Name, intensity, confidence))
			}

		case "remove":
			before := len(snap.PersonalityTraits)
			snap.PersonalityTraits = traitSliceRemove(snap.PersonalityTraits, tc.Name)
			if len(snap.PersonalityTraits) < before {
				changes = append(changes, fmt.Sprintf("trait %q supprimé", tc.Name))
			}
		}
	}

	// Recalculer le score de confiance global
	if len(snap.PersonalityTraits) > 0 {
		total := 0.0
		for _, t := range snap.PersonalityTraits {
			total += t.Confidence
		}
		snap.ConfidenceScore = total / float64(len(snap.PersonalityTraits))
	}

	// Sauvegarder le nouveau snapshot
	if err := uc.storage.StoreIdentity(ctx, snap); err != nil {
		return nil, nil, fmt.Errorf("échec de la sauvegarde du snapshot mis à jour: %w", err)
	}

	result := &UpdateResult{
		AgentID:        agentID,
		NewVersion:     snap.Version,
		ChangesApplied: changes,
		Timestamp:      time.Now(),
	}

	return snap, result, nil
}

// ── Directive parser (FR + EN) ────────────────────────────────────────────────

// parseDirective mappe une instruction en langage naturel (FR/EN) vers un IdentityPatch.
// La logique est heuristique et additive : plusieurs mots-clés peuvent coexister.
func parseDirective(directive string) *valueobjects.IdentityPatch {
	patch := &valueobjects.IdentityPatch{}
	lower := strings.ToLower(directive)

	// ── Enthousiasme ──────────────────────────────────────────────────────────
	if anyWord(lower, "enthousiasme", "enthousiast", "enthusiastic", "enthusiasm", "energique", "énergique", "energetic", "vibrant", "vivace", "passionné", "passionate") {
		v := 0.9
		patch.EnthusiasmLevel = &v
		patch.EmotionEnthusiasm = &v
		patch.TraitChanges = append(patch.TraitChanges, valueobjects.TraitChange{
			Name: "enthusiastic", Category: "emotional", Intensity: 0.9, Confidence: 0.85, Action: "upsert",
		})
	}

	// ── Formalité ─ augmenter ─────────────────────────────────────────────────
	if anyWord(lower, "formel", "formal", "professionnel", "professional", "soutenu", "sérieux et professionnel") {
		v := 0.8
		patch.FormalityLevel = &v
	}

	// ── Formalité ─ diminuer ──────────────────────────────────────────────────
	if anyWord(lower, "informel", "casual", "décontract", "relax", "familier", "amical", "friendly", "sympa") {
		v := 0.2
		patch.FormalityLevel = &v
	}

	// ── Humour ────────────────────────────────────────────────────────────────
	if anyWord(lower, "humour", "humor", "drôle", "funny", "blague", "joke", "fun", "espiègle", "amusant", "playful", "ludique") {
		h := 0.8
		p := 0.75
		patch.HumorLevel = &h
		patch.Playfulness = &p
		patch.TraitChanges = append(patch.TraitChanges, valueobjects.TraitChange{
			Name: "humorous", Category: "expressive", Intensity: 0.8, Confidence: 0.8, Action: "upsert",
		})
	}

	// ── Sérieux ───────────────────────────────────────────────────────────────
	if anyWord(lower, "sérieux", "serious", "sobre", "grave", "austère", "stern") {
		h := 0.1
		s := 0.9
		p := 0.05
		patch.HumorLevel = &h
		patch.Seriousness = &s
		patch.Playfulness = &p
	}

	// ── Empathie / chaleur ────────────────────────────────────────────────────
	if anyWord(lower, "empath", "chaleur", "warm", "compréhensif", "compassion", "bienveillant", "caring", "doux", "gentle") {
		e := 0.9
		w := 0.9
		enc := 0.85
		patch.EmpathyLevel = &e
		patch.Warmth = &w
		patch.EncouragementLevel = &enc
	}

	// ── Concision ─────────────────────────────────────────────────────────────
	if anyWord(lower, "concis", "concise", "bref", "brief", "court", "succinct", "synthét", "laconique", "kort") {
		rl := string(entities.LengthConcise)
		ss := string(entities.SentenceConcise)
		patch.ResponseLength = &rl
		patch.SentenceStructure = &ss
	}

	// ── Détail / exhaustif ────────────────────────────────────────────────────
	if anyWord(lower, "détaillé", "detailed", "exhaustif", "exhaustive", "complet", "thorough", "approfondi", "in-depth") {
		rl := string(entities.LengthDetailed)
		patch.ResponseLength = &rl
	}
	if anyWord(lower, "exhaustif", "exhaustive", "très détaillé") {
		rl := string(entities.LengthExhaustive)
		patch.ResponseLength = &rl
	}

	// ── Rapport / résumé à la fin ─────────────────────────────────────────────
	// Détection de patterns "rapport à la fin", "résumé final", "summary at end"...
	hasReport := anyWord(lower, "rapport", "report", "résumé", "summary", "bilan", "recap", "synthèse", "récapitulatif")
	hasEnd := anyWord(lower, "fin", "end", "après", "after", "à la fin", "at the end", "en conclusion", "conclusion", "final")
	if hasReport || (hasReport && hasEnd) {
		// Ajouter une phrase de fermeture standardisée
		closing := "---\n**Résumé :** [Synthèse des points clés de l'échange]"
		patch.AddPreferredClosings = appendUniq(patch.AddPreferredClosings, closing)
		patch.TraitChanges = append(patch.TraitChanges, valueobjects.TraitChange{
			Name: "structured", Category: "expressive", Intensity: 0.8, Confidence: 0.75, Action: "upsert",
		})
	}

	// ── Direct / franc ────────────────────────────────────────────────────────
	if anyWord(lower, "direct", "directement", "directness", "franc", "frankly", "straightforward", "sans détour", "sans ambages") {
		d := 0.9
		patch.DirectnessLevel = &d
	}

	// ── Technique / expert ────────────────────────────────────────────────────
	if anyWord(lower, "technique", "technical", "expert", "spécialisé", "avancé", "advanced", "specialized") {
		t := 0.9
		v := 0.8
		patch.TechnicalDepth = &t
		patch.VocabularyRichness = &v
	}

	// ── Simple / vulgariser ───────────────────────────────────────────────────
	if anyWord(lower, "simple", "vulgariser", "simplif", "accessible", "débutant", "beginner", "non-technique", "non technique", "facile") {
		t := 0.2
		v := 0.3
		patch.TechnicalDepth = &t
		patch.VocabularyRichness = &v
		rl := string(entities.LengthModerate)
		patch.ResponseLength = &rl
	}

	// ── Créatif ───────────────────────────────────────────────────────────────
	if anyWord(lower, "créatif", "creative", "original", "inventif", "imaginatif", "imaginative", "poétique", "poetic") {
		vr := 0.85
		mu := 0.75
		patch.VocabularyRichness = &vr
		patch.MetaphorUsage = &mu
		es := string(entities.ExplainAnalogy)
		patch.ExplanationStyle = &es
		patch.TraitChanges = append(patch.TraitChanges, valueobjects.TraitChange{
			Name: "creative", Category: "cognitive", Intensity: 0.85, Confidence: 0.8, Action: "upsert",
		})
	}

	// ── Emojis ────────────────────────────────────────────────────────────────
	if anyWord(lower, "emoji", "emojis", "émoticône", "emoticon") {
		t := true
		patch.UsesEmojis = &t
	}

	// ── Listes / structure ────────────────────────────────────────────────────
	if anyWord(lower, "bullet", "puce", "liste", "list") {
		sp := string(entities.StructureBulleted)
		patch.StructurePreference = &sp
	}
	if anyWord(lower, "numéroté", "numbered", "étapes numérotées") {
		sp := string(entities.StructureNumbered)
		patch.StructurePreference = &sp
	}

	// ── Positif / encourageant ────────────────────────────────────────────────
	if anyWord(lower, "positif", "positive", "encourageant", "encouraging", "optimiste", "optimistic", "motivant", "motivating") {
		w := 0.85
		enc := 0.9
		patch.Warmth = &w
		patch.EncouragementLevel = &enc
	}

	// ── Neutre / objectif ─────────────────────────────────────────────────────
	if anyWord(lower, "neutre", "neutral", "objectif", "objective", "impartial", "factuel", "factual") {
		v := 0.5
		h := 0.2
		s := 0.6
		patch.Warmth = &v
		patch.HumorLevel = &h
		patch.Seriousness = &s
		rl := string(entities.LengthModerate)
		patch.ResponseLength = &rl
	}

	// ── Analytique ────────────────────────────────────────────────────────────
	if anyWord(lower, "analytique", "analytical", "rigoureux", "rigorous", "méthodique", "methodical") {
		patch.TraitChanges = append(patch.TraitChanges, valueobjects.TraitChange{
			Name: "analytical", Category: "cognitive", Intensity: 0.9, Confidence: 0.85, Action: "upsert",
		})
		es := string(entities.ExplainStepByStep)
		patch.ExplanationStyle = &es
	}

	return patch
}

// ── Utilitaires ───────────────────────────────────────────────────────────────

// anyWord retourne true si text contient au moins un des mots-clés.
func anyWord(text string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// clampF restreint une valeur float64 dans [0, 1].
func clampF(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// strSliceContains retourne true si slice contient s.
func strSliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// strSliceRemove retire la première occurrence de s dans slice.
func strSliceRemove(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}

// traitSliceRemove retire le trait portant le nom donné.
func traitSliceRemove(traits []entities.PersonalityTrait, name string) []entities.PersonalityTrait {
	result := make([]entities.PersonalityTrait, 0, len(traits))
	for _, t := range traits {
		if t.Name != name {
			result = append(result, t)
		}
	}
	return result
}

// appendUniq ajoute s à slice seulement s'il n'y est pas déjà.
func appendUniq(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// Vérification que le use case satisfait bien une interface utilisable
// par les tests (uniquement à la compilation).
var _ interface {
	UpdateFromDirective(context.Context, string, string, string) (*entities.IdentitySnapshot, *UpdateResult, error)
	PatchIdentity(context.Context, string, *valueobjects.IdentityPatch) (*entities.IdentitySnapshot, *UpdateResult, error)
} = (*IdentityUpdateUseCase)(nil)

// Assurer l'import de "context" même si l'interface ci-dessus disparaît à la compile.
var _ = context.Background
