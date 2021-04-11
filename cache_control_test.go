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
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			h := CacheControl(tt.exp, tt.version)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			h.ServeHTTP(w, req)
			resp := w.Result()
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
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			req.Header.Set("key", tt.version)
			w := httptest.NewRecorder()

			fn := func(r *http.Request) string {
				return r.Header.Get("key")
			}
			h := CacheControlDynamic(tt.exp, fn)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			h.ServeHTTP(w, req)
			resp := w.Result()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			t.Logf("%+v", resp.Header)
			assert.Equal(t, `"`+tt.etag+`"`, resp.Header.Get("Etag"))
			assert.Equal(t, `max-age=`+strconv.Itoa(int(tt.exp.Seconds()))+", no-cache", resp.Header.Get("Cache-Control"))

		})
	}
}
