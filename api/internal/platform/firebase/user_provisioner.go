package firebase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"
)

type EmailPasswordUserInput struct {
	Email          string
	Password       string
	DisplayName    string
	EmailVerified  bool
	UpdateExisting bool
}

type EmailPasswordUserResult struct {
	User    *firebaseauth.UserRecord
	Created bool
	Updated bool
}

type EmailPasswordUserClient interface {
	CreateUser(context.Context, *firebaseauth.UserToCreate) (*firebaseauth.UserRecord, error)
	GetUserByEmail(context.Context, string) (*firebaseauth.UserRecord, error)
	UpdateUser(context.Context, string, *firebaseauth.UserToUpdate) (*firebaseauth.UserRecord, error)
}

// EnsureEmailPasswordUser creates a Firebase Auth user when the email is
// missing. Callers may opt into resetting the existing account fields; normal
// API startup leaves an existing account untouched.
func EnsureEmailPasswordUser(
	ctx context.Context,
	client EmailPasswordUserClient,
	input EmailPasswordUserInput,
) (EmailPasswordUserResult, error) {
	if client == nil {
		return EmailPasswordUserResult{}, errors.New("firebase auth client is not configured")
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Email == "" {
		return EmailPasswordUserResult{}, errors.New("email is required")
	}
	if len(input.Password) < 6 {
		return EmailPasswordUserResult{}, errors.New("password must be at least 6 characters")
	}

	existing, err := client.GetUserByEmail(ctx, input.Email)
	switch {
	case err == nil:
		return ensureExistingEmailPasswordUser(ctx, client, existing, input)
	case !firebaseauth.IsUserNotFound(err):
		return EmailPasswordUserResult{}, fmt.Errorf("look up Firebase Auth user: %w", err)
	}

	params := (&firebaseauth.UserToCreate{}).
		Email(input.Email).
		Password(input.Password).
		DisplayName(input.DisplayName).
		EmailVerified(input.EmailVerified)
	user, err := client.CreateUser(ctx, params)
	if err == nil {
		return EmailPasswordUserResult{User: user, Created: true}, nil
	}
	if !firebaseauth.IsEmailAlreadyExists(err) {
		return EmailPasswordUserResult{}, fmt.Errorf("create Firebase Auth user: %w", err)
	}

	// Another API instance may have created the same account after our lookup.
	existing, err = client.GetUserByEmail(ctx, input.Email)
	if err != nil {
		return EmailPasswordUserResult{}, fmt.Errorf("read concurrently created Firebase Auth user: %w", err)
	}

	return ensureExistingEmailPasswordUser(ctx, client, existing, input)
}

func ensureExistingEmailPasswordUser(
	ctx context.Context,
	client EmailPasswordUserClient,
	existing *firebaseauth.UserRecord,
	input EmailPasswordUserInput,
) (EmailPasswordUserResult, error) {
	if !input.UpdateExisting {
		return EmailPasswordUserResult{User: existing}, nil
	}

	params := (&firebaseauth.UserToUpdate{}).
		Password(input.Password).
		DisplayName(input.DisplayName).
		EmailVerified(input.EmailVerified).
		Disabled(false)
	user, err := client.UpdateUser(ctx, existing.UID, params)
	if err != nil {
		return EmailPasswordUserResult{}, fmt.Errorf("update Firebase Auth user: %w", err)
	}

	return EmailPasswordUserResult{User: user, Updated: true}, nil
}
