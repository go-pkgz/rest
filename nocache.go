package rest

import (
	"net/http"
	"time"
)

// borrowed from https://github.com/go-chi/chi/blob/master/middleware/nocache.go

var epoch = time.Unix(0, 0).UTC().Format(http.TimeFormat)

var noCacheHeaders = map[string]string{
	"Expires":         epoch,
	"Cache-Control":   "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

// NoCache is a simple piece of middleware that sets a number of HTTP headers to prevent
// a router (or subrouter) from being cached by an upstream proxy and/or client.
//
// As per http://wiki.nginx.org/HttpProxyModule - NoCache sets:
//
//	Expires: Thu, 01 Jan 1970 00:00:00 GMT
//	Cache-Control: no-cache, no-store, no-transform, must-revalidate, private, max-age=0
//	X-Accel-Expires: 0
//	Pragma: no-cache (for HTTP/1.0 proxies/clients)
//
// It also removes the conditional request headers (If-None-Match, If-Modified-Since, If-Match,
// If-Unmodified-Since, If-Range) and ETag from the request, so a handler that respects them, such as
// http.ServeContent, answers neither 304 Not Modified nor 412 Precondition Failed.
func NoCache(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		// delete any ETag headers that may have been set
		for _, v := range etagHeaders {
			if r.Header.Get(v) != "" {
				r.Header.Del(v)
			}
		}

		// set our NoCache headers
		for k, v := range noCacheHeaders {
			w.Header().Set(k, v)
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
