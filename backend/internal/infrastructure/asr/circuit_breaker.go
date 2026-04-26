// Package asr contains the gRPC client implementation of the ASRClient port
// plus the circuit breaker that guards every call into the standalone C++
// service. The breaker trips after N consecutive transport-level failures
// (UNAVAILABLE / DEADLINE_EXCEEDED) and stays open for a cooldown period
// before allowing a single half-open probe.
package asr

import (
	"sync/atomic"
	"time"
)

// breakerState is encoded as int32 for atomic CAS.
const (
	stateClosed   int32 = 0
	stateOpen     int32 = 1
	stateHalfOpen int32 = 2
)

// CircuitBreaker is safe for concurrent use. The contract:
//
//	if !cb.Allow() { return ErrASRUnavailable }
//	resp, err := callASR(...)
//	if err != nil { cb.OnFailure() } else { cb.OnSuccess() }
//
// In half-open state Allow returns true exactly once; subsequent callers see
// false until the probe reports back via OnSuccess / OnFailure.
type CircuitBreaker struct {
	state         atomic.Int32
	failures      atomic.Int32
	lastTrippedNs atomic.Int64
	probeInFlight atomic.Bool

	failureThreshold int32
	cooldown         time.Duration
}

// NewCircuitBreaker returns a closed breaker with the given threshold and
// cooldown. A failureThreshold ≤ 0 is treated as 1.
func NewCircuitBreaker(failureThreshold int, cooldown time.Duration) *CircuitBreaker {
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	return &CircuitBreaker{
		failureThreshold: int32(failureThreshold),
		cooldown:         cooldown,
	}
}

// Allow reports whether the next call may proceed.
func (c *CircuitBreaker) Allow() bool {
	switch c.state.Load() {
	case stateClosed:
		return true
	case stateOpen:
		if time.Since(time.Unix(0, c.lastTrippedNs.Load())) < c.cooldown {
			return false
		}
		// Cooldown elapsed. Promote to half-open and admit exactly one probe.
		if c.state.CompareAndSwap(stateOpen, stateHalfOpen) {
			c.probeInFlight.Store(true)
			return true
		}
		// Some other goroutine flipped state first.
		return false
	case stateHalfOpen:
		// In half-open we admit only the single in-flight probe; anyone else
		// must wait for OnSuccess / OnFailure to resolve the state.
		return false
	}
	return false
}

// OnSuccess records a successful call and resets the breaker.
func (c *CircuitBreaker) OnSuccess() {
	c.failures.Store(0)
	c.state.Store(stateClosed)
	c.probeInFlight.Store(false)
}

// OnFailure records a transport-level failure. Trips the breaker on the
// configured threshold.
func (c *CircuitBreaker) OnFailure() {
	if c.state.Load() == stateHalfOpen {
		// Probe failed → re-open immediately.
		c.lastTrippedNs.Store(time.Now().UnixNano())
		c.state.Store(stateOpen)
		c.probeInFlight.Store(false)
		return
	}
	n := c.failures.Add(1)
	if n >= c.failureThreshold && c.state.CompareAndSwap(stateClosed, stateOpen) {
		c.lastTrippedNs.Store(time.Now().UnixNano())
	}
}

// State returns one of "closed" | "open" | "half-open" for logging / metrics.
func (c *CircuitBreaker) State() string {
	switch c.state.Load() {
	case stateClosed:
		return "closed"
	case stateOpen:
		return "open"
	case stateHalfOpen:
		return "half-open"
	}
	return "unknown"
}
