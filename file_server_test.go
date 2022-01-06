package rest

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-pkgz/rest/logger"
)

func TestFileServerDefault(t *testing.T) {
	fh1, err := NewFileServer("/static", "./testdata/root")
	require.NoError(t, err)

	fh2, err := FileServer("/static", "./testdata/root", nil)
	require.NoError(t, err)

	ts1 := httptest.NewServer(logger.Logger(fh1))
	defer ts1.Close()
	ts2 := httptest.NewServer(logger.Logger(fh2))
	defer ts2.Close()

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
		{"/static/%2e%2e%2f%2e%2e%2f%2e%2e%2f/", "testdata/index.html", 200},
	}

	for i, tt := range tbl {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			for _, ts := range []*httptest.Server{ts1, ts2} {
				req, err := http.NewRequest("GET", ts.URL+tt.req, http.NoBody)
				require.NoError(t, err)
				resp, err := client.Do(req)
				require.NoError(t, err)
				t.Logf("headers: %v", resp.Header)
				assert.Equal(t, tt.status, resp.StatusCode)
				if resp.StatusCode == http.StatusNotFound {
					msg, e := io.ReadAll(resp.Body)
					require.NoError(t, e)
					assert.Equal(t, "404 page not found\n", string(msg))
					return
				}
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.body, string(body))
			}
		})
	}
}

func TestFileServerWithListing(t *testing.T) {
	fh, err := NewFileServer("/static", "./testdata/root", FsOptListing)
	require.NoError(t, err)
	ts := httptest.NewServer(logger.Logger(fh))
	defer ts.Close()
	client := http.Client{Timeout: 599 * time.Second}

	{
		req, err := http.NewRequest("GET", ts.URL+"/static/1", http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		msg, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		exp := `<pre>
<a href="f1.html">f1.html</a>
<a href="f2.html">f2.html</a>
</pre>
`
		assert.Equal(t, exp, string(msg))
	}

	{
		req, err := http.NewRequest("GET", ts.URL+"/static/xyz.js", http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		msg, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "testdata/xyz.js", string(msg))
	}

	{
		req, err := http.NewRequest("GET", ts.URL+"/static/no-such-thing.html", http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	}
}

func TestFileServer_Custom404(t *testing.T) {
	nf := FsOptCustom404(bytes.NewBufferString("custom 404"))
	fh, err := NewFileServer("/static", "./testdata/root", nf)
	require.NoError(t, err)
	ts := httptest.NewServer(logger.Logger(fh))
	defer ts.Close()
	client := http.Client{Timeout: 599 * time.Second}

	{
		req, err := http.NewRequest("GET", ts.URL+"/static/xyz.js", http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		msg, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "testdata/xyz.js", string(msg))
	}

	{
		req, err := http.NewRequest("GET", ts.URL+"/static/nofile.js", http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		msg, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "custom 404", string(msg))
		assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	}

	{
		req, err := http.NewRequest("GET", ts.URL+"/xyz.html", http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		msg, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "custom 404", string(msg))
		assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	}

	{
		req, err := http.NewRequest("GET", ts.URL+"/static/xyz.js", http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		msg, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "testdata/xyz.js", string(msg))
	}
}

func TestFileServerSPA(t *testing.T) {
	fh1, err := NewFileServer("/static", "./testdata/root", FsOptSPA)
	require.NoError(t, err)
	fh2, err := FileServerSPA("/static", "./testdata/root", nil)
	require.NoError(t, err)

	ts1 := httptest.NewServer(logger.Logger(fh1))
	defer ts1.Close()
	ts2 := httptest.NewServer(logger.Logger(fh2))
	defer ts2.Close()
	client := http.Client{Timeout: 599 * time.Second}

	tbl := []struct {
		req    string
		body   string
		status int
	}{
		{"/static/blah", "testdata/index.html", 200},
		{"/static/blah/foo/123.html", "testdata/index.html", 200},
		{"/static", "testdata/index.html", 200},
		{"/static/index.html", "testdata/index.html", 200},
		{"/static/xyz.js", "testdata/xyz.js", 200},
		{"/static/1/", "", 404},
		{"/static/1/nothing", "testdata/index.html", 200},
		{"/static/1/f1.html", "testdata/1/f1.html", 200},
		{"/static/2/", "testdata/2/index.html", 200},
		{"/static/2", "testdata/2/index.html", 200},
		{"/static/2/index.html", "testdata/2/index.html", 200},
		{"/static/2/index", "testdata/index.html", 200},
		{"/static/2/f123.txt", "testdata/2/f123.txt", 200},
		{"/static/1/../", "testdata/index.html", 200},
		{"/static/../", "testdata/index.html", 200},
		{"/static/../../", "testdata/index.html", 200},
		{"/static/../../../", "testdata/index.html", 200},
		{"/static/%2e%2e%2f%2e%2e%2f%2e%2e%2f/", "testdata/index.html", 200},
	}

	for i, tt := range tbl {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			for _, ts := range []*httptest.Server{ts1, ts2} {
				req, err := http.NewRequest("GET", ts.URL+tt.req, http.NoBody)
				require.NoError(t, err)
				resp, err := client.Do(req)
				require.NoError(t, err)
				t.Logf("headers: %v", resp.Header)
				assert.Equal(t, tt.status, resp.StatusCode)
				if resp.StatusCode == http.StatusNotFound {
					msg, e := io.ReadAll(resp.Body)
					require.NoError(t, e)
					assert.Equal(t, "404 page not found\n", string(msg))
					return
				}
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.body, string(body))
			}
		})
	}
}
