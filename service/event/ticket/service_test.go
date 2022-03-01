package ticket_test

import (
	"context"
	"database/sql"
	"log"
	"testing"

	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/event/ticket"
	"github.com/GGP1/groove/test"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

var (
	db       *sql.DB
	rdb      *redis.Client
	ticketSv ticket.Service
	ctx      context.Context
)

func TestMain(m *testing.M) {
	test.Main(
		m,
		func(s *sql.DB, r *redis.Client) {
			db = s
			rdb = r
			roleSv := role.NewService(db, r)
			ticketSv = ticket.NewService(db, r, roleSv)
			tx, err := s.Begin()
			if err != nil {
				log.Fatal(err)
			}
			ctx = txgroup.NewContext(context.Background(), txgroup.NewSQLTx(tx))
		},
		test.Postgres, test.Redis,
	)
}

func TestTicketService(t *testing.T) {
	eventID := test.CreateEvent(t, db, "ticketeer")
	userID := test.CreateUser(t, db, "email@mail.com", "username")
	role := model.Role{Name: "role", PermissionKeys: []string{"key"}}
	test.CreateRole(t, db, eventID, role)
	session := auth.Session{
		ID:          ulid.NewString(),
		Username:    "Gasti",
		DeviceToken: "",
		Type:        model.Personal,
	}
	ticketName := "test"
	availableCount := uint64(10)
	createTicket := model.Ticket{
		Name:           ticketName,
		Description:    "",
		AvailableCount: &availableCount,
		Cost:           &availableCount,
		LinkedRole:     role.Name,
	}

	t.Run("Create", func(t *testing.T) {
		err := ticketSv.Create(ctx, eventID, createTicket)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)
	})

	t.Run("Available", func(t *testing.T) {
		gotAvCount, err := ticketSv.Available(ctx, eventID, ticketName)
		assert.NoError(t, err)
		assert.Equal(t, int64(availableCount), gotAvCount)
	})

	t.Run("Get", func(t *testing.T) {
		tickets, err := ticketSv.Get(ctx, eventID)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(tickets))
		assert.Equal(t, createTicket, tickets[0])
	})

	t.Run("GetByName", func(t *testing.T) {
		gotTicket, err := ticketSv.GetByName(ctx, eventID, ticketName)
		assert.NoError(t, err)
		assert.Equal(t, createTicket, gotTicket)
	})

	t.Run("Buy", func(t *testing.T) {
		err := ticketSv.Buy(ctx, session, eventID, ticketName, []string{userID})
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		gotAvCount, err := ticketSv.Available(ctx, eventID, ticketName)
		assert.NoError(t, err)
		assert.Equal(t, int64(availableCount-1), gotAvCount)
	})

	t.Run("Refund", func(t *testing.T) {
		err := ticketSv.Refund(ctx, session, eventID, ticketName)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		gotAvCount, err := ticketSv.Available(ctx, eventID, ticketName)
		assert.NoError(t, err)
		assert.Equal(t, int64(availableCount), gotAvCount)
	})

	t.Run("Update", func(t *testing.T) {
		desc := "description"
		updateTicket := model.UpdateTicket{Description: &desc}
		err := ticketSv.Update(ctx, eventID, ticketName, updateTicket)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		gotTicket, err := ticketSv.GetByName(ctx, eventID, ticketName)
		assert.NoError(t, err)
		assert.Equal(t, desc, gotTicket.Description)
	})

	t.Run("Delete", func(t *testing.T) {
		err := ticketSv.Delete(ctx, eventID, ticketName)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		_, err = ticketSv.GetByName(ctx, eventID, ticketName)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}
