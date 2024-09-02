package repository

import (
	"errors"
	"fmt"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
)

var ErrOwnerTypeNotMatch = errors.New("owner type not match")
var ErrNoDataDeleted = errors.New("no data deleted")
var ErrNoDataUpdated = errors.New("no data updated")

func newPageTokenErr(err error) error {
	return fmt.Errorf("%w: invalid page token: %w", errdomain.ErrInvalidArgument, err)
}
