package rest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_AppInfo(t *testing.T) {
	err := os.Setenv("MHOST", "host1")
	assert.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(AppInfo("app-name", "Umputun", "12345")(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, "blah blah", string(b))
	assert.Equal(t, "app-name", resp.Header.Get("App-Name"))
	assert.Equal(t, "12345", resp.Header.Get("App-Version"))
	assert.Equal(t, "Umputun", resp.Header.Get("Author"))
	assert.Equal(t, "host1", resp.Header.Get("Host"))
}

func TestMiddleware_Ping(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Ping(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/ping")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(b))

	resp, err = http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))
}

func TestMiddleware_Recoverer(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/failed" {
			panic("oh my!")
		}
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})
	l := &mockLgr{}
	ts := httptest.NewServer(Recoverer(l)(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/failed")
	s := l.buf.String()
	t.Log("->> ", s)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	assert.Contains(t, s, "request panic, oh my!")
	assert.Contains(t, s, "goroutine")
	assert.Contains(t, s, "github.com/go-pkgz/rest.TestMiddleware_Recoverer")

	resp, err = http.Get(ts.URL + "/blah")
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))
}

func TestWrap(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%s", r.URL.String())
		assert.Equal(t, "/something/1/2", r.URL.Path)
	})

	mw1 := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path += "/1"
			h.ServeHTTP(w, r)
		})
	}
	mw2 := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path += "/2"
			h.ServeHTTP(w, r)
		})
	}

	h := Wrap(handler, mw1, mw2)
	ts := httptest.NewServer(h)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/something")
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/something", nil)
	w := httptest.NewRecorder()

	h := Headers("h1:v1", "bad", "h2:v2")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	h.ServeHTTP(w, req)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	t.Logf("%+v", req.Header)
	assert.Equal(t, "v1", req.Header.Get("h1"))
	assert.Equal(t, "v2", req.Header.Get("h2"))
	assert.Equal(t, 2, len(req.Header))
}

type mockLgr struct {
	buf bytes.Buffer
}

func (m *mockLgr) Logf(format string, args ...interface{}) {
	_, _ = m.buf.WriteString(fmt.Sprintf(format+"\n", args...))
}
