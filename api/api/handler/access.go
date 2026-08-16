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
	"brainhub/usecase/output_port"
)

type AccessHandler struct {
	useCase input_port.AccessUseCase
}

func NewAccessHandler(useCase input_port.AccessUseCase) *AccessHandler {
	return &AccessHandler{useCase: useCase}
}

func (h *AccessHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := sourceIDFromRequest(w, r)
	if !ok {
		return
	}
	var request schema.CreateInvitationRequest
	if !decodeAccessJSON(w, r, &request) {
		return
	}
	current, _ := middleware.Current(r)
	invitation, rawToken, err := h.useCase.CreateInvitation(r.Context(), sourceID, current.User.ID, input_port.CreateInvitationInput{
		Email: request.Email, Role: entity.Role(request.Role), ExpiresAt: request.ExpiresAt,
	})
	if accessError(w, err) {
		return
	}
	writeJSON(w, http.StatusCreated, schema.CreatedInvitationResponse{
		InvitationResponse: schema.InvitationResponseFromEntity(invitation), Token: rawToken,
	})
}

func (h *AccessHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := sourceIDFromRequest(w, r)
	if !ok {
		return
	}
	current, _ := middleware.Current(r)
	invitations, err := h.useCase.ListInvitations(r.Context(), sourceID, current.User.ID)
	if accessError(w, err) {
		return
	}
	response := make([]schema.InvitationResponse, len(invitations))
	for i, invitation := range invitations {
		response[i] = schema.InvitationResponseFromEntity(invitation)
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AccessHandler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.Current(r)
	err := h.useCase.RevokeInvitation(r.Context(), r.PathValue("id"), current.User.ID)
	if accessError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AccessHandler) PreviewInvitation(w http.ResponseWriter, r *http.Request) {
	preview, err := h.useCase.PreviewInvitation(r.Context(), r.PathValue("token"))
	if accessError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, schema.InvitationPreviewResponseFromInput(preview))
}

func (h *AccessHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.Current(r)
	accepted, err := h.useCase.AcceptInvitation(r.Context(), r.PathValue("token"), current.User)
	if accessError(w, err) {
		return
	}
	writeJSON(w, http.StatusCreated, schema.AcceptedInvitationResponse{
		SourceID: accepted.SourceID.String(), BrainName: accepted.BrainName, Role: string(accepted.Membership.Role),
	})
}

func (h *AccessHandler) IssueClient(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := sourceIDFromRequest(w, r)
	if !ok {
		return
	}
	var request schema.CreateClientRequest
	if !decodeAccessJSON(w, r, &request) {
		return
	}
	current, _ := middleware.Current(r)
	client, err := h.useCase.IssueClient(r.Context(), sourceID, current.User.ID, request.Label)
	if errors.Is(err, input_port.ErrGBrainAdmin) {
		writeJSON(w, http.StatusBadGateway, schema.IssuedClientErrorResponse{Error: err.Error(), Client: schema.IssuedClientResponseFromEntity(client)})
		return
	}
	if accessError(w, err) {
		return
	}
	writeJSON(w, http.StatusCreated, schema.IssuedClientResponseFromEntity(client))
}

func (h *AccessHandler) ListClients(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := sourceIDFromRequest(w, r)
	if !ok {
		return
	}
	current, _ := middleware.Current(r)
	clients, err := h.useCase.ListClients(r.Context(), sourceID, current.User.ID)
	if accessError(w, err) {
		return
	}
	response := make([]schema.IssuedClientResponse, len(clients))
	for i, client := range clients {
		response[i] = schema.IssuedClientResponseFromEntity(client)
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *AccessHandler) RevokeClient(w http.ResponseWriter, r *http.Request) {
	current, _ := middleware.Current(r)
	client, err := h.useCase.RevokeClient(r.Context(), r.PathValue("id"), current.User.ID)
	if errors.Is(err, input_port.ErrGBrainAdmin) {
		writeJSON(w, http.StatusBadGateway, schema.IssuedClientErrorResponse{Error: err.Error(), Client: schema.IssuedClientResponseFromEntity(client)})
		return
	}
	if accessError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AccessHandler) RevokeMembership(w http.ResponseWriter, r *http.Request) {
	sourceID, ok := sourceIDFromRequest(w, r)
	if !ok {
		return
	}
	current, _ := middleware.Current(r)
	err := h.useCase.RevokeMembership(r.Context(), sourceID, current.User.ID, r.PathValue("userID"))
	if accessError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeAccessJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxJSONBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return false
	}
	return true
}

func sourceIDFromRequest(w http.ResponseWriter, r *http.Request) (entity.SourceID, bool) {
	sourceID, err := constructor.NewSourceID(r.PathValue("sourceID"))
	if err != nil {
		http.Error(w, "invalid source ID", http.StatusBadRequest)
		return "", false
	}
	return sourceID, true
}

func accessError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, input_port.ErrInvalidInput), errors.Is(err, input_port.ErrUnsupportedClient):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, input_port.ErrForbidden), errors.Is(err, input_port.ErrInvitationEmail):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, input_port.ErrInvitationNotFound), errors.Is(err, input_port.ErrMembershipNotFound), errors.Is(err, input_port.ErrIssuedClientNotFound), errors.Is(err, input_port.ErrBrainNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, input_port.ErrInvitationAccepted), errors.Is(err, input_port.ErrMembershipExists), errors.Is(err, input_port.ErrCannotRevokeSelf), errors.Is(err, output_port.ErrConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, input_port.ErrGBrainAdmin):
		http.Error(w, err.Error(), http.StatusBadGateway)
	default:
		http.Error(w, "access operation failed", http.StatusInternalServerError)
	}
	return true
}
