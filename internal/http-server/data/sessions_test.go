package data

import (
	"reflect"
	"testing"
	"time"
)

func TestSessionStoreListEmpty(t *testing.T) {
	store := NewSessionStore()

	got := store.List()

	if len(got) != 0 {
		t.Fatalf("List() length = %d, want 0", len(got))
	}
}

func TestSessionStoreUpsertAddsAndReplacesByChargePointID(t *testing.T) {
	store := NewSessionStore()
	connectedAt := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	replacementConnectedAt := connectedAt.Add(time.Minute)

	store.Upsert(Session{
		ChargePointID: "CP2",
		Protocol:      "ocpp1.6",
		RemoteAddr:    "127.0.0.1:1000",
		LocalAddr:     "127.0.0.1:8081",
		ConnectedAt:   connectedAt,
	})
	store.Upsert(Session{
		ChargePointID: "CP1",
		Protocol:      "ocpp1.6",
		RemoteAddr:    "127.0.0.1:1001",
		LocalAddr:     "127.0.0.1:8081",
		ConnectedAt:   connectedAt,
	})
	store.Upsert(Session{
		ChargePointID: "CP2",
		Protocol:      "ocpp2.1",
		RemoteAddr:    "127.0.0.1:2000",
		LocalAddr:     "127.0.0.1:8081",
		ConnectedAt:   replacementConnectedAt,
	})

	got := store.List()
	want := []Session{
		{
			ChargePointID: "CP1",
			Protocol:      "ocpp1.6",
			RemoteAddr:    "127.0.0.1:1001",
			LocalAddr:     "127.0.0.1:8081",
			ConnectedAt:   connectedAt,
		},
		{
			ChargePointID: "CP2",
			Protocol:      "ocpp2.1",
			RemoteAddr:    "127.0.0.1:2000",
			LocalAddr:     "127.0.0.1:8081",
			ConnectedAt:   replacementConnectedAt,
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func TestSessionStoreDeleteRemovesOnlyMatchingSession(t *testing.T) {
	store := NewSessionStore()
	connectedAt := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)

	store.Upsert(Session{ChargePointID: "CP1", Protocol: "ocpp1.6", ConnectedAt: connectedAt})
	store.Upsert(Session{ChargePointID: "CP2", Protocol: "ocpp1.6", ConnectedAt: connectedAt})

	store.Delete("CP1")

	got := store.List()
	want := []Session{{ChargePointID: "CP2", Protocol: "ocpp1.6", ConnectedAt: connectedAt}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() after Delete() = %#v, want %#v", got, want)
	}
}

func TestSessionStoreGet(t *testing.T) {
	store := NewSessionStore()
	connectedAt := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	want := Session{
		ChargePointID: "CP1",
		Protocol:      "ocpp1.6",
		RemoteAddr:    "127.0.0.1:1001",
		LocalAddr:     "127.0.0.1:8081",
		ConnectedAt:   connectedAt,
	}

	store.Upsert(want)

	got, ok := store.Get("CP1")
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if got != want {
		t.Fatalf("Get() = %#v, want %#v", got, want)
	}

	if _, ok := store.Get("missing"); ok {
		t.Fatal("Get() missing ok = true, want false")
	}
}
