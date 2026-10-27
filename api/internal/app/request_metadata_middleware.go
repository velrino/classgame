package app

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"

	"classgame-api/internal/shared/requestmeta"
)

func requestMetadataMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx *echo.Context) error {
		started := time.Now()
		requestContext := requestmeta.FromRequest(ctx.Request())
		ctx.SetRequest(ctx.Request().WithContext(requestContext))
		requestmeta.Inject(requestContext, ctx.Response().Header())
		err := next(ctx)
		route := ctx.Path()
		if route == "" {
			route = "<unmatched>"
		}
		slog.InfoContext(requestContext, "api request",
			"request_id", requestmeta.RequestID(requestContext),
			"trace_id", requestmeta.TraceID(requestContext),
			"method", ctx.Request().Method,
			"route", route,
			"status", responseStatusCode(ctx, err),
			"duration_ms", time.Since(started).Milliseconds(),
		)
		return err
	}
}
