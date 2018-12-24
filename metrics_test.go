package rest

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics(t *testing.T) {

	router := chi.NewRouter()
	router.Use(Metrics("127.0.0.1"))
	router.Get("/blah", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("blah blah"))
	})
	ts := httptest.NewServer(router)
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
