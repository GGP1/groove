package media

import (
	"encoding/json"
	"mime/multipart"
	"net/http"

	"github.com/GGP1/groove/internal/response"
)

// TODO: store (both?) structs inside /model

// Media ..
type Media struct {
	File       multipart.File
	FileHeader *multipart.FileHeader
	Bucket     string
}

type resp struct {
	URL string
}

// Handler ..
type Handler struct {
	service Service
}

// NewHandler ..
func NewHandler() Handler {
	return Handler{}
}

// Upload ..
func (h Handler) Upload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := r.ParseMultipartForm(maxSize)
		if err != nil {
			response.Error(w, http.StatusBadRequest, errImageTooLarge)
			return
		}

		file, fileHeader, err := r.FormFile("profile_picture")
		if err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer file.Close()

		m := Media{
			File:       file,
			FileHeader: fileHeader,
		}
		if err := json.NewDecoder(r.Body).Decode(&m.Bucket); err != nil {
			response.Error(w, http.StatusBadRequest, err)
			return
		}
		defer r.Body.Close()

		id, err := h.service.Upload(ctx, m)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err)
			return
		}

		response.JSON(w, http.StatusOK, response.ID{ID: id})
	}
}
