package rest

import (
	"crypto/rand"
	"crypto/sha1" //nolint:gosec //not used for cryptography
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/go-pkgz/rest/internal/requestid"
)

type contextKey string

const traceHeader = "X-Request-ID"

// Trace looks for header X-Request-ID and makes it as random id if not found, then populates it to the result's header
// and to request context
func Trace(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(traceHeader)
		if traceID == "" {
			traceID = randToken()
		}
		w.Header().Set(traceHeader, traceID)
		r = r.WithContext(requestid.With(r.Context(), traceID))
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// GetTraceID returns request id from the context
func GetTraceID(r *http.Request) string {
	return requestid.From(r.Context())
}

func randToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().Nanosecond())
	}
	sum := sha1.Sum(b) //nolint:gosec //not used for cryptography
	return hex.EncodeToString(sum[:])
}
