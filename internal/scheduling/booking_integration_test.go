package scheduling

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBookAppointmentAndRejectDuplicate(t *testing.T) {
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

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Create isolated test records inside the transaction.
	var doctorID, patientID int64

	err = tx.QueryRow(ctx,
		`INSERT INTO doctors (name) VALUES ('Booking Test Doctor') RETURNING id`,
	).Scan(&doctorID)
	if err != nil {
		t.Fatalf("create test doctor: %v", err)
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO patients (name) VALUES ('Booking Test Patient') RETURNING id`,
	).Scan(&patientID)
	if err != nil {
		t.Fatalf("create test patient: %v", err)
	}

	location, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		t.Fatal(err)
	}

	// Choose a future weekday and give this doctor working hours
	// that cover the chosen slot.
	now := time.Now().In(location)
	var start time.Time

	for days := 2; days < 10; days++ {
		candidate := time.Date(
			now.Year(), now.Month(), now.Day()+days,
			10, 0, 0, 0, location,
		)
		if candidate.Weekday() >= time.Monday &&
			candidate.Weekday() <= time.Friday {
			start = candidate
			break
		}
	}
	if start.IsZero() {
		t.Fatal("could not find a future weekday")
	}

	weekday := int(start.Weekday())

	_, err = tx.Exec(ctx, `
		INSERT INTO doctor_working_hours
			(doctor_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, '09:00:00', '12:00:00')
	`, doctorID, weekday)
	if err != nil {
		t.Fatalf("create working hours: %v", err)
	}

	// The booking function uses the pool, not this transaction.
	// Therefore this transaction must be committed before calling it.
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit test setup: %v", err)
	}

	// Clean up only these isolated records after the assertions.
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cleanupCancel()

		_, _ = db.Exec(cleanupCtx,
			`DELETE FROM appointments WHERE doctor_id = $1`, doctorID)
		_, _ = db.Exec(cleanupCtx,
			`DELETE FROM doctor_working_hours WHERE doctor_id = $1`, doctorID)
		_, _ = db.Exec(cleanupCtx,
			`DELETE FROM patients WHERE id = $1`, patientID)
		_, _ = db.Exec(cleanupCtx,
			`DELETE FROM doctors WHERE id = $1`, doctorID)
	}()

	appointment, err := BookAppointment(ctx, db, doctorID, patientID, start)
	if err != nil {
		t.Fatalf("book valid appointment: %v", err)
	}

	if appointment.ID < 1 {
		t.Errorf("expected persisted appointment ID, got %d", appointment.ID)
	}
	if appointment.Status != "booked" {
		t.Errorf("expected status booked, got %q", appointment.Status)
	}
	if !appointment.EndsAt.Equal(start.Add(30 * time.Minute)) {
		t.Errorf("expected appointment to end 30 minutes later, got %v", appointment.EndsAt)
	}

	_, err = BookAppointment(ctx, db, doctorID, patientID, start)
	if !errors.Is(err, ErrSlotUnavailable) {
		t.Errorf("expected ErrSlotUnavailable for duplicate slot, got %v", err)
	}
}
