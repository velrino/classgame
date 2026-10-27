package insights

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"classgame-api/internal/auth"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/pagination"
	"classgame-api/internal/transport/httpx"
)

const (
	defaultDashboardTimeout          = 15 * time.Second
	defaultDashboardRateLimitRate    = 0.5
	defaultDashboardRateLimitBurst   = 20
	defaultDashboardRateLimitExpires = 5 * time.Minute
)

var classIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,39}$`)

type Authenticator = auth.Authenticator

type DashboardService interface {
	GetOverview(context.Context, auth.Identity, string) (DashboardOverview, error)
	GetOverviewPage(context.Context, auth.Identity, string, pagination.Params) (DashboardOverview, error)
	GetStudentHistory(context.Context, auth.Identity, string, string, pagination.Params) (StudentHistoryResponse, error)
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

type ErrorResponse = httpx.ErrorResponse

type ErrorResponseBody = httpx.ErrorResponseBody

func RegisterRoutes(server *echo.Echo, options HandlerOptions) {
	handler := &Handler{
		authenticator:    options.Authenticator,
		dashboardTimeout: resolveDashboardTimeout(options.DashboardTimeout),
		dashboardService: options.DashboardService,
	}

	group := server.Group("/dashboard", dashboardRateLimiter(options))
	group.GET("/overview", handler.GetOverview)
	group.GET("/overview/page", handler.GetOverviewPage)
	group.GET("/students/:student_id/attempts", handler.GetStudentHistory)
}

func (handler *Handler) GetOverview(ctx *echo.Context) error     { return handler.overview(ctx, false) }
func (handler *Handler) GetOverviewPage(ctx *echo.Context) error { return handler.overview(ctx, true) }

func (handler *Handler) overview(ctx *echo.Context, paged bool) error {
	if handler.authenticator == nil || handler.dashboardService == nil {
		slog.Error("dashboard dependencies are not configured")
		return httpx.JSONError(ctx, http.StatusInternalServerError, "not_configured", "Dashboard is not configured.")
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
			return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
		}
	}

	var overview DashboardOverview
	if paged {
		page, parseErr := pagination.Parse(ctx.QueryParams())
		if ctx.QueryParam("limit") == "" {
			page.Limit = 25
		}
		if parseErr != nil {
			return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", parseErr.Error())
		}
		overview, err = handler.dashboardService.GetOverviewPage(requestContext, identity, classID, page)
	} else {
		overview, err = handler.dashboardService.GetOverview(requestContext, identity, classID)
	}
	if err != nil {
		return handler.serviceError(ctx, err)
	}

	return ctx.JSON(http.StatusOK, ToOverviewResponse(overview))
}

func (handler *Handler) GetStudentHistory(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.dashboardService == nil {
		return httpx.JSONError(ctx, http.StatusInternalServerError, "not_configured", "Dashboard is not configured.")
	}
	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.dashboardTimeout)
	defer cancel()
	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}
	classID := ctx.QueryParam("class_id")
	if err := validateClassID(classID); err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", "A valid class_id is required.")
	}
	page, err := pagination.Parse(ctx.QueryParams())
	if ctx.QueryParam("limit") == "" {
		page.Limit = 5
	}
	if err != nil {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}
	result, err := handler.dashboardService.GetStudentHistory(requestContext, identity, classID, ctx.Param("student_id"), page)
	if err != nil {
		return handler.serviceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (handler *Handler) authenticate(
	ctx context.Context,
	echoCtx *echo.Context,
) (auth.Identity, bool, error) {
	return httpx.Authenticate(ctx, echoCtx, handler.authenticator)
}

func (handler *Handler) serviceError(ctx *echo.Context, err error) error {
	if status.Code(err) == codes.FailedPrecondition {
		return httpx.JSONError(ctx, http.StatusServiceUnavailable, "query_index_required", "The query indexes are not ready. Ask the operator to review firestore.indexes.json before using this view.")
	}

	if errors.Is(err, ErrInvalidPage) {
		return httpx.JSONError(ctx, http.StatusBadRequest, "validation_failed", "Invalid page or cursor. History limits are 1 to 10.")
	}
	if errors.Is(err, ErrPaginationRequired) {
		return httpx.JSONError(ctx, http.StatusConflict, "pagination_required", "This class needs pages. Use /dashboard/overview/page and the student history endpoint.")
	}
	if errors.Is(err, classrooms.ErrClassNotFound) {
		return httpx.JSONError(ctx, http.StatusNotFound, "class_not_found", "Class not found.")
	}
	if errors.Is(err, ErrTeacherAuthRequired) || errors.Is(err, classrooms.ErrTeacherAuthRequired) {
		return httpx.JSONError(ctx, http.StatusForbidden, "forbidden", "Teacher sign-in is required.")
	}
	if errors.Is(err, classrooms.ErrClassOwnedByAnotherTeacher) {
		return httpx.JSONError(ctx, http.StatusForbidden, "forbidden", "This class code already belongs to another teacher.")
	}

	slog.Error("dashboard request failed", "error", err)
	return httpx.JSONError(ctx, http.StatusInternalServerError, "dashboard_failed", "Could not load teacher dashboard.")
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
			return httpx.JSONError(ctx, http.StatusTooManyRequests, "rate_limited", "Too many dashboard requests.")
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
