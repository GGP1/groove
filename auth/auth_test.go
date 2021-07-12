package auth_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/GGP1/groove/auth"
	"github.com/GGP1/groove/config"
	"github.com/GGP1/groove/test"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	db := test.StartPostgres(t)
	dc := test.StartDgraph(t)
	rdb := test.StartRedis(t)
	session := auth.NewSession(db, rdb, config.Sessions{VerifyEmails: false})

	ctx := context.Background()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	email := "test@test.com"
	password := "test"

	err := test.CreateUser(ctx, db, dc, uuid.NewString(), email, "username", password)
	assert.NoError(t, err)

	err = session.Login(ctx, w, r, email, password)
	assert.NoError(t, err)

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
