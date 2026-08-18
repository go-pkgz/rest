package rest

import (
	"crypto/sha1" //nolint not used for cryptography
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CacheControl is a middleware setting cache expiration. Using url+version for etag
func CacheControl(expiration time.Duration, version string) func(http.Handler) http.Handler {
	return CacheControlDynamic(expiration, func(*http.Request) string { return version })
}

// CacheControlDynamic is a middleware setting cache expiration. Using url+ func(r) for etag.
// Conditional requests are handled for GET and HEAD only, answering 304 when If-None-Match carries
// the current etag. Other methods are passed to the handler, as this middleware doesn't know enough
// about the resource to enforce their preconditions.
func CacheControlDynamic(expiration time.Duration, versionFn func(r *http.Request) string) func(http.Handler) http.Handler {

	etag := func(r *http.Request, version string) string {
		s := fmt.Sprintf("%s:%s", version, r.URL.String())
		return fmt.Sprintf("%x", sha1.Sum([]byte(s))) //nolint
	}

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			e := `"` + etag(r, versionFn(r)) + `"`
			w.Header().Set("Etag", e)
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, no-cache", int(expiration.Seconds())))

			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				if match := r.Header.Get("If-None-Match"); etagMatches(match, e) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// etagMatches reports whether an If-None-Match header carries the given etag.
// Handles comma-separated lists and the W/ weak-validator prefix, comparing with the weak
// comparison of RFC 9110. The "*" wildcard is deliberately not matched here, as it asks whether
// any representation exists and the middleware can't answer that before the handler runs.
func etagMatches(header, etag string) bool {
	if header == "" {
		return false
	}
	etag = strings.TrimPrefix(strings.TrimSpace(etag), "W/")
	for tag := range strings.SplitSeq(header, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "" || tag == "*" {
			continue
		}
		if strings.TrimPrefix(tag, "W/") == etag {
			return true
		}
	}
	return false
}
