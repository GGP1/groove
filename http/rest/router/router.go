package router

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/GGP1/groove/auth"
	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/http/rest/middleware"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/user"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/dgraph-io/dgo/v210"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
func New(config config.Config, db *sqlx.DB, dc *dgo.Dgraph, rdb *redis.Client, mc *memcache.Client) http.Handler {
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
		router.middlewares = append(router.middlewares, middleware.NewRateLimiter(config.RateLimiter, rdb).Limit)
	}

	eventService := event.NewService(db, dc, mc)
	userService := user.NewService(db, dc, mc, config.Admins)
	session := auth.NewSession(db, rdb, config.Sessions)

	authMw := middleware.NewAuth(db, session, userService)
	adminsOnly := authMw.AdminsOnly
	requireAPIKey := authMw.RequireAPIKey
	requireLogin := authMw.RequireLogin
	// OwnUserOnly already checks if the user is logged in
	ownUserOnly := authMw.OwnUserOnly

	// auth
	router.post("/login", auth.Login(session))
	router.get("/login/basic", auth.BasicAuth(session))
	router.get("/logout", auth.Logout(session), requireLogin)

	// home
	router.get("/", home())

	// pprof
	debug := router.group("/debug")
	{
		debug.use(adminsOnly)
		debug.get("/pprof", fnToHandler(pprof.Index))
		debug.get("/cmdline", fnToHandler(pprof.Cmdline))
		debug.get("/profile", fnToHandler(pprof.Profile))
		debug.get("/symbol", fnToHandler(pprof.Symbol))
		debug.get("/trace", fnToHandler(pprof.Trace))
		debug.get("/allocs", pprof.Handler("allocs"))
		debug.get("/heap", pprof.Handler("heap"))
		debug.get("/goroutine", pprof.Handler("goroutine"))
		debug.get("/mutex", pprof.Handler("mutex"))
		debug.get("/block", pprof.Handler("block"))
		debug.get("/threadcreate", pprof.Handler("threadcreate"))
	}

	// Prometheus default metrics
	promHandler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		Registry: prometheus.DefaultRegisterer,
		// The response is already compressed by the gzip middleware, avoid double compression
		DisableCompression: true,
	})
	router.get("/metrics", promHandler, adminsOnly)

	events := event.NewHandler(eventService, mc)
	ev := router.group("/events")
	{
		// /events/:id
		id := ev.group("/:id")
		{
			id.get("/bans", events.GetBans(), requireLogin)
			id.post("/bans/add", events.AddBanned(), requireLogin)
			id.get("/bans/following/:user_id", events.GetBansFollowing())
			id.post("/bans/remove", events.RemoveBanned(), requireLogin)
			id.get("/confirmed", events.GetConfirmed(), requireLogin)
			id.post("/confirmed/add", events.AddConfirmed(), requireLogin)
			id.get("/confirmed/following/:user_id", events.GetConfirmedFollowing())
			id.post("/confirmed/remove", events.RemoveConfirmed(), requireLogin)
			id.post("/create/media", events.CreateMedia(), requireLogin)
			id.post("/create/permission", events.CreatePermission(), requireLogin)
			id.post("/create/product", events.CreateProduct(), requireLogin)
			id.post("/create/role", events.CreateRole(), requireLogin)
			id.delete("/delete", events.Delete(), requireLogin)
			id.get("/get", events.GetByID())
			id.get("/hosts", events.GetHosts(), requireLogin)
			id.get("/invited", events.GetInvited(), requireLogin)
			id.post("/invited/add", events.AddInvited(), requireLogin)
			id.get("/invited/following/:user_id", events.GetInvitedFollowing())
			id.post("/invited/remove", events.RemoveInvited(), requireLogin)
			id.get("/likes", events.GetLikes(), requireLogin)
			id.post("/likes/add", events.AddLike(), requireLogin)
			id.get("/likes/following/:user_id", events.GetLikesFollowing())
			id.post("/likes/remove", events.RemoveLike(), requireLogin)
			id.get("/media", events.GetMedia())
			id.get("/permissions", events.GetPermissions(), requireLogin)
			id.post("/permissions/clone", events.ClonePermissions(), requireLogin)
			id.get("/products", events.GetProducts())
			id.get("/roles", events.GetRoles(), requireLogin)
			id.get("/roles/clone", events.CloneRoles(), requireLogin)
			id.post("/roles/get", events.GetRole(), requireLogin)
			id.post("/roles/set", events.SetRole(), requireLogin)
			id.get("/reports", events.GetReports(), requireLogin)
			id.put("/update", events.Update(), requireLogin)
			id.put("/update/media", events.UpdateMedia(), requireLogin)
			id.put("/update/product", events.UpdateProduct(), requireLogin)
		}
	}

	users := user.NewHandler(userService, mc)
	us := router.group("/users")
	{
		// /users/:id
		id := us.group("/:id")
		{
			id.post("/block", users.Block(), ownUserOnly)
			id.get("/blocked", users.GetBlocked())
			id.get("/blocked_by", users.GetBlockedBy())
			id.delete("/delete", users.Delete(), ownUserOnly)
			id.post("/follow", users.Follow(), ownUserOnly)
			id.get("/followers", users.GetFollowers())
			id.get("/following", users.GetFollowing())
			id.get("/get", users.GetByID(), requireAPIKey)
			id.post("/unblock", users.Unblock(), ownUserOnly)
			id.post("/unfollow", users.Unfollow(), ownUserOnly)
			id.put("/update", users.Update(), ownUserOnly)

			// /users/:id/events
			evs := id.group("/events")
			{
				evs.get("/banned", users.GetBannedEvents())
				evs.get("/confirmed", users.GetConfirmedEvents())
				evs.get("/hosted", users.GetHostedEvents())
				evs.get("/invited", users.GetInvitedEvents())
				evs.get("/liked", users.GetLikedEvents())
			}
		}
	}

	router.post("/create/event", events.Create())
	router.post("/create/user", users.Create())
	router.get("/search/event/:query", events.Search())
	router.get("/search/user/:query", users.Search())

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
		response.JSONMessage(w, http.StatusOK, "groove")
	}
}

func panicHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	response.Error(w, http.StatusInternalServerError, errors.New(fmt.Sprint("recovered from panic:", err)))
	log.DPanic("panic", zap.Any("error", err))
}

// fnToHandler takes a handler function and returns a handler.
func fnToHandler(f http.HandlerFunc) http.Handler {
	return f
}
