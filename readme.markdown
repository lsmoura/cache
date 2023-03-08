# Cache

Behaves like a browser storing local copies of retrieved pages, 
sending ETag values when available and respect 304 responses.

This project provides a memory cache provider and a redis cache 
provider. New providers can be added by implementing the 
Provider interface.

A custom http client can also be provided, it just needs to implement
the HttpRequester interface, which requires nothing but a `Do` method.

## Usage

Using the cache is as simple as:

```go
cache.New(memoryprovider.New())
req, err := http.NewRequest("GET", "http://example.com", nil)
if err != nil {
    panic(err)
}
resp, err := cache.Do(req)
if err != nil {
    panic(err)
}
defer resp.Body.Close()

data := io.ReadAll(resp.Body)
```

If the response has proper expiry headers, the cache will prevent extra http
calls to be made.

If the response providers a proper ETag header, the cache will store the response
and return it on subsequent requests and 304 responses will be properly dealt with.

### Storing cached data

The cache can use any data store that implements the `Provider` interface.

```go
type Provider interface {
	// Get returns the value for the given key. Should only return an error if the value could be checked for existence or if communication fails.
	// If the value is not found just return nil, nil.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set sets the value for the given key. Should return an error if the value could not be set.
	Set(ctx context.Context, key string, value []byte, expiry time.Duration) error
}
```

Two providers are provided:

* **memoryprovider** - stores data in memory
* **redisprovider** - takes a redis connection and stores data in redis

### Setting parameters to calls

By modifying the context, the behaviour of the cache can be modified.

* **WithIgnoreExpired** - the cache will return expired parameters without trying to refresh them
* **WithIgnoreCache** - ignores any return values from the cache. Http responses are still cached.
* **WithOnlyCached** - returns only a cached value, if it exists. Returns an `ErrCacheMiss` error if the value is not cached.

# Author

* [Sergio Moura](https://sergio.moura.ca/)
