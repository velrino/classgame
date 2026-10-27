# Rate limiting

Rate limiting has one technical home: `api/internal/platform/ratelimit`. Business modules choose which limits apply. API composition supplies the current values and provider connections.

```text
internal/platform/ratelimit/
  decision.go              # Decision and Limiter contract
  policy.go                # Run rules in order; stop on denial or failure
  policy_test.go           # Existing policy checks
  redis/
    fixed_window.go        # Atomic Redis counters, expiry, and hashed keys
  http/
    headers.go             # Limit and Retry-After response headers
    headers_test.go        # Existing header checks
```

The core package uses only the Go standard library. The Redis and HTTP adapters import the core; the core imports neither adapter. This keeps the dependency direction clear and avoids cycles.

- Mission and student modules own their rate-limit rules and error handling.
- `cmd/api` wires the Redis adapter to the existing connection.
- `platform/redis` owns connection setup, health, and TLS.
- The rate-limit HTTP adapter writes response headers. Shared HTTP error envelopes and authentication remain in `transport/httpx`.

This move keeps existing quota values, Redis keys, atomic script, expiry, error results, and header rounding. It adds no provider, cache, or infrastructure.

Run the existing API tests, build, and vet checks after changing these boundaries. Test a Redis outage separately before claiming that a live workflow handles it correctly.
