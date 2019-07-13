package rateopener

import (
	"sync"
	"time"

	"github.com/cep21/aimdopener"

	"github.com/cep21/circuit/v3"
)

// Closer is a rateopener closer that allows requests according to a rate limiter.
type Closer struct {
	// Rater is the rate limiter of this closer
	Rater aimdopener.RateLimiter
	// CloseOnHappyDuration is how long we should see zero failing requests before we close the rateopener.
	CloseOnHappyDuration time.Duration
	lastFailedReserve    time.Time
	mu                   sync.Mutex
}

// OpenerConfig configures defaults for Closer.
type CloserConfig struct {
	// RateLimiter constructs new rate limiters for circuits.  We default to a reasonable AIMD configuration.
	// That configuration happens to be AIMDConstructor(.1, .5, float64(time.Microsecond/time.Second), 10) right now.
	RateLimiter func() aimdopener.RateLimiter
	// CloseOnHappyDuration gives a duration that passing requests cause the rateopener to close.
	// We default to a reasonable short value.  It happens to be 10 seconds right now.
	CloseOnHappyDuration time.Duration
}

func (o *CloserConfig) merge(other CloserConfig) {
	if o.CloseOnHappyDuration == 0 {
		o.CloseOnHappyDuration = other.CloseOnHappyDuration
	}
	if o.RateLimiter == nil {
		o.RateLimiter = other.RateLimiter
	}
}

var defaultConfig = CloserConfig{
	RateLimiter:          aimdopener.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
	CloseOnHappyDuration: time.Second * 10,
}

// CloserFactory is injectable into a rateopener's configuration to create a factory of rate limit closers for a rateopener.
func CloserFactory(conf CloserConfig) func() circuit.OpenToClosed {
	return func() circuit.OpenToClosed {
		c := conf
		c.merge(defaultConfig)
		return &Closer{
			Rater:                c.RateLimiter(),
			CloseOnHappyDuration: c.CloseOnHappyDuration,
			lastFailedReserve:    time.Now(),
		}
	}
}

// Success sends the rater a success message.
func (c *Closer) Success(now time.Time, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Rater.OnSuccess(now)
}

// ErrFailure sends the rater a failure message.
func (c *Closer) ErrFailure(now time.Time, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Rater.OnFailure(now)
	c.lastFailedReserve = now
}

// ErrTimeout sends the rater a failure message.
func (c *Closer) ErrTimeout(now time.Time, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Rater.OnFailure(now)
	c.lastFailedReserve = now
}

// ErrBadRequest is ignored and exists only to satisfy the closer interface.
func (c *Closer) ErrBadRequest(now time.Time, duration time.Duration) {
}

// ErrInterrupt is ignored and exists only to satisfy the closer interface.
func (c *Closer) ErrInterrupt(now time.Time, duration time.Duration) {
}

// ErrConcurrencyLimitReject is ignored and exists only to satisfy the closer interface.
func (c *Closer) ErrConcurrencyLimitReject(now time.Time) {
}

// ErrShortCircuit is ignored and exists only to satisfy the closer interface.
func (c *Closer) ErrShortCircuit(now time.Time) {
}

// Closed resets the rater
func (c *Closer) Closed(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFailedReserve = now
	c.Rater.Reset(now)
}

// Opened resets the rater
func (c *Closer) Opened(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFailedReserve = now
	c.Rater.Reset(now)
}

// ShouldClose returns true if the rateopener has been successful for CloseOnHappyDuration amount of time.
func (c *Closer) ShouldClose(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return now.Sub(c.lastFailedReserve) > c.CloseOnHappyDuration
}

// Allow attempts to get a reservation from the rater.  If we are unable to reserve a value, we count this as a failure
// for the rater.
func (c *Closer) Allow(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ret := c.Rater.AttemptReserve(now)
	if !ret {
		c.lastFailedReserve = now
	}
	return ret
}

// Type check we are implementing the correct types for our rateopener
var _ circuit.OpenToClosed = &Closer{}
