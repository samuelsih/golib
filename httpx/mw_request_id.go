package httpx

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"time"
)

// RequestIDHeader is the HTTP header used to accept and return request IDs.
const RequestIDHeader = "X-Request-ID"

const (
	requestIDMinLength = 20
	requestIDMaxLength = 200
)

type requestIDCtx struct{}

var requestIDCtxKey = requestIDCtx{}

// MiddlewareRequestID puts X-Request-ID to the header and the context.
func MiddlewareRequestID() Middleware {
	return func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			id := requestID(r)
			w.Header().Set(RequestIDHeader, id)

			return next(w, r.WithContext(context.WithValue(r.Context(), requestIDCtxKey, id)))
		}
	}
}

// RequestID returns the request ID set by MiddlewareRequestID, or an empty
// string when the middleware did not run.
func RequestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDCtxKey).(string)
	return id
}

func requestID(r *http.Request) string {
	id := r.Header.Get(RequestIDHeader)
	if len(id) < requestIDMinLength || len(id) > requestIDMaxLength {
		return generateRequestID()
	}
	return id
}

func generateRequestID() string {
	var b [16]byte
	binary.BigEndian.PutUint64(b[:8], uint64(time.Now().UnixNano()))
	rand.Read(b[8:])
	return base64.RawURLEncoding.EncodeToString(b[:])
}
