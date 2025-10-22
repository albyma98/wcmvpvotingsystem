package chi

import (
	"context"
	"net/http"
	"strings"
	"sync"
)

// Router represents the minimal chi router interface used in the project.
type Router interface {
	http.Handler
	Get(pattern string, handler http.HandlerFunc)
	Post(pattern string, handler http.HandlerFunc)
	Put(pattern string, handler http.HandlerFunc)
	Delete(pattern string, handler http.HandlerFunc)
}

// Mux is a lightweight HTTP multiplexer providing chi-compatible routing
// features required by the application.
type Mux struct {
	mu        sync.RWMutex
	routes    []route
	notFound  http.Handler
	methodNot http.Handler
}

type route struct {
	method  string
	pattern []segment
	handler http.HandlerFunc
}

type segment struct {
	literal string
	param   string
	isParam bool
}

type contextKey struct{}

// paramsKey is used to store URL parameters in the request context.
var paramsKey contextKey

// NewRouter creates a new chi-compatible router implementation.
func NewRouter() *Mux {
	return &Mux{
		notFound:  http.NotFoundHandler(),
		methodNot: http.HandlerFunc(methodNotAllowed),
	}
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
}

// ServeHTTP matches the incoming request against the registered routes and
// dispatches it to the appropriate handler.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	routes := append([]route(nil), m.routes...)
	notFound := m.notFound
	methodNot := m.methodNot
	m.mu.RUnlock()

	var allowed bool
	for _, rt := range routes {
		if !strings.EqualFold(r.Method, rt.method) {
			if strings.EqualFold(rt.method, http.MethodGet) && r.Method == http.MethodHead {
				allowed = true
			}
			continue
		}

		params, ok := matchPattern(rt.pattern, r.URL.Path)
		if !ok {
			continue
		}

		ctx := context.WithValue(r.Context(), paramsKey, params)
		rt.handler.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	if !allowed {
		allowed = hasAllowedMethod(routes, r.URL.Path)
	}

	if allowed {
		methodNot.ServeHTTP(w, r)
		return
	}

	notFound.ServeHTTP(w, r)
}

func hasAllowedMethod(routes []route, path string) bool {
	for _, rt := range routes {
		if _, ok := matchPattern(rt.pattern, path); ok {
			return true
		}
	}
	return false
}

// Get registers a handler for HTTP GET requests.
func (m *Mux) Get(pattern string, handler http.HandlerFunc) {
	m.add(http.MethodGet, pattern, handler)
}

// Post registers a handler for HTTP POST requests.
func (m *Mux) Post(pattern string, handler http.HandlerFunc) {
	m.add(http.MethodPost, pattern, handler)
}

// Put registers a handler for HTTP PUT requests.
func (m *Mux) Put(pattern string, handler http.HandlerFunc) {
	m.add(http.MethodPut, pattern, handler)
}

// Delete registers a handler for HTTP DELETE requests.
func (m *Mux) Delete(pattern string, handler http.HandlerFunc) {
	m.add(http.MethodDelete, pattern, handler)
}

func (m *Mux) add(method, pattern string, handler http.HandlerFunc) {
	if handler == nil {
		panic("chi: nil handler")
	}

	segs := parsePattern(pattern)

	m.mu.Lock()
	m.routes = append(m.routes, route{method: method, pattern: segs, handler: handler})
	m.mu.Unlock()
}

func parsePattern(pattern string) []segment {
	if pattern == "" {
		pattern = "/"
	}
	if pattern == "/" {
		return []segment{{literal: ""}}
	}
	trimmed := strings.Trim(pattern, "/")
	if trimmed == "" {
		return []segment{{literal: ""}}
	}
	parts := strings.Split(trimmed, "/")
	segs := make([]segment, 0, len(parts))
	for _, part := range parts {
		if len(part) >= 2 && part[0] == '{' && part[len(part)-1] == '}' {
			name := strings.TrimSpace(part[1 : len(part)-1])
			segs = append(segs, segment{param: name, isParam: true})
			continue
		}
		segs = append(segs, segment{literal: part})
	}
	return segs
}

func matchPattern(pattern []segment, path string) (map[string]string, bool) {
	if path == "" {
		path = "/"
	}
	if path == "/" {
		if len(pattern) == 1 && !pattern[0].isParam && pattern[0].literal == "" {
			return map[string]string{}, true
		}
		if len(pattern) == 0 {
			return map[string]string{}, true
		}
		return nil, false
	}
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		trimmed = ""
	}
	parts := []string{}
	if trimmed != "" {
		parts = strings.Split(trimmed, "/")
	}
	if len(parts) != len(pattern) {
		return nil, false
	}

	params := make(map[string]string)
	for idx, seg := range pattern {
		candidate := parts[idx]
		if seg.isParam {
			params[seg.param] = candidate
			continue
		}
		if seg.literal != candidate {
			return nil, false
		}
	}
	return params, true
}

// URLParam returns the value of the URL parameter from the request.
func URLParam(r *http.Request, key string) string {
	if r == nil {
		return ""
	}
	val := r.Context().Value(paramsKey)
	if val == nil {
		return ""
	}
	params, ok := val.(map[string]string)
	if !ok {
		return ""
	}
	return params[key]
}

// Use allows registering middleware handlers.
func (m *Mux) Use(middlewares ...func(http.Handler) http.Handler) {
	if len(middlewares) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	// Apply middleware to all existing handlers.
	for i, rt := range m.routes {
		h := http.Handler(rt.handler)
		for idx := len(middlewares) - 1; idx >= 0; idx-- {
			h = middlewares[idx](h)
		}
		if hf, ok := h.(http.HandlerFunc); ok {
			m.routes[i].handler = hf
		} else {
			// Wrap handler interface into function.
			m.routes[i].handler = func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, r)
			}
		}
	}
}

// NotFound sets the handler used when no route matches the request.
func (m *Mux) NotFound(handler http.Handler) {
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	m.mu.Lock()
	m.notFound = handler
	m.mu.Unlock()
}

// MethodNotAllowed sets the handler used when the route exists but the method does not match.
func (m *Mux) MethodNotAllowed(handler http.Handler) {
	if handler == nil {
		handler = http.HandlerFunc(methodNotAllowed)
	}
	m.mu.Lock()
	m.methodNot = handler
	m.mu.Unlock()
}
