package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cookie"
	"github.com/GGP1/groove/internal/httperr"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/userip"
	"github.com/GGP1/sqan"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var errAccessDenied = errors.New("Access denied")

// Service provides auth operations.
type Service interface {
	AlreadyLoggedIn(ctx context.Context, r *http.Request) (Session, bool)
	Login(ctx context.Context, w http.ResponseWriter, r *http.Request, login Login) (userSession, error)
	Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	TokensFromID(ctx context.Context, id string) []string
}

type service struct {
	db     *sql.DB
	rdb    *redis.Client
	config config.Sessions
}

// NewService creates a new session with the necessary dependencies.
func NewService(db *sql.DB, rdb *redis.Client, config config.Sessions) Service {
	return &service{
		db:     db,
		rdb:    rdb,
		config: config,
	}
}

// AlreadyLoggedIn returns if the user is logged in or not.
func (s service) AlreadyLoggedIn(ctx context.Context, r *http.Request) (Session, bool) {
	session, err := GetSession(ctx, r)
	if err != nil {
		return Session{}, false
	}

	ok, err := s.rdb.SIsMember(ctx, session.ID, session.DeviceToken).Result()
	if err != nil {
		return Session{}, false
	}

	// If the device token isn't part of the set it means the cookie was modified since the log in
	return session, ok
}

// Login attempts to log a user in.
func (s service) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, login Login) (userSession, error) {
	// Won't collide with the rate limiter as this last has the prefix "rate:"
	ip := userip.Get(ctx, r)
	attempts, err := s.rdb.Get(ctx, ip).Int64()
	if err != nil && err != redis.Nil {
		return userSession{}, errors.Wrap(err, "retrieving client attempts")
	}

	if attempts > 4 {
		return userSession{}, httperr.Forbidden(fmt.Sprintf("please wait %v before trying again", delay(attempts)))
	}

	query := `SELECT 
	id, email, username, password, verified_email, profile_image_url, type
	FROM users 
	WHERE username=$1 OR email=$1`
	rows, err := s.db.QueryContext(ctx, query, login.Username)
	if err != nil {
		return userSession{}, errors.Wrap(err, "querying user credentials")
	}

	var user userSession
	if err := sqan.Row(&user, rows); err != nil {
		_ = s.addDelay(ctx, ip)
		log.Debug("database error", zap.Error(err))
		return userSession{}, httperr.Forbidden("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(login.Password)); err != nil {
		_ = s.addDelay(ctx, ip)
		log.Debug("password mismatch", zap.Error(err))
		return userSession{}, httperr.Forbidden("invalid email or password")
	}

	if s.config.VerifyEmails && !user.VerifiedEmail {
		return userSession{}, httperr.Forbidden("please verify your email before logging in")
	}

	if login.DeviceToken == "" {
		salt := make([]byte, saltLen)
		if _, err := rand.Read(salt); err != nil {
			return userSession{}, errors.Wrap(err, "generating salt")
		}
		login.DeviceToken = string(salt)
	}

	id := user.ID.String()
	if err := s.rdb.SAdd(ctx, id, login.DeviceToken).Err(); err != nil {
		return userSession{}, errors.Wrap(err, "storing session")
	}
	cookieValue := parseSessionToken(id, login.Username, login.DeviceToken, user.Type)
	if err := cookie.Set(w, cookie.Session, cookieValue, "/"); err != nil {
		return userSession{}, errors.Wrap(err, "setting cookie")
	}

	return user, nil
}

// Logout removes the user session and its cookies.
func (s service) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// Ignore error as the session is already loaded in context
	session, _ := GetSession(ctx, r)
	cookie.Delete(w, cookie.Session)

	if err := s.rdb.SRem(ctx, session.ID, session.DeviceToken).Err(); err != nil {
		return errors.Wrap(err, "deleting session")
	}
	return nil
}

// TokensFromID returns a list of device tokens corresponding to the user with the id passed.
func (s service) TokensFromID(ctx context.Context, id string) []string {
	// Here we return all the tokens, not only the one stored in the user session.
	tokens := s.rdb.SMembers(ctx, id).Val()
	// Remove salts
	for i, token := range tokens {
		if len(token) == saltLen {
			tokens = append(tokens[:i], tokens[i+1:]...)
		}
	}
	return tokens
}

func (s *service) addDelay(ctx context.Context, key string) error {
	v := s.rdb.Incr(ctx, key).Val()
	return s.rdb.Expire(ctx, key, delay(v)).Err()
}

// delay in seconds given n, where n is the number of attempts.
func delay(n int64) time.Duration { return time.Duration(n*2) * time.Second }
