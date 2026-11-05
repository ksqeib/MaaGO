package updater

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"MaaGO/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

func response(status int, remaining string) *http.Response {
	header := http.Header{}
	if remaining != "" {
		header.Set("X-RateLimit-Remaining", remaining)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       &trackedBody{Reader: strings.NewReader("body")},
	}
}

func TestIsRateLimited(t *testing.T) {
	if !isRateLimited(response(http.StatusForbidden, "0")) {
		t.Fatal("403 with no remaining quota must be a rate limit")
	}
	if isRateLimited(response(http.StatusForbidden, "12")) {
		t.Fatal("403 with remaining quota is not a rate limit")
	}
	if !isRateLimited(response(http.StatusTooManyRequests, "")) {
		t.Fatal("429 must be treated as rate limited")
	}
	if isRateLimited(response(http.StatusOK, "0")) {
		t.Fatal("200 is not rate limited")
	}
	if isRateLimited(nil) {
		t.Fatal("nil response is not rate limited")
	}
}

func TestDirectFallbackRetriesRateLimitedRequest(t *testing.T) {
	primaryBody := &trackedBody{Reader: strings.NewReader("limited")}
	primary := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"X-Ratelimit-Remaining": []string{"0"}},
			Body:       primaryBody,
		}, nil
	})
	directCalls := 0
	direct := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		directCalls++
		if request.URL.Host != "api.github.com" {
			t.Fatalf("unexpected host: %s", request.URL.Host)
		}
		return response(http.StatusOK, "42"), nil
	})
	transport := &directFallbackTransport{proxied: primary, direct: direct}
	request, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/x/y", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", result.StatusCode)
	}
	if directCalls != 1 {
		t.Fatalf("direct calls = %d", directCalls)
	}
	if !primaryBody.closed {
		t.Fatal("rate-limited body must be closed before retrying")
	}
}

func TestDirectFallbackLeavesOtherErrorsAlone(t *testing.T) {
	directCalls := 0
	transport := &directFallbackTransport{
		proxied: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return response(http.StatusInternalServerError, ""), nil
		}),
		direct: roundTripFunc(func(*http.Request) (*http.Response, error) {
			directCalls++
			return response(http.StatusOK, "42"), nil
		}),
	}
	request, err := http.NewRequest(http.MethodGet, "https://api.github.com/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d", result.StatusCode)
	}
	if directCalls != 0 {
		t.Fatalf("direct calls = %d", directCalls)
	}
}

func TestDirectFallbackSkipsUnreplayableBody(t *testing.T) {
	directCalls := 0
	transport := &directFallbackTransport{
		proxied: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return response(http.StatusForbidden, "0"), nil
		}),
		direct: roundTripFunc(func(*http.Request) (*http.Response, error) {
			directCalls++
			return response(http.StatusOK, "42"), nil
		}),
	}
	request, err := http.NewRequest(http.MethodPost, "https://api.github.com/x", struct{ io.Reader }{strings.NewReader("data")})
	if err != nil {
		t.Fatal(err)
	}
	if request.GetBody != nil {
		t.Fatal("test body must not be replayable")
	}
	result, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", result.StatusCode)
	}
	if directCalls != 0 {
		t.Fatalf("direct calls = %d", directCalls)
	}
}

func TestHTTPClientWrapsConfiguredProxyWithFallback(t *testing.T) {
	cfg := config.AppConfig{}
	cfg.Proxy.Enabled = true
	cfg.Proxy.URL = "socks5://127.0.0.1:7890"
	client, err := httpClient(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := client.Transport.(*directFallbackTransport); !ok {
		t.Fatalf("transport = %T, want *directFallbackTransport", client.Transport)
	}
	plain, err := httpClientWithFallback(cfg, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := plain.Transport.(*directFallbackTransport); ok {
		t.Fatal("proxy test client must not fall back")
	}

	cfg.Proxy.Enabled = false
	direct, err := httpClient(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := direct.Transport.(*directFallbackTransport); ok {
		t.Fatal("client without proxy must not need a fallback")
	}
}

func TestAPIStatusErrorExplainsRateLimit(t *testing.T) {
	limited := response(http.StatusForbidden, "0")
	limited.Header.Set("X-RateLimit-Reset", "1793863820")
	err := apiStatusError("check update failed", limited)
	if err == nil || !strings.Contains(err.Error(), "rate limit") || !strings.Contains(err.Error(), "resets at") {
		t.Fatalf("error = %v", err)
	}
	other := apiStatusError("check update failed", response(http.StatusForbidden, "20"))
	if other == nil || other.Error() != "check update failed: HTTP 403" {
		t.Fatalf("error = %v", other)
	}
}
