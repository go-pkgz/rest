// Package requestid keeps the request id Trace sets in the request context, so the logger can read it
// without importing the rest package, which imports the logger.
package requestid

import "context"

type ctxKey struct{}

// With returns a copy of ctx carrying id
func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// From returns the id stored in ctx, or an empty string
func From(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
