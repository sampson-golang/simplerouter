package simplerouter

import (
	"net/http"
	"slices"
	"strings"
)

type (
	middleware func(http.Handler) http.Handler
	Router     struct {
		mux   *muxWrapper
		chain []middleware
	}
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

func toStatusInterceptor(w http.ResponseWriter, r *http.Request) *statusInterceptor {
	if si, ok := w.(*statusInterceptor); ok {
		return si
	}
	return &statusInterceptor{
		ResponseWriter: w,
		originalPath:   r.URL.Path,
	}
}

type muxWrapper struct {
	*http.ServeMux
	httpHandler     middleware
	notFoundHandler http.Handler
	rootPath        string
}

func (m *muxWrapper) fullPattern(pattern string) string {
	if m.rootPath == "" {
		return pattern
	}

	if i := strings.IndexAny(pattern, " \t"); i >= 0 {
		method, path := pattern[:i], strings.TrimLeft(pattern[i:], " \t")
		return method + " " + m.rootPath + path
	}

	return m.rootPath + pattern
}

func (m *muxWrapper) Handle(pattern string, handler http.Handler) {
	pattern = m.fullPattern(pattern)
	logger.Debug("Handle", "pattern", pattern)
	m.ServeMux.Handle(pattern, handler)

	if strings.Contains(pattern, "-") {
		logger.Debug("Handle -", "pattern", strings.ReplaceAll(pattern, "-", "_"))
		m.ServeMux.Handle(strings.ReplaceAll(pattern, "-", "_"), handler)
	} else if strings.Contains(pattern, "_") {
		logger.Debug("Handle _", "pattern", strings.ReplaceAll(pattern, "_", "-"))
		m.ServeMux.Handle(strings.ReplaceAll(pattern, "_", "-"), handler)
	}
}

func (m *muxWrapper) HandleFunc(pattern string, handler http.HandlerFunc) {
	pattern = m.fullPattern(pattern)

	m.ServeMux.HandleFunc(pattern, handler)

	if strings.Contains(pattern, "-") {
		m.ServeMux.HandleFunc(strings.ReplaceAll(pattern, "-", "_"), handler)
	} else if strings.Contains(pattern, "_") {
		m.ServeMux.HandleFunc(strings.ReplaceAll(pattern, "_", "-"), handler)
	}
}

func (m *muxWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Handling Path", "path", r.URL.Path)
	w = toStatusInterceptor(w, r)

	if m.notFoundHandler != nil {
		_, matchedPattern := m.ServeMux.Handler(r)
		if matchedPattern == "" {
			m.notFoundHandler.ServeHTTP(w, r)
			return
		}
	}

	if m.httpHandler != nil {
		m.httpHandler(m.ServeMux).ServeHTTP(w, r)
	} else {
		m.ServeMux.ServeHTTP(w, r)
	}
}

func newMuxWrapper(paths ...string) *muxWrapper {
	return &muxWrapper{ServeMux: http.NewServeMux(), rootPath: buildRootPath(paths...)}
}

func buildRootPath(paths ...string) string {
	path := ""
	for _, p := range paths {
		if p == "" || p == "/" {
			continue
		}
		p = strings.TrimSuffix(strings.TrimPrefix(p, "/"), "/")
		path = path + "/" + p
	}
	return path
}

func NewRouter(chain ...middleware) *Router {
	return &Router{mux: newMuxWrapper(), chain: chain}
}

func (r *Router) SetHandler(handler middleware) {
	r.mux.httpHandler = handler
}

func (r *Router) SetNotFoundHandler(handler http.Handler) {
	r.mux.notFoundHandler = handler
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) BasePath() string {
	return r.mux.rootPath
}

func (r *Router) SetBasePath(path string) {
	r.mux.rootPath = path
}

func (r *Router) AppendPath(path string) {
	r.mux.rootPath = buildRootPath(r.mux.rootPath, path)
}

// Use appends one or more middlewares onto the Router stack.
//
// The middleware stack for any Mux will execute before searching for a matching
// route to a specific handler, which provides opportunity to respond early,
// change the course of the request execution, or set request-scoped values for
// the next http.Handler.
func (r *Router) Use(chain ...middleware) {
	r.chain = append(r.chain, chain...)
}

// Creates a sub-router with the a cloned middleware stack.
// this router uses the same ServeMux as the parent router, but the middleware
// stack is independent of external changes to the parent router.
func (r *Router) Group(fn func(r *Router)) {
	fn(&Router{mux: r.mux, chain: slices.Clone(r.chain)})
}

// Creates a sub-router with the a cloned middleware stack.
// unlike Group, this router creates a new sub-mux
func (r *Router) Route(path string, fn func(r *Router), chain ...middleware) *Router {
	subRouter := &Router{
		mux: &muxWrapper{
			ServeMux:        http.NewServeMux(),
			rootPath:        buildRootPath(r.mux.rootPath, path),
			notFoundHandler: r.mux.notFoundHandler,
		},
		chain: chain,
	}

	if fn != nil {
		fn(subRouter)
	}

	r.Mount(path, subRouter)

	return subRouter
}

func (r *Router) Mount(path string, h http.Handler, chain ...middleware) {
	path = strings.TrimSuffix(path, "/") + "/"

	r.mux.Handle(path, r.wrap(h.ServeHTTP, chain))
}

func (r *Router) Get(path string, fn http.HandlerFunc, chain ...middleware) {
	r.handle(http.MethodGet, path, fn, chain)
}

func (r *Router) Post(path string, fn http.HandlerFunc, chain ...middleware) {
	r.handle(http.MethodPost, path, fn, chain)
}

func (r *Router) Put(path string, fn http.HandlerFunc, chain ...middleware) {
	r.handle(http.MethodPut, path, fn, chain)
}

func (r *Router) Delete(path string, fn http.HandlerFunc, chain ...middleware) {
	r.handle(http.MethodDelete, path, fn, chain)
}

func (r *Router) Head(path string, fn http.HandlerFunc, chain ...middleware) {
	r.handle(http.MethodHead, path, fn, chain)
}

func (r *Router) Options(path string, fn http.HandlerFunc, chain ...middleware) {
	r.handle(http.MethodOptions, path, fn, chain)
}

func (r *Router) Any(path string, fn http.HandlerFunc, chain ...middleware) {
	r.mux.Handle(path, r.wrap(fn, chain))
}

// allow dynamic methods
func (r *Router) Handle(method, path string, fn http.HandlerFunc, chain ...middleware) {
	r.handle(method, path, fn, chain)
}

func (r *Router) NotFound(writer http.ResponseWriter, req *http.Request) {
	if r.mux.notFoundHandler != nil {
		r.mux.notFoundHandler.ServeHTTP(writer, req)
	} else {
		writer.WriteHeader(http.StatusNotFound)
		writer.Write([]byte(`{"error": "Not Found"}`))
	}
}

func (r *Router) handle(method, path string, fn http.HandlerFunc, chain []middleware) {
	r.mux.Handle(method+" "+path, r.wrap(fn, chain))
}

func (r *Router) wrap(fn http.HandlerFunc, chain []middleware) (out http.Handler) {
	out = http.Handler(fn)

	for idx := len(chain) - 1; idx >= 0; idx-- {
		out = chain[idx](out)
	}

	for idx := len(r.chain) - 1; idx >= 0; idx-- {
		out = r.chain[idx](out)
	}

	return out
}
