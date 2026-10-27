package firebase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebaseadmin "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"classgame-api/internal/platform/firebase/emulator"
)

var (
	ErrMissingCredentials = errors.New("firebase credentials are required")
	ErrMissingProjectID   = errors.New("firebase project id is required")
)

const DefaultPingTimeout = 5 * time.Second

type Options struct {
	EmulatorEnabled                                      bool
	Environment, AuthEmulatorHost, FirestoreEmulatorHost string
	ProjectID                                            string
	CredentialsFile                                      string
	CredentialsJSON                                      string
	PingTimeout                                          time.Duration
}

type Connection struct {
	app             *firebaseadmin.App
	authClient      *auth.Client
	firestoreClient *firestore.Client
	projectID       string
	enabled         bool
}

func Connect(ctx context.Context, input Options) (*Connection, error) {
	resolved := resolveOptions(input)
	if strings.TrimSpace(resolved.ProjectID) == "" {
		return nil, ErrMissingProjectID
	}

	if err := (emulator.Options{
		Enabled:               resolved.EmulatorEnabled,
		Environment:           resolved.Environment,
		ProjectID:             resolved.ProjectID,
		AuthHost:              resolved.AuthEmulatorHost,
		FirestoreHost:         resolved.FirestoreEmulatorHost,
		CredentialsConfigured: resolved.CredentialsJSON != "" || resolved.CredentialsFile != "",
	}).Validate(); err != nil {
		return nil, err
	}
	if os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != resolved.AuthEmulatorHost || os.Getenv("FIRESTORE_EMULATOR_HOST") != resolved.FirestoreEmulatorHost {
		return nil, errors.New("Firebase emulator environment does not match validated options")
	}
	clientOption, err := buildClientOption(resolved)
	if err != nil {
		return nil, err
	}

	app, err := firebaseadmin.NewApp(ctx, &firebaseadmin.Config{
		ProjectID: resolved.ProjectID,
	}, clientOption)
	if err != nil {
		return nil, fmt.Errorf("create firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("create firebase auth client: %w", err)
	}

	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("create firestore client: %w", err)
	}

	connection := &Connection{
		app:             app,
		authClient:      authClient,
		firestoreClient: firestoreClient,
		projectID:       resolved.ProjectID,
		enabled:         true,
	}

	pingContext, cancel := context.WithTimeout(ctx, resolved.PingTimeout)
	defer cancel()

	if err := connection.Ping(pingContext); err != nil {
		_ = connection.Disconnect(context.Background())
		return nil, fmt.Errorf("ping firebase: %w", err)
	}

	return connection, nil
}

func (connection *Connection) IsEnabled() bool {
	return connection != nil && connection.enabled
}

func (connection *Connection) ProjectID() string {
	if connection == nil {
		return ""
	}

	return connection.projectID
}

func (connection *Connection) AuthClient() *auth.Client {
	if connection == nil {
		return nil
	}

	return connection.authClient
}

func (connection *Connection) FirestoreClient() *firestore.Client {
	if connection == nil {
		return nil
	}

	return connection.firestoreClient
}

func (connection *Connection) Ping(ctx context.Context) error {
	if !connection.IsEnabled() {
		return ErrMissingCredentials
	}

	iter := connection.firestoreClient.Collection("_health").Limit(1).Documents(ctx)
	defer iter.Stop()

	_, err := iter.Next()
	if err == nil || errors.Is(err, iterator.Done) {
		return nil
	}

	return err
}

func (connection *Connection) Disconnect(_ context.Context) error {
	if !connection.IsEnabled() {
		return nil
	}

	connection.enabled = false
	return connection.firestoreClient.Close()
}

func resolveOptions(input Options) Options {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.CredentialsFile = strings.TrimSpace(input.CredentialsFile)
	input.CredentialsJSON = strings.TrimSpace(input.CredentialsJSON)

	if input.PingTimeout <= 0 {
		input.PingTimeout = DefaultPingTimeout
	}

	return input
}

func buildClientOption(input Options) (option.ClientOption, error) {
	if input.EmulatorEnabled {
		return option.WithoutAuthentication(), nil
	}
	if input.CredentialsJSON != "" {
		return option.WithCredentialsJSON([]byte(input.CredentialsJSON)), nil
	}
	if input.CredentialsFile != "" {
		return option.WithCredentialsFile(input.CredentialsFile), nil
	}

	return nil, ErrMissingCredentials
}
