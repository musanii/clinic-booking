package scheduling

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkingPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func GetWorkingHours(
	ctx context.Context,
	db *pgxpool.Pool,
	doctorID int64,
	weekday int,
) ([]WorkingPeriod, error) {
	if weekday < 1 || weekday > 7 {
		return nil, fmt.Errorf("weekday must be between 1 and 7")
	}

	rows, err := db.Query(ctx, `
		SELECT
			TO_CHAR(start_time, 'HH24:MI:SS'),
			TO_CHAR(end_time, 'HH24:MI:SS')
		FROM doctor_working_hours
		WHERE doctor_id = $1
		  AND day_of_week = $2
		ORDER BY start_time
	`, doctorID, weekday)
	if err != nil {
		return nil, fmt.Errorf("query working hours: %w", err)
	}
	defer rows.Close()

	var periods []WorkingPeriod

	for rows.Next() {
		var period WorkingPeriod

		if err := rows.Scan(&period.Start, &period.End); err != nil {
			return nil, fmt.Errorf("scan working hours: %w", err)
		}

		periods = append(periods, period)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate working hours: %w", err)
	}

	return periods, nil
}