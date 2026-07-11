package repobadparam

import "context"

// Bad references a SQL parameter :missing that has no matching method argument.
//
// @Repository(generate=true)
type Bad interface {
	// @Query(`SELECT id FROM users WHERE id = :missing`)
	Find(ctx context.Context, id string) (string, error)
}
