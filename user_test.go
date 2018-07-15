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

func (u uinfo) String() string { return u.name + "/" + u.id }

func TestUser_GetAndSet(t *testing.T) {
	r, err := http.NewRequest("GET", "http://blah.com", nil)
	assert.Nil(t, err)
	_, err = GetUserInfo(r)
	assert.NotNil(t, err, "no user info")

	r = SetUserInfo(r, uinfo{name: "test", id: "id"})
	u, err := GetUserInfo(r)
	assert.Nil(t, err)
	assert.Equal(t, uinfo{name: "test", id: "id"}, u)
}
