package report

import (
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/internal/validate"

	"github.com/julienschmidt/httprouter"
)

// Handler handles events endpoints.
type Handler struct {
	service Service
}

// NewHandler returns an event handler.
func NewHandler(service Service) Handler {
	return Handler{
		service: service,
	}
}

// CreateReport creates a new report inside an event.
func (h *Handler) CreateReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var report Report
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		report.ID = ulid.NewString()
		if err := h.service.CreateReport(ctx, report); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, report)
	}
}

// GetReports gets an event's reports.
func (h *Handler) GetReports() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		reportedID := httprouter.ParamsFromContext(ctx).ByName("reported_id")
		if err := validate.ULID(reportedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		reports, err := h.service.GetReports(ctx, reportedID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, reports)
	}
}
