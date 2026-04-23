// Package composition implements identity prompt composition.
// Transforms identity snapshots into prompts injectable into LLM context window.
package composition

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/pkoukk/tiktoken-go"
)

// SoulComposerService implements ports.IdentityComposer.
// Generates natural and effective identity prompts for LLMs.
type SoulComposerService struct {
	baseTemplate       string
	reinforceTemplate string
	alertTemplate     string
	tokenizer         *tiktoken.Tiktoken
}

// NewSoulComposerService creates a new composition service.
func NewSoulComposerService() *SoulComposerService {
	tok, err := tiktoken.EncodingForModel("gpt-3.5-turbo")
	if err != nil {
		slog.Warn("failed to load tiktoken tokenizer, falling back to char-based estimation", "error", err)
		tok = nil
	}
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
		tokenizer: tok,
	}
}

// ComposeIdentityPrompt generates a complete identity prompt
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

// EstimateTokenCount estimates the number of tokens in a text.
func (s *SoulComposerService) EstimateTokenCount(ctx context.Context, prompt string) (int, error) {
	if s.tokenizer != nil {
		tokens := s.tokenizer.Encode(prompt, nil, nil)
		return len(tokens), nil
	}
	// Fallback: char-based estimation (~4 chars per token for English)
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
	currentTokens, err := s.EstimateTokenCount(context.Background(), prompt)
	if err != nil || currentTokens <= budgetTokens {
		return prompt
	}

	// Smart reduction strategy: prioritize sections by importance
	sections := parseSections(prompt)
	if len(sections) == 0 {
		return truncateByTokens(prompt, budgetTokens, s)
	}

	// Section priority (higher = more important)
	sectionPriority := map[string]int{
		"voice_profile":        90,
		"personality_traits":   85,
		"communication_style": 70,
		"value_system":        60,
		"behavioral_signature": 50,
		"emotional_tone":       40,
	}

	type section struct {
		name    string
		content string
		tokens  int
		lines   int
	}

	parsed := make([]section, 0, len(sections))
	for name, content := range sections {
		tok, _ := s.EstimateTokenCount(context.Background(), content)
		parsed = append(parsed, section{
			name:    name,
			content: content,
			tokens:  tok,
			lines:   strings.Count(content, "\n") + 1,
		})
	}

	// Sort by priority (descending)
	for i := 0; i < len(parsed); i++ {
		for j := i + 1; j < len(parsed); j++ {
			pi, pj := sectionPriority[parsed[i].name], sectionPriority[parsed[j].name]
			if pi < pj {
				parsed[i], parsed[j] = parsed[j], parsed[i]
			}
		}
	}

	// Greedily select sections to fit budget
	var result string
	remaining := budgetTokens

	for _, sec := range parsed {
		if sec.tokens <= remaining-5 { // Keep 5 tokens buffer
			result += "### " + formatSectionName(sec.name) + "\n\n" + sec.content + "\n\n"
			remaining -= sec.tokens
		} else if remaining > 20 {
			// Partial section
			result += "### " + formatSectionName(sec.name) + "\n\n" +
				truncateByTokens(sec.content, remaining-5, s) + "\n\n"
			break
		}
	}

	if result == "" {
		result = "## Your Identity\n\n" + truncateByTokens(prompt, budgetTokens-10, s)
	}

	return strings.TrimSpace(result)
}

func parseSections(prompt string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(prompt, "\n")

	var currentSection string
	var currentContent strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### ") {
			if currentSection != "" {
				sections[currentSection] = strings.TrimSpace(currentContent.String())
				currentContent.Reset()
			}
			currentSection = strings.ToLower(strings.TrimPrefix(trimmed, "### "))
		} else if currentSection != "" {
			currentContent.WriteString(line + "\n")
		}
	}

	if currentSection != "" {
		sections[currentSection] = strings.TrimSpace(currentContent.String())
	}

	return sections
}

func formatSectionName(name string) string {
	// Convert snake_case or joined to Title Case
	name = strings.ReplaceAll(name, "_", " ")
	return strings.Title(name)
}

func truncateByTokens(text string, maxTokens int, s *SoulComposerService) string {
	if maxTokens <= 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	var result strings.Builder
	remaining := maxTokens

	for _, line := range lines {
		lineTokens, _ := s.EstimateTokenCount(context.Background(), line)
		if lineTokens <= remaining {
			result.WriteString(line + "\n")
			remaining -= lineTokens
		} else if remaining > 10 {
			// Partial line
			words := strings.Fields(line)
			for _, word := range words {
				wordTokens, _ := s.EstimateTokenCount(context.Background(), word+" ")
				if wordTokens <= remaining-1 {
					result.WriteString(word + " ")
					remaining -= wordTokens
				} else {
					break
				}
			}
			break
		} else {
			break
		}
	}

	return strings.TrimSpace(result.String())
}

func replaceTag(text, tag, replacement string) string {
	return strings.ReplaceAll(text, tag, replacement)
}
