package middleware

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/handler"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/errmsg"
)

// RecoveryInterceptorOpt - panic handler
func RecoveryInterceptorOpt() grpc_recovery.Option {
	return grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
		return status.Errorf(codes.Unknown, "panic triggered: %v", p)
	})
}

// UnaryAppendMetadataInterceptor - append metadatas for unary
func UnaryAppendMetadataInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "can not extract metadata")
	}

	newCtx := metadata.NewIncomingContext(ctx, md)
	h, err := handler(newCtx, req)

	return h, AsGRPCError(err)
}

// StreamAppendMetadataInterceptor - append metadatas for stream
func StreamAppendMetadataInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Error(codes.Internal, "can not extract metadata")
	}

	newCtx := metadata.NewIncomingContext(stream.Context(), md)
	wrapped := grpc_middleware.WrapServerStream(stream)
	wrapped.WrappedContext = newCtx

	err := handler(srv, wrapped)

	return err
}

// AsGRPCError sets the gRPC status and error message according to the error
// type and metadata.
func AsGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// If it's already a status, respect the code.
	// If, additionally, an end-user message has been set for the error, it has
	// has priority over the status message.
	if st, ok := status.FromError(err); ok {
		if msg := errmsg.Message(err); msg != "" {
			// This conversion is used to preserve the status details.
			p := st.Proto()
			p.Message = msg
			st = status.FromProto(p)
		}

		return st.Err()
	}

	var code codes.Code
	switch {
	case
		errors.Is(err, gorm.ErrDuplicatedKey),
		errors.Is(err, repository.ErrNameExists):

		code = codes.AlreadyExists
	case
		errors.Is(err, gorm.ErrRecordNotFound),
		errors.Is(err, repository.ErrNoDataDeleted),
		errors.Is(err, repository.ErrNoDataUpdated),
		errors.Is(err, service.ErrNotFound),
		errors.Is(err, acl.ErrMembershipNotFound):

		code = codes.NotFound
	case
		errors.Is(err, repository.ErrOwnerTypeNotMatch),
		errors.Is(err, repository.ErrPageTokenDecode),
		errors.Is(err, bcrypt.ErrMismatchedHashAndPassword),
		errors.Is(err, handler.ErrCheckUpdateImmutableFields),
		errors.Is(err, handler.ErrCheckOutputOnlyFields),
		errors.Is(err, handler.ErrCheckRequiredFields),
		errors.Is(err, service.ErrExceedMaxBatchSize),
		errors.Is(err, service.ErrCanNotUsePlaintextSecret),
		errors.Is(err, handler.ErrFieldMask),
		errors.Is(err, handler.ErrResourceID),
		errors.Is(err, handler.ErrSematicVersion),
		errors.Is(err, handler.ErrUpdateMask):

		code = codes.InvalidArgument
	case
		errors.Is(err, service.ErrNoPermission),
		errors.Is(err, service.ErrCanNotTriggerNonLatestPipelineRelease):

		code = codes.PermissionDenied
	case
		errors.Is(err, service.ErrUnauthenticated):

		code = codes.Unauthenticated

	case
		errors.Is(err, service.ErrRateLimiting):

		code = codes.ResourceExhausted
	default:
		code = codes.Unknown
	}

	return status.Error(code, errmsg.MessageOrErr(err))
}
