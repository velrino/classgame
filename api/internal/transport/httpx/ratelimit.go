package httpx

import (
	"net/http"
	"strconv"
	"time"

	"classgame-api/internal/ratelimit"
)

func SetRateLimitHeaders(header http.Header, decision ratelimit.Decision) {
	header.Set("X-RateLimit-Limit", strconv.FormatInt(decision.Limit, 10))
	header.Set("X-RateLimit-Remaining", strconv.FormatInt(decision.Remaining, 10))
	if decision.Allowed {
		return
	}

	retryAfterSeconds := int64((decision.RetryAfter + time.Second - 1) / time.Second)
	if retryAfterSeconds < 1 {
		retryAfterSeconds = 1
	}
	header.Set("Retry-After", strconv.FormatInt(retryAfterSeconds, 10))
}
