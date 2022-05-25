package cache

import (
	"context"
	"time"
)

type Provider interface {
	// Get returns the value for the given key. Should only return an error if the value could be checked for existence or if communication fails.
	// If the value is not found just return nil, nil.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set sets the value for the given key. Should return an error if the value could not be set.
	Set(ctx context.Context, key string, value []byte, expiry time.Duration) error
}
