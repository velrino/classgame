// evalgenkit scores a single synthetic example from the comparison runner.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/evaluators"
	sdk "google.golang.org/genai"
)

const model = "gemini-3.6-flash"

type verdict struct {
	Pass   *bool  `json:"pass"`
	Reason string `json:"reason"`
}

func judge(ctx context.Context, prompt string) (*ai.Score, error) {
	if prompt == "" || len(prompt) > 6000 {
		return nil, errors.New("judge input must be 1 to 6000 bytes")
	}
	client, err := sdk.NewClient(ctx, &sdk.ClientConfig{
		Backend: sdk.BackendVertexAI, Project: os.Getenv("GENAI_PROJECT_ID"), Location: "global",
		HTTPOptions: sdk.HTTPOptions{RetryOptions: &sdk.HTTPRetryOptions{Attempts: sdk.Ptr(int32(1))}},
	})
	if err != nil {
		return nil, err
	}
	response, err := client.Models.GenerateContent(ctx, model, sdk.Text(prompt), &sdk.GenerateContentConfig{
		Temperature: sdk.Ptr(float32(0)), MaxOutputTokens: 512,
		ThinkingConfig:     &sdk.ThinkingConfig{ThinkingLevel: sdk.ThinkingLevelMinimal},
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: map[string]any{"type": "object", "properties": map[string]any{"pass": map[string]any{"type": "boolean"}, "reason": map[string]any{"type": "string"}}, "required": []string{"pass", "reason"}},
	})
	if err != nil {
		return nil, err
	}
	var v verdict
	if err := json.Unmarshal([]byte(response.Text()), &v); err != nil {
		return nil, err
	}
	u := response.UsageMetadata
	if v.Pass == nil || v.Reason == "" || u == nil || u.TotalTokenCount <= 0 {
		return nil, errors.New("incomplete verdict or usage")
	}
	status := ai.ScoreStatusFail
	if *v.Pass {
		status = ai.ScoreStatusPass
	}
	return &ai.Score{Id: "source_only", Status: status.String(), Score: *v.Pass, Details: map[string]any{
		"reason": v.Reason, "model_version": response.ModelVersion, "input_tokens": u.PromptTokenCount,
		"output_tokens": u.CandidatesTokenCount, "thinking_tokens": u.ThoughtsTokenCount, "total_tokens": u.TotalTokenCount,
	}}, nil
}

func run() error {
	if len(os.Args) != 2 {
		return errors.New("expected ui, ids, genkit-live, or judge-live")
	}
	if os.Args[1] == "ui" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		genkit.Init(ctx, genkit.WithPlugins(&evaluators.GenkitEval{
			Metrics: []evaluators.MetricConfig{{MetricType: evaluators.EvaluatorDeepEqual}},
		}))
		<-ctx.Done()
		return nil
	}
	var example ai.Example
	if err := json.NewDecoder(os.Stdin).Decode(&example); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	var evaluator ai.Evaluator
	switch os.Args[1] {
	case "ids":
		evaluator = evaluators.ConfigureMetric(evaluators.MetricConfig{MetricType: evaluators.EvaluatorDeepEqual})
	case "genkit-live", "judge-live":
		prompt, ok := example.Input.(string)
		if !ok {
			return errors.New("expected a judge prompt")
		}
		if os.Args[1] == "judge-live" {
			score, err := judge(ctx, prompt)
			if err != nil {
				return err
			}
			return json.NewEncoder(os.Stdout).Encode(score)
		}
		evaluator = ai.NewEvaluatorAction("classgame/source_only", &ai.EvaluatorOptions{IsBilled: true}, func(ctx context.Context, req *ai.EvaluatorCallbackRequest, _ struct{}) (*ai.EvaluationResult, error) {
			score, err := judge(ctx, prompt)
			if err != nil {
				return nil, err
			}
			return &ai.EvaluationResult{TestCaseId: req.Input.TestCaseId, Evaluation: []ai.Score{*score}}, nil
		})
	default:
		return errors.New("unknown evaluation mode")
	}
	results, err := evaluator.Evaluate(ctx, &ai.EvaluatorRequest{Dataset: []*ai.Example{&example}, EvaluationId: "classgame-framework-comparison"})
	if err != nil {
		return err
	}
	if len(*results) != 1 || len((*results)[0].Evaluation) != 1 {
		return errors.New("expected one score")
	}
	score := (*results)[0].Evaluation[0]
	if score.Error != "" {
		return errors.New(score.Error)
	}
	return json.NewEncoder(os.Stdout).Encode(score)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
