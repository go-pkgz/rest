package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/stretchr/testify/assert"
)

func TestLimitHandler(t *testing.T) {

	t.Run("basic request", func(t *testing.T) {
		lmt := tollbooth.NewLimiter(1, nil)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrapped := LimitHandler(lmt)(handler)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.RemoteAddr = "127.0.0.1:12345"
		wrapped.ServeHTTP(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		lmt := tollbooth.NewLimiter(0.1, nil) // only allow one request per 10 seconds
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrapped := LimitHandler(lmt)(handler)

		// first request
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		r1.RemoteAddr = "127.0.0.1:12345"
		wrapped.ServeHTTP(w1, r1)

		// immediate second request should fail
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		r2.RemoteAddr = "127.0.0.1:12345"
		wrapped.ServeHTTP(w2, r2)

		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
		assert.Contains(t, w2.Body.String(), "maximum request limit")
	})

	t.Run("context cancelled", func(t *testing.T) {
		lmt := tollbooth.NewLimiter(1, nil)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrapped := LimitHandler(lmt)(handler)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx, cancel := context.WithCancel(r.Context())
		cancel()
		r = r.WithContext(ctx)
		wrapped.ServeHTTP(w, r)
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "Context was canceled")
	})

	t.Run("custom error handler", func(t *testing.T) {
		lmt := tollbooth.NewLimiter(0.1, nil) // only allow one request per 10 seconds
		customMsg := "custom limit reached"
		lmt.SetMessage(customMsg)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrapped := LimitHandler(lmt)(handler)

		// first request
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		r1.RemoteAddr = "127.0.0.1:12345"
		wrapped.ServeHTTP(w1, r1)

		// immediate second request should fail
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		r2.RemoteAddr = "127.0.0.1:12345"
		wrapped.ServeHTTP(w2, r2)

		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
		assert.Contains(t, w2.Body.String(), customMsg)
	})

	t.Run("default IP lookup", func(t *testing.T) {
		lmt := tollbooth.NewLimiter(0.1, nil)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrapped := LimitHandler(lmt)(handler)

		// first request
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		r1.RemoteAddr = "127.0.0.1:12345"
		wrapped.ServeHTTP(w1, r1)

		// second request should fail as default RemoteAddr will be used
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		r2.RemoteAddr = "127.0.0.1:12345"
		wrapped.ServeHTTP(w2, r2)

		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	})

	t.Run("custom IP lookup", func(t *testing.T) {
		lmt := tollbooth.NewLimiter(0.1, nil)
		lmt.SetIPLookup(limiter.IPLookup{Name: "X-Real-IP"})
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		wrapped := LimitHandler(lmt)(handler)

		// first request
		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		r1.Header.Set("X-Real-IP", "5.5.5.5")
		wrapped.ServeHTTP(w1, r1)

		// second request with same X-Real-IP should fail
		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		r2.Header.Set("X-Real-IP", "5.5.5.5")
		wrapped.ServeHTTP(w2, r2)

		assert.Equal(t, http.StatusTooManyRequests, w2.Code)

		// request with different X-Real-IP should pass
		w3 := httptest.NewRecorder()
		r3 := httptest.NewRequest(http.MethodGet, "/test", nil)
		r3.Header.Set("X-Real-IP", "6.6.6.6")
		wrapped.ServeHTTP(w3, r3)

		assert.Equal(t, http.StatusOK, w3.Code)
	})
}
