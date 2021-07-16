package report_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/GGP1/groove/service/report"
	"github.com/GGP1/groove/test"

	"github.com/google/uuid"
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
	eventID := uuid.NewString()
	userID := uuid.NewString()

	err := createUser(ctx, userID, "report@email.com", "report")
	assert.NoError(t, err)

	err = createEvent(ctx, eventID, "reports")
	assert.NoError(t, err)

	expectedType := "report"
	t.Run("CreateReport", func(t *testing.T) {
		report := report.Report{
			ReportedID: eventID,
			ReporterID: userID,
			Type:       expectedType,
			Details:    "-",
		}
		err = reportSv.CreateReport(ctx, report)
		assert.NoError(t, err)
	})

	t.Run("GetReports", func(t *testing.T) {
		reports, err := reportSv.GetReports(ctx, eventID)
		assert.NoError(t, err)

		assert.Equal(t, 1, len(reports))
		assert.Equal(t, expectedType, reports[0].Type)
	})
}

func createEvent(ctx context.Context, id, name string) error {
	q := `INSERT INTO events 
	(id, name, type, public, virtual, ticket_cost, slots, start_time, end_Time) 
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := db.ExecContext(ctx, q, id, name, 1, true, false, 10, 100, 15000, 320000)
	return err
}

func createUser(ctx context.Context, id, email, username string) error {
	q := "INSERT INTO users (id, name, email, username, password, birth_date) VALUES ($1,$2,$3,$4,$5,$6)"
	_, err := db.ExecContext(ctx, q, id, "test", email, username, "password", time.Now())

	return err
}
