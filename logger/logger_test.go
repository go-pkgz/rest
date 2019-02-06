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
		SubjFn(func(r *http.Request) (string, error) {
			return "subj", nil
		}),
	)

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
	assert.True(t, strings.Contains(s, "[INFO] REST POST - /blah - 127.0.0.1!masked - 200 (9) -"), s)
	assert.True(t, strings.HasSuffix(s, "- user - subj - 1234567890 abcdefg"))
}

func TestLoggerTraceID(t *testing.T) {
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
		SubjFn(func(r *http.Request) (string, error) {
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
	assert.Equal(t, `user1/id1`, user, "no user")

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
		{"https://aaa.example.com:9090/aa/bb", "https://aaa.example.com:9090/aa/bb"},
		{"https://aaa.example.com:9090/aa/bb?xyz=123", "https://aaa.example.com:9090/aa/bb?xyz=123"},
		{"/aa/bb?xyz=123&seCret=asdfghjk", "/aa/bb?seCret=********&xyz=123"},
		{"/aa/bb?xyz=123&secret=asdfghjk&key=val", "/aa/bb?key=val&secret=********&xyz=123"},
		{"/aa/bb?xyz=123&secret=asdfghjk&key=val&password=1234", "/aa/bb?key=val&password=********&secret=********&xyz=123"},
		{"/aa/bb?xyz=тест&passwoRD=1234", "/aa/bb?passwoRD=********&xyz=тест"},
		{"/aa/bb?xyz=тест&password=1234&bar=buzz", "/aa/bb?bar=buzz&password=********&xyz=тест"},
		{"/aa/bb?xyz=тест&password=пароль&bar=buzz", "/aa/bb?bar=buzz&password=********&xyz=тест"},
		{"http://xyz.example.com/aa/bb?xyz=тест&password=пароль&bar=buzz&q=?sss?ccc", "http://xyz.example.com/aa/bb?bar=buzz&password=********&q=?sss?ccc&xyz=тест"},
	}
	l := New()
	for i, tt := range tbl {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.out, l.sanitizeQuery(tt.in), "check #%d, %s", i, tt.in)
		})
	}
}
