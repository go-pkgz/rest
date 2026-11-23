package rest

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

func TestBasicAuth(t *testing.T) {

	mw := BasicAuth(func(user, passwd string) bool {
		return user == "dev" && passwd == "good"
	})

	ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %s", r.URL)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("blah"))
		require.NoError(t, err)
		assert.True(t, IsAuthorized(r.Context()))
	})))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "good")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "bad")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
}

func TestBasicAuthWithUserPasswd(t *testing.T) {
	mw := BasicAuthWithUserPasswd("dev", "good")

	ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %s", r.URL)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("blah"))
		require.NoError(t, err)
		assert.True(t, IsAuthorized(r.Context()))
	})))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "good")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "bad")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	}
}

func TestBasicAuthWithPrompt(t *testing.T) {
	mw := BasicAuthWithPrompt("dev", "good")

	ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %s", r.URL)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("blah"))
		require.NoError(t, err)
		assert.True(t, IsAuthorized(r.Context()))
	})))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, `Basic realm="restricted", charset="UTF-8"`, resp.Header.Get("WWW-Authenticate"))
	}

	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "good")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
	{
		req, err := http.NewRequest("GET", u, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "bad")
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, `Basic realm="restricted", charset="UTF-8"`, resp.Header.Get("WWW-Authenticate"))
	}
}

func TestBasicAuthWithHash(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("good"), bcrypt.MinCost)
	require.NoError(t, err)
	t.Logf("hashed password: %s", string(hashedPassword))

	mw := BasicAuthWithBcryptHash("dev", string(hashedPassword))

	ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %s", r.URL)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("blah"))
		require.NoError(t, err)
		assert.True(t, IsAuthorized(r.Context()))
	})))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")
	client := http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{
			name:           "no auth provided",
			username:       "",
			password:       "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "correct credentials",
			username:       "dev",
			password:       "good",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong username",
			username:       "wrong",
			password:       "good",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "wrong password",
			username:       "dev",
			password:       "bad",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "empty password",
			username:       "dev",
			password:       "",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", u, http.NoBody)
			require.NoError(t, err)

			if tc.username != "" || tc.password != "" {
				req.SetBasicAuth(tc.username, tc.password)
			}

			resp, err := client.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
		})
	}
}

func TestBasicAuthWithArgon2Hash(t *testing.T) {
	password := "good"
	hash, salt, err := GenerateArgon2Hash(password)
	require.NoError(t, err)
	t.Logf("hash: %s, salt: %s", hash, salt)

	// verify the returned values are valid base64
	_, err = base64.StdEncoding.DecodeString(hash)
	require.NoError(t, err, "hash should be valid base64")
	_, err = base64.StdEncoding.DecodeString(salt)
	require.NoError(t, err, "salt should be valid base64")

	mw := BasicAuthWithArgon2Hash("dev", hash, salt)

	ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %s", r.URL)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("blah"))
		require.NoError(t, err)
		assert.True(t, IsAuthorized(r.Context()))
	})))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")
	client := http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{
			name:           "no auth provided",
			username:       "",
			password:       "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "correct credentials",
			username:       "dev",
			password:       "good",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong username",
			username:       "wrong",
			password:       "good",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "wrong password",
			username:       "dev",
			password:       "bad",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", u, http.NoBody)
			require.NoError(t, err)

			if tc.username != "" || tc.password != "" {
				req.SetBasicAuth(tc.username, tc.password)
			}

			resp, err := client.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
		})
	}
}

func TestHashGenerationFunctions(t *testing.T) {
	t.Run("bcrypt hash generation", func(t *testing.T) {
		hash, err := GenerateBcryptHash("testpassword")
		require.NoError(t, err)
		require.NotEmpty(t, hash)

		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("testpassword"))
		require.NoError(t, err)
	})

	t.Run("argon2 hash generation", func(t *testing.T) {
		hash, salt, err := GenerateArgon2Hash("testpassword")
		require.NoError(t, err)
		require.NotEmpty(t, hash)
		require.NotEmpty(t, salt)

		// verify the values are valid base64
		hashBytes, err := base64.StdEncoding.DecodeString(hash)
		require.NoError(t, err, "hash should be valid base64")
		saltBytes, err := base64.StdEncoding.DecodeString(salt)
		require.NoError(t, err, "salt should be valid base64")

		// verify the hash works
		newHash := argon2.IDKey([]byte("testpassword"), saltBytes, 1, 64*1024, 4, 32)
		require.Equal(t, hashBytes, newHash)

		// test with wrong password
		wrongHash := argon2.IDKey([]byte("wrongpassword"), saltBytes, 1, 64*1024, 4, 32)
		require.NotEqual(t, hashBytes, wrongHash)
	})
}

func TestArgon2InvalidInputs(t *testing.T) {
	t.Run("invalid base64 salt", func(t *testing.T) {
		mw := BasicAuthWithArgon2Hash("dev", "validbase64==", "invalid-base64")
		ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called with invalid base64")
		})))
		defer ts.Close()

		req, err := http.NewRequest("GET", ts.URL, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "password")

		client := http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("invalid base64 hash", func(t *testing.T) {
		mw := BasicAuthWithArgon2Hash("dev", "invalid-base64", "validbase64==")
		ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called with invalid base64")
		})))
		defer ts.Close()

		req, err := http.NewRequest("GET", ts.URL, http.NoBody)
		require.NoError(t, err)
		req.SetBasicAuth("dev", "password")

		client := http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestBasicAuthWithBcryptHashAndPrompt(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("good"), bcrypt.MinCost)
	require.NoError(t, err)
	t.Logf("hashed password: %s", string(hashedPassword))

	mw := BasicAuthWithBcryptHashAndPrompt("dev", string(hashedPassword))

	ts := httptest.NewServer(mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request %s", r.URL)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("blah"))
		require.NoError(t, err)
		assert.True(t, IsAuthorized(r.Context()))
	})))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")
	client := http.Client{Timeout: 5 * time.Second}

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
		checkPrompt    bool
	}{
		{
			name:           "no auth provided",
			username:       "",
			password:       "",
			expectedStatus: http.StatusUnauthorized,
			checkPrompt:    true,
		},
		{
			name:           "correct credentials",
			username:       "dev",
			password:       "good",
			expectedStatus: http.StatusOK,
			checkPrompt:    false,
		},
		{
			name:           "wrong username",
			username:       "wrong",
			password:       "good",
			expectedStatus: http.StatusUnauthorized,
			checkPrompt:    true,
		},
		{
			name:           "wrong password",
			username:       "dev",
			password:       "bad",
			expectedStatus: http.StatusUnauthorized,
			checkPrompt:    true,
		},
		{
			name:           "empty password",
			username:       "dev",
			password:       "",
			expectedStatus: http.StatusUnauthorized,
			checkPrompt:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", u, http.NoBody)
			require.NoError(t, err)

			if tc.username != "" || tc.password != "" {
				req.SetBasicAuth(tc.username, tc.password)
			}

			resp, err := client.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			if tc.checkPrompt {
				assert.Equal(t, `Basic realm="restricted", charset="UTF-8"`, resp.Header.Get("WWW-Authenticate"),
					"should include WWW-Authenticate header")
			}
		})
	}
}

func TestBasicAuthHashedWrongUserStillHashes(t *testing.T) {
	bcryptHash, err := GenerateBcryptHash("good")
	require.NoError(t, err)
	argonHash, argonSalt, err := GenerateArgon2Hash("good")
	require.NoError(t, err)

	tbl := []struct {
		name   string
		mw     func(http.Handler) http.Handler
		status int
	}{
		{name: "bcrypt", mw: BasicAuthWithBcryptHash("dev", bcryptHash), status: http.StatusForbidden},
		{name: "bcrypt with prompt", mw: BasicAuthWithBcryptHashAndPrompt("dev", bcryptHash), status: http.StatusUnauthorized},
		{name: "argon2", mw: BasicAuthWithArgon2Hash("dev", argonHash, argonSalt), status: http.StatusForbidden},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
			// a wrong user name must cost a hash comparison too, otherwise the response time tells valid
			// user names apart; only a lower bound is checked, as a loaded machine can only make it slower
			fastest := time.Duration(1<<63 - 1)
			for range 3 {
				req := httptest.NewRequest("GET", "/something", http.NoBody)
				req.SetBasicAuth("wrong", "good")
				rr := httptest.NewRecorder()
				st := time.Now()
				h.ServeHTTP(rr, req)
				fastest = min(fastest, time.Since(st))
				assert.Equal(t, tt.status, rr.Code)
			}
			assert.GreaterOrEqual(t, fastest, time.Millisecond, "wrong user answered without hashing")
		})
	}
}
