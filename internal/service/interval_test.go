package service

import (
	"testing"
	"time"
)

func TestNormalizeCheckInterval_DefaultsOnInvalid(t *testing.T) {
	if NormalizeCheckInterval(-5*time.Minute) != DefaultCheckInterval() {
		t.Fatalf("expected default interval for invalid input")
	}
}

func TestNormalizeCheckInterval_PassesThroughValid(t *testing.T) {
	configured := 7 * time.Minute
	if NormalizeCheckInterval(configured) != configured {
		t.Fatalf("expected %v to pass through", configured)
	}
}

func TestKnockCadenceFromTTL_DefaultsWhenUnset(t *testing.T) {
	if KnockCadenceFromTTL(0) != DefaultCheckInterval() {
		t.Fatalf("expected default cadence when ttl unset")
	}
}

func TestKnockCadenceFromTTL_UsesTenPercentBuffer(t *testing.T) {
	result := KnockCadenceFromTTL(120)
	expected := 108 * time.Second
	if result != expected {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestKnockCadenceFromTTL_DoesNotExceedTTL(t *testing.T) {
	result := KnockCadenceFromTTL(3)
	if result > 3*time.Second {
		t.Fatalf("expected cadence <= ttl, got %v", result)
	}
}

func TestKnockCadenceFromTTL_RespectsOneSecondFloor(t *testing.T) {
	if KnockCadenceFromTTL(1) != 1*time.Second {
		t.Fatalf("expected 1s cadence when ttl=1s")
	}
}

func TestRestartDelay_DefaultWhenNoTTL(t *testing.T) {
	if RestartDelay(0) != 30*time.Second {
		t.Fatalf("expected 30s restart delay")
	}
}

func TestRestartDelay_QuarterTTLWithinBounds(t *testing.T) {
	if RestartDelay(120) != 30*time.Second {
		t.Fatalf("expected 30s restart delay for ttl=120s")
	}

	if RestartDelay(60) != 15*time.Second {
		t.Fatalf("expected 15s restart delay for ttl=60s")
	}
}

func TestRestartDelay_MinimumApplied(t *testing.T) {
	if RestartDelay(8) != 5*time.Second {
		t.Fatalf("expected 5s restart delay for ttl=8s")
	}
}
