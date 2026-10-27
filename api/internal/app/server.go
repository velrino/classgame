package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"classgame-api/internal/apidocs"
	"classgame-api/internal/config"
	"classgame-api/internal/modules/classrooms"
	"classgame-api/internal/modules/health"
	"classgame-api/internal/modules/insights"
	"classgame-api/internal/modules/missions"
	"classgame-api/internal/modules/students"
	"classgame-api/internal/platform/analytics"
	"classgame-api/internal/platform/requestmeta"
)

const defaultRequestBodyLimitBytes int64 = 256 * 1024

type Options struct {
	Analytics    analytics.Tracker
	Config       config.Config
	Dashboard    insights.HandlerOptions
	HealthChecks []health.Check
	StartedAt    time.Time
	TeacherClass classrooms.HandlerOptions
	Mission      missions.HandlerOptions
	Users        students.HandlerOptions
}

func New(options Options) *echo.Echo {
	server := echo.New()
	server.HTTPErrorHandler = httpErrorHandler
	if options.Config.AppEnv == "production" {
		server.IPExtractor = echo.ExtractIPFromXFFHeader()
	} else {
		server.IPExtractor = echo.ExtractIPDirect()
	}
	server.Use(requestMetadataMiddleware)

	if options.Analytics != nil && options.Analytics.Enabled() {
		server.Use(serverErrorCaptureMiddleware(options.Analytics))
	}
	server.Use(middleware.Recover())
	if options.Analytics != nil && options.Analytics.Enabled() {
		server.Use(echo.WrapMiddleware(options.Analytics.RequestContextMiddleware))
	}
	server.Use(middleware.BodyLimit(defaultRequestBodyLimitBytes))
	origins := strings.Split(options.Config.CORSAllowedOrigins, ",")
	if options.Config.CORSAllowedOrigins == "" {
		origins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}
	for index := range origins {
		origins[index] = strings.TrimSpace(origins[index])
	}
	server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders: []string{requestmeta.RequestIDHeader, "traceparent", "Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowHeaders: []string{
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			echo.HeaderContentType,
			echo.HeaderOrigin,
			requestmeta.RequestIDHeader,
			"traceparent",
			"X-PostHog-Distinct-Id",
			"X-PostHog-Session-Id",
			"X-PostHog-Window-Id",
		},
		AllowMethods: []string{
			http.MethodDelete,
			http.MethodGet,
			http.MethodOptions,
			http.MethodPost,
		},
		AllowOrigins: origins,
	}))

	apidocs.RegisterRoutes(server)

	health.RegisterRoutes(server, health.HandlerOptions{
		AppEnv:         options.Config.AppEnv,
		Checks:         options.HealthChecks,
		CheckCacheTTL:  time.Minute,
		ServiceTimeout: 3 * time.Second,
		StartedAt:      options.StartedAt,
	})

	students.RegisterRoutes(server, options.Users)
	insights.RegisterRoutes(server, options.Dashboard)
	classrooms.RegisterRoutes(server, options.TeacherClass)
	missions.RegisterRoutes(server, options.Mission)

	return server
}
