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
	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Logf("%+v", r)
		assert.Equal(t, "/xyzzz/params?foo=bar", r.URL.String())
		assert.Equal(t, "/xyzzz/params", r.URL.Path)
		assert.Equal(t, "/api/v1/params", r.Header.Get("X-Original-URL"))
	})

	handler := Rewrite("/api/v1/(.*)", "/xyzzz/$1?foo=bar")(testHandler)
	req, err := http.NewRequest("GET", "/api/v1/params", http.NoBody)
	require.NoError(t, err)
	handler.ServeHTTP(rr, req)
}

func TestRewriteCleanup(t *testing.T) {

	rr := httptest.NewRecorder()
	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Logf("%+v", r)
		assert.Equal(t, "/xyzzz/params?foo=bar", r.URL.String())
		assert.Equal(t, "/xyzzz/params", r.URL.Path)
		assert.Equal(t, "/api/v1/params", r.Header.Get("X-Original-URL"))
	})

	handler := Rewrite("/api/v1/(.*)", "/xyzzz/abc/../$1?foo=bar")(testHandler)
	req, err := http.NewRequest("GET", "/api/v1/params", http.NoBody)
	require.NoError(t, err)
	handler.ServeHTTP(rr, req)
}

func TestRewriteCleanupWithSlash(t *testing.T) {

	rr := httptest.NewRecorder()
	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Logf("%+v", r)
		assert.Equal(t, "/xyzzz/params/", r.URL.String())
		assert.Equal(t, "/xyzzz/params/", r.URL.Path)
		assert.Equal(t, "/api/v1/params/", r.Header.Get("X-Original-URL"))
	})

	handler := Rewrite("/api/v1/(.*)/", "/xyzzz/abc/../$1/")(testHandler)
	req, err := http.NewRequest("GET", "/api/v1/params/", http.NoBody)
	require.NoError(t, err)
	handler.ServeHTTP(rr, req)
}

func TestRewrite_NoMatch(t *testing.T) {
	handlerCalled := false
	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		assert.Equal(t, "/other/path", r.URL.Path)
		assert.Empty(t, r.Header.Get("X-Original-URL"), "X-Original-URL should not be set when no rewrite")
	})

	rr := httptest.NewRecorder()
	handler := Rewrite("/api/v1/(.*)", "/xyzzz/$1")(testHandler)
	req, err := http.NewRequest("GET", "/other/path", http.NoBody)
	require.NoError(t, err)
	handler.ServeHTTP(rr, req)
	assert.True(t, handlerCalled, "handler should be called for non-matching path")
}

func TestRewrite_DoubleRewritePrevention(t *testing.T) {
	rewriteCount := 0
	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// after two rewrites in chain, path should only be rewritten once
		assert.Equal(t, "/first/params", r.URL.Path)
	})

	// first rewrite
	firstRewrite := Rewrite("/api/(.*)", "/first/$1")
	// second rewrite that would match the result of first
	secondRewrite := Rewrite("/first/(.*)", "/second/$1")

	// wrap with counter to see if second rewrite applies
	countingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Original-URL") != "" {
			rewriteCount++
		}
		testHandler.ServeHTTP(w, r)
	})

	rr := httptest.NewRecorder()
	handler := firstRewrite(secondRewrite(countingHandler))
	req, err := http.NewRequest("GET", "/api/params", http.NoBody)
	require.NoError(t, err)
	handler.ServeHTTP(rr, req)
	// only first rewrite should be applied
	assert.Equal(t, 1, rewriteCount, "only one rewrite should be applied")
}
