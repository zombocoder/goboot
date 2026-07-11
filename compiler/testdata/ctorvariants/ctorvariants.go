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

// Thing is provided by a bean returning (T, error).
type Thing struct{}

// ProvideThing is a bean with a (T, error) signature.
//
// @Bean
func ProvideThing() (*Thing, error) { return &Thing{}, nil }
