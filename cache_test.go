package cache

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/lsmoura/cache/memoryprovider"
)

type fakeRequester struct {
	requestCount int
	data         map[string]*cacheEntry
	requestLog   []*http.Request
}

func (f *fakeRequester) Do(req *http.Request) (*http.Response, error) {
	f.requestCount++
	f.requestLog = append(f.requestLog, req)
	if f.data == nil {
		return nil, errors.New("requester not initialized")
	}

	entry, ok := f.data[req.URL.String()]
	if !ok {
		return &http.Response{StatusCode: http.StatusNotFound}, nil
	}

	return entry.asHttpResponse(req), nil
}

func TestCache_Do(t *testing.T) {
	const regularURL = "http://example.com/"
	const expiredURL = "http://example.com/alwaysExpired"
	const nonExistingURL = "http://example.com/nonExisting"

	ctx := context.Background()

	requester := fakeRequester{
		data: map[string]*cacheEntry{
			regularURL: {
				Ts:         time.Now(),
				StatusCode: 200,
				Data:       []byte("Hello World"),
				Headers: map[string]string{
					"Expires": time.Now().Add(time.Hour).Format(time.RFC1123),
				},
			},
			expiredURL: {
				Ts:         time.Now(),
				StatusCode: 200,
				Data:       []byte("Hello World"),
				Headers: map[string]string{
					"Expires": time.Now().Add(-time.Hour).Format(time.RFC1123),
				},
			},
		},
	}
	cache := New(memoryprovider.New())
	cache.HttpClient = &requester

	t.Run("regular request", func(t *testing.T) {
		req, err := http.NewRequest("GET", regularURL, nil)
		require.NoError(t, err, "http.NewRequest")

		res, err := cache.Do(req)
		require.NoError(t, err, "cache.Do")

		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(res.Body)

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err, "io.ReadAll")

		require.Equal(t, "Hello World", string(body), "Expected body to be Hello World")
	})

	t.Run("cached request", func(t *testing.T) {
		initialCount := requester.requestCount
		// same request should hit cache
		req, err := http.NewRequest("GET", regularURL, nil)
		require.NoError(t, err, "http.NewRequest()")

		_, err = cache.Do(req)
		require.NoError(t, err, "cache.Do()")

		require.Equalf(t, initialCount, requester.requestCount, "Expected request count to be %d", initialCount)
	})

	t.Run("ignoring cache should increase request count", func(t *testing.T) {
		initialCount := requester.requestCount

		// set ignore cache and expect request count to increase
		req, err := http.NewRequestWithContext(WithIgnoreCache(ctx, true), "GET", regularURL, nil)
		require.NoError(t, err, "http.NewRequestWithContext")

		_, err = cache.Do(req)
		require.NoError(t, err, "cache.Do")

		require.Equalf(t, initialCount+1, requester.requestCount, "Expected request count to be %d", initialCount+1)
	})

	t.Run("ignore expired", func(t *testing.T) {
		initialCount := requester.requestCount
		reqCtx := WithIgnoreExpired(ctx, true)

		req, err := http.NewRequestWithContext(ctx, "GET", expiredURL, nil)
		require.NoError(t, err, "http.NewRequestWithContext")

		// request counter should increase, because it's not cached yet
		_, err = cache.Do(req.Clone(reqCtx))
		require.NoError(t, err, "cache.Do")
		require.Equalf(t, initialCount+1, requester.requestCount, "Expected request count to increase and be %d", initialCount+1)

		// now, even though the entry is expired, it should be returned from cache
		_, err = cache.Do(req.Clone(reqCtx))
		require.NoError(t, err, "cache.Do")
		require.Equal(t, initialCount+1, requester.requestCount, "Expected request count not to increase and be %d", initialCount+1)
	})

	t.Run("only cached", func(t *testing.T) {
		// if we request the cache to never make the request and the entry does not exist, should return an error
		req, err := http.NewRequestWithContext(WithOnlyCached(ctx, true), "GET", nonExistingURL, nil)
		require.NoError(t, err, "http.NewRequestWithContext")

		if _, err := cache.Do(req); !errors.Is(err, ErrCacheMiss) {
			t.Fatalf("Expected error to be ErrCacheMiss, got %v", err)
		}
	})
}

func TestCache_Etag(t *testing.T) {
	const cacheURL = "http://example.com/"
	const etag = "\"123456789\""

	requester := fakeRequester{
		data: map[string]*cacheEntry{
			cacheURL: {
				Ts:         time.Now(),
				StatusCode: 200,
				Data:       []byte("Hello World"),
				Headers: map[string]string{
					"Expires": time.Now().Add(-time.Hour).Format(time.RFC1123),
					"ETag":    etag,
				},
			},
		},
	}

	cache := New(memoryprovider.New())
	cache.HttpClient = &requester

	req, err := http.NewRequest("GET", cacheURL, nil)
	require.NoError(t, err, "http.NewRequest")
	_, err = cache.Do(req)
	require.NoError(t, err, "cache.Do")

	req, err = http.NewRequest("GET", cacheURL, nil)
	require.NoError(t, err, "http.NewRequest")
	_, err = cache.Do(req)
	require.NoError(t, err, "cache.Do")

	require.Equal(t, 2, len(requester.requestLog))

	lastReq := requester.requestLog[len(requester.requestLog)-1]
	ifNoneMatchHeader, ok := lastReq.Header["If-None-Match"]
	require.False(t, !ok, "Expected If-None-Match header to be set")
	require.Equalf(t, ifNoneMatchHeader[0], etag, "Expected If-None-Match header to match etag value")
}

func TestCache_Keygen(t *testing.T) {
	const cacheURL1 = "http://example.com/1"
	const cacheURL2 = "http://example.com/2"
	const etag = "\"123456789\""

	requester := fakeRequester{
		data: map[string]*cacheEntry{
			cacheURL1: {
				Ts:         time.Now(),
				StatusCode: 200,
				Data:       []byte("Hello World"),
				Headers: map[string]string{
					"Expires": time.Now().Add(time.Hour).Format(time.RFC1123),
					"ETag":    etag,
				},
			},
			cacheURL2: {
				Ts:         time.Now(),
				StatusCode: 200,
				Data:       []byte("Should not happen"),
				Headers: map[string]string{
					"Expires": time.Now().Add(time.Hour).Format(time.RFC1123),
					"ETag":    "abcd",
				},
			},
		},
	}

	cache := New(memoryprovider.New())
	cache.HttpClient = &requester
	cache.KeyGenerator = func(req *http.Request) string {
		return "foo"
	}

	req, err := http.NewRequest(http.MethodGet, cacheURL1, nil)
	require.NoError(t, err, "http.NewRequest")
	_, err = cache.Do(req)
	require.NoError(t, err, "cache.Do")

	req, err = http.NewRequest(http.MethodGet, cacheURL2, nil)
	require.NoError(t, err, "http.NewRequest")
	_, err = cache.Do(req)
	require.NoError(t, err, "cache.Do")

	require.Equal(t, 1, len(requester.requestLog), "Request count mismatch")
}

func TestCache_Etag304(t *testing.T) {
	const cacheURL = "http://example.com/"
	const etag = "\"123456789\""

	requester := fakeRequester{
		data: map[string]*cacheEntry{
			cacheURL: {
				Ts:         time.Now(),
				StatusCode: 200,
				Data:       []byte("Hello World"),
				Headers: map[string]string{
					"Expires": time.Now().Add(-time.Hour).Format(time.RFC1123),
					"ETag":    etag,
				},
			},
		},
	}

	provider := memoryprovider.New()

	cache := New(provider)
	cache.HttpClient = &requester

	req, err := http.NewRequest(http.MethodGet, cacheURL, nil)
	require.NoError(t, err, "http.NewRequest")
	_, err = cache.Do(req)
	require.NoError(t, err, "cache.Do")

	requester.data[cacheURL].StatusCode = http.StatusNotModified
	requester.data[cacheURL].Data = []byte("")

	req, err = http.NewRequest(http.MethodGet, cacheURL, nil)
	require.NoError(t, err, "http.NewRequest")

	resp, err := cache.Do(req)
	require.NoError(t, err, "cache.Do")

	require.Equalf(t, len(requester.requestLog), 2, "Expected request log to have 2 entries, got %d", len(requester.requestLog))

	lastReq := requester.requestLog[len(requester.requestLog)-1]
	ifNoneMatchHeader, ok := lastReq.Header["If-None-Match"]
	require.True(t, ok, "Expected If-None-Match header to be set")
	require.Equalf(t, ifNoneMatchHeader[0], etag, "Expected If-None-Match header to be %s, got %s", etag, ifNoneMatchHeader[0])
	require.Equalf(t, resp.StatusCode, http.StatusNotModified, "Expected status code to be %d, got %d", http.StatusNotModified, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "io.ReadAll")
	require.Equal(t, "Hello World", string(body))
}
