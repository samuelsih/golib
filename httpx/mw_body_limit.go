package httpx

import (
	"context"
	"io"
	"net/http"
)

type maxBytesCtx struct{}

var maxBytesCtxKey = maxBytesCtx{}

// MiddlewareMaxBytes is a middleware that limit http read request body.
func MiddlewareMaxBytes(limit int64) Middleware {
	return func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			ctx := r.Context()
			limiter, _ := ctx.Value(maxBytesCtxKey).(*bodyLimitReader)
			if limiter != nil {
				limiter.limit = limit
				return next(w, r)
			}

			limiter = &bodyLimitReader{
				Reader: r.Body,
				Closer: r.Body,
				limit:  limit,
			}

			ctx = context.WithValue(ctx, maxBytesCtxKey, limiter)
			r = r.WithContext(ctx)
			r.Body = limiter

			return next(w, r)
		}
	}
}

type bodyLimitReader struct {
	io.Reader
	io.Closer
	limit int64
	n     int64 // bytes read so far
	err   error // sticky error
}

func (r *bodyLimitReader) Read(p []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}
	if len(p) == 0 {
		return 0, nil
	}

	// Ask the body for at most one byte past the limit.
	remaining := r.limit - r.n + 1
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err = r.Reader.Read(p)
	r.n += int64(n)

	// Within the limit: hand out everything and remember the error.
	if r.n <= r.limit {
		r.err = err
		return n, err
	}

	// Past the limit: hand out only the bytes within the limit and fail.
	n -= int(r.n - r.limit)
	r.err = &http.MaxBytesError{Limit: r.limit}
	return n, r.err
}
