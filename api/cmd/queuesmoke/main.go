// Command queuesmoke only runs synthetic handlers against an isolated local Redis.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"classgame-api/internal/platform/redisqueue"
	"classgame-api/internal/platform/requestmeta"
	"classgame-api/internal/taskqueue"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	raw := os.Getenv("QUEUE_SMOKE_REDIS_URL")
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "redis" || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" || parsed.Port() == "6379" || parsed.User != nil {
		return errors.New("smoke requires isolated 127.0.0.1 Redis on a non-default port")
	}
	queue := os.Getenv("QUEUE_SMOKE_NAME")
	if !strings.HasPrefix(queue, "smoke-") {
		return errors.New("smoke queue must start with smoke-")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	client, err := redisqueue.Connect(ctx, redisqueue.Options{URL: raw, Queue: queue})
	if err != nil {
		return err
	}
	defer client.Close()
	var mutex sync.Mutex
	counts := map[string]int{}
	processed := map[string]time.Time{}
	correlation := false
	handler := func(ctx context.Context, message taskqueue.Message) error {
		mutex.Lock()
		defer mutex.Unlock()
		counts[message.ID]++
		processed[message.ID] = time.Now()
		switch message.ID {
		case "retry":
			if counts[message.ID] == 1 {
				return errors.New("synthetic temporary failure")
			}
		case "terminal":
			return redisqueue.Terminal(errors.New("synthetic invalid work"))
		case "correlation":
			correlation = requestmeta.RequestID(ctx) == "synthetic-request-0001" && requestmeta.TraceID(ctx) == "0123456789abcdef0123456789abcdef"
		}
		return nil
	}
	worker, err := client.Worker(redisqueue.WorkerOptions{Concurrency: 2, Handlers: map[string]redisqueue.Handler{"synthetic:nop": handler}, RetryDelay: time.Second, CheckInterval: 50 * time.Millisecond, ShutdownTimeout: time.Second})
	if err != nil {
		return err
	}
	defer worker.Shutdown()
	destination := client.Destination("synthetic:nop")
	enqueue := func(id string, schedule time.Time, ctx context.Context) error {
		return destination.Enqueue(ctx, taskqueue.Message{ID: id, Payload: []byte(`{"synthetic":true}`), ScheduleAt: schedule})
	}
	if err := enqueue("nop", time.Time{}, ctx); err != nil {
		return err
	}
	if err := destination.Enqueue(ctx, taskqueue.Message{ID: "nop", Payload: []byte("{\n\"synthetic\": true}")}); err != nil {
		return fmt.Errorf("same-body replay: %w", err)
	}
	if err := destination.Enqueue(ctx, taskqueue.Message{ID: "nop", Payload: []byte(`{"synthetic":false}`)}); !errors.Is(err, redisqueue.ErrIDConflict) {
		return fmt.Errorf("changed-body replay must conflict: %v", err)
	}
	delayUntil := time.Now().Add(2 * time.Second)
	for _, id := range []string{"retry", "terminal", "delayed"} {
		scheduled := time.Time{}
		if id == "delayed" {
			scheduled = delayUntil
		}
		if err := enqueue(id, scheduled, ctx); err != nil {
			return err
		}
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://synthetic.invalid", nil)
	request.Header.Set("X-Request-ID", "synthetic-request-0001")
	request.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
	if err := enqueue("correlation", time.Time{}, requestmeta.FromRequest(request)); err != nil {
		return err
	}
	states := map[string]string{"nop": "completed", "retry": "completed", "terminal": "archived", "delayed": "completed", "correlation": "completed"}
	for id, want := range states {
		for {
			state, err := client.Status(id)
			if err == nil && state == want {
				break
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("job %s did not reach %s", id, want)
			case <-time.After(50 * time.Millisecond):
			}
		}
	}
	if err := enqueue("terminal", time.Time{}, ctx); !errors.Is(err, redisqueue.ErrArchived) {
		return fmt.Errorf("archived replay must require operator action: %v", err)
	}
	mutex.Lock()
	defer mutex.Unlock()
	if counts["nop"] != 1 || counts["retry"] != 2 || counts["terminal"] != 1 || !correlation || processed["delayed"].Before(delayUntil.Add(-time.Second)) {
		return fmt.Errorf("unexpected execution counts/correlation/delay: %v", counts)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "passed", "checks": []string{"completion", "canonical-replay", "changed-payload-conflict", "scheduled-delivery", "retry", "terminal-archive", "request-trace-correlation", "archived-replay-rejected"}, "handler_calls": counts, "real_provider_calls": 0, "database_calls": 0})
}
