package server

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"
	"sync"
	"time"
)

// Server is the main HTTP server. It manages a TCP listener, routes requests
// through a middleware chain, and handles graceful shutdown.
type Server struct {
	host      string
	port      int
	staticDir string

	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration

	router     *Router
	middleware []MiddlewareFunc

	listener net.Listener
	wg       sync.WaitGroup
	mu       sync.Mutex
}

// MiddlewareFunc wraps a HandlerFunc with additional behaviour.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Options holds server construction parameters.
type Options struct {
	Host         string
	Port         int
	StaticDir    string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// New creates a new Server with the given options.
func New(opts Options) *Server {
	return &Server{
		host:         opts.Host,
		port:         opts.Port,
		staticDir:    opts.StaticDir,
		readTimeout:  opts.ReadTimeout,
		writeTimeout: opts.WriteTimeout,
		idleTimeout:  opts.IdleTimeout,
		router:       NewRouter(),
	}
}

// Use registers a middleware function to be applied to all requests.
// Middleware is applied in the order it is registered.
func (s *Server) Use(m MiddlewareFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.middleware = append(s.middleware, m)
}

// Router returns the server's router for registering routes.
func (s *Server) Router() *Router {
	return s.router
}

// Start begins listening for TCP connections on the configured address.
// It blocks until ctx is cancelled or an unrecoverable error occurs.
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()

	fmt.Printf("http-server listening on http://%s\n", addr)

	// Watch for context cancellation to trigger shutdown
	go func() {
		<-ctx.Done()
		s.listener.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Check if the error is due to the listener being closed (shutdown)
			select {
			case <-ctx.Done():
				// Expected shutdown — wait for active connections to finish
				s.wg.Wait()
				return nil
			default:
				return fmt.Errorf("accept: %w", err)
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// Stop closes the listener, preventing new connections.
// Active connections are allowed to complete.
func (s *Server) Stop() {
	s.mu.Lock()
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}

// Addr returns the address the server is listening on.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// handleConnection manages the lifecycle of a single TCP connection.
// It loops handling requests for keep-alive connections.
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	for {
		// Apply read deadline for the next request
		if s.readTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(s.readTimeout))
		}

		// Parse the incoming request
		req, err := ParseRequest(conn, conn.RemoteAddr().String())
		if err != nil {
			// Connection closed or timed out — stop the loop silently
			return
		}

		// Reset read deadline during handler execution
		conn.SetReadDeadline(time.Time{})

		// Apply write deadline for sending the response
		if s.writeTimeout > 0 {
			conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
		}

		res := NewResponseWriter(conn)

		// Dispatch through the middleware chain
		handler := s.buildChain(req, res)
		handler(req, res)

		// Flush the response to the wire
		keepAlive := req.KeepAlive()
		if err := res.Flush(keepAlive); err != nil {
			return
		}

		conn.SetWriteDeadline(time.Time{})

		// Close the connection if the client or handler requested it
		if !keepAlive {
			return
		}

		// Apply idle timeout while waiting for the next request
		if s.idleTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(s.idleTimeout))
		}
	}
}

// buildChain constructs the middleware chain and returns the outermost handler.
// The chain is: middleware[0] → middleware[1] → ... → dispatch
func (s *Server) buildChain(req *Request, res *ResponseWriter) HandlerFunc {
	// The innermost handler: dispatch to router or static files
	final := s.dispatchHandler()

	// Wrap with recovery to catch panics in any middleware or handler
	final = recoveryMiddleware(final)

	// Apply middleware in reverse so middleware[0] is the outermost wrapper
	s.mu.Lock()
	mw := make([]MiddlewareFunc, len(s.middleware))
	copy(mw, s.middleware)
	s.mu.Unlock()

	for i := len(mw) - 1; i >= 0; i-- {
		final = mw[i](final)
	}

	return final
}

// dispatchHandler returns a HandlerFunc that routes the request to a
// registered route handler or the static file handler.
func (s *Server) dispatchHandler() HandlerFunc {
	return func(req *Request, res *ResponseWriter) {
		// Try registered routes first
		matched := false
		for _, route := range s.router.routes {
			params, ok := matchPath(route.parts, splitPath(req.Path))
			if ok {
				matched = true
				if route.method == req.Method {
					for k, v := range params {
						req.Params[k] = v
					}
					route.handler(req, res)
					return
				}
			}
		}

		if matched {
			// Path matched but no method match — let router write 405
			s.router.Dispatch(req, res)
			return
		}

		// Fall through to static file handler
		if s.staticDir != "" {
			staticHandler := StaticHandler(s.staticDir)
			staticHandler(req, res)
			return
		}

		res.WriteText(404, "Not Found")
	}
}

// recoveryMiddleware catches panics in handlers and returns 500.
func recoveryMiddleware(next HandlerFunc) HandlerFunc {
	return func(req *Request, res *ResponseWriter) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("panic recovered: %v\n%s\n", r, debug.Stack())
				if !res.written {
					res.WriteText(500, "Internal Server Error")
				}
			}
		}()
		next(req, res)
	}
}
