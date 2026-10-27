// learning-recovery lists one bounded recovery page. It never enqueues jobs,
// writes shared data, runs a model, bootstraps accounts, or changes queue state.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"classgame-api/internal/modules/students"
	"classgame-api/internal/platform/firebase"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	project := flag.String("project", os.Getenv("FIREBASE_PROJECT_ID"), "Firebase project to read")
	credentials := flag.String("credentials-file", os.Getenv("FIREBASE_CREDENTIALS_FILE"), "Local credentials file")
	limit := flag.Int("limit", 25, "Maximum documents scanned, 1..100")
	after := flag.String("after", "", "Document cursor from the previous page")
	flag.Parse()
	if *project == "" || *limit < 1 || *limit > 100 {
		log.Fatal("project and limit 1..100 are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connection, err := firebase.Connect(ctx, firebase.Options{ProjectID: *project, CredentialsFile: *credentials, CredentialsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON")})
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Disconnect(context.Background())
	page, err := students.NewFirestoreRepository(connection.FirestoreClient()).ListPendingFinalizations(ctx, *after, *limit, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	jobs := make([]students.LearningJob, 0, len(page.Attempts))
	for _, attempt := range page.Attempts {
		jobs = append(jobs, students.FinalizationJob(attempt))
	}
	if err := json.NewEncoder(os.Stdout).Encode(struct {
		Scanned    int                    `json:"scanned"`
		NextCursor string                 `json:"next_cursor,omitempty"`
		Jobs       []students.LearningJob `json:"jobs"`
	}{page.Scanned, page.NextCursor, jobs}); err != nil {
		log.Fatal(err)
	}
}
