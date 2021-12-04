package router

import (
	"database/sql"
	"net/http"
	"net/http/pprof"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/http/rest/middleware"
	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/permissions"
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type register struct {
	authMw       middleware.Auth
	event        event.Handler
	zone         zone.Handler
	role         role.Handler
	user         user.Handler
	post         post.Handler
	product      product.Handler
	notification notification.Handler
	ticket       ticket.Handler
	auth         auth.Handler
	prometheus   http.Handler
	report       report.Handler
	router       *Router
	config       config.Config
}

func registerEndpoints(router *Router, db *sql.DB, dc *dgo.Dgraph, rdb *redis.Client, cache cache.Client, config config.Config) {
	authService := auth.NewService(db, rdb, config.Sessions)
	roleService := role.NewService(db, dc, cache)
	notificationService := notification.NewService(db, dc, config.Notifications, authService, roleService)
	eventService := event.NewService(db, dc, cache, notificationService, roleService)
	postService := post.NewService(db, dc, cache, notificationService)
	productService := product.NewService(db, cache)
	userService := user.NewService(db, dc, cache, config.Admins, notificationService, roleService)
	ticketService := ticket.NewService(db, cache, roleService)
	zoneService := zone.NewService(db, cache)

	register := &register{
		config: config,
		router: router,
		authMw: middleware.NewAuth(db, authService, eventService, userService, roleService),
		// Prometheus default metrics
		prometheus: promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
			Registry: prometheus.DefaultRegisterer,
			// The response is already compressed by the gzip middleware, avoid double compression
			DisableCompression: true,
		}),
		auth:         auth.NewHandler(authService),
		role:         role.NewHandler(db, cache, roleService),
		notification: notification.NewHandler(db, notificationService),
		event:        event.NewHandler(db, cache, eventService, roleService),
		post:         post.NewHandler(db, dc, postService),
		product:      product.NewHandler(db, productService),
		report:       report.NewHandler(report.NewService(db)),
		user:         user.NewHandler(db, cache, userService),
		ticket:       ticket.NewHandler(db, ticketService),
		zone:         zone.NewHandler(db, cache, zoneService, roleService),
	}

	register.All()
}

func (r register) All() {
	r.Auth()
	r.Debug()
	r.Events()
	r.Posts()
	r.Products()
	r.Roles()
	r.Tickets()
	r.Zones()
	r.Metrics()
	r.Notifications()
	r.Reports()
	r.Users()
	r.Others()
}

func (r register) Auth() {
	r.router.post("/login", r.auth.Login())
	r.router.get("/login/basic", r.auth.BasicAuth())
	r.router.get("/logout", r.auth.Logout(), r.authMw.RequireLogin)
}

func (r register) Debug() {
	debug := r.router.group("/debug")
	debug.use(r.authMw.AdminsOnly)

	debug.get("/pprof", http.HandlerFunc(pprof.Index))
	debug.get("/cmdline", http.HandlerFunc(pprof.Cmdline))
	debug.get("/profile", http.HandlerFunc(pprof.Profile))
	debug.get("/symbol", http.HandlerFunc(pprof.Symbol))
	debug.get("/trace", http.HandlerFunc(pprof.Trace))
	debug.get("/allocs", pprof.Handler("allocs"))
	debug.get("/heap", pprof.Handler("heap"))
	debug.get("/goroutine", pprof.Handler("goroutine"))
	debug.get("/mutex", pprof.Handler("mutex"))
	debug.get("/block", pprof.Handler("block"))
	debug.get("/threadcreate", pprof.Handler("threadcreate"))
}

func (r register) Events() {
	events := r.router.group("/events/:id")

	events.get("/", r.event.GetByID(), r.authMw.NotBanned, r.authMw.EventPrivacyFilter)
	events.delete("/delete", r.event.Delete(), r.authMw.RequirePermissions(permissions.All))
	events.get("/hosts", r.event.GetHosts(), r.authMw.NotBanned, r.authMw.EventPrivacyFilter)
	events.post("/join", r.event.Join(), r.authMw.NotBanned, r.authMw.EventPrivacyFilter)
	events.get("/stats", r.event.GetStatistics(), r.authMw.NotBanned, r.authMw.EventPrivacyFilter)
	events.put("/update", r.event.Update(), r.authMw.RequirePermissions(permissions.UpdateEvent))

	// /events/:id/bans
	bans := events.group("/bans")
	{
		bans.use(r.authMw.NotBanned)
		bans.get("/", r.event.GetBans(), r.authMw.EventPrivacyFilter)
		bans.post("/add", r.event.AddBanned(), r.authMw.RequirePermissions(permissions.BanUsers))
		bans.get("/friends", r.event.GetBannedFriends(), r.authMw.EventPrivacyFilter)
		bans.post("/remove", r.event.RemoveBanned(), r.authMw.RequirePermissions(permissions.BanUsers))
	}

	// /events/:id/invited
	invited := events.group("/invited")
	{
		invited.use(r.authMw.NotBanned)
		invited.get("/", r.event.GetInvited(), r.authMw.EventPrivacyFilter)
		if r.config.Development {
			invited.post("/add", r.event.AddInvited(), r.authMw.RequirePermissions(permissions.InviteUsers))
		}
		invited.get("/friends", r.event.GetInvitedFriends(), r.authMw.EventPrivacyFilter)
	}

	// /events/:id/likes
	likes := events.group("/likes")
	{
		likes.use(r.authMw.NotBanned, r.authMw.EventPrivacyFilter)
		likes.get("/", r.event.GetLikes())
		likes.post("/add", r.event.AddLike())
		likes.get("/friends", r.event.GetLikedByFriends())
		likes.post("/remove", r.event.RemoveLike())
	}
}

func (r register) Posts() {
	// The event ID is preserved in the path in both posts and comments endpoints
	// to be able to use the middleware and verify the user has access to them
	posts := r.router.group("/events/:id/posts")
	posts.use(r.authMw.NotBanned)

	posts.get("/", r.post.GetPosts(), r.authMw.EventPrivacyFilter)
	posts.get("/:post_id", r.post.GetPost(), r.authMw.EventPrivacyFilter)
	posts.get("/:post_id/comments", r.post.GetPostComments(), r.authMw.EventPrivacyFilter)
	posts.get("/:post_id/like", r.post.LikePost(), r.authMw.EventPrivacyFilter)
	posts.get("/:post_id/likes", r.post.GetPostLikes(), r.authMw.EventPrivacyFilter)
	posts.get("/:post_id/liked", r.post.UserLikedPost(), r.authMw.EventPrivacyFilter)
	posts.post("/create", r.post.CreatePost(), r.authMw.RequirePermissions(permissions.ModifyPosts))
	posts.delete("/delete/:post_id", r.post.DeletePost(), r.authMw.RequirePermissions(permissions.ModifyPosts))
	posts.put("/update/:post_id", r.post.UpdatePost(), r.authMw.RequirePermissions(permissions.ModifyPosts))

	comments := r.router.group("/events/:id/comments")
	comments.use(r.authMw.NotBanned, r.authMw.EventPrivacyFilter)

	comments.get("/:comment_id", r.post.GetComment())
	comments.get("/:comment_id/like", r.post.LikeComment())
	comments.get("/:comment_id/likes", r.post.GetCommentLikes())
	comments.get("/:comment_id/liked", r.post.UserLikedComment())
	comments.delete("/delete/:comment_id", r.post.DeleteComment())
	comments.post("/create", r.post.CreateComment())
}

func (r register) Products() {
	products := r.router.group("/events/:id/products")

	products.get("/", r.product.Get(), r.authMw.EventPrivacyFilter)
	products.delete("/delete/:product_id", r.product.Delete(), r.authMw.RequirePermissions(permissions.ModifyProducts))
	products.post("/create", r.product.Create(), r.authMw.RequirePermissions(permissions.ModifyProducts))
	products.put("/update/:product_id", r.product.Update(), r.authMw.RequirePermissions(permissions.ModifyProducts))
}

func (r register) Roles() {
	roles := r.router.group("/events/:id/roles")
	roles.use(r.authMw.NotBanned)

	roles.get("/", r.role.GetRoles(), r.authMw.EventPrivacyFilter)
	roles.get("/members", r.role.GetMembers(), r.authMw.RequirePermissions(permissions.ModifyRoles))
	roles.get("/members/friends", r.role.GetMembersFriends(), r.authMw.RequirePermissions(permissions.ModifyRoles))
	roles.get("/role/:name", r.role.GetRole(), r.authMw.EventPrivacyFilter)
	roles.post("/clone", r.role.CloneRoles(), r.authMw.RequirePermissions(permissions.ModifyRoles))
	roles.post("/create", r.role.CreateRole(), r.authMw.RequirePermissions(permissions.ModifyRoles))
	roles.delete("/delete/:name", r.role.DeleteRole(), r.authMw.RequirePermissions(permissions.ModifyRoles))
	roles.post("/set", r.role.SetRoles(), r.authMw.RequirePermissions(permissions.SetUserRole))
	roles.post("/user", r.role.GetUserRole(), r.authMw.EventPrivacyFilter)
	roles.put("/update/:name", r.role.UpdateRole(), r.authMw.RequirePermissions(permissions.ModifyRoles))

	perm := r.router.group("/events/:id/permissions")
	perm.use(r.authMw.NotBanned, r.authMw.RequirePermissions(permissions.ModifyPermissions))

	perm.get("/", r.role.GetPermissions())
	perm.get("/:key", r.role.GetPermission())
	perm.post("/clone", r.role.ClonePermissions())
	perm.post("/create", r.role.CreatePermission())
	perm.delete("/delete/:key", r.role.DeletePermission())
	perm.put("/update/:key", r.role.UpdatePermission())
}

func (r register) Tickets() {
	tickets := r.router.group("/events/:id/tickets")
	tickets.use(r.authMw.NotBanned)

	tickets.get("/", r.ticket.Get())
	tickets.get("/ticket/:name", r.ticket.GetByName())
	tickets.get("/available/:name", r.ticket.Available())
	tickets.post("/buy/:name", r.ticket.Buy())
	tickets.post("/create", r.ticket.Create(), r.authMw.RequirePermissions(permissions.ModifyTickets))
	tickets.delete("/delete/:name", r.ticket.Delete(), r.authMw.RequirePermissions(permissions.ModifyTickets))
	tickets.get("/refund/:name", r.ticket.Refund())
	tickets.put("/update/:name", r.ticket.Update(), r.authMw.RequirePermissions(permissions.ModifyTickets))
}

func (r register) Zones() {
	zones := r.router.group("/events/:id/zones")
	zones.use(r.authMw.NotBanned)

	zones.get("/", r.zone.Get(), r.authMw.EventPrivacyFilter)
	zones.get("/zone/:name", r.zone.GetByName(), r.authMw.EventPrivacyFilter)
	zones.get("/access/:name", r.zone.Access(), r.authMw.EventPrivacyFilter)
	zones.post("/create", r.zone.Create(), r.authMw.RequirePermissions(permissions.ModifyZones))
	zones.delete("/delete/:name", r.zone.Delete(), r.authMw.RequirePermissions(permissions.ModifyZones))
	zones.put("/update/:name", r.zone.Update(), r.authMw.RequirePermissions(permissions.ModifyZones))
}

func (r register) Metrics() {
	r.router.get("/metrics", r.prometheus) // adminsOnly
}

func (r register) Notifications() {
	notifications := r.router.group("/notifications")

	notifications.post("/answer/:id", r.notification.Answer(), r.authMw.RequireLogin)
	notifications.get("/user/:user_id", r.notification.GetFromUser(), r.authMw.RequireLogin)
}

func (r register) Reports() {
	reports := r.router.group("/reports")

	reports.get("/", r.report.Get(), r.authMw.AdminsOnly)
	reports.post("/create", r.report.Create(), r.authMw.RequireLogin)
}

func (r register) Users() {
	// /users/:id
	users := r.router.group("/users/:id")

	users.get("/", r.user.GetByID(), r.authMw.UserPrivacyFilter)
	users.get("/stats", r.user.GetStatistics(), r.authMw.UserPrivacyFilter)
	users.post("/block", r.user.Block(), r.authMw.OwnUserOnly)
	users.get("/blocked", r.user.GetBlocked(), r.authMw.UserPrivacyFilter)
	users.get("/blocked_by", r.user.GetBlockedBy(), r.authMw.UserPrivacyFilter)
	users.get("/followers", r.user.GetFollowers(), r.authMw.UserPrivacyFilter)
	users.get("/following", r.user.GetFollowing(), r.authMw.UserPrivacyFilter)
	users.delete("/delete", r.user.Delete(), r.authMw.OwnUserOnly)
	users.post("/unblock", r.user.Unblock(), r.authMw.OwnUserOnly)
	users.put("/update", r.user.Update(), r.authMw.OwnUserOnly)
	// Would be better to have this inside the event handler but the dependecies match with the user service
	users.post("/invite", r.user.InviteToEvent(), r.authMw.RequirePermissions(permissions.InviteUsers))

	// /users/:id/events
	events := users.group("/events")
	{
		events.use(r.authMw.OwnUserOnly)
		events.get("/attending", r.user.GetAttendingEvents())
		events.get("/banned", r.user.GetBannedEvents())
		events.get("/hosted", r.user.GetHostedEvents())
		events.get("/invited", r.user.GetInvitedEvents())
		events.get("/liked", r.user.GetLikedEvents())
	}

	// /users/:id/friends
	friends := users.group("/friends")
	{
		friends.get("/", r.user.GetFriends(), r.authMw.UserPrivacyFilter)
		if r.config.Development {
			friends.post("/add", r.user.AddFriend(), r.authMw.OwnUserOnly)
		}
		friends.get("/common/:friend_id", r.user.GetFriendsInCommon(), r.authMw.OwnUserOnly)
		friends.get("/notcommon/:friend_id", r.user.GetFriendsNotInCommon(), r.authMw.OwnUserOnly)
		friends.post("/request", r.user.SendFriendRequest(), r.authMw.OwnUserOnly)
		friends.post("/remove", r.user.RemoveFriend(), r.authMw.OwnUserOnly)
	}
}

func (r register) Others() {
	r.router.get("/", home())
	// The endpoints below are here because they had conflicts with their respective groups
	r.router.post("/create/event", r.event.Create(), r.authMw.RequireLogin)
	r.router.post("/create/user", r.user.Create())
	r.router.get("/home/posts", r.post.GetHomePosts(), r.authMw.RequireLogin)
	r.router.post("/recommended/events", r.event.GetRecommended(), r.authMw.RequireLogin)
	r.router.get("/search/events", r.event.Search(), r.authMw.RequireLogin)
	r.router.post("/search/events/location", r.event.SearchByLocation(), r.authMw.RequireLogin)
	r.router.get("/search/users", r.user.Search(), r.authMw.RequireLogin)
}
