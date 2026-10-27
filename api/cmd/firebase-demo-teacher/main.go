// Command firebase-demo-teacher manually creates or re-keys the shared teacher
// with a password supplied out of band. Normal API startup creates the account
// when missing; use this command to rotate credentials or repair drift:
//
//	go run ./cmd/firebase-demo-teacher --project allthingsagentichackatho-ee359 \
//	  --credentials-file /path/to/service-account.json --password '...'
//
// It creates the user if missing, otherwise sets the password on the existing
// account (keeping any Google sign-in already linked to it).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	firebaseadmin "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	"classgame-api/internal/demoaccount"
	"classgame-api/internal/platform/firebase"
)

const (
	expectedProjectID = "allthingsagentichackatho-ee359"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var credentialsFile string
	var displayName string
	var email string
	var password string
	var projectID string

	flag.StringVar(&projectID, "project", "", "Firebase project ID")
	flag.StringVar(&credentialsFile, "credentials-file", "", "Firebase service account JSON path (default: Application Default Credentials)")
	flag.StringVar(&email, "email", demoaccount.DefaultEmail, "Demo teacher email")
	flag.StringVar(&password, "password", os.Getenv("DEMO_TEACHER_PASSWORD"), "Password to set (or DEMO_TEACHER_PASSWORD)")
	flag.StringVar(&displayName, "display-name", demoaccount.DefaultDisplayName, "Display name")
	flag.Parse()

	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return errors.New("missing --project")
	}
	if projectID != expectedProjectID && os.Getenv("ALLOW_OTHER_FIREBASE_PROJECT") != "1" {
		return fmt.Errorf("refusing to touch project %q; expected %q", projectID, expectedProjectID)
	}
	if len(password) < 6 {
		return errors.New("--password must be at least 6 characters (Firebase minimum)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	opts := []option.ClientOption{}
	if strings.TrimSpace(credentialsFile) != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}
	app, err := firebaseadmin.NewApp(ctx, &firebaseadmin.Config{ProjectID: projectID}, opts...)
	if err != nil {
		return fmt.Errorf("create Firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return fmt.Errorf("create Firebase Auth client: %w", err)
	}

	result, err := firebase.EnsureEmailPasswordUser(ctx, client, firebase.EmailPasswordUserInput{
		Email:          email,
		Password:       password,
		DisplayName:    displayName,
		EmailVerified:  true,
		UpdateExisting: true,
	})
	if err != nil {
		return fmt.Errorf("ensure demo teacher: %w", err)
	}
	if result.Created {
		fmt.Printf("Created %s (uid %s)\n", result.User.Email, result.User.UID)
	} else {
		fmt.Printf("Updated %s (uid %s); providers: %s\n", result.User.Email, result.User.UID, providerList(result.User))
	}

	return nil
}

func providerList(user *auth.UserRecord) string {
	ids := make([]string, 0, len(user.ProviderUserInfo))
	for _, info := range user.ProviderUserInfo {
		if info != nil {
			ids = append(ids, info.ProviderID)
		}
	}
	return strings.Join(ids, ",")
}
