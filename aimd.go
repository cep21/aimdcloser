package aimdcloser

import (
	"math"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter is any object that can dynamically alter its reservation rate to allow more or less requests over time.
type RateLimiter interface {
	// OnFailure is triggered each time we should lower our request rate.
	OnFailure(now time.Time)
	// OnSuccess is triggered each time we should increase our request rate.
	OnSuccess(now time.Time)
	// AttemptReserve is called when the application wants to ask if it should allow a request.
	AttemptReserve(now time.Time) bool
	// Reset the internal configuration of the rate limiter back to defaults.
	Reset(now time.Time)
}

// AIMD is https://en.wikipedia.org/wiki/Additive_increase/multiplicative_decrease
// It is *NOT* thread safe
type AIMD struct {
	// How many requests / sec are allowed in addition when a success happens.  A default o zero
	// does not increase the rate.
	AdditiveIncrease float64
	// What % (0.0, 1.0) of requests to allow fewer of on a failure.  A default of zero
	// does not decrease the rate.
	MultiplicativeDecrease float64
	// The initial rate of requests / sec to set an AIMD at when reset.
	// Default of zero means infinite bursts per second.  However, with a burst of zero it is zero
	InitialRate float64
	// Allow Burst limits in the period
	// Default 0 turns off AIMD entirely.
	Burst int

	// TODO: We may want to implement some of this ourselves.  Use and optimize later
	l *rate.Limiter
}

// AIMDConstructor constructs rate limiters according to the given parameters.  See documentation for AIMD for
// what each parameter means.
func AIMDConstructor(additiveIncrease float64, multiplicativeDecrease float64, initialRate float64, burst int) func() RateLimiter {
	return func() RateLimiter {
		return &AIMD{
			AdditiveIncrease:       additiveIncrease,
			MultiplicativeDecrease: multiplicativeDecrease,
			InitialRate:            initialRate,
			Burst:                  burst,
		}
	}
}

// Reset the RateLimiter back to the initial rate and burst
func (a *AIMD) Reset(now time.Time) {
	a.l = rate.NewLimiter(rate.Limit(a.InitialRate), a.Burst)
}

func (a *AIMD) init(now time.Time) {
	if a.l == nil {
		a.Reset(now)
	}
}

// OnFailure changes the limiter to decrease the current limit by MultiplicativeDecrease
func (a *AIMD) OnFailure(now time.Time) {
	a.init(now)
	a.l.SetLimitAt(now, rate.Limit(float64(a.l.Limit())*a.MultiplicativeDecrease))
}

// AttemptReserve tries to reserve a request inside the current time window.  Returns if the rate limiter allows
// you to reserve a request.
func (a *AIMD) AttemptReserve(now time.Time) bool {
	a.init(now)
	return a.l.AllowN(now, 1)
}

// Rate returns the current rate.
func (a *AIMD) Rate() float64 {
	if a.l == nil {
		if a.InitialRate == 0 {
			return math.Inf(1)
		}
		return a.InitialRate
	}
	return float64(a.l.Limit())
}

// OnSuccess increases the reserved limit for this period.
func (a *AIMD) OnSuccess(now time.Time) {
	a.init(now)
	a.l.SetLimitAt(now, rate.Limit(float64(a.l.Limit())+a.AdditiveIncrease))
}

var _ RateLimiter = &AIMD{}
