package report_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/report"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

var (
	reportSv report.Service
	db       *sql.DB
)

func TestMain(m *testing.M) {
	poolPg, resourcePg, postgres, err := test.RunPostgres()
	if err != nil {
		log.Fatal(err)
	}

	db = postgres
	reportSv = report.NewService(postgres)

	code := m.Run()

	if err := poolPg.Purge(resourcePg); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestReports(t *testing.T) {
	ctx := context.Background()
	eventID := ulid.NewString()
	userID := ulid.NewString()

	err := createUser(ctx, userID, "report@email.com", "report")
	assert.NoError(t, err)

	err = createEvent(ctx, eventID, "reports")
	assert.NoError(t, err)

	expectedType := "report"
	t.Run("CreateReport", func(t *testing.T) {
		report := report.CreateReport{
			ReportedID: eventID,
			ReporterID: userID,
			Type:       expectedType,
			Details:    "-",
		}
		err = reportSv.Create(ctx, report)
		assert.NoError(t, err)
	})

	t.Run("GetReports", func(t *testing.T) {
		reports, err := reportSv.Get(ctx, eventID)
		assert.NoError(t, err)

		assert.Equal(t, 1, len(reports))
		assert.Equal(t, expectedType, reports[0].Type)
	})
}

func createEvent(ctx context.Context, id, name string) error {
	q := `INSERT INTO events 
	(id, name, type, public, virtual, slots, cron) 
	VALUES ($1,$2,$3,$4,$5,$6,$7)`
	_, err := db.ExecContext(ctx, q, id, name, 1, true, false, 100, "15 20 5 12 2 120")
	return err
}

func createUser(ctx context.Context, id, email, username string) error {
	q := "INSERT INTO users (id, name, email, username, password, birth_date) VALUES ($1,$2,$3,$4,$5,$6)"
	_, err := db.ExecContext(ctx, q, id, "test", email, username, "password", time.Now())

	return err
}
