package logger

import (
	"net/http"
)

// Option func type
type Option func(l *Middleware)

// WithBody triggers request body logging.
func WithBody(l *Middleware) {
	l.logBody = true
}

// MaxBodySize functional option defines the largest body size to log.
func MaxBodySize(max int) Option {
	return func(l *Middleware) {
		if max >= 0 {
			l.maxBodySize = max
		}
	}
}

// Prefix functional option defines log line prefix.
func Prefix(prefix string) Option {
	return func(l *Middleware) {
		l.prefix = prefix
	}
}

// IPfn functional option defines ip masking function.
func IPfn(ipFn func(ip string) string) Option {
	return func(l *Middleware) {
		l.ipFn = ipFn
	}
}

// UserFn triggers user name logging if userFn is not nil.
func UserFn(userFn func(r *http.Request) (string, error)) Option {
	return func(l *Middleware) {
		l.userFn = userFn
	}
}

// SubjFn functional option defines subject function.
func SubjFn(userFn func(r *http.Request) (string, error)) Option {
	return func(l *Middleware) {
		l.subjFn = userFn
	}
}

// Log functional option defines loging backend.
func Log(log Backend) Option {
	return func(l *Middleware) {
		l.log = log
	}
}
