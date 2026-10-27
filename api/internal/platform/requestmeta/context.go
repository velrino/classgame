package requestmeta

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const RequestIDHeader = "X-Request-ID"

type contextKey struct{}

type metadata struct {
	requestID       string
	generationScope string
}

// FromRequest accepts correlation metadata, never identity or authorization.
func FromRequest(request *http.Request) context.Context {
	ctx := propagation.TraceContext{}.Extract(request.Context(), propagation.HeaderCarrier(request.Header))
	id := strings.TrimSpace(request.Header.Get(RequestIDHeader))
	if !validRequestID(id) {
		id = rand.Text()
	}
	var scope string
	if bearer := strings.TrimSpace(request.Header.Get("Authorization")); bearer != "" {
		digest := sha256.Sum256([]byte(bearer))
		scope = hex.EncodeToString(digest[:])
	}
	ctx = context.WithValue(ctx, contextKey{}, metadata{requestID: id, generationScope: scope})

	parent := trace.SpanContextFromContext(ctx)
	traceID := parent.TraceID()
	if !traceID.IsValid() {
		if _, err := rand.Read(traceID[:]); err != nil {
			return ctx
		}
	}
	var spanID trace.SpanID
	if _, err := rand.Read(spanID[:]); err != nil {
		return ctx
	}
	// Propagation-only correlation does not install an SDK or export spans.
	return trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: parent.TraceFlags(),
	}))
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(contextKey{}).(metadata)
	return value.requestID
}

func TraceID(ctx context.Context) string {
	span := trace.SpanContextFromContext(ctx)
	if !span.IsValid() {
		return ""
	}
	return span.TraceID().String()
}

// GenerationScope is an opaque, process-local coalescing boundary, never a log field.
func GenerationScope(ctx context.Context) string {
	value, _ := ctx.Value(contextKey{}).(metadata)
	return value.generationScope
}

func Inject(ctx context.Context, headers http.Header) {
	if id := RequestID(ctx); id != "" {
		headers.Set(RequestIDHeader, id)
	}
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(headers))
}

func validRequestID(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_') {
			return false
		}
	}
	return true
}
