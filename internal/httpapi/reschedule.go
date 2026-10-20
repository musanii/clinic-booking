package httpapi

import (
	"clinic-booking/internal/scheduling"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

type rescheduleAppointmentRequest struct {
	StartsAt time.Time `json:"starts_at"`
}

func (api *API) rescheduleAppointment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_appointment_id",
			"Appointment ID must be a positive integer.")
		return
	}

	var request rescheduleAppointmentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json",
			"Provide starts_at as a valid timestamp.")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json",
			"Request body must contain exactly one JSON object.")
		return
	}

	ctx, cancel := timeLimit(r)
	defer cancel()

	appointment, err := scheduling.RescheduleAppointment(
		ctx, api.db, id, request.StartsAt,
	)

	switch {
	case err == nil:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(appointment)
	case errors.Is(err, scheduling.ErrInvalidBooking):
		writeError(w, http.StatusBadRequest, "invalid_booking",
			"The requested time is invalid, outside working hours, or too soon.")
	case errors.Is(err, scheduling.ErrNotFound):
		writeError(w, http.StatusNotFound, "doctor_or_patient_not_found",
			"The doctor or patient does not exist or the doctor is inactive.")
	case errors.Is(err, scheduling.ErrAppointmentNotFound):
		writeError(w, http.StatusNotFound, "appointment_not_found",
			"Appointment not found.")
	case errors.Is(err, scheduling.ErrCannotReschedule):
		writeError(w, http.StatusConflict, "cannot_reschedule",
			"Cancelled appointments cannot be rescheduled.")
	case errors.Is(err, scheduling.ErrSlotUnavailable):
		writeError(w, http.StatusConflict, "slot_unavailable",
			"This appointment slot is already booked.")
	default:
		writeError(w, http.StatusInternalServerError, "reschedule_failed",
			"Could not reschedule the appointment.")
	}
}
