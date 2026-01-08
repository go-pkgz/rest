package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Metrics("127.0.0.1")(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metrics")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	b, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(b), "cmdline"))
	assert.True(t, strings.Contains(string(b), "memstats"))
}

func TestMetricsRejected(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Metrics("1.1.1.1")(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metrics")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 403, resp.StatusCode)
}

func TestMetricsContentType(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Metrics("1.1.1.1")(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
}

func TestMetrics_NonGetRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("handler response"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Metrics("127.0.0.1")(handler))
	defer ts.Close()

	// POST to /metrics should pass to handler
	resp, err := http.Post(ts.URL+"/metrics", "text/plain", strings.NewReader("data"))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "handler response", string(b))
}

func TestMetrics_NonMetricsPath(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("other path"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Metrics("127.0.0.1")(handler))
	defer ts.Close()

	// GET to other path should pass to handler
	resp, err := http.Get(ts.URL + "/other")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "other path", string(b))
}
