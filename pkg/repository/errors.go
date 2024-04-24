package repository

import (
	"errors"

	"gorm.io/gorm"
)

var ErrPageTokenDecode = errors.New("page token decode error")
var ErrOwnerTypeNotMatch = errors.New("owner type not match")
var ErrNoDataDeleted = errors.New("no data deleted")
var ErrNoDataUpdated = errors.New("no data updated")
var ErrNotFound = gorm.ErrRecordNotFound
var ErrNameExists = errors.New("name or ID already exists")
