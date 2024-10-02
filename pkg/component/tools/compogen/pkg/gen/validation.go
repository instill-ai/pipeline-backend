package gen

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

func fieldErrorMessage(fe validator.FieldError) string {
	var msg string
	switch fe.Tag() {
	case "required", "required_if":
		msg = "is required"
	case "len":
		msg = "has an invalid length"
	case "gt":
		msg = "doesn't reach the minimum value / number of elements"
	case "semver":
		msg = "must be valid SemVer 2.0.0"
	case "url":
		msg = "must be a valid URL"
	default:
		return fe.Error() // default error
	}

	return fmt.Sprintf("%s: %s field %s", fe.Namespace(), fe.Field(), msg)
}

func asValidationError(err error) error {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return err
	}

	errs := make([]error, len(ve))
	for i, fe := range ve {
		errs[i] = fmt.Errorf(fieldErrorMessage(fe))
	}

	return errors.Join(errs...)
}
