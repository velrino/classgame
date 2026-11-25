package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
)

var ErrTeacherNotAllowed = errors.New("teacher account is not allowed")

// TeacherEmailPolicy is separate from token verification and sign-in providers.
// An empty list keeps the development sign-in policy; production config forbids it.
func TeacherEmailPolicy(next Authenticator, emails string) (Authenticator, error) {
	if next == nil {
		return nil, errors.New("teacher authenticator is required")
	}
	allowed, err := ParseTeacherEmails(emails)
	if err != nil {
		return nil, err
	}
	return teacherEmailPolicy{next: next, allowed: allowed}, nil
}

type teacherEmailPolicy struct {
	next    Authenticator
	allowed map[string]bool
}

func (policy teacherEmailPolicy) AuthenticateBearerToken(ctx context.Context, header string) (Identity, error) {
	identity, err := policy.next.AuthenticateBearerToken(ctx, header)
	if err != nil {
		return Identity{}, err
	}
	if !TeacherEmailAllowed(identity, policy.allowed) {
		return Identity{}, ErrTeacherNotAllowed
	}
	return identity, nil
}

func TeacherEmailAllowed(identity Identity, allowed map[string]bool) bool {
	return IsTeacherAuthMethod(identity.AuthMethod) && !identity.Disabled && (len(allowed) == 0 || identity.EmailVerified && allowed[strings.ToLower(strings.TrimSpace(identity.Email))])
}

func ParseTeacherEmails(emails string) (map[string]bool, error) {
	allowed := map[string]bool{}
	if strings.TrimSpace(emails) != "" {
		for _, value := range strings.Split(emails, ",") {
			value = strings.ToLower(strings.TrimSpace(value))
			address, err := mail.ParseAddress(value)
			if err != nil || address.Address != value || strings.ContainsAny(value, "*<>") {
				return nil, errors.New("teacher allowlist requires exact email addresses")
			}
			allowed[value] = true
		}
	}
	return allowed, nil
}
