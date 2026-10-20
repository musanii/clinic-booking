package scheduling

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAppointmentNotFound = errors.New("appointment not found")
	ErrAlreadyCancelled    = errors.New("appointment already cancelled")
	ErrInvalidReason       = errors.New("cancellation reason is required")
)

func CancelAppointment(
	ctx context.Context,
	db *pgxpool.Pool,
	appointmentID int64,
	reason string,
) error {
	if appointmentID < 1 {
		return ErrAppointmentNotFound
	}
	if reason == "" {
		return ErrInvalidReason
	}

	tag, err := db.Exec(ctx, `
		UPDATE appointments
		SET status = 'cancelled',
		    cancellation_reason = $2,
		    updated_at = NOW()
		WHERE id = $1 AND status = 'booked'
	`, appointmentID, reason)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return fmt.Errorf("cancel appointment: %w", err)
		}
		return fmt.Errorf("cancel appointment: %w", err)
	}

	if tag.RowsAffected() == 1 {
		return nil
	}

	var status string
	err = db.QueryRow(ctx,
		`SELECT status FROM appointments WHERE id = $1`,
		appointmentID,
	).Scan(&status)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAppointmentNotFound
	}
	if err != nil {

		return fmt.Errorf("check appointment status: %w", err)
	}
	if status == "cancelled" {
		return ErrAlreadyCancelled
	}

	return ErrAppointmentNotFound
}
