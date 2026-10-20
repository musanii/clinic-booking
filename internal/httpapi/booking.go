package httpapi

import (
	"clinic-booking/internal/scheduling"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

type createAppointmentRequest struct {
	DoctorID  int64     `json:"doctor_id"`
	PatientID int64     `json:"patient_id"`
	StartsAt  time.Time `json:"starts_at"`
}

func (api *API) createAppointment(w http.ResponseWriter, r *http.Request) {
	var request createAppointmentRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json",
			"Provide doctor_id, patient_id, and starts_at as valid JSON.")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json",
			"Request body must contain exactly one JSON object.")
		return
	}

	ctx, cancel := timeLimit(r)
	defer cancel()

	appointment, err := scheduling.BookAppointment(
		ctx,
		api.db,
		request.DoctorID,
		request.PatientID,
		request.StartsAt,
	)
	if err != nil {
		switch {
		case errors.Is(err, scheduling.ErrInvalidBooking):
			writeError(w, http.StatusBadRequest, "invalid_booking",
				"The requested time is invalid, outside working hours, or too soon.")
		case errors.Is(err, scheduling.ErrNotFound):
			writeError(w, http.StatusNotFound, "doctor_or_patient_not_found",
				"The doctor or patient does not exist or the doctor is inactive.")
		case errors.Is(err, scheduling.ErrSlotUnavailable):
			writeError(w, http.StatusConflict, "slot_unavailable",
				"This appointment slot is already booked.")
		default:
			writeError(w, http.StatusInternalServerError, "booking_failed",
				"Could not create the appointment.")

		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(appointment)
}
