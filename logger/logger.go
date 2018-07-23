package logger

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
)

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

// Middleware for logging rest requests
type Middleware struct {
	prefix      string
	maxBodySize int
	flags       []Flag
	ipFn        func(ip string) string
	userFn      func(r *http.Request) (string, error)
}

// Flag type
type Flag int

// logger flags enum
const (
	All Flag = iota
	User
	Body
	None
)

// New makes rest Logger with given options
func New(options ...Option) Middleware {
	res := Middleware{
		prefix:      "",
		maxBodySize: 1024,
		flags:       []Flag{All},
	}
	for _, opt := range options {
		opt(&res)
	}
	return res
}

// Handler middleware prints http log
func (l *Middleware) Handler(next http.Handler) http.Handler {

	fn := func(w http.ResponseWriter, r *http.Request) {

		if l.inLogFlags(None) { // skip logging
			next.ServeHTTP(w, r)
			return
		}

		ww := middleware.NewWrapResponseWriter(w, 1)
		body, user := l.getBodyAndUser(r)
		t1 := time.Now()
		defer func() {
			t2 := time.Now()

			q := r.URL.String()
			if qun, err := url.QueryUnescape(q); err == nil {
				q = qun
			}

			remoteIP := strings.Split(r.RemoteAddr, ":")[0]
			if strings.HasPrefix(r.RemoteAddr, "[") {
				remoteIP = strings.Split(r.RemoteAddr, "]:")[0] + "]"
			}

			if l.ipFn != nil { // mask ip with ipFn
				remoteIP = l.ipFn(remoteIP)
			}

			log.Printf("%s %s - %s - %s - %d (%d) - %v %s %s",
				l.prefix, r.Method, q, remoteIP, ww.Status(), ww.BytesWritten(), t2.Sub(t1), user, body)
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

func (l *Middleware) getBodyAndUser(r *http.Request) (body string, user string) {
	ctx := r.Context()
	if ctx == nil {
		return "", ""
	}

	if l.inLogFlags(Body) {
		if content, err := ioutil.ReadAll(r.Body); err == nil {
			body = string(content)
			r.Body = ioutil.NopCloser(bytes.NewReader(content))

			if len(body) > 0 {
				body = strings.Replace(body, "\n", " ", -1)
				body = reMultWhtsp.ReplaceAllString(body, " ")
			}

			if len(body) > l.maxBodySize {
				body = body[:l.maxBodySize] + "..."
			}
		}
	}

	if l.inLogFlags(User) && l.userFn != nil {
		u, err := l.userFn(r)
		if err == nil && u != "" {
			user = fmt.Sprintf(" - %s", u)
		}
	}

	return body, user
}

func (l *Middleware) inLogFlags(f Flag) bool {
	for _, flg := range l.flags {
		if (flg == All && f != None) || flg == f {
			return true
		}
	}
	return false
}
