package rest

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Metrics("127.0.0.1")(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/metrics")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(b), "cmdline"))
	assert.True(t, strings.Contains(string(b), "memstats"))
}

func TestMetricsRejected(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
