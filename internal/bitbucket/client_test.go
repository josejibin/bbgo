package bitbucket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewClient("testuser", "testpass", false)
	client.BaseURL = server.URL
	return client, server
}

func TestDoBasicAuth(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			t.Errorf("expected basic auth testuser:testpass, got %q:%q (ok=%v)", user, pass, ok)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	resp, err := client.Get("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}

func TestDo401ReturnsAuthError(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := client.Get("/test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*AuthError); !ok {
		t.Errorf("expected *AuthError, got %T: %v", err, err)
	}
}

func TestDo404ReturnsNotFoundError(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := client.Get("/test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("expected *NotFoundError, got %T: %v", err, err)
	}
}

func TestDo429RetriesAndFails(t *testing.T) {
	var attempts atomic.Int32
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := client.Get("/test")
	if err == nil {
		t.Fatal("expected error after retries, got nil")
	}
	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Errorf("expected max retries error, got: %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestDo429RetriesWithBodyReplay(t *testing.T) {
	var attempts atomic.Int32
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		// Third attempt: verify body was replayed
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body on attempt %d: %v", n, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body["key"] != "value" {
			t.Errorf("expected body key=value, got %v", body)
		}
		w.WriteHeader(http.StatusOK)
	})

	resp, err := client.Post("/test", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestDo400ReturnsErrorWithBody(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad input"}`))
	})

	_, err := client.Get("/test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "bad input") {
		t.Errorf("expected error body in message, got: %v", err)
	}
}

func TestGetJSON(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	var result map[string]string
	if err := client.GetJSON("/test", &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", result)
	}
}

func TestPostJSON(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"received": body["msg"]})
	})

	var result map[string]string
	if err := client.PostJSON("/test", map[string]string{"msg": "hello"}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["received"] != "hello" {
		t.Errorf("expected received=hello, got %v", result)
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{Msg: "Auth failed"}
	if err.Error() != "Auth failed" {
		t.Errorf("got %q, want %q", err.Error(), "Auth failed")
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{Msg: "Not found"}
	if err.Error() != "Not found" {
		t.Errorf("got %q, want %q", err.Error(), "Not found")
	}
}

func TestDeleteMethod(t *testing.T) {
	client, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	resp, err := client.Delete("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
}
