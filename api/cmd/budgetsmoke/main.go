// Command budgetsmoke checks production shared admission on an owned local Redis.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"classgame-api/internal/platform/genai"
	"classgame-api/internal/platform/genai/redisbudget"
	"github.com/redis/go-redis/v9"
)

type result struct {
	ID     string `json:"id"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	flags := flag.NewFlagSet("budgetsmoke", flag.ContinueOnError)
	mode := flags.String("mode", "basic", "basic, instance, or outage")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || (*mode != "basic" && *mode != "instance" && *mode != "outage") {
		return errors.New("invalid budget smoke arguments")
	}
	raw := os.Getenv("BUDGET_SMOKE_REDIS_URL")
	u, err := url.Parse(raw)
	scope := os.Getenv("BUDGET_SMOKE_SCOPE")
	if err != nil || u.Scheme != "redis" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.Port() == "6379" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || !strings.HasPrefix(scope, "smoke-") || len(scope) > 80 {
		return errors.New("budget smoke needs owned loopback Redis on a non-default port and smoke- scope")
	}
	options, err := redis.ParseURL(raw)
	if err != nil {
		return errors.New("invalid synthetic Redis URL")
	}
	options.ContextTimeoutEnabled = true
	options.MaxRetries = -1
	options.DialTimeout = 500 * time.Millisecond
	options.ReadTimeout = 500 * time.Millisecond
	options.WriteTimeout = 500 * time.Millisecond
	client := redis.NewClient(options)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return errors.New("synthetic Redis unavailable")
	}
	pid, pidErr := strconv.Atoi(os.Getenv("BUDGET_SMOKE_REDIS_PID"))
	info, infoErr := client.Info(ctx, "server").Result()
	if pidErr != nil || pid < 1 || infoErr != nil || !strings.Contains(info, fmt.Sprintf("\r\nprocess_id:%d\r\n", pid)) {
		return errors.New("synthetic Redis PID does not match the owned process")
	}
	if *mode == "instance" {
		return instance(ctx, client, scope)
	}
	if *mode == "outage" {
		return outage(ctx, client, scope)
	}
	return basic(ctx, client, scope)
}
func settings(scope string) redisbudget.Options {
	return redisbudget.Options{Scope: scope, Concurrent: 2, RequestsPerWindow: 100, OutputTokensPerWindow: 1000000, Window: time.Minute, Lease: 5 * time.Second}
}
func create(client *redis.Client, options redisbudget.Options) *redisbudget.Budget {
	value, err := redisbudget.New(client, options)
	if err != nil {
		panic(err)
	}
	return value
}
func check(ok bool) error {
	if !ok {
		return errors.New("synthetic assertion failed")
	}
	return nil
}
func basic(ctx context.Context, client *redis.Client, scope string) error {
	serverTime, err := client.Time(ctx).Result()
	if err != nil {
		return err
	}
	remaining := time.Minute - time.Duration(serverTime.UnixNano()%int64(time.Minute))
	if remaining < 2*time.Second {
		time.Sleep(remaining + time.Millisecond)
	}
	results := []result{}
	run := func(id string, work func() error) {
		r := result{ID: id, Passed: true}
		if err := work(); err != nil {
			r.Passed = false
			r.Detail = err.Error()
		}
		results = append(results, r)
	}
	run("shared-concurrency-and-safe-release", func() error {
		o := settings(scope + "-capacity")
		a, b := create(client, o), create(client, o)
		defer a.Close()
		defer b.Close()
		first, e := a.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		second, e := b.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		_, e = a.Acquire(ctx, 512)
		if !errors.Is(e, genai.ErrSharedBudget) {
			return errors.New("capacity was not shared")
		}
		if e = first.Release(ctx); e != nil {
			return e
		}
		third, e := b.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		if e = first.Release(ctx); e != nil {
			return e
		}
		s, e := a.Snapshot(ctx)
		if e != nil {
			return e
		}
		if s.Active != 2 {
			return errors.New("old release removed a current owner")
		}
		second.Release(ctx)
		third.Release(ctx)
		return nil
	})
	run("request-window-is-not-refunded", func() error {
		o := settings(scope + "-requests")
		o.RequestsPerWindow = 1
		a, b := create(client, o), create(client, o)
		defer a.Close()
		defer b.Close()
		l, e := a.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		l.Release(ctx)
		_, e = b.Acquire(ctx, 512)
		return check(errors.Is(e, genai.ErrSharedBudget))
	})
	run("output-reservation-is-shared", func() error {
		o := settings(scope + "-tokens")
		o.OutputTokensPerWindow = 700
		a, b := create(client, o), create(client, o)
		defer a.Close()
		defer b.Close()
		l, e := a.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		l.Release(ctx)
		_, e = b.Acquire(ctx, 512)
		s, se := a.Snapshot(ctx)
		if se != nil {
			return se
		}
		return check(errors.Is(e, genai.ErrSharedBudget) && s.Requests == 1 && s.ReservedOutputTokens == 512)
	})
	run("expired-owner-cannot-release-successor", func() error {
		o := settings(scope + "-expiry")
		o.Concurrent = 1
		o.Lease = 200 * time.Millisecond
		a, b := create(client, o), create(client, o)
		defer a.Close()
		defer b.Close()
		old, e := a.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		time.Sleep(250 * time.Millisecond)
		current, e := b.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		old.Release(ctx)
		_, e = a.Acquire(ctx, 512)
		current.Release(ctx)
		return check(errors.Is(e, genai.ErrSharedBudget))
	})
	run("mixed-instance-settings-fail-closed", func() error {
		o := settings(scope + "-policy")
		a := create(client, o)
		defer a.Close()
		l, e := a.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		defer l.Release(ctx)
		o.Concurrent++
		b := create(client, o)
		defer b.Close()
		_, e = b.Acquire(ctx, 512)
		return check(errors.Is(e, genai.ErrBudgetPolicy))
	})
	run("provider-error-releases-capacity-keeps-usage", func() error {
		a := create(client, settings(scope+"-failure"))
		defer a.Close()
		want := errors.New("synthetic provider failure")
		e := genai.RunAdmitted(ctx, a, 512, func() error { return want })
		s, se := a.Snapshot(ctx)
		if se != nil {
			return se
		}
		return check(errors.Is(e, want) && s.Active == 0 && s.Requests == 1 && s.ReservedOutputTokens == 512)
	})
	run("canceled-provider-releases-capacity", func() error {
		a := create(client, settings(scope+"-cancel"))
		defer a.Close()
		cancelCtx, cancel := context.WithCancel(ctx)
		e := genai.RunAdmitted(cancelCtx, a, 512, func() error { cancel(); return cancelCtx.Err() })
		s, se := a.Snapshot(ctx)
		if se != nil {
			return se
		}
		return check(errors.Is(e, context.Canceled) && s.Active == 0 && s.Requests == 1)
	})
	run("nil-budget-never-calls-provider", func() error {
		called := false
		e := genai.RunAdmitted(ctx, nil, 512, func() error { called = true; return nil })
		return check(errors.Is(e, genai.ErrBudgetUnavailable) && !called)
	})
	run("window-recovers-after-expiry", func() error {
		o := settings(scope + "-window")
		o.RequestsPerWindow = 1
		o.Window = 200 * time.Millisecond
		a := create(client, o)
		defer a.Close()
		first, e := a.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		first.Release(ctx)
		time.Sleep(250 * time.Millisecond)
		second, e := a.Acquire(ctx, 512)
		if e != nil {
			return e
		}
		second.Release(ctx)
		return nil
	})
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"mode": "local-redis", "real_provider_calls": 0, "database_calls": 0, "passed": passed, "total": len(results), "results": results})
	if passed != len(results) {
		return errors.New("shared budget smoke failed")
	}
	return nil
}

var observe = redis.NewScript(`local active=redis.call('INCR',KEYS[1]);local peak=tonumber(redis.call('GET',KEYS[2])) or 0;if active>peak then redis.call('SET',KEYS[2],active) end;return active`)

func instance(ctx context.Context, client *redis.Client, scope string) error {
	budget := create(client, settings(scope+"-process"))
	defer budget.Close()
	fmt.Println(`{"ready":true}`)
	if _, e := bufio.NewReader(os.Stdin).ReadString('\n'); e != nil {
		return e
	}
	var wg sync.WaitGroup
	var accepted, denied, failures atomic.Int64
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := genai.RunAdmitted(ctx, budget, 512, func() error {
				if e := observe.Run(ctx, client, []string{scope + ":active", scope + ":peak"}).Err(); e != nil {
					return e
				}
				time.Sleep(200 * time.Millisecond)
				return client.Decr(ctx, scope+":active").Err()
			})
			if e == nil {
				accepted.Add(1)
			} else if errors.Is(e, genai.ErrSharedBudget) {
				denied.Add(1)
			} else {
				failures.Add(1)
			}
		}()
	}
	wg.Wait()
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"accepted": accepted.Load(), "denied": denied.Load(), "failures": failures.Load()})
}
func outage(ctx context.Context, client *redis.Client, scope string) error {
	budget := create(client, settings(scope+"-outage"))
	defer budget.Close()
	l, e := budget.Acquire(ctx, 512)
	if e != nil {
		return e
	}
	if e = l.Release(ctx); e != nil {
		return e
	}
	fmt.Println(`{"ready":true}`)
	if _, e := bufio.NewReader(os.Stdin).ReadString('\n'); e != nil {
		return e
	}
	start := time.Now()
	called := false
	e = genai.RunAdmitted(ctx, budget, 512, func() error { called = true; return nil })
	elapsed := time.Since(start)
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"unavailable": errors.Is(e, genai.ErrBudgetUnavailable), "provider_called": called, "elapsed_ms": elapsed.Milliseconds()})
}
