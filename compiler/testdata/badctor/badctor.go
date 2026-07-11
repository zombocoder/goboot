package badctor

// Bad has a constructor with an unsupported signature: three return values.
//
// @Service
type Bad struct{}

// NewBad returns three values, which is not a valid constructor signature.
func NewBad() (*Bad, int, error) { return &Bad{}, 0, nil }
