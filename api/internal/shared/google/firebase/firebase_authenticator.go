package firebase

import (
	sharedauth "classgame-api/internal/shared/auth"
	"context"
	"fmt"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"
)

// Compatibility aliases keep existing adapters and callers source compatible.
type Identity = sharedauth.Identity
type IdentityProvider = sharedauth.IdentityProvider
type AuthMethod = sharedauth.AuthMethod

const (
	IdentityProviderFirebase = sharedauth.IdentityProviderFirebase
	AuthMethodAnonymous      = sharedauth.AuthMethodAnonymous
	AuthMethodEmail          = sharedauth.AuthMethodEmail
	AuthMethodGoogle         = sharedauth.AuthMethodGoogle
	AuthMethodUnknown        = sharedauth.AuthMethodUnknown
)

var ErrMissingBearerToken = sharedauth.ErrMissingBearerToken
var ErrInvalidBearerToken = sharedauth.ErrInvalidBearerToken

func IsTeacherAuthMethod(method AuthMethod) bool { return sharedauth.IsTeacherAuthMethod(method) }

type FirebaseAuthenticator struct {
	client *firebaseauth.Client
}

func NewFirebaseAuthenticator(client *firebaseauth.Client) *FirebaseAuthenticator {
	return &FirebaseAuthenticator{
		client: client,
	}
}

func (authenticator *FirebaseAuthenticator) AuthenticateBearerToken(
	ctx context.Context,
	authorizationHeader string,
) (Identity, error) {
	idToken, err := extractBearerToken(authorizationHeader)
	if err != nil {
		return Identity{}, err
	}

	if authenticator == nil || authenticator.client == nil {
		return Identity{}, ErrInvalidBearerToken
	}

	token, err := authenticator.client.VerifyIDTokenAndCheckRevoked(ctx, idToken)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %v", ErrInvalidBearerToken, err)
	}

	user, err := authenticator.client.GetUser(ctx, token.UID)
	if err != nil {
		return Identity{}, fmt.Errorf("get firebase user: %w", err)
	}

	providerIDs := readProviderIDs(user.ProviderUserInfo)

	return Identity{
		Provider:       IdentityProviderFirebase,
		ProviderUserID: token.UID,
		AuthMethod: resolveFirebaseAuthMethod(resolveFirebaseAuthMethodInput{
			Email:          user.Email,
			ProviderIDs:    providerIDs,
			SignInProvider: token.Firebase.SignInProvider,
		}),
		Email:          user.Email,
		EmailVerified:  user.EmailVerified,
		DisplayName:    user.DisplayName,
		PhotoURL:       user.PhotoURL,
		Disabled:       user.Disabled,
		ProviderIDs:    providerIDs,
		SignInProvider: token.Firebase.SignInProvider,
		CustomClaims:   user.CustomClaims,
		TokenAuthTime:  token.AuthTime,
		TokenExpiresAt: token.Expires,
		TokenIssuedAt:  token.IssuedAt,
		TokenIssuer:    token.Issuer,
		TokenAudience:  token.Audience,
	}, nil
}

func extractBearerToken(authorizationHeader string) (string, error) {
	authorizationHeader = strings.TrimSpace(authorizationHeader)
	if authorizationHeader == "" {
		return "", ErrMissingBearerToken
	}

	parts := strings.Fields(authorizationHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrMissingBearerToken
	}

	return parts[1], nil
}

func readProviderIDs(providers []*firebaseauth.UserInfo) []string {
	providerIDs := make([]string, 0, len(providers))

	for _, provider := range providers {
		if provider == nil || provider.ProviderID == "" {
			continue
		}

		providerIDs = append(providerIDs, provider.ProviderID)
	}

	return providerIDs
}

type resolveFirebaseAuthMethodInput struct {
	Email          string
	ProviderIDs    []string
	SignInProvider string
}

func resolveFirebaseAuthMethod(input resolveFirebaseAuthMethodInput) AuthMethod {
	switch input.SignInProvider {
	case "anonymous":
		return AuthMethodAnonymous
	case "google.com":
		return AuthMethodGoogle
	case "password":
		return AuthMethodEmail
	}

	// Unknown token providers must never inherit linked-account privileges.
	if strings.TrimSpace(input.SignInProvider) != "" {
		return AuthMethodUnknown
	}

	if hasProviderID(input.ProviderIDs, "google.com") {
		return AuthMethodGoogle
	}
	if hasProviderID(input.ProviderIDs, "password") || strings.TrimSpace(input.Email) != "" {
		return AuthMethodEmail
	}

	return AuthMethodUnknown
}

func hasProviderID(providerIDs []string, expectedProviderID string) bool {
	for _, providerID := range providerIDs {
		if providerID == expectedProviderID {
			return true
		}
	}

	return false
}
