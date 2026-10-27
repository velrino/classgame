// Package httpx contains the shared HTTP contract and authentication boundary.
package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"classgame-api/internal/auth"
)

type ErrorResponse struct {
	Error ErrorResponseBody `json:"error"`
}
type ErrorResponseBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSONError(ctx *echo.Context, status int, code, message string) error {
	return ctx.JSON(status, ErrorResponse{Error: ErrorResponseBody{Code: code, Message: message}})
}
func Authenticate(ctx context.Context, transport *echo.Context, authenticator auth.Authenticator) (auth.Identity, bool, error) {
	if authenticator == nil {
		return auth.Identity{}, false, JSONError(transport, http.StatusServiceUnavailable, "not_configured", "Authentication is not configured.")
	}
	identity, err := authenticator.AuthenticateBearerToken(ctx, transport.Request().Header.Get(echo.HeaderAuthorization))
	if err == nil {
		return identity, true, nil
	}
	if !errors.Is(err, auth.ErrMissingBearerToken) && !errors.Is(err, auth.ErrInvalidBearerToken) {
		slog.Warn("request authentication failed", "error", err)
	}
	return auth.Identity{}, false, JSONError(transport, http.StatusUnauthorized, "unauthorized", "Invalid Firebase ID token.")
}
