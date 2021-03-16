package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewrite(t *testing.T) {

	rr := httptest.NewRecorder()
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%+v", r)
		assert.Equal(t, "/xyzzz/params?foo=bar", r.URL.String())
		assert.Equal(t, "/xyzzz/params", r.URL.Path)
		assert.Equal(t, "/api/v1/params", r.Header.Get("X-Original-URL"))
	})

	handler := Rewrite("/api/v1/(.*)", "/xyzzz/$1?foo=bar")(testHandler)
	req, err := http.NewRequest("GET", "/api/v1/params", nil)
	require.NoError(t, err)
	handler.ServeHTTP(rr, req)
}

func TestRewriteCleanup(t *testing.T) {

	rr := httptest.NewRecorder()
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%+v", r)
		assert.Equal(t, "/xyzzz/params?foo=bar", r.URL.String())
		assert.Equal(t, "/xyzzz/params", r.URL.Path)
		assert.Equal(t, "/api/v1/params", r.Header.Get("X-Original-URL"))
	})

	handler := Rewrite("/api/v1/(.*)", "/xyzzz/abc/../$1?foo=bar")(testHandler)
	req, err := http.NewRequest("GET", "/api/v1/params", nil)
	require.NoError(t, err)
	handler.ServeHTTP(rr, req)
}
