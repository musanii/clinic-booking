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

var (
	ErrInvalidBooking  = errors.New("invalid booking")
	ErrSlotUnavailable = errors.New("slot unavailable")
	ErrNotFound        = errors.New("doctor or patient not found")
)

type Appointment struct {
	ID        int64     `json:"id"`
	DoctorID  int64     `json:"doctor_id"`
	PatientID int64     `json:"patient_id"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	Status    string    `json:"status"`
}

func validateAppointmentSlot(
	ctx context.Context,
	db *pgxpool.Pool,
	doctorID, patientID int64,
	start time.Time,
) (time.Time, time.Time, error) {
	if doctorID < 1 || patientID < 1 {
		return time.Time{}, time.Time{}, ErrInvalidBooking
	}

	location, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("load clinic timezone: %w", err)
	}

	start = start.In(location)
	now := time.Now().In(location)

	if start.Second() != 0 || start.Nanosecond() != 0 ||
		start.Minute()%30 != 0 || start.Before(now.Add(time.Hour)) {
		return time.Time{}, time.Time{}, ErrInvalidBooking
	}

	var doctorExists, patientExists bool

	err = db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM doctors WHERE id = $1 AND active = TRUE)`,
		doctorID,
	).Scan(&doctorExists)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("check doctor: %w", err)
	}

	err = db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM patients WHERE id = $1)`,
		patientID,
	).Scan(&patientExists)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("check patient: %w", err)
	}

	if !doctorExists || !patientExists {
		return time.Time{}, time.Time{}, ErrNotFound
	}

	weekday := int(start.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	periods, err := GetWorkingHours(ctx, db, doctorID, weekday)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	end := start.Add(30 * time.Minute)
	withinWorkingHours := false

	for _, period := range periods {
		periodStart, err := time.ParseInLocation("15:04:05", period.Start, location)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("parse working period start: %w", err)
		}

		periodEnd, err := time.ParseInLocation("15:04:05", period.End, location)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("parse working period end: %w", err)
		}

		open := time.Date(start.Year(), start.Month(), start.Day(),
			periodStart.Hour(), periodStart.Minute(), periodStart.Second(), 0, location)
		close := time.Date(start.Year(), start.Month(), start.Day(),
			periodEnd.Hour(), periodEnd.Minute(), periodEnd.Second(), 0, location)

		if !start.Before(open) && !end.After(close) {
			withinWorkingHours = true
			break
		}
	}

	if !withinWorkingHours {
		return time.Time{}, time.Time{}, ErrInvalidBooking
	}

	return start, end, nil
}

func BookAppointment(
	ctx context.Context,
	db *pgxpool.Pool,
	doctorID, patientID int64,
	start time.Time,
) (Appointment, error) {
	start, end, err := validateAppointmentSlot(ctx, db, doctorID, patientID, start)
	if err != nil {
		return Appointment{}, err
	}

	var appointment Appointment

	err = db.QueryRow(ctx, `
		INSERT INTO appointments
			(doctor_id, patient_id, starts_at, ends_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, doctor_id, patient_id, starts_at, ends_at, status
	`, doctorID, patientID, start, end).Scan(
		&appointment.ID,
		&appointment.DoctorID,
		&appointment.PatientID,
		&appointment.StartsAt,
		&appointment.EndsAt,
		&appointment.Status,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Appointment{}, ErrSlotUnavailable
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return Appointment{}, ErrNotFound
		}
		return Appointment{}, fmt.Errorf("insert appointment: %w", err)
	}

	return appointment, nil
}
