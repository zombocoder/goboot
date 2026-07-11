// Package api is a fixture that trips each lint rule: a duplicate operationId
// (two List handlers), an uppercase path segment, and a trailing slash.
package api

import "context"

// @Application(name="lint-demo")
type Application struct{}

// Users is a well-formed controller.
//
// @RestController
// @RequestMapping(path="/users")
type Users struct{}

// NewUsers constructs the controller.
func NewUsers() *Users { return &Users{} }

// List handles GET /users.
//
// @GetMapping(path="")
func (c *Users) List(ctx context.Context) error { return nil }

// Accounts uses a non-lowercase base path and duplicates the List operationId.
//
// @RestController
// @RequestMapping(path="/Accounts")
type Accounts struct{}

// NewAccounts constructs the controller.
func NewAccounts() *Accounts { return &Accounts{} }

// List handles GET /Accounts — duplicate operationId "List" (LINT001) plus a
// non-lowercase segment (LINT002).
//
// @GetMapping(path="")
func (c *Accounts) List(ctx context.Context) error { return nil }

// Legacy handles GET /Accounts/legacy/ — a trailing slash (LINT003).
//
// @GetMapping(path="/legacy/")
func (c *Accounts) Legacy(ctx context.Context) error { return nil }
