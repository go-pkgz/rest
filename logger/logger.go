package logger

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Middleware is a logger for rest requests.
type Middleware struct {
	prefix      string
	logBody     bool
	maxBodySize int
	ipFn        func(ip string) string
	userFn      func(r *http.Request) (string, error)
	subjFn      func(r *http.Request) (string, error)
	log         Backend
}

// Backend is logging backend
type Backend interface {
	Logf(format string, args ...interface{})
}

type stdBackend struct{}

func (s stdBackend) Logf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Logger is a default logger middleware with "REST" prefix
func Logger(next http.Handler) http.Handler {
	l := New(Prefix("REST"))
	return l.Handler(next)

}

// New makes rest logger with given options
func New(options ...Option) *Middleware {
	res := Middleware{
		prefix:      "",
		maxBodySize: 1024,
		log:         stdBackend{},
	}
	for _, opt := range options {
		opt(&res)
	}
	return &res
}

// Handler middleware prints http log
func (l *Middleware) Handler(next http.Handler) http.Handler {

	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := newCustomResponseWriter(w)

		user := ""
		if l.userFn != nil {
			if u, err := l.userFn(r); err == nil {
				user = u
			}
		}

		body := l.getBody(r)
		t1 := time.Now()
		defer func() {
			t2 := time.Now()

			u := *r.URL // shallow copy
			u.RawQuery = sanitizeQuery(u.RawQuery)
			rawurl := u.String()
			if unescURL, err := url.QueryUnescape(rawurl); err == nil {
				rawurl = unescURL
			}

			remoteIP := strings.Split(r.RemoteAddr, ":")[0]
			if strings.HasPrefix(r.RemoteAddr, "[") {
				remoteIP = strings.Split(r.RemoteAddr, "]:")[0] + "]"
			}

			if l.ipFn != nil { // mask ip with ipFn
				remoteIP = l.ipFn(remoteIP)
			}

			var bld strings.Builder
			if l.prefix != "" {
				bld.WriteString(l.prefix)
				bld.WriteString(" ")
			}

			bld.WriteString(fmt.Sprintf("%s - %s - %s - %d (%d) - %v", r.Method, rawurl, remoteIP, ww.status, ww.size, t2.Sub(t1)))

			if user != "" {
				bld.WriteString(" - ")
				bld.WriteString(user)
			}

			if l.subjFn != nil {
				if subj, err := l.subjFn(r); err == nil {
					bld.WriteString(" - ")
					bld.WriteString(subj)
				}
			}

			if traceID := r.Header.Get("X-Request-ID"); traceID != "" {
				bld.WriteString(" - ")
				bld.WriteString(traceID)
			}

			if body != "" {
				bld.WriteString(" - ")
				bld.WriteString(body)
			}

			l.log.Logf("%s", bld.String())
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

func (l *Middleware) getBody(r *http.Request) string {
	if !l.logBody {
		return ""
	}

	content, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return ""
	}

	body := string(content)
	r.Body = ioutil.NopCloser(bytes.NewReader(content))

	if len(body) > 0 {
		body = strings.Replace(body, "\n", " ", -1)
		body = reMultWhtsp.ReplaceAllString(body, " ")
	}

	if len(body) > l.maxBodySize {
		body = body[:l.maxBodySize] + "..."
	}

	return body
}

var keysToHide = []string{"password", "passwd", "secret", "credentials", "token"}

// Hide query values for keysToHide. May change order of query params.
// May escape unescaped query params.
func sanitizeQuery(rawQuery string) string {
	// note that we skip non-nil error further
	query, err := url.ParseQuery(rawQuery)

	isHidden := func(key string) bool {
		for _, k := range keysToHide {
			if strings.EqualFold(k, key) {
				return true
			}
		}
		return false
	}

	present := false
	for key, values := range query {
		if isHidden(key) {
			present = true
			for i := range values {
				values[i] = "********"
			}
		}
	}

	// short circuit
	if (err == nil) && !present {
		return rawQuery
	}

	return query.Encode()
}

// customResponseWriter is an HTTP response logger that keeps HTTP status code and
// the number of bytes written.
// It implements http.ResponseWriter, http.Flusher and http.Hijacker.
// Note that type assertion from http.ResponseWriter(customResponseWriter) to
// http.Flusher and http.Hijacker is always succeed but underlying http.ResponseWriter
// may not implement them.
type customResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func newCustomResponseWriter(w http.ResponseWriter) *customResponseWriter {
	return &customResponseWriter{
		ResponseWriter: w,
		status:         200,
	}
}

func (c *customResponseWriter) WriteHeader(status int) {
	c.status = status
	c.ResponseWriter.WriteHeader(status)
}

func (c *customResponseWriter) Write(b []byte) (int, error) {
	size, err := c.ResponseWriter.Write(b)
	c.size += size
	return size, err
}

func (c *customResponseWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (c *customResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := c.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not implement the Hijacker interface")
}
