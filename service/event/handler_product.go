package event

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/response"
	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/service/event/product"

	"github.com/julienschmidt/httprouter"
)

// CreateProduct creates an image/video inside an event.
func (h *Handler) CreateProduct() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyProducts); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var product product.Product
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		if err := product.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.CreateProduct(ctx, sqlTx, eventID, product); err != nil {
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

// DeleteProduct removes a product from an event.
func (h *Handler) DeleteProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		ctxParams := httprouter.ParamsFromContext(ctx)
		eventID := ctxParams.ByName("id")
		productID := ctxParams.ByName("product_id")
		if err := validate.ULIDs(eventID, productID); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		errStatus, err := h.service.SQLTx(ctx, false, func(tx *sql.Tx) (int, error) {
			if err := h.requirePermissions(ctx, r, tx, eventID, permissions.ModifyProducts); err != nil {
				return http.StatusForbidden, err
			}
			if err := h.service.DeleteProduct(ctx, tx, eventID, productID); err != nil {
				return http.StatusInternalServerError, err
			}
			return 0, nil
		})
		if err != nil {
			response.Error(w, errStatus, err)
			return
		}

		response.JSONMessage(w, http.StatusOK, eventID)
	}
}

// GetProducts gets the products of an event.
func (h *Handler) GetProducts() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, true)
		defer sqlTx.Rollback()

		if err := h.privacyFilter(ctx, r, sqlTx, eventID); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		params, err := params.ParseQuery(r.URL.RawQuery, params.Product)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		products, err := h.service.GetProducts(ctx, sqlTx, eventID, params)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		var nextCursor string
		if len(products) > 0 {
			nextCursor = products[len(products)-1].ID
		}

		type resp struct {
			NextCursor string            `json:"next_cursor,omitempty"`
			Products   []product.Product `json:"products,omitempty"`
		}
		response.JSON(w, http.StatusOK, resp{NextCursor: nextCursor, Products: products})
	}
}

// UpdateProduct updates a product of an event.
func (h *Handler) UpdateProduct() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		eventID, err := params.IDFromCtx(ctx)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		sqlTx := h.service.BeginSQLTx(ctx, false)
		defer sqlTx.Rollback()

		if err := h.requirePermissions(ctx, r, sqlTx, eventID, permissions.ModifyProducts); err != nil {
			response.Error(w, http.StatusForbidden, err)
			return
		}

		var product product.UpdateProduct
		if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}
		defer r.Body.Close()

		if err := product.Validate(); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}

		if err := h.service.UpdateProduct(ctx, sqlTx, eventID, product); err != nil {
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
