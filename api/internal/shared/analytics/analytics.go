package analytics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"time"

	posthog "github.com/posthog/posthog-go"
)

const (
	defaultServiceName       = "classgame-api"
	maxExceptionMessageRunes = 2_000
)

type Options struct {
	Enabled      bool
	Environment  string
	Host         string
	ProjectToken string
	ServiceName  string
}

type Tracker interface {
	Enabled() bool
	Capture(context.Context, Event, string, Properties)
	CaptureException(context.Context, error, Properties)
	RequestContextMiddleware(http.Handler) http.Handler
	Close() error
}

type Client struct {
	client posthog.Client
}

func New(options Options) (*Client, error) {
	if !options.Enabled {
		return &Client{}, nil
	}

	projectToken := strings.TrimSpace(options.ProjectToken)
	if projectToken == "" {
		return nil, errors.New("posthog project token is required when analytics is enabled")
	}

	serviceName := strings.TrimSpace(options.ServiceName)
	if serviceName == "" {
		serviceName = defaultServiceName
	}

	client, err := posthog.NewWithConfig(projectToken, posthog.Config{
		Endpoint:        strings.TrimRight(strings.TrimSpace(options.Host), "/"),
		ShutdownTimeout: 5 * time.Second,
		DefaultEventProperties: posthog.NewProperties().
			Set("environment", strings.TrimSpace(options.Environment)).
			Set("service", serviceName),
	})
	if err != nil {
		return nil, fmt.Errorf("initialize PostHog client: %w", err)
	}

	return &Client{client: client}, nil
}

func (client *Client) Enabled() bool {
	return client != nil && client.client != nil
}

func (client *Client) Capture(
	ctx context.Context,
	event Event,
	distinctID string,
	properties Properties,
) {
	if !client.Enabled() {
		return
	}

	message := posthog.Capture{
		DistinctId: strings.TrimSpace(distinctID),
		Event:      string(event),
		Properties: toPostHogProperties(properties),
	}
	if err := posthog.EnqueueWithContext(contextOrBackground(ctx), client.client, message); err != nil {
		slog.Warn("posthog event enqueue failed", "error", err, "event", event)
	}
}

func (client *Client) CaptureException(
	ctx context.Context,
	err error,
	properties Properties,
) {
	if !client.Enabled() {
		return
	}
	if err == nil {
		err = errors.New("api request returned an unexpected server error")
	}

	exception := posthog.NewDefaultException(
		time.Now().UTC(),
		"",
		exceptionType(err),
		truncateExceptionMessage(err.Error()),
	)
	exception.Properties = toPostHogProperties(properties)
	if fingerprint := exceptionFingerprint(properties); fingerprint != "" {
		exception.ExceptionFingerprint = &fingerprint
	}

	enqueueContext := contextOrBackground(ctx)
	if _, exists := posthog.RequestContextFromContext(enqueueContext); !exists {
		enqueueContext = posthog.WithFreshRequestContext(enqueueContext, posthog.RequestContext{})
	}
	if enqueueErr := posthog.EnqueueWithContext(enqueueContext, client.client, exception); enqueueErr != nil {
		slog.Warn("posthog exception enqueue failed", "error", enqueueErr)
	}
}

func (client *Client) RequestContextMiddleware(next http.Handler) http.Handler {
	if !client.Enabled() {
		return next
	}

	return posthog.NewRequestContextMiddleware(
		next,
		posthog.WithCapturePanics(client.client),
	)
}

func (client *Client) Close() error {
	if !client.Enabled() {
		return nil
	}

	return client.client.Close()
}

func Track(
	tracker Tracker,
	ctx context.Context,
	event Event,
	distinctID string,
	properties Properties,
) {
	if tracker == nil {
		return
	}

	tracker.Capture(ctx, event, distinctID, properties)
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

func exceptionFingerprint(properties Properties) string {
	if properties == nil {
		return ""
	}
	statusCode, hasStatusCode := properties["http_status"]
	if !hasStatusCode {
		return ""
	}

	return fmt.Sprintf(
		"mia-api:%v:%v:%v",
		statusCode,
		properties["http_method"],
		properties["http_route"],
	)
}

func exceptionType(err error) string {
	errorType := reflect.TypeOf(err)
	if errorType == nil {
		return "APIError"
	}

	return strings.TrimPrefix(errorType.String(), "*")
}

func truncateExceptionMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "api request returned an unexpected server error"
	}

	runes := []rune(message)
	if len(runes) <= maxExceptionMessageRunes {
		return message
	}

	return string(runes[:maxExceptionMessageRunes])
}

func toPostHogProperties(properties Properties) posthog.Properties {
	result := posthog.NewProperties()
	for key, value := range properties {
		if strings.TrimSpace(key) == "" {
			continue
		}

		result.Set(key, value)
	}

	return result
}
