package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"brainhub/api/middleware"
	"brainhub/api/schema"
	"brainhub/domain/constructor"
	"brainhub/domain/entconst"
	"brainhub/usecase/input_port"
)

type BrainHandler struct {
	useCase input_port.BrainUseCase
}

func NewBrainHandler(useCase input_port.BrainUseCase) *BrainHandler {
	return &BrainHandler{useCase: useCase}
}

func (h *BrainHandler) Create(w http.ResponseWriter, r *http.Request) {
	auth, _ := middleware.Current(r)
	var request schema.CreateBrainRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxJSONBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}
	brain, err := h.useCase.Create(r.Context(), auth.User.ID, input_port.CreateBrainInput{
		SourceID:    request.SourceID,
		Name:        request.Name,
		Description: request.Description,
		Visibility:  entconst.Visibility(request.Visibility),
	})
	if errors.Is(err, input_port.ErrInvalidInput) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, input_port.ErrBrainAlreadyExists) {
		if brain.ID == "" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSON(w, http.StatusConflict, schema.BrainErrorResponse{Error: err.Error(), Brain: schema.BrainResponseFromEntity(brain)})
		return
	}
	if errors.Is(err, input_port.ErrProvisioningFailed) {
		writeJSON(w, http.StatusBadGateway, schema.BrainErrorResponse{Error: err.Error(), Brain: schema.BrainResponseFromEntity(brain)})
		return
	}
	if err != nil {
		http.Error(w, "failed to create brain", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, schema.BrainResponseFromEntity(brain))
}

func (h *BrainHandler) Adopt(w http.ResponseWriter, r *http.Request) {
	auth, _ := middleware.Current(r)
	sourceID, err := constructor.NewSourceID(r.PathValue("sourceID"))
	if err != nil {
		http.Error(w, "invalid source ID", http.StatusBadRequest)
		return
	}
	var request schema.AdoptBrainRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxJSONBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}
	brain, err := h.useCase.Adopt(r.Context(), auth.User.ID, sourceID, input_port.AdoptBrainInput{
		Name:        request.Name,
		Description: request.Description,
		Visibility:  entconst.Visibility(request.Visibility),
	})
	switch {
	case errors.Is(err, input_port.ErrInvalidInput):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, input_port.ErrSourceNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, input_port.ErrBrainAlreadyExists):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, input_port.ErrSourceLookupFailed):
		http.Error(w, err.Error(), http.StatusBadGateway)
	case errors.Is(err, input_port.ErrProvisioningFailed):
		writeJSON(w, http.StatusBadGateway, schema.BrainErrorResponse{Error: err.Error(), Brain: schema.BrainResponseFromEntity(brain)})
	case err != nil:
		http.Error(w, "failed to adopt brain", http.StatusInternalServerError)
	default:
		writeJSON(w, http.StatusCreated, schema.BrainResponseFromEntity(brain))
	}
}

func (h *BrainHandler) ReissueWriter(w http.ResponseWriter, r *http.Request) {
	auth, _ := middleware.Current(r)
	sourceID, err := constructor.NewSourceID(r.PathValue("sourceID"))
	if err != nil {
		http.Error(w, "invalid source ID", http.StatusBadRequest)
		return
	}
	err = h.useCase.ReissueWriter(r.Context(), sourceID, auth.User.ID)
	switch {
	case errors.Is(err, input_port.ErrBrainNotFound):
		http.Error(w, "brain not found", http.StatusNotFound)
	case errors.Is(err, input_port.ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	case err != nil:
		http.Error(w, "failed to reissue writer client", http.StatusBadGateway)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *BrainHandler) List(w http.ResponseWriter, r *http.Request) {
	brains, err := h.useCase.List(r.Context(), viewerID(r))
	if err != nil {
		http.Error(w, "failed to list brains", http.StatusInternalServerError)
		return
	}
	response := make([]schema.BrainResponse, len(brains))
	for i, brain := range brains {
		response[i] = schema.BrainResponseFromEntity(brain)
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *BrainHandler) Get(w http.ResponseWriter, r *http.Request) {
	sourceID, err := constructor.NewSourceID(r.PathValue("sourceID"))
	if err != nil {
		http.Error(w, "invalid source ID", http.StatusBadRequest)
		return
	}
	brain, err := h.useCase.Get(r.Context(), sourceID, viewerID(r))
	if errors.Is(err, input_port.ErrBrainNotFound) {
		http.Error(w, "brain not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get brain", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, schema.BrainResponseFromEntity(brain))
}

func viewerID(r *http.Request) string {
	auth, ok := middleware.Current(r)
	if !ok {
		return ""
	}
	return auth.User.ID
}
