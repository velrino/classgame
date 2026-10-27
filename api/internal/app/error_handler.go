package app

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"classgame-api/internal/transport/httpx"
)

type errorResponse = httpx.ErrorResponse

type errorResponseBody = httpx.ErrorResponseBody

func httpErrorHandler(ctx *echo.Context, err error) {
	if response, unwrapErr := echo.UnwrapResponse(ctx.Response()); unwrapErr == nil && response.Committed {
		return
	}

	statusCode := resolveHTTPStatusCode(err)
	message := http.StatusText(statusCode)
	if message == "" {
		message = "Internal Server Error"
	}

	if statusCode >= http.StatusInternalServerError {
		slog.Error(
			"unhandled http error",
			"error", err,
			"method", ctx.Request().Method,
			"path", requestPath(ctx),
			"status_code", statusCode,
		)
	}

	if ctx.Request().Method == http.MethodHead {
		if writeErr := ctx.NoContent(statusCode); writeErr != nil {
			slog.Error("http error handler failed to write response", "error", writeErr)
		}
		return
	}

	if writeErr := ctx.JSON(statusCode, errorResponse{
		Error: errorResponseBody{
			Code:    errorCodeForStatus(statusCode),
			Message: message,
		},
	}); writeErr != nil {
		slog.Error("http error handler failed to write response", "error", writeErr)
	}
}

func resolveHTTPStatusCode(err error) int {
	var statusCoder echo.HTTPStatusCoder
	if errors.As(err, &statusCoder) && statusCoder.StatusCode() != 0 {
		return statusCoder.StatusCode()
	}

	return http.StatusInternalServerError
}

func isRecoveredPanic(err error) bool {
	var panicStackError *middleware.PanicStackError
	return errors.As(err, &panicStackError)
}

func errorCodeForStatus(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusRequestTimeout:
		return "request_timeout"
	case http.StatusRequestEntityTooLarge:
		return "request_too_large"
	case http.StatusUnsupportedMediaType:
		return "unsupported_media_type"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusBadGateway:
		return "bad_gateway"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	default:
		if statusCode >= http.StatusInternalServerError {
			return "internal_server_error"
		}

		return "http_error"
	}
}

func requestPath(ctx *echo.Context) string {
	if routePath := ctx.Path(); routePath != "" {
		return routePath
	}

	return ctx.Request().URL.Path
}
