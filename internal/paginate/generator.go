package paginate

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TokenGeneratorWithSalt(salt string) TokenGenerator {
	return &tokenGenerator{salt}
}

// TokenGenerator generates a page token for a given index.
type TokenGenerator interface {
	Encode(uint64) string
	Decode(string) (uint64, error)
}

// InvalidTokenErr is the error returned if the token provided is not
// parseable by the TokenGenerator.
var InvalidTokenErr = status.Errorf(
	codes.InvalidArgument,
	"The field `page_token` is invalid.")

type tokenGenerator struct {
	salt string
}

func (t *tokenGenerator) Encode(i uint64) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%d", t.salt, i)))
}

func (t *tokenGenerator) Decode(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}

	bs, err := base64.StdEncoding.DecodeString(s)

	if err != nil {
		return 0, InvalidTokenErr
	}

	if !strings.HasPrefix(string(bs), t.salt) {
		return 0, InvalidTokenErr
	}

	i, err := strconv.ParseUint(strings.TrimPrefix(string(bs), t.salt), 10, 64)
	if err != nil {
		return 0, InvalidTokenErr
	}
	return i, nil
}
