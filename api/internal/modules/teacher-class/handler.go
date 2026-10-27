package teacherclass

import (
	"classgame-api/internal/pagination"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	sharedanalytics "classgame-api/internal/shared/analytics"
	sharedfirebase "classgame-api/internal/shared/google/firebase"
)

const (
	defaultTeacherClassTimeout          = 8 * time.Second
	defaultTeacherClassRateLimitRate    = 0.5
	defaultTeacherClassRateLimitBurst   = 20
	defaultTeacherClassRateLimitExpires = 5 * time.Minute
)

var classIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,39}$`)

type Authenticator interface {
	AuthenticateBearerToken(context.Context, string) (sharedfirebase.Identity, error)
}

type TeacherClassService interface {
	Get(context.Context, sharedfirebase.Identity, string) (TeacherClassDocument, bool, error)
	ListPage(context.Context, sharedfirebase.Identity, pagination.Params) ([]TeacherClassDocument, string, error)
	Upsert(context.Context, sharedfirebase.Identity, UpsertTeacherClassInput) (TeacherClassDocument, error)
}

type HandlerOptions struct {
	Analytics                    sharedanalytics.Tracker
	Authenticator                Authenticator
	TeacherClassTimeout          time.Duration
	TeacherClassRateLimitRate    float64
	TeacherClassRateLimitBurst   int
	TeacherClassRateLimitExpires time.Duration
	TeacherClassService          TeacherClassService
}

type Handler struct {
	analytics           sharedanalytics.Tracker
	authenticator       Authenticator
	teacherClassTimeout time.Duration
	teacherClassService TeacherClassService
}

type UpsertTeacherClassRequest struct {
	ClassID    string `json:"class_id"`
	ClassName  string `json:"class_name"`
	Subject    string `json:"subject"`
	GradeLevel string `json:"grade_level"`
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
		analytics:           options.Analytics,
		authenticator:       options.Authenticator,
		teacherClassTimeout: resolveTeacherClassTimeout(options.TeacherClassTimeout),
		teacherClassService: options.TeacherClassService,
	}

	group := server.Group("/teacher-classes", teacherClassRateLimiter(options))
	group.POST("", handler.PostTeacherClass)
	group.GET("", handler.GetTeacherClasses)
	group.GET("/:class_id", handler.GetTeacherClass)
}

func (handler *Handler) GetTeacherClasses(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.teacherClassService == nil {
		slog.Error("teacher class dependencies are not configured")
		return jsonError(ctx, http.StatusInternalServerError, "not_configured", "Teacher classes are not configured.")
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.teacherClassTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}

	page, err := pagination.Parse(ctx.QueryParams())
	if err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}
	teacherClasses, next, err := handler.teacherClassService.ListPage(requestContext, identity, page)
	if err != nil {
		return handler.serviceError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{
		"teacher_classes": ToResponses(teacherClasses),
		"next_cursor":     next,
	})
}

func (handler *Handler) PostTeacherClass(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.teacherClassService == nil {
		slog.Error("teacher class dependencies are not configured")
		return jsonError(ctx, http.StatusInternalServerError, "not_configured", "Teacher classes are not configured.")
	}

	var request UpsertTeacherClassRequest
	if err := ctx.Bind(&request); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "bad_request", "Invalid JSON body.")
	}
	if err := request.Validate(); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.teacherClassTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}

	teacherClass, err := handler.teacherClassService.Upsert(requestContext, identity, UpsertTeacherClassInput{
		ClassID:    request.ClassID,
		ClassName:  request.ClassName,
		Subject:    request.Subject,
		GradeLevel: request.GradeLevel,
	})
	if err != nil {
		return handler.serviceError(ctx, err)
	}
	if strings.TrimSpace(request.ClassID) == "" {
		sharedanalytics.Track(
			handler.analytics,
			requestContext,
			sharedanalytics.EventClassCreated,
			identity.ProviderUserID,
			sharedanalytics.Properties{"class_id": teacherClass.ClassID},
		)
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"teacher_class": ToResponse(teacherClass),
	})
}

func (handler *Handler) GetTeacherClass(ctx *echo.Context) error {
	if handler.authenticator == nil || handler.teacherClassService == nil {
		slog.Error("teacher class dependencies are not configured")
		return jsonError(ctx, http.StatusInternalServerError, "not_configured", "Teacher classes are not configured.")
	}

	classID := ctx.Param("class_id")
	if err := validateClassID(classID); err != nil {
		return jsonError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	}

	requestContext, cancel := context.WithTimeout(ctx.Request().Context(), handler.teacherClassTimeout)
	defer cancel()

	identity, ok, err := handler.authenticate(requestContext, ctx)
	if err != nil || !ok {
		return err
	}

	teacherClass, exists, err := handler.teacherClassService.Get(requestContext, identity, classID)
	if err != nil {
		return handler.serviceError(ctx, err)
	}
	if !exists {
		return jsonError(ctx, http.StatusNotFound, "not_found", "Teacher class not found.")
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"teacher_class": ToResponse(teacherClass),
	})
}

func (handler *Handler) authenticate(
	ctx context.Context,
	echoCtx *echo.Context,
) (sharedfirebase.Identity, bool, error) {
	identity, err := handler.authenticator.AuthenticateBearerToken(
		ctx,
		echoCtx.Request().Header.Get(echo.HeaderAuthorization),
	)
	if err == nil {
		return identity, true, nil
	}

	if errors.Is(err, sharedfirebase.ErrMissingBearerToken) ||
		errors.Is(err, sharedfirebase.ErrInvalidBearerToken) {
		return sharedfirebase.Identity{}, false, jsonError(echoCtx, http.StatusUnauthorized, "unauthorized", "Invalid Firebase ID token.")
	}

	slog.Warn("teacher class auth failed", "error", err)
	return sharedfirebase.Identity{}, false, jsonError(echoCtx, http.StatusUnauthorized, "unauthorized", "Invalid Firebase ID token.")
}

func (handler *Handler) serviceError(ctx *echo.Context, err error) error {
	if errors.Is(err, ErrTeacherRequiresGoogleAuth) {
		return jsonError(ctx, http.StatusForbidden, "forbidden", "Teacher classes require Google sign in.")
	}
	if errors.Is(err, ErrClassOwnedByAnotherTeacher) {
		return jsonError(ctx, http.StatusForbidden, "forbidden", "This class code already belongs to another teacher.")
	}

	slog.Error("teacher class request failed", "error", err)
	return jsonError(ctx, http.StatusInternalServerError, "teacher_class_failed", "Could not save teacher class.")
}

func (request UpsertTeacherClassRequest) Validate() error {
	if strings.TrimSpace(request.ClassID) != "" {
		if err := validateClassID(request.ClassID); err != nil {
			return err
		}
	}

	return validation.ValidateStruct(&request,
		validation.Field(&request.ClassName,
			validation.Length(0, 120),
		),
		validation.Field(&request.Subject,
			validation.Required,
			validation.Length(2, 80),
		),
		validation.Field(&request.GradeLevel,
			validation.Length(0, 80),
		),
	)
}

func validateClassID(value string) error {
	return validation.Validate(value,
		validation.Required,
		validation.Length(2, 40),
		validation.Match(classIDPattern),
	)
}

func teacherClassRateLimiter(options HandlerOptions) echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      resolveTeacherClassRateLimitRate(options.TeacherClassRateLimitRate),
			Burst:     resolveTeacherClassRateLimitBurst(options.TeacherClassRateLimitBurst),
			ExpiresIn: resolveTeacherClassRateLimitExpires(options.TeacherClassRateLimitExpires),
		}),
		IdentifierExtractor: func(ctx *echo.Context) (string, error) {
			return "teacher_class:" + ctx.RealIP(), nil
		},
		DenyHandler: func(ctx *echo.Context, _ string, _ error) error {
			return jsonError(ctx, http.StatusTooManyRequests, "rate_limited", "Too many teacher class requests.")
		},
	})
}

func resolveTeacherClassTimeout(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultTeacherClassTimeout
	}

	return value
}

func resolveTeacherClassRateLimitRate(value float64) float64 {
	if value <= 0 {
		return defaultTeacherClassRateLimitRate
	}

	return value
}

func resolveTeacherClassRateLimitBurst(value int) int {
	if value <= 0 {
		return defaultTeacherClassRateLimitBurst
	}

	return value
}

func resolveTeacherClassRateLimitExpires(value time.Duration) time.Duration {
	if value <= 0 {
		return defaultTeacherClassRateLimitExpires
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
