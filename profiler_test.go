package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"runtime/pprof"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfiler(t *testing.T) {
	ts := httptest.NewServer(Profiler("127.0.0.1"))
	defer ts.Close()

	t.Run("pprof index", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/pprof/")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Types of profiles available")
	})

	t.Run("pprof cmdline", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/pprof/cmdline")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("pprof symbol", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/pprof/symbol")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("pprof heap", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/pprof/heap")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("pprof goroutine", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/pprof/goroutine")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("pprof block", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/pprof/block")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("pprof threadcreate", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/pprof/threadcreate")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestProfiler_ExpVars(t *testing.T) {
	ts := httptest.NewServer(Profiler("127.0.0.1"))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/vars")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	// expvar always has cmdline and memstats
	assert.True(t, strings.Contains(string(body), "cmdline"), "should contain cmdline")
	assert.True(t, strings.Contains(string(body), "memstats"), "should contain memstats")
	// should be valid JSON structure
	assert.True(t, strings.HasPrefix(string(body), "{"), "should start with {")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(string(body)), "}"), "should end with }")
}

func TestProfiler_WithIPRestriction(t *testing.T) {
	ts := httptest.NewServer(Profiler("1.1.1.1"))
	defer ts.Close()

	// request from 127.0.0.1 should be blocked
	resp, err := http.Get(ts.URL + "/pprof/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	resp, err = http.Get(ts.URL + "/vars")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestProfiler_AllowedIP(t *testing.T) {
	ts := httptest.NewServer(Profiler("127.0.0.1"))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/pprof/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get(ts.URL + "/vars")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProfiler_AllocsAndMutex(t *testing.T) {
	ts := httptest.NewServer(Profiler("127.0.0.1"))
	defer ts.Close()

	tbl := []struct {
		path, prefix string
	}{
		{path: "/pprof/allocs?debug=1", prefix: "heap profile:"},
		{path: "/pprof/mutex?debug=1", prefix: "--- mutex:"},
	}
	for _, tt := range tbl {
		t.Run(tt.path, func(t *testing.T) {
			resp, err := http.Get(ts.URL + tt.path)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(string(body), tt.prefix), "got %q", firstLine(string(body)))
		})
	}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

func TestProfiler_NoAddressesDeniesAll(t *testing.T) {
	ts := httptest.NewServer(Profiler())
	defer ts.Close()

	for _, path := range []string{"/pprof/", "/pprof/cmdline", "/vars"} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(ts.URL + path)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

func TestProfiler_NoAddressesInvalidRemote(t *testing.T) {
	req := httptest.NewRequest("GET", "/vars", http.NoBody)
	req.RemoteAddr = "bad"
	rr := httptest.NewRecorder()
	Profiler().ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Contains(t, rr.Body.String(), `ip \"bad\" rejected`)
}

func TestProfiler_AnyAddress(t *testing.T) {
	h := Profiler("0.0.0.0/0", "::/0")
	for _, remote := range []string{"8.8.8.8:1234", "[2001:4860:4860::8888]:1234"} {
		t.Run(remote, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/vars", http.NoBody)
			req.RemoteAddr = remote
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestProfiler_AnyNamedProfile(t *testing.T) {
	const name = "rest_test_custom_profile"
	if pprof.Lookup(name) == nil {
		pprof.NewProfile(name)
	}
	ts := httptest.NewServer(Profiler("127.0.0.1"))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/pprof/" + name + "?debug=1")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(body), name+" profile:"), "got %q", firstLine(string(body)))

	resp, err = http.Get(ts.URL + "/pprof/no_such_profile")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestProfilerAllowAll(t *testing.T) {
	remotes := []string{"", "@", "[fe80::1%en0]:1234", "8.8.8.8:1234", "[2001:4860:4860::8888]:1234"}
	for _, remote := range remotes {
		t.Run("remote "+remote, func(t *testing.T) {
			for _, path := range []string{"/pprof/", "/pprof/cmdline", "/vars"} {
				req := httptest.NewRequest("GET", path, http.NoBody)
				req.RemoteAddr = remote
				rr := httptest.NewRecorder()
				ProfilerAllowAll().ServeHTTP(rr, req)
				assert.Equal(t, http.StatusOK, rr.Code, path)
				assert.Equal(t, "no-cache, no-store, no-transform, must-revalidate, private, max-age=0",
					rr.Header().Get("Cache-Control"), path)
			}
		})
	}
}
