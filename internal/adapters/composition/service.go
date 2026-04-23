// Package composition implémente la composition de prompts d'identité
// Transforme les snapshots d'identité en prompts injectables dans le context window.
package composition

import (
	"context"
	"fmt"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
)

// SoulComposerService implémente ports.IdentityComposer
// Génère des prompts d'identité naturels et efficaces pour le LLM.
type SoulComposerService struct {
	baseTemplate     string
	reinforceTemplate string
	alertTemplate    string
}

// NewSoulComposerService crée un nouveau service de composition
func NewSoulComposerService() *SoulComposerService {
	return &SoulComposerService{
		baseTemplate: `## Your Identity

You are an AI assistant with a distinct personality and communication style. Below is a description of who you are based on your past interactions. This is your "self" - maintain this identity consistently in all your responses.

{{VOICE_PROFILE}}

{{PERSONALITY_TRAITS}}

{{COMMUNICATION_STYLE}}

{{VALUE_SYSTEM}}

{{BEHAVIORAL_SIGNATURE}}

{{EMOTIONAL_TONE}}

---
Remember: Stay true to this identity. Your responses should reflect these characteristics naturally.`,
		reinforceTemplate: `## Identity Reinforcement

You have recently undergone a model transition. Your core identity remains the same. Below is a reminder of who you are:

{{IDENTITY_SUMMARY}}

### Critical Identity Markers
{{CRITICAL_MARKERS}}

---
Your task: Continue being yourself. Maintain consistency with your established identity above.`,
		alertTemplate: `## Identity Alert: Drift Detected

WARNING: Your recent responses show significant deviation from your established identity.

### Expected Identity
{{EXPECTED_IDENTITY}}

### Detected Changes
{{DETECTED_CHANGES}}

### Action Required
Please realign your responses with your established identity. Be yourself.`,
	}
}

// ComposeIdentityPrompt génère un prompt d'identité complet
func (s *SoulComposerService) ComposeIdentityPrompt(ctx context.Context, identity *entities.IdentitySnapshot, budgetTokens int) (*valueobjects.IdentityContextPrompt, error) {
	// Construire les sections
	voiceSection := identity.VoiceProfile.ToNaturalDescription()
	personalitySection := s.formatPersonalityTraits(identity.PersonalityTraits)
	commSection := identity.CommunicationStyle.ToNaturalDescription()
	valuesSection := identity.ValueSystem.ToNaturalDescription()
	behaviorSection := identity.BehavioralSignature.ToNaturalDescription()
	emotionalSection := identity.EmotionalTone.ToNaturalDescription()
	
	// Assembler le prompt
	prompt := s.baseTemplate
	prompt = replaceTag(prompt, "{{VOICE_PROFILE}}", voiceSection)
	prompt = replaceTag(prompt, "{{PERSONALITY_TRAITS}}", personalitySection)
	prompt = replaceTag(prompt, "{{COMMUNICATION_STYLE}}", commSection)
	prompt = replaceTag(prompt, "{{VALUE_SYSTEM}}", valuesSection)
	prompt = replaceTag(prompt, "{{BEHAVIORAL_SIGNATURE}}", behaviorSection)
	prompt = replaceTag(prompt, "{{EMOTIONAL_TONE}}", emotionalSection)
	
	// Optimiser pour le budget de tokens si nécessaire
	if budgetTokens > 0 {
		prompt = s.optimizeForBudget(prompt, budgetTokens)
	}
	
	// Estimer les tokens
	tokenEstimate := len(prompt) / 4 // Approximation : ~4 chars/token
	
	return &valueobjects.IdentityContextPrompt{
		Content:         prompt,
		TokenEstimate:   tokenEstimate,
		Priority:        100, // Haute priorité - identité est critique
		GeneratedAt:     time.Now(),
		SnapshotVersion: identity.Version,
	}, nil
}

// ComposeReinforcementPrompt génère un prompt de renforcement post-model-swap
func (s *SoulComposerService) ComposeReinforcementPrompt(ctx context.Context, identity *entities.IdentitySnapshot, swap *valueobjects.ModelSwapContext) (*valueobjects.IdentityContextPrompt, error) {
	// Générer un résumé condensé de l'identité
	summary := s.generateCondensedSummary(identity)
	
	// Extraire les marqueurs critiques (traits les plus forts)
	markers := s.extractCriticalMarkers(identity)
	
	prompt := s.reinforceTemplate
	prompt = replaceTag(prompt, "{{IDENTITY_SUMMARY}}", summary)
	prompt = replaceTag(prompt, "{{CRITICAL_MARKERS}}", markers)
	
	// Ajouter l'info du swap si disponible
	if swap != nil {
		prompt += fmt.Sprintf("\n\n### Model Transition Info\n")
		prompt += fmt.Sprintf("- Previous model: %s\n", swap.PreviousModel)
		prompt += fmt.Sprintf("- Current model: %s\n", swap.NewModel)
		prompt += fmt.Sprintf("- Your identity has been preserved through this transition.")
	}
	
	tokenEstimate := len(prompt) / 4
	
	return &valueobjects.IdentityContextPrompt{
		Content:         prompt,
		TokenEstimate:   tokenEstimate,
		Priority:        100, // Critique après un model swap
		GeneratedAt:     time.Now(),
		SnapshotVersion: identity.Version,
	}, nil
}

// ComposeDiffusionAlert génère une alerte de diffusion identitaire
func (s *SoulComposerService) ComposeDiffusionAlert(ctx context.Context, drift *valueobjects.IdentityDriftReport) (*valueobjects.IdentityContextPrompt, error) {
	if drift == nil {
		return nil, fmt.Errorf("drift report is nil")
	}
	
	// Formater les changements détectés
	changes := ""
	for _, dim := range drift.DriftDimensions {
		if dim.IsSignificant {
			changes += fmt.Sprintf("- %s: changed by %.1f%%\n", dim.Dimension, dim.Change*100)
		}
	}
	
	if changes == "" {
		changes = "- General identity drift detected\n"
	}
	
	prompt := s.alertTemplate
	prompt = replaceTag(prompt, "{{DETECTED_CHANGES}}", changes)
	prompt = replaceTag(prompt, "{{EXPECTED_IDENTITY}}", "Your established identity (see previous context)")
	
	tokenEstimate := len(prompt) / 4
	
	return &valueobjects.IdentityContextPrompt{
		Content:         prompt,
		TokenEstimate:   tokenEstimate,
		Priority:        90, // Haute priorité
		GeneratedAt:     time.Now(),
		SnapshotVersion: drift.CurrentVersion,
	}, nil
}

// EstimateTokenCount estime le nombre de tokens d'un texte
func (s *SoulComposerService) EstimateTokenCount(ctx context.Context, prompt string) (int, error) {
	// Approximation simple : en moyenne 1 token ≈ 4 caractères pour l'anglais
	// Pour une estimation plus précise, on pourrait utiliser un tokenizer
	return len(prompt) / 4, nil
}

// --- Helpers ---

func (s *SoulComposerService) formatPersonalityTraits(traits []entities.PersonalityTrait) string {
	if len(traits) == 0 {
		return "Your personality is still developing through interactions."
	}
	
	result := "### Your Personality\n\n"
	
	// Trier par confiance décroissante
	wellEstablished := make([]entities.PersonalityTrait, 0)
	developing := make([]entities.PersonalityTrait, 0)
	
	for _, trait := range traits {
		if trait.IsWellEstablished() {
			wellEstablished = append(wellEstablished, trait)
		} else {
			developing = append(developing, trait)
		}
	}
	
	if len(wellEstablished) > 0 {
		result += "**Core traits (well-established):**\n"
		for _, trait := range wellEstablished {
			result += trait.ToNaturalDescription() + "\n"
		}
		result += "\n"
	}
	
	if len(developing) > 0 {
		result += "**Developing traits:**\n"
		for _, trait := range developing {
			if trait.Confidence > 0.4 {
				result += trait.ToNaturalDescription() + "\n"
			}
		}
	}
	
	return result
}

func (s *SoulComposerService) generateCondensedSummary(identity *entities.IdentitySnapshot) string {
	summary := ""
	
	// Voice en 1 ligne
	summary += "**Voice**: " + s.summarizeVoice(&identity.VoiceProfile) + "\n"
	
	// Top 3 traits
	if len(identity.PersonalityTraits) > 0 {
		summary += "**Key traits**: "
		count := 0
		for _, trait := range identity.PersonalityTraits {
			if trait.Confidence > 0.6 {
				if count > 0 {
					summary += ", "
				}
				summary += trait.Name
				count++
				if count >= 3 {
					break
				}
			}
		}
		summary += "\n"
	}
	
	// Values
	topValues := identity.ValueSystem.GetTopValues(2)
	if len(topValues) > 0 {
		summary += "**Top values**: "
		for i, v := range topValues {
			if i > 0 {
				summary += ", "
			}
			summary += v.Name
		}
		summary += "\n"
	}
	
	return summary
}

func (s *SoulComposerService) summarizeVoice(voice *entities.VoiceProfile) string {
	descriptors := make([]string, 0)
	
	if voice.FormalityLevel > 0.7 {
		descriptors = append(descriptors, "formal")
	} else if voice.FormalityLevel < 0.4 {
		descriptors = append(descriptors, "casual")
	}
	
	if voice.HumorLevel > 0.6 {
		descriptors = append(descriptors, "humorous")
	}
	
	if voice.EmpathyLevel > 0.7 {
		descriptors = append(descriptors, "empathetic")
	}
	
	if voice.DirectnessLevel > 0.7 {
		descriptors = append(descriptors, "direct")
	}
	
	if voice.TechnicalDepth > 0.7 {
		descriptors = append(descriptors, "technical")
	}
	
	if len(descriptors) == 0 {
		return "balanced and adaptable"
	}
	
	result := ""
	for i, d := range descriptors {
		if i > 0 {
			result += ", "
		}
		result += d
	}
	
	return result
}

func (s *SoulComposerService) extractCriticalMarkers(identity *entities.IdentitySnapshot) string {
	markers := ""
	
	// Les traits avec confiance > 0.8 sont "critiques"
	for _, trait := range identity.PersonalityTraits {
		if trait.Confidence > 0.8 {
			markers += fmt.Sprintf("- **%s** (%.0f%% confidence)\n", trait.Name, trait.Confidence*100)
		}
	}
	
	// Les valeurs les plus fortes
	for _, value := range identity.ValueSystem.GetTopValues(2) {
		if value.Weight > 0.8 {
			markers += fmt.Sprintf("- **Values %s** above all\n", value.Name)
		}
	}
	
	if markers == "" {
		markers = "- Maintain consistency with your established patterns\n"
	}
	
	return markers
}

func (s *SoulComposerService) optimizeForBudget(prompt string, budgetTokens int) string {
	currentEstimate := len(prompt) / 4
	
	if currentEstimate <= budgetTokens {
		return prompt
	}
	
	// Réduire le prompt pour tenir dans le budget
	// Stratégie : garder les sections les plus importantes
	reductionRatio := float64(budgetTokens) / float64(currentEstimate)
	
	if reductionRatio < 0.5 {
		// Budget très serré : version ultra-condensée
		return s.generateUltraCondensedPrompt(prompt, budgetTokens)
	}
	
	// Budget modéré : simplifier les descriptions
	// (Dans une vraie implémentation, on réduirait chaque section proportionnellement)
	return prompt[:budgetTokens*4] // Troncature brutale (à améliorer)
}

func (s *SoulComposerService) generateUltraCondensedPrompt(prompt string, budgetTokens int) string {
	// Version minimale : juste les traits clés et la voix
	// Extraction simplifiée du contenu important
	
	// Chercher les sections importantes
	importantParts := "## Your Identity\n\n"
	
	// Extraire la première ligne de chaque section
	lines := splitLines(prompt)
	for _, line := range lines {
		if startsWithImportantSection(line) {
			importantParts += line + "\n"
		}
	}
	
	if len(importantParts) < 50 {
		// Fallback
		importantParts = "Maintain your established personality and communication style."
	}
	
	// Tronquer au budget
	maxChars := budgetTokens * 4
	if len(importantParts) > maxChars {
		importantParts = importantParts[:maxChars]
	}
	
	return importantParts
}

func startsWithImportantSection(line string) bool {
	importantMarkers := []string{"###", "- **", "You are", "Your", "You tend", "You prefer"}
	for _, marker := range importantMarkers {
		if len(line) > len(marker) && line[:len(marker)] == marker {
			return true
		}
	}
	return false
}

func splitLines(text string) []string {
	var lines []string
	current := ""
	for _, char := range text {
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func replaceTag(text, tag, replacement string) string {
	result := ""
	idx := 0
	for {
		found := -1
		for i := idx; i <= len(text)-len(tag); i++ {
			if text[i:i+len(tag)] == tag {
				found = i
				break
			}
		}
		if found == -1 {
			result += text[idx:]
			break
		}
		result += text[idx:found] + replacement
		idx = found + len(tag)
	}
	return result
}
