package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type contextKey string

// GetUserInfo returns user from request context
func GetUserInfo(r *http.Request) (user fmt.Stringer, err error) {

	ctx := r.Context()
	if u, ok := ctx.Value(contextKey("user")).(fmt.Stringer); ok {
		return u, nil
	}

	return nil, errors.New("user can't extracted from ctx")
}

// SetUserInfo sets user into request context
func SetUserInfo(r *http.Request, user fmt.Stringer) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, contextKey("user"), user)
	return r.WithContext(ctx)
}
