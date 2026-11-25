package genai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	sdk "google.golang.org/genai"
)

func (client *Client) logUsage(ctx context.Context, request StructuredRequest, usage *sdk.GenerateContentResponseUsageMetadata, elapsed time.Duration) {
	fields := []any{
		"provider", client.Provider(), "model", request.Model, "workflow", request.Labels["workflow"],
		"duration_ms", elapsed.Milliseconds(), "input_tokens", usage.PromptTokenCount,
		"cached_input_tokens", usage.CachedContentTokenCount, "output_tokens", usage.CandidatesTokenCount,
		"thinking_tokens", usage.ThoughtsTokenCount, "total_tokens", usage.TotalTokenCount,
	}
	if client.options.RequestID != nil {
		fields = append(fields, "request_id", client.options.RequestID(ctx))
	}
	estimate := "unavailable"
	if client.Provider() == ProviderGeminiAPI && strings.TrimPrefix(request.Model, "models/") == "gemini-3.1-flash-lite" && usage.ToolUsePromptTokenCount == 0 {
		// Gemini API standard paid rates per million tokens, including implicit cache discounts.
		// https://ai.google.dev/gemini-api/docs/pricing#gemini-3.1-flash-lite
		cost := float64(usage.PromptTokenCount-usage.CachedContentTokenCount)*0.25 + float64(usage.CachedContentTokenCount)*0.025
		for _, detail := range usage.PromptTokensDetails {
			if detail != nil && detail.Modality == sdk.MediaModalityAudio {
				cost += float64(detail.TokenCount) * 0.25
			}
		}
		for _, detail := range usage.CacheTokensDetails {
			if detail != nil && detail.Modality == sdk.MediaModalityAudio {
				cost -= float64(detail.TokenCount) * 0.225
			}
		}
		cost += (float64(usage.CandidatesTokenCount) + float64(usage.ThoughtsTokenCount)) * 1.50
		estimate = fmt.Sprintf("%.8f", cost/1_000_000)
	}
	slog.InfoContext(ctx, "genai usage", append(fields, "estimated_standard_cost_usd", estimate)...)
}
