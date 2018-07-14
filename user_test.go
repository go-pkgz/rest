package rest

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type uinfo struct {
	id   string
	name string
}

func (u uinfo) ID() string   { return u.id }
func (u uinfo) Name() string { return u.name }

func TestGetUserInfo(t *testing.T) {
	r, err := http.NewRequest("GET", "http://blah.com", nil)
	assert.Nil(t, err)
	_, err = GetUserInfo(r)
	assert.NotNil(t, err, "no user info")

	r = SetUserInfo(r, uinfo{name: "test", id: "id"})
	u, err := GetUserInfo(r)
	assert.Nil(t, err)
	assert.Equal(t, uinfo{name: "test", id: "id"}, u)
}
