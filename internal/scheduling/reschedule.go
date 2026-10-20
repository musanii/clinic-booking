package scheduling

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCannotReschedule = errors.New("appointment cannot be rescheduled")

func RescheduleAppointment(
	ctx context.Context,
	db *pgxpool.Pool,
	appointmentID int64,
	newStart time.Time,
) (Appointment, error) {
	if appointmentID < 1 {
		return Appointment{}, ErrAppointmentNotFound
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return Appointment{}, fmt.Errorf("begin reschedule: %w", err)
	}
	defer tx.Rollback(ctx)

	var current Appointment
	err = tx.QueryRow(ctx, `
		SELECT id, doctor_id, patient_id, starts_at, ends_at, status
		FROM appointments
		WHERE id = $1
		FOR UPDATE
	`, appointmentID).Scan(
		&current.ID,
		&current.DoctorID,
		&current.PatientID,
		&current.StartsAt,
		&current.EndsAt,
		&current.Status,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return Appointment{}, ErrAppointmentNotFound
	}
	if err != nil {
		return Appointment{}, fmt.Errorf("load appointment: %w", err)
	}
	if current.Status != "booked" {
		return Appointment{}, ErrCannotReschedule
	}

	start, end, err := validateAppointmentSlot(
		ctx, db, current.DoctorID, current.PatientID, newStart,
	)
	if err != nil {
		return Appointment{}, err
	}

	err = tx.QueryRow(ctx, `
		UPDATE appointments
		SET starts_at = $2,
		    ends_at = $3,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, doctor_id, patient_id, starts_at, ends_at, status
	`, appointmentID, start, end).Scan(
		&current.ID,
		&current.DoctorID,
		&current.PatientID,
		&current.StartsAt,
		&current.EndsAt,
		&current.Status,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Appointment{}, ErrSlotUnavailable
		}
		return Appointment{}, fmt.Errorf("update appointment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Appointment{}, fmt.Errorf("commit reschedule: %w", err)
	}

	return current, nil
}
