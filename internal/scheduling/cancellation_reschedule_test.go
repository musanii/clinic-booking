package scheduling

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCancellationAndRescheduling(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

	// Create dedicated records for this test.
	var doctorID, patientID int64

	err = db.QueryRow(ctx,
		`INSERT INTO doctors (name) VALUES ('Cancel Reschedule Test Doctor') RETURNING id`,
	).Scan(&doctorID)
	if err != nil {
		t.Fatalf("create doctor: %v", err)
	}

	err = db.QueryRow(ctx,
		`INSERT INTO patients (name) VALUES ('Cancel Reschedule Test Patient') RETURNING id`,
	).Scan(&patientID)
	if err != nil {
		_, _ = db.Exec(ctx, `DELETE FROM doctors WHERE id = $1`, doctorID)
		t.Fatalf("create patient: %v", err)
	}

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

	// Pick a future weekday.
	now := time.Now().In(location)
	var firstStart time.Time
	for days := 2; days < 10; days++ {
		candidate := time.Date(
			now.Year(), now.Month(), now.Day()+days,
			10, 0, 0, 0, location,
		)
		if candidate.Weekday() >= time.Monday &&
			candidate.Weekday() <= time.Friday {
			firstStart = candidate
			break
		}
	}
	if firstStart.IsZero() {
		t.Fatal("could not find a future weekday")
	}

	// Give the test doctor working hours on that weekday.
	weekday := int(firstStart.Weekday())
	_, err = db.Exec(ctx, `
		INSERT INTO doctor_working_hours
			(doctor_id, day_of_week, start_time, end_time)
		VALUES ($1, $2, '09:00:00', '12:00:00')
	`, doctorID, weekday)
	if err != nil {
		t.Fatalf("create working hours: %v", err)
	}

	// Book the first appointment.
	first, err := BookAppointment(ctx, db, doctorID, patientID, firstStart)
	if err != nil {
		t.Fatalf("book first appointment: %v", err)
	}

	// Reschedule to another slot on the same working day.
	secondStart := firstStart.Add(30 * time.Minute)
	rescheduled, err := RescheduleAppointment(ctx, db, first.ID, secondStart)
	if err != nil {
		t.Fatalf("reschedule appointment: %v", err)
	}
	if !rescheduled.StartsAt.Equal(secondStart) {
		t.Errorf("expected new start %v, got %v", secondStart, rescheduled.StartsAt)
	}

	// Cancel the appointment.
	if err := CancelAppointment(ctx, db, first.ID, "Test cancellation"); err != nil {
		t.Fatalf("cancel appointment: %v", err)
	}

	// Cancelling it again must return the expected error.
	err = CancelAppointment(ctx, db, first.ID, "Repeated cancellation")
	if !errors.Is(err, ErrAlreadyCancelled) {
		t.Errorf("expected ErrAlreadyCancelled, got %v", err)
	}

	// A cancelled appointment must not be rescheduled.
	_, err = RescheduleAppointment(ctx, db, first.ID, firstStart)
	if !errors.Is(err, ErrCannotReschedule) {
		t.Errorf("expected ErrCannotReschedule, got %v", err)
	}

	// An invalid reason must be rejected.
	err = CancelAppointment(ctx, db, first.ID, "")
	if !errors.Is(err, ErrInvalidReason) {
		t.Errorf("expected ErrInvalidReason, got %v", err)
	}
}
