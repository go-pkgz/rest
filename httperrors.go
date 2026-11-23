package rest

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/go-pkgz/rest/logger"
)

// ErrorLogger wraps logger.Backend
type ErrorLogger struct {
	l logger.Backend
}

// NewErrorLogger creates ErrorLogger for given Backend
func NewErrorLogger(l logger.Backend) *ErrorLogger {
	return &ErrorLogger{l: l}
}

// Log sends json error message {error: msg} with error code and logging error and caller
func (e *ErrorLogger) Log(w http.ResponseWriter, r *http.Request, httpCode int, err error, msg ...string) {
	m := ""
	if len(msg) > 0 {
		m = strings.Join(msg, ". ")
	}
	if e.l != nil {
		e.l.Logf("%s", errDetailsMsg(r, httpCode, err, m))
	}
	renderJSONWithStatus(w, JSON{"error": m}, httpCode)
}

// SendErrorJSON sends {error: msg} with error code and logging error and caller
func SendErrorJSON(w http.ResponseWriter, r *http.Request, l logger.Backend, code int, err error, msg string) {
	if l != nil {
		l.Logf("%s", errDetailsMsg(r, code, err, msg))
	}
	renderJSONWithStatus(w, JSON{"error": msg}, code)
}

func errDetailsMsg(r *http.Request, code int, err error, msg string) string {

	q := r.URL.String()
	if qun, e := url.QueryUnescape(q); e == nil {
		q = qun
	}

	srcFileInfo := ""
	if pc, file, line, ok := runtime.Caller(2); ok {
		funcNameElems := strings.Split(runtime.FuncForPC(pc).Name(), "/")
		srcFileInfo = fmt.Sprintf(" [caused by %s:%d %s]", shortCallerPath(file), line, funcNameElems[len(funcNameElems)-1])
	}

	// RemoteAddr is host:port as set by the server, or a bare address once RealIP has rewritten it
	remoteIP := r.RemoteAddr
	if host, _, splitErr := net.SplitHostPort(remoteIP); splitErr == nil {
		remoteIP = host
	}
	if err == nil {
		err = errors.New("no error")
	}
	return fmt.Sprintf("%s - %v - %d - %s - %s%s", msg, err, code, remoteIP, q, srcFileInfo)
}

// shortCallerPath keeps the last three elements of the caller's file path, or all of them if there are fewer,
// as for a file in the root of a single-element module built with -trimpath
func shortCallerPath(file string) string {
	elems := strings.Split(file, "/")
	return strings.Join(elems[max(0, len(elems)-3):], "/")
}
