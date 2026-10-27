// Package redisqueue delivers background messages through pinned Asynq workers.
package redisqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"classgame-api/internal/platform/requestmeta"
	"classgame-api/internal/taskqueue"
)

const (
	Attachments = "classgame:mission-attachments:v1"
	Learning    = "classgame:student-learning:v1"
	maxPayload  = 100 * 1024
)

var (
	ErrIDConflict = errors.New("background job ID has a different payload")
	ErrArchived   = errors.New("background job is archived; operator retry is required")
)

type Options struct {
	URL        string
	Queue      string
	Production bool
}

type envelope struct {
	Version  int               `json:"version"`
	ID       string            `json:"id"`
	Payload  json.RawMessage   `json:"payload"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Client struct {
	client    *asynq.Client
	inspector *asynq.Inspector
	redis     asynq.RedisConnOpt
	queue     string
}

type Destination struct {
	client *Client
	kind   string
}

func Connect(ctx context.Context, options Options) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(options.URL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "redis" && parsed.Scheme != "rediss") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("TASKS_REDIS_URL must be a Redis URL without query or fragment")
	}
	password := ""
	if parsed.User != nil {
		password, _ = parsed.User.Password()
	}
	if options.Production && (parsed.Scheme != "rediss" || password == "") {
		return nil, errors.New("production task Redis requires TLS and authentication")
	}
	if !safeName(options.Queue, 60) {
		return nil, errors.New("task queue name is invalid")
	}
	connection, err := asynq.ParseRedisURI(options.URL)
	if err != nil {
		return nil, errors.New("invalid task Redis connection")
	}
	if _, ok := connection.(asynq.RedisClientOpt); !ok {
		return nil, errors.New("task Redis requires a standalone compatible endpoint")
	}
	client := &Client{client: asynq.NewClient(connection), inspector: asynq.NewInspector(connection), redis: connection, queue: options.Queue}
	if err := ctx.Err(); err != nil {
		client.Close()
		return nil, err
	}
	if err := client.client.Ping(); err != nil {
		client.Close()
		return nil, errors.New("task Redis is unavailable")
	}
	return client, nil
}

func (client *Client) Destination(kind string) *Destination {
	return &Destination{client: client, kind: kind}
}

func (destination *Destination) Enqueue(ctx context.Context, message taskqueue.Message) error {
	if destination == nil || destination.client == nil || destination.kind == "" || !safeName(message.ID, 500) || len(message.Payload) > maxPayload {
		return errors.New("background message is invalid")
	}
	var payload any
	decoder := json.NewDecoder(bytes.NewReader(message.Payload))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return errors.New("background message requires JSON")
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("background message requires one JSON value")
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return errors.New("background message cannot be encoded")
	}
	carrier := make(http.Header)
	requestmeta.Inject(ctx, carrier)
	metadata := map[string]string{}
	for _, key := range []string{requestmeta.RequestIDHeader, "traceparent"} {
		if value := message.Metadata[key]; len(value) <= 128 && value != "" {
			metadata[key] = value
		}
		if value := carrier.Get(key); value != "" {
			metadata[key] = value
		}
	}
	encoded, err := json.Marshal(envelope{Version: 1, ID: message.ID, Payload: canonical, Metadata: metadata})
	if err != nil {
		return err
	}
	options := []asynq.Option{asynq.TaskID(message.ID), asynq.Queue(destination.client.queue), asynq.MaxRetry(10), asynq.Timeout(5 * time.Minute), asynq.Retention(24 * time.Hour)}
	if !message.ScheduleAt.IsZero() {
		options = append(options, asynq.ProcessAt(message.ScheduleAt))
	}
	_, err = destination.client.client.EnqueueContext(ctx, asynq.NewTask(destination.kind, encoded), options...)
	if !errors.Is(err, asynq.ErrTaskIDConflict) {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	existing, readErr := destination.client.inspector.GetTaskInfo(destination.client.queue, message.ID)
	if readErr != nil {
		return errors.New("cannot verify existing background job")
	}
	var previous envelope
	if json.Unmarshal(existing.Payload, &previous) != nil || existing.Type != destination.kind || !bytes.Equal(previous.Payload, canonical) {
		return ErrIDConflict
	}
	if existing.State == asynq.TaskStateArchived {
		return ErrArchived
	}
	return nil
}

func (client *Client) Close() error {
	first := client.client.Close()
	if err := client.inspector.Close(); first == nil {
		first = err
	}
	return first
}

// Retry reuses an archived job's stored body and identity; it never reconstructs business data.
func (client *Client) Retry(id string) error {
	if !safeName(id, 500) {
		return errors.New("job ID is invalid")
	}
	return client.inspector.RunTask(client.queue, id)
}

func (client *Client) Status(id string) (string, error) {
	if !safeName(id, 500) {
		return "", errors.New("job ID is invalid")
	}
	info, err := client.inspector.GetTaskInfo(client.queue, id)
	if err != nil {
		return "", err
	}
	return info.State.String(), nil
}

func (client *Client) Stats() (map[string]int, error) {
	info, err := client.inspector.GetQueueInfo(client.queue)
	if err != nil {
		return nil, err
	}
	return map[string]int{"pending": info.Pending, "active": info.Active, "scheduled": info.Scheduled, "retry": info.Retry, "archived": info.Archived, "completed": info.Completed}, nil
}

func safeName(value string, limit int) bool {
	if value == "" || len(value) > limit {
		return false
	}
	for _, char := range value {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_') {
			return false
		}
	}
	return true
}

func terminal(err error) error { return fmt.Errorf("%w: %v", asynq.SkipRetry, err) }
