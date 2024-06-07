package service

import (
	"fmt"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	"github.com/instill-ai/x/errmsg"
)

var ErrNoPermission = fmt.Errorf("no permission")
var ErrUnauthenticated = fmt.Errorf("unauthenticated")
var ErrRateLimiting = fmt.Errorf("rate limiting")
var ErrCanNotTriggerNonLatestPipelineRelease = fmt.Errorf("can not trigger non-latest pipeline release")
var ErrExceedMaxBatchSize = fmt.Errorf("the batch size can not exceed 32")
var ErrTriggerFail = fmt.Errorf("failed to trigger the pipeline")

var errCanNotUsePlaintextSecret = errmsg.AddMessage(
	fmt.Errorf("%w: plaintext value in credential field", errdomain.ErrInvalidArgument),
	"Plaintext values are forbidden in credential fields. You can create a secret and reference it with the syntax ${secrets.my-secret}.",
)
