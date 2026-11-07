package service

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FarisZR/knocker-cli/internal/api"
)

func TestNetworkStatusDecisions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		statusCode int
		body       string
		wantKnocks int
		wantErr    bool
	}{
		{"already covered", 200, `{"ip_address":"203.0.113.4","whitelisted":true,"expires_at":2000000000,"expires_in_seconds":120}`, 0, false},
		{"inactive", 200, `{"ip_address":"203.0.113.4","whitelisted":false,"expires_at":null,"expires_in_seconds":0}`, 1, false},
		{"unauthorized", 401, `{}`, 0, true},
		{"unavailable", 503, `{}`, 0, true},
		{"malformed", 200, `{}`, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			knocks := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/status" {
					w.WriteHeader(tc.statusCode)
					io.WriteString(w, tc.body)
					return
				}
				if r.URL.Path != "/knock" {
					t.Errorf("unexpected endpoint %s", r.URL.Path)
				}
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if _, ok := body["ip_address"]; ok {
					t.Error("must let server observe source IP")
				}
				knocks++
				json.NewEncoder(w).Encode(api.KnockResponse{WhitelistedEntry: "203.0.113.4", ExpiresAt: 2000000000, ExpiresInSeconds: 60})
			}))
			defer server.Close()
			svc := NewService(api.NewClient(server.URL, "key"), time.Hour, true, time.Minute, 600, "ttl", "test", log.New(io.Discard, "", 0))
			svc.updateNextKnock(time.Now().Add(time.Hour))
			original := svc.nextKnockUnix
			knocked, err := svc.checkStatus()
			if (err != nil) != tc.wantErr || knocks != tc.wantKnocks || knocked != (tc.wantKnocks > 0) {
				t.Fatalf("knocked=%v, knocks=%d, err=%v", knocked, knocks, err)
			}
			if svc.nextKnockUnix != original {
				t.Fatal("status check changed renewal deadline")
			}
			if knocked && svc.Cadence != KnockCadenceFromTTL(60) {
				t.Fatal("knock did not update TTL cadence")
			}
		})
	}
}

func TestServiceNetworkEvents(t *testing.T) {
	var active atomic.Bool
	active.Store(true)
	var fail atomic.Bool
	var knocks atomic.Int32
	statuses := make(chan struct{}, 20)
	knocked := make(chan struct{}, 20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			defer func() { statuses <- struct{}{} }()
			if fail.Load() {
				w.WriteHeader(503)
				return
			}
			expiry := time.Now().Add(time.Hour).Unix()
			json.NewEncoder(w).Encode(api.StatusResponse{IPAddress: "203.0.113.5", Whitelisted: active.Load(), ExpiresAt: &expiry, ExpiresInSeconds: 3600})
			return
		}
		if r.URL.Path != "/knock" {
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
		knocks.Add(1)
		active.Store(true)
		json.NewEncoder(w).Encode(api.KnockResponse{WhitelistedEntry: "203.0.113.5", ExpiresAt: time.Now().Add(time.Hour).Unix(), ExpiresInSeconds: 3600})
		knocked <- struct{}{}
	}))
	defer server.Close()
	svc := NewService(api.NewClient(server.URL, "key"), time.Hour, true, time.Hour, 0, "ttl", "test", log.New(io.Discard, "", 0))
	changes := make(chan struct{}, 20)
	svc.networkChanges = changes
	svc.networkDebounce = 10 * time.Millisecond
	svc.retryDelay = 20 * time.Millisecond
	quit := make(chan struct{})
	done := make(chan struct{})
	go func() { defer close(done); svc.Run(quit) }()
	defer func() { close(quit); <-done }()
	await := func(ch <-chan struct{}) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out")
		}
	}
	await(statuses) // active startup does not knock
	if knocks.Load() != 0 {
		t.Fatal("active startup knocked")
	}
	for i := 0; i < 10; i++ {
		changes <- struct{}{}
	}
	await(statuses)
	if knocks.Load() != 0 {
		t.Fatal("active network change knocked")
	}
	fail.Store(true)
	changes <- struct{}{}
	await(statuses)
	if knocks.Load() != 0 {
		t.Fatal("unknown status knocked")
	}
	active.Store(false)
	fail.Store(false)
	await(knocked) // retry after connectivity recovers
	if knocks.Load() != 1 {
		t.Fatal("expected one knock")
	}
}

func TestServiceKnocksImmediatelyAndRenewsWithoutNetworkWatch(t *testing.T) {
	knocked := make(chan struct{}, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/knock" {
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
		// Unknown TTL preserves the short test cadence.
		json.NewEncoder(w).Encode(api.KnockResponse{WhitelistedEntry: "203.0.113.6"})
		knocked <- struct{}{}
	}))
	defer server.Close()
	svc := NewService(api.NewClient(server.URL, "key"), 10*time.Millisecond, false, time.Hour, 0, "ttl", "test", log.New(io.Discard, "", 0))
	quit := make(chan struct{})
	done := make(chan struct{})
	go func() { defer close(done); svc.Run(quit) }()
	defer func() { close(quit); <-done }()
	for i := 0; i < 2; i++ {
		select {
		case <-knocked:
		case <-time.After(time.Second):
			t.Fatal("missing knock")
		}
	}
}

func TestServiceAdjustsCadenceFromServerTTL(t *testing.T) {
	for _, watch := range []bool{false, true} {
		svc := NewService(nil, time.Hour, watch, time.Minute, 600, "ttl", "test", log.New(io.Discard, "", 0))
		svc.handleWhitelistResponse(&api.KnockResponse{ExpiresInSeconds: 120}, TriggerSourceSchedule)
		if svc.Cadence != KnockCadenceFromTTL(120) || svc.cadenceSrc != "ttl_response" {
			t.Fatal("TTL response must drive renewal in both modes")
		}
	}
}

func TestStatusPollingPreservesRenewalDeadline(t *testing.T) {
	var active atomic.Bool
	knocks := make(chan struct{}, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			expiry := time.Now().Add(time.Hour).Unix()
			json.NewEncoder(w).Encode(api.StatusResponse{IPAddress: "203.0.113.7", Whitelisted: active.Load(), ExpiresAt: &expiry, ExpiresInSeconds: 3600})
			return
		}
		active.Store(true)
		json.NewEncoder(w).Encode(api.KnockResponse{WhitelistedEntry: "203.0.113.7"})
		knocks <- struct{}{}
	}))
	defer server.Close()
	svc := NewService(api.NewClient(server.URL, "key"), 120*time.Millisecond, true, 20*time.Millisecond, 0, "ttl", "test", log.New(io.Discard, "", 0))
	svc.networkChanges = make(chan struct{})
	quit := make(chan struct{})
	done := make(chan struct{})
	go func() { defer close(done); svc.Run(quit) }()
	defer func() { close(quit); <-done }()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for i := 0; i < 2; i++ {
		select {
		case <-knocks:
		case <-deadline.C:
			t.Fatal("active status checks postponed renewal")
		}
	}
}
