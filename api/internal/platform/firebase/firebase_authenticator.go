package firebase

import (
	"context"
	"fmt"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"

	"classgame-api/internal/auth"
)

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
) (auth.Identity, error) {
	idToken, err := extractBearerToken(authorizationHeader)
	if err != nil {
		return auth.Identity{}, err
	}

	if authenticator == nil || authenticator.client == nil {
		return auth.Identity{}, auth.ErrInvalidBearerToken
	}

	token, err := authenticator.client.VerifyIDTokenAndCheckRevoked(ctx, idToken)
	if err != nil {
		return auth.Identity{}, fmt.Errorf("%w: %v", auth.ErrInvalidBearerToken, err)
	}

	user, err := authenticator.client.GetUser(ctx, token.UID)
	if err != nil {
		return auth.Identity{}, fmt.Errorf("get firebase user: %w", err)
	}

	providerIDs := readProviderIDs(user.ProviderUserInfo)

	return auth.Identity{
		Provider:       auth.IdentityProviderFirebase,
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
		return "", auth.ErrMissingBearerToken
	}

	parts := strings.Fields(authorizationHeader)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", auth.ErrMissingBearerToken
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

func resolveFirebaseAuthMethod(input resolveFirebaseAuthMethodInput) auth.AuthMethod {
	switch input.SignInProvider {
	case "anonymous":
		return auth.AuthMethodAnonymous
	case "google.com":
		return auth.AuthMethodGoogle
	case "password":
		return auth.AuthMethodEmail
	}

	// Unknown token providers must never inherit linked-account privileges.
	if strings.TrimSpace(input.SignInProvider) != "" {
		return auth.AuthMethodUnknown
	}

	if hasProviderID(input.ProviderIDs, "google.com") {
		return auth.AuthMethodGoogle
	}
	if hasProviderID(input.ProviderIDs, "password") || strings.TrimSpace(input.Email) != "" {
		return auth.AuthMethodEmail
	}

	return auth.AuthMethodUnknown
}

func hasProviderID(providerIDs []string, expectedProviderID string) bool {
	for _, providerID := range providerIDs {
		if providerID == expectedProviderID {
			return true
		}
	}

	return false
}
