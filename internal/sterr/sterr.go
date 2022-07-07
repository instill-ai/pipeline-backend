package sterr

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/x/sterr"
)

// CreateErrorBadRequest is a wrapper for the CreateErrorBadRequest in x/sterr
func CreateErrorBadRequest(msg string, field string, desc string) *status.Status {
	logger, _ := logger.GetZapLogger()
	st, err := sterr.CreateErrorBadRequest(
		msg,
		[]*errdetails.BadRequest_FieldViolation{
			{
				Field:       field,
				Description: desc,
			},
		},
	)
	if err != nil {
		logger.Error(err.Error())
	}
	return st
}

// CreateErrorPreconditionFailure is a wrapper for the CreateErrorPreconditionFailure in x/sterr
func CreateErrorPreconditionFailure(msg string, tp string, sub string, desc string) *status.Status {
	logger, _ := logger.GetZapLogger()
	st, err := sterr.CreateErrorPreconditionFailure(
		msg,
		[]*errdetails.PreconditionFailure_Violation{
			{
				Type:        tp,
				Subject:     sub,
				Description: desc,
			},
		})
	if err != nil {
		logger.Error(err.Error())
	}
	return st
}

// CreateErrorResourceInfo is a wrapper for the CreateErrorResourceInfo in x/sterr
func CreateErrorResourceInfo(code codes.Code, msg string, rscType string, rscName string, owner string, desc string) *status.Status {
	logger, _ := logger.GetZapLogger()
	st, err := sterr.CreateErrorResourceInfo(code, msg, rscType, rscName, owner, desc)
	if err != nil {
		logger.Error(err.Error())
	}
	return st
}
