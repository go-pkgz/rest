package rest

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlackwords(t *testing.T) {

	tbl := []struct {
		inp  string
		code int
	}{
		{"", 200},
		{"blah blah body", 200},
		{"blah blah body bad1", 403},
		{"blah blah body bad", 200},
		{"blah bad2 body bad", 403},
		{`{"fld": 123, "aa": {"$where": {"aaa": 567}}}`, 403},
	}

	bwMiddleware := BlackWords("bad1", "bad2", "$where")
	ts := httptest.NewServer(bwMiddleware(getTestHandlerBlah()))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	for n, tt := range tbl {
		t.Run(fmt.Sprintf("test-%d", n), func(t *testing.T) {
			req, err := http.NewRequest("GET", u, bytes.NewBuffer([]byte(tt.inp)))
			assert.Nil(t, err)

			r, err := client.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, tt.code, r.StatusCode)
		})
	}
}

func TestBlackwordsFn(t *testing.T) {
	tbl := []struct {
		inp  string
		code int
	}{
		{"", 200},
		{"blah blah body", 200},
		{"blah blah body bad1", 403},
		{"blah blah body bad", 200},
		{"blah bad2 body bad", 403},
		{`{"fld": 123, "aa": {"$where": {"aaa": 567}}}`, 403},
	}

	bwMiddleware := BlackWordsFn(func() []string {
		return []string{"bad1", "bad2", "$where"}
	})

	ts := httptest.NewServer(bwMiddleware(getTestHandlerBlah()))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	for n, tt := range tbl {
		t.Run(fmt.Sprintf("test-%d", n), func(t *testing.T) {
			req, err := http.NewRequest("GET", u, bytes.NewBuffer([]byte(tt.inp)))
			assert.Nil(t, err)

			r, err := client.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, tt.code, r.StatusCode)
		})
	}
}

func TestBlackwordsContentType(t *testing.T) {
	bwMiddleware := BlackWords("bad1", "bad2")
	ts := httptest.NewServer(bwMiddleware(getTestHandlerBlah()))
	defer ts.Close()

	client := http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", ts.URL+"/something", bytes.NewBuffer([]byte("contains bad1 word")))
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
}

// errReader fails partway through, mimicking a truncated or aborted request body
type errReader struct {
	data []byte
	pos  int
}

func (e *errReader) Read(p []byte) (int, error) {
	if e.pos >= len(e.data) {
		return 0, errors.New("read failed")
	}
	n := copy(p, e.data[e.pos:])
	e.pos += n
	return n, nil
}

func TestBlackwordsUnreadableBody(t *testing.T) {
	var called bool
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true })

	req := httptest.NewRequest("POST", "/", &errReader{data: []byte("badword and more")})
	w := httptest.NewRecorder()
	BlackWords("badword")(handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, called, "handler must not run on a body that can't be inspected")
}

func TestBlackwordsFnDynamic(t *testing.T) {
	var mu sync.Mutex
	words := []string{"bad1"}
	bwMiddleware := BlackWordsFn(func() []string {
		mu.Lock()
		defer mu.Unlock()
		return words
	})

	ts := httptest.NewServer(bwMiddleware(getTestHandlerBlah()))
	defer ts.Close()

	send := func(body string) int {
		req, err := http.NewRequest("POST", ts.URL+"/something", bytes.NewBufferString(body))
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		return resp.StatusCode
	}

	assert.Equal(t, http.StatusForbidden, send("has bad1 in it"))
	assert.Equal(t, http.StatusOK, send("has bad2 in it"))

	// the list is changed after the middleware was created, it should be picked up by the next request
	mu.Lock()
	words = []string{"bad2"}
	mu.Unlock()

	assert.Equal(t, http.StatusOK, send("has bad1 in it"))
	assert.Equal(t, http.StatusForbidden, send("has bad2 in it"))
}
