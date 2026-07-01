package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/johnmaddison/ocpp-tester/internal/http-server/data"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/service"
)

func TestSessionsControllerAPISessions(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}

	var got sessionsResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(got.Sessions) != 1 {
		t.Fatalf("sessions length = %d, want 1", len(got.Sessions))
	}
	assertTestSession(t, got.Sessions[0])
}

func TestSessionsControllerSessionsView(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/sessions", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", contentType)
	}

	body := rr.Body.String()
	for _, value := range []string{"CP123", "/sessions/CP123", "ocpp1.6", "127.0.0.1:12345", "127.0.0.1:8081", "2026-07-02T10:00:00Z"} {
		if !strings.Contains(body, value) {
			t.Fatalf("body does not contain %q:\n%s", value, body)
		}
	}
}

func TestSessionsControllerRootView(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "Active OCPP Sessions") {
		t.Fatalf("body does not contain sessions title:\n%s", rr.Body.String())
	}
}

func TestSessionsControllerSessionView(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/sessions/CP123", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if contentType := rr.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", contentType)
	}

	body := rr.Body.String()
	for _, value := range []string{
		"CP123",
		"Send OCPP 1.6 Call",
		"GetConfiguration",
		"Reset",
		`/api/chargepoints/`,
		`/api/ocpp16/`,
		"2026-07-02T10:00:00Z",
		"<th>Time</th>",
		"<th>Dir</th>",
		"<th>UID</th>",
		"<th>Message</th>",
	} {
		if !strings.Contains(body, value) {
			t.Fatalf("body does not contain %q:\n%s", value, body)
		}
	}
}

func TestSessionsControllerSessionViewMissing(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/sessions/missing", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestSessionsControllerMethodNotAllowed(t *testing.T) {
	for _, path := range []string{"/api/sessions", "/sessions", "/sessions/CP123"} {
		t.Run(path, func(t *testing.T) {
			mux := newTestMux()

			req := httptest.NewRequest(http.MethodPost, path, nil)
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
			}
			if allow := rr.Header().Get("Allow"); allow != "GET, HEAD" {
				t.Fatalf("Allow = %q, want GET, HEAD", allow)
			}
		})
	}
}

func newTestMux() *http.ServeMux {
	store := data.NewSessionStore()
	store.Upsert(testSession())

	mux := http.NewServeMux()
	NewSessionsController(service.NewSessionsService(store)).RegisterRoutes(mux)
	return mux
}

func testSession() data.Session {
	return data.Session{
		ChargePointID: "CP123",
		Protocol:      "ocpp1.6",
		RemoteAddr:    "127.0.0.1:12345",
		LocalAddr:     "127.0.0.1:8081",
		ConnectedAt:   time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC),
	}
}

func assertTestSession(t *testing.T, session data.Session) {
	t.Helper()

	want := testSession()
	if session != want {
		t.Fatalf("session = %#v, want %#v", session, want)
	}
}
