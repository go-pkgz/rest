package realip

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFromHeaders(t *testing.T) {
	t.Run("single X-Real-IP", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("Something", "1234567")
		req.Header.Add("X-Real-IP", "8.8.8.8")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "8.8.8.8", adr)
	})
	t.Run("X-Forwarded-For last public", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("Something", "1234567")
		req.Header.Add("X-Forwarded-For", "8.8.8.8,1.1.1.2, 30.30.30.1")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "30.30.30.1", adr)
	})
	t.Run("X-Forwarded-For last private", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("Something", "1234567")
		req.Header.Add("X-Forwarded-For", "8.8.8.8,1.1.1.2,192.168.1.1,10.0.0.65")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "1.1.1.2", adr)
	})
	t.Run("X-Forwarded-For public im the middle", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("Something", "1234567")
		req.Header.Add("X-Forwarded-For", "192.168.1.1, 8.8.8.8, 10.0.0.65")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "8.8.8.8", adr)
	})
	t.Run("X-Forwarded-For all private", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("Something", "1234567")
		req.Header.Add("X-Forwarded-For", "192.168.1.1,10.0.0.65")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.65", adr)
	})
	t.Run("X-Forwarded-For public, X-Real-IP private", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("Something", "1234567")
		req.Header.Add("X-Forwarded-For", "30.30.30.1")
		req.Header.Add("X-Real-Ip", "10.0.0.1")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "30.30.30.1", adr)
	})
	t.Run("X-Forwarded-For and X-Real-IP public", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("Something", "1234567")
		req.Header.Add("X-Forwarded-For", "30.30.30.1")
		req.Header.Add("X-Real-Ip", "8.8.8.8")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "30.30.30.1", adr)
	})
	t.Run("X-Forwarded-For private and X-Real-IP public", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("Something", "1234567")
		req.Header.Add("X-Forwarded-For", "10.0.0.2,192.168.1.1")
		req.Header.Add("X-Real-Ip", "8.8.8.8")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "8.8.8.8", adr)
	})
	t.Run("RemoteAddr fallback", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.RemoteAddr = "192.0.2.1:1234"
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "192.0.2.1", adr)
	})
	t.Run("X-Forwarded-For and X-Real-IP missing, no RemoteAddr either", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		ip, err := Get(req)
		assert.Error(t, err)
		assert.Equal(t, "", ip)
	})
	t.Run("X-Real-IP IPv6", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("X-Real-IP", "2001:0db8:85a3:0000:0000:8a2e:0370:7334")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "2001:0db8:85a3:0000:0000:8a2e:0370:7334", adr)
	})
	t.Run("X-Forwarded-For last IPv6 public", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("X-Forwarded-For", "2001:db8::ff00:42:8329,::1,fc00::")
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "2001:db8::ff00:42:8329", adr)
	})

	t.Run("RemoteAddr IPv6 fallback", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/something", http.NoBody)
		assert.NoError(t, err)
		req.RemoteAddr = "[2001:db8::ff00:42:8329]:1234"
		adr, err := Get(req)
		require.NoError(t, err)
		assert.Equal(t, "2001:db8::ff00:42:8329", adr)
	})
}

func TestGetFromRemoteAddr(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		log.Printf("%v", r)
		adr, err := Get(r)
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1", adr)
	}))

	req, err := http.NewRequest("GET", ts.URL+"/something", http.NoBody)
	require.NoError(t, err)
	client := http.Client{Timeout: time.Second}
	_, err = client.Do(req)
	require.NoError(t, err)
}

func TestGetWithVariousIPFormats(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		wantIP     string
		wantErr    bool
	}{
		{
			name:       "IPv4 with port",
			remoteAddr: "192.168.1.1:8080",
			wantIP:     "192.168.1.1",
			wantErr:    false,
		},
		{
			name:       "IPv4 without port",
			remoteAddr: "127.0.0.1",
			wantIP:     "127.0.0.1",
			wantErr:    false,
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[::1]:8080",
			wantIP:     "::1",
			wantErr:    false,
		},
		{
			name:       "IPv6 without port",
			remoteAddr: "::1",
			wantIP:     "::1",
			wantErr:    false,
		},
		{
			name:       "X-Forwarded-For",
			remoteAddr: "127.0.0.1:8080",
			headers:    map[string]string{"X-Forwarded-For": "203.0.113.1"},
			wantIP:     "203.0.113.1",
			wantErr:    false,
		},
		{
			name:       "Invalid IP",
			remoteAddr: "invalid-ip",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			require.NoError(t, err)

			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			gotIP, err := Get(req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantIP, gotIP)
		})
	}
}
