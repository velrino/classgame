package app

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"classgame-api/internal/config"
	"classgame-api/internal/modules/dashboard"
	healthcheck "classgame-api/internal/modules/health-check"
	mission "classgame-api/internal/modules/mission"
	teacherclass "classgame-api/internal/modules/teacher-class"
	"classgame-api/internal/modules/users"
	sharedanalytics "classgame-api/internal/shared/analytics"
	"classgame-api/internal/shared/requestmeta"
)

const defaultRequestBodyLimitBytes int64 = 256 * 1024

type Options struct {
	Analytics           sharedanalytics.Tracker
	Config              config.Config
	Dashboard           dashboard.HandlerOptions
	HealthChecks        []healthcheck.Check
	StartedAt           time.Time
	TeacherClass        teacherclass.HandlerOptions
	Mission             mission.HandlerOptions
	MissionAttachment   http.Handler
	StudentLearningTask http.Handler
	Users               users.HandlerOptions
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
	server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders: []string{requestmeta.RequestIDHeader, "traceparent"},
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
		AllowOrigins: []string{"*"},
	}))

	healthcheck.RegisterRoutes(server, healthcheck.HandlerOptions{
		AppEnv:         options.Config.AppEnv,
		Checks:         options.HealthChecks,
		CheckCacheTTL:  time.Minute,
		ServiceTimeout: 3 * time.Second,
		StartedAt:      options.StartedAt,
	})
	if options.MissionAttachment != nil {
		server.POST("/internal/tasks/mission-attachments", echo.WrapHandler(options.MissionAttachment))
	}
	if options.StudentLearningTask != nil {
		server.POST("/internal/tasks/student-learning", echo.WrapHandler(options.StudentLearningTask))
	}

	users.RegisterRoutes(server, options.Users)
	dashboard.RegisterRoutes(server, options.Dashboard)
	teacherclass.RegisterRoutes(server, options.TeacherClass)
	mission.RegisterRoutes(server, options.Mission)

	return server
}
