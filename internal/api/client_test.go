package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected to request '/health', got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	err := client.HealthCheck()
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestKnock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/knock" {
			t.Errorf("Expected to request '/knock', got %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(KnockResponse{
			WhitelistedEntry: "127.0.0.1",
			ExpiresAt:        time.Now().Unix() + 3600,
			ExpiresInSeconds: 3600,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-api-key")
	_, err := client.Knock("", 0)
	if err != nil {
		t.Errorf("Knock failed: %v", err)
	}
}