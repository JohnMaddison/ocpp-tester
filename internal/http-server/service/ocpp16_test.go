package service

import (
	"errors"
	"testing"

	"github.com/johnmaddison/ocpp-go/ocpp16"
)

func TestBuildOCPP16CallPayload(t *testing.T) {
	got, err := buildOCPP16CallPayload("GetConfiguration", []byte(`{"key":["HeartbeatInterval"]}`))
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}

	payload, ok := got.(ocpp16.GetConfigurationRequest)
	if !ok {
		t.Fatalf("payload type = %T, want ocpp16.GetConfigurationRequest", got)
	}
	if len(payload.Key) != 1 || payload.Key[0] != "HeartbeatInterval" {
		t.Fatalf("key = %#v", payload.Key)
	}
}

func TestBuildOCPP16CallPayloadEmptyBodyForEmptyPayloadAction(t *testing.T) {
	got, err := buildOCPP16CallPayload("ClearCache", nil)
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}
	if _, ok := got.(ocpp16.ClearCacheRequest); !ok {
		t.Fatalf("payload type = %T, want ocpp16.ClearCacheRequest", got)
	}
}

func TestBuildOCPP16CallPayloadNullBodyForEmptyPayloadAction(t *testing.T) {
	got, err := buildOCPP16CallPayload("GetLocalListVersion", []byte(`null`))
	if err != nil {
		t.Fatalf("build payload: %v", err)
	}
	if _, ok := got.(ocpp16.GetLocalListVersionRequest); !ok {
		t.Fatalf("payload type = %T, want ocpp16.GetLocalListVersionRequest", got)
	}
}

func TestBuildOCPP16CallPayloadUnsupportedAction(t *testing.T) {
	_, err := buildOCPP16CallPayload("BootNotification", nil)
	if !errors.Is(err, ErrUnsupportedAction) {
		t.Fatalf("err = %v, want ErrUnsupportedAction", err)
	}
}

func TestBuildOCPP16CallPayloadMalformedJSON(t *testing.T) {
	_, err := buildOCPP16CallPayload("Reset", []byte(`{`))
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestBuildOCPP16CallPayloadValidatesRequiredFields(t *testing.T) {
	_, err := buildOCPP16CallPayload("Reset", []byte(`{}`))
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestBuildOCPP16CallPayloadValidatesNestedRequiredFields(t *testing.T) {
	_, err := buildOCPP16CallPayload("SetChargingProfile", []byte(`{"connectorId":1,"csChargingProfiles":{}}`))
	if !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestBuildOCPP16CallPayloadRejectsChargePointInitiatedAction(t *testing.T) {
	_, err := buildOCPP16CallPayload("Heartbeat", nil)
	if !errors.Is(err, ErrUnsupportedAction) {
		t.Fatalf("err = %v, want ErrUnsupportedAction", err)
	}
}
