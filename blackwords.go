package rest

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-chi/render"
)

// BlackWords middleware doesn't allow some words in the request body
func BlackWords(words ...string) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			if content, err := ioutil.ReadAll(r.Body); err == nil {
				body := strings.ToLower(string(content))
				r.Body = ioutil.NopCloser(bytes.NewReader(content))

				if len(body) > 0 {
					for _, word := range words {
						if strings.Contains(body, strings.ToLower(word)) {
							render.Status(r, http.StatusForbidden)
							render.JSON(w, r, JSON{"error": "one of blacklisted words detected"})
							return
						}
					}
				}
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
