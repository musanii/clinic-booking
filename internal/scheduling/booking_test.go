package scheduling

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBookAppointmentValidation(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create database pool: %v", err)
	}
	defer db.Close()

	location, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		t.Fatal(err)
	}

	// Invalid IDs should be rejected before any database lookup.
	_, err = BookAppointment(ctx, db, 0, 1, time.Now().In(location).Add(48*time.Hour))
	if !errors.Is(err, ErrInvalidBooking) {
		t.Fatalf("expected ErrInvalidBooking for invalid doctor ID, got %v", err)
	}

	// A time less than one hour from now must be rejected.
	tooSoon := time.Now().In(location).Add(30 * time.Minute)
	tooSoon = time.Date(
		tooSoon.Year(), tooSoon.Month(), tooSoon.Day(),
		tooSoon.Hour(), tooSoon.Minute(), 0, 0, location,
	)

	_, err = BookAppointment(ctx, db, 1, 1, tooSoon)
	if !errors.Is(err, ErrInvalidBooking) {
		t.Errorf("expected ErrInvalidBooking for a booking too soon, got %v", err)
	}

	// A time outside the doctor's working hours must be rejected.
	future := time.Now().In(location).AddDate(0, 0, 7)
	future = time.Date(
		future.Year(), future.Month(), future.Day(),
		18, 0, 0, 0, location,
	)

	_, err = BookAppointment(ctx, db, 1, 1, future)
	if !errors.Is(err, ErrInvalidBooking) {
		t.Errorf("expected ErrInvalidBooking outside working hours, got %v", err)
	}
}
