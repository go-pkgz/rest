package rest

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipCustom(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("Lorem Ipsum is simply dummy text of the printing and typesetting industry. " +
			"Lorem Ipsum has been the industry’s standard dummy text ever since the 1500s, when an unknown printer took " +
			"a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries," +
			" but also the leap into electronic typesetting, remaining essentially unchanged. It was popularized" +
			" in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, " +
			"and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum."))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Gzip("text/plain", "text/html")(handler))
	defer ts.Close()

	client := http.Client{}

	{
		req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, 357, len(b), "compressed size")

		gzr, err := gzip.NewReader(bytes.NewBuffer(b))
		require.NoError(t, err)
		b, err = io.ReadAll(gzr)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(b), "Lorem Ipsum"), string(b))
	}

	{
		req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Type", "something")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, 576, len(b), "uncompressed size")
	}

}

func TestGzipVary(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(strings.Repeat("compress me. ", 50)))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Gzip()(handler))
	defer ts.Close()

	tbl := []struct {
		name           string
		acceptEncoding string
		encoded        bool
	}{
		{"gzip accepted", "gzip", true},
		{"gzip not accepted", "", false},
		{"other encoding", "br", false},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "text/plain")
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			} else {
				req.Header.Set("Accept-Encoding", "identity")
			}

			resp, err := http.DefaultTransport.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// caches must be told the body varies by Accept-Encoding whether or not it got compressed
			assert.Contains(t, resp.Header.Values("Vary"), "Accept-Encoding")
			if tt.encoded {
				assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
				return
			}
			assert.Empty(t, resp.Header.Get("Content-Encoding"))
		})
	}
}

func TestGzipWriteHeader(t *testing.T) {
	// test that explicit WriteHeader call works with gzip middleware
	longText := strings.Repeat("This is a test message for gzip compression. ", 20)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(longText))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Gzip()(handler))
	defer ts.Close()

	client := http.Client{}
	req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "text/plain")
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	// verify it's compressed (smaller than original)
	assert.Less(t, len(b), len(longText), "response should be compressed")

	gzr, err := gzip.NewReader(bytes.NewBuffer(b))
	require.NoError(t, err)
	decompressed, err := io.ReadAll(gzr)
	require.NoError(t, err)
	assert.Equal(t, longText, string(decompressed))
}

func TestGzipDefault(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte("Lorem Ipsum is simply dummy text of the printing and typesetting industry. " +
			"Lorem Ipsum has been the industry’s standard dummy text ever since the 1500s, when an unknown printer took " +
			"a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries," +
			" but also the leap into electronic typesetting, remaining essentially unchanged. It was popularized" +
			" in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, " +
			"and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum."))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Gzip()(handler))
	defer ts.Close()

	client := http.Client{}

	{
		req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Type", "text/plain")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, 357, len(b), "compressed size")

		gzr, err := gzip.NewReader(bytes.NewBuffer(b))
		require.NoError(t, err)
		b, err = io.ReadAll(gzr)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(b), "Lorem Ipsum"), string(b))
	}

	{
		req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.Nil(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, 576, len(b), "uncompressed size")
	}

	{
		req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Type", "something")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, 576, len(b), "uncompressed size")
	}

}
