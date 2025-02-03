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

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/x/checkfield"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	customotel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

func (h *PublicHandler) CreateUserSecret(ctx context.Context, req *pb.CreateUserSecretRequest) (resp *pb.CreateUserSecretResponse, err error) {
	r, err := h.CreateNamespaceSecret(ctx, &pb.CreateNamespaceSecretRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Secret:      req.Secret,
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateUserSecretResponse{Secret: r.Secret}, nil
}

func (h *PublicHandler) CreateOrganizationSecret(ctx context.Context, req *pb.CreateOrganizationSecretRequest) (resp *pb.CreateOrganizationSecretResponse, err error) {
	r, err := h.CreateNamespaceSecret(ctx, &pb.CreateNamespaceSecretRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Secret:      req.Secret,
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateOrganizationSecretResponse{Secret: r.Secret}, nil
}

func (h *PublicHandler) CreateNamespaceSecret(ctx context.Context, req *pb.CreateNamespaceSecretRequest) (resp *pb.CreateNamespaceSecretResponse, err error) {

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

	ns, err := h.service.GetNamespaceByID(ctx, req.GetNamespaceId())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	secretToCreate := req.GetSecret()

	secret, err := h.service.CreateNamespaceSecret(ctx, ns, secretToCreate)

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

	return &pb.CreateNamespaceSecretResponse{Secret: secret}, nil
}

func (h *PublicHandler) ListUserSecrets(ctx context.Context, req *pb.ListUserSecretsRequest) (resp *pb.ListUserSecretsResponse, err error) {
	r, err := h.ListNamespaceSecrets(ctx, &pb.ListNamespaceSecretsRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ListUserSecretsResponse{Secrets: r.Secrets, NextPageToken: r.NextPageToken, TotalSize: r.TotalSize}, nil
}

func (h *PublicHandler) ListOrganizationSecrets(ctx context.Context, req *pb.ListOrganizationSecretsRequest) (resp *pb.ListOrganizationSecretsResponse, err error) {
	r, err := h.ListNamespaceSecrets(ctx, &pb.ListNamespaceSecretsRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ListOrganizationSecretsResponse{Secrets: r.Secrets, NextPageToken: r.NextPageToken, TotalSize: r.TotalSize}, nil
}

func (h *PublicHandler) ListNamespaceSecrets(ctx context.Context, req *pb.ListNamespaceSecretsRequest) (resp *pb.ListNamespaceSecretsResponse, err error) {

	eventName := "ListNamespaceSecrets"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbSecrets, totalSize, nextPageToken, err := h.service.ListNamespaceSecrets(ctx, ns, req.GetPageSize(), req.GetPageToken(), filtering.Filter{})
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

	return &pb.ListNamespaceSecretsResponse{
		Secrets:       pbSecrets,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}, nil
}

func (h *PublicHandler) GetUserSecret(ctx context.Context, req *pb.GetUserSecretRequest) (resp *pb.GetUserSecretResponse, err error) {
	splits := strings.Split(req.Name, "/")
	r, err := h.GetNamespaceSecret(ctx, &pb.GetNamespaceSecretRequest{NamespaceId: splits[1], SecretId: splits[3]})
	if err != nil {
		return nil, err
	}
	return &pb.GetUserSecretResponse{Secret: r.Secret}, nil
}

func (h *PublicHandler) GetOrganizationSecret(ctx context.Context, req *pb.GetOrganizationSecretRequest) (resp *pb.GetOrganizationSecretResponse, err error) {
	splits := strings.Split(req.Name, "/")
	r, err := h.GetNamespaceSecret(ctx, &pb.GetNamespaceSecretRequest{NamespaceId: splits[1], SecretId: splits[3]})
	if err != nil {
		return nil, err
	}
	return &pb.GetOrganizationSecretResponse{Secret: r.Secret}, nil
}

func (h *PublicHandler) GetNamespaceSecret(ctx context.Context, req *pb.GetNamespaceSecretRequest) (*pb.GetNamespaceSecretResponse, error) {

	eventName := "GetNamespaceSecret"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbSecret, err := h.service.GetNamespaceSecretByID(ctx, ns, req.SecretId)

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

	return &pb.GetNamespaceSecretResponse{Secret: pbSecret}, nil
}

func (h *PublicHandler) UpdateUserSecret(ctx context.Context, req *pb.UpdateUserSecretRequest) (resp *pb.UpdateUserSecretResponse, err error) {
	splits := strings.Split(req.Secret.Name, "/")
	r, err := h.UpdateNamespaceSecret(ctx, &pb.UpdateNamespaceSecretRequest{
		NamespaceId: splits[1],
		SecretId:    splits[3],
		Secret:      req.Secret,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateUserSecretResponse{Secret: r.Secret}, nil
}

func (h *PublicHandler) UpdateOrganizationSecret(ctx context.Context, req *pb.UpdateOrganizationSecretRequest) (resp *pb.UpdateOrganizationSecretResponse, err error) {
	splits := strings.Split(req.Secret.Name, "/")
	r, err := h.UpdateNamespaceSecret(ctx, &pb.UpdateNamespaceSecretRequest{
		NamespaceId: splits[1],
		SecretId:    splits[3],
		Secret:      req.Secret,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateOrganizationSecretResponse{Secret: r.Secret}, nil
}

func (h *PublicHandler) UpdateNamespaceSecret(ctx context.Context, req *pb.UpdateNamespaceSecretRequest) (*pb.UpdateNamespaceSecretResponse, error) {

	eventName := "UpdateNamespaceSecret"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbSecretReq := req.GetSecret()
	if pbSecretReq.Id == "" {
		pbSecretReq.Id = req.SecretId
	}
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

	getResp, err := h.GetNamespaceSecret(ctx, &pb.GetNamespaceSecretRequest{NamespaceId: req.NamespaceId, SecretId: req.SecretId})
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
		return &pb.UpdateNamespaceSecretResponse{Secret: getResp.GetSecret()}, nil
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

	pbSecret, err := h.service.UpdateNamespaceSecretByID(ctx, ns, req.SecretId, pbSecretToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	return &pb.UpdateNamespaceSecretResponse{Secret: pbSecret}, nil
}

func (h *PublicHandler) DeleteUserSecret(ctx context.Context, req *pb.DeleteUserSecretRequest) (resp *pb.DeleteUserSecretResponse, err error) {
	splits := strings.Split(req.Name, "/")
	_, err = h.DeleteNamespaceSecret(ctx, &pb.DeleteNamespaceSecretRequest{NamespaceId: splits[1], SecretId: splits[3]})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteUserSecretResponse{}, nil
}

func (h *PublicHandler) DeleteOrganizationSecret(ctx context.Context, req *pb.DeleteOrganizationSecretRequest) (resp *pb.DeleteOrganizationSecretResponse, err error) {
	splits := strings.Split(req.Name, "/")
	_, err = h.DeleteNamespaceSecret(ctx, &pb.DeleteNamespaceSecretRequest{NamespaceId: splits[1], SecretId: splits[3]})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteOrganizationSecretResponse{}, nil
}

func (h *PublicHandler) DeleteNamespaceSecret(ctx context.Context, req *pb.DeleteNamespaceSecretRequest) (*pb.DeleteNamespaceSecretResponse, error) {

	eventName := "DeleteNamespaceSecret"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	_, err = h.GetNamespaceSecret(ctx, &pb.GetNamespaceSecretRequest{NamespaceId: req.NamespaceId, SecretId: req.SecretId})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := h.service.DeleteNamespaceSecretByID(ctx, ns, req.SecretId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return &pb.DeleteNamespaceSecretResponse{}, nil
}
