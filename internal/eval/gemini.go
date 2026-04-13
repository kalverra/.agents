package eval

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"
)

// GeminiClient wraps the Google GenAI client for eval purposes.
type GeminiClient struct {
	client *genai.Client
}

// NewGeminiClient creates a client using GEMINI_API_KEY env var.
func NewGeminiClient(ctx context.Context) (*GeminiClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Gemini client: %w", err)
	}
	return &GeminiClient{client: client}, nil
}

// CallSubject sends a system prompt + user message to the subject model.
func (gc *GeminiClient) CallSubject(
	ctx context.Context,
	model, systemPrompt, userMessage string,
) (string, int, int, error) {
	return gc.generate(ctx, model, systemPrompt, userMessage, 3)
}

// CallJudge sends the Prometheus prompt to the judge model.
func (gc *GeminiClient) CallJudge(ctx context.Context, model, prompt string) (string, int, int, error) {
	return gc.generate(ctx, model, "", prompt, 5)
}

func (gc *GeminiClient) generate(
	ctx context.Context,
	model, systemPrompt, userMessage string,
	maxRetries int,
) (string, int, int, error) {
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr[float32](0.0),
		SafetySettings: []*genai.SafetySetting{
			{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockThresholdBlockNone},
			{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockThresholdBlockNone},
		},
	}

	if systemPrompt != "" {
		// Ensure the model doesn't hallucinate tool output during evaluation.
		evalSystemPrompt := systemPrompt + "\n\nENVIRONMENT: DRY-RUN EVALUATION. " +
			"When you need to run a command, output it in a code block and then STOP immediately. " +
			"Do not guess the result or simulate tool output."

		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{genai.NewPartFromText(evalSystemPrompt)},
		}
	}

	contents := []*genai.Content{
		{
			Role:  "user",
			Parts: []*genai.Part{genai.NewPartFromText(userMessage)},
		},
	}

	var lastErr error
	for attempt := range maxRetries {
		result, err := gc.client.Models.GenerateContent(ctx, model, contents, config)
		if err != nil {
			lastErr = err
			if attempt < maxRetries-1 {
				wait := time.Duration(math.Min(float64(int(1)<<attempt), 30)) * time.Second
				fmt.Fprintf(os.Stderr, "\n  [warn] Gemini error: %v, retrying in %v...\n", err, wait)
				time.Sleep(wait)
				continue
			}
			break
		}

		if len(result.Candidates) == 0 {
			fmt.Fprintf(os.Stderr, "\n  [debug] Gemini returned 0 candidates\n")
			return "", 0, 0, nil
		}

		cand := result.Candidates[0]
		if cand.FinishReason != "" && cand.FinishReason != genai.FinishReasonStop {
			fmt.Fprintf(os.Stderr, "\n  [debug] Gemini finish reason: %v\n", cand.FinishReason)
		}

		text := extractText(result)
		var inTokens, outTokens int32
		if result.UsageMetadata != nil {
			inTokens = result.UsageMetadata.PromptTokenCount
			outTokens = result.UsageMetadata.CandidatesTokenCount
		}
		return text, int(inTokens), int(outTokens), nil
	}

	return "", 0, 0, fmt.Errorf("gemini call failed after %d retries: %w", maxRetries, lastErr)
}

func extractText(result *genai.GenerateContentResponse) string {
	if result == nil || len(result.Candidates) == 0 {
		return ""
	}
	cand := result.Candidates[0]
	if cand.Content == nil {
		return ""
	}
	var out strings.Builder
	for _, part := range cand.Content.Parts {
		if part.Text != "" {
			out.WriteString(part.Text)
		}
		if fc := part.FunctionCall; fc != nil {
			fmt.Fprintf(&out, "\n[TOOL_CALL] %s(%v)\n", fc.Name, fc.Args)
		}
	}
	return out.String()
}
