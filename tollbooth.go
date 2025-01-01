package rest

import (
	"net/http"

	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
)

// based on https://github.com/didip/tollbooth_chi/blob/master/tollbooth_chi.go
// added support of v8 and simplified, removed chi dependency
// one notable difference is that this middleware sets IP lookup to RemoteAddr by default,
// however, it can be overridden by setting it in the limiter

// LimitHandler wraps http.Handler with tollbooth limiter
func LimitHandler(lmt *limiter.Limiter) func(http.Handler) http.Handler {
	// // set IP lookup only if not set
	if lmt.GetIPLookup().Name == "" {
		lmt.SetIPLookup(limiter.IPLookup{Name: "RemoteAddr"})
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				http.Error(w, "Context was canceled", http.StatusServiceUnavailable)
				return
			default:
				if httpError := tollbooth.LimitByRequest(lmt, w, r); httpError != nil {
					lmt.ExecOnLimitReached(w, r)
					w.Header().Add("Content-Type", lmt.GetMessageContentType())
					w.WriteHeader(httpError.StatusCode)
					w.Write([]byte(httpError.Message)) //nolint:gosec // not much we can do here with failed write
					return
				}
				next.ServeHTTP(w, r)
			}
		})
	}
}
