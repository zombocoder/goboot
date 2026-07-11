package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// Application ties an HTTP server to a Lifecycle and drives orderly startup and
// graceful shutdown (§31). Generated wiring builds one of these and calls Run.
type Application struct {
	// Server is the HTTP server to run; may be nil for a non-HTTP application.
	Server *http.Server
	// Lifecycle manages component start/stop hooks; may be nil.
	Lifecycle *Lifecycle
	// Scheduler runs @Scheduled background tasks; may be nil.
	Scheduler *Scheduler
}

// Run starts the lifecycle, serves HTTP, and blocks until the context is
// cancelled or the server stops on its own, then shuts down gracefully (§31). A
// lifecycle startup failure aborts before the server starts and is returned.
func (a *Application) Run(ctx context.Context) error {
	if a.Lifecycle != nil {
		if err := a.Lifecycle.Start(ctx); err != nil {
			return err
		}
	}
	if a.Scheduler != nil {
		a.Scheduler.Start(ctx)
	}

	serverErr := make(chan error, 1)
	if a.Server != nil {
		go func() {
			err := a.Server.ListenAndServe()
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			}
			serverErr <- err
		}()
	}

	select {
	case <-ctx.Done():
		return a.Shutdown(context.WithoutCancel(ctx))
	case err := <-serverErr:
		if a.Scheduler != nil {
			a.Scheduler.Stop()
		}
		stopErr := a.stopLifecycle(context.Background())
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return stopErr
	}
}

// Shutdown gracefully stops the HTTP server and then the lifecycle, within the
// configured shutdown timeout (§30.5).
func (a *Application) Shutdown(ctx context.Context) error {
	timeout := DefaultShutdownTimeout
	if a.Lifecycle != nil {
		timeout = a.Lifecycle.ShutdownTimeout()
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Stop scheduled tasks before tearing down the server and components.
	if a.Scheduler != nil {
		a.Scheduler.Stop()
	}

	var errs []error
	if a.Server != nil {
		if err := a.Server.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("http shutdown: %w", err))
		}
	}
	if err := a.stopLifecycle(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// stopLifecycle stops the lifecycle if present.
func (a *Application) stopLifecycle(ctx context.Context) error {
	if a.Lifecycle == nil {
		return nil
	}
	return a.Lifecycle.Stop(ctx)
}
