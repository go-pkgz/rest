package logger

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerMinimal(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	lb := &mockLgr{}
	l := New(Prefix("[INFO] REST"), Log(lb))

	ts := httptest.NewServer(l.Handler(handler))
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/blah", "", bytes.NewBufferString("1234567890 abcdefg"))
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))

	s := lb.buf.String()
	t.Log(s)
	prefix := "[INFO] REST POST - /blah - 127.0.0.1 - 200 (9) - "
	assert.True(t, strings.HasPrefix(s, prefix), s)
	_, err = time.ParseDuration(s[len(prefix):])
	assert.NoError(t, err)
}

func TestLogger(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	lb := &mockLgr{}
	l := New(Prefix("[INFO] REST"), WithBody,
		Log(lb),
		IPfn(func(ip string) string {
			return ip + "!masked"
		}),
		WithUser(func(r *http.Request) (string, error) {
			return "user", nil
		}),
		WithSubj(func(r *http.Request) (string, error) {
			return "subj", nil
		}),
	)

	ts := httptest.NewServer(l.Handler(handler))
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/blah?password=secret&key=val&var=123", "", bytes.NewBufferString("1234567890 abcdefg"))
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))

	s := lb.buf.String()
	t.Log(s)
	assert.True(t, strings.Contains(s, "[INFO] REST POST - /blah?key=val&password=********&var=123 - 127.0.0.1!masked - 200 (9) -"), s)
	assert.True(t, strings.HasSuffix(s, "- user - subj - 1234567890 abcdefg"))
}

func TestLoggerTraceID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	lb := &mockLgr{}
	l := New(Prefix("[INFO] REST"), WithBody,
		Log(lb),
		IPfn(func(ip string) string {
			return ip + "!masked"
		}),
		WithUser(func(r *http.Request) (string, error) {
			return "user", nil
		}),
		WithSubj(func(r *http.Request) (string, error) {
			return "subj", nil
		}),
	)

	ts := httptest.NewServer(l.Handler(handler))
	defer ts.Close()

	clint := http.Client{}
	req, err := http.NewRequest("GET", ts.URL+"/blah", nil)
	require.NoError(t, err)
	req.Header.Set("X-Request-ID", "0000-reqid")
	resp, err := clint.Do(req)
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	s := lb.buf.String()
	t.Log(s)
	assert.True(t, strings.HasSuffix(s, "- user - subj - 0000-reqid"))

	req, err = http.NewRequest("POST", ts.URL+"/blah", bytes.NewBufferString("1234567890 abcdefg"))
	require.NoError(t, err)
	req.Header.Set("X-Request-ID", "11111-reqid")
	resp, err = clint.Do(req)
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	s = lb.buf.String()
	t.Log(s)
	assert.True(t, strings.HasSuffix(s, "- user - subj - 11111-reqid - 1234567890 abcdefg"))
}

func TestLoggerMaxBodySize(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	lb := &mockLgr{}
	l := New(Prefix("[INFO] REST"), WithBody, Log(lb), MaxBodySize(10))

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
	assert.Equal(t, "", body)
	assert.Equal(t, "", user, "no user")

	l = New(WithBody)
	body, user = l.getBodyAndUser(req)
	assert.Equal(t, "body", body)
	assert.Equal(t, "", user, "no user")

	l = New(WithUser(func(r *http.Request) (string, error) {
		return "user1/id1", nil
	}))
	body, user = l.getBodyAndUser(req)
	assert.Equal(t, "", body)
	assert.Equal(t, `user1/id1`, user, "no user")

	l = New(WithUser(func(r *http.Request) (string, error) {
		return "", errors.New("err")
	}))
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
		{"xyz=123", "xyz=123"},
		{"foo=bar&foo=buzz", "foo=bar&foo=buzz"},
		{"foo=%2&password=1234", "password=********"},
		{"xyz=123&seCret=asdfghjk", "seCret=********&xyz=123"},
		{"xyz=123&secret=asdfghjk&key=val", "key=val&secret=********&xyz=123"},
		{"xyz=123&secret=asdfghjk&key=val&password=1234", "key=val&password=********&secret=********&xyz=123"},
		{"xyz=тест&passwoRD=1234", "passwoRD=********&xyz=тест"},
		{"xyz=тест&password=1234&bar=buzz", "bar=buzz&password=********&xyz=тест"},
		{"xyz=тест&password=пароль&bar=buzz", "bar=buzz&password=********&xyz=тест"},
		{"xyz=тест&password=пароль&bar=buzz&q=?sss?ccc", "bar=buzz&password=********&q=?sss?ccc&xyz=тест"},
	}
	unesc := func(s string) string {
		s, _ = url.QueryUnescape(s)
		return s
	}
	for i, tt := range tbl {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.out, unesc(sanitizeQuery(tt.in)), "check #%d, %s", i, tt.in)
		})
	}
}
