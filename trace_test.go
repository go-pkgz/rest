package rest

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTraceNoID(t *testing.T) {

	ts := httptest.NewServer(Trace(getTestHandlerBlah()))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/something")
	assert.NoError(t, err)
	if res != nil {
		defer func() { _ = res.Body.Close() }()
	}

	b, err := ioutil.ReadAll(res.Body)
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
	req, err := http.NewRequest("GET", ts.URL+"/something", nil)
	assert.NoError(t, err)
	req.Header.Add("X-Request-Id", "123456")
	res, err := client.Do(req)
	assert.NoError(t, err)
	if res != nil {
		defer func() { _ = res.Body.Close() }()
	}

	b, err := ioutil.ReadAll(res.Body)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "blah", string(b))

	traceHeader := res.Header.Get("X-Request-ID")
	t.Logf("headers - %+v", res.Header)
	assert.Equal(t, "123456", traceHeader, "passing original trace-id")
}
