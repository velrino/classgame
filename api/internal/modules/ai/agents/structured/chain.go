// Package structured runs stateless LangChainGo workflows with ClassGame's
// structured generation client. The client owns budgets, file parts and decoding.
package structured

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"classgame-api/internal/shared/requestmeta"

	"classgame-api/internal/modules/ai/providers/genai"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/prompts"
)

func RunJSON(ctx context.Context, client genai.StructuredGenerator, template string, input any, request genai.StructuredRequest, target any) (genai.StructuredResult, error) {
	payload, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return genai.StructuredResult{}, err
	}
	prompt := prompts.NewPromptTemplate(template, []string{"input"})
	return Run(ctx, client, prompt, map[string]any{"input": string(payload)}, request, target)
}

func Run(ctx context.Context, client genai.StructuredGenerator, prompt prompts.PromptTemplate, values map[string]any, request genai.StructuredRequest, target any) (genai.StructuredResult, error) {
	chain := chains.NewTransform(func(ctx context.Context, inputs map[string]any, _ ...chains.ChainCallOption) (map[string]any, error) {
		text, err := prompt.Format(inputs)
		if err != nil {
			return nil, err
		}
		request.Prompt = text
		usage, err := client.GenerateStructured(ctx, request, target)
		return map[string]any{"result": usage}, err
	}, prompt.InputVariables, []string{"result"})

	start := time.Now()
	outputs, err := chains.Call(ctx, chain, values)
	result, _ := outputs["result"].(genai.StructuredResult)
	slog.InfoContext(ctx, "langchain workflow", "workflow", request.Labels["workflow"], "request_id", requestmeta.RequestID(ctx), "trace_id", requestmeta.TraceID(ctx), "duration_ms", time.Since(start).Milliseconds(), "success", err == nil)
	return result, err
}
