// Command queuesmoke only runs synthetic handlers against an isolated local Redis.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

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
	flags := flag.NewFlagSet("queuesmoke", flag.ContinueOnError)
	mode := flags.String("mode", "basic", "synthetic mode: basic, worker, enqueue, wait, outage")
	id := flags.String("id", "", "synthetic job ID")
	count := flags.Int("count", 1, "concurrent duplicate producers")
	waitFor := flags.String("wait-for", "completed", "started or completed")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *count < 1 || *count > 32 {
		return errors.New("invalid smoke arguments")
	}
	switch *mode {
	case "basic", "worker", "outage":
		if *id != "" {
			return errors.New("this mode does not accept a job ID")
		}
	case "enqueue", "wait":
		if *id != "distributed-once" && *id != "crash-recover" {
			return errors.New("unknown synthetic job ID")
		}
	default:
		return errors.New("unknown smoke mode")
	}
	if (*mode != "enqueue" && *count != 1) || (*mode != "wait" && *waitFor != "completed") {
		return errors.New("smoke flags do not apply to this mode")
	}
	if *waitFor != "started" && *waitFor != "completed" {
		return errors.New("unknown synthetic wait state")
	}
	raw, queue, err := smokeOptions()
	if err != nil {
		return err
	}
	switch *mode {
	case "worker":
		return runWorker(raw, queue)
	case "enqueue":
		return runEnqueue(raw, queue, *id, *count)
	case "wait":
		return runWait(raw, queue, *id, *waitFor)
	case "outage":
		return runOutage(raw, queue)
	default:
		return runBasic(raw, queue)
	}
}

func smokeOptions() (string, string, error) {
	raw := os.Getenv("QUEUE_SMOKE_REDIS_URL")
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "redis" || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" || parsed.Port() == "6379" || parsed.User != nil {
		return "", "", errors.New("smoke requires isolated 127.0.0.1 Redis on a non-default port")
	}
	queue := os.Getenv("QUEUE_SMOKE_NAME")
	if !strings.HasPrefix(queue, "smoke-") {
		return "", "", errors.New("smoke queue must start with smoke-")
	}
	return raw, queue, nil
}

func runBasic(raw, queue string) error {
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
	// Hold the existing immediate job briefly before starting workers. This is
	// one controlled pending-age sample, not a traffic percentile or an SLO.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}
	connection, err := asynq.ParseRedisURI(raw)
	if err != nil {
		return err
	}
	inspector := asynq.NewInspector(connection)
	defer inspector.Close()
	queueInfo, err := inspector.GetQueueInfo(queue)
	if err != nil || queueInfo.Pending != 1 {
		return fmt.Errorf("pending-age observation requires one pending job: %v", err)
	}
	queueAge := map[string]any{"oldest_pending_age_us": queueInfo.Latency.Microseconds(), "pending_jobs": queueInfo.Pending, "samples": 1, "worker_held_ms": 100, "source": "asynq.Inspector.GetQueueInfo.Latency", "scope": "controlled local sample; not a traffic percentile or an SLO"}
	worker, err := client.Worker(redisqueue.WorkerOptions{Concurrency: 2, Handlers: map[string]redisqueue.Handler{"synthetic:nop": handler}, RetryDelay: time.Second, CheckInterval: 50 * time.Millisecond, ShutdownTimeout: time.Second})
	if err != nil {
		return err
	}
	defer worker.Shutdown()
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
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "passed", "checks": []string{"completion", "canonical-replay", "changed-payload-conflict", "scheduled-delivery", "retry", "terminal-archive", "request-trace-correlation", "archived-replay-rejected"}, "handler_calls": counts, "queue_age": queueAge, "real_provider_calls": 0, "database_calls": 0})
}

// These helpers use Redis only as an observable synthetic effect counter.
func syntheticStore(raw string) (*redis.Client, error) {
	options, err := redis.ParseURL(raw)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(options), nil
}
func effectKey(queue, id, kind string) string { return "smokecheck:" + queue + ":" + id + ":" + kind }
func runWorker(raw, queue string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, err := redisqueue.Connect(ctx, redisqueue.Options{URL: raw, Queue: queue})
	if err != nil {
		return err
	}
	defer client.Close()
	store, err := syntheticStore(raw)
	if err != nil {
		return err
	}
	defer store.Close()
	worker, err := client.Worker(redisqueue.WorkerOptions{Concurrency: 1, RetryDelay: time.Second, CheckInterval: 50 * time.Millisecond, ShutdownTimeout: time.Second, Handlers: map[string]redisqueue.Handler{"synthetic:process": func(ctx context.Context, message taskqueue.Message) error {
		started, err := store.Incr(ctx, effectKey(queue, message.ID, "started")).Result()
		if err != nil {
			return err
		}
		if message.ID == "crash-recover" && started == 1 {
			<-ctx.Done()
			return ctx.Err()
		}
		return store.Incr(ctx, effectKey(queue, message.ID, "effect")).Err()
	}}})
	if err != nil {
		return err
	}
	defer worker.Shutdown()
	if err := json.NewEncoder(os.Stdout).Encode(map[string]bool{"ready": true}); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}
func runEnqueue(raw, queue, id string, count int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := redisqueue.Connect(ctx, redisqueue.Options{URL: raw, Queue: queue})
	if err != nil {
		return err
	}
	defer client.Close()
	destination := client.Destination("synthetic:process")
	failures := make(chan error, count)
	var workers sync.WaitGroup
	for range count {
		workers.Go(func() {
			failures <- destination.Enqueue(ctx, taskqueue.Message{ID: id, Payload: []byte(`{"synthetic":true}`)})
		})
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			return err
		}
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"accepted": count, "id": id})
}
func runWait(raw, queue, id, want string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 160*time.Second)
	defer cancel()
	client, err := redisqueue.Connect(ctx, redisqueue.Options{URL: raw, Queue: queue})
	if err != nil {
		return err
	}
	defer client.Close()
	store, err := syntheticStore(raw)
	if err != nil {
		return err
	}
	defer store.Close()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		starts, _ := store.Get(ctx, effectKey(queue, id, "started")).Int()
		effects, _ := store.Get(ctx, effectKey(queue, id, "effect")).Int()
		state, err := client.Status(id)
		if err == nil && ((want == "started" && starts >= 1 && state == "active") || (want == "completed" && state == "completed")) {
			return json.NewEncoder(os.Stdout).Encode(map[string]any{"id": id, "state": state, "starts": starts, "effects": effects})
		}
		select {
		case <-ctx.Done():
			return errors.New("synthetic job did not reach expected state")
		case <-ticker.C:
		}
	}
}
func runOutage(raw, queue string) error {
	client, err := redisqueue.Connect(context.Background(), redisqueue.Options{URL: raw, Queue: queue})
	if err != nil {
		return err
	}
	defer client.Close()
	if err := json.NewEncoder(os.Stdout).Encode(map[string]bool{"ready": true}); err != nil {
		return err
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || line != "broker-down\n" {
		return errors.New("missing outage coordination")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	err = client.Destination("synthetic:process").Enqueue(ctx, taskqueue.Message{ID: "outage", Payload: []byte(`{"synthetic":true}`)})
	elapsed := time.Since(started)
	if err == nil || elapsed > 6*time.Second {
		return fmt.Errorf("broker outage must fail within six seconds; failed=%t elapsed=%v", err != nil, elapsed)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"failed": true, "elapsed_ms": elapsed.Milliseconds(), "limit_ms": 6000})
}
