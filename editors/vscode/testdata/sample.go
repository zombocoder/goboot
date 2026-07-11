// Package sample is a scratch file for eyeballing the goboot annotation
// highlighting. It is excluded from the packaged extension.
package sample

import "context"

// @Application(name="sample")
type Application struct{}

// @RestController
// @RequestMapping(path="/widgets")
type WidgetController struct{}

// @GetMapping(path="/{id}")
// @Traced
// @Timed
// @Timeout("2s")
// @Retry(maxAttempts=3, delay="20ms")
func (c *WidgetController) Get(ctx context.Context) error { return nil }

// @Repository(generate=true, entity="Widget", table="widgets")
type WidgetRepository interface {
	// @Query(`SELECT id, name FROM widgets WHERE id = :id`)
	FindByID(ctx context.Context, id string) error
}

// @Scheduled(fixedRate=2, timeUnit=TimeUnit.MINUTES)
// @Profile(["prod", "staging"])
func tick() {}
