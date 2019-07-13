package aimdopener

import (
	"sync"
	"time"

	"github.com/cep21/circuit/v3"
)

type RateLimiter interface {
	OnFailure(now time.Time)
	OnSuccess(now time.Time)
	AttemptReserve(now time.Time) bool
	Reset(now time.Time)
}

type Closer struct {
	Rater                RateLimiter
	CloseOnHappyDuration time.Duration
	lastFailedReserve    time.Time
	mu                   sync.Mutex
}

type OpenerConfig struct {
	// RateLimiter constructs new rate limiters for circuits.
	RateLimiter func() RateLimiter
	// CloseOnHappyDuration gives a duration that passing requests cause the circuit to close.
	CloseOnHappyDuration time.Duration
}

func (o *OpenerConfig) merge(other OpenerConfig) {
	if o.CloseOnHappyDuration == 0 {
		o.CloseOnHappyDuration = other.CloseOnHappyDuration
	}
	if o.RateLimiter == nil {
		o.RateLimiter = other.RateLimiter
	}
}

var defaultConfig = OpenerConfig{
	RateLimiter:          AIMDConstructor(.1, .5, float64(time.Microsecond/time.Second), 10),
	CloseOnHappyDuration: time.Second * 5,
}

func CloserFactory(conf OpenerConfig) func() circuit.OpenToClosed {
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

func (c *Closer) Success(now time.Time, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Rater.OnSuccess(now)
}

func (c *Closer) ErrFailure(now time.Time, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Rater.OnFailure(now)
	c.lastFailedReserve = now
}

func (c *Closer) ErrTimeout(now time.Time, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Rater.OnFailure(now)
	c.lastFailedReserve = now
}

func (c *Closer) ErrBadRequest(now time.Time, duration time.Duration) {
}

func (c *Closer) ErrInterrupt(now time.Time, duration time.Duration) {
}

func (c *Closer) ErrConcurrencyLimitReject(now time.Time) {
}

func (c *Closer) ErrShortCircuit(now time.Time) {
}

func (c *Closer) Closed(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFailedReserve = now
	c.Rater.Reset(now)
}

func (c *Closer) Opened(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFailedReserve = now
	c.Rater.Reset(now)
}

func (c *Closer) ShouldClose(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return now.Sub(c.lastFailedReserve) > c.CloseOnHappyDuration
}

func (c *Closer) Allow(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ret := c.Rater.AttemptReserve(now)
	if !ret {
		c.lastFailedReserve = now
	}
	return ret
}

var _ circuit.OpenToClosed = &Closer{}
