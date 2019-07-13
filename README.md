# AIMDopener
[![Build Status](https://travis-ci.org/cep21/aimdopener.svg?branch=master)](https://travis-ci.org/cep21/aimdopener)
[![GoDoc](https://godoc.org/github.com/cep21/aimdopener?status.svg)](https://godoc.org/github.com/cep21/aimdopener)
[![Coverage Status](https://coveralls.io/repos/github/cep21/aimdopener/badge.svg)](https://coveralls.io/github/cep21/aimdopener)

Aimdopener is an opener implementation for [github.com/cep21/circuit](https://github.com/cep21/circuit).
It is a closer that increases how many requests it allows in an opened circuit according to 
[Additive increase/multiplicative decrease](https://en.wikipedia.org/wiki/Additive_increase/multiplicative_decrease)
algorithm.  The circuit closes when for a configured duration:

* No requests have failed
* No requests have been not allowed (additive increase is high enough for the rate) 

# Usage

Have your manager use a `rateopener.CloserFactory` and your circuits will be created of this type.

```go
    func ExampleCloserFactory() {
        // Tell your circuit manager to use the rate limited closer
        m := circuit.Manager{
            DefaultCircuitProperties: []circuit.CommandPropertiesConstructor{
                func(_ string) circuit.Config {
                    return circuit.Config{
                        General: circuit.GeneralConfig{
                            OpenToClosedFactory:CloserFactory(CloserConfig{
                                CloseOnHappyDuration: time.Second * 10,
                            }),
                        },
                    }
                },
            },
        }
        // Make circuit from manager
        c := m.MustCreateCircuit("example_circuit")
        // The closer should be a closer of this type
        _ = c.OpenToClose.(*Closer)
        // Output:
    }
```

# Benchmarks

Run on my mac.

```
    < go test -bench .
    goos: darwin
    goarch: amd64
    pkg: github.com/cep21/aimdopener/rateopener
    BenchmarkCloser_Allow_10-16                	10000000	       155 ns/op	       0 B/op	       0 allocs/op
    BenchmarkCloser_AllowSuccess_10-16         	 5000000	       392 ns/op	       0 B/op	       0 allocs/op
    BenchmarkCloser_AllowSuccessClose_10-16    	 3000000	       480 ns/op	       0 B/op	       0 allocs/op
    BenchmarkCloser_AllowFailureClose_10-16    	 3000000	       513 ns/op	       0 B/op	       0 allocs/op
    PASS
    ok  	github.com/cep21/aimdopener/rateopener	8.108s
```