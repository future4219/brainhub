package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"brainhub/api/api/middleware"
	"brainhub/api/api/schema"
	"brainhub/domain/constructor"
	"brainhub/domain/entity"
	"brainhub/usecase/input_port"
)

type PageHandler struct {
	useCase input_port.PageUseCase
}

const maxPageJSONBodyBytes = 8 << 20

func (h *PageHandler) ListTypes(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := pageSourceID(w, r)
	if !ok {
		return
	}
	auth, _ := middleware.Current(r)
	types, err := h.useCase.ListTypes(r.Context(), sourceID, auth.User.ID)
	if handlePageError(w, err) {
		return
	}
	response := make([]schema.PageTypeResponse, len(types))
	for i, pageType := range types {
		response[i] = schema.PageTypeResponseFromEntity(pageType)
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *PageHandler) Create(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := pageSourceID(w, r)
	if !ok {
		return
	}
	var request schema.CreatePageRequest
	if !decodePageRequest(w, r, &request) {
		return
	}
	auth, _ := middleware.Current(r)
	page, err := h.useCase.Create(r.Context(), sourceID, auth.User.ID, input_port.CreatePageInput{
		Slug: request.Slug, Title: request.Title, Type: request.Type, Tags: request.Tags,
		SupersededBy: request.SupersededBy, CompiledTruth: request.CompiledTruth, TimelineEntry: request.TimelineEntry,
	})
	if handlePageError(w, err) {
		return
	}
	writeJSON(w, http.StatusCreated, schema.PageDetailResponseFromEntity(page))
}

func (h *PageHandler) Update(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := pageSourceID(w, r)
	if !ok {
		return
	}
	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "invalid page slug", http.StatusBadRequest)
		return
	}
	var request schema.UpdatePageRequest
	if !decodePageRequest(w, r, &request) {
		return
	}
	auth, _ := middleware.Current(r)
	page, err := h.useCase.Update(r.Context(), sourceID, slug, auth.User.ID, input_port.UpdatePageInput{
		Title: request.Title, Type: request.Type, Tags: request.Tags,
		SupersededBy: request.SupersededBy, CompiledTruth: request.CompiledTruth, TimelineEntry: request.TimelineEntry,
	})
	if handlePageError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, schema.PageDetailResponseFromEntity(page))
}

func pageSourceID(w http.ResponseWriter, r *http.Request) (entity.SourceID, bool) {
	sourceID, err := constructor.NewSourceID(r.PathValue("sourceID"))
	if err != nil {
		http.Error(w, "invalid source ID", http.StatusBadRequest)
		return "", false
	}
	return sourceID, true
}

func decodePageRequest(w http.ResponseWriter, r *http.Request, destination any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxPageJSONBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return false
	}
	return true
}

func handlePageError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, input_port.ErrInvalidInput):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, input_port.ErrPageAlreadyExists):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, input_port.ErrBrainNotFound), errors.Is(err, input_port.ErrPageNotFound):
		http.Error(w, "page not found", http.StatusNotFound)
	case errors.Is(err, input_port.ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	default:
		http.Error(w, "GBrain page operation failed", http.StatusBadGateway)
	}
	return true
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
