package router

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/http/rest/middleware"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event"
	"github.com/GGP1/groove/service/event/post"
	"github.com/GGP1/groove/service/event/product"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/event/ticket"
	"github.com/GGP1/groove/service/event/zone"
	"github.com/GGP1/groove/service/notification"
	"github.com/GGP1/groove/service/report"
	"github.com/GGP1/groove/service/user"

	"github.com/dgraph-io/dgo/v210"
	"github.com/go-redis/redis/v8"
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
func New(config config.Config, db *sql.DB, dc *dgo.Dgraph, rdb *redis.Client, cache cache.Client) http.Handler {
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
		router.middlewares = append(router.middlewares, rateLimiter.Limit)
	}

	// Services
	authService := auth.NewService(db, rdb, config.Sessions)
	roleService := role.NewService(db, dc, cache)
	notificationService := notification.NewService(db, dc, config.Notifications, authService, roleService)
	eventService := event.NewService(db, dc, cache, notificationService, roleService)
	postService := post.NewService(db, dc, cache, notificationService)
	productService := product.NewService(db, cache)
	userService := user.NewService(db, dc, cache, config.Admins, notificationService)
	ticketService := ticket.NewService(db, cache, roleService)
	zoneService := zone.NewService(db, cache)

	authMw := middleware.NewAuth(db, authService, eventService, userService)
	adminsOnly := authMw.AdminsOnly
	// requireAPIKey := authMw.RequireAPIKey
	requireLogin := authMw.RequireLogin
	// The two below already checks if the user is logged in
	ownUserOnly := authMw.OwnUserOnly
	notBanned := authMw.NotBanned

	// auth
	auth := auth.NewHandler(authService)
	router.post("/login", auth.Login())
	router.get("/login/basic", auth.BasicAuth())
	router.get("/logout", auth.Logout(), requireLogin)

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
	router.get("/metrics", promHandler) // adminsOnly

	events := event.NewHandler(db, cache, eventService, roleService)
	ev := router.group("/events")
	{
		// /events/:id
		id := ev.group("/:id")
		{
			id.get("/", events.GetByID())
			id.delete("/delete", events.Delete(), requireLogin)
			id.get("/hosts", events.GetHosts())
			id.get("/stats", events.GetStatistics())
			id.put("/update", events.Update(), requireLogin)

			// /events/:id/bans
			bans := id.group("/bans")
			{
				bans.use(notBanned)
				bans.get("/", events.GetBans())
				bans.post("/add", events.AddBanned())
				bans.get("/friends", events.GetBannedFriends())
				bans.post("/remove", events.RemoveBanned())
			}

			// /events/:id/invited
			invited := id.group("/invited")
			{
				invited.use(notBanned)
				invited.get("/", events.GetInvited())
				if config.Development {
					invited.post("/add", events.AddInvited())
				}
				invited.get("/friends", events.GetInvitedFriends())
				invited.post("/remove", events.RemoveInvited())
			}

			// /events/:id/likes
			likes := id.group("/likes")
			{
				likes.use(notBanned)
				likes.get("/", events.GetLikes())
				likes.post("/add", events.AddLike())
				likes.get("/friends", events.GetLikedByFriends())
				likes.post("/remove", events.RemoveLike())
			}

			// /events/:id/posts
			posts := post.NewHandler(db, postService, roleService)
			psts := id.group("/posts")
			{
				psts.use(notBanned)
				psts.get("/", posts.GetPosts())
				psts.get("/:post_id", posts.GetPost())
				psts.get("/:post_id/comments", posts.GetPostComments())
				psts.get("/:post_id/like", posts.LikePost())
				psts.get("/:post_id/likes", posts.GetPostLikes())
				psts.post("/create", posts.CreatePost())
				psts.delete("/delete/:post_id", posts.DeletePost())
				psts.put("/update/:post_id", posts.UpdatePost())
			}

			// /events/:id/comments
			comments := id.group("/comments")
			{
				comments.use(notBanned)
				comments.get("/:comment_id", posts.GetComment())
				comments.get("/:comment_id/like", posts.LikeComment())
				comments.get("/:comment_id/likes", posts.GetCommentLikes())
				comments.delete("/delete/:comment_id", posts.DeleteComment())
				comments.post("/create", posts.CreateComment())
			}

			// /events/:id/permissions
			roles := role.NewHandler(db, cache, roleService)
			perm := id.group("/permissions")
			{
				perm.use(notBanned)
				perm.get("/", roles.GetPermissions())
				perm.get("/:key", roles.GetPermission())
				perm.post("/clone", roles.ClonePermissions())
				perm.post("/create", roles.CreatePermission())
				perm.delete("/delete/:key", roles.DeletePermission())
				perm.put("/update/:key", roles.UpdatePermission())
			}

			// /events/:id/products
			pr := id.group("/products")
			{
				products := product.NewHandler(db, productService, roleService)
				pr.get("/", products.Get())
				pr.delete("/delete/:product_id", products.Delete())
				pr.post("/create", products.Create(), requireLogin)
				pr.put("/update/:product_id", products.Update(), requireLogin)
			}

			// /events/:id/roles
			r := id.group("/roles")
			{
				r.use(notBanned)
				r.get("/", roles.GetRoles())
				r.get("/members", roles.GetMembers())
				r.get("/members/friends", roles.GetMembersFriends())
				r.get("/role/:name", roles.GetRole())
				r.post("/clone", roles.CloneRoles())
				r.post("/create", roles.CreateRole())
				r.delete("/delete/:name", roles.DeleteRole())
				r.post("/set", roles.SetRoles())
				r.post("/user", roles.GetUserRole())
				r.put("/update/:name", roles.UpdateRole())
			}

			t := id.group("/tickets")
			{
				tickets := ticket.NewHandler(db, ticketService, roleService)
				t.use(notBanned)
				t.get("/", tickets.Get())
				t.get("/available/:name", tickets.Available())
				t.get("/buy/:name", tickets.Buy())
				t.post("/create", tickets.Create())
				t.delete("/delete/:name", tickets.Delete())
				t.get("/refund/:name", tickets.Refund())
				t.put("/update/:name", tickets.Update())
			}

			// /events/:id/zones
			z := id.group("/zones")
			{
				zones := zone.NewHandler(db, cache, zoneService, roleService)
				z.use(notBanned)
				z.get("/", zones.Get())
				z.get("/zone/:name", zones.GetByName())
				z.get("/access/:name", zones.Access())
				z.post("/create", zones.Create())
				z.delete("/delete/:name", zones.Delete())
				z.put("/update/:name", zones.Update())
			}
		}
	}

	ntf := router.group("/notifications")
	{
		notifications := notification.NewHandler(db, notificationService)
		ntf.post("/answer/:id", notifications.Answer(), requireLogin)
		ntf.get("/user/:user_id", notifications.GetFromUser(), requireLogin)
	}

	rp := router.group("/reports")
	{
		reports := report.NewHandler(report.NewService(db))
		rp.get("/", reports.Get(), adminsOnly)
		rp.post("/create", reports.Create(), requireLogin)
	}

	users := user.NewHandler(db, cache, userService, roleService)
	us := router.group("/users")
	{
		// /users/:id
		id := us.group("/:id")
		{
			id.get("/", users.GetByID())
			id.get("/stats", users.GetStatistics())
			id.post("/block", users.Block(), ownUserOnly)
			id.get("/blocked", users.GetBlocked())
			id.get("/blocked_by", users.GetBlockedBy())
			id.delete("/delete", users.Delete(), ownUserOnly)
			id.post("/unblock", users.Unblock(), ownUserOnly)
			id.put("/update", users.Update(), ownUserOnly)
			id.post("/invite", users.InviteToEvent(), requireLogin) // TODO: change, not intuitive as endpoint

			// /users/:id/events
			evs := id.group("/events")
			{
				evs.use(ownUserOnly)
				evs.get("/attending", users.GetAttendingEvents())
				evs.get("/banned", users.GetBannedEvents())
				evs.get("/hosted", users.GetHostedEvents())
				evs.get("/invited", users.GetInvitedEvents())
				evs.get("/liked", users.GetLikedEvents())
			}

			// /users/:id/friends
			friends := id.group("/friends")
			{
				friends.get("/", users.GetFriends())
				if config.Development {
					friends.post("/add", users.AddFriend(), ownUserOnly)
				}
				friends.get("/common/:friend_id", users.GetFriendsInCommon(), ownUserOnly)
				friends.get("/notcommon/:friend_id", users.GetFriendsNotInCommon(), ownUserOnly)
				friends.post("/request", users.SendFriendRequest(), ownUserOnly)
				friends.post("/remove", users.RemoveFriend(), ownUserOnly)
			}
		}
	}

	router.post("/create/event", events.Create(), requireLogin)
	router.post("/create/user", users.Create())
	router.get("/search/events", events.Search())
	router.post("/search/events/location", events.SearchByLocation())
	router.get("/search/users", users.Search())

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
