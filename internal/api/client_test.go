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

func TestKnockWithTTL(t *testing.T) {
	expectedTTL := 7200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/knock" {
			t.Errorf("Expected to request '/knock', got %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Decode the request body to verify TTL
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		ttlValue, ok := requestBody["ttl"]
		if !ok {
			t.Error("TTL not found in request body")
			return
		}

		// Safely convert ttl to int with type checking
		var actualTTL int
		switch v := ttlValue.(type) {
		case float64:
			actualTTL = int(v)
		case int:
			actualTTL = v
		case int64:
			actualTTL = int(v)
		default:
			t.Errorf("TTL has unexpected type %T, value: %v", ttlValue, ttlValue)
			return
		}

		if actualTTL != expectedTTL {
			t.Errorf("Expected TTL %d, got %d", expectedTTL, actualTTL)
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
	_, err := client.Knock("", expectedTTL)
	if err != nil {
		t.Errorf("Knock with TTL failed: %v", err)
	}
}

func TestStatus(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
		wantErr    bool
	}{
		{"active IPv6", `{"ip_address":"2001:db8::1","whitelisted":true,"expires_at":2000000000,"expires_in_seconds":30}`, 200, false},
		{"inactive", `{"ip_address":"203.0.113.1","whitelisted":false,"expires_at":null,"expires_in_seconds":0}`, 200, false},
		{"missing boolean", `{"ip_address":"203.0.113.1","expires_in_seconds":0}`, 200, true},
		{"missing expiry", `{"ip_address":"203.0.113.1","whitelisted":true,"expires_in_seconds":30}`, 200, true},
		{"invalid IP", `{"ip_address":"invalid","whitelisted":false,"expires_in_seconds":0}`, 200, true},
		{"malformed JSON", `{`, 200, true},
		{"unauthorized", `{}`, 401, true},
		{"rate limited", `{}`, 429, true},
		{"old server", `{}`, 404, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/status" || r.Header.Get("X-Api-Key") != "key" {
					t.Errorf("incorrect status request")
				}
				w.WriteHeader(tc.code)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			_, err := NewClient(server.URL, "key").Status()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
