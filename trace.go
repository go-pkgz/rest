package rest

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"net/http"
	"time"
)

type contextKey string

const traceHeader = "X-Request-ID"

// Trace looking for header X-Request-ID and makes it as uuid if not found, then populates it the result's header
func Trace(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(traceHeader)
		if traceID == "" {
			traceID = randToken()
		}
		w.Header().Set(traceHeader, traceID)
		ctx := context.WithValue(r.Context(), contextKey("requestID"), traceID)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// GetTraceID returns request id from the context
func GetTraceID(r *http.Request) string {
	if id, ok := r.Context().Value(contextKey("requestID")).(string); ok {
		return id
	}
	return ""
}

func randToken() string {
	fallback := func() string {
		return fmt.Sprintf("%x", time.Now().Nanosecond())
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fallback()
	}
	s := sha1.New()
	if _, err := s.Write(b); err != nil {
		return fallback()
	}
	return fmt.Sprintf("%x", s.Sum(nil))
}
