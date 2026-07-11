package cycle

// A depends on B and B depends on A, forming a cycle.

// @Service
type A struct{}

func NewA(b *B) *A { return &A{} }

// @Service
type B struct{}

func NewB(a *A) *B { return &B{} }
