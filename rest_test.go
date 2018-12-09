package rest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRest_RenderJSONFromBytes(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		RenderJSONFromBytes(w, r, []byte("some data"))
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/test")
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "some data", string(body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
}

func TestRest_RenderJSONWithHTML(t *testing.T) {
	router := chi.NewRouter()
	j1 := JSON{"key1": "val1", "key2": 2.0, "html": `<div> blah </div>`}
	router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		j := JSON{"key1": "val1", "key2": 2.0, "html": `<div> blah </div>`}
		require.Nil(t, RenderJSONWithHTML(w, r, j))
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/test")
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	j2 := JSON{}
	err = json.Unmarshal(body, &j2)
	require.NoError(t, err)

	assert.Equal(t, j1, j2)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
}

func getTestHandlerBlah() http.HandlerFunc {
	fn := func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte("blah"))
	}
	return http.HandlerFunc(fn)
}
