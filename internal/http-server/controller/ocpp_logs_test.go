package controller

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/johnmaddison/ocpp-tester/internal/http-server/data"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/service"
)

func TestOCPPLogControllerAPILog(t *testing.T) {
	mux := newOCPPLogTestMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/chargepoints/CP123/ocpp-log", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var got ocppLogResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ChargePointID != "CP123" {
		t.Fatalf("chargePointId = %q, want CP123", got.ChargePointID)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(got.Messages))
	}
	if got.Messages[0].ChargePointID != "CP123" || got.Messages[0].Message != `[2,"uid-1","BootNotification",{}]` {
		t.Fatalf("message = %#v, want CP123 boot notification", got.Messages[0])
	}
}

func TestOCPPLogControllerView(t *testing.T) {
	mux := newOCPPLogTestMux(t)

	req := httptest.NewRequest(http.MethodGet, "/chargepoints/CP123/ocpp-log", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, value := range []string{"CP123", "received", "BootNotification", "uid-1", `[2,&#34;uid-1&#34;,&#34;BootNotification&#34;,{}]`} {
		if !strings.Contains(body, value) {
			t.Fatalf("body does not contain %q:\n%s", value, body)
		}
	}
}

func TestOCPPLogControllerInvalidLimit(t *testing.T) {
	mux := newOCPPLogTestMux(t)

	req := httptest.NewRequest(http.MethodGet, "/api/chargepoints/CP123/ocpp-log?limit=invalid", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func newOCPPLogTestMux(t *testing.T) *http.ServeMux {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "ocpp-log.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := data.NewOCPPLogStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := store.RecordOCPPMessage(context.Background(), data.OCPPLogMessage{
		ChargePointID: "CP123",
		Protocol:      "ocpp1.6",
		Direction:     data.OCPPMessageDirectionReceived,
		Message:       `[2,"uid-1","BootNotification",{}]`,
		CreatedAt:     time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("record CP123 message: %v", err)
	}
	if err := store.RecordOCPPMessage(context.Background(), data.OCPPLogMessage{
		ChargePointID: "CP999",
		Protocol:      "ocpp1.6",
		Direction:     data.OCPPMessageDirectionReceived,
		Message:       `[2,"uid-2","Heartbeat",{}]`,
		CreatedAt:     time.Date(2026, 7, 16, 10, 1, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("record CP999 message: %v", err)
	}

	mux := http.NewServeMux()
	NewOCPPLogController(service.NewOCPPLogService(store)).RegisterRoutes(mux)
	return mux
}
