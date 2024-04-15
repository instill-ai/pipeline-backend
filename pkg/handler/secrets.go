package handler

import (
	"context"
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

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/x/checkfield"

	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type CreateNamespaceSecretRequestInterface interface {
	GetSecret() *pipelinePB.Secret
}

func (h *PublicHandler) CreateUserSecret(ctx context.Context, req *pipelinePB.CreateUserSecretRequest) (resp *pipelinePB.CreateUserSecretResponse, err error) {
	ns, err := h.service.GetCtxUserNamespace(ctx)
	if err != nil {
		return nil, err
	}
	resp = &pipelinePB.CreateUserSecretResponse{}
	resp.Secret, err = h.createNamespaceSecret(ctx, ns, req)
	return resp, err
}

func (h *PublicHandler) CreateOrganizationSecret(ctx context.Context, req *pipelinePB.CreateOrganizationSecretRequest) (resp *pipelinePB.CreateOrganizationSecretResponse, err error) {

	ns, _, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetParent())
	if err != nil {
		return nil, err
	}

	resp = &pipelinePB.CreateOrganizationSecretResponse{}
	resp.Secret, err = h.createNamespaceSecret(ctx, ns, req)
	return resp, err
}

func (h *PublicHandler) createNamespaceSecret(ctx context.Context, ns resource.Namespace, req CreateNamespaceSecretRequestInterface) (secret *pipelinePB.Secret, err error) {

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
		return nil, ErrResourceID
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

	logger.Info(string(custom_otel.NewLogMessage(
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
}

func (h *PublicHandler) ListUserSecrets(ctx context.Context, req *pipelinePB.ListUserSecretsRequest) (resp *pipelinePB.ListUserSecretsResponse, err error) {
	ns, err := h.service.GetCtxUserNamespace(ctx)
	if err != nil {
		return nil, err
	}
	resp = &pipelinePB.ListUserSecretsResponse{}
	resp.Secrets, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceSecrets(ctx, ns, req)
	return resp, err
}

func (h *PublicHandler) ListOrganizationSecrets(ctx context.Context, req *pipelinePB.ListOrganizationSecretsRequest) (resp *pipelinePB.ListOrganizationSecretsResponse, err error) {
	ns, _, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetParent())
	if err != nil {
		return nil, err
	}
	resp = &pipelinePB.ListOrganizationSecretsResponse{}
	resp.Secrets, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceSecrets(ctx, ns, req)
	return resp, err
}

func (h *PublicHandler) listNamespaceSecrets(ctx context.Context, ns resource.Namespace, req ListNamespaceSecretsRequestInterface) (pbSecrets []*pipelinePB.Secret, nextPageToken string, totalSize int32, err error) {

	eventName := "ListNamespaceSecrets"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	pbSecrets, totalSize, nextPageToken, err = h.service.ListNamespaceSecrets(ctx, ns, req.GetPageSize(), req.GetPageToken(), filtering.Filter{})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
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

func (h *PublicHandler) GetUserSecret(ctx context.Context, req *pipelinePB.GetUserSecretRequest) (resp *pipelinePB.GetUserSecretResponse, err error) {
	resp = &pipelinePB.GetUserSecretResponse{}
	resp.Secret, err = h.getNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) GetOrganizationSecret(ctx context.Context, req *pipelinePB.GetOrganizationSecretRequest) (resp *pipelinePB.GetOrganizationSecretResponse, err error) {
	resp = &pipelinePB.GetOrganizationSecretResponse{}
	resp.Secret, err = h.getNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) getNamespaceSecret(ctx context.Context, req GetNamespaceSecretRequestInterface) (*pipelinePB.Secret, error) {

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

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return pbSecret, nil
}

type UpdateNamespaceSecretRequestInterface interface {
	GetSecret() *pipelinePB.Secret
	GetUpdateMask() *fieldmaskpb.FieldMask
}

func (h *PublicHandler) UpdateUserSecret(ctx context.Context, req *pipelinePB.UpdateUserSecretRequest) (resp *pipelinePB.UpdateUserSecretResponse, err error) {
	resp = &pipelinePB.UpdateUserSecretResponse{}
	resp.Secret, err = h.updateNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) UpdateOrganizationSecret(ctx context.Context, req *pipelinePB.UpdateOrganizationSecretRequest) (resp *pipelinePB.UpdateOrganizationSecretResponse, err error) {
	resp = &pipelinePB.UpdateOrganizationSecretResponse{}
	resp.Secret, err = h.updateNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) updateNamespaceSecret(ctx context.Context, req UpdateNamespaceSecretRequestInterface) (*pipelinePB.Secret, error) {

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

	getResp, err := h.GetUserSecret(ctx, &pipelinePB.GetUserSecretRequest{Name: pbSecretReq.GetName()})
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

func (h *PublicHandler) DeleteUserSecret(ctx context.Context, req *pipelinePB.DeleteUserSecretRequest) (resp *pipelinePB.DeleteUserSecretResponse, err error) {
	resp = &pipelinePB.DeleteUserSecretResponse{}
	err = h.deleteNamespaceSecret(ctx, req)
	return resp, err
}

func (h *PublicHandler) DeleteOrganizationSecret(ctx context.Context, req *pipelinePB.DeleteOrganizationSecretRequest) (resp *pipelinePB.DeleteOrganizationSecretResponse, err error) {
	resp = &pipelinePB.DeleteOrganizationSecretResponse{}
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
	_, err = h.GetUserSecret(ctx, &pipelinePB.GetUserSecretRequest{Name: req.GetName()})
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

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return nil
}
