package rest

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var gzDefaultContentTypes = []string{
	"text/css",
	"text/javascript",
	"text/xml",
	"text/html",
	"text/plain",
	"application/javascript",
	"application/x-javascript",
	"application/json",
}

var gzPool = sync.Pool{
	New: func() any { return gzip.NewWriter(io.Discard) },
}

// gzipResponseWriter defers the compression decision until the response content type is known,
// either from the header the handler set or sniffed from the first chunk of the body.
type gzipResponseWriter struct {
	http.ResponseWriter

	gzCts       []string
	gz          *gzip.Writer
	status      int
	decided     bool
	wroteHeader bool
	hijacked    bool
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status

	// with a content type in hand the decision can be made right away, otherwise it waits for the
	// first Write so the body can be sniffed
	if ctype := w.Header().Get("Content-Type"); ctype != "" {
		w.decide(ctype)
		w.commit()
	}
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.decided {
		ctype := w.Header().Get("Content-Type")
		if ctype == "" {
			ctype = http.DetectContentType(b)
			w.Header().Set("Content-Type", ctype)
		}
		w.decide(ctype)
	}
	if !w.wroteHeader {
		w.commit()
	}
	if w.gz != nil {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// decide turns compression on if the response content type is one of the configured types
func (w *gzipResponseWriter) decide(ctype string) {
	w.decided = true

	if w.status == http.StatusNoContent || w.status == http.StatusNotModified {
		return // these carry no body to compress
	}

	for _, c := range w.gzCts {
		if !strings.HasPrefix(strings.ToLower(ctype), strings.ToLower(c)) {
			continue
		}
		gz := gzPool.Get().(*gzip.Writer)
		gz.Reset(w.ResponseWriter)
		w.gz = gz
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // the handler's length describes the uncompressed body
		return
	}
}

func (w *gzipResponseWriter) commit() {
	w.wroteHeader = true
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(w.status)
}

// close finishes the gzip stream and makes sure the status reaches the client even if the handler
// wrote no body at all
func (w *gzipResponseWriter) close() {
	if w.hijacked {
		return // the handler owns the connection now, nothing may be written to it
	}
	if !w.decided {
		w.decide(w.Header().Get("Content-Type"))
	}
	if !w.wroteHeader {
		w.commit()
	}
	if w.gz == nil {
		return
	}
	_ = w.gz.Close()
	gzPool.Put(w.gz)
	w.gz = nil
}

// Flush pushes buffered data out, keeping streaming responses working through the compressor
func (w *gzipResponseWriter) Flush() {
	if w.hijacked {
		return
	}
	if !w.wroteHeader {
		w.commit()
	}
	if w.gz != nil {
		_ = w.gz.Flush()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack passes through to the underlying writer for protocol upgrades
func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("http.Hijacker not supported")
	}
	conn, rw, err := h.Hijack()
	if err == nil {
		w.hijacked = true
	}
	return conn, rw, err
}

// Unwrap exposes the underlying writer to http.ResponseController
func (w *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Gzip is a middleware compressing response. The decision is made on the response content type,
// so it applies to what the handler actually produced. Content types default to the common textual
// ones and can be overridden by the caller.
func Gzip(contentTypes ...string) func(http.Handler) http.Handler {

	gzCts := gzDefaultContentTypes
	if len(contentTypes) > 0 {
		gzCts = contentTypes
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// the representation depends on Accept-Encoding, caches must key on it even when not compressing
			w.Header().Add("Vary", "Accept-Encoding")

			if !acceptsGzip(r.Header.Get("Accept-Encoding")) {
				next.ServeHTTP(w, r)
				return
			}

			gw := &gzipResponseWriter{ResponseWriter: w, gzCts: gzCts}
			defer gw.close()
			next.ServeHTTP(gw, r)
		})
	}
}

// acceptsGzip reports whether the client accepts gzip, honoring an explicit q=0 rejection
func acceptsGzip(header string) bool {
	for enc := range strings.SplitSeq(header, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(enc), ";")
		if n := strings.ToLower(strings.TrimSpace(name)); n != "gzip" && n != "*" {
			continue
		}
		if q, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(params)), "q="); ok {
			if v, err := strconv.ParseFloat(strings.TrimSpace(q), 64); err == nil && v == 0 {
				continue // explicitly not acceptable
			}
		}
		return true
	}
	return false
}
