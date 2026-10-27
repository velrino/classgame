package genai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"golang.org/x/sync/singleflight"
	googlegenai "google.golang.org/genai"

	"classgame-api/internal/platform/requestmeta"
)

const (
	ProviderGeminiAPI = "google_genai_gemini_api"
	ProviderVertexAI  = "google_genai_vertex_ai"

	maximumConcurrentGenerations       = 4
	maximumAdmittedCalls               = 16
	maximumPromptBytes                 = 256 * 1024
	maximumInlineBytes                 = 20 * 1024 * 1024
	maximumInlineParts                 = 5
	maximumResponseBytes               = 256 * 1024
	maximumOutputTokens          int32 = 8192
	maximumRequestTimeout              = 2 * time.Minute
)

var (
	ErrDisabled      = errors.New("genai is disabled")
	ErrEmptyResponse = errors.New("genai returned an empty response")
	ErrBusy          = errors.New("genai process concurrency limit reached")
	ErrInputBudget   = errors.New("genai input exceeds the request budget")
	ErrOutputBudget  = errors.New("genai response exceeds the output budget")
)

type Options struct {
	Admission       ProviderAdmission
	Enabled         bool
	Backend         string
	Model           string
	APIKey          string
	ProjectID       string
	Location        string
	CredentialsFile string
	CredentialsJSON string
	RequestTimeout  time.Duration
	MaxOutputTokens int32
}

type Client struct {
	client        *googlegenai.Client
	options       Options
	providerSlots chan struct{}
	callSlots     chan struct{}
	inflight      singleflight.Group
}

type StructuredRequest struct {
	SystemInstruction  string
	Prompt             string
	FileParts          []FilePart
	ResponseJSONSchema any
	Temperature        float32
	MaxOutputTokens    int32
	Model              string
	Labels             map[string]string
}

type FilePart struct {
	Data     []byte
	MIMEType string
}

type StructuredResult struct {
	Model                string
	ModelVersion         string
	Provider             string
	PromptTokenCount     int32
	CandidatesTokenCount int32
	ThoughtsTokenCount   int32
	TotalTokenCount      int32
	RawText              string
}

func Connect(ctx context.Context, input Options) (*Client, error) {
	options := resolveOptions(input)
	if !options.Enabled {
		return &Client{options: options}, nil
	}
	if options.Admission == nil {
		return nil, errors.New("enabled genai requires shared provider admission")
	}

	clientConfig := &googlegenai.ClientConfig{
		APIKey:   options.APIKey,
		Backend:  resolveBackend(options.Backend),
		Project:  options.ProjectID,
		Location: options.Location,
		HTTPOptions: googlegenai.HTTPOptions{
			Timeout: &options.RequestTimeout,
		},
	}

	credentials, err := loadCredentials(options)
	if err != nil {
		return nil, err
	}
	clientConfig.Credentials = credentials

	client, err := googlegenai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}

	return &Client{
		client:        client,
		options:       options,
		providerSlots: make(chan struct{}, maximumConcurrentGenerations),
		callSlots:     make(chan struct{}, maximumAdmittedCalls),
	}, nil
}

func (client *Client) IsEnabled() bool {
	return client != nil && client.options.Enabled && client.client != nil
}

func (client *Client) Provider() string {
	if client == nil || client.options.Backend == "gemini_api" {
		return ProviderGeminiAPI
	}

	return ProviderVertexAI
}

func (client *Client) DefaultModel() string {
	if client == nil {
		return ""
	}

	return client.options.Model
}

func (client *Client) GenerateStructured(
	ctx context.Context,
	request StructuredRequest,
	target any,
) (StructuredResult, error) {
	if !client.IsEnabled() {
		return StructuredResult{}, ErrDisabled
	}
	if target == nil {
		return StructuredResult{}, errors.New("target is required")
	}
	if err := ctx.Err(); err != nil {
		return StructuredResult{}, err
	}
	select {
	case client.callSlots <- struct{}{}:
	default:
		return StructuredResult{}, ErrBusy
	}
	releaseAdmission := true
	defer func() {
		if releaseAdmission {
			<-client.callSlots
		}
	}()

	request, schemaJSON, err := client.prepareRequest(request)
	if err != nil {
		return StructuredResult{}, err
	}
	callContext, cancel := context.WithTimeout(ctx, client.options.RequestTimeout)
	defer cancel()

	var result StructuredResult
	if scope := requestmeta.GenerationScope(ctx); scope != "" {
		key, err := generationKey(scope, request, schemaJSON)
		if err != nil {
			return StructuredResult{}, err
		}
		pending := client.inflight.DoChan(key, func() (any, error) {
			return client.generate(callContext, request)
		})
		select {
		case completed := <-pending:
			if completed.Err != nil {
				return StructuredResult{}, completed.Err
			}
			result = completed.Val.(StructuredResult)
		case <-callContext.Done():
			// Keep admission until the shared call releases its retained result channel.
			// This bounds canceled waiters without detaching provider cancellation.
			releaseAdmission = false
			go func() {
				<-pending
				<-client.callSlots
			}()
			return StructuredResult{}, callContext.Err()
		}
	} else {
		result, err = client.generate(callContext, request)
		if err != nil {
			return StructuredResult{}, err
		}
	}
	if err := callContext.Err(); err != nil {
		return StructuredResult{}, err
	}
	if err := json.Unmarshal([]byte(result.RawText), target); err != nil {
		return result, fmt.Errorf("decode structured genai response: %w", err)
	}
	return result, nil
}

func (client *Client) prepareRequest(request StructuredRequest) (StructuredRequest, []byte, error) {
	schemaJSON, err := json.Marshal(request.ResponseJSONSchema)
	if err != nil {
		return StructuredRequest{}, nil, fmt.Errorf("encode genai response schema: %w", err)
	}
	if len(request.SystemInstruction)+len(request.Prompt)+len(schemaJSON) > maximumPromptBytes ||
		len(request.FileParts) > maximumInlineParts {
		return StructuredRequest{}, nil, ErrInputBudget
	}
	inlineBytes := 0
	for _, part := range request.FileParts {
		if len(part.Data) > maximumInlineBytes-inlineBytes {
			return StructuredRequest{}, nil, ErrInputBudget
		}
		inlineBytes += len(part.Data)
	}
	request.Model = strings.TrimSpace(request.Model)
	if request.Model == "" {
		request.Model = client.options.Model
	}
	if request.MaxOutputTokens <= 0 {
		request.MaxOutputTokens = client.options.MaxOutputTokens
	}
	request.MaxOutputTokens = min(request.MaxOutputTokens, client.options.MaxOutputTokens, maximumOutputTokens)
	return request, schemaJSON, nil
}

func generationKey(scope string, request StructuredRequest, schemaJSON []byte) (string, error) {
	digest := sha256.New()
	metadata := struct {
		Instruction string
		Prompt      string
		Model       string
		Schema      json.RawMessage
		Temperature float32
		Output      int32
		Labels      map[string]string
	}{request.SystemInstruction, request.Prompt, request.Model, schemaJSON,
		request.Temperature, request.MaxOutputTokens, request.Labels}
	if err := json.NewEncoder(digest).Encode(metadata); err != nil {
		return "", fmt.Errorf("fingerprint genai request: %w", err)
	}
	for _, part := range request.FileParts {
		file := struct {
			MIMEType string
			Digest   [32]byte
		}{part.MIMEType, sha256.Sum256(part.Data)}
		if err := json.NewEncoder(digest).Encode(file); err != nil {
			return "", fmt.Errorf("fingerprint genai attachment: %w", err)
		}
	}
	return scope + ":" + hex.EncodeToString(digest.Sum(nil)), nil
}

func (client *Client) generate(ctx context.Context, request StructuredRequest) (result StructuredResult, err error) {
	// singleflight runs asynchronously, outside HTTP panic recovery.
	defer func() {
		if recover() != nil {
			result = StructuredResult{}
			err = errors.New("genai provider call failed unexpectedly")
		}
	}()
	select {
	case client.providerSlots <- struct{}{}:
		defer func() { <-client.providerSlots }()
	default:
		return StructuredResult{}, ErrBusy
	}

	model := request.Model
	temperature := request.Temperature
	responseSchema, responseJSONSchema := responseSchemaConfig(request.ResponseJSONSchema)

	config := &googlegenai.GenerateContentConfig{
		MaxOutputTokens:    request.MaxOutputTokens,
		ResponseMIMEType:   "application/json",
		ResponseSchema:     responseSchema,
		ResponseJsonSchema: responseJSONSchema,
		Temperature:        &temperature,
		Labels:             request.Labels,
		HTTPOptions: &googlegenai.HTTPOptions{
			Timeout: &client.options.RequestTimeout,
		},
		SystemInstruction: googlegenai.NewContentFromText(request.SystemInstruction, googlegenai.RoleUser),
		ThinkingConfig:    thinkingConfigForModel(model),
	}

	var response *googlegenai.GenerateContentResponse
	err = RunAdmitted(ctx, client.options.Admission, request.MaxOutputTokens, func() error {
		var providerErr error
		response, providerErr = client.client.Models.GenerateContent(ctx, model, buildUserContents(request), config)
		return providerErr
	})
	if err != nil {
		return StructuredResult{}, fmt.Errorf("generate content: %w", err)
	}

	text := normalizeJSONText(response.Text())
	if text == "" {
		return StructuredResult{}, ErrEmptyResponse
	}
	if len(text) > maximumResponseBytes {
		return StructuredResult{}, ErrOutputBudget
	}

	return buildStructuredResult(model, client.Provider(), text, response), nil
}

func thinkingConfigForModel(model string) *googlegenai.ThinkingConfig {
	model = strings.ToLower(strings.TrimSpace(model))
	if separator := strings.LastIndex(model, "/"); separator >= 0 {
		model = model[separator+1:]
	}

	if strings.HasPrefix(model, "gemini-3") {
		return &googlegenai.ThinkingConfig{
			ThinkingLevel: googlegenai.ThinkingLevelLow,
		}
	}
	if strings.HasPrefix(model, "gemini-2.5-flash") {
		thinkingBudget := int32(0)
		return &googlegenai.ThinkingConfig{
			ThinkingBudget: &thinkingBudget,
		}
	}

	return nil
}

func buildUserContents(request StructuredRequest) []*googlegenai.Content {
	parts := []*googlegenai.Part{googlegenai.NewPartFromText(request.Prompt)}
	for _, filePart := range request.FileParts {
		if len(filePart.Data) == 0 || strings.TrimSpace(filePart.MIMEType) == "" {
			continue
		}

		parts = append(parts, googlegenai.NewPartFromBytes(filePart.Data, strings.TrimSpace(filePart.MIMEType)))
	}

	return []*googlegenai.Content{
		googlegenai.NewContentFromParts(parts, googlegenai.RoleUser),
	}
}

func responseSchemaConfig(schema any) (*googlegenai.Schema, any) {
	if schema == nil {
		return nil, nil
	}
	if typed, ok := schema.(*googlegenai.Schema); ok {
		return typed, nil
	}

	return nil, sanitizeResponseJSONSchema(schema)
}

func sanitizeResponseJSONSchema(schema any) any {
	return sanitizeResponseJSONSchemaValue(schema)
}

func sanitizeResponseJSONSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, child := range typed {
			if key == "maxItems" {
				continue
			}
			sanitized[key] = sanitizeResponseJSONSchemaValue(child)
		}
		return sanitized
	case []any:
		sanitized := make([]any, len(typed))
		for index, child := range typed {
			sanitized[index] = sanitizeResponseJSONSchemaValue(child)
		}
		return sanitized
	case []string:
		sanitized := make([]string, len(typed))
		copy(sanitized, typed)
		return sanitized
	default:
		return value
	}
}

func resolveOptions(input Options) Options {
	input.Backend = strings.TrimSpace(input.Backend)
	input.Model = strings.TrimSpace(input.Model)
	input.APIKey = strings.TrimSpace(input.APIKey)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.Location = strings.TrimSpace(input.Location)
	input.CredentialsFile = strings.TrimSpace(input.CredentialsFile)
	input.CredentialsJSON = strings.TrimSpace(input.CredentialsJSON)

	if input.Backend == "" {
		input.Backend = "vertex"
	}
	if input.Location == "" {
		input.Location = "global"
	}
	if input.RequestTimeout <= 0 {
		input.RequestTimeout = 30 * time.Second
	}
	input.RequestTimeout = min(input.RequestTimeout, maximumRequestTimeout)
	if input.MaxOutputTokens <= 0 {
		input.MaxOutputTokens = 4096
	}
	input.MaxOutputTokens = min(input.MaxOutputTokens, maximumOutputTokens)

	return input
}

func resolveBackend(value string) googlegenai.Backend {
	if value == "gemini_api" {
		return googlegenai.BackendGeminiAPI
	}

	return googlegenai.BackendVertexAI
}

func loadCredentials(options Options) (*auth.Credentials, error) {
	if options.CredentialsFile == "" && options.CredentialsJSON == "" {
		return nil, nil
	}
	if options.CredentialsFile != "" && options.CredentialsJSON != "" {
		return nil, errors.New("configure either genai credentials file or JSON, not both")
	}

	var raw []byte
	var err error
	if options.CredentialsFile != "" {
		raw, err = os.ReadFile(options.CredentialsFile)
		if err != nil {
			return nil, fmt.Errorf("read genai credentials file: %w", err)
		}
	} else {
		raw = []byte(options.CredentialsJSON)
	}

	credentialType, err := detectCredentialType(raw)
	if err != nil {
		return nil, err
	}

	return credentials.NewCredentialsFromJSON(credentialType, raw, &credentials.DetectOptions{
		Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
	})
}

func detectCredentialType(raw []byte) (credentials.CredType, error) {
	var body struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", fmt.Errorf("decode genai credentials header: %w", err)
	}

	switch credentials.CredType(body.Type) {
	case credentials.ServiceAccount:
		return credentials.ServiceAccount, nil
	case credentials.AuthorizedUser:
		return credentials.AuthorizedUser, nil
	default:
		return "", fmt.Errorf("unsupported genai credentials type %q", body.Type)
	}
}

func normalizeJSONText(value string) string {
	text := strings.TrimSpace(value)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")

	return strings.TrimSpace(text)
}

func buildStructuredResult(
	model string,
	provider string,
	text string,
	response *googlegenai.GenerateContentResponse,
) StructuredResult {
	result := StructuredResult{
		Model:        model,
		ModelVersion: response.ModelVersion,
		Provider:     provider,
		RawText:      text,
	}

	if response.UsageMetadata != nil {
		result.PromptTokenCount = response.UsageMetadata.PromptTokenCount
		result.CandidatesTokenCount = response.UsageMetadata.CandidatesTokenCount
		result.ThoughtsTokenCount = response.UsageMetadata.ThoughtsTokenCount
		result.TotalTokenCount = response.UsageMetadata.TotalTokenCount
	}

	return result
}
