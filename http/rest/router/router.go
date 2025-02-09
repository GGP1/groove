package router

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/http/rest/middleware"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/response"

	"github.com/go-redis/redis/v8"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

// Middleware represents a middleware function.
type Middleware func(http.Handler) http.Handler

// Router is the router used by groove.
type Router struct {
	Router      *httprouter.Router
	prefix      string
	middlewares []Middleware
}

// New returns a new router.
func New(config config.Config, db *sql.DB, rdb *redis.Client) http.Handler {
	router := &Router{
		Router: &httprouter.Router{
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      false,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          false,
			NotFound:               http.NotFoundHandler(),
			PanicHandler:           panicHandler,
		},
		middlewares: []Middleware{
			middleware.Secure,
			middleware.Cors,
			middleware.GzipCompress,
			middleware.NewMetrics().Scrap,
		},
	}

	if config.RateLimiter.Rate > 0 {
		rateLimiter := middleware.NewRateLimiter(config.RateLimiter, rdb)
		// Prepend so it's executed first
		router.middlewares = append([]Middleware{rateLimiter.Limit}, router.middlewares...)
	}

	registerEndpoints(router, db, rdb, config)

	return router
}

// ServeHTTP satisfies the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var handler http.Handler = r.Router
	// Middlewares used must be last
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}
	handler.ServeHTTP(w, req)
}

// handle registers a handler for the given method and path.
func (r *Router) handle(method, path string, handler http.Handler, middleware ...Middleware) {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	r.Router.Handler(method, r.prefix+path, handler)
}

// handleFunc registers a func handler for the given method and path.
func (r *Router) handleFunc(method, path string, handler http.HandlerFunc, middleware ...Middleware) {
	r.handle(method, path, handler, middleware...)
}

// delete registers a DELETE handler for the given path.
func (r *Router) delete(path string, handler http.Handler, middleware ...Middleware) {
	r.handle(http.MethodDelete, path, handler, middleware...)
}

// get registers a GET handler for the given path.
func (r *Router) get(path string, handler http.Handler, middleware ...Middleware) {
	r.handle(http.MethodGet, path, handler, middleware...)
}

// options registers a OPTIONS handler for the given path.
func (r *Router) options(path string, handler http.Handler, middleware ...Middleware) {
	r.handle(http.MethodOptions, path, handler, middleware...)
}

// patch registers a PATCH handler for the given path.
func (r *Router) patch(path string, handler http.Handler, middleware ...Middleware) {
	r.handle(http.MethodPatch, path, handler, middleware...)
}

// post registers a POST handler for the given path.
func (r *Router) post(path string, handler http.Handler, middleware ...Middleware) {
	r.handle(http.MethodPost, path, handler, middleware...)
}

// put registers a PUT handler for the given path.
func (r *Router) put(path string, handler http.Handler, middleware ...Middleware) {
	r.handle(http.MethodPut, path, handler, middleware...)
}

// group creates a new group of routes.
func (r *Router) group(prefix string) *Router {
	return &Router{
		Router: r.Router,
		prefix: r.prefix + prefix, // Join the current prefix (if any) and the new prefix
	}
}

// use sets middlewares for a router.
func (r *Router) use(middleware ...Middleware) {
	r.middlewares = append(r.middlewares, middleware...)
}

func home() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, struct{ Message string }{Message: "groove"})
	}
}

func panicHandler(w http.ResponseWriter, r *http.Request, err any) {
	response.Error(w, http.StatusInternalServerError, errors.New(fmt.Sprint("recovered from panic:", err)))
	log.DPanic("panic", zap.Any("error", err))
}
