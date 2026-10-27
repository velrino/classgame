package dashboard

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	teacherclass "classgame-api/internal/modules/teacher-class"
	sharedauth "classgame-api/internal/shared/auth"
)

const (
	defaultDashboardTimeout          = 15 * time.Second
	defaultDashboardRateLimitRate    = 0.5
	defaultDashboardRateLimitBurst   = 20
	defaultDashboardRateLimitExpires = 5 * time.Minute
)

var classIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,39}$`)

type Authenticator interface {
	AuthenticateBearerToken(context.Context, string) (sharedauth.Identity, error)
}

type DashboardService interface {
	GetOverview(context.Context, sharedauth.Identity, string) (DashboardOverview, error)
}

type HandlerOptions struct {
	Authenticator             Authenticator
	DashboardTimeout          time.Duration
	DashboardRateLimitRate    float64
	DashboardRateLimitBurst   int
	DashboardRateLimitExpires time.Duration
	DashboardService          DashboardService
}

type Handler struct {
	authenticator    Authenticator
	dashboardTimeout time.Duration
	dashboardService DashboardService
}

type ErrorResponse struct {
	Error ErrorResponseBody `json:"error"`
}

type ErrorResponseBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func RegisterRoutes(server *echo.Echo, options HandlerOptions) {
	handler := &Handler{
		authenticator:    options.Authenticator,
		dashboardTimeout: resolveDashboardTimeout(options.DashboardTimeout),
		dashboardService: options.DashboardService,
	}

	group := server.Group("/dashboard", dashboardRateLimiter(options))
	group.GET("/overview", handler.GetOverview)
}

func (handler *Handler) GetOverview(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.dashboardService == nil {
		slog.Error("dashboard dependencies are not configured")
		return jsonError(ctx, http.StatusInternalServerError, "not_configured", "Dashboard is not configured.")
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.dashboardTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}

	classID := ctx.QueryParam("class_id")
	if classID != "" {
		if err := validateClassID(classID); err != nil {
			return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
		}
	}

	overview, err := handler.dashboardService.GetOverview(requestContext, identity, classID)
	if err != nil {
		return handler.serviceError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, ToOverviewResponse(overview))
}

func (handler *Handler) authenticate(
	ctx context.Context,
	echoCtx *echo.Context,
) (sharedauth.Identity, bool, error) {
	identity, err := handler.authenticator.AuthenticateBearerToken(
		ctx,
		echoCtx.Request().Header.Get(echo.HeaderAuthorization),
	)
	if err == nil {
		return identity, true, nil
	}

	if errors.Is(err, sharedauth.ErrMissingBearerToken) ||
		errors.Is(err, sharedauth.ErrInvalidBearerToken) {
		return sharedauth.Identity{}, false, jsonError(echoCtx, http.StatusUnauthorized, "unauthorized", "Invalid Firebase ID token.")
	}

	slog.Warn("dashboard auth failed", "error", err)
	return sharedauth.Identity{}, false, jsonError(echoCtx, http.StatusUnauthorized, "unauthorized", "Invalid Firebase ID token.")
}

func (handler *Handler) serviceError(ctx *echo.Context, err error) error {
	if errors.Is(err, ErrTeacherRequiresGoogleAuth) || errors.Is(err, teacherclass.ErrTeacherRequiresGoogleAuth) {
		return jsonError(ctx, http.StatusForbidden, "forbidden", "Dashboard requires Google sign in.")
	}
	if errors.Is(err, teacherclass.ErrClassOwnedByAnotherTeacher) {
		return jsonError(ctx, http.StatusForbidden, "forbidden", "This class code already belongs to another teacher.")
	}

	slog.Error("dashboard request failed", "error", err)
	return jsonError(ctx, http.StatusInternalServerError, "dashboard_failed", "Could not load teacher dashboard.")
}

func validateClassID(value string) error {
	return validation.Validate(value,
		validation.Required,
		validation.Length(2, 40),
		validation.Match(classIDPattern),
	)
}

func dashboardRateLimiter(options HandlerOptions) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      resolveDashboardRateLimitRate(options.DashboardRateLimitRate),
			Burst:     resolveDashboardRateLimitBurst(options.DashboardRateLimitBurst),
			ExpiresIn: resolveDashboardRateLimitExpires(options.DashboardRateLimitExpires),
		}),
		IdentifierExtractor: func(ctx *echo.Context) (string, error) {
			return "dashboard:" + ctx.RealIP(), nil
		},
		DenyHandler: func(ctx *echo.Context, _ string, _ error) error {
			return jsonError(ctx, http.StatusTooManyRequests, "rate_limited", "Too many dashboard requests.")
		},
	})
}

func resolveDashboardTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultDashboardTimeout
	}

	return value
}

func resolveDashboardRateLimitRate(value float64) float64 {
	if value <= 0 {
		return defaultDashboardRateLimitRate
	}

	return value
}

func resolveDashboardRateLimitBurst(value int) int {
	if value <= 0 {
		return defaultDashboardRateLimitBurst
	}

	return value
}

func resolveDashboardRateLimitExpires(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultDashboardRateLimitExpires
	}

	return value
}

func jsonError(ctx *echo.Context, statusCode int, code string, message string) error {
	return ctx.JSON(statusCode, ErrorResponse{
		Error: ErrorResponseBody{
			Code:    code,
			Message: message,
		},
	})
}
