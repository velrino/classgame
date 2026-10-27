package app

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"

	"classgame-api/internal/platform/analytics"
)

func serverErrorCaptureMiddleware(tracker analytics.Tracker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx *echo.Context) error {
			err := next(ctx)
			statusCode := responseStatusCode(ctx, err)
			if statusCode < http.StatusInternalServerError || isRecoveredPanic(err) {
				return err
			}

			captureErr := err
			if captureErr == nil {
				captureErr = fmt.Errorf("api request returned HTTP %d", statusCode)
			}
			tracker.CaptureException(ctx.Request().Context(), captureErr, analytics.Properties{
				"error_source": "http",
				"http_method":  ctx.Request().Method,
				"http_route":   requestPath(ctx),
				"http_status":  statusCode,
			})

			return err
		}
	}
}

func responseStatusCode(ctx *echo.Context, err error) int {
	if err != nil {
		var statusCoder echo.HTTPStatusCoder
		if errors.As(err, &statusCoder) && statusCoder.StatusCode() != 0 {
			return statusCoder.StatusCode()
		}

		return http.StatusInternalServerError
	}

	if ctx == nil || ctx.Response() == nil {
		return http.StatusOK
	}
	response, unwrapErr := echo.UnwrapResponse(ctx.Response())
	if unwrapErr != nil {
		return http.StatusOK
	}

	return response.Status
}
