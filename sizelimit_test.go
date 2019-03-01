package rest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSizeLimit(t *testing.T) {

	tbl := []struct {
		method string
		body   string
		code   int
	}{
		{"GET", "", 200},
		{"POST", "1234567", 200},
		{"POST", "1234567890", 200},
		{"POST", "12345678901", 413},
		{"POST", "1234567", 200},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err, "body read failed")
		_, err = w.Write(body)
		require.NoError(t, err, "body write failed")
		dump, _ := httputil.DumpRequest(r, true)
		t.Log(string(dump))
	})

	ts := httptest.NewServer(SizeLimit(10)(handler))
	defer ts.Close()

	for i, tt := range tbl {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			client := http.Client{Timeout: 1 * time.Second}
			req, err := http.NewRequest(tt.method, ts.URL+"/"+strconv.Itoa(i), strings.NewReader(tt.body))
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, tt.code, resp.StatusCode)

			if resp.StatusCode != http.StatusRequestEntityTooLarge {
				body, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)
				defer resp.Body.Close()
				assert.Equal(t, tt.body, string(body), "body match")
			}
		})
	}
}
