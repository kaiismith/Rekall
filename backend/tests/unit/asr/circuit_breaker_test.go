package asr_test

import (
	"sync"
	"testing"
	"time"

	"github.com/rekall/backend/internal/infrastructure/asr"
)

func TestCircuitBreaker_StartsClosed(t *testing.T) {
	cb := asr.NewCircuitBreaker(3, 30*time.Second)
	if !cb.Allow() {
		t.Fatalf("expected closed breaker to allow")
	}
	if got := cb.State(); got != "closed" {
		t.Fatalf("state: want closed, got %s", got)
	}
}

func TestCircuitBreaker_TripsOnThreshold(t *testing.T) {
	cb := asr.NewCircuitBreaker(3, 30*time.Second)
	cb.OnFailure()
	cb.OnFailure()
	if cb.State() != "closed" {
		t.Fatalf("breaker tripped early at state %s", cb.State())
	}
	cb.OnFailure()
	if cb.State() != "open" {
		t.Fatalf("breaker should be open after 3 failures, got %s", cb.State())
	}
	if cb.Allow() {
		t.Fatalf("open breaker must deny calls")
	}
}

func TestCircuitBreaker_HalfOpenProbeAfterCooldown(t *testing.T) {
	cb := asr.NewCircuitBreaker(1, 10*time.Millisecond)
	cb.OnFailure()
	if cb.State() != "open" {
		t.Fatalf("breaker should open after first failure")
	}
	if cb.Allow() {
		t.Fatalf("breaker open: must deny")
	}
	time.Sleep(20 * time.Millisecond)
	if !cb.Allow() {
		t.Fatalf("after cooldown the first probe must be admitted")
	}
	if cb.Allow() {
		t.Fatalf("only one half-open probe may be admitted at a time")
	}
	cb.OnSuccess()
	if cb.State() != "closed" {
		t.Fatalf("success should close the breaker, got %s", cb.State())
	}
}

func TestCircuitBreaker_ProbeFailureReopens(t *testing.T) {
	cb := asr.NewCircuitBreaker(1, 10*time.Millisecond)
	cb.OnFailure()
	time.Sleep(20 * time.Millisecond)
	if !cb.Allow() {
		t.Fatalf("expected half-open admission")
	}
	cb.OnFailure()
	if cb.State() != "open" {
		t.Fatalf("probe failure must reopen the breaker, got %s", cb.State())
	}
}

func TestCircuitBreaker_ConcurrentSafe(t *testing.T) {
	cb := asr.NewCircuitBreaker(50, time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.OnSuccess()
			_ = cb.Allow()
			cb.OnFailure()
		}()
	}
	wg.Wait()
	// State is implementation-defined here but the breaker must not panic or
	// be left in an invalid state.
	switch cb.State() {
	case "closed", "open", "half-open":
	default:
		t.Fatalf("unexpected state after storm: %s", cb.State())
	}
}
