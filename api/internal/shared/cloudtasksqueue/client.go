package cloudtasksqueue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"classgame-api/internal/shared/requestmeta"
)

const (
	defaultDispatchDeadline = 10 * time.Minute
	createTaskTimeout       = 10 * time.Second
	minimumDispatchDeadline = 15 * time.Second
	maximumDispatchDeadline = 30 * time.Minute
)

type Options struct {
	ProjectID          string
	Location           string
	Queue              string
	TargetURL          string
	OIDCServiceAccount string
	OIDCAudience       string
	CredentialsFile    string
	CredentialsJSON    string
	DispatchDeadline   time.Duration
}

type Task struct {
	ID           string
	Body         []byte
	Headers      map[string]string
	ScheduleTime time.Time
}

type Client struct {
	client             *cloudtasks.Client
	parent             string
	targetURL          string
	oidcServiceAccount string
	oidcAudience       string
	dispatchDeadline   time.Duration
}

func Connect(ctx context.Context, input Options) (*Client, error) {
	resolved, err := resolveOptions(input)
	if err != nil {
		return nil, err
	}

	clientOptions, err := buildClientOptions(resolved)
	if err != nil {
		return nil, err
	}

	client, err := cloudtasks.NewClient(ctx, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("create cloud tasks client: %w", err)
	}

	return &Client{
		client:             client,
		parent:             fmt.Sprintf("projects/%s/locations/%s/queues/%s", resolved.ProjectID, resolved.Location, resolved.Queue),
		targetURL:          resolved.TargetURL,
		oidcServiceAccount: resolved.OIDCServiceAccount,
		oidcAudience:       resolved.OIDCAudience,
		dispatchDeadline:   resolved.DispatchDeadline,
	}, nil
}

func (client *Client) Enqueue(ctx context.Context, input Task) error {
	if client == nil || client.client == nil {
		return errors.New("cloud tasks client is not configured")
	}
	if err := validateTask(input); err != nil {
		return err
	}

	task := client.buildTask(ctx, input)
	createContext, cancel := context.WithTimeout(ctx, createTaskTimeout)
	defer cancel()

	_, err := client.client.CreateTask(createContext, &taskspb.CreateTaskRequest{
		Parent: client.parent,
		Task:   task,
	})
	if status.Code(err) == codes.AlreadyExists {
		slog.InfoContext(ctx, "cloud task already exists", "task_id", input.ID,
			"request_id", requestmeta.RequestID(ctx), "trace_id", requestmeta.TraceID(ctx))
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue cloud task: %w", err)
	}
	slog.InfoContext(ctx, "cloud task enqueued", "task_id", input.ID,
		"request_id", requestmeta.RequestID(ctx), "trace_id", requestmeta.TraceID(ctx))

	return nil
}

func (client *Client) Close() error {
	if client == nil || client.client == nil {
		return nil
	}

	return client.client.Close()
}

func (client *Client) buildTask(ctx context.Context, input Task) *taskspb.Task {
	carrier := make(http.Header, len(input.Headers)+3)
	for key, value := range input.Headers {
		carrier.Set(key, value)
	}
	if carrier.Get("Content-Type") == "" {
		carrier.Set("Content-Type", "application/json")
	}
	requestmeta.Inject(ctx, carrier)
	headers := make(map[string]string, len(carrier))
	for key := range carrier {
		headers[key] = carrier.Get(key)
	}

	httpRequest := &taskspb.HttpRequest{
		Url:        client.targetURL,
		HttpMethod: taskspb.HttpMethod_POST,
		Headers:    headers,
		Body:       input.Body,
		AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
			OidcToken: &taskspb.OidcToken{
				ServiceAccountEmail: client.oidcServiceAccount,
				Audience:            client.oidcAudience,
			},
		},
	}

	task := &taskspb.Task{
		Name: client.parent + "/tasks/" + input.ID,
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: httpRequest,
		},
		DispatchDeadline: durationpb.New(client.dispatchDeadline),
	}
	if !input.ScheduleTime.IsZero() {
		task.ScheduleTime = timestamppb.New(input.ScheduleTime)
	}

	return task
}

func resolveOptions(input Options) (Options, error) {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.Location = strings.TrimSpace(input.Location)
	input.Queue = strings.TrimSpace(input.Queue)
	input.TargetURL = strings.TrimSpace(input.TargetURL)
	input.OIDCServiceAccount = strings.TrimSpace(input.OIDCServiceAccount)
	input.OIDCAudience = strings.TrimSpace(input.OIDCAudience)
	input.CredentialsFile = strings.TrimSpace(input.CredentialsFile)
	input.CredentialsJSON = strings.TrimSpace(input.CredentialsJSON)

	for field, value := range map[string]string{
		"project id":           input.ProjectID,
		"location":             input.Location,
		"queue":                input.Queue,
		"target url":           input.TargetURL,
		"oidc service account": input.OIDCServiceAccount,
		"oidc audience":        input.OIDCAudience,
	} {
		if value == "" {
			return Options{}, fmt.Errorf("cloud tasks %s is required", field)
		}
	}
	if err := validateHTTPSURL(input.TargetURL); err != nil {
		return Options{}, fmt.Errorf("cloud tasks target url: %w", err)
	}
	if err := validateHTTPSURL(input.OIDCAudience); err != nil {
		return Options{}, fmt.Errorf("cloud tasks oidc audience: %w", err)
	}
	if input.CredentialsFile != "" && input.CredentialsJSON != "" {
		return Options{}, errors.New("configure either cloud tasks credentials file or json, not both")
	}

	if input.DispatchDeadline == 0 {
		input.DispatchDeadline = defaultDispatchDeadline
	}
	if input.DispatchDeadline < minimumDispatchDeadline || input.DispatchDeadline > maximumDispatchDeadline {
		return Options{}, fmt.Errorf("cloud tasks dispatch deadline must be between %s and %s", minimumDispatchDeadline, maximumDispatchDeadline)
	}

	return input, nil
}

func validateHTTPSURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("must be an absolute https URL")
	}

	return nil
}

func validateTask(input Task) error {
	if strings.TrimSpace(input.ID) == "" {
		return errors.New("cloud task id is required")
	}
	if len(input.ID) > 500 {
		return errors.New("cloud task id cannot be longer than 500 characters")
	}
	for _, character := range input.ID {
		if !((character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_') {
			return errors.New("cloud task id contains an invalid character")
		}
	}
	if len(input.Body) > 100*1024 {
		return errors.New("cloud task body cannot be larger than 100 KB")
	}

	return nil
}

func buildClientOptions(input Options) ([]option.ClientOption, error) {
	if input.CredentialsJSON != "" {
		return []option.ClientOption{option.WithCredentialsJSON([]byte(input.CredentialsJSON))}, nil
	}
	if input.CredentialsFile != "" {
		return []option.ClientOption{option.WithCredentialsFile(input.CredentialsFile)}, nil
	}

	return nil, nil
}
