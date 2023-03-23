package rest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicAuth(t *testing.T) {

	mw := BasicAuth(func(user, passwd string) bool {
		return user == "dev" && passwd == "good"
	})

	ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %s", r.URL)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("blah"))
		require.NoError(t, err)
		assert.True(t, IsAuthorized(r.Context()))
	})))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "good")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "bad")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
}

func TestBasicAuthWithUserPasswd(t *testing.T) {
	mw := BasicAuthWithUserPasswd("dev", "good")

	ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %s", r.URL)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("blah"))
		require.NoError(t, err)
		assert.True(t, IsAuthorized(r.Context()))
	})))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "good")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "bad")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
}

func TestBasicAuthWithPrompt(t *testing.T) {
	mw := BasicAuthWithPrompt("dev", "good")

	ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %s", r.URL)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("blah"))
		require.NoError(t, err)
		assert.True(t, IsAuthorized(r.Context()))
	})))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, `Basic realm="restricted", charset="UTF-8"`, resp.Header.Get("WWW-Authenticate"))
	}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "good")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "bad")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, `Basic realm="restricted", charset="UTF-8"`, resp.Header.Get("WWW-Authenticate"))
	}
}
