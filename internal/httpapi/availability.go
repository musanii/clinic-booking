package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"clinic-booking/internal/scheduling"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *API {
	return &API{db: db}
}

func (api *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /doctors/{id}/availability", api.getAvailability)
	mux.HandleFunc("POST /appointments", api.createAppointment)
	mux.HandleFunc("PATCH /appointments/{id}/cancel", api.cancelAppointment)
	mux.HandleFunc("PATCH /appointments/{id}/reschedule", api.rescheduleAppointment)
	mux.HandleFunc("GET /patients/{id}/appointments", api.getPatientAppointments)
	return mux
}

func (api *API) getAvailability(w http.ResponseWriter, r *http.Request) {
	doctorID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || doctorID < 1 {
		writeError(w, http.StatusBadRequest, "invalid_doctor_id", "Doctor ID must be a positive integer.")
		return
	}

	dateValue := r.URL.Query().Get("date")
	if dateValue == "" {
		writeError(w, http.StatusBadRequest, "missing_date", "The date query parameter is required (YYYY-MM-DD).")
		return
	}

	location, err := time.LoadLocation("Africa/Nairobi")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load clinic timezone.")
		return
	}

	date, err := time.ParseInLocation("2006-01-02", dateValue, location)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_date", "Date must use YYYY-MM-DD format.")
		return
	}

	date = date.In(location)

	ctx, cancel := timeLimit(r)
	defer cancel()

	slots, err := scheduling.GetAvailability(ctx, api.db, doctorID, date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "availability_failed", "Could not retrieve availability.")
		return
	}

	if slots == nil {
		slots = []scheduling.Slot{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"doctor_id": doctorID,
		"date":      date.Format("2006-01-02"),
		"slots":     slots,
	})
}

func timeLimit(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 5*time.Second)
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: errorDetail{
			Code:    code,
			Message: message,
		},
	})
}
