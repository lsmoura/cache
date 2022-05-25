package cache

import (
	"context"
	"errors"
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
	provider := memoryprovider.New()

	cache := New(provider)
	cache.HttpClient = &requester

	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := cache.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "Hello World" {
		t.Fatal("Expected body to be Hello World")
	}

	// same request should hit cache
	req, err = http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err = cache.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if requester.requestCount != 1 {
		t.Fatal("Expected request count to be 1")
	}

	// set ignore cache and expect request count to increase
	req, err = http.NewRequestWithContext(WithIgnoreCache(context.Background(), true), "GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err = cache.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if requester.requestCount != 2 {
		t.Fatalf("Expected request count to be 2, got %d", requester.requestCount)
	}

	// if requesting an expired URL with the ignore expired flag, the request count shouldn't increase, unless cache has not cached the entry yet
	countBefore := requester.requestCount
	req, err = http.NewRequestWithContext(WithIgnoreExpired(context.Background(), true), "GET", expiredURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err = cache.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if requester.requestCount != countBefore+1 {
		t.Fatalf("Expected request count to be %d, got %d", countBefore, requester.requestCount)
	}

	// do it again, this time it should be cached
	req, err = http.NewRequestWithContext(WithIgnoreExpired(context.Background(), true), "GET", expiredURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err = cache.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if requester.requestCount != countBefore+1 {
		t.Fatalf("Expected request count to be %d, got %d", countBefore, requester.requestCount)
	}

	// if we request the cache to never make the request and the entry does not exists, should return an error
	req, err = http.NewRequestWithContext(WithOnlyCached(context.Background(), true), "GET", nonExistingURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = cache.Do(req)
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("Expected error to be ErrCacheMiss, got %v", err)
	}
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

	provider := memoryprovider.New()

	cache := New(provider)
	cache.HttpClient = &requester

	req, err := http.NewRequest("GET", cacheURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Do(req); err != nil {
		t.Fatal(err)
	}

	req, err = http.NewRequest("GET", cacheURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := cache.Do(req); err != nil {
		t.Fatal(err)
	}

	if len(requester.requestLog) != 2 {
		t.Fatalf("Expected request log to have 2 entries, got %d", len(requester.requestLog))
	}

	lastReq := requester.requestLog[len(requester.requestLog)-1]
	ifNoneMatchHeader, ok := lastReq.Header["If-None-Match"]
	if !ok {
		t.Fatal("Expected If-None-Match header to be set")
	}
	if ifNoneMatchHeader[0] != etag {
		t.Fatalf("Expected If-None-Match header to be %s, got %s", etag, ifNoneMatchHeader[0])
	}
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

	if req, err := http.NewRequest("GET", cacheURL1, nil); err != nil {
		t.Fatal(err)
	} else if _, err := cache.Do(req); err != nil {
		t.Fatal(err)
	}

	if req, err := http.NewRequest("GET", cacheURL2, nil); err != nil {
		t.Fatal(err)
	} else if _, err := cache.Do(req); err != nil {
		t.Fatal(err)
	}

	if len(requester.requestLog) != 1 {
		t.Fatalf("Expected request log to have 1 entry, got %d", len(requester.requestLog))
	}
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

	req, err := http.NewRequest("GET", cacheURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Do(req); err != nil {
		t.Fatal(err)
	}

	requester.data[cacheURL].StatusCode = http.StatusNotModified
	requester.data[cacheURL].Data = []byte("")

	req, err = http.NewRequest("GET", cacheURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := cache.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if len(requester.requestLog) != 2 {
		t.Fatalf("Expected request log to have 2 entries, got %d", len(requester.requestLog))
	}

	lastReq := requester.requestLog[len(requester.requestLog)-1]
	ifNoneMatchHeader, ok := lastReq.Header["If-None-Match"]
	if !ok {
		t.Fatal("Expected If-None-Match header to be set")
	}
	if ifNoneMatchHeader[0] != etag {
		t.Fatalf("Expected If-None-Match header to be %s, got %s", etag, ifNoneMatchHeader[0])
	}
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("Expected status code to be %d, got %d", http.StatusNotModified, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "Hello World" {
		t.Fatalf("Expected body to be '\"Hello World\", got %s", body)
	}
}
