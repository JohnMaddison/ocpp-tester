package data

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestOCPPLogStoreMigrateCreatesTableAndIndex(t *testing.T) {
	store := newTestOCPPLogStore(t)

	var tableName string
	if err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'ocpp_logs'`).Scan(&tableName); err != nil {
		t.Fatalf("query table: %v", err)
	}
	if tableName != "ocpp_logs" {
		t.Fatalf("table name = %q, want ocpp_logs", tableName)
	}

	var indexName string
	if err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'index' AND name = 'idx_ocpp_logs_charge_point_created_at'`).Scan(&indexName); err != nil {
		t.Fatalf("query index: %v", err)
	}
	if indexName != "idx_ocpp_logs_charge_point_created_at" {
		t.Fatalf("index name = %q, want idx_ocpp_logs_charge_point_created_at", indexName)
	}
}

func TestOCPPLogStoreRecordsSentAndReceivedMessages(t *testing.T) {
	store := newTestOCPPLogStore(t)
	ctx := context.Background()

	sent := `[2,"sent-1","GetConfiguration",{}]`
	received := `[3,"received-1",{}]`
	recordTestOCPPMessage(t, store, OCPPLogMessage{
		ChargePointID: "CP1",
		Protocol:      "ocpp1.6",
		Direction:     OCPPMessageDirectionSent,
		Message:       sent,
		CreatedAt:     time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
	})
	recordTestOCPPMessage(t, store, OCPPLogMessage{
		ChargePointID: "CP1",
		Protocol:      "ocpp1.6",
		Direction:     OCPPMessageDirectionReceived,
		Message:       received,
		CreatedAt:     time.Date(2026, 7, 16, 10, 1, 0, 0, time.UTC),
	})

	messages, err := store.ListOCPPMessages(ctx, "CP1", 10)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("messages length = %d, want 2", len(messages))
	}
	if messages[0].Direction != OCPPMessageDirectionReceived || messages[0].Message != received {
		t.Fatalf("newest message = %#v, want received payload", messages[0])
	}
	if messages[1].Direction != OCPPMessageDirectionSent || messages[1].Message != sent {
		t.Fatalf("oldest message = %#v, want sent payload", messages[1])
	}
}

func TestOCPPLogStoreListByChargePointNewestFirst(t *testing.T) {
	store := newTestOCPPLogStore(t)
	ctx := context.Background()

	recordTestOCPPMessage(t, store, OCPPLogMessage{
		ChargePointID: "CP1",
		Protocol:      "ocpp1.6",
		Direction:     OCPPMessageDirectionReceived,
		Message:       `[2,"old","Heartbeat",{}]`,
		CreatedAt:     time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
	})
	recordTestOCPPMessage(t, store, OCPPLogMessage{
		ChargePointID: "CP2",
		Protocol:      "ocpp1.6",
		Direction:     OCPPMessageDirectionReceived,
		Message:       `[2,"other","Heartbeat",{}]`,
		CreatedAt:     time.Date(2026, 7, 16, 10, 2, 0, 0, time.UTC),
	})
	recordTestOCPPMessage(t, store, OCPPLogMessage{
		ChargePointID: "CP1",
		Protocol:      "ocpp1.6",
		Direction:     OCPPMessageDirectionReceived,
		Message:       `[2,"new","BootNotification",{}]`,
		CreatedAt:     time.Date(2026, 7, 16, 10, 1, 0, 0, time.UTC),
	})

	messages, err := store.ListOCPPMessages(ctx, "CP1", 10)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("messages length = %d, want 2", len(messages))
	}
	if *messages[0].UniqueID != "new" || *messages[0].Action != "BootNotification" {
		t.Fatalf("first message metadata = %#v, want newest CP1 message", messages[0])
	}
	if *messages[1].UniqueID != "old" || *messages[1].Action != "Heartbeat" {
		t.Fatalf("second message metadata = %#v, want older CP1 message", messages[1])
	}
}

func TestOCPPLogStoreRecordsMalformedJSON(t *testing.T) {
	store := newTestOCPPLogStore(t)
	ctx := context.Background()

	recordTestOCPPMessage(t, store, OCPPLogMessage{
		ChargePointID: "CP1",
		Protocol:      "ocpp1.6",
		Direction:     OCPPMessageDirectionReceived,
		Message:       `not-json`,
		CreatedAt:     time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC),
	})

	messages, err := store.ListOCPPMessages(ctx, "CP1", 10)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages length = %d, want 1", len(messages))
	}
	if messages[0].Message != "not-json" {
		t.Fatalf("message = %q, want raw malformed payload", messages[0].Message)
	}
	if messages[0].MessageTypeID != nil || messages[0].UniqueID != nil || messages[0].Action != nil {
		t.Fatalf("metadata = %#v, want empty metadata", messages[0])
	}
}

func newTestOCPPLogStore(t *testing.T) *OCPPLogStore {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "ocpp-log.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := NewOCPPLogStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store
}

func recordTestOCPPMessage(t *testing.T, store *OCPPLogStore, message OCPPLogMessage) {
	t.Helper()

	if err := store.RecordOCPPMessage(context.Background(), message); err != nil {
		t.Fatalf("record message: %v", err)
	}
}
