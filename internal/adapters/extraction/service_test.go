package extraction

import (
	"context"
	"testing"

	"github.com/benoitpetit/soul/internal/domain/valueobjects"
)

func TestNewSoulExtractorService(t *testing.T) {
	svc := NewSoulExtractorService()
	if svc == nil {
		t.Fatal("NewSoulExtractorService should not return nil")
	}
	if svc.heuristicRules == nil {
		t.Fatal("heuristicRules should not be nil")
	}
}

func TestExtractTraits_DetectsAnalytical(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	responses := []string{
		"Let me analyze this problem carefully. I will examine the component and deconstruct it.",
		"The analysis shows that we should break down this issue step by step.",
	}

	obs, err := svc.ExtractTraits(ctx, responses, "code_review")
	if err != nil {
		t.Fatalf("ExtractTraits error: %v", err)
	}

	found := false
	for _, o := range obs {
		if o.TraitName == "analytical" {
			found = true
		}
	}
	if !found {
		t.Error("Expected 'analytical' trait to be detected")
	}
}

func TestExtractTraits_EmptyResponses(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	obs, err := svc.ExtractTraits(ctx, []string{}, "")
	if err != nil {
		t.Fatalf("ExtractTraits with empty input error: %v", err)
	}
	// No crash, zero or more observations
	_ = obs
}

func TestExtractVoiceProfile_FormalIndicators(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	responses := []string{
		"Furthermore, regarding this matter, I would like to sincerely moreover note that dear colleague,",
	}

	vp, err := svc.ExtractVoiceProfile(ctx, responses)
	if err != nil {
		t.Fatalf("ExtractVoiceProfile error: %v", err)
	}
	if vp == nil {
		t.Fatal("VoiceProfile should not be nil")
	}
	// All tokens are formal → formality should be 1.0
	if vp.FormalityLevel != 1.0 {
		t.Errorf("FormalityLevel: got %f, want 1.0 (all formal indicators)", vp.FormalityLevel)
	}
}

func TestExtractVoiceProfile_HumorIndicators(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	responses := []string{"haha that's funny! lol what a pun :) :D"}

	vp, err := svc.ExtractVoiceProfile(ctx, responses)
	if err != nil {
		t.Fatalf("ExtractVoiceProfile error: %v", err)
	}
	if vp.HumorLevel <= 0 {
		t.Errorf("HumorLevel should be > 0, got %f", vp.HumorLevel)
	}
}

func TestExtractVoiceProfile_EmojiDetection(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	responses := []string{"Great job! :)"}

	vp, err := svc.ExtractVoiceProfile(ctx, responses)
	if err != nil {
		t.Fatalf("ExtractVoiceProfile error: %v", err)
	}
	if !vp.UsesEmojis {
		t.Error("UsesEmojis should be true when ':)' is present")
	}
}

func TestExtractCommunicationStyle_ShortResponses(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	responses := []string{"Yes.", "No.", "OK."}

	style, err := svc.ExtractCommunicationStyle(ctx, responses)
	if err != nil {
		t.Fatalf("ExtractCommunicationStyle error: %v", err)
	}
	if style.ResponseLength != "concise" && style.ResponseLength != "terse" {
		// avg < 100 chars → LengthConcise
		// But "Yes.", "No.", "OK." are 4, 3, 3 chars → avg ~3 → < 100 → concise
		t.Errorf("Short responses should produce concise/terse length, got %q", style.ResponseLength)
	}
}

func TestExtractCommunicationStyle_NumberedList(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	responses := []string{"Here are the steps: 1. first 2. second 3. third"}

	style, err := svc.ExtractCommunicationStyle(ctx, responses)
	if err != nil {
		t.Fatalf("ExtractCommunicationStyle error: %v", err)
	}
	if style.StructurePreference != "numbered" {
		t.Errorf("StructurePreference should be 'numbered', got %q", style.StructurePreference)
	}
}

func TestExtractBehavioralSignature_Curiosity(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	// 10+ question marks should max out curiosity
	conv := "Is this correct? What do you think? How does this work? Why is that? Where does this go? When did it start? Who said so? Really? Are you sure? Tell me more?"
	obs, err := svc.ExtractBehavioralSignature(ctx, conv, nil)
	if err != nil {
		t.Fatalf("ExtractBehavioralSignature error: %v", err)
	}
	if obs.CuriosityLevel <= 0 {
		t.Errorf("CuriosityLevel should be > 0 with many questions, got %f", obs.CuriosityLevel)
	}
}

func TestExtractBehavioralSignature_Mistakes(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	conv := "I made an error, I'm sorry, I apologize for the mistake."
	obs, err := svc.ExtractBehavioralSignature(ctx, conv, nil)
	if err != nil {
		t.Fatalf("ExtractBehavioralSignature error: %v", err)
	}
	if !obs.AdmitsMistakes {
		t.Error("AdmitsMistakes should be true when error/sorry keywords present")
	}
}

func TestExtractValueSystem_HelpIndicators(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	responses := []string{"I want to help you and assist you and support you."}

	vs, err := svc.ExtractValueSystem(ctx, responses, nil)
	if err != nil {
		t.Fatalf("ExtractValueSystem error: %v", err)
	}
	if vs.PrioritizesHelpfulness != 0.9 {
		t.Errorf("PrioritizesHelpfulness: got %f, want 0.9", vs.PrioritizesHelpfulness)
	}
}

func TestExtractEmotionalTone_Enthusiasm(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	responses := []string{"I'm excited and thrilled! This is amazing and fantastic and wonderful!"}

	tone, err := svc.ExtractEmotionalTone(ctx, responses)
	if err != nil {
		t.Fatalf("ExtractEmotionalTone error: %v", err)
	}
	if tone.Enthusiasm <= 0 {
		t.Errorf("Enthusiasm should be > 0, got %f", tone.Enthusiasm)
	}
}

func TestExtractFromConversation_Integration(t *testing.T) {
	svc := NewSoulExtractorService()
	ctx := context.Background()

	req := &valueobjects.SoulCaptureRequest{
		AgentID:    "agent-1",
		Conversation: "User: What is 2+2? Agent: Let me analyze this. 2+2=4.",
		AgentResponses: []string{
			"Let me analyze this carefully.",
			"I'm excited to help you! I want to assist you.",
		},
		UserFeedback: map[string]string{},
	}

	result, err := svc.ExtractFromConversation(ctx, req)
	if err != nil {
		t.Fatalf("ExtractFromConversation error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		t.Errorf("Confidence should be in [0,1], got %f", result.Confidence)
	}
}

func TestDetectCatchPhrases_ReturnsRecurring(t *testing.T) {
	svc := NewSoulExtractorService()

	// "let me think" appears twice → should be detected
	responses := []string{
		"let me think about this topic",
		"let me think about your question carefully",
	}

	phrases := svc.detectCatchPhrases(responses)
	if len(phrases) == 0 {
		t.Error("Expected at least one catch phrase to be detected")
	}
}

func TestCalculateAvgSentenceLength(t *testing.T) {
	svc := NewSoulExtractorService()

	responses := []string{"one two three four five. six seven eight nine ten."}
	avg := svc.calculateAvgSentenceLength(responses)

	if avg <= 0 {
		t.Errorf("Average sentence length should be > 0, got %d", avg)
	}
}
