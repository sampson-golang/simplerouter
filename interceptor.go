package simplerouter

import (
	"bufio"
	"net"
	"net/http"
)

type statusInterceptor struct {
	http.ResponseWriter
	originalPath string
	Status int
}

func (wrapper *statusInterceptor) WriteHeader(code int) {
	if code == http.StatusMovedPermanently {
		location := wrapper.Header().Get("Location")
		if location == wrapper.originalPath + "/" {
			wrapper.Status = http.StatusTemporaryRedirect
			wrapper.ResponseWriter.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
	}
	wrapper.ResponseWriter.WriteHeader(code)
}

// Hijacker interface support
func (wrapper *statusInterceptor) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := wrapper.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Flusher interface support
func (wrapper *statusInterceptor) Flush() {
	if flusher, ok := wrapper.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Pusher interface support
func (wrapper *statusInterceptor) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := wrapper.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func toStatusInterceptor(w http.ResponseWriter, r *http.Request) *statusInterceptor {
	if si, ok := w.(*statusInterceptor); ok {
		return si
	}
	return &statusInterceptor{
		ResponseWriter: w,
		originalPath:   r.URL.Path,
	}
}
