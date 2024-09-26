package recipe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/x/errmsg"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
)

var (
	// ErrInvalidConnectionReference indicates a malformed or missing
	// connection reference.
	ErrInvalidConnectionReference = fmt.Errorf("%w: connection reference", errdomain.ErrInvalidArgument)

	connRefPrefix, connRefSuffix   = fmt.Sprintf("${%s.", constant.SegConnection), "}"
	lConnRefPrefix, lConnRefSuffix = len(connRefPrefix), len(connRefSuffix)
)

// ConnectionIDFromReference ...
func ConnectionIDFromReference(ref string) (string, error) {
	if !(strings.HasPrefix(ref, connRefPrefix) && strings.HasSuffix(ref, connRefSuffix)) {
		return "", errmsg.AddMessage(
			ErrInvalidConnectionReference,
			"String setup only supports connection references (${connection.<conn-id>}).",
		)
	}

	return ref[lConnRefPrefix : len(ref)-lConnRefSuffix], nil
}

// FetchReferencedSetup takes a connection reference, fetches that connection
// from a repository and returns its setup.
//
// The connection fetcher method is passed as an argument. Ideally, this
// should be defined as a method in service.Service but the current dependency
// hierarchy prevents packages like worker to leverage it. Due to the
// considerable refactor this would imply, this method was extracted to the
// recipe package, avoiding a dependency cycle.
func FetchReferencedSetup(
	ctx context.Context,
	ref string,
	getNamespaceConnectionByID func(context.Context, string) (*datamodel.Connection, error),
) (map[string]any, error) {
	id, err := ConnectionIDFromReference(ref)
	if err != nil {
		return nil, err
	}

	conn, err := getNamespaceConnectionByID(ctx, id)
	if err != nil {
		if !errors.Is(err, errdomain.ErrNotFound) {
			return nil, fmt.Errorf("fetching connection: %w", err)
		}

		return nil, errmsg.AddMessage(
			ErrInvalidConnectionReference,
			fmt.Sprintf("Connection %s doesn't exist.", id),
		)
	}

	var setup map[string]any
	if err := json.Unmarshal(conn.Setup, &setup); err != nil {
		return nil, fmt.Errorf("unmarshalling setup: %w", err)
	}

	return setup, nil
}
