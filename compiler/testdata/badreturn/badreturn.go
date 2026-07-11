package badreturn

// Bad has a constructor whose second return value is not error.
//
// @Service
type Bad struct{}

// NewBad returns (T, int); the second value must be error.
func NewBad() (*Bad, int) { return &Bad{}, 0 }
