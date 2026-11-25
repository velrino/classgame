package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebaseadmin "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"classgame-api/internal/shared/demoaccount"
)

const (
	authDeleteBatchSize = 1000
	expectedProjectID   = "allthingsagentichackatho-ee359"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var credentialsFile string
	var dryRun bool
	var projectID string
	var skipAuth bool
	var skipFirestore bool
	var yes bool
	var keepEmails string

	flag.StringVar(&projectID, "project", "", "Firebase project ID to clean")
	flag.StringVar(&keepEmails, "keep-emails", demoaccount.DefaultEmail, "Comma-separated emails of Firebase Auth users to keep")
	flag.StringVar(&credentialsFile, "credentials-file", "", "Firebase service account JSON path")
	flag.BoolVar(&dryRun, "dry-run", false, "List what would be deleted")
	flag.BoolVar(&skipAuth, "skip-auth", false, "Do not delete Firebase Authentication users")
	flag.BoolVar(&skipFirestore, "skip-firestore", false, "Do not delete Firestore documents")
	flag.BoolVar(&yes, "yes", false, "Required to delete data")
	flag.Parse()

	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return errors.New("missing --project")
	}
	if projectID != expectedProjectID && os.Getenv("ALLOW_OTHER_FIREBASE_PROJECT") != "1" {
		return fmt.Errorf("refusing to clean project %q; expected %q", projectID, expectedProjectID)
	}
	if !dryRun && !yes {
		return errors.New("refusing to delete Firebase data without --yes")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	app, err := newFirebaseApp(ctx, projectID, credentialsFile)
	if err != nil {
		return err
	}

	if !skipFirestore {
		firestoreClient, err := app.Firestore(ctx)
		if err != nil {
			return fmt.Errorf("create Firestore client: %w", err)
		}
		defer firestoreClient.Close()

		deleted, err := resetFirestore(ctx, firestoreClient, dryRun)
		if err != nil {
			return err
		}
		if dryRun {
			fmt.Printf("Firestore documents found: %d\n", deleted)
		} else {
			fmt.Printf("Firestore documents deleted: %d\n", deleted)
		}
	}

	if !skipAuth {
		authClient, err := app.Auth(ctx)
		if err != nil {
			return fmt.Errorf("create Firebase Auth client: %w", err)
		}

		keep := parseEmailSet(keepEmails)
		deleted, kept, err := resetAuth(ctx, authClient, keep, dryRun)
		if err != nil {
			return err
		}
		if dryRun {
			fmt.Printf("Firebase Auth users found: %d (kept: %d)\n", deleted, kept)
		} else {
			fmt.Printf("Firebase Auth users deleted: %d (kept: %d)\n", deleted, kept)
		}
	}

	return nil
}

func newFirebaseApp(ctx context.Context, projectID string, credentialsFile string) (*firebaseadmin.App, error) {
	options := []option.ClientOption{}
	credentialsFile = strings.TrimSpace(credentialsFile)
	if credentialsFile != "" {
		options = append(options, option.WithCredentialsFile(credentialsFile))
	}

	app, err := firebaseadmin.NewApp(ctx, &firebaseadmin.Config{
		ProjectID: projectID,
	}, options...)
	if err != nil {
		return nil, fmt.Errorf("create firebase app: %w", err)
	}

	return app, nil
}

func resetFirestore(ctx context.Context, client *firestore.Client, dryRun bool) (int, error) {
	collections := client.Collections(ctx)
	total := 0

	for {
		collection, err := collections.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return total, fmt.Errorf("list Firestore collections: %w", err)
		}

		deleted, err := resetCollection(ctx, client, collection, dryRun)
		total += deleted
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

func resetCollection(
	ctx context.Context,
	client *firestore.Client,
	collection *firestore.CollectionRef,
	dryRun bool,
) (int, error) {
	documents := collection.DocumentRefs(ctx)
	total := 0
	var writer *firestore.BulkWriter
	if !dryRun {
		writer = client.BulkWriter(ctx)
		defer func() {
			writer.End()
			writer.Flush()
		}()
	}

	for {
		document, err := documents.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return total, fmt.Errorf("list Firestore documents in %s: %w", collection.ID, err)
		}

		deleted, err := resetDocument(ctx, client, document, writer, dryRun)
		total += deleted
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

func resetDocument(
	ctx context.Context,
	client *firestore.Client,
	document *firestore.DocumentRef,
	writer *firestore.BulkWriter,
	dryRun bool,
) (int, error) {
	total := 0
	subcollections := document.Collections(ctx)

	for {
		subcollection, err := subcollections.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return total, fmt.Errorf("list Firestore subcollections for %s: %w", document.ID, err)
		}

		deleted, err := resetCollection(ctx, client, subcollection, dryRun)
		total += deleted
		if err != nil {
			return total, err
		}
	}

	total++
	if dryRun {
		return total, nil
	}

	if _, err := writer.Delete(document); err != nil {
		return total, fmt.Errorf("delete Firestore document %s: %w", document.ID, err)
	}

	return total, nil
}

// parseEmailSet lower-cases and trims a comma-separated list, dropping blanks.
func parseEmailSet(list string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, email := range strings.Split(list, ",") {
		email = strings.ToLower(strings.TrimSpace(email))
		if email != "" {
			set[email] = struct{}{}
		}
	}
	return set
}

// resetAuth deletes every user except those whose email is in keep. Returns (deleted or found, kept).
func resetAuth(ctx context.Context, client authUserManager, keep map[string]struct{}, dryRun bool) (int, int, error) {
	uids, kept, err := listAuthUserIDs(ctx, client, keep)
	if err != nil {
		return 0, kept, err
	}

	if dryRun || len(uids) == 0 {
		return len(uids), kept, nil
	}

	var deleted int
	var failed int
	for start := 0; start < len(uids); start += authDeleteBatchSize {
		end := min(start+authDeleteBatchSize, len(uids))
		result, err := client.DeleteUsers(ctx, uids[start:end])
		if err != nil {
			return deleted, kept, fmt.Errorf("delete Firebase Auth users batch %d-%d: %w", start, end-1, err)
		}

		deleted += result.SuccessCount
		failed += result.FailureCount
	}

	if failed > 0 {
		return deleted, kept, fmt.Errorf("failed to delete %d Firebase Auth users", failed)
	}

	return deleted, kept, nil
}

type authUserManager interface {
	DeleteUsers(context.Context, []string) (*auth.DeleteUsersResult, error)
	Users(context.Context, string) *auth.UserIterator
}

// listAuthUserIDs returns the UIDs to delete and how many users were kept because their email is in keep.
func listAuthUserIDs(ctx context.Context, client authUserManager, keep map[string]struct{}) ([]string, int, error) {
	userIterator := client.Users(ctx, "")
	uids := []string{}
	kept := 0

	for {
		user, err := userIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, kept, fmt.Errorf("list Firebase Auth users: %w", err)
		}

		if _, keepUser := keep[strings.ToLower(user.Email)]; keepUser {
			kept++
			continue
		}

		uids = append(uids, user.UID)
	}

	return uids, kept, nil
}
