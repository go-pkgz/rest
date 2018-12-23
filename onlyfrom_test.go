package rest

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnlyFromAllowed(t *testing.T) {

	router := chi.NewRouter()

	router.With(OnlyFrom("127.0.0.1")).Get("/blah", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("blah blah"))
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))
}

func TestOnlyFromAllowedHeaders(t *testing.T) {

	router := chi.NewRouter()
	router.With(OnlyFrom("1.1.1.1")).Get("/blah", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("blah blah"))
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	reqWithHeader := func(header string) (*http.Request, error) {
		req, err := http.NewRequest("GET", ts.URL+"/blah", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set(header, "1.1.1.1")
		return req, err
	}
	client := http.Client{}

	req, err := reqWithHeader("X-Real-IP")
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	req, err = reqWithHeader("X-Forwarded-For")
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	req, err = reqWithHeader("RemoteAddr")
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
}

func TestOnlyFromAllowedCIDR(t *testing.T) {

	router := chi.NewRouter()
	router.With(OnlyFrom("1.1.1.0/24")).Get("/blah", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("blah blah"))
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	client := http.Client{}
	req, err := http.NewRequest("GET", ts.URL+"/blah", nil)
	require.NoError(t, err)
	req.Header.Set("X-Real-IP", "1.1.1.1")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	req.Header.Set("X-Real-IP", "1.1.2.0")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 403, resp.StatusCode)
}

func TestOnlyFromRejected(t *testing.T) {

	router := chi.NewRouter()

	router.With(OnlyFrom("127.0.0.2")).Get("/blah", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("blah blah"))
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 403, resp.StatusCode)
}
