// Command rebuild-class-evidence reads one bounded class page by default.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"classgame-api/internal/modules/students"
	"classgame-api/internal/platform/firebase"
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
	flags := flag.NewFlagSet("rebuild-class-evidence", flag.ContinueOnError)
	project := flags.String("project", os.Getenv("FIREBASE_PROJECT_ID"), "project to read")
	credentials := flags.String("credentials-file", os.Getenv("FIREBASE_CREDENTIALS_FILE"), "local credentials file")
	classID := flags.String("class", "", "one class ID to inspect")
	after := flags.String("after", "", "document cursor from the previous page")
	limit := flags.Int("limit", 25, "maximum source rows, 1..100")
	apply := flags.Bool("apply", false, "explicitly write this page's private evidence projection")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *project == "" || *classID == "" || *limit < 1 || *limit > 100 {
		return errors.New("project,class and limit 1..100 are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connection, err := firebase.Connect(ctx, firebase.Options{ProjectID: *project, CredentialsFile: *credentials, CredentialsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON")})
	if err != nil {
		return err
	}
	defer connection.Disconnect(context.Background())
	result, err := students.NewFirestoreRepository(connection.FirestoreClient()).RebuildClassEvidencePage(ctx, *classID, *after, *limit, *apply)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
