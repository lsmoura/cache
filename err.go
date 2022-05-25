package cache

import "errors"

var (
	ErrCacheExpired       = errors.New("cache expired")
	ErrCacheExpiryIgnored = errors.New("cache expiry ignored")
	ErrCacheMiss          = errors.New("cache miss")
)
