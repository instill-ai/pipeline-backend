package service

import "errors"

var ErrNoPermission = errors.New("no permission")
var ErrNotFound = errors.New("not found")
var ErrUnauthenticated = errors.New("unauthenticated")
var ErrRateLimiting = errors.New("rate limiting")
var ErrNamespaceTriggerQuotaExceed = errors.New("namespace trigger quota exceed")
var ErrNamespacePrivatePipelineQuotaExceed = errors.New("namespace private pipeline quota exceed")
