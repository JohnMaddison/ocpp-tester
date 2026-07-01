package ocppserver

import (
	"context"
	"testing"

	"github.com/johnmaddison/ocpp-go"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/data"
)

func TestRecordOCPPMessageRecordsCallbackInfo(t *testing.T) {
	recorder := &fakeOCPPMessageRecorder{}

	recordOCPPMessage(recorder, ocpp.MessageInfo{
		ConnectionInfo: ocpp.ConnectionInfo{ChargePointID: "CP123"},
		Protocol:       "ocpp1.6",
		Direction:      ocpp.MessageDirectionReceived,
		Message:        []byte(`[2,"uid-1","BootNotification",{}]`),
	})

	if len(recorder.messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(recorder.messages))
	}

	got := recorder.messages[0]
	if got.ChargePointID != "CP123" {
		t.Fatalf("chargePointId = %q, want CP123", got.ChargePointID)
	}
	if got.Protocol != "ocpp1.6" {
		t.Fatalf("protocol = %q, want ocpp1.6", got.Protocol)
	}
	if got.Direction != data.OCPPMessageDirectionReceived {
		t.Fatalf("direction = %q, want received", got.Direction)
	}
	if got.Message != `[2,"uid-1","BootNotification",{}]` {
		t.Fatalf("message = %q, want raw payload", got.Message)
	}
}

type fakeOCPPMessageRecorder struct {
	messages []data.OCPPLogMessage
}

func (f *fakeOCPPMessageRecorder) RecordOCPPMessage(ctx context.Context, message data.OCPPLogMessage) error {
	f.messages = append(f.messages, message)
	return nil
}
