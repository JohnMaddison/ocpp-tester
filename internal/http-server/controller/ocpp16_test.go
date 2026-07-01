package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/johnmaddison/ocpp-go"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/service"
)

func TestOCPP16ControllerSendCall(t *testing.T) {
	fake := &fakeOCPP16Service{
		result: &service.SendCallResult{
			ChargePointID: "CP123",
			Action:        "GetConfiguration",
			Payload: map[string]any{
				"configurationKey": []any{},
			},
		},
	}
	mux := http.NewServeMux()
	NewOCPP16Controller(fake).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/ocpp16/CP123/calls/GetConfiguration", strings.NewReader(`{"key":["HeartbeatInterval"]}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if fake.chargePointID != "CP123" {
		t.Fatalf("chargePointID = %q, want CP123", fake.chargePointID)
	}
	if fake.action != "GetConfiguration" {
		t.Fatalf("action = %q, want GetConfiguration", fake.action)
	}
	if string(fake.payloadJSON) != `{"key":["HeartbeatInterval"]}` {
		t.Fatalf("payloadJSON = %q", string(fake.payloadJSON))
	}

	var got service.SendCallResult
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ChargePointID != "CP123" || got.Action != "GetConfiguration" {
		t.Fatalf("response = %#v", got)
	}
}

func TestOCPP16ControllerLegacyGetConfigurationRouteRemoved(t *testing.T) {
	mux := http.NewServeMux()
	NewOCPP16Controller(&fakeOCPP16Service{}).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/ocpp16/CP123/getconfiguration", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestOCPP16ControllerSendCallSessionNotFound(t *testing.T) {
	mux := http.NewServeMux()
	NewOCPP16Controller(&fakeOCPP16Service{err: service.ErrSessionNotFound}).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/ocpp16/missing/calls/GetConfiguration", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestOCPP16ControllerSendCallUnknownAction(t *testing.T) {
	mux := http.NewServeMux()
	NewOCPP16Controller(&fakeOCPP16Service{err: service.ErrUnsupportedAction}).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/ocpp16/CP123/calls/BootNotification", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestOCPP16ControllerSendCallInvalidPayload(t *testing.T) {
	mux := http.NewServeMux()
	NewOCPP16Controller(&fakeOCPP16Service{err: service.ErrInvalidPayload}).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/ocpp16/CP123/calls/Reset", strings.NewReader(`{`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestOCPP16ControllerSendCallCallError(t *testing.T) {
	mux := http.NewServeMux()
	NewOCPP16Controller(&fakeOCPP16Service{
		result: &service.SendCallResult{
			ChargePointID: "CP123",
			Action:        "Reset",
			CallError:     &ocpp.CallError{ErrorCode: "NotSupported", ErrorDescription: "unsupported"},
		},
	}).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/ocpp16/CP123/calls/Reset", strings.NewReader(`{"type":"Soft"}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestOCPP16ControllerSendCallTimeout(t *testing.T) {
	mux := http.NewServeMux()
	NewOCPP16Controller(&fakeOCPP16Service{err: context.DeadlineExceeded}).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/ocpp16/CP123/calls/Reset", strings.NewReader(`{"type":"Soft"}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusGatewayTimeout)
	}
}

func TestOCPP16ControllerSendCallSendError(t *testing.T) {
	mux := http.NewServeMux()
	NewOCPP16Controller(&fakeOCPP16Service{err: errors.New("send failed")}).RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/ocpp16/CP123/calls/GetConfiguration", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
}

type fakeOCPP16Service struct {
	chargePointID string
	action        string
	payloadJSON   []byte
	result        *service.SendCallResult
	err           error
}

func (s *fakeOCPP16Service) SendCall(ctx context.Context, chargePointID string, action string, payloadJSON []byte) (*service.SendCallResult, error) {
	s.chargePointID = chargePointID
	s.action = action
	s.payloadJSON = payloadJSON
	return s.result, s.err
}
