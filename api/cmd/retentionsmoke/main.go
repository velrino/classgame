// Command retentionsmoke uses owned synthetic records in a local demo emulator.
package main

import (
	"classgame-api/internal/datarecovery"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type result struct {
	ID     string `json:"id"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	optIn := flag.Bool("run", false, "allow owned synthetic emulator records")
	project := flag.String("project", "demo-classgame-retention", "demo project only")
	flag.Parse()
	if os.Getenv("RETENTION_BACKUP_COMMAND") == "" {
		return errors.New("use the Python retention runner so the reviewed backup command is built first")
	}
	if !*optIn || flag.NArg() != 0 || !strings.HasPrefix(*project, "demo-") || os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		return errors.New("require -run, a demo project and loopback emulator")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	store, e := datarecovery.Open(ctx, *project)
	if e != nil {
		return e
	}
	defer store.Close()
	random := make([]byte, 10)
	if _, e = rand.Read(random); e != nil {
		return e
	}
	s := &suite{ctx: ctx, store: store, prefix: "evalret-" + hex.EncodeToString(random)}
	if e = s.seed(); e != nil {
		s.cleanup()
		return e
	}
	temp, e := os.MkdirTemp("", "classgame-private-retention-")
	if e != nil {
		s.cleanup()
		return e
	}
	defer os.RemoveAll(temp)
	key := make([]byte, 32)
	if _, e = rand.Read(key); e != nil {
		s.cleanup()
		return e
	}
	keyPath := filepath.Join(temp, "key")
	if e = os.WriteFile(keyPath, key, 0600); e != nil {
		s.cleanup()
		return e
	}
	artifact := filepath.Join(temp, "class.enc")
	results := s.cases(artifact, keyPath, temp)
	cleanup := s.cleanup()
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{"mode": "firestore-emulator-retention", "project": *project, "fixture_prefix": s.prefix, "passed": passed, "total": len(results), "cleanup_complete": cleanup == nil, "real_provider_calls": 0, "results": results})
	if passed != len(results) || cleanup != nil {
		return fmt.Errorf("retention evaluation failed; cleanup: %v", cleanup)
	}
	return nil
}
