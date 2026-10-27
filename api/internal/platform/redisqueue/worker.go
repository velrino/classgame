package redisqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/hibiken/asynq"

	"classgame-api/internal/platform/requestmeta"
	"classgame-api/internal/taskqueue"
)

type Handler func(context.Context, taskqueue.Message) error

type WorkerOptions struct {
	Concurrency int
	Handlers    map[string]Handler
	// Short intervals are useful for isolated synthetic smoke runs.
	RetryDelay      time.Duration
	CheckInterval   time.Duration
	ShutdownTimeout time.Duration
}

type Worker struct{ server *asynq.Server }

func (client *Client) Worker(options WorkerOptions) (*Worker, error) {
	if options.Concurrency < 1 || options.Concurrency > 32 || len(options.Handlers) == 0 {
		return nil, errors.New("worker requires handlers and concurrency between 1 and 32")
	}
	configuration := asynq.Config{Concurrency: options.Concurrency, Queues: map[string]int{client.queue: 1}, ShutdownTimeout: 30 * time.Second,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			id, _ := asynq.GetTaskID(ctx)
			var stored envelope
			if json.Unmarshal(task.Payload(), &stored) == nil {
				ctx = messageContext(ctx, stored.Metadata)
			}
			retry, _ := asynq.GetRetryCount(ctx)
			slog.WarnContext(ctx, "background job failed", "job_id", id, "job_type", task.Type(), "retry", retry, "request_id", requestmeta.RequestID(ctx), "trace_id", requestmeta.TraceID(ctx))
		})}
	if options.RetryDelay > 0 {
		configuration.RetryDelayFunc = func(int, error, *asynq.Task) time.Duration { return options.RetryDelay }
	}
	if options.CheckInterval > 0 {
		configuration.DelayedTaskCheckInterval = options.CheckInterval
		configuration.TaskCheckInterval = options.CheckInterval
	}
	if options.ShutdownTimeout > 0 {
		configuration.ShutdownTimeout = options.ShutdownTimeout
	}
	mux := asynq.NewServeMux()
	for kind, handler := range options.Handlers {
		if kind == "" || handler == nil {
			return nil, errors.New("worker handler is invalid")
		}
		mux.HandleFunc(kind, func(ctx context.Context, task *asynq.Task) error {
			var stored envelope
			decoder := json.NewDecoder(bytes.NewReader(task.Payload()))
			decoder.DisallowUnknownFields()
			actualID, _ := asynq.GetTaskID(ctx)
			if decoder.Decode(&stored) != nil || stored.Version != 1 || !safeName(stored.ID, 500) || stored.ID != actualID || len(stored.Payload) > maxPayload || !errors.Is(decoder.Decode(new(any)), io.EOF) {
				return terminal(errors.New("invalid background envelope"))
			}
			ctx = requestmeta.WithGenerationScope(messageContext(ctx, stored.Metadata), stored.ID)
			slog.InfoContext(ctx, "process background job", "job_id", stored.ID, "job_type", kind, "request_id", requestmeta.RequestID(ctx), "trace_id", requestmeta.TraceID(ctx))
			return handler(ctx, taskqueue.Message{ID: stored.ID, Payload: stored.Payload, Metadata: stored.Metadata})
		})
	}
	server := asynq.NewServer(client.redis, configuration)
	if err := server.Start(mux); err != nil {
		return nil, err
	}
	return &Worker{server: server}, nil
}

func (worker *Worker) Shutdown() { worker.server.Shutdown() }

// Terminal archives work that cannot succeed on retry. It does not delete the job.
func Terminal(err error) error { return terminal(err) }

func messageContext(ctx context.Context, metadata map[string]string) context.Context {
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://worker.invalid", nil)
	for _, key := range []string{requestmeta.RequestIDHeader, "traceparent"} {
		if value := metadata[key]; len(value) <= 128 {
			request.Header.Set(key, value)
		}
	}
	return requestmeta.FromRequest(request)
}
