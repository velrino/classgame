package app

import (
	"net/http/httptest"
	"testing"

	"classgame-api/internal/config"
)

func TestProductionServerExtractsClientIPBehindTrustedProxy(t *testing.T) {
	server := New(Options{Config: config.Config{AppEnv: "production", TrustedProxyCIDRs: "10.0.0.4/31"}})
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.0.0.5:4242"
	request.Header.Set("X-Forwarded-For", "198.51.100.99, 203.0.113.10, 10.0.0.4")
	ctx := server.NewContext(request, httptest.NewRecorder())

	if got := ctx.RealIP(); got != "203.0.113.10" {
		t.Fatalf("RealIP() = %q, want %q", got, "203.0.113.10")
	}
}

func TestDevelopmentServerIgnoresForwardedIP(t *testing.T) {
	server := New(Options{Config: config.Config{AppEnv: "development"}})
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "192.0.2.1:4242"
	request.Header.Set("X-Forwarded-For", "203.0.113.10")
	ctx := server.NewContext(request, httptest.NewRecorder())

	if got := ctx.RealIP(); got != "192.0.2.1" {
		t.Fatalf("RealIP() = %q, want %q", got, "192.0.2.1")
	}
}
