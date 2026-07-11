package missingdep

// Missing is an unsatisfiable dependency: no component provides it.
type Missing interface{ Do() }

// Consumer requires a Missing, which nothing provides.
//
// @Service
type Consumer struct{}

// NewConsumer requires a Missing dependency.
func NewConsumer(m Missing) *Consumer { return &Consumer{} }
