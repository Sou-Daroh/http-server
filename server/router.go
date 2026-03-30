package server

import (
	"strings"
)

// HandlerFunc is the signature for all route handlers and middleware wrappers.
type HandlerFunc func(req *Request, res *ResponseWriter)

// route holds a registered route — method, path pattern, and handler.
type route struct {
	method  string
	pattern string
	parts   []string // pre-split pattern segments
	handler HandlerFunc
}

// Router maps (method, path) pairs to handler functions.
// Supports exact paths and parameterized paths (e.g. /users/:id).
type Router struct {
	routes []route
}

// NewRouter creates an empty Router.
func NewRouter() *Router {
	return &Router{}
}

// Handle registers a handler for the given method and path pattern.
func (r *Router) Handle(method, pattern string, handler HandlerFunc) {
	r.routes = append(r.routes, route{
		method:  strings.ToUpper(method),
		pattern: pattern,
		parts:   splitPath(pattern),
		handler: handler,
	})
}

// GET registers a handler for GET requests.
func (r *Router) GET(pattern string, handler HandlerFunc) {
	r.Handle("GET", pattern, handler)
}

// POST registers a handler for POST requests.
func (r *Router) POST(pattern string, handler HandlerFunc) {
	r.Handle("POST", pattern, handler)
}

// PUT registers a handler for PUT requests.
func (r *Router) PUT(pattern string, handler HandlerFunc) {
	r.Handle("PUT", pattern, handler)
}

// DELETE registers a handler for DELETE requests.
func (r *Router) DELETE(pattern string, handler HandlerFunc) {
	r.Handle("DELETE", pattern, handler)
}

// PATCH registers a handler for PATCH requests.
func (r *Router) PATCH(pattern string, handler HandlerFunc) {
	r.Handle("PATCH", pattern, handler)
}

// Dispatch matches the incoming request to a registered route and calls its
// handler. Returns a handler function that writes the appropriate error
// response if no match is found.
//
// Matching rules:
//  1. Path segments must match exactly, or the route segment must start with ':'
//     (parameter), in which case the value is captured into req.Params.
//  2. If the path matches but the method does not, 405 is returned with an
//     Allow header listing the valid methods.
//  3. If no path matches at all, 404 is returned.
func (r *Router) Dispatch(req *Request, res *ResponseWriter) {
	requestParts := splitPath(req.Path)

	var allowedMethods []string

	for _, route := range r.routes {
		params, matched := matchPath(route.parts, requestParts)
		if !matched {
			continue
		}

		// Path matched — check method
		if route.method != req.Method {
			allowedMethods = append(allowedMethods, route.method)
			continue
		}

		// Full match — populate params and call handler
		for k, v := range params {
			req.Params[k] = v
		}
		route.handler(req, res)
		return
	}

	// Path matched but wrong method
	if len(allowedMethods) > 0 {
		res.SetHeader("Allow", strings.Join(allowedMethods, ", "))
		res.WriteText(405, "Method Not Allowed")
		return
	}

	// No match at all
	res.WriteText(404, "Not Found")
}

// matchPath checks if the request path segments match a route pattern.
// Returns extracted parameters and true on a match, or nil and false otherwise.
func matchPath(patternParts, requestParts []string) (map[string]string, bool) {
	if len(patternParts) != len(requestParts) {
		return nil, false
	}

	params := make(map[string]string)

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			// Parameter segment — capture value
			paramName := part[1:]
			params[paramName] = requestParts[i]
		} else if part != requestParts[i] {
			// Literal segment mismatch
			return nil, false
		}
	}

	return params, true
}

// splitPath splits a URL path into non-empty segments.
// "/users/42/" → ["users", "42"]
func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
