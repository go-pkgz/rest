package rest

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRest_cacheControl(t *testing.T) {

	tbl := []struct {
		url     string
		version string
		exp     time.Duration
		etag    string
		maxAge  int
	}{
		{"http://example.com/foo", "v1", time.Hour, "b433be1ea19edaee9dc92ca4b895b6bdf3c058cb", 3600},
		{"http://example.com/foo2", "v1", 10 * time.Hour, "6d8466aef3246c1057452561acddf7ad9d0d99e0", 36000},
		{"http://example.com/foo", "v2", time.Hour, "481700c52aab0dfbca99f3ffc2a4fbb27884c114", 3600},
		{"https://example.com/foo", "v2", time.Hour, "bebd4f1b87f474792c4e75e5affe31fbf67f5778", 3600},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, http.NoBody)
			w := httptest.NewRecorder()

			h := CacheControl(tt.exp, tt.version)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
			h.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			t.Logf("%+v", resp.Header)
			assert.Equal(t, `"`+tt.etag+`"`, resp.Header.Get("Etag"))
			assert.Equal(t, `max-age=`+strconv.Itoa(int(tt.exp.Seconds()))+", no-cache", resp.Header.Get("Cache-Control"))
		})
	}

}

func TestCacheControlDynamic(t *testing.T) {
	tbl := []struct {
		url     string
		version string
		exp     time.Duration
		etag    string
		maxAge  int
	}{
		{"http://example.com/foo", "v1", time.Hour, "b433be1ea19edaee9dc92ca4b895b6bdf3c058cb", 3600},
		{"http://example.com/foo2", "v1", 10 * time.Hour, "6d8466aef3246c1057452561acddf7ad9d0d99e0", 36000},
		{"http://example.com/foo", "v2", time.Hour, "481700c52aab0dfbca99f3ffc2a4fbb27884c114", 3600},
		{"https://example.com/foo", "v2", time.Hour, "bebd4f1b87f474792c4e75e5affe31fbf67f5778", 3600},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, http.NoBody)
			req.Header.Set("key", tt.version)
			w := httptest.NewRecorder()

			fn := func(r *http.Request) string {
				return r.Header.Get("key")
			}
			h := CacheControlDynamic(tt.exp, fn)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
			h.ServeHTTP(w, req)
			resp := w.Result()
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			t.Logf("%+v", resp.Header)
			assert.Equal(t, `"`+tt.etag+`"`, resp.Header.Get("Etag"))
			assert.Equal(t, `max-age=`+strconv.Itoa(int(tt.exp.Seconds()))+", no-cache", resp.Header.Get("Cache-Control"))
		})
	}
}

func TestCacheControl_IfNoneMatch(t *testing.T) {
	t.Run("matching etag returns 304", func(t *testing.T) {
		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			_, _ = w.Write([]byte("response body"))
		})
		req := httptest.NewRequest("GET", "http://example.com/foo", http.NoBody)
		// pre-computed etag for http://example.com/foo with version v1
		req.Header.Set("If-None-Match", `"b433be1ea19edaee9dc92ca4b895b6bdf3c058cb"`)
		w := httptest.NewRecorder()

		h := CacheControl(time.Hour, "v1")(handler)
		h.ServeHTTP(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotModified, resp.StatusCode)
		assert.False(t, handlerCalled, "handler should not be called on 304")
	})

	t.Run("non-matching etag returns 200", func(t *testing.T) {
		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			_, _ = w.Write([]byte("response body"))
		})
		req := httptest.NewRequest("GET", "http://example.com/foo", http.NoBody)
		req.Header.Set("If-None-Match", `"wrong-etag"`)
		w := httptest.NewRecorder()

		h := CacheControl(time.Hour, "v1")(handler)
		h.ServeHTTP(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, handlerCalled, "handler should be called on cache miss")
	})

	t.Run("no if-none-match returns 200", func(t *testing.T) {
		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			_, _ = w.Write([]byte("response body"))
		})
		req := httptest.NewRequest("GET", "http://example.com/foo", http.NoBody)
		w := httptest.NewRecorder()

		h := CacheControl(time.Hour, "v1")(handler)
		h.ServeHTTP(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, handlerCalled, "handler should be called without If-None-Match")
	})
}

func TestCacheControlDynamic_IfNoneMatch(t *testing.T) {
	versionFn := func(r *http.Request) string {
		return r.Header.Get("key")
	}

	t.Run("matching etag returns 304", func(t *testing.T) {
		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			_, _ = w.Write([]byte("response body"))
		})
		req := httptest.NewRequest("GET", "http://example.com/foo", http.NoBody)
		req.Header.Set("key", "v1")
		req.Header.Set("If-None-Match", `"b433be1ea19edaee9dc92ca4b895b6bdf3c058cb"`)
		w := httptest.NewRecorder()

		h := CacheControlDynamic(time.Hour, versionFn)(handler)
		h.ServeHTTP(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotModified, resp.StatusCode)
		assert.False(t, handlerCalled, "handler should not be called on 304")
	})

	t.Run("non-matching etag returns 200", func(t *testing.T) {
		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			handlerCalled = true
			_, _ = w.Write([]byte("response body"))
		})
		req := httptest.NewRequest("GET", "http://example.com/foo", http.NoBody)
		req.Header.Set("key", "v1")
		req.Header.Set("If-None-Match", `"wrong-etag"`)
		w := httptest.NewRecorder()

		h := CacheControlDynamic(time.Hour, versionFn)(handler)
		h.ServeHTTP(w, req)
		resp := w.Result()
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, handlerCalled, "handler should be called on cache miss")
	})
}
