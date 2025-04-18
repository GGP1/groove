package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/log"

	"github.com/pkg/errors"
)

// Connect establishes a connection with the postgres database.
func Connect(ctx context.Context, c config.Postgres) (*sql.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s sslcert=%s sslkey=%s",
		c.Host, c.Port, c.Username, c.Password, c.Name, c.SSLMode, c.SSLRootCert, c.SSLCert, c.SSLKey)

	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, errors.Wrap(err, "connecting with postgres")
	}
	db.SetMaxIdleConns(c.MaxIdleConns)
	db.SetConnMaxIdleTime(c.ConnMaxIdleTime * time.Second)

	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Wrap(err, "ping error")
	}

	if err := CreateTables(ctx, db); err != nil {
		return nil, err
	}
	if err := CreateProcedures(ctx, db); err != nil {
		return nil, err
	}

	runMetrics(db, c)

	log.Sugar().Infof("Connected to postgres on %s:%s", c.Host, c.Port)
	return db, nil
}

// CreateTables creates postgres tables.
func CreateTables(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, tables); err != nil {
		return errors.Wrap(err, "creating tables")
	}
	return nil
}

// CreateProcedures creates pre-defined postgres procedures.
func CreateProcedures(ctx context.Context, db *sql.DB) error {
	likePost := `CREATE OR REPLACE PROCEDURE likePost(postID text, userID text) AS $$
	BEGIN
		IF EXISTS (SELECT 1 FROM events_posts_likes WHERE post_id=postID AND user_id=userID) THEN
	   		DELETE FROM events_posts_likes WHERE post_id=postID AND user_id=userID;
	   	ELSE
	   		INSERT INTO events_posts_likes (post_id, user_id) VALUES (postID, userID);
	   	END IF;
	END $$ LANGUAGE plpgsql`
	if _, err := db.ExecContext(ctx, likePost); err != nil {
		return errors.Wrap(err, "creating likePost procedure")
	}

	likeComment := `CREATE OR REPLACE PROCEDURE likeComment(commentID text, userID text) AS $$ 
	BEGIN
		IF EXISTS (SELECT 1 FROM events_posts_comments_likes WHERE comment_id=commentID AND user_id=userID) THEN
			DELETE FROM events_posts_comments_likes WHERE comment_id=commentID AND user_id=userID;
	   	ELSE
	   		INSERT INTO events_posts_comments_likes (comment_id, user_id) VALUES (commentID, userID);
	   	END IF;
	END $$ LANGUAGE plpgsql`
	if _, err := db.ExecContext(ctx, likeComment); err != nil {
		return errors.Wrap(err, "creating likeComment procedure")
	}

	return nil
}

const tables = `
CREATE TABLE IF NOT EXISTS users
(
    id varchar(26),
	name varchar(40) NOT NULL,
    username varchar(24) NOT NULL UNIQUE,
    email varchar(120) NOT NULL UNIQUE,
    password bytea NOT NULL,
	description varchar(200),
	birth_date timestamp,
	profile_image_url varchar(240),
    is_admin boolean DEFAULT false,
	private boolean DEFAULT false,
	type smallint NOT NULL CHECK (type > 0 AND type < 3),
	invitations smallint NOT NULL CHECK (invitations > 0 AND invitations < 3),
    verified_email boolean DEFAULT false,
	search tsvector,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

CREATE INDEX ON users USING GIN (search);

CREATE OR REPLACE FUNCTION users_tsvector_trigger() RETURNS trigger AS $$
BEGIN
	new.search :=
	setweight(to_tsvector('english', new.username), 'A')
	|| setweight(to_tsvector('english', new.name), 'B');
	return new;
END $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_tsvector_update ON users;

CREATE TRIGGER users_tsvector_update BEFORE INSERT OR UPDATE
    ON users FOR EACH ROW EXECUTE PROCEDURE users_tsvector_trigger();

CREATE TABLE IF NOT EXISTS users_friends
(
	user_id varchar(26) NOT NULL,
	friend_id varchar(26) NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	FOREIGN KEY (friend_id) REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE(user_id, friend_id)
);

CREATE TABLE IF NOT EXISTS users_followers
(
	user_id varchar(26) NOT NULL,
	follower_id varchar(26) NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	FOREIGN KEY (follower_id) REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE(user_id, follower_id)
);

CREATE TABLE IF NOT EXISTS users_blocked
(
	user_id varchar(26) NOT NULL,
	blocked_id varchar(26) NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	FOREIGN KEY (blocked_id) REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE(user_id, blocked_id)
);

CREATE TABLE IF NOT EXISTS events
(
	id varchar(26),
	name varchar(60) NOT NULL,
	description varchar(200),
	type smallint NOT NULL CHECK (type > 0),
	ticket_type smallint NOT NULL CHECK (ticket_type > 0),
	virtual bool NOT NULL,
	logo_url varchar(240),
	header_url varchar(240),
	url varchar(240),
	address varchar(120),
	latitude double precision,
	longitude double precision,
	public boolean NOT NULL,
	slots integer NOT NULL CHECK (slots >= -1),
	cron varchar(40) NOT NULL,
	start_date timestamp with time zone NOT NULL,
	end_date timestamp with time zone NOT NULL,
	min_age smallint DEFAULT 0 CHECK (min_age >= 0),
	search tsvector,
	created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone,
    CONSTRAINT events_pkey PRIMARY KEY (id)
);

CREATE INDEX ON events USING GIN (search);
CREATE INDEX ON events (latitude);
CREATE INDEX ON events (longitude);

CREATE OR REPLACE FUNCTION events_tsvector_trigger() RETURNS trigger AS $$
BEGIN
	new.search := setweight(to_tsvector('english', new.name), 'A')
	|| setweight(to_tsvector('english', new.address), 'B');
	return new;
END $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS events_tsvector_update ON events;

CREATE TRIGGER events_tsvector_update BEFORE INSERT OR UPDATE
    ON events FOR EACH ROW EXECUTE PROCEDURE events_tsvector_trigger();

CREATE TABLE IF NOT EXISTS events_bans
(
	event_id varchar(26) NOT NULL,
	user_id varchar(26) NOT NULL,
	FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE(event_id, user_id)
);

CREATE TABLE IF NOT EXISTS events_likes
(
	event_id varchar(26) NOT NULL,
	user_id varchar(26) NOT NULL,
	FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE(event_id, user_id)
);

CREATE TABLE IF NOT EXISTS events_permissions
(
 	event_id varchar(26) NOT NULL,
	key varchar(30) NOT NULL,
 	name varchar(40) NOT NULL,
 	description varchar(200),
    created_at timestamp with time zone DEFAULT NOW(),
	FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE,
	UNIQUE(event_id, key)
);

CREATE INDEX ON events_permissions (key);

CREATE TABLE IF NOT EXISTS events_roles
(
	event_id varchar(26) NOT NULL,
	name varchar(40) NOT NULL,
 	permission_keys text[] NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
	FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE,
	UNIQUE(event_id, name)
);

CREATE INDEX ON events_roles (name);

CREATE TABLE IF NOT EXISTS events_users_roles
(
	event_id varchar(26) NOT NULL,
	user_id varchar(26) NOT NULL,
 	role_name varchar(40) NOT NULL,
	FOREIGN KEY (event_id) REFERENCES events (id) ON UPDATE CASCADE ON DELETE CASCADE,
 	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE(event_id, user_id, role_name)
);

CREATE INDEX ON events_users_roles (role_name);

CREATE TABLE IF NOT EXISTS events_tickets
(
	event_id varchar(26) NOT NULL,
	available_count integer NOT NULL CHECK (available_count >= 0),
	name varchar(60) NOT NULL,
	description varchar(200),
	cost integer NOT NULL CHECK (cost >= 0),
	linked_role varchar(40) DEFAULT 'attendant',
	FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE,
	UNIQUE(event_id, name)
);

CREATE TABLE IF NOT EXISTS events_posts 
(
	id varchar(26),
	event_id varchar(26) NOT NULL,
	media text[],
	content varchar(1024),
	comments_count integer DEFAULT 0,
	created_at timestamp with time zone DEFAULT NOW(),
	updated_at timestamp with time zone,
	CONSTRAINT events_posts_pkey PRIMARY KEY (id),
	FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_posts_likes
(
	post_id varchar(26) NOT NULL,
	user_id varchar(26) NOT NULL,
	FOREIGN KEY (post_id) REFERENCES events_posts (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE(post_id, user_id)
);

CREATE TABLE IF NOT EXISTS events_posts_comments 
(
	id varchar(26),
	parent_comment_id varchar(26),
	post_id varchar(26),
	user_id varchar(26) NOT NULL,
	content varchar(1024),
	replies_count integer DEFAULT 0,
	created_at timestamp with time zone DEFAULT NOW(),
	CONSTRAINT events_posts_comments_pkey PRIMARY KEY (id),
	FOREIGN KEY (parent_comment_id) REFERENCES events_posts_comments (id) ON DELETE CASCADE,
	FOREIGN KEY (post_id) REFERENCES events_posts (id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_posts_comments_likes
(
	comment_id varchar(26) NOT NULL,
	user_id varchar(26) NOT NULL,
	FOREIGN KEY (comment_id) REFERENCES events_posts_comments (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
	UNIQUE(comment_id, user_id)
);

CREATE TABLE IF NOT EXISTS events_products
(
    id varchar(26),
    event_id varchar(26) NOT NULL,
    stock integer NOT NULL CHECK (stock >= 0),
    brand varchar(60) NOT NULL,
	type varchar(60) NOT NULL,
    description varchar(200),
    discount integer,
	taxes integer,
    subtotal integer NOT NULL,
    total integer NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone,
	CONSTRAINT events_products_pkey PRIMARY KEY (id),
    FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_reports
(
	id varchar(26),
	reported_id varchar(26) NOT NULL,
	reporter_id varchar(26) NOT NULL,
	type varchar(60) NOT NULL,
	details varchar(1024) NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
	CONSTRAINT events_reports_pkey PRIMARY KEY (id),
    FOREIGN KEY (reporter_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_zones
(
	event_id varchar(26),
	name varchar(40) NOT NULL,
	required_permission_keys text[],
    FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE,
	UNIQUE(event_id, name)
);

CREATE INDEX ON events_zones (name);

CREATE TABLE IF NOT EXISTS notifications
(
	id varchar(26),
	sender_id varchar(26) NOT NULL,
	receiver_id varchar(26) NOT NULL,
	event_id varchar(26),
	type integer NOT NULL CHECK (type > 0 AND type < 5),
	content varchar(240),
	seen boolean DEFAULT FALSE,
    created_at timestamp with time zone DEFAULT NOW(),
	CONSTRAINT notifications_pkey PRIMARY KEY (id),
    FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (receiver_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE,
	UNIQUE(sender_id, receiver_id, type)
);`
