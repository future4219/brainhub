package handler

import (
	"encoding/json"
	"net/http"

	"brainhub/api/api/schema"
	"brainhub/usecase/input_port"
)

type BrainHandler struct {
	useCase input_port.BrainUseCase
}

func NewBrainHandler(useCase input_port.BrainUseCase) *BrainHandler {
	return &BrainHandler{useCase: useCase}
}

func (h *BrainHandler) List(w http.ResponseWriter, r *http.Request) {
	sources, err := h.useCase.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list brains", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(schema.BrainListResponseFromEntities(sources))
}
