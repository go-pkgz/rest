package rest

import (
	"net/http"

	"github.com/go-pkgz/rest/utils"
)

// RealIP is a middleware that sets a http.Request's RemoteAddr to the results
// of parsing either the X-Forwarded-For or X-Real-IP headers.
//
// This middleware should only be used if user can trust the headers sent with request.
// If reverse proxies are configured to pass along arbitrary header values from the client,
// or if this middleware used without a reverse proxy, malicious clients could set anything
// as X-Forwarded-For header and attack the server in various ways.
func RealIP(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if rip, err := utils.GetIPAddress(r); err == nil {
			r.RemoteAddr = rip
		}
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// GetIPAddress returns real ip from the given request, if ip can be extracted returns ""
func GetIPAddress(r *http.Request) string {
	if rip, err := utils.GetIPAddress(r); err == nil {
		return rip
	}
	return ""
}
