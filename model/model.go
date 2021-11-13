package model

// Model contains an object's properties
var (
	Comment      Model = comment{}
	Event        Model = event{}
	Notification Model = notification{}
	Post         Model = post{}
	Product      Model = product{}
	User         Model = user{}
)

// Model represents models' default properties.
type Model interface {
	DefaultFields() string
	CacheKey(id string) string
	DgraphRDF() (predicate string, object string)
	URLQueryKey() string
	Tablename() string
	ValidField(field string) bool
}

type comment struct{}

func (comment) DefaultFields() string {
	return "id, user_id, content, likes_count, replies_count, created_at"
}
func (comment) URLQueryKey() string {
	return "comment.fields"
}
func (comment) CacheKey(id string) string {
	return id + "_comments"
}
func (comment) DgraphRDF() (predicate string, object string) {
	return "comment_id", "Comment"
}
func (comment) Tablename() string {
	return "events_posts_comments"
}
func (comment) ValidField(field string) bool {
	switch field {
	case "id", "parent_comment_id", "post_id", "user_id",
		"content", "likes_count", "replies_count", "created_at":
		return true
	}
	return false
}

type event struct{}

func (event) DefaultFields() string {
	return "id, name, description, virtual, type, ticket_type, public, address, latitude, longitude, cron, start_date, end_date, min_age, slots"
}
func (event) URLQueryKey() string {
	return "event.fields"
}
func (event) CacheKey(id string) string {
	return id + "_events"
}
func (event) DgraphRDF() (predicate string, object string) {
	return "event_id", "Event"
}
func (event) Tablename() string {
	return "events"
}
func (event) ValidField(field string) bool {
	switch field {
	case "id", "created_at", "updated_at", "name", "description",
		"type", "ticket_type", "public", "virtual", "address", "latitude",
		"longitude", "slots", "cron", "start_date", "end_date", "min_age", "url",
		"logo_url", "header_url":
		return true
	}
	return false
}

type notification struct{}

func (notification) DefaultFields() string {
	return "id, sender_id, receiver_id, event_id, type, content, seen, created_at"
}
func (notification) URLQueryKey() string {
	return "notification.fields"
}
func (notification) CacheKey(id string) string {
	return id + "_notifications"
}
func (notification) DgraphRDF() (predicate string, object string) {
	return "notification_id", "Notification"
}
func (notification) Tablename() string {
	return "notifications"
}
func (notification) ValidField(field string) bool {
	switch field {
	case "id", "sender_id", "receiver_id", "event_id", "type",
		"content", "seen", "created_at":
		return true
	}
	return false
}

type post struct{}

func (post) DefaultFields() string {
	return "id, event_id, content, media, likes_count, comments_count, created_at, updated_at"
}
func (post) URLQueryKey() string {
	return "post.fields"
}
func (post) CacheKey(eventID string) string {
	return eventID + "_posts"
}
func (post) DgraphRDF() (predicate string, object string) {
	return "post_id", "Post"
}
func (post) Tablename() string {
	return "events_posts"
}
func (post) ValidField(field string) bool {
	switch field {
	case "id", "event_id", "content", "media", "likes_count",
		"comments_count", "created_at", "updated_at":
		return true
	}
	return false
}

type product struct{}

func (product) DefaultFields() string {
	return "id, event_id, stock, brand, type, subtotal, total"
}
func (product) URLQueryKey() string {
	return "product.fields"
}
func (product) CacheKey(eventID string) string {
	return eventID + "_product"
}
func (product) DgraphRDF() (predicate string, object string) {
	return "product_id", "Product"
}
func (product) Tablename() string {
	return "events_products"
}
func (product) ValidField(field string) bool {
	switch field {
	case "id", "event_id", "stock", "brand", "type", "description",
		"discount", "taxes", "subtotal", "total", "created_at", "updated_at":
		return true
	}
	return false
}

type user struct{}

func (user) DefaultFields() string {
	return "id, name, username, email, private, type, invitations, created_at, updated_at"
}
func (user) URLQueryKey() string {
	return "user.fields"
}
func (user) CacheKey(id string) string {
	return id + "_users"
}
func (user) DgraphRDF() (predicate string, object string) {
	return "user_id", "User"
}
func (user) Tablename() string {
	return "users"
}
func (user) ValidField(field string) bool {
	switch field {
	case "id", "created_at", "updated_at", "name", "user_id", "username",
		"email", "description", "birth_date", "profile_image_url",
		"type", "invitations", "private", "verified_email":
		return true
	}
	return false
}
