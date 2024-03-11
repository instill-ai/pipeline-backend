package service

import "errors"

var ErrNoPermission = errors.New("no permission")
var ErrNotFound = errors.New("not found")
var ErrConnectorNameError = errors.New("connector name error")
var ErrConnectorNotFound = errors.New("connector not found")
var ErrUnauthenticated = errors.New("unauthenticated")
var ErrRateLimiting = errors.New("rate limiting")
var ErrCanNotTriggerNonLatestPipelineRelease = errors.New("can not trigger non-latest pipeline release")
var ErrExceedMaxBatchSize = errors.New("the batch size can not exceed 32")
