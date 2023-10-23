package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnlyFromAllowedIP(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(OnlyFrom("127.0.0.1")(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	b, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))
}

func TestOnlyFromAllowedHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(OnlyFrom("1.1.1.1")(handler))
	defer ts.Close()

	reqWithHeader := func(header string) (*http.Request, error) {
		req, err := http.NewRequest("GET", ts.URL+"/blah", http.NoBody)
		if err != nil {
			return nil, err
		}
		req.Header.Set(header, "1.1.1.1")
		return req, err
	}
	client := http.Client{}

	t.Run("X-Real-IP", func(t *testing.T) {
		req, err := reqWithHeader("X-Real-IP")
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("X-Forwarded-For", func(t *testing.T) {
		req, err := reqWithHeader("X-Forwarded-For")
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("X-Forwarded-For and X-Real-IP missing", func(t *testing.T) {
		req, err := reqWithHeader("blah")
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 403, resp.StatusCode)
	})
}

func TestOnlyFromAllowedCIDR(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(OnlyFrom("1.1.1.0/24")(handler))
	defer ts.Close()

	client := http.Client{}
	req, err := http.NewRequest("GET", ts.URL+"/blah", http.NoBody)
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
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(OnlyFrom("127.0.0.2")(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 403, resp.StatusCode)
}

func TestOnlyFromErrors(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		status     int
	}{
		{
			name:       "Invalid RemoteAddr",
			remoteAddr: "bad-addr",
			status:     http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.RemoteAddr = tt.remoteAddr
				OnlyFrom("1.1.1.1")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, err := w.Write([]byte("blah blah"))
					require.NoError(t, err)
				})).ServeHTTP(w, r)
			})

			ts := httptest.NewServer(handler)
			defer ts.Close()

			req, err := http.NewRequest("GET", ts.URL+"/blah", http.NoBody)
			require.NoError(t, err)

			client := http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}
