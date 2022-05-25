package cache

import "net/http"

func DefaultKeyGenerator(req *http.Request) string {
	return req.URL.String()
}
