// package errors contains domain errors that different layers can use to add
// meaning to an error and that middleware can transform to a status code or
// retry policy. This is implemented as a separate package in order to avoid
// cycle import errors.
//
// TODO When transforming domain errors to response codes, the middleware
// package should eventually use these errors only.
package errors

import (
	"fmt"
)

var (
	// ErrInvalidArgument is used when the provided argument is incorrect (e.g.
	// format, reserved).
	ErrInvalidArgument = fmt.Errorf("invalid resource ID")
	// ErrNotFound is used when a resource doesn't exist.
	ErrNotFound = fmt.Errorf("not found")
	// ErrInvalidCloneTarget is used when the pipeline clone target is not
	// valid. The format should be `<user-id>/<pipeline-id>` or
	// `<org-id>/<pipeline-id>`.
	ErrInvalidCloneTarget = fmt.Errorf("invalid target")
)
