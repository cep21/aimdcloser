package aimdopener

import (
	"sync"
	"time"

	"github.com/cep21/circuit/v3"
)

type RateLimiter interface {
	OnFailure(now time.Time)
	OnSuccess(now time.Time)
	Reset(now time.Time)
	AttemptReserve(now time.Time) bool
}

type Closer struct {
	Rater                RateLimiter
	CloseOnHappyDuration time.Duration
	lastFailedReserve    time.Time
	mu                   sync.Mutex
}

type OpenerConfig struct {
	AdditiveIncrease       float64
	// You *really* want this to be (0,1)
	MultiplicativeDecrease float64
	// You want this to be very fast at first, and rely on MultiplicativeDecrease to level out
	InitialRate            float64
	Burst                  int
	CloseOnHappyDuration   time.Duration
}

func (o OpenerConfig) merge(other OpenerConfig) {
	if o.AdditiveIncrease == 0 {
		o.AdditiveIncrease = other.AdditiveIncrease
	}
	if o.MultiplicativeDecrease == 0 {
		o.MultiplicativeDecrease = other.MultiplicativeDecrease
	}
	if o.InitialRate == 0 {
		o.InitialRate = other.InitialRate
	}
	if o.Burst == 0 {
		o.Burst = other.Burst
	}
	if o.CloseOnHappyDuration == 0 {
		o.CloseOnHappyDuration = other.CloseOnHappyDuration
	}
}

var defaultConfig = OpenerConfig{
	AdditiveIncrease:       .1,
	MultiplicativeDecrease: .5,
	InitialRate:            float64(time.Microsecond / time.Second),
	Burst:                  10,
	CloseOnHappyDuration:   time.Second * 5,
}

func CloserFactory(conf OpenerConfig) func() circuit.OpenToClosed {
	return func() circuit.OpenToClosed {
		c := conf
		c.merge(defaultConfig)
		return &Closer{
			Rater: &AIMD{
				AdditiveIncrease:       c.AdditiveIncrease,
				MultiplicativeDecrease: c.MultiplicativeDecrease,
				InitialRate:            c.InitialRate,
				Burst:                  c.Burst,
			},
			CloseOnHappyDuration: c.CloseOnHappyDuration,
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
