// Package domain holds the interfaces and entities the example application
// wires together.
package domain

// User is a domain entity.
type User struct {
	ID   string
	Name string
}

// UserUseCase is the service-layer contract implemented by the service and
// injected into the controller.
type UserUseCase interface {
	GetUser(id string) (*User, error)
}

// UserRepository is the persistence contract implemented by the repository and
// injected into the service.
type UserRepository interface {
	FindByID(id string) (*User, error)
}

// IDGenerator produces identifiers and is provided by a bean.
type IDGenerator interface {
	NewID() string
}
