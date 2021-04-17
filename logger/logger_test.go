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
	prefix := "[INFO] REST POST - /blah - 127.0.0.1 - 127.0.0.1 - 200 (9) - "
	assert.True(t, strings.HasPrefix(s, prefix), s)
	_, err = time.ParseDuration(s[len(prefix):])
	assert.NoError(t, err)

}

func TestLoggerMinimalLocalhost(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	lb := &mockLgr{}
	l := New(Prefix("[INFO] REST"), Log(lb))

	ts := httptest.NewServer(l.Handler(handler))
	defer ts.Close()

	port := strings.Split(ts.URL, ":")[2]
	resp, err := http.Post("http://localhost:"+port+"/blah", "", bytes.NewBufferString("1234567890 abcdefg"))
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))

	s := lb.buf.String()
	t.Log(s)
	prefix := "[INFO] REST POST - /blah - localhost - 127.0.0.1 - 200 (9) - "
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
		UserFn(func(r *http.Request) (string, error) {
			return "user", nil
		}),
		SubjFn(func(r *http.Request) (string, error) {
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
	assert.True(t, strings.Contains(s, "[INFO] REST POST - /blah?key=val&password=********&var=123 - 127.0.0.1 - 127.0.0.1!masked - 200 (9) -"), s)
	assert.True(t, strings.HasSuffix(s, "- user - subj - 1234567890 abcdefg"))
}

func TestLoggerIP(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	lb := &mockLgr{}
	l := New(Log(lb), Prefix("[INFO] REST"), IPfn(func(ip string) string { return ip + "!masked" }))

	ts := httptest.NewServer(l.Handler(handler))
	defer ts.Close()

	clint := http.Client{}

	req, err := http.NewRequest("GET", ts.URL+"/blah", nil)
	require.NoError(t, err)
	resp, err := clint.Do(req)
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	s := lb.buf.String()
	t.Log(s)
	assert.True(t, strings.Contains(s, "- 127.0.0.1!masked -"))

	lb.buf.Reset()
	req, err = http.NewRequest("GET", ts.URL+"/blah", nil)
	require.NoError(t, err)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	resp, err = clint.Do(req)
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	s = lb.buf.String()
	t.Log(s)
	assert.True(t, strings.Contains(s, "- 1.2.3.4!masked -"))
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
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "1234567890 abcdefg", string(body))
		_, err = w.Write([]byte("blah blah"))
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
	assert.True(t, strings.Contains(s, "[INFO] REST POST - /blah - 127.0.0.1 - 127.0.0.1 - 200 (9) -"), s)
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
	_, _ = m.buf.WriteString(fmt.Sprintf(format, args...))
}

func TestGetBody(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com/request", strings.NewReader("body"))
	require.Nil(t, err)

	l := New()
	body := l.getBody(req)
	assert.Equal(t, "", body)

	l = New(WithBody)
	body = l.getBody(req)
	assert.Equal(t, "body", body)
}

func TestPeek(t *testing.T) {
	cases := []struct {
		body    string
		n       int64
		excerpt string
		hasMore bool
	}{
		{"", -1, "", false},
		{"", 0, "", false},
		{"", 1024, "", false},
		{"123456", -1, "", true},
		{"123456", 0, "", true},
		{"123456", 4, "1234", true},
		{"123456", 5, "12345", true},
		{"123456", 6, "123456", false},
		{"123456", 7, "123456", false},
	}

	for _, c := range cases {
		r, excerpt, hasMore, err := peek(strings.NewReader(c.body), c.n)
		if !assert.NoError(t, err) {
			continue
		}
		body, err := ioutil.ReadAll(r)
		if !assert.NoError(t, err) {
			continue
		}
		assert.Equal(t, c.body, string(body))
		assert.Equal(t, c.excerpt, excerpt)
		assert.Equal(t, c.hasMore, hasMore)
	}

	_, _, _, err := peek(errReader{}, 1024)
	assert.Error(t, err)
}

type errReader struct{}

func (errReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("test error")
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
	var l *Middleware
	for i, tt := range tbl {
		i := i
		tt := tt
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.out, unesc(l.sanitizeQuery(tt.in)), "check #%d, %s", i, tt.in)
		})
	}
}

func TestLoggerApacheCombined(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	lb := &mockLgr{}
	l := New(Log(lb), ApacheCombined,
		IPfn(func(ip string) string {
			return ip + "!masked"
		}),
		UserFn(func(r *http.Request) (string, error) {
			return "user", nil
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
	assert.True(t, strings.HasPrefix(s, "127.0.0.1!masked - user ["))
	assert.True(t, strings.HasSuffix(s, ` "POST /blah?key=val&password=********&var=123" HTTP/1.1" 200 9 "" "Go-http-client/1.1"`), s)
}
