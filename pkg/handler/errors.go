package handler

import "errors"

var ErrCheckUpdateImmutableFields = errors.New("update immutable fields error")
var ErrCheckOutputOnlyFields = errors.New("can not contain output only fields")
var ErrCheckRequiredFields = errors.New("required fields missing")
var ErrFieldMask = errors.New("field mask error")
var ErrResourceID = errors.New("resource ID error")
var ErrSematicVersion = errors.New("not a legal version, should be the formate vX.Y.Z or vX.Y.Z-identifiers")
var ErrUpdateMask = errors.New("update mask error")
