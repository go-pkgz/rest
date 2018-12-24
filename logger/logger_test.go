package logger

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_Logger(t *testing.T) {
	buf := bytes.Buffer{}
	log.SetOutput(&buf)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	l := New(Prefix("[INFO] REST"), Flags(All),
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

	s := buf.String()
	t.Log(s)
	assert.True(t, strings.Contains(s, "[INFO] REST GET - /blah - 127.0.0.1!masked - 200 (9) -"))
	assert.True(t, strings.Contains(s, " - user"))
}

func TestMiddleware_LoggerNone(t *testing.T) {
	buf := bytes.Buffer{}
	log.SetOutput(&buf)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("blah blah"))
		require.NoError(t, err)
	})

	l := New(Prefix("[INFO] REST"), Flags(None))
	ts := httptest.NewServer(l.Handler(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/blah")
	require.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(b))
	assert.Equal(t, "", buf.String())
}

func TestMiddleware_GetBodyAndUser(t *testing.T) {
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
