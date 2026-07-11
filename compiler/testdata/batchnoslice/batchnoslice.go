// Package batchnoslice has an @Batch method with no slice parameter, which must
// raise GOBREP001.
package batchnoslice

import "context"

// @Application(name="batch-no-slice")
type Application struct{}

// Thing is the entity.
type Thing struct{ ID string }

// ThingRepository is a generated repository.
//
// @Repository(generate=true)
type ThingRepository interface {
	// Bad has @Batch but nothing to iterate.
	//
	// @Batch(`INSERT INTO things (id) VALUES (:id)`)
	Bad(ctx context.Context, id string) error
}
