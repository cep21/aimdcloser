package aimdopener

import (
	"time"

	"golang.org/x/time/rate"
)

// AIMD is https://en.wikipedia.org/wiki/Additive_increase/multiplicative_decrease
// It is *NOT* thread safe
type AIMD struct {
	// How many requests / sec are allowed in addition when a success happens
	// Default .1
	AdditiveIncrease float64
	// What % (0.0, 1.0) of requests to allow fewer of on a failure.  Default .1
	MultiplicativeDecrease float64
	// The initial rate of requests / sec to set an AIMD at when reset
	// Default 1
	InitialRate float64
	// Allow Burst limits over current rate / sec.  Default 0
	Burst int

	l *rate.Limiter
}

func (a *AIMD) every() rate.Limit {
	if a.InitialRate == 0 {
		return rate.Every(time.Second)
	}
	return rate.Limit(a.InitialRate)
}

func (a *AIMD) Reset(now time.Time) {
	a.l = rate.NewLimiter(a.every(), a.Burst)
}

func (a *AIMD) OnFailure(now time.Time) {
	a.l.SetLimitAt(now, rate.Limit(float64(a.l.Limit())*a.MultiplicativeDecrease))
}

func (a *AIMD) AttemptReserve(now time.Time) bool {
	return a.l.AllowN(now, 1)
}

func (a *AIMD) Rate() float64 {
	return float64(a.l.Limit())
}

func (a *AIMD) OnSuccess(now time.Time) {
	a.l.SetLimitAt(now, rate.Limit(float64(a.l.Limit())+a.AdditiveIncrease))
}

var _ RateLimiter = &AIMD{}
