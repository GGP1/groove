package product

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/storage/postgres"

	"github.com/julienschmidt/httprouter"
)

// Handler handles ticket service endpoints.
type Handler struct {
	db *sql.DB

	service     Service
	roleService role.Service
}

// NewHandler returns a new ticket handler.
func NewHandler(db *sql.DB, service Service, roleService role.Service) Handler {
	return Handler{
		db:          db,
		service:     service,
		roleService: roleService,
	}
}

// Create creates an image/video inside an event.
func (h Handler) Create() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyProducts); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var product Product
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := product.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Create(ctx, eventID, product); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, product)
	})
}

// Delete removes a product from an event.
func (h Handler) Delete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		productID := ctxParams.ByName("product_id")
		if err := validate.ULIDs(eventID, productID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyProducts); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}
		if err := h.service.Delete(ctx, eventID, productID); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.NoContent(w)
	}
}

// Get gets the products of an event.
func (h Handler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		eventID, err := params.IDFromCtx(rctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, true)
		defer sqlTx.Rollback()

		if err := h.roleService.PrivacyFilter(ctx, r, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.Parse(r.URL.RawQuery, model.Product)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		products, err := h.service.Get(ctx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(products) > 0 {
			nextCursor = products[len(products)-1].ID
		}

		response.JSONCursor(w, nextCursor, "products", products)

	}
}

// Update updates a product of an event.
func (h Handler) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(rctx)
		eventID := ctxParams.ByName("id")
		productID := ctxParams.ByName("product_id")
		if err := validate.ULIDs(eventID, productID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx, ctx := postgres.BeginTx(rctx, h.db, false)
		defer sqlTx.Rollback()

		if err := h.roleService.RequirePermissions(ctx, r, eventID, permissions.ModifyProducts); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var product UpdateProduct
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		if err := product.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.Update(ctx, eventID, productID, product); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		if err := sqlTx.Commit(); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, eventID)
	}
}
