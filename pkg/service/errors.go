package service

import (
	"errors"

	"github.com/instill-ai/x/errmsg"
)

var ErrNoPermission = errors.New("no permission")
var ErrNotFound = errors.New("not found")
var ErrUnauthenticated = errors.New("unauthenticated")
var ErrRateLimiting = errors.New("rate limiting")
var ErrCanNotTriggerNonLatestPipelineRelease = errors.New("can not trigger non-latest pipeline release")
var ErrExceedMaxBatchSize = errors.New("the batch size can not exceed 32")
var ErrTriggerFail = errors.New("failed to trigger the pipeline")

// ErrCanNotUsePlaintextSecret prevents users from adding connection details
// with plaintext values.
var ErrCanNotUsePlaintextSecret = errmsg.AddMessage(
	errors.New("plaintext value in credential field"),
	"Plaintext values are forbidden in credential fields. You can create a secret and reference it with the syntax ${secrets.my-secret}.",
)
