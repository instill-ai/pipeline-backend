package service

import "errors"

var ErrNoPermission = errors.New("no permission")
var ErrNotFound = errors.New("not found")
var ErrUnauthenticated = errors.New("unauthenticated")
var ErrRateLimiting = errors.New("rate limiting")
var ErrCanNotTriggerNonLatestPipelineRelease = errors.New("can not trigger non-latest pipeline release")
var ErrExceedMaxBatchSize = errors.New("the batch size can not exceed 32")
var ErrCanNotUsePlaintextSecret = errors.New("cannot use plaintext value for credential fields; must use secret reference")
var ErrTriggerFail = errors.New("failed to trigger the pipeline")
