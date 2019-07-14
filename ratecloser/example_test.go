package ratecloser_test

import (
	"time"

	"github.com/cep21/aimdcloser/ratecloser"
	"github.com/cep21/circuit/v3"
)

func ExampleCloserFactory() {
	// Tell your circuit manager to use the rate limited closer
	m := circuit.Manager{
		DefaultCircuitProperties: []circuit.CommandPropertiesConstructor{
			func(_ string) circuit.Config {
				return circuit.Config{
					General: circuit.GeneralConfig{
						OpenToClosedFactory: ratecloser.CloserFactory(ratecloser.CloserConfig{
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
	_ = c.OpenToClose.(*ratecloser.Closer)
	// Output:
}
