package logger

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	lb := &mockLgr{}
	l := New(Prefix("[INFO] REST"), Flags(All),
		Log(lb),
		IPfn(func(ip string) string {
			return ip + "!masked"
		}),
		UserFn(func(r *http.Request) (string, error) {
			return "user", nil
		}),
	)

	ts := httptest.NewServer(l.Handler(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))

	s := lb.buf.String()
	t.Log(s)
	assert.True(t, strings.Contains(s, "[INFO] REST GET - /blah - 127.0.0.1!masked - 200 (9) -"), s)
	assert.True(t, strings.Contains(s, " - user"), s)
}

func TestLoggerMaxBodySize(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	lb := &mockLgr{}
	l := New(Prefix("[INFO] REST"), Flags(All), Log(lb), MaxBodySize(10))

	ts := httptest.NewServer(l.Handler(handler))
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/blah", "", bytes.NewBufferString("1234567890 abcdefg"))
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))

	s := lb.buf.String()
	t.Log(s)
	assert.True(t, strings.Contains(s, "[INFO] REST POST - /blah - 127.0.0.1 - 200 (9) -"), s)
	assert.True(t, strings.Contains(s, "1234567890..."), s)
}

func TestLoggerDefault(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	ts := httptest.NewServer(Logger(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))
}

func TestLoggerNone(t *testing.T) {
	lb := &mockLgr{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	l := New(Prefix("[INFO] REST"), Flags(None), Log(lb))
	ts := httptest.NewServer(l.Handler(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))
	assert.Equal(t, "", lb.buf.String())
}

type mockLgr struct {
	buf bytes.Buffer
}

func (m *mockLgr) Logf(format string, args ...interface{}) {
	m.buf.WriteString(fmt.Sprintf(format, args...))
}

func TestGetBodyAndUser(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/request", strings.NewReader("body"))
	require.Nil(t, err)
	l := New()

	body, user := l.getBodyAndUser(req)
	assert.Equal(t, "body", body)
	assert.Equal(t, "", user, "no user")

	l = New(Flags(User, Body), UserFn(func(r *http.Request) (string, error) {
		return "user1/id1", nil
	}))
	_, user = l.getBodyAndUser(req)
	assert.Equal(t, ` - user1/id1`, user, "no user")

	l = New(UserFn(func(r *http.Request) (string, error) {
		return "", errors.New("err")
	}))
	body, user = l.getBodyAndUser(req)
	assert.Equal(t, "body", body)
	assert.Equal(t, "", user, "no user")

	l = New(Flags(User))
	body, user = l.getBodyAndUser(req)
	assert.Equal(t, "", body)
	assert.Equal(t, "", user, "no user")
}

func TestSanitizeReqURL(t *testing.T) {
	tbl := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"/aa/bb?xyz=123", "/aa/bb?xyz=123"},
		{"/aa/bb?xyz=123&secret=asdfghjk", "/aa/bb?xyz=123&secret=********"},
		{"/aa/bb?xyz=123&secret=asdfghjk&key=val", "/aa/bb?xyz=123&secret=********&key=val"},
		{"/aa/bb?xyz=123&secret=asdfghjk&key=val&password=1234", "/aa/bb?xyz=123&secret=********&key=val&password=****"},
	}
	l := New()
	for i, tt := range tbl {
		assert.Equal(t, tt.out, l.sanitizeQuery(tt.in), "check #%d, %s", i, tt.in)
	}
}
