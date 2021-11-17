package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRest_RenderJSON(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		j := JSON{"key1": 1, "key2": "222"}
		RenderJSON(w, j)
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/test")
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, `{"key1":1,"key2":"222"}`+"\n", string(body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
}

func TestRest_RenderJSONFromBytes(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, RenderJSONFromBytes(w, r, []byte("some data")))
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/test")
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "some data", string(body))
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
}

func TestRest_RenderJSONWithHTML(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		j := JSON{"key1": "val1", "key2": 2.0, "html": `<div> blah </div>`}
		require.NoError(t, RenderJSONWithHTML(w, r, j))
	}))
	defer ts.Close()

	j1 := JSON{"key1": "val1", "key2": 2.0, "html": `<div> blah </div>`}

	resp, err := http.Get(ts.URL + "/test")
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
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
	return fn
}
