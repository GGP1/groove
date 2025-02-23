package report

import (
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"

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

// Create creates a new report inside an event.
func (h *Handler) Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var report model.CreateReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		reportID, err := h.service.Create(ctx, report)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, response.ID{ID: reportID})
	}
}

// Get gets an event's reports.
func (h *Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		reportedID := httprouter.ParamsFromContext(ctx).ByName("reported_id")
		if err := validate.ULID(reportedID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		reports, err := h.service.Get(ctx, reportedID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, reports)
	}
}

// GetByID looks for a report by its id and returns it.
func (h *Handler) GetByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		id, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		report, err := h.service.GetByID(ctx, id)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, report)
	}
}
