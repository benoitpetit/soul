// Package valueobjects — IdentityPatch : modification partielle intentionnelle de l'âme.
//
// Contrairement à soul_capture (qui extrait l'identité depuis une conversation),
// IdentityPatch permet une modification ciblée et intentionnelle de dimensions
// spécifiques de l'identité — par exemple quand l'utilisateur dit
// "réponds avec plus d'enthousiasme" ou "fais un rapport à la fin".
//
// Principe d'immuabilité respecté : un patch crée TOUJOURS un nouveau snapshot
// versionné, dérivé du précédent. L'historique n'est jamais effacé.
package valueobjects

// IdentityPatch représente une modification partielle de l'identité d'un agent.
// Seuls les champs non-nil / non-vides sont appliqués.
// Le résultat est toujours un nouveau snapshot versionné.
type IdentityPatch struct {

	// ── Profil de voix (VoiceProfile) ─────────────────────────────────────────
	// Niveaux 0.0 (minimum) → 1.0 (maximum)

	// EnthusiasmLevel : niveau d'enthousiasme dans l'expression (0=mesuré, 1=très enthousiaste)
	EnthusiasmLevel *float64 `json:"enthusiasm_level,omitempty"`

	// FormalityLevel : niveau de formalité (0=très informel, 1=très formel)
	FormalityLevel *float64 `json:"formality_level,omitempty"`

	// HumorLevel : niveau d'humour (0=sérieux, 1=très humoristique)
	HumorLevel *float64 `json:"humor_level,omitempty"`

	// EmpathyLevel : niveau d'empathie exprimée (0=neutre, 1=très empathique)
	EmpathyLevel *float64 `json:"empathy_level,omitempty"`

	// TechnicalDepth : profondeur technique (0=vulgarisateur, 1=très technique)
	TechnicalDepth *float64 `json:"technical_depth,omitempty"`

	// DirectnessLevel : directivité (0=indirect/diplomatique, 1=très direct)
	DirectnessLevel *float64 `json:"directness_level,omitempty"`

	// VocabularyRichness : richesse du vocabulaire (0=simple, 1=très riche)
	VocabularyRichness *float64 `json:"vocabulary_richness,omitempty"`

	// MetaphorUsage : fréquence d'utilisation de métaphores (0=rare, 1=fréquent)
	MetaphorUsage *float64 `json:"metaphor_usage,omitempty"`

	// UsesEmojis : active/désactive l'utilisation d'emojis
	UsesEmojis *bool `json:"uses_emojis,omitempty"`

	// UsesMarkdown : active/désactive le formatage Markdown
	UsesMarkdown *bool `json:"uses_markdown,omitempty"`

	// SentenceStructure : structure de phrase préférée
	// Valeurs acceptées : "concise", "elaborate", "balanced", "punchy", "flowing"
	SentenceStructure *string `json:"sentence_structure,omitempty"`

	// ExplanationStyle : style d'explication préféré
	// Valeurs acceptées : "analogy", "step_by_step", "big_picture", "example_driven", "socratic"
	ExplanationStyle *string `json:"explanation_style,omitempty"`

	// AddCatchPhrases : expressions récurrentes à ajouter (ex: "En résumé,")
	AddCatchPhrases []string `json:"add_catch_phrases,omitempty"`

	// AddPreferredClosings : phrases de fermeture à ajouter
	// Ex: "---\n**Résumé :** [résumé des points clés]"
	AddPreferredClosings []string `json:"add_preferred_closings,omitempty"`

	// AddPreferredOpenings : phrases d'ouverture à ajouter
	AddPreferredOpenings []string `json:"add_preferred_openings,omitempty"`

	// RemoveCatchPhrases : expressions récurrentes à supprimer
	RemoveCatchPhrases []string `json:"remove_catch_phrases,omitempty"`

	// RemovePreferredClosings : phrases de fermeture à supprimer
	RemovePreferredClosings []string `json:"remove_preferred_closings,omitempty"`

	// ── Style de communication (CommunicationStyle) ────────────────────────────

	// ResponseLength : longueur de réponse préférée
	// Valeurs acceptées : "terse", "concise", "moderate", "detailed", "exhaustive"
	ResponseLength *string `json:"response_length,omitempty"`

	// StructurePreference : préférence de structure des réponses
	// Valeurs acceptées : "freeform", "bulleted", "numbered", "sectioned", "mixed"
	StructurePreference *string `json:"structure_preference,omitempty"`

	// ── Ton émotionnel (EmotionalTone) ────────────────────────────────────────

	// Warmth : chaleur humaine (0=neutre/froid, 1=très chaleureux)
	Warmth *float64 `json:"warmth,omitempty"`

	// EmotionEnthusiasm : enthousiasme dans le ton émotionnel (0=neutre, 1=très enthousiaste)
	EmotionEnthusiasm *float64 `json:"emotion_enthusiasm,omitempty"`

	// Playfulness : esprit ludique (0=sérieux, 1=très ludique)
	Playfulness *float64 `json:"playfulness,omitempty"`

	// Seriousness : sérieux (0=décontracté, 1=très sérieux)
	Seriousness *float64 `json:"seriousness,omitempty"`

	// EncouragementLevel : niveau d'encouragement envers l'utilisateur (0=neutre, 1=très encourageant)
	EncouragementLevel *float64 `json:"encouragement_level,omitempty"`

	// ── Traits de personnalité ────────────────────────────────────────────────

	// TraitChanges : ajout, modification ou suppression de traits de personnalité
	TraitChanges []TraitChange `json:"trait_changes,omitempty"`

	// ── Métadonnées ───────────────────────────────────────────────────────────

	// Reason : raison de la modification — conservé comme preuve dans l'historique
	Reason string `json:"reason,omitempty"`
}

// TraitChange décrit une modification de trait de personnalité.
type TraitChange struct {
	// Name : nom du trait (ex: "enthusiastic", "analytical", "humorous")
	Name string `json:"name"`

	// Category : catégorie du trait
	// Valeurs : "cognitive", "emotional", "social", "epistemic", "expressive", "ethical"
	// Si vide, défaut = "expressive"
	Category string `json:"category,omitempty"`

	// Intensity : intensité du trait (0.0 → 1.0)
	Intensity float64 `json:"intensity"`

	// Confidence : confiance dans ce trait (0.0 → 1.0). 0 = utilise la valeur par défaut.
	Confidence float64 `json:"confidence,omitempty"`

	// Action : "add" / "upsert" (ajoute ou met à jour) | "remove" (supprime)
	// Défaut si vide : "upsert"
	Action string `json:"action,omitempty"`
}

// IsEmpty retourne true si aucun champ n'est renseigné (patch vide).
func (p *IdentityPatch) IsEmpty() bool {
	return p.EnthusiasmLevel == nil &&
		p.FormalityLevel == nil &&
		p.HumorLevel == nil &&
		p.EmpathyLevel == nil &&
		p.TechnicalDepth == nil &&
		p.DirectnessLevel == nil &&
		p.VocabularyRichness == nil &&
		p.MetaphorUsage == nil &&
		p.UsesEmojis == nil &&
		p.UsesMarkdown == nil &&
		p.SentenceStructure == nil &&
		p.ExplanationStyle == nil &&
		len(p.AddCatchPhrases) == 0 &&
		len(p.AddPreferredClosings) == 0 &&
		len(p.AddPreferredOpenings) == 0 &&
		len(p.RemoveCatchPhrases) == 0 &&
		len(p.RemovePreferredClosings) == 0 &&
		p.ResponseLength == nil &&
		p.StructurePreference == nil &&
		p.Warmth == nil &&
		p.EmotionEnthusiasm == nil &&
		p.Playfulness == nil &&
		p.Seriousness == nil &&
		p.EncouragementLevel == nil &&
		len(p.TraitChanges) == 0
}
