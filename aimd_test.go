package aimdcloser

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"
)

func equalFloat(t *testing.T, expected float64, given float64) {
	t.Helper()
	delta := math.Abs(expected - given)
	if delta >= .00001 {
		t.Errorf("Unexpected value.  Expected %f Given %f", expected, given)
	}
}

func equalInt(t *testing.T, expected int, given int) {
	t.Helper()
	if expected != given {
		t.Errorf("Unexpected value.  Expected %d Given %d", expected, given)
	}
}

func expect(t *testing.T, b bool, msg string) {
	t.Helper()
	if !b {
		t.Error(msg)
	}
}

func expectNilErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Error(err.Error())
	}
}

func TestAIMDConstructor(t *testing.T) {
	c := AIMDConstructor(10, .1, 10, 10)
	r := c().(*AIMD)
	equalFloat(t, 10, r.AdditiveIncrease)
	equalFloat(t, .1, r.MultiplicativeDecrease)
	equalFloat(t, 10, r.InitialRate)
	equalInt(t, 10, r.Burst)
	equalFloat(t, 10, r.Rate())

}

func TestAIMDempty(t *testing.T) {
	a := AIMD{}
	equalFloat(t, math.Inf(1), a.Rate())
}

func TestAIMDNormalRate(t *testing.T) {
	a := AIMD{
		// Allow burst item every half second
		InitialRate: 2,
		Burst:       1,
	}
	now := time.Now()
	for i := 0; i < 10; i++ {
		expect(t, a.AttemptReserve(now), "expected to be able to reserve")
		expect(t, !a.AttemptReserve(now), "expected to not be able to reserve")
		now = now.Add(time.Second / 2)

	}
}

func TestAIMDFailures(t *testing.T) {
	a := AIMD{
		// Allow burst item every half second
		InitialRate:            2,
		Burst:                  1,
		AdditiveIncrease:       .1,
		MultiplicativeDecrease: .9,
	}
	now := time.Now()
	a.Reset(now)
	a.OnFailure(now)
	a.OnFailure(now)
	a.OnFailure(now)
	equalFloat(t, a.Rate(), 2*.9*.9*.9)
}

type RateLimitedService struct {
	ProcessRate   time.Duration
	requestBuffer chan struct{}

	wg   sync.WaitGroup
	done chan struct{}
}

func (r *RateLimitedService) Close() error {
	close(r.done)
	r.wg.Wait()
	return nil
}

func (r *RateLimitedService) GiveRequest() error {
	select {
	case <-r.done:
		return errors.New("done")
	case r.requestBuffer <- struct{}{}:
		return nil
	default:
		return errors.New("overflow")
	}
}

func (r *RateLimitedService) Process() {
	r.wg.Add(1)
	r.done = make(chan struct{})
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-r.done:
				return
			case <-time.After(r.ProcessRate):
				select {
				case <-r.done:
					return
				case <-r.requestBuffer:
				}
			}
		}
	}()
}

func TestAIMDLevelingOut(t *testing.T) {
	// This test is really too random to test for, but I expect something like 10 req / sec in the final rate
	//t.Skip("Just for experimentation")
	s := RateLimitedService{
		ProcessRate:   time.Millisecond * 100,
		requestBuffer: make(chan struct{}, 10),
	}
	s.Process()
	a := AIMD{
		InitialRate:            float64(time.Second / time.Microsecond),
		Burst:                  10,
		AdditiveIncrease:       1,
		MultiplicativeDecrease: .5,
	}
	errs := 0
	success := 0
	a.OnFailure(time.Now())
	for start := time.Now(); time.Since(start) < time.Second*3; {
		// 1000-ish requests / sec
		time.Sleep(time.Millisecond)
		if a.AttemptReserve(time.Now()) {
			if s.GiveRequest() != nil {
				a.OnFailure(time.Now())
				errs++
			} else {
				a.OnSuccess(time.Now())
				success++
			}
		}
	}
	expectNilErr(t, s.Close())
	t.Logf("Ration of %d / %d = %f", errs, errs+success, float64(errs)/(float64(errs+success)))
	t.Logf("final rate was %f", a.Rate())
}

func TestAIMDBurst(t *testing.T) {
	a := AIMD{
		InitialRate: 1,
		Burst:       10,
	}

	now := time.Now()
	for i := 0; i < a.Burst; i++ {
		if !a.AttemptReserve(now) {
			t.Errorf("expected burst at %d", i)
		}
	}
	if a.AttemptReserve(now) {
		t.Error("expected not to be able to burst")
	}

	// Almost at the end of the period
	now = now.Add(time.Second - time.Nanosecond*2)
	if a.AttemptReserve(now) {
		t.Error("expected to not burst at end of period")
	}

	now = now.Add(time.Nanosecond * 2)
	if !a.AttemptReserve(now) {
		t.Error("expected to allow an item")
	}
}
