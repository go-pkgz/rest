package rest

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-pkgz/rest/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileServer(t *testing.T) {
	fh, err := FileServer("/static", "./testdata/root")
	require.NoError(t, err)
	ts := httptest.NewServer(logger.Logger(fh))
	defer ts.Close()
	client := http.Client{Timeout: 599 * time.Second}

	tbl := []struct {
		req    string
		body   string
		status int
	}{
		{"/static", "testdata/index.html", 200},
		{"/static/index.html", "testdata/index.html", 200},
		{"/static/xyz.js", "testdata/xyz.js", 200},
		{"/static/1/", "", 404},
		{"/static/1/nothing", "", 404},
		{"/static/1/f1.html", "testdata/1/f1.html", 200},
		{"/static/2/", "testdata/2/index.html", 200},
		{"/static/2", "testdata/2/index.html", 200},
		{"/static/2/index.html", "testdata/2/index.html", 200},
		{"/static/2/index", "", 404},
		{"/static/2/f123.txt", "testdata/2/f123.txt", 200},
		{"/static/1/../", "testdata/index.html", 200},
		{"/static/../", "testdata/index.html", 200},
		{"/static/../../", "testdata/index.html", 200},
		{"/static/../../../", "testdata/index.html", 200},
	}

	for i, tt := range tbl {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			req, err := http.NewRequest("GET", ts.URL+tt.req, nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tt.status, resp.StatusCode)
			if resp.StatusCode != http.StatusOK {
				return
			}
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.body, string(body))

		})
	}
}
