package rest

import (
	"context"
	"errors"
	"net/http"
)

type contextKey string

type UserInfo interface {
	ID() string
	Name() string
}

// GetUserInfo returns user from request context
func GetUserInfo(r *http.Request) (user UserInfo, err error) {

	ctx := r.Context()
	if ctx == nil {
		return nil, errors.New("no info about user")
	}
	if u, ok := ctx.Value(contextKey("user")).(UserInfo); ok {
		return u, nil
	}

	return nil, errors.New("user can't be parsed")
}

// SetUserInfo sets user into request context
func SetUserInfo(r *http.Request, user UserInfo) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, contextKey("user"), user)
	return r.WithContext(ctx)
}
