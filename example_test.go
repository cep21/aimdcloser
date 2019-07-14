package aimdcloser_test

import (
	"fmt"
	"time"

	"github.com/cep21/aimdcloser"
)

func ExampleAIMD_AttemptReserve() {
	x := aimdcloser.AIMD{
		// Add .1 req / sec for each successful request
		AdditiveIncrease: .1,
		// Decrease the rate by .5 for each failure
		MultiplicativeDecrease: .5,
		// Allows one request per millisecond
		InitialRate: 1 / time.Millisecond.Seconds(),
		// Burst to 10 in a time period
		Burst: 10,
	}
	if x.AttemptReserve(time.Now()) {
		fmt.Println("We make a request")
	} else {
		fmt.Println("We skip making a request")
	}
	// Output: We make a request
}

func ExampleAIMD_OnSuccess() {
	x := aimdcloser.AIMD{
		// Add .1 req / sec for each successful request
		AdditiveIncrease: .1,
		// Decrease the rate by .5 for each failure
		MultiplicativeDecrease: .5,
		// Allows one request per millisecond
		InitialRate: 1 / time.Millisecond.Seconds(),
		// Burst to 10 in a time period
		Burst: 10,
	}
	if x.AttemptReserve(time.Now()) {
		fmt.Println("Request worked")
		x.OnSuccess(time.Now())
	}
	// Output: Request worked
}

func ExampleAIMD_OnFailure() {
	x := aimdcloser.AIMD{
		// Add .1 req / sec for each successful request
		AdditiveIncrease: .1,
		// Decrease the rate by .5 for each failure
		MultiplicativeDecrease: .5,
		// Allows one request per millisecond
		InitialRate: 1 / time.Millisecond.Seconds(),
		// Burst to 10 in a time period
		Burst: 10,
	}
	if x.AttemptReserve(time.Now()) {
		fmt.Println("Request failed")
		x.OnFailure(time.Now())
	}
	// Output: Request failed
}
