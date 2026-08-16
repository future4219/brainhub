package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"brainhub/api/api/middleware"
	"brainhub/api/api/schema"
	"brainhub/domain/constructor"
	"brainhub/usecase/input_port"
)

type PageHandler struct {
	useCase input_port.PageUseCase
}

func NewPageHandler(useCase input_port.PageUseCase) *PageHandler {
	return &PageHandler{useCase: useCase}
}

func (h *PageHandler) List(w http.ResponseWriter, r *http.Request) {
	sourceID, err := constructor.NewSourceID(r.PathValue("sourceID"))
	if err != nil {
		http.Error(w, "invalid source ID", http.StatusBadRequest)
		return
	}

	viewerID := ""
	if auth, ok := middleware.Current(r); ok {
		viewerID = auth.User.ID
	}
	pages, err := h.useCase.List(r.Context(), sourceID, viewerID)
	if errors.Is(err, input_port.ErrBrainNotFound) {
		http.Error(w, "brain not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to list pages", http.StatusBadGateway)
		return
	}

	response := make([]schema.PageResponse, len(pages))
	for i, page := range pages {
		response[i] = schema.PageResponseFromEntity(page)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (h *PageHandler) Get(w http.ResponseWriter, r *http.Request) {
	sourceID, err := constructor.NewSourceID(r.PathValue("sourceID"))
	if err != nil {
		http.Error(w, "invalid source ID", http.StatusBadRequest)
		return
	}
	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "invalid page slug", http.StatusBadRequest)
		return
	}

	viewerID := ""
	if auth, ok := middleware.Current(r); ok {
		viewerID = auth.User.ID
	}
	page, err := h.useCase.Get(r.Context(), sourceID, slug, viewerID)
	if errors.Is(err, input_port.ErrBrainNotFound) || errors.Is(err, input_port.ErrPageNotFound) {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get page", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(schema.PageDetailResponseFromEntity(page))
}
