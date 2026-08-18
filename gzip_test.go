package rest

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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
		// the request content type has no say in the decision, the response is what counts
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
		assert.Equal(t, 357, len(b), "compressed size")
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
		// the request content type has no say in the decision, the response is what counts
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
		assert.Equal(t, 357, len(b), "compressed size")
	}

}

func TestGzipResponseContentType(t *testing.T) {
	body := strings.Repeat("compress me if the response type says so. ", 40)

	tbl := []struct {
		name        string
		respType    string // set by the handler, empty means let it be sniffed
		reqType     string // request content type, must not influence the outcome
		wantEncoded bool
	}{
		{name: "json response", respType: "application/json", wantEncoded: true},
		{name: "html response", respType: "text/html; charset=utf-8", wantEncoded: true},
		{name: "octet-stream response", respType: "application/octet-stream", wantEncoded: false},
		{name: "image response", respType: "image/png", wantEncoded: false},
		{name: "sniffed as text", respType: "", wantEncoded: true},
		{name: "json response, misleading request type", respType: "application/json", reqType: "image/png", wantEncoded: true},
		{name: "binary response, misleading request type", respType: "image/png", reqType: "application/json", wantEncoded: false},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.respType != "" {
					w.Header().Set("Content-Type", tt.respType)
				}
				_, err := w.Write([]byte(body))
				require.NoError(t, err)
			})
			ts := httptest.NewServer(Gzip()(handler))
			defer ts.Close()

			req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
			require.NoError(t, err)
			req.Header.Set("Accept-Encoding", "gzip")
			if tt.reqType != "" {
				req.Header.Set("Content-Type", tt.reqType)
			}

			resp, err := http.DefaultTransport.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Contains(t, resp.Header.Values("Vary"), "Accept-Encoding")
			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if !tt.wantEncoded {
				assert.Empty(t, resp.Header.Get("Content-Encoding"))
				assert.Equal(t, body, string(raw))
				return
			}

			require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
			assert.Less(t, len(raw), len(body), "compressed payload should be smaller")
			gzr, err := gzip.NewReader(bytes.NewReader(raw))
			require.NoError(t, err)
			decoded, err := io.ReadAll(gzr)
			require.NoError(t, err)
			assert.Equal(t, body, string(decoded))
		})
	}
}

func TestGzipAcceptEncoding(t *testing.T) {
	tbl := []struct {
		header string
		want   bool
	}{
		{"gzip", true},
		{"", false},
		{"identity", false},
		{"br", false},
		{"br, gzip", true},
		{"gzip;q=0.8", true},
		{"gzip;q=0", false},
		{"gzip;q=0.0", false},
		{"*", true},
		{"*;q=0", false},
		{"deflate, gzip;q=1.0, *;q=0.5", true},
		{"GZIP", true},
	}

	for _, tt := range tbl {
		t.Run(tt.header, func(t *testing.T) {
			assert.Equal(t, tt.want, acceptsGzip(tt.header))
		})
	}
}

func TestGzipNoBodyStatuses(t *testing.T) {
	tbl := []struct {
		name   string
		status int
	}{
		{"no content", http.StatusNoContent},
		{"not modified", http.StatusNotModified},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
			})
			ts := httptest.NewServer(Gzip()(handler))
			defer ts.Close()

			req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
			require.NoError(t, err)
			req.Header.Set("Accept-Encoding", "gzip")

			resp, err := http.DefaultTransport.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Empty(t, resp.Header.Get("Content-Encoding"), "bodyless response should not be compressed")
		})
	}
}

func TestGzipStreaming(t *testing.T) {
	// a handler that flushes between chunks must keep working through the compressor
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		for i := range 3 {
			_, err := fmt.Fprintf(w, "chunk-%d\n", i)
			require.NoError(t, err)
			w.(http.Flusher).Flush()
		}
	})
	ts := httptest.NewServer(Gzip()(handler))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/stream", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
	gzr, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)
	decoded, err := io.ReadAll(gzr)
	require.NoError(t, err)
	assert.Equal(t, "chunk-0\nchunk-1\nchunk-2\n", string(decoded))
}

func TestGzipEmptyBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
	})
	ts := httptest.NewServer(Gzip()(handler))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// the status has to reach the client even though the handler wrote no body
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzr, err := gzip.NewReader(bytes.NewReader(b))
		require.NoError(t, err)
		b, err = io.ReadAll(gzr)
		require.NoError(t, err)
	}
	assert.Empty(t, b)
}

func TestGzipContentLengthDropped(t *testing.T) {
	body := strings.Repeat("x", 500)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, err := w.Write([]byte(body))
		require.NoError(t, err)
	})
	ts := httptest.NewServer(Gzip()(handler))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultTransport.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
	assert.NotEqual(t, strconv.Itoa(len(body)), resp.Header.Get("Content-Length"),
		"the uncompressed length must not survive onto a compressed body")
}

func TestGzipWriterInterfaces(t *testing.T) {
	t.Run("hijack passes through and suppresses the deferred write", func(t *testing.T) {
		done := make(chan error, 1)
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			h, ok := w.(http.Hijacker)
			if !ok {
				done <- errors.New("the wrapper must offer Hijack")
				return
			}
			conn, _, err := h.Hijack()
			if err != nil {
				done <- err
				return
			}
			// once hijacked nothing may write a status to the connection, including the deferred close
			_ = conn.Close()
			done <- nil
		})
		ts := httptest.NewServer(Gzip()(handler))
		defer ts.Close()

		req, err := http.NewRequest("GET", ts.URL+"/upgrade", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Accept-Encoding", "gzip")
		//nolint:bodyclose // the handler hijacks and closes the connection, there is no response to close
		_, err = http.DefaultTransport.RoundTrip(req)
		require.Error(t, err, "hijacked connection yields no response")
		require.NoError(t, <-done)
	})

	t.Run("hijack unsupported by the underlying writer", func(t *testing.T) {
		gw := &gzipResponseWriter{ResponseWriter: httptest.NewRecorder(), gzCts: gzDefaultContentTypes}
		_, _, err := gw.Hijack()
		assert.Error(t, err)
	})

	t.Run("unwrap returns the underlying writer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		gw := &gzipResponseWriter{ResponseWriter: rec, gzCts: gzDefaultContentTypes}
		assert.Same(t, rec, gw.Unwrap())
	})

	t.Run("repeated WriteHeader ignored", func(t *testing.T) {
		rec := httptest.NewRecorder()
		gw := &gzipResponseWriter{ResponseWriter: rec, gzCts: gzDefaultContentTypes}
		gw.Header().Set("Content-Type", "text/plain")
		gw.WriteHeader(http.StatusTeapot)
		gw.WriteHeader(http.StatusOK)
		gw.close()
		assert.Equal(t, http.StatusTeapot, rec.Code)
	})

	t.Run("flush without a body commits the status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		gw := &gzipResponseWriter{ResponseWriter: rec, gzCts: gzDefaultContentTypes}
		gw.Flush()
		gw.close()
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
