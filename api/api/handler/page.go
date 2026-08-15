package handler

import (
	"encoding/json"
	"errors"
	"net/http"

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

	pages, err := h.useCase.List(r.Context(), sourceID)
	if errors.Is(err, input_port.ErrSourceNotFound) {
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
