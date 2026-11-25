// evalretrievalmodel is a separate, opt-in synthetic live experiment. It is not
// used by the offline evaluation runner or the deployed API.
package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"classgame-api/internal/modules/generation"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/students"
	"classgame-api/internal/shared/genai"
	sdk "google.golang.org/genai"
)

//go:embed compact.txt
var compactInstruction string

type provider struct {
	variant         string
	instructionHash string
	client          *sdk.Client
	model           string
	calls           int
}

func (p *provider) IsEnabled() bool      { return true }
func (p *provider) DefaultModel() string { return p.model }
func (p *provider) Provider() string     { return "google_genai_vertex_ai" }
func (p *provider) GenerateStructured(ctx context.Context, request genai.StructuredRequest, target any) (genai.StructuredResult, error) {
	if p.calls >= 1 {
		return genai.StructuredResult{}, errors.New("live experiment call budget exhausted")
	}
	p.calls++
	attempts := int32(1)
	if p.variant == "compact" {
		request.SystemInstruction = strings.TrimSpace(compactInstruction)
	}
	p.instructionHash = fmt.Sprintf("%x", sha256.Sum256([]byte(request.SystemInstruction)))
	temperature := request.Temperature
	response, err := p.client.Models.GenerateContent(ctx, p.model, sdk.Text(request.Prompt), &sdk.GenerateContentConfig{
		SystemInstruction: sdk.NewContentFromText(request.SystemInstruction, sdk.RoleUser),
		ResponseMIMEType:  "application/json", ResponseJsonSchema: request.ResponseJSONSchema,
		Temperature: &temperature, MaxOutputTokens: 2048,
		HTTPOptions: &sdk.HTTPOptions{RetryOptions: &sdk.HTTPRetryOptions{Attempts: &attempts}},
	})
	if err != nil {
		return genai.StructuredResult{}, errors.New("live provider request failed; inspect private stderr")
	}
	if err = json.Unmarshal([]byte(response.Text()), target); err != nil {
		return genai.StructuredResult{}, errors.New("live provider returned invalid JSON")
	}
	result := genai.StructuredResult{Model: p.model, ModelVersion: response.ModelVersion, Provider: p.Provider()}
	if usage := response.UsageMetadata; usage != nil {
		result.PromptTokenCount = usage.PromptTokenCount
		result.CandidatesTokenCount = usage.CandidatesTokenCount
		result.ThoughtsTokenCount = usage.ThoughtsTokenCount
		result.TotalTokenCount = usage.TotalTokenCount
	}
	return result, nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	flags := flag.NewFlagSet("evalretrievalmodel", flag.ContinueOnError)
	enabled := flags.Bool("run", false, "explicitly authorize one possible paid call")
	dataset := flags.String("dataset", "../evals/platform/retrieval-diagnostics/cases-v2.json", "synthetic fixture")
	caseID := flags.String("case", "retrieval-curly-quotes-v2", "synthetic case ID")
	phase := flags.String("phase", "", "before or after label; does not change retrieval")
	variant := flags.String("prompt", "current", "system instruction: current or compact (experiment only)")
	model := flags.String("model", "", "explicit model ID")
	output := flags.String("output", "", "new exclusive JSON output file")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if *variant != "current" && *variant != "compact" {
		return errors.New("unknown prompt variant")
	}
	if !*enabled || flags.NArg() != 0 || (*phase != "before" && *phase != "after") || *model == "" || *output == "" {
		return errors.New("requires -run, -phase before|after, -model and a new -output path; at most one provider request, no retries")
	}
	raw, err := os.ReadFile(*dataset)
	if err != nil {
		return err
	}
	var fixture struct {
		Synthetic bool                           `json:"synthetic"`
		Mission   missions.MissionDocument       `json:"mission"`
		Context   generation.BuildTutorTurnInput `json:"tutor_context"`
		Cases     []struct {
			ID        string `json:"id"`
			Retrieval struct {
				Turn  students.SaveStudentTutorTurnInput `json:"turn"`
				Limit int                                `json:"limit"`
			} `json:"retrieval"`
		} `json:"cases"`
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		return err
	}
	if !fixture.Synthetic || fixture.Mission.ID != "synthetic-water-cycle" {
		return errors.New("only the owned synthetic water-cycle fixture is allowed")
	}
	selected := -1
	for i, c := range fixture.Cases {
		if c.ID == *caseID {
			selected = i
		}
	}
	if selected < 0 {
		return errors.New("unknown synthetic case")
	}
	c := fixture.Cases[selected]
	if c.Retrieval.Limit < 1 || c.Retrieval.Limit > 5 {
		return errors.New("invalid retrieval limit")
	}
	file, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	attempts := int32(1)
	client, err := sdk.NewClient(ctx, &sdk.ClientConfig{Backend: sdk.BackendVertexAI, Project: os.Getenv("GENAI_PROJECT_ID"), Location: os.Getenv("GENAI_LOCATION"), HTTPOptions: sdk.HTTPOptions{RetryOptions: &sdk.HTTPRetryOptions{Attempts: &attempts}}})
	if err != nil {
		return errors.New("cannot initialize live provider; output file is incomplete")
	}
	p := &provider{client: client, model: *model, variant: *variant}
	fixture.Context.RetrievedChunks = students.RetrieveMissionKnowledge(fixture.Mission, c.Retrieval.Turn, c.Retrieval.Limit)
	fixture.Context.Turn.Prompt = c.Retrieval.Turn.Prompt
	start := time.Now()
	answer, err := generation.NewService(generation.Dependencies{Client: p}).BuildTutorTurn(ctx, fixture.Context)
	if err != nil {
		return errors.New("production tutor generation failed")
	}
	// Do not publish raw provider errors, local paths, private observations or prompts.
	if answer.FallbackReason != "" && p.calls > 0 {
		answer.FallbackReason = "Live provider call or response validation failed"
	}
	answer.TeacherObservation = ""
	ids := make([]string, 0, len(fixture.Context.RetrievedChunks))
	for _, chunk := range fixture.Context.RetrievedChunks {
		ids = append(ids, chunk.ID)
	}
	digest := sha256.Sum256(raw)
	revision, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	dirty, _ := exec.Command("git", "status", "--porcelain").Output()
	record := map[string]any{"prompt_variant": *variant, "system_instruction_sha256": p.instructionHash, "phase": *phase, "case_id": *caseID, "timestamp": time.Now().UTC().Format(time.RFC3339Nano), "revision": string(revision), "dirty": len(dirty) > 0, "dataset_sha256": hex.EncodeToString(digest[:]), "max_provider_calls": 1, "sdk_attempts": 1, "real_provider_calls": p.calls, "model_requested": *model, "retrieved_chunk_ids": ids, "elapsed_milliseconds": time.Since(start).Milliseconds(), "tutor": answer, "scope": "Synthetic, single live sample through production retrieval and tutor generation; dedicated SDK adapter with no retries, no database or application configuration loading."}
	if err = json.NewEncoder(file).Encode(record); err != nil {
		return err
	}
	fmt.Printf("%s: provider calls=%d generated=%t sources=%v\n", *phase, p.calls, answer.GeneratedByAI, ids)
	if p.calls > 0 && !answer.GeneratedByAI {
		return errors.New("provider did not produce an accepted live answer; inspect the saved observation")
	}
	return nil
}
