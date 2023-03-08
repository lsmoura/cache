package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HttpRequester interface {
	Do(req *http.Request) (*http.Response, error)
}

type KeyGenerator func(req *http.Request) string

type Cache struct {
	HttpClient   HttpRequester // custom http client provider, or nil for http.DefaultClient
	KeyGenerator KeyGenerator  // custom key generator, or nil for default
	provider     Provider

	LogExtractor LoggerExtractor
}

type cacheStat string

const (
	cacheStatExpired       cacheStat = "expired"
	cacheStatHit           cacheStat = "hit"
	cacheStatIgnoreCheck   cacheStat = "ignored_check"
	cacheStatIgnored       cacheStat = "ignored"
	cacheStatIgnoredExpiry cacheStat = "ignored_expiry"
	cacheStatMiss          cacheStat = "miss"
)

func New(provider Provider) *Cache {
	return &Cache{
		provider: provider,
	}
}

func (r Cache) httpClient() HttpRequester {
	if r.HttpClient == nil {
		return http.DefaultClient
	}
	return r.HttpClient
}

func (r Cache) read(ctx context.Context, key string) (*cacheEntry, error) {
	value, err := r.provider.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("provider.Get(): %w", err)
	}

	if len(value) == 0 {
		return nil, nil
	}

	var entry cacheEntry
	if err := json.Unmarshal(value, &entry); err != nil {
		fmt.Println("error unmarshalling cache entry:", err)
		return nil, nil
	}

	if entry.expired() {
		if IgnoreExpired(ctx) {
			return &entry, ErrCacheExpiryIgnored
		}
		return &entry, ErrCacheExpired
	}
	return &entry, nil
}

func (r Cache) write(ctx context.Context, key string, entry *cacheEntry) error {
	dataBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("json.Marshal(): %w", err)
	}

	// TODO: optionally retrieve the expiration from the headers
	// TODO: optionally retrieve the expiration from the context
	if err := r.provider.Set(ctx, key, dataBytes, 0); err != nil {
		return fmt.Errorf("provider.Set(): %w", err)
	}
	return nil
}

func (r Cache) store(ctx context.Context, key string, resp *http.Response) (*cacheEntry, error) {
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			r.logInfo(ctx, "error closing response body", "error", err)
		}
	}(resp.Body)
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll(): %w", err)
	}

	e := cacheEntry{
		Ts:         time.Now(),
		StatusCode: resp.StatusCode,
		Data:       data,
		Headers:    make(map[string]string),
	}
	for k, v := range resp.Header {
		e.Headers[k] = v[0]
	}

	if err := r.write(ctx, key, &e); err != nil {
		return nil, fmt.Errorf("r.write(): %w", err)
	}

	return &e, nil
}

func (r Cache) key(req *http.Request) string {
	if r.KeyGenerator == nil {
		return DefaultKeyGenerator(req)
	}
	return r.KeyGenerator(req)
}

func (r Cache) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	event := r.logger(ctx)
	event = event.With("url", req.URL.String())
	var stat cacheStat
	defer func() {
		if stat != "" {
			event = event.With("cache", stat)
		}
		event.Info("cache.Do")
	}()
	if req.Method != http.MethodGet {
		return r.httpClient().Do(req)
	}

	key := r.key(req)
	event = event.With("cache-key", key)

	var entry *cacheEntry

	if IgnoreCache(ctx) {
		stat = cacheStatIgnored
	} else {
		var err error
		entry, err = r.read(ctx, key)
		if err != nil {
			if errors.Is(err, ErrCacheExpired) {
				stat = cacheStatExpired
			} else if errors.Is(err, ErrCacheExpiryIgnored) {
				stat = cacheStatIgnoredExpiry
				return entry.asHttpResponse(req), nil
			} else {
				event.Error("error", "err", err)
				return nil, err
			}
		} else if entry != nil {
			stat = cacheStatHit
			return entry.asHttpResponse(req), nil
		} else {
			stat = cacheStatMiss
		}
	}

	if OnlyCached(ctx) {
		stat = cacheStatIgnoreCheck
		if entry == nil {
			return nil, ErrCacheMiss
		}
		return entry.asHttpResponse(req), nil
	}

	if entry != nil {
		// find ETAG
		etag, ok := entry.Headers["ETag"]
		if ok && etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
	}

	start := time.Now()
	resp, err := r.httpClient().Do(req)
	if err != nil {
		event.Error("error", "err", err)
		return nil, fmt.Errorf("http.Do(): %w", err)
	}
	event = event.With("elapsed", time.Since(start))
	event = event.With("status", resp.StatusCode)

	if resp.StatusCode == http.StatusNotModified {
		// update expires and last-modified
		if entry == nil {
			// we don't have any data to use as "not modified"
			err := errors.New("no cached entry to return")
			event.Error("error", "err", err)
			return nil, err
		}
		if expires, ok := entry.Headers["Expires"]; ok {
			resp.Header.Set("Expires", expires)
		}
		if lastModified, ok := entry.Headers["Last-Modified"]; ok {
			resp.Header.Set("Last-Modified", lastModified)
		}
		if err := r.write(ctx, key, entry); err != nil {
			event.Error("error", "err", err)
		}

		resp.Body = io.NopCloser(bytes.NewReader(entry.Data))

		return resp, nil
	}

	e, err := r.store(ctx, key, resp)
	if err != nil {
		event.Error("error", "err", err)
		return nil, fmt.Errorf("r.store(): %w", err)
	}

	return e.asHttpResponse(req), nil
}
