package scheduling

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGetWorkingHours(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create database pool: %v", err)
	}
	defer db.Close()

	periods, err := GetWorkingHours(ctx, db, 1, 1)
	if err != nil {
		t.Fatalf("get working hours: %v", err)
	}

	if len(periods) != 2 {
		t.Fatalf("expected 2 working periods, got %d", len(periods))
	}

	if periods[0].Start != "09:00:00" || periods[0].End != "12:00:00" {
		t.Errorf("unexpected first period: %+v", periods[0])
	}

	if periods[1].Start != "13:00:00" || periods[1].End != "17:00:00" {
		t.Errorf("unexpected second period: %+v", periods[1])
	}
}