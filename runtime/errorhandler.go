package runtime

import (
	"context"
	"errors"
	"net/http"
)

// ErrorHandler renders an error as an HTTP response (§22). It is the single
// place controller errors are converted into a Problem body.
type ErrorHandler interface {
	Handle(ctx context.Context, w http.ResponseWriter, r *http.Request, err error)
}

// DefaultErrorHandler maps errors to Problem responses using ToProblem and
// writes them with the configured ResponseWriter.
type DefaultErrorHandler struct {
	Writer ResponseWriter
	// ExposeDetail forces inclusion of the underlying error message even for
	// 5xx responses. It defaults to false so internal messages are not leaked
	// in production (§50).
	ExposeDetail bool
}

// Handle implements ErrorHandler.
func (h DefaultErrorHandler) Handle(ctx context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	writer := h.Writer
	if writer == nil {
		writer = JSONResponseWriter{}
	}
	problem := toProblem(err, h.ExposeDetail)
	_ = writer.Write(ctx, w, problem.Status, problem)
}

// ToProblem converts an error into a Problem, following the resolution priority
// of §23.5: validation errors first, then explicit HTTP-status/coded errors,
// then a generic internal error. Internal (5xx) details are withheld unless
// exposeDetail is set (§50).
func ToProblem(err error) Problem { return toProblem(err, false) }

func toProblem(err error, exposeDetail bool) Problem {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return Problem{
			Type:   "validation_error",
			Title:  "Request validation failed",
			Status: http.StatusBadRequest,
			Code:   ve.Code(),
			Errors: ve.Fields,
		}
	}

	status := StatusOf(err)
	code := CodeOf(err)
	problem := Problem{
		Type:   problemType(code),
		Title:  http.StatusText(status),
		Status: status,
		Code:   code,
	}
	if status < http.StatusInternalServerError || exposeDetail {
		problem.Detail = err.Error()
	}
	return problem
}

// problemType returns a stable problem type identifier.
func problemType(code string) string {
	if code != "" {
		return code
	}
	return "error"
}
