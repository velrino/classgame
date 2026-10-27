package health

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"

	"classgame-api/internal/timeutil"
)

const (
	defaultCheckCacheTTL       = time.Minute
	defaultServiceCheckTimeout = 3 * time.Second
)

type ServiceCheck func(context.Context) error

type HealthCheckServiceName string

const (
	HealthCheckServiceFirebase HealthCheckServiceName = "firebase"
	HealthCheckServiceRedis    HealthCheckServiceName = "redis"
)

type HealthCheckStatus string

const (
	HealthCheckStatusUp   HealthCheckStatus = "up"
	HealthCheckStatusDown HealthCheckStatus = "down"
)

type HealthCheckResponseStatus string

const (
	HealthCheckResponseStatusOK       HealthCheckResponseStatus = "ok"
	HealthCheckResponseStatusDegraded HealthCheckResponseStatus = "degraded"
)

type HandlerOptions struct {
	AppEnv         string
	Checks         []Check
	CheckCacheTTL  time.Duration
	ServiceTimeout time.Duration
	StartedAt      time.Time
}

type Handler struct {
	appEnv         string
	checks         []Check
	checkCacheTTL  time.Duration
	cache          healthCheckCache
	serviceTimeout time.Duration
	startedAt      time.Time
}

type Check struct {
	Name    HealthCheckServiceName
	Run     ServiceCheck
	Timeout time.Duration
}

type AuthorResponse struct {
	Name      string `json:"name"`
	GitHub    string `json:"github"`
	LinkedIn  string `json:"linkedin"`
	Instagram string `json:"instagram"`
	Website   string `json:"website"`
}

type HealthCheckResponse struct {
	Name       string                     `json:"name"`
	Status     HealthCheckResponseStatus  `json:"status"`
	AppEnv     string                     `json:"app_env"`
	Author     AuthorResponse             `json:"author"`
	Cache      HealthCheckCacheResponse   `json:"cache"`
	ResultTime HealthCheckResultTime      `json:"result_time"`
	Checks     HealthCheckResult          `json:"checks"`
	Runtime    HealthCheckRuntimeResponse `json:"runtime"`
}

type HealthCheckCacheResponse struct {
	Hit              bool   `json:"hit"`
	CreatedAt        string `json:"created_at"`
	ExpiresAt        string `json:"expires_at"`
	SecondsRemaining int64  `json:"seconds_remaining"`
}

type HealthCheckRuntimeResponse struct {
	LastDeploy string                       `json:"last_deploy"`
	Uptime     timeutil.TimeBetweenDatesDTO `json:"uptime"`
}

type HealthCheckResultTime struct {
	Total   int64 `json:"total"`
	Average int64 `json:"average"`
}

type HealthCheckResult map[HealthCheckServiceName]HealthCheckServiceResponse

type HealthCheckServiceResponse struct {
	Status       HealthCheckStatus `json:"status"`
	ResponseTime int64             `json:"response_time"`
	Message      string            `json:"message,omitempty"`
}

type healthCheckResultItem struct {
	Name     HealthCheckServiceName
	Response HealthCheckServiceResponse
}

type healthCheckSnapshot struct {
	Checks HealthCheckResult
	Cache  healthCheckCacheInfo
}

type healthCheckCacheInfo struct {
	Hit       bool
	CreatedAt time.Time
	ExpiresAt time.Time
}

type healthCheckCache struct {
	mu        sync.Mutex
	checks    HealthCheckResult
	createdAt time.Time
	expiresAt time.Time
	ready     bool
}

var classGameAuthor = AuthorResponse{
	Name:      "velrino",
	GitHub:    "https://github.com/velrino/",
	LinkedIn:  "https://www.linkedin.com/in/velrino/",
	Instagram: "https://www.instagram.com/velrino",
	Website:   "https://velrino.vercel.app/",
}

func RegisterRoutes(server *echo.Echo, options HandlerOptions) {
	serviceTimeout := options.ServiceTimeout
	if serviceTimeout <= 0 {
		serviceTimeout = defaultServiceCheckTimeout
	}
	checkCacheTTL := options.CheckCacheTTL
	if checkCacheTTL <= 0 {
		checkCacheTTL = defaultCheckCacheTTL
	}

	handler := &Handler{
		appEnv:         options.AppEnv,
		checks:         options.Checks,
		checkCacheTTL:  checkCacheTTL,
		serviceTimeout: serviceTimeout,
		startedAt:      options.StartedAt,
	}

	server.GET("/", handler.GetHealthCheck)
}

func (handler *Handler) GetHealthCheck(ctx *echo.Context) error {
	startedAt := time.Now()

	snapshot := handler.buildChecks(context.WithoutCancel(ctx.Request().Context()))
	status := handler.buildStatus(snapshot.Checks)
	now := time.Now()

	response := HealthCheckResponse{
		Name:   "ClassGame API",
		Status: status,
		AppEnv: handler.appEnv,
		Author: classGameAuthor,
		Cache:  snapshot.Cache.response(),
		ResultTime: HealthCheckResultTime{
			Total:   responseTimeMilliseconds(time.Since(startedAt)),
			Average: averageResponseTimeMilliseconds(snapshot.Checks),
		},
		Checks: snapshot.Checks,
		Runtime: HealthCheckRuntimeResponse{
			LastDeploy: handler.startedAt.Format(time.RFC3339Nano),
			Uptime:     timeutil.CalculateTimeBetweenDates(handler.startedAt, now),
		},
	}

	return ctx.JSON(handler.httpStatusCode(status), response)
}

func (handler *Handler) buildChecks(ctx context.Context) healthCheckSnapshot {
	if checks, cacheInfo, ok := handler.readCachedChecks(); ok {
		return healthCheckSnapshot{
			Checks: checks,
			Cache:  cacheInfo,
		}
	}

	result := make(HealthCheckResult, len(handler.checks))
	resultChannel := make(chan healthCheckResultItem, len(handler.checks))

	for _, check := range handler.checks {
		go func(check Check) {
			resultChannel <- healthCheckResultItem{
				Name:     check.Name,
				Response: handler.checkService(ctx, check),
			}
		}(check)
	}

	for range handler.checks {
		checkResult := <-resultChannel
		result[checkResult.Name] = checkResult.Response
	}

	cacheInfo := handler.writeCachedChecks(result)

	return healthCheckSnapshot{
		Checks: result,
		Cache:  cacheInfo,
	}
}

func (handler *Handler) checkService(ctx context.Context, check Check) HealthCheckServiceResponse {
	startedAt := time.Now()

	if check.Run == nil {
		return HealthCheckServiceResponse{
			Status:       HealthCheckStatusDown,
			ResponseTime: responseTimeMilliseconds(time.Since(startedAt)),
			Message:      "CHECK_NOT_CONFIGURED",
		}
	}

	timeout := check.Timeout
	if timeout <= 0 {
		timeout = handler.serviceTimeout
	}

	checkContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := check.Run(checkContext); err != nil {
		slog.Warn("health check failed", "service", check.Name, "error", err)

		return HealthCheckServiceResponse{
			Status:       HealthCheckStatusDown,
			ResponseTime: responseTimeMilliseconds(time.Since(startedAt)),
			Message:      healthCheckErrorMessage(err),
		}
	}

	return HealthCheckServiceResponse{
		Status:       HealthCheckStatusUp,
		ResponseTime: responseTimeMilliseconds(time.Since(startedAt)),
	}
}

func (handler *Handler) buildStatus(checks HealthCheckResult) HealthCheckResponseStatus {
	for _, check := range checks {
		if check.Status == HealthCheckStatusDown {
			return HealthCheckResponseStatusDegraded
		}
	}

	return HealthCheckResponseStatusOK
}

func (handler *Handler) httpStatusCode(status HealthCheckResponseStatus) int {
	if status == HealthCheckResponseStatusDegraded {
		return http.StatusServiceUnavailable
	}

	return http.StatusOK
}

func (handler *Handler) readCachedChecks() (HealthCheckResult, healthCheckCacheInfo, bool) {
	handler.cache.mu.Lock()
	defer handler.cache.mu.Unlock()

	if !handler.cache.ready || time.Now().After(handler.cache.expiresAt) {
		return nil, healthCheckCacheInfo{}, false
	}

	return cloneHealthCheckResult(handler.cache.checks), healthCheckCacheInfo{
		Hit:       true,
		CreatedAt: handler.cache.createdAt,
		ExpiresAt: handler.cache.expiresAt,
	}, true
}

func (handler *Handler) writeCachedChecks(checks HealthCheckResult) healthCheckCacheInfo {
	handler.cache.mu.Lock()
	defer handler.cache.mu.Unlock()

	createdAt := time.Now()
	expiresAt := createdAt.Add(handler.checkCacheTTL)

	handler.cache.checks = cloneHealthCheckResult(checks)
	handler.cache.createdAt = createdAt
	handler.cache.expiresAt = expiresAt
	handler.cache.ready = true

	return healthCheckCacheInfo{
		Hit:       false,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}
}

func (cacheInfo healthCheckCacheInfo) response() HealthCheckCacheResponse {
	return HealthCheckCacheResponse{
		Hit:              cacheInfo.Hit,
		CreatedAt:        cacheInfo.CreatedAt.Format(time.RFC3339Nano),
		ExpiresAt:        cacheInfo.ExpiresAt.Format(time.RFC3339Nano),
		SecondsRemaining: secondsRemaining(cacheInfo.ExpiresAt),
	}
}

func secondsRemaining(expiresAt time.Time) int64 {
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return 0
	}

	return int64(math.Ceil(remaining.Seconds()))
}

func healthCheckErrorMessage(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "TIMEOUT_EXCEEDED"
	}
	if errors.Is(err, context.Canceled) {
		return "REQUEST_CANCELLED"
	}

	errorMessage := strings.ToLower(err.Error())
	if strings.Contains(errorMessage, "auth") ||
		strings.Contains(errorMessage, "unauthorized") ||
		strings.Contains(errorMessage, "wrongpass") ||
		strings.Contains(errorMessage, "noauth") {
		return "AUTH_FAILED"
	}

	return "UNAVAILABLE"
}

func responseTimeMilliseconds(duration time.Duration) int64 {
	milliseconds := duration.Milliseconds()
	if duration > 0 && milliseconds == 0 {
		milliseconds = 1
	}

	return milliseconds
}

func averageResponseTimeMilliseconds(checks HealthCheckResult) int64 {
	if len(checks) == 0 {
		return 0
	}

	var total int64
	for _, check := range checks {
		total += check.ResponseTime
	}

	return total / int64(len(checks))
}

func cloneHealthCheckResult(input HealthCheckResult) HealthCheckResult {
	output := make(HealthCheckResult, len(input))
	for key, value := range input {
		output[key] = value
	}

	return output
}
