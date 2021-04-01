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

	ts := httptest.NewServer(mw(getTestHandlerBlah()))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	{
		req, err := http.NewRequest("GET", u, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("GET", u, nil)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "good")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("GET", u, nil)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "bad")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
}
