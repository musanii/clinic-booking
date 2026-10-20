package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"


	"clinic-booking/internal/scheduling"
)

func (api *API) getPatientAppointments(w http.ResponseWriter, r *http.Request) {
	patientID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || patientID < 1 {
		writeError(w, http.StatusBadRequest, "invalid_patient_id",
			"Patient ID must be a positive integer.")
		return
	}

	ctx, cancel := timeLimit(r)
	defer cancel()

	var exists bool
	err = api.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM patients WHERE id = $1)`,
		patientID,
	).Scan(&exists)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database_error",
			"Could not retrieve patient.")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "patient_not_found",
			"Patient not found.")
		return
	}

	rows, err := api.db.Query(ctx, `
		SELECT id, doctor_id, patient_id, starts_at, ends_at, status
		FROM appointments
		WHERE patient_id = $1
		  AND status = 'booked'
		  AND starts_at > NOW()
		ORDER BY starts_at ASC
	`, patientID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database_error",
			"Could not retrieve appointments.")
		return
	}
	defer rows.Close()

	appointments := make([]scheduling.Appointment, 0)

	for rows.Next() {
		var appointment scheduling.Appointment

		if err := rows.Scan(
			&appointment.ID,
			&appointment.DoctorID,
			&appointment.PatientID,
			&appointment.StartsAt,
			&appointment.EndsAt,
			&appointment.Status,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "database_error",
				"Could not read appointments.")
			return
		}

		appointments = append(appointments, appointment)
	}

	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "database_error",
			"Could not read appointments.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"patient_id":   patientID,
		"appointments": appointments,
	})
}
