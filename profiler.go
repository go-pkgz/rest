package rest

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
)

// Profiler is a convenient subrouter used for mounting net/http/pprof. ie.
//
//	func MyService() http.Handler {
//	  r := chi.NewRouter()
//	  // ..middlewares
//	  r.Mount("/debug", middleware.Profiler())
//	  // ..routes
//	  return r
//	}
func Profiler(onlyIps ...string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/pprof/", pprofByName)
	mux.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/pprof/profile", pprof.Profile)
	mux.HandleFunc("/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/pprof/trace", pprof.Trace)
	mux.HandleFunc("/vars", expVars)

	return Wrap(mux, NoCache, OnlyFrom(onlyIps...))
}

// pprofByName serves the index for /pprof/ and the named profile for /pprof/{name}. pprof.Index only
// resolves names under /debug/pprof/, so with the prefix stripped it would answer every name with the index
func pprofByName(w http.ResponseWriter, r *http.Request) {
	if name := strings.TrimPrefix(r.URL.Path, "/pprof/"); name != "" {
		pprof.Handler(name).ServeHTTP(w, r)
		return
	}
	pprof.Index(w, r)
}

// expVars copied from stdlib expvar.go as is not public.
func expVars(w http.ResponseWriter, _ *http.Request) {
	first := true
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\n")
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}
