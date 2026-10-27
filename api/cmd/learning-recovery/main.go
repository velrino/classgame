// Command learning-recovery lists a bounded page by default. Enqueue/redrive
// require explicit flags; no mode runs a model or a domain job processor.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"classgame-api/internal/modules/students"
	"classgame-api/internal/platform/firebase"
	"classgame-api/internal/platform/redisqueue"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	_ = godotenv.Load()
	flags := flag.NewFlagSet("learning-recovery", flag.ContinueOnError)
	project := flags.String("project", os.Getenv("FIREBASE_PROJECT_ID"), "Firebase project to read")
	credentials := flags.String("credentials-file", os.Getenv("FIREBASE_CREDENTIALS_FILE"), "Local credentials file")
	limit := flags.Int("limit", 25, "Maximum documents scanned, 1..100")
	after := flags.String("after", "", "Document cursor from the previous page")
	enqueue := flags.Bool("enqueue", false, "Explicitly enqueue this page's pending finalizations with stable IDs")
	redrive := flags.Bool("redrive", false, "Explicitly advance dispatch generations and enqueue this reviewed page")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || (*enqueue && *redrive) || *project == "" || *limit < 1 || *limit > 100 || len(*after) > 120 || strings.Contains(*after, "/") {
		return fmt.Errorf("requires project, limit 1..100, a bounded document cursor and at most one write action")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var queue *redisqueue.Client
	if *enqueue || *redrive {
		name := os.Getenv("TASKS_QUEUE")
		if name == "" {
			name = "classgame-background"
		}
		var err error
		queue, err = redisqueue.Connect(ctx, redisqueue.Options{URL: os.Getenv("TASKS_REDIS_URL"), Queue: name, Production: os.Getenv("APP_ENV") == "production"})
		if err != nil {
			return err
		}
		defer queue.Close()
	}
	connection, err := firebase.Connect(ctx, firebase.Options{ProjectID: *project, CredentialsFile: *credentials, CredentialsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON")})
	if err != nil {
		return err
	}
	defer connection.Disconnect(context.Background())
	repository := students.NewFirestoreRepository(connection.FirestoreClient())
	if queue != nil {
		service := students.NewService(students.Dependencies{Recovery: repository, Tasks: students.NewLearningTaskScheduler(queue.Destination(redisqueue.Learning), 2*time.Minute)})
		var report students.RecoveryReport
		if *redrive {
			report, err = service.RedrivePendingFinalizations(ctx, *after, *limit)
		} else {
			report, err = service.RecoverPendingFinalizations(ctx, *after, *limit)
		}
		if encodeErr := json.NewEncoder(os.Stdout).Encode(report); encodeErr != nil {
			return encodeErr
		}
		return err
	}
	page, err := repository.ListPendingFinalizations(ctx, *after, *limit, time.Now())
	if err != nil {
		return err
	}
	jobs := make([]students.LearningJob, 0, len(page.Attempts))
	for _, attempt := range page.Attempts {
		jobs = append(jobs, students.FinalizationJob(attempt))
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Scanned    int                    `json:"scanned"`
		NextCursor string                 `json:"next_cursor,omitempty"`
		Jobs       []students.LearningJob `json:"jobs"`
	}{page.Scanned, page.NextCursor, jobs})
}
