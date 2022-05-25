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

```
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

# Author

* [Sergio Moura](https://sergio.moura.ca/)
