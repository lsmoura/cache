package cache

import (
	"bytes"
	"context"
	"fmt"
	"github.com/lsmoura/cache/memoryprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fakeLogger struct {
	buf    *bytes.Buffer
	params []any
}

func (f *fakeLogger) Write(p []byte) (n int, err error) {
	return f.buf.Write(p)
}

func (f *fakeLogger) WriteString(s string) (n int, err error) {
	return f.buf.WriteString(s)
}

func (f *fakeLogger) String() string {
	return f.buf.String()
}

func (f *fakeLogger) Reset() {
	f.buf.Reset()
	f.params = nil
}

func (l *fakeLogger) log(level logLevel, msg string, params ...any) {
	p := append(l.params, params...)

	pieces := []string{msg}

	var key *string
	for _, param := range p {
		if key == nil {
			s, ok := param.(string)
			if !ok {
				panic("invalid key")
			}

			if strings.Contains(s, "\"") {
				s = fmt.Sprintf("%q", s)
			}

			key = &s
		} else {
			pieces = append(pieces, fmt.Sprintf("%s=%v", *key, param))
			key = nil
		}
	}

	_, err := fmt.Fprintf(l.buf, "%s %s\n", level, strings.Join(pieces, " "))
	if err != nil {
		panic(err)
	}
}

func (l *fakeLogger) Debug(msg string, params ...any) {
	l.log(logLevelDebug, msg, params...)
}
func (l *fakeLogger) Info(msg string, params ...any) {
	l.log(logLevelInfo, msg, params...)
}
func (l *fakeLogger) Error(msg string, params ...any) {
	l.log(logLevelError, msg, params...)
}
func (l *fakeLogger) With(params ...any) Logger {
	return &fakeLogger{
		buf:    l.buf,
		params: append(l.params, params...),
	}
}

func TestLogging(t *testing.T) {
	const regularURL = "http://example.com/"
	const expiredURL = "http://example.com/alwaysExpired"

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

	logger := fakeLogger{buf: &bytes.Buffer{}}

	cache.LogExtractor = func(context.Context) Logger { return &logger }

	t.Run("regular request", func(t *testing.T) {
		req, err := http.NewRequest("GET", regularURL, nil)
		require.NoError(t, err)

		_, err = cache.Do(req)
		require.NoError(t, err)

		assert.Contains(t, logger.String(), "cache=miss")
		logger.Reset()

		_, err = cache.Do(req)
		require.NoError(t, err)

		assert.Contains(t, logger.String(), "cache=hit")
		logger.Reset()
	})

	t.Run("cached only", func(t *testing.T) {
		req, err := http.NewRequest("GET", expiredURL, nil)
		require.NoError(t, err)

		_, err = cache.Do(req.WithContext(WithOnlyCached(req.Context(), true)))
		assert.Contains(t, logger.String(), "cache=ignored_check")

		fmt.Println(logger.String())
		logger.Reset()
	})
}
