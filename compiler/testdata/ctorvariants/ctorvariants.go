package ctorvariants

// Proto is prototype-scoped and has a (T, error) constructor.
//
// @Service(scope="prototype")
type Proto struct{}

// NewProto returns (T, error).
func NewProto() (*Proto, error) { return &Proto{}, nil }

// Clock is a constructorless zero-field component (§13.5).
//
// @Component
type Clock struct{}

// Thing is provided by a nut returning (T, error).
type Thing struct{}

// ProvideThing is a nut with a (T, error) signature.
//
// @Nut
func ProvideThing() (*Thing, error) { return &Thing{}, nil }
