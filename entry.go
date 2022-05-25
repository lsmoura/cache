package cache

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

type cacheEntry struct {
	Ts         time.Time         `json:"ts"`
	StatusCode int               `json:"status_code"`
	Data       []byte            `json:"data"`
	Headers    map[string]string `json:"headers"`
}

func (e cacheEntry) asHttpResponse(req *http.Request) *http.Response {
	headers := make(map[string][]string)
	for k, v := range e.Headers {
		headers[k] = []string{v}
	}

	return &http.Response{
		StatusCode:    e.StatusCode,
		Body:          io.NopCloser(bytes.NewReader(e.Data)),
		Header:        headers,
		Request:       req,
		ContentLength: int64(len(e.Data)),
	}
}

// expired returns true if the entry is expired.
func (e cacheEntry) expired() bool {
	expiry, ok := e.Headers["Expires"]
	if !ok {
		return true
	}

	expires, err := time.Parse(time.RFC1123, expiry)
	if err != nil {
		return true
	}

	return expires.Before(time.Now())
}
