package auth

import (
	"context"
	"crypto/rand"
	"net/http"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/cookie"
	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/userip"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Session provides auth operations.
type Session interface {
	AlreadyLoggedIn(ctx context.Context, r *http.Request) (SessionInfo, bool)
	Login(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) error
	Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) error
}

type session struct {
	db           *sqlx.DB
	rdb          *redis.Client
	expiration   time.Duration
	verifyEmails bool
}

// NewSession creates a new session with the necessary dependencies.
func NewSession(db *sqlx.DB, rdb *redis.Client, config config.Sessions) Session {
	return &session{
		db:           db,
		rdb:          rdb,
		expiration:   config.Expiration,
		verifyEmails: config.VerifyEmails,
	}
}

// AlreadyLoggedIn returns if the user is logged in or not.
func (s *session) AlreadyLoggedIn(ctx context.Context, r *http.Request) (SessionInfo, bool) {
	sessionInfo, err := GetSessionInfo(ctx, r)
	if err != nil {
		return SessionInfo{}, false
	}

	res, err := s.rdb.Get(ctx, sessionInfo.ID).Result()
	if err != nil {
		return SessionInfo{}, false
	}

	// If the salt doens't match it means the cookie was modified since the log in
	return sessionInfo, res == sessionInfo.Salt
}

// Login attempts to log a user in.
func (s *session) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) error {
	// Won't collide with the rate limiter as this last has the prefix "rate:"
	ip := userip.Get(ctx, r)
	attempts, err := s.rdb.Get(ctx, ip).Int64()
	if err != nil && err != redis.Nil {
		return errors.Wrap(err, "retrieving client attempts")
	}

	if attempts > 4 {
		return errors.Errorf("please wait %v before trying again", delay(attempts))
	}

	query := "SELECT id, email, password, premium, verified_email FROM users WHERE email=$1"
	row := s.db.QueryRowContext(ctx, query, email)
	var user userSession
	err = row.Scan(&user.ID, &user.Email, &user.Password, &user.Premium, &user.VerifiedEmail)
	if err != nil {
		_ = s.addDelay(ctx, ip)
		log.Debug("database error", zap.Error(err))
		return errors.New("invalid email or password")
	}

	if s.verifyEmails && !user.VerifiedEmail {
		return errors.New("please verify your email before logging in")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		_ = s.addDelay(ctx, ip)
		log.Debug("password mismatch", zap.Error(err))
		return errors.New("invalid email or password")
	}

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return errors.Wrap(err, "generating salt")
	}

	id := user.ID.String()
	if err := s.rdb.Set(ctx, id, salt, s.expiration).Err(); err != nil {
		return errors.Wrap(err, "storing session")
	}
	cookieValue := parseSessionInfo(id, salt, user.Premium)
	if err := cookie.Set(w, cookie.Session, cookieValue, "/"); err != nil {
		return errors.Wrap(err, "setting cookie")
	}
	return nil
}

func (s *session) addDelay(ctx context.Context, key string) error {
	v := s.rdb.Incr(ctx, key).Val()
	return s.rdb.Expire(ctx, key, delay(v)).Err()
}

// Logout removes the user session and its cookies.
func (s *session) Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	sessionInfo, _ := GetSessionInfo(ctx, r)
	cookie.Delete(w, cookie.Session)
	if err := s.rdb.Del(ctx, sessionInfo.ID).Err(); err != nil {
		return errors.Wrap(err, "deleting the session")
	}
	return nil
}

// delay in seconds given n, where n is the number of attempts.
func delay(n int64) time.Duration { return time.Duration(n*2) * time.Second }
