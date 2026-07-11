package config

import "github.com/zombocoder/goboot/compiler/testdata/diapp/domain"

// Config groups the application's bean providers.
//
// @Configuration
type Config struct{}

// sequentialGen is a trivial IDGenerator implementation.
type sequentialGen struct{}

func (sequentialGen) NewID() string { return "generated-id" }

// ProvideIDGenerator supplies the domain.IDGenerator bean.
//
// @Bean(name="idGenerator")
func ProvideIDGenerator() domain.IDGenerator {
	return sequentialGen{}
}
