package oidc

import "context"

// contextKey is the key for oidc claims in a context
type contextKey struct{}

// newSessionFlagKey is the key for the new session flag in a context
type newSessionFlagKey struct{}

// NewContext makes a new context that contains the OpenID connect claims in a map.
func NewContext(parent context.Context, c map[string]any) context.Context {
	return context.WithValue(parent, contextKey{}, c)
}

// FromContext returns the claims map stored in a context, or nil if there isn't one.
func FromContext(ctx context.Context) map[string]any {
	s, _ := ctx.Value(contextKey{}).(map[string]any)
	return s
}

// NewContextSessionFlag makes a new context that contains the new session flag.
func NewContextSessionFlag(ctx context.Context, flag bool) context.Context {
	return context.WithValue(ctx, newSessionFlagKey{}, flag)
}

// NewSessionFlagFromContext returns the new session flag stored in a context.
func NewSessionFlagFromContext(ctx context.Context) bool {
	s, _ := ctx.Value(newSessionFlagKey{}).(bool)
	return s
}
