package postgres

import (
	"context"
	"fmt"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/log"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// TODO: benchmark performance sqlx.Select and Get vs sql standard library.
// Potentially drop sqlx dependency (should import pq/pgx driver)

// Connect establishes a connection with the postgres database.
func Connect(ctx context.Context, c config.Postgres) (*sqlx.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s sslrootcert=%s sslcert=%s sslkey=%s",
		c.Host, c.Port, c.Username, c.Password, c.Name, c.SSLMode, c.SSLRootCert, c.SSLCert, c.SSLKey)

	db, err := sqlx.ConnectContext(ctx, "postgres", url)
	if err != nil {
		return nil, errors.Wrap(err, "connecting with postgres")
	}

	if err := CreateTables(ctx, db); err != nil {
		return nil, errors.Wrap(err, "creating tables")
	}

	log.Sugar().Infof("Connected to postgres on %s:%s", c.Host, c.Port)
	return db, nil
}

// CreateTables creates postgres tables.
func CreateTables(ctx context.Context, db *sqlx.DB) error {
	if _, err := db.ExecContext(ctx, tables); err != nil {
		return errors.Wrap(err, "creating tables")
	}
	return nil
}

// ALTER TYPE invitations ADD VALUE 'selected';
const tables = `
CREATE TABLE IF NOT EXISTS events
(
	id UUID NOT NULL,
	name text NOT NULL,
	type integer NOT NULL,
	public boolean NOT NULL,
	virtual boolean NOT NULL,
	ticket_cost integer NOT NULL,
	slots integer NOT NULL,
	start_time integer NOT NULL,
	end_time integer NOT NULL,
	min_age integer DEFAULT 0,
	created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone,
    CONSTRAINT events_pkey PRIMARY KEY (id)
);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname='invitations') THEN
	CREATE TYPE invitations AS enum ('anyone', 'mutual_follow', 'nobody');
END IF;
END$$;

CREATE TABLE IF NOT EXISTS users
(
    id UUID NOT NULL,
	name varchar NOT NULL,
    username text NOT NULL UNIQUE,
    email text NOT NULL UNIQUE,
    password bytea NOT NULL,
	description varchar(150),
	birth_date timestamp NOT NULL,
	profile_image_url text,
    is_admin boolean DEFAULT false,
	premium boolean DEFAULT false,
	private boolean DEFAULT false,
	invitations invitations DEFAULT 'anyone',
    verified_email boolean DEFAULT false,
    created_at timestamp with time zone DEFAULT NOW(),
    updated_at timestamp with time zone,
    CONSTRAINT users_pkey PRIMARY KEY (id)
);

CREATE UNIQUE INDEX ON users (email, username);

CREATE TABLE IF NOT EXISTS users_locations
(
	user_id UUID NOT NULL,
	country text,
	state text,
	city text,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_permissions
(
 	event_id UUID NOT NULL,
	key varchar(20) NOT NULL,
 	name varchar(20) NOT NULL,
 	description varchar(50),
    created_at timestamp with time zone DEFAULT NOW(),
	FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_roles
(
	event_id UUID NOT NULL,
	name varchar(20) NOT NULL,
 	permissions_keys text NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
	FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_users_roles
(
	event_id UUID NOT NULL,
	user_id UUID NOT NULL,
 	role_name varchar(20) NOT NULL,
	FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE,
 	FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_locations
(
    event_id UUID NOT NULL,
	country text,
	state text,
	city text,
	address text,
    FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_media
(
    id UUID NOT NULL,
    event_id UUID NOT NULL,
	url text NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_products
(
    id UUID NOT NULL,
    event_id UUID NOT NULL,
    stock integer NOT NULL,
    brand text NOT NULL,
	type text NOT NULL,
    description text,
    discount integer,
	taxes integer,
    subtotal integer NOT NULL,
    total integer NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS events_reports
(
	event_id UUID NOT NULL,
	user_id UUID NOT NULL,
	type text NOT NULL,
	details text NOT NULL,
    created_at timestamp with time zone DEFAULT NOW(),
    FOREIGN KEY (event_id) REFERENCES events (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);`

// CREATE INDEX ON events(created_at);
// CREATE INDEX ON users(created_at);
// CREATE INDEX ON events_media(created_at);
// CREATE INDEX ON events_roles(created_at);
// CREATE INDEX ON events_permissions(created_at);
// CREATE INDEX ON events_products(created_at);
// CREATE INDEX ON events_reports(created_at);
