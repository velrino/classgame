// Command evalfirestore checks production transactions with synthetic emulator fixtures.
// It never loads application configuration, credentials, or a model provider.
package main

import (
	"classgame-api/internal/modules/students"
	"cloud.google.com/go/firestore"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"
)

type result struct {
	ID     string `json:"id"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}
type report struct {
	Mode              string   `json:"mode"`
	Project           string   `json:"project"`
	FixturePrefix     string   `json:"fixture_prefix"`
	RealProviderCalls int      `json:"real_provider_calls"`
	Passed            int      `json:"passed"`
	Total             int      `json:"total"`
	CleanupComplete   bool     `json:"cleanup_complete"`
	Results           []result `json:"results"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	optIn := flag.Bool("run", false, "explicitly allow synthetic emulator writes")
	project := flag.String("project", "demo-classgame", "demo- emulator project")
	flag.Parse()
	if !*optIn || flag.NArg() != 0 {
		return errors.New("require -run and no positional arguments; emulator writes are opt-in")
	}
	endpoint := os.Getenv("FIRESTORE_EMULATOR_HOST")
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return errors.New("FIRESTORE_EMULATOR_HOST must be a numeric loopback host:port")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 || !regexp.MustCompile(`^demo-[a-z0-9][a-z0-9-]{0,50}$`).MatchString(*project) {
		return errors.New("require a valid port and a demo- project ID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	client, err := firestore.NewClient(ctx, *project, option.WithEndpoint(endpoint), option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		return fmt.Errorf("create emulator client: %w", err)
	}
	defer client.Close()
	entropy := make([]byte, 10)
	if _, err := rand.Read(entropy); err != nil {
		return err
	}
	s := &suite{ctx: ctx, client: client, repo: students.NewFirestoreRepository(client), prefix: "evalfs-" + hex.EncodeToString(entropy)}
	out := report{Mode: "firestore-emulator", Project: *project, FixturePrefix: s.prefix}
	fmt.Fprintln(os.Stderr, "synthetic fixture prefix:", s.prefix)
	cases := []struct {
		id  string
		run func() error
	}{
		{"same-event-concurrent-reservation-v1", s.concurrentReservation},
		{"durable-evidence-and-replay-v1", s.durableEvidence},
		{"concurrent-attempt-budget-v1", s.concurrentBudget},
		{"expired-reservation-owner-fence-v1", s.reservationFence},
		{"concurrent-finalization-claim-v1", s.concurrentFinalization},
		{"stale-finalizer-after-new-owner-v1", func() error { return s.staleFinalizer("owner") }},
		{"stale-finalizer-after-evidence-v1", func() error { return s.staleFinalizer("evidence") }},
		{"stale-finalizer-after-resume-v1", func() error { return s.staleFinalizer("resume") }},
		{"late-finalization-failure-after-ready-v1", s.finalizationReady},
		{"bounded-recovery-page-v1", s.recoveryPage},
	}
	for _, c := range cases {
		r := result{ID: c.id, Passed: true}
		if err := c.run(); err != nil {
			r.Passed = false
			r.Detail = err.Error()
		} else {
			out.Passed++
		}
		out.Results = append(out.Results, r)
		fmt.Fprintf(os.Stderr, "%s: %t\n", r.ID, r.Passed)
	}
	out.Total = len(out.Results)
	if err := s.cleanup(); err != nil {
		out.Results = append(out.Results, result{ID: "fixture-cleanup-v1", Passed: false, Detail: err.Error()})
	} else {
		out.CleanupComplete = true
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		return err
	}
	if out.Passed != out.Total || !out.CleanupComplete {
		return errors.New("Firestore emulator evaluation failed")
	}
	return nil
}
