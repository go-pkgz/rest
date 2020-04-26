package rest

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBlackwords(t *testing.T) {

	tbl := []struct {
		inp  string
		code int
	}{
		{"", 200},
		{"blah blah body", 200},
		{"blah blah body bad1", 403},
		{"blah blah body bad", 200},
		{"blah bad2 body bad", 403},
		{`{"fld": 123, "aa": {"$where": {"aaa": 567}}}`, 403},
	}

	bwMiddleware := BlackWords("bad1", "bad2", "$where")
	ts := httptest.NewServer(bwMiddleware(getTestHandlerBlah()))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	for n, tt := range tbl {
		tt := tt
		t.Run(fmt.Sprintf("test-%d", n), func(t *testing.T) {
			req, err := http.NewRequest("GET", u, bytes.NewBuffer([]byte(tt.inp)))
			assert.Nil(t, err)

			r, err := client.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, tt.code, r.StatusCode)
		})
	}
}

func TestBlackwordsFn(t *testing.T) {
	tbl := []struct {
		inp  string
		code int
	}{
		{"", 200},
		{"blah blah body", 200},
		{"blah blah body bad1", 403},
		{"blah blah body bad", 200},
		{"blah bad2 body bad", 403},
		{`{"fld": 123, "aa": {"$where": {"aaa": 567}}}`, 403},
	}

	bwMiddleware := BlackWordsFn(func() []string {
		return []string{"bad1", "bad2", "$where"}
	})

	ts := httptest.NewServer(bwMiddleware(getTestHandlerBlah()))
	defer ts.Close()

	u := fmt.Sprintf("%s%s", ts.URL, "/something")

	client := http.Client{Timeout: 5 * time.Second}

	for n, tt := range tbl {
		tt := tt
		t.Run(fmt.Sprintf("test-%d", n), func(t *testing.T) {
			req, err := http.NewRequest("GET", u, bytes.NewBuffer([]byte(tt.inp)))
			assert.Nil(t, err)

			r, err := client.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, tt.code, r.StatusCode)
		})
	}
}
