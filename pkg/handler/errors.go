package handler

import "errors"

var ErrCheckUpdateImmutableFields = errors.New("update immutable fields error")
var ErrCheckOutputOnlyFields = errors.New("can not contain output only fields")
var ErrCheckRequiredFields = errors.New("required fields missing")
var ErrFieldMask = errors.New("field mask error")
var ErrResourceID = errors.New("resource ID error")
var ErrSematicVersion = errors.New("not a sematic version")
var ErrUpdateMask = errors.New("update mask error")
