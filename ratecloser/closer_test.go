package ratecloser

import (
	"sync"
	"testing"
	"time"

	"github.com/cep21/aimdcloser"
	"github.com/cep21/circuit/v3"
)

func TestCloserConfig(t *testing.T) {
	c := CloserConfig{}
	c.merge(defaultConfig)
	if c.RateLimiter == nil {
		t.Error("expect non nil rate limiter")
	}
	if c.CloseOnHappyDuration == 0 {
		t.Error("Expect non zero happy duration")
	}
}

func TestCloserFactory(t *testing.T) {
	// Compile test to verify we implement the correct type for circuit.
	_ = circuit.GeneralConfig{
		OpenToClosedFactory: CloserFactory(defaultConfig),
	}
	t.Run("default", func(t *testing.T) {
		cfg := CloserConfig{}
		factory := CloserFactory(cfg)
		closer := factory().(*Closer)
		if closer.CloseOnHappyDuration != defaultConfig.CloseOnHappyDuration {
			t.Error("Expect to get happy duration from default")
		}
		if closer.Rater == nil {
			t.Error("Expect to get a non nil rater")
		}
	})
	t.Run("explicit", func(t *testing.T) {
		cfg := CloserConfig{
			CloseOnHappyDuration: time.Second,
		}
		factory := CloserFactory(cfg)
		closer := factory().(*Closer)
		if closer.CloseOnHappyDuration != time.Second {
			t.Error("Expect to get happy duration from explicit set")
		}
	})
}

func TestCloser_ShouldClose(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, float64(time.Microsecond/time.Second), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("atstart", func(t *testing.T) {
		closer := factory().(*Closer)
		if closer.ShouldClose(time.Now()) {
			t.Error("Expect not to be able to close at start")
		}
	})
	t.Run("much_later", func(t *testing.T) {
		closer := factory().(*Closer)
		now := time.Now()
		closer.Success(now, time.Millisecond)
		now = now.Add(time.Second * 6)
		if !closer.ShouldClose(now) {
			t.Error("Expect to be able to close after 6 sec")
		}
	})
	t.Run("withfailure", func(t *testing.T) {
		closer := factory().(*Closer)
		now := time.Now()
		closer.Success(now, time.Millisecond)
		now = now.Add(time.Second * 3)
		closer.ErrFailure(now, time.Millisecond)
		now = now.Add(time.Second)
		if closer.ShouldClose(now) {
			t.Error("Expect to not be able to close with a failure")
		}
	})
	t.Run("with too many requests", func(t *testing.T) {
		now := time.Now()
		closer := factory()
		for i := 0; i < 1000; i++ {
			closer.Allow(now)
			closer.Success(now, time.Millisecond)
			if closer.ShouldClose(now) {
				t.Error("expect to not be able to close with so many at once")
			}
		}
		now = now.Add(time.Nanosecond)
		if closer.Allow(now) {
			t.Error("Should not be able to allow after 1000 req in same period")
		}
		if !closer.(*Closer).lastFailedReserve.Equal(now) {
			t.Error("Should reset lastFailedReserve to now (and some)")
		}
	})
}

func TestCloser_Allow(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("atstart", func(t *testing.T) {
		closer := factory()
		if !closer.Allow(time.Now()) {
			t.Error("Expect to allow a request at the start")
		}
	})
	t.Run("atburst", func(t *testing.T) {
		closer := factory()
		now := time.Now()
		for i := 0; i < 10; i++ {
			if !closer.Allow(now) {
				t.Error("Expect to allow a request in the burst range")
			}
		}
		if closer.Allow(now) {
			t.Error("Expect to now allow requests outside the burst range")
		}
	})
}

func TestCloser_Opened(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("rapidreset", func(t *testing.T) {
		closer := factory()
		now := time.Now()
		for i := 0; i < 1000; i++ {
			if !closer.Allow(now) {
				t.Error("Expect to allow a request in the burst range")
			}
			closer.Opened(now)
		}
	})
}

func TestCloser_Success(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("increases", func(t *testing.T) {
		closer := factory().(*Closer)
		now := time.Now()
		startingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		closer.Success(now, time.Millisecond)
		endingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		if endingRate <= startingRate {
			t.Error("expected rate to increase with success")
		}
	})
}

func TestCloser_ErrFailure(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("decrease", func(t *testing.T) {
		closer := factory().(*Closer)
		now := time.Now()
		startingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		closer.ErrFailure(now, time.Millisecond)
		endingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		if endingRate >= startingRate {
			t.Error("expected rate to decrease with ErrFailure")
		}
	})
}

func TestCloser_Closed(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("rapidreset", func(t *testing.T) {
		closer := factory()
		now := time.Now()
		for i := 0; i < 1000; i++ {
			if !closer.Allow(now) {
				t.Error("Expect to allow a request in the burst range")
			}
			closer.Closed(now)
		}
	})
}

func TestCloser_ErrBadRequest(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("same", func(t *testing.T) {
		closer := factory().(*Closer)
		now := time.Now()
		startingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		closer.ErrBadRequest(now, time.Millisecond)
		endingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		if endingRate != startingRate {
			t.Error("expected rate to stay same with ErrBadRequest")
		}
	})
}

func TestCloser_ErrConcurrencyLimitReject(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("same", func(t *testing.T) {
		closer := factory().(*Closer)
		now := time.Now()
		startingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		closer.ErrConcurrencyLimitReject(now)
		endingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		if endingRate != startingRate {
			t.Error("expected rate to stay same with ErrConcurrencyLimitReject")
		}
	})
}

func TestCloser_ErrInterrupt(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("same", func(t *testing.T) {
		closer := factory().(*Closer)
		now := time.Now()
		startingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		closer.ErrInterrupt(now, time.Millisecond)
		endingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		if endingRate != startingRate {
			t.Error("expected rate to stay same with ErrInterrupt")
		}
	})
}

func TestCloser_ErrShortCircuit(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("same", func(t *testing.T) {
		closer := factory().(*Closer)
		now := time.Now()
		startingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		closer.ErrShortCircuit(now)
		endingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		if endingRate != startingRate {
			t.Error("expected rate to stay same with ErrShortCircuit")
		}
	})
}

func TestCloser_ErrTimeout(t *testing.T) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	t.Run("same", func(t *testing.T) {
		closer := factory().(*Closer)
		now := time.Now()
		startingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		closer.ErrTimeout(now, time.Second)
		endingRate := closer.Rater.(*aimdcloser.AIMD).Rate()
		if endingRate >= startingRate {
			t.Error("expected rate to decrease with ErrTimeout")
		}
	})
}

func BenchmarkCloser_Allow_10(b *testing.B) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	b.ReportAllocs()
	closer := factory()
	now := time.Now()
	closer.Opened(now)
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N/10; i++ {
				now = now.Add(time.Nanosecond)
				closer.Allow(now)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkCloser_AllowSuccess_10(b *testing.B) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	b.ReportAllocs()
	closer := factory()
	now := time.Now()
	closer.Opened(now)
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N/10; i++ {
				now = now.Add(time.Nanosecond)
				closer.Allow(now)
				closer.Success(now, time.Millisecond)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkCloser_AllowSuccessClose_10(b *testing.B) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	b.ReportAllocs()
	closer := factory()
	now := time.Now()
	closer.Opened(now)
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N/10; i++ {
				now = now.Add(time.Nanosecond)
				closer.Allow(now)
				closer.Success(now, time.Millisecond)
				closer.ShouldClose(now)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkCloser_AllowFailureClose_10(b *testing.B) {
	factory := CloserFactory(CloserConfig{
		RateLimiter:          aimdcloser.AIMDConstructor(.1, .5, 1/time.Microsecond.Seconds(), 10),
		CloseOnHappyDuration: time.Second * 5,
	})
	b.ReportAllocs()
	closer := factory()
	now := time.Now()
	closer.Opened(now)
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N/10; i++ {
				now = now.Add(time.Nanosecond)
				closer.Allow(now)
				closer.ErrFailure(now, time.Millisecond)
				closer.ShouldClose(now)
			}
		}()
	}
	wg.Wait()
}
