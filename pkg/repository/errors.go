package repository

import (
	"errors"
)

var ErrPageTokenDecode = errors.New("page token decode error")
var ErrOwnerTypeNotMatch = errors.New("owner type not match")
var ErrNoDataDeleted = errors.New("no data deleted")
var ErrNoDataUpdated = errors.New("no data updated")
var ErrNameExists = errors.New("name or ID already exists")
