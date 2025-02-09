package model

import (
	"reflect"
	"strings"
)

// T contains all model interfaces.
var T = &t{
	Comment: comment{
		fields: getFields(Comment{}),
	},
	Event: event{
		fields: getFields(Event{}),
	},
	Notification: notification{
		fields: getFields(Notification{}),
	},
	Post: post{
		fields: getFields(Post{}),
	},
	Product: product{
		fields: getFields(Product{}),
	},
	User: user{
		fields: getFields(User{}),
	},
}

// Model represents models' default properties.
type Model interface {
	Alias() string
	DefaultFields(useAlias bool) string
	CacheKey(id string) string
	URLQueryKey() string
	Tablename() string
	ValidField(field string) bool
}

type t struct {
	Comment      comment
	Event        event
	Notification notification
	Post         post
	Product      product
	User         user
}

type comment struct {
	fields map[string]struct{}
}

func (c comment) Alias() string {
	return "c"
}
func (c comment) DefaultFields(useAlias bool) string {
	if useAlias {
		return "c.id, c.user_id, c.content, c.replies_count, c.created_at"
	}
	return "id, user_id, content, replies_count, created_at"
}
func (c comment) URLQueryKey() string {
	return "comment.fields"
}
func (c comment) CacheKey(id string) string {
	return "comments:" + id
}
func (c comment) Tablename() string {
	return "events_posts_comments"
}
func (c comment) ValidField(field string) bool {
	_, ok := c.fields[field]
	return ok
}

type event struct {
	fields map[string]struct{}
}

func (e event) Alias() string {
	return "e"
}
func (e event) DefaultFields(useAlias bool) string {
	if useAlias {
		return `e.id, e.name, e.description, e.virtual, e.type, e.ticket_type, e.public, e.address, e.latitude,
		e.longitude, e.cron, e.start_date, e.end_date, e.min_age, e.slots, e.created_at`
	}
	return `id, name, description, virtual, type, ticket_type, public, address, latitude,
	longitude, cron, start_date, end_date, min_age, slots, created_at`
}
func (e event) URLQueryKey() string {
	return "event.fields"
}
func (e event) CacheKey(id string) string {
	return "events:" + id
}
func (e event) Tablename() string {
	return "events"
}
func (e event) ValidField(field string) bool {
	_, ok := e.fields[field]
	return ok
}

type notification struct {
	fields map[string]struct{}
}

func (n notification) Alias() string {
	return "n"
}
func (n notification) DefaultFields(useAlias bool) string {
	if useAlias {
		return "n.id, n.sender_id, n.receiver_id, n.event_id, n.type, n.content, n.seen, n.created_at"
	}
	return "id, sender_id, receiver_id, event_id, type, content, seen, created_at"
}
func (n notification) URLQueryKey() string {
	return "notification.fields"
}
func (n notification) CacheKey(id string) string {
	return "notifications:" + id
}
func (n notification) Tablename() string {
	return "notifications"
}
func (n notification) ValidField(field string) bool {
	_, ok := n.fields[field]
	return ok
}

type post struct {
	fields map[string]struct{}
}

func (p post) Alias() string {
	return "p"
}
func (p post) DefaultFields(useAlias bool) string {
	if useAlias {
		return "p.id, p.event_id, p.content, p.media, p.comments_count, p.created_at, p.updated_at"
	}
	return "id, event_id, content, media, comments_count, created_at, updated_at"
}
func (p post) URLQueryKey() string {
	return "post.fields"
}
func (p post) CacheKey(eventID string) string {
	return "posts:" + eventID
}
func (p post) Tablename() string {
	return "events_posts"
}
func (p post) ValidField(field string) bool {
	_, ok := p.fields[field]
	return ok
}

type product struct {
	fields map[string]struct{}
}

func (pr product) Alias() string {
	return "pr"
}
func (pr product) DefaultFields(useAlias bool) string {
	if useAlias {
		return "pr.id, pr.event_id, pr.stock, pr.brand, pr.type, pr.subtotal, pr.total"
	}
	return "id, event_id, stock, brand, type, subtotal, total"
}
func (pr product) URLQueryKey() string {
	return "product.fields"
}
func (pr product) CacheKey(eventID string) string {
	return "product:" + eventID
}
func (pr product) Tablename() string {
	return "events_products"
}
func (pr product) ValidField(field string) bool {
	_, ok := pr.fields[field]
	return ok
}

type user struct {
	fields map[string]struct{}
}

func (u user) Alias() string {
	return "u"
}
func (u user) DefaultFields(useAlias bool) string {
	if useAlias {
		return "u.id, u.name, u.username, u.email, u.profile_image_url, u.private, u.type, u.invitations"
	}
	return "id, name, username, email, profile_image_url, private, type, invitations"
}
func (u user) URLQueryKey() string {
	return "user.fields"
}
func (u user) CacheKey(id string) string {
	return ":users" + id
}
func (u user) Tablename() string {
	return "users"
}
func (u user) ValidField(field string) bool {
	_, ok := u.fields[field]
	return ok
}

// getFields returns a slice with the name of the fields of an object.
func getFields(obj any) map[string]struct{} {
	vPtr := reflect.ValueOf(obj)
	v := reflect.Indirect(vPtr)
	t := baseType(v.Type())

	fields := make(map[string]struct{}, 0)
	mapFields(t, fields)
	return fields
}

func mapFields(t reflect.Type, fields map[string]struct{}) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		bType := baseType(field.Type)
		kind := bType.Kind()
		if kind == reflect.Struct {
			// if the field's base type is a struct, map it as well
			mapFields(bType, fields)
		} else if kind == reflect.Slice && bType.Elem().Kind() == reflect.Struct {
			continue
		}

		fieldName := ""
		if tag := field.Tag.Get("db"); tag != "" {
			fieldName = tag
		} else {
			fieldName = strings.ToLower(field.Name)
		}

		fields[fieldName] = struct{}{}
	}
}

func baseType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
