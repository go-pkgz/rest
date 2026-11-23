package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-pkgz/rest/logger"
)

func TestTraceNoID(t *testing.T) {

	ts := httptest.NewServer(Trace(getTestHandlerBlah()))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/something")
	assert.NoError(t, err)
	if res != nil {
		defer func() { _ = res.Body.Close() }()
	}

	b, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "blah", string(b))

	traceHeader := res.Header.Get("X-Request-ID")
	t.Logf("headers - %+v", res.Header)
	assert.True(t, traceHeader != "", "non-empty header")
}

func TestTraceWithID(t *testing.T) {

	handler := func() http.HandlerFunc {
		fn := func(rw http.ResponseWriter, req *http.Request) {
			traceID := GetTraceID(req)
			assert.Equal(t, "123456", traceID)
			_, _ = rw.Write([]byte("blah"))
		}
		return fn
	}

	ts := httptest.NewServer(Trace(handler()))
	defer ts.Close()

	client := http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
	assert.NoError(t, err)
	req.Header.Add("X-Request-Id", "123456")
	res, err := client.Do(req)
	assert.NoError(t, err)
	if res != nil {
		defer func() { _ = res.Body.Close() }()
	}

	b, err := io.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "blah", string(b))

	traceHeader := res.Header.Get("X-Request-ID")
	t.Logf("headers - %+v", res.Header)
	assert.Equal(t, "123456", traceHeader, "passing original trace-id")
}

func TestGetTraceID_NoContext(t *testing.T) {
	// test GetTraceID when called without Trace middleware (no trace ID in context)
	req, err := http.NewRequest("GET", "/test", http.NoBody)
	assert.NoError(t, err)
	traceID := GetTraceID(req)
	assert.Empty(t, traceID, "trace ID should be empty without Trace middleware")
}

func TestTraceIDLogged(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	tbl := []struct {
		name  string
		build func(l *mockLgr) http.Handler
	}{
		{name: "trace outside logger", build: func(l *mockLgr) http.Handler {
			return Wrap(handler, Trace, logger.New(logger.Log(l)).Handler)
		}},
		{name: "trace inside logger", build: func(l *mockLgr) http.Handler {
			return Wrap(handler, logger.New(logger.Log(l)).Handler, Trace)
		}},
		{name: "timeout between trace and logger", build: func(l *mockLgr) http.Handler {
			// Timeout hands the inner chain its own header map, so the logger can't see Trace's response header
			return Wrap(handler, Trace, Timeout(time.Minute), logger.New(logger.Log(l)).Handler)
		}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			l := &mockLgr{}
			rr := httptest.NewRecorder()
			tt.build(l).ServeHTTP(rr, httptest.NewRequest("GET", "/something", http.NoBody))
			id := rr.Header().Get("X-Request-ID")
			require.NotEmpty(t, id)
			assert.Contains(t, l.buf.String(), " - "+id, "generated request id is not in the log line")
		})
	}
}
