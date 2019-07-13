package aimdopener

import (
	"math"
	"time"

	"golang.org/x/time/rate"
)

// AIMD is https://en.wikipedia.org/wiki/Additive_increase/multiplicative_decrease
// It is *NOT* thread safe
type AIMD struct {
	// How many requests / sec are allowed in addition when a success happens
	AdditiveIncrease float64
	// What % (0.0, 1.0) of requests to allow fewer of on a failure.
	MultiplicativeDecrease float64
	// The initial rate of requests / sec to set an AIMD at when reset
	// Default of zero means infinite
	InitialRate float64
	// Allow Burst limits in the period
	// Default 0 turns off
	Burst int

	l *rate.Limiter
}

func (a *AIMD) Reset(now time.Time) {
	a.l = rate.NewLimiter(rate.Limit(a.InitialRate), a.Burst)
}

func (a *AIMD) init(now time.Time) {
	if a.l == nil {
		a.Reset(now)
	}
}

func (a *AIMD) OnFailure(now time.Time) {
	a.init(now)
	a.l.SetLimitAt(now, rate.Limit(float64(a.l.Limit())*a.MultiplicativeDecrease))
}

func (a *AIMD) AttemptReserve(now time.Time) bool {
	a.init(now)
	return a.l.AllowN(now, 1)
}

func (a *AIMD) Rate() float64 {
	if a.l == nil {
		if a.InitialRate == 0 {
			return math.Inf(1)
		}
		return a.InitialRate
	}
	return float64(a.l.Limit())
}

func (a *AIMD) OnSuccess(now time.Time) {
	a.init(now)
	a.l.SetLimitAt(now, rate.Limit(float64(a.l.Limit())+a.AdditiveIncrease))
}

var _ RateLimiter = &AIMD{}
