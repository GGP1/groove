package auth_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	db := test.StartPostgres(t)
	rdb := test.StartRedis(t)
	session := auth.NewService(db, rdb, config.Sessions{VerifyEmails: false})

	ctx := context.Background()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	email := "test@test.com"
	password := "test"
	q := "INSERT INTO users (id, name, email, username, password, birth_date, type, invitations) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)"

	_, err := db.ExecContext(context.Background(), q, ulid.NewString(), "test", email, "random", password, time.Now(), model.Personal, model.Friends)
	if err != nil {
		t.Fatal(err)
	}

	login := model.Login{Username: email, Password: password}
	user, err := session.Login(ctx, w, r, login)
	assert.NoError(t, err)
	assert.Equal(t, email, user.Email)

	// Add cookies from the recorder to the request
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}

	_, ok := session.AlreadyLoggedIn(ctx, r)
	assert.True(t, ok)

	err = session.Logout(ctx, w, r)
	assert.NoError(t, err)

	_, ok2 := session.AlreadyLoggedIn(ctx, r)
	assert.False(t, ok2)
}
