package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/johnmaddison/ocpp-tester/internal/http-server/service"
)

type OCPP16Controller struct {
	service ocpp16Service
}

type ocpp16Service interface {
	SendCall(ctx context.Context, chargePointID string, action string, payloadJSON []byte) (*service.SendCallResult, error)
}

func NewOCPP16Controller(service ocpp16Service) *OCPP16Controller {
	return &OCPP16Controller{service: service}
}

func (c *OCPP16Controller) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/ocpp16/{chargePointID}/calls/{action}", c.handleSendCall)
}

func (c *OCPP16Controller) handleSendCall(w http.ResponseWriter, r *http.Request) {
	chargePointID := r.PathValue("chargePointID")
	if chargePointID == "" {
		http.Error(w, "charge point id is required", http.StatusBadRequest)
		return
	}
	action := r.PathValue("action")
	if action == "" {
		http.Error(w, "action is required", http.StatusBadRequest)
		return
	}

	payloadJSON, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	result, err := c.service.SendCall(r.Context(), chargePointID, action, payloadJSON)
	if errors.Is(err, service.ErrUnsupportedAction) {
		http.Error(w, "unsupported ocpp 1.6 action", http.StatusBadRequest)
		return
	}
	if errors.Is(err, service.ErrInvalidPayload) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, service.ErrSessionNotFound) {
		http.Error(w, "ocpp session not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		http.Error(w, "ocpp call timed out", http.StatusGatewayTimeout)
		return
	}
	if err != nil {
		http.Error(w, "failed to send ocpp call", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, "failed to encode ocpp call response", http.StatusInternalServerError)
	}
}
