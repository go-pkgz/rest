package rest

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/go-pkgz/rest/realip"
)

// Profiler is a convenient subrouter used for mounting net/http/pprof. ie.
//
//	mux := http.NewServeMux()
//	mux.Handle("/debug/", http.StripPrefix("/debug", rest.Profiler("127.0.0.1")))
//
// The handler expects paths starting at /pprof/ and /vars, so mount it with the prefix stripped. The index is
// then at /debug/pprof/, with the trailing slash: a request for /debug/pprof is redirected to /pprof/,
// outside the mount.
//
// Access is limited to the given source addresses, in any form OnlyFrom takes. Called without any, it
// rejects every request, as Metrics does, since /pprof/cmdline and /vars publish the command line.
// Use ProfilerAllowAll to serve every source on purpose.
func Profiler(onlyIps ...string) http.Handler {
	if len(onlyIps) == 0 {
		return Wrap(profilerMux(), NoCache, rejectAll)
	}
	return Wrap(profilerMux(), NoCache, OnlyFrom(onlyIps...))
}

// ProfilerAllowAll serves the Profiler routes to any source, without any ip check, so it also serves a Unix
// socket client, whose address is not an IP. /pprof/cmdline and /vars publish the command line, which usually
// carries the flag values the process was started with, so only use this where something else already keeps
// the endpoint private.
func ProfilerAllowAll() http.Handler {
	return Wrap(profilerMux(), NoCache)
}

func profilerMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/pprof/", pprofByName)
	mux.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/pprof/profile", pprof.Profile)
	mux.HandleFunc("/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/pprof/trace", pprof.Trace)
	mux.HandleFunc("/vars", expVars)
	return mux
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

// rejectAll answers every request with 403, the way OnlyFrom rejects an address it doesn't list
func rejectAll(_ http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := realip.Get(r)
		if err != nil {
			ip = r.RemoteAddr
		}
		_ = EncodeJSON(w, http.StatusForbidden, JSON{"error": fmt.Sprintf("ip %q rejected", ip)})
	})
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
