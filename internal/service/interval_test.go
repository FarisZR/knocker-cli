package service

import (
	"testing"
	"time"
)

func TestEffectiveInterval_DefaultsToConfiguredWhenNoTTL(t *testing.T) {
	configured := 2 * time.Minute
	result := EffectiveInterval(configured, 0)

	if result != configured {
		t.Fatalf("expected %v, got %v", configured, result)
	}
}

func TestEffectiveInterval_UsesTenPercentTTLBuffer(t *testing.T) {
	configured := 5 * time.Minute
	result := EffectiveInterval(configured, 120)

	expected := 108 * time.Second
	if result != expected {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestEffectiveInterval_RespectsConfiguredWhenShorterThanTTLBuffer(t *testing.T) {
	configured := 20 * time.Second
	result := EffectiveInterval(configured, 600)

	if result != configured {
		t.Fatalf("expected %v, got %v", configured, result)
	}
}

func TestEffectiveInterval_ProducesExpectedBufferAndMinimums(t *testing.T) {
	configured := 10 * time.Minute
	if EffectiveInterval(configured, 1) != 1*time.Second {
		t.Fatalf("expected 1s interval when ttl=1s")
	}

	expected := (6 * time.Second * 9) / 10
	if EffectiveInterval(configured, 6) != expected {
		t.Fatalf("expected %v interval when ttl=6s", expected)
	}
}

func TestEffectiveInterval_ResultDoesNotExceedTTL(t *testing.T) {
	configured := 10 * time.Minute
	result := EffectiveInterval(configured, 3)

	if result > 3*time.Second {
		t.Fatalf("expected interval <= 3s, got %v", result)
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
