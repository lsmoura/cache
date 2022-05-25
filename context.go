package cache

import "context"

type contextKey string

const (
	contextKeyIgnoreExpired contextKey = "contextKeyIgnoreExpired"
	contextKeyIgnoreCache   contextKey = "contextKeyIgnoreCache"
	contextKeyOnlyCached    contextKey = "contextKeyOnlyCached"
)

// WithIgnoreExpired returns a copy of parent context with ignoreExpired flag set to the given parameter.
// If this flag is set to true, the cache will return expired parameters without trying to refresh them.
func WithIgnoreExpired(ctx context.Context, shouldIgnore bool) context.Context {
	return context.WithValue(ctx, contextKeyIgnoreExpired, shouldIgnore)
}

func IgnoreExpired(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v := ctx.Value(contextKeyIgnoreExpired)
	if v == nil {
		return false
	}
	return v.(bool)
}

// WithIgnoreCache ignores any return values from the cache. Http responses are still cached.
// Takes precedence over IgnoreExpired.
func WithIgnoreCache(ctx context.Context, shouldIgnore bool) context.Context {
	return context.WithValue(ctx, contextKeyIgnoreCache, shouldIgnore)
}

func IgnoreCache(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v := ctx.Value(contextKeyIgnoreCache)
	if v == nil {
		return false
	}
	return v.(bool)
}

// WithOnlyCached returns only a cached value, if it exists. Returns an ErrCacheMiss error if the value is not cached.
func WithOnlyCached(ctx context.Context, shouldOnly bool) context.Context {
	return context.WithValue(ctx, contextKeyOnlyCached, shouldOnly)
}
func OnlyCached(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v := ctx.Value(contextKeyOnlyCached)
	if v == nil {
		return false
	}
	return v.(bool)
}
