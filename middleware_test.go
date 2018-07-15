package rest

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_AppInfo(t *testing.T) {
	router := chi.NewRouter()
	router.With(AppInfo("app-name", "Umputun", "12345")).Get("/blah", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("blah blah"))
	})
	ts := httptest.NewServer(router)
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
	assert.Equal(t, "Umputun", resp.Header.Get("Org"))
}

func TestMiddleware_Ping(t *testing.T) {
	router := chi.NewRouter()
	router.Use(Ping)
	router.Get("/blah", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("blah blah"))
	})
	ts := httptest.NewServer(router)
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

type lockedBuf struct {
	buf  bytes.Buffer
	lock sync.Mutex
}

func (b *lockedBuf) Write(p []byte) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuf) Read(p []byte) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.buf.Read(p)
}

func (b *lockedBuf) String() string {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.buf.String()
}

func TestMiddleware_Recoverer(t *testing.T) {
	buf := lockedBuf{}
	log.SetOutput(&buf)

	router := chi.NewRouter()
	router.Use(Recoverer)
	router.Get("/failed", func(w http.ResponseWriter, r *http.Request) {
		panic("oh my!")
	})
	router.Get("/blah", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("blah blah"))
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/failed")
	s := buf.String()
	t.Log("->> ", s)
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	assert.Contains(t, s, "[WARN] request panic, oh my!")
	assert.Contains(t, s, "goroutine")
	assert.Contains(t, s, "github.com/go-pkgz/rest.TestMiddleware_Recoverer")

	resp, err = http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))
}

func TestMiddleware_Logger(t *testing.T) {
	buf := bytes.Buffer{}
	log.SetOutput(&buf)

	router := chi.NewRouter()
	router.Use(Logger("[INFO] REST", func(ip string) string {
		return ip + "!masked"
	}, LogAll))
	router.Get("/blah", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("blah blah"))
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))

	s := buf.String()
	t.Log(s)
	assert.True(t, strings.Contains(s, "[INFO] REST GET - /blah - 127.0.0.1!masked - 200 (9) -"))
}

func TestMiddleware_GetBodyAndUser(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/request", strings.NewReader("body"))
	require.Nil(t, err)

	body, user := getBodyAndUser(req, []LoggerFlag{LogAll})
	assert.Equal(t, "body", body)
	assert.Equal(t, "", user, "no user")

	req = SetUserInfo(req, uinfo{id: "id1", name: "user1"})
	_, user = getBodyAndUser(req, []LoggerFlag{LogAll})
	assert.Equal(t, ` - user1/id1`, user, "no user")

	body, user = getBodyAndUser(req, nil)
	assert.Equal(t, "", body)
	assert.Equal(t, "", user, "no user")

	body, user = getBodyAndUser(req, []LoggerFlag{LogNone})
	assert.Equal(t, "", body)
	assert.Equal(t, "", user, "no user")

	body, user = getBodyAndUser(req, []LoggerFlag{LogUser})
	assert.Equal(t, "", body)
	assert.Equal(t, ` - user1/id1`, user, "no user")
}
