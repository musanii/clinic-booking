package httpapi

import (
	"clinic-booking/internal/scheduling"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

type CancelAppointmentRequest struct {
	Reason string `json:"reason"`
}

func (api *API) cancelAppointment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_appointment_id",
			"Appointment ID must be a positive integer.")
		return
	}

	var request CancelAppointmentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json",
			"Provide a cancellation reason.")
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json",
			"Request body must contain exactly one JSON object.")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json",
			"Provide a cancellation reason.")
		return
	}

	ctx, cancel := timeLimit(r)
	defer cancel()

	err = scheduling.CancelAppointment(ctx, api.db, id, request.Reason)

	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, scheduling.ErrInvalidReason):
		writeError(w, http.StatusBadRequest, "invalid_reason",
			"A cancellation reason is required.")
	case errors.Is(err, scheduling.ErrAlreadyCancelled):
		writeError(w, http.StatusConflict, "already_cancelled",
			"This appointment has already been cancelled.")
	case errors.Is(err, scheduling.ErrAppointmentNotFound):
		writeError(w, http.StatusNotFound, "appointment_not_found",
			"Appointment not found.")
	default:
		writeError(w, http.StatusInternalServerError, "cancellation_failed",
			"Could not cancel the appointment.")
	}
}
