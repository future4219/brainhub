package handler

import (
	"errors"
	"log"
	"net/http"

	"brainhub/api/schema"
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

	pages, err := h.useCase.List(r.Context(), sourceID, viewerID(r))
	if errors.Is(err, input_port.ErrBrainNotFound) {
		http.Error(w, "brain not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("list pages source=%q: %v", sourceID, err)
		http.Error(w, "failed to list pages", http.StatusBadGateway)
		return
	}

	response := make([]schema.PageResponse, len(pages))
	for i, page := range pages {
		response[i] = schema.PageResponseFromEntity(page)
	}

	writeJSON(w, http.StatusOK, response)
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

	page, err := h.useCase.Get(r.Context(), sourceID, slug, viewerID(r))
	if errors.Is(err, input_port.ErrBrainNotFound) || errors.Is(err, input_port.ErrPageNotFound) {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get page source=%q slug=%q: %v", sourceID, slug, err)
		http.Error(w, "failed to get page", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, schema.PageDetailResponseFromEntity(page))
}
