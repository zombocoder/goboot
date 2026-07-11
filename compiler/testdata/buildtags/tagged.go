//go:build goboot_on

package buildtags

// TaggedService is compiled only under the goboot_on build tag.
//
// @Service(name="taggedService")
type TaggedService struct{}
