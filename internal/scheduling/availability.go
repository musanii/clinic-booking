
package scheduling

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Slot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func GetAvailability(
	ctx context.Context,
	db *pgxpool.Pool,
	doctorID int64,
	date time.Time,
) ([]Slot, error) {
	location, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		return nil, fmt.Errorf("load clinic timezone: %w", err)
	}

	date = date.In(location)
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	periods, err := GetWorkingHours(ctx, db, doctorID, weekday)
	if err != nil {
		return nil, err
	}

	bookedRows, err := db.Query(ctx, `
		SELECT starts_at
		FROM appointments
		WHERE doctor_id = $1
		  AND status = 'booked'
		  AND starts_at >= $2
		  AND starts_at < $3
	`, doctorID,
		time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location),
		time.Date(date.Year(), date.Month(), date.Day()+1, 0, 0, 0, 0, location),
	)
	if err != nil {
		return nil, fmt.Errorf("query booked appointments: %w", err)
	}
	defer bookedRows.Close()

	booked := make(map[int64]bool)

	for bookedRows.Next() {
		var start time.Time
		if err := bookedRows.Scan(&start); err != nil {
			return nil, fmt.Errorf("scan booked appointment: %w", err)
		}
		booked[start.Unix()] = true
	}

	if err := bookedRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate booked appointments: %w", err)
	}

	var available []Slot

	for _, period := range periods {
		startTime, err := time.ParseInLocation("15:04:05", period.Start, location)
		if err != nil {
			return nil, fmt.Errorf("parse working period start: %w", err)
		}

		endTime, err := time.ParseInLocation("15:04:05", period.End, location)
		if err != nil {
			return nil, fmt.Errorf("parse working period end: %w", err)
		}

		slotStart := time.Date(
			date.Year(), date.Month(), date.Day(),
			startTime.Hour(), startTime.Minute(), startTime.Second(), 0,
			location,
		)

		periodEnd := time.Date(
			date.Year(), date.Month(), date.Day(),
			endTime.Hour(), endTime.Minute(), endTime.Second(), 0,
			location,
		)

		for slotStart.Add(30 * time.Minute).Compare(periodEnd) <= 0 {
			slotEnd := slotStart.Add(30 * time.Minute)

			if !booked[slotStart.Unix()] {
				available = append(available, Slot{
					Start: slotStart,
					End:   slotEnd,
				})
			}

			slotStart = slotEnd
		}
	}

	return available, nil
}
