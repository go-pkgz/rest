// Package logger implements logging middleware
package logger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-pkgz/rest/internal/requestid"
	"github.com/go-pkgz/rest/realip"
)

// Middleware is a logger for rest requests.
type Middleware struct {
	prefix         string
	logBody        bool
	maxBodySize    int
	ipFn           func(ip string) string
	userFn         func(r *http.Request) (string, error)
	subjFn         func(r *http.Request) (string, error)
	bodyFn         func(body string, truncated bool) string
	log            Backend
	apacheCombined bool
}

// Backend is logging backend
type Backend interface {
	Logf(format string, args ...any)
}

type logParts struct {
	duration   time.Duration
	rawURL     string
	method     string
	remoteIP   string
	statusCode int
	respSize   int
	host       string
	traceID    string

	prefix string
	user   string
	body   string
}

type stdBackend struct{}

func (s stdBackend) Logf(format string, args ...any) {
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

	formater := l.formatDefault
	if l.apacheCombined {
		formater = l.formatApacheCombined
	}

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
			u.RawQuery = l.sanitizeQuery(u.RawQuery)
			rawurl := u.String()
			if unescURL, err := url.QueryUnescape(rawurl); err == nil {
				rawurl = unescURL
			}
			// unescaping can surface line breaks the encoded url hid, collapse them the same way
			// the body is collapsed so an embedded break can't forge additional log records
			rawurl = lineBreaks.Replace(rawurl)

			remoteIP, err := realip.Get(r)
			if err != nil {
				remoteIP = "unknown ip"
			}
			if l.ipFn != nil { // mask ip with ipFn
				remoteIP = l.ipFn(remoteIP)
			}

			// a request id comes from the client or, when Trace generated one, from the request context
			// (Trace outside the logger) or the response headers (Trace inside it)
			traceID := r.Header.Get("X-Request-ID")
			if traceID == "" {
				traceID = requestid.From(r.Context())
			}
			if traceID == "" {
				traceID = ww.Header().Get("X-Request-ID")
			}
			// the handler can put anything in its response header, keep it on one line like the url and body
			traceID = lineBreaks.Replace(traceID)

			server := r.URL.Hostname()
			if server == "" {
				server = strings.Split(r.Host, ":")[0]
			}

			p := &logParts{
				duration:   t2.Sub(t1),
				rawURL:     rawurl,
				method:     r.Method,
				host:       server,
				traceID:    traceID,
				remoteIP:   remoteIP,
				statusCode: ww.status,
				respSize:   ww.size,
				prefix:     l.prefix,
				user:       user,
				body:       body,
			}

			l.log.Logf("%s", formater(r, p))
		}()

		next.ServeHTTP(wrapResponseWriter(ww), r)
	}
	return http.HandlerFunc(fn)
}

func (l *Middleware) formatDefault(r *http.Request, p *logParts) string {
	var bld strings.Builder
	if l.prefix != "" {
		_, _ = bld.WriteString(l.prefix)
		_, _ = bld.WriteString(" ")
	}

	_, _ = fmt.Fprintf(&bld, "%s - %s - %s - %s - %d (%d) - %v",
		p.method, p.rawURL, p.host, p.remoteIP, p.statusCode, p.respSize, p.duration)

	if p.user != "" {
		_, _ = bld.WriteString(" - ")
		_, _ = bld.WriteString(p.user)
	}

	if l.subjFn != nil {
		if subj, err := l.subjFn(r); err == nil {
			_, _ = bld.WriteString(" - ")
			_, _ = bld.WriteString(subj)
		}
	}

	if p.traceID != "" {
		_, _ = bld.WriteString(" - ")
		_, _ = bld.WriteString(p.traceID)
	}

	if p.body != "" {
		_, _ = bld.WriteString(" - ")
		_, _ = bld.WriteString(p.body)
	}
	return bld.String()
}

// 127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326 "http://www.example.com/start.html" "Mozilla/4.08 [en] (Win98; I ;Nav)"
// nolint gosec
func (l *Middleware) formatApacheCombined(r *http.Request, p *logParts) string {
	username := "-"
	if p.user != "" {
		username = p.user
	}

	var bld strings.Builder
	bld.WriteString(p.remoteIP)
	bld.WriteString(" - ")
	bld.WriteString(username)
	bld.WriteString(" [")
	bld.WriteString(time.Now().Format("02/Jan/2006:15:04:05 -0700"))
	bld.WriteString(`] "`)
	bld.WriteString(p.method)
	bld.WriteString(" ")
	bld.WriteString(p.rawURL)
	bld.WriteString(" ")
	bld.WriteString(r.Proto)
	bld.WriteString(`" `)
	bld.WriteString(strconv.Itoa(p.statusCode))
	bld.WriteString(" ")
	bld.WriteString(strconv.Itoa(p.respSize))

	bld.WriteString(` "`)
	bld.WriteString(r.Header.Get("Referer"))
	bld.WriteString(`" "`)
	bld.WriteString(r.Header.Get("User-Agent"))
	bld.WriteString(`"`)
	return bld.String()
}

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

// lineBreaks maps every character that can start a new line to a space, so a body
// can't forge extra log records. reMultWhtsp only collapses runs of two or more, so
// a lone CR or a Unicode line separator would otherwise slip through.
var lineBreaks = strings.NewReplacer(
	"\n", " ", // LF
	"\r", " ", // CR
	"\v", " ", // vertical tab
	"\f", " ", // form feed
	"\u0085", " ", // NEL
	"\u2028", " ", // line separator
	"\u2029", " ", // paragraph separator
)

func (l *Middleware) getBody(r *http.Request) string {
	if !l.logBody {
		return ""
	}

	reader, body, hasMore, err := peek(r.Body, int64(l.maxBodySize))
	if err != nil {
		return ""
	}

	// "The Server will close the request body. The ServeHTTP Handler does not need to."
	// https://golang.org/pkg/net/http/#Request
	// so we can use ioutil.NopCloser() to make io.ReadCloser.
	// note that below assignment is not approved by the docs:
	// "Except for reading the body, handlers should not modify the provided Request."
	// https://golang.org/pkg/net/http/#Handler
	r.Body = io.NopCloser(reader)

	// the transform owns the logged body: it receives the body (capped at
	// maxBodySize) and a flag telling it whether more was dropped, and decides
	// how to render it - mask values, summarize, or emit a marker for a
	// truncated body. an empty body has nothing to transform, so it is left
	// alone. without a transform the body is logged as read, with the "..."
	// marker appended when it was truncated.
	switch {
	case l.bodyFn != nil && body != "":
		body = l.bodyFn(body, hasMore)
	case hasMore:
		body += "..."
	}

	// always collapse to a single line, regardless of the transform, so an
	// embedded line break in the body can't forge additional log lines.
	if body != "" {
		body = lineBreaks.Replace(body)
		body = reMultWhtsp.ReplaceAllString(body, " ")
	}

	return body
}

// peek the first n bytes as string
func peek(r io.Reader, n int64) (reader io.Reader, s string, hasMore bool, err error) {
	if n < 0 {
		n = 0
	}

	buf := new(bytes.Buffer)
	_, err = io.CopyN(buf, r, n+1)
	if err == io.EOF {
		str := buf.String()
		return buf, str, false, nil
	}
	if err != nil {
		return r, "", false, err
	}

	// one extra byte is successfully read
	s = buf.String()
	s = s[:len(s)-1]

	return io.MultiReader(buf, r), s, true, nil
}

var keysToHide = []string{"password", "passwd", "secret", "credentials", "token"}

// Hide query values for keysToHide. May change order of query params.
// May escape unescaped query params.
func (l *Middleware) sanitizeQuery(rawQuery string) string {
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

// AnonymizeIP resets the last part of IPv4 to 0 and the low 80 bits of IPv6 to 0,
// keeping the /48 network: 123.212.12.78 becomes 123.212.12.0, 2001:db8:abcd:12::1 becomes 2001:db8:abcd::.
// An IPv4 address in IPv6 form keeps its form, ::ffff:808:404 becomes ::ffff:8.8.4.0. A port, square brackets
// and an IPv6 zone are dropped. A value that is not an IP address is reset after its last dot if it has
// four dot-separated parts, otherwise returned unchanged.
func AnonymizeIP(ip string) string {
	if ip == "" {
		return ""
	}

	addr, err := netip.ParseAddr(strings.TrimSuffix(strings.TrimPrefix(ip, "["), "]"))
	if err != nil {
		if addrPort, portErr := netip.ParseAddrPort(ip); portErr == nil {
			addr, err = addrPort.Addr(), nil
		}
	}
	if err == nil {
		bits := 24 // IPv4
		switch {
		case addr.Is4In6():
			bits = 96 + 24 // the IPv4 part sits in the low 32 bits
		case addr.Is6():
			bits = 48
		}
		if prefix, prefixErr := addr.Prefix(bits); prefixErr == nil {
			return prefix.Addr().String()
		}
	}

	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}

	return strings.Join(parts[:3], ".") + ".0"
}

// customResponseWriter is an HTTP response logger that keeps HTTP status code and
// the number of bytes written. The handler gets it through wrapResponseWriter, which offers
// http.Flusher and http.Hijacker only when the writer beneath does.
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

// WriteHeader implements http.ResponseWriter and saves status
func (c *customResponseWriter) WriteHeader(status int) {
	c.status = status
	c.ResponseWriter.WriteHeader(status)
}

// Write implements http.ResponseWriter and tracks number of bytes written
func (c *customResponseWriter) Write(b []byte) (int, error) {
	size, err := c.ResponseWriter.Write(b)
	c.size += size
	return size, err
}

// Unwrap exposes the underlying writer to http.ResponseController
func (c *customResponseWriter) Unwrap() http.ResponseWriter {
	return c.ResponseWriter
}

// the wrapper must offer exactly the optional interfaces the underlying writer has, otherwise a
// handler's type assertion succeeds and the call then does nothing, which is how a Flush under
// rest.Timeout (offering neither) would be silently lost
type flushWriter struct{ *customResponseWriter }

func (w flushWriter) Flush() { w.ResponseWriter.(http.Flusher).Flush() }

// FlushError lets http.ResponseController report a failed flush, which Flush has no way to return
func (w flushWriter) FlushError() error { return http.NewResponseController(w.ResponseWriter).Flush() }

type hijackWriter struct{ *customResponseWriter }

func (w hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

type flushHijackWriter struct{ *customResponseWriter }

func (w flushHijackWriter) Flush() { w.ResponseWriter.(http.Flusher).Flush() }

// FlushError lets http.ResponseController report a failed flush, which Flush has no way to return
func (w flushHijackWriter) FlushError() error {
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func (w flushHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// wrapResponseWriter picks the variant matching the capabilities of the writer underneath
func wrapResponseWriter(c *customResponseWriter) http.ResponseWriter {
	_, isFlusher := c.ResponseWriter.(http.Flusher)
	_, isHijacker := c.ResponseWriter.(http.Hijacker)

	switch {
	case isFlusher && isHijacker:
		return flushHijackWriter{c}
	case isFlusher:
		return flushWriter{c}
	case isHijacker:
		return hijackWriter{c}
	}
	return c
}
