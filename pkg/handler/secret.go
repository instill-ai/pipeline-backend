package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/x/checkfield"

	customotel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type CreateNamespaceSecretRequestInterface interface {
	GetSecret() *pb.Secret
	GetParent() string
}

func (h *PublicHandler) CreateUserSecret(ctx context.Context, req *pb.CreateUserSecretRequest) (resp *pb.CreateUserSecretResponse, err error) {
	resp = &pb.CreateUserSecretResponse{}
	resp.Secret, err = h.createNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) CreateOrganizationSecret(ctx context.Context, req *pb.CreateOrganizationSecretRequest) (resp *pb.CreateOrganizationSecretResponse, err error) {
	resp = &pb.CreateOrganizationSecretResponse{}
	resp.Secret, err = h.createNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) createNamespaceSecret(ctx context.Context, req CreateNamespaceSecretRequestInterface) (secret *pb.Secret, err error) {

	eventName := "CreateNamespaceSecret"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload secret resource
	if err := checkfield.CheckRequiredFields(req.GetSecret(), append(createSecretRequiredFields, immutableSecretFields...)); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckRequiredFields
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload secret resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetSecret(), outputOnlySecretFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckOutputOnlyFields
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.GetSecret().GetId()); err != nil {
		span.SetStatus(1, err.Error())
		return nil, fmt.Errorf("%w: invalid pipeline ID: %w", errdomain.ErrInvalidArgument, err)
	}

	ns, _, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetParent())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	secretToCreate := req.GetSecret()

	secret, err = h.service.CreateNamespaceSecret(ctx, ns, secretToCreate)

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return secret, nil
}

type ListNamespaceSecretsRequestInterface interface {
	GetPageSize() int32
	GetPageToken() string
	GetParent() string
}

func (h *PublicHandler) ListUserSecrets(ctx context.Context, req *pb.ListUserSecretsRequest) (resp *pb.ListUserSecretsResponse, err error) {
	resp = &pb.ListUserSecretsResponse{}
	resp.Secrets, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceSecrets(ctx, req)
	return resp, err
}

func (h *PublicHandler) ListOrganizationSecrets(ctx context.Context, req *pb.ListOrganizationSecretsRequest) (resp *pb.ListOrganizationSecretsResponse, err error) {
	resp = &pb.ListOrganizationSecretsResponse{}
	resp.Secrets, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceSecrets(ctx, req)
	return resp, err
}

func (h *PublicHandler) listNamespaceSecrets(ctx context.Context, req ListNamespaceSecretsRequestInterface) (pbSecrets []*pb.Secret, nextPageToken string, totalSize int32, err error) {

	eventName := "ListNamespaceSecrets"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, _, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetParent())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	pbSecrets, totalSize, nextPageToken, err = h.service.ListNamespaceSecrets(ctx, ns, req.GetPageSize(), req.GetPageToken(), filtering.Filter{})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return pbSecrets, nextPageToken, int32(totalSize), nil
}

type GetNamespaceSecretRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) GetUserSecret(ctx context.Context, req *pb.GetUserSecretRequest) (resp *pb.GetUserSecretResponse, err error) {
	resp = &pb.GetUserSecretResponse{}
	resp.Secret, err = h.getNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) GetOrganizationSecret(ctx context.Context, req *pb.GetOrganizationSecretRequest) (resp *pb.GetOrganizationSecretResponse, err error) {
	resp = &pb.GetOrganizationSecretResponse{}
	resp.Secret, err = h.getNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) getNamespaceSecret(ctx context.Context, req GetNamespaceSecretRequestInterface) (*pb.Secret, error) {

	eventName := "GetNamespaceSecret"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbSecret, err := h.service.GetNamespaceSecretByID(ctx, ns, id)

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return pbSecret, nil
}

type UpdateNamespaceSecretRequestInterface interface {
	GetSecret() *pb.Secret
	GetUpdateMask() *fieldmaskpb.FieldMask
}

func (h *PublicHandler) UpdateUserSecret(ctx context.Context, req *pb.UpdateUserSecretRequest) (resp *pb.UpdateUserSecretResponse, err error) {
	resp = &pb.UpdateUserSecretResponse{}
	resp.Secret, err = h.updateNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) UpdateOrganizationSecret(ctx context.Context, req *pb.UpdateOrganizationSecretRequest) (resp *pb.UpdateOrganizationSecretResponse, err error) {
	resp = &pb.UpdateOrganizationSecretResponse{}
	resp.Secret, err = h.updateNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) updateNamespaceSecret(ctx context.Context, req UpdateNamespaceSecretRequestInterface) (*pb.Secret, error) {

	eventName := "UpdateNamespaceSecret"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetSecret().Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbSecretReq := req.GetSecret()
	pbUpdateMask := req.GetUpdateMask()

	// metadata field is type google.protobuf.Struct, which needs to be updated as a whole
	for idx, path := range pbUpdateMask.Paths {
		if strings.Contains(path, "metadata") {
			pbUpdateMask.Paths[idx] = "metadata"
		}
		if strings.Contains(path, "recipe") {
			pbUpdateMask.Paths[idx] = "recipe"
		}
	}
	// Validate the field mask
	if !pbUpdateMask.IsValid(pbSecretReq) {
		return nil, ErrUpdateMask
	}

	getResp, err := h.GetUserSecret(ctx, &pb.GetUserSecretRequest{Name: pbSecretReq.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlySecretFields)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckOutputOnlyFields
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrFieldMask
	}

	if mask.IsEmpty() {
		return getResp.GetSecret(), nil
	}

	pbSecretToUpdate := getResp.GetSecret()

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbSecretReq, pbSecretToUpdate, immutableSecretFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckUpdateImmutableFields
	}

	// Only the fields mentioned in the field mask will be copied to `pbSecretToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbSecretReq, pbSecretToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbSecret, err := h.service.UpdateNamespaceSecretByID(ctx, ns, id, pbSecretToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	return pbSecret, nil
}

type DeleteNamespaceSecretRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeleteUserSecret(ctx context.Context, req *pb.DeleteUserSecretRequest) (resp *pb.DeleteUserSecretResponse, err error) {
	resp = &pb.DeleteUserSecretResponse{}
	err = h.deleteNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) DeleteOrganizationSecret(ctx context.Context, req *pb.DeleteOrganizationSecretRequest) (resp *pb.DeleteOrganizationSecretResponse, err error) {
	resp = &pb.DeleteOrganizationSecretResponse{}
	err = h.deleteNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) deleteNamespaceSecret(ctx context.Context, req DeleteNamespaceSecretRequestInterface) error {

	eventName := "DeleteNamespaceSecret"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	_, err = h.GetUserSecret(ctx, &pb.GetUserSecretRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	if err := h.service.DeleteNamespaceSecretByID(ctx, ns, id); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return nil
}
