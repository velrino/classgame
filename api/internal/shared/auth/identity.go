package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMissingBearerToken = errors.New("missing bearer token")
	ErrInvalidBearerToken = errors.New("invalid bearer token")
)

type IdentityProvider string

const (
	IdentityProviderFirebase IdentityProvider = "firebase"
)

type AuthMethod string

const (
	AuthMethodAnonymous AuthMethod = "anonymous"
	AuthMethodEmail     AuthMethod = "email"
	AuthMethodGoogle    AuthMethod = "google"
	AuthMethodUnknown   AuthMethod = "unknown"
)

// IsTeacherAuthMethod reports whether a sign-in method is accepted for teachers:
// Google, or email + password (the shared teacher account). Students are anonymous.
func IsTeacherAuthMethod(method AuthMethod) bool {
	return method == AuthMethodGoogle || method == AuthMethodEmail
}

type Identity struct {
	Provider       IdentityProvider
	ProviderUserID string
	AuthMethod     AuthMethod
	Email          string
	EmailVerified  bool
	DisplayName    string
	PhotoURL       string
	Disabled       bool
	ProviderIDs    []string
	SignInProvider string
	CustomClaims   map[string]interface{}
	TokenAuthTime  int64
	TokenExpiresAt int64
	TokenIssuedAt  int64
	TokenIssuer    string
	TokenAudience  string
}

// Authenticator is the narrow inbound authentication port.
type Authenticator interface {
	AuthenticateBearerToken(context.Context, string) (Identity, error)
}

// RestrictProviders enforces the configured sign-in allowlist after token verification.
func RestrictProviders(next Authenticator, providers string) (Authenticator, error) {
	allowed := map[AuthMethod]bool{}
	for _, provider := range strings.Split(providers, ",") {
		switch strings.TrimSpace(provider) {
		case "google":
			allowed[AuthMethodGoogle] = true
		case "password":
			allowed[AuthMethodEmail] = true
		case "anonymous":
			allowed[AuthMethodAnonymous] = true
		default:
			return nil, fmt.Errorf("unsupported authentication provider %q", strings.TrimSpace(provider))
		}
	}
	if next == nil || len(allowed) == 0 {
		return nil, errors.New("authenticator and providers are required")
	}
	return providerPolicy{next: next, allowed: allowed}, nil
}

type providerPolicy struct {
	next    Authenticator
	allowed map[AuthMethod]bool
}

func (policy providerPolicy) AuthenticateBearerToken(ctx context.Context, header string) (Identity, error) {
	identity, err := policy.next.AuthenticateBearerToken(ctx, header)
	if err != nil {
		return Identity{}, err
	}
	if identity.Disabled || !policy.allowed[identity.AuthMethod] {
		return Identity{}, ErrInvalidBearerToken
	}
	return identity, nil
}
