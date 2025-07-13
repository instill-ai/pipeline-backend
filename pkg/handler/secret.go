package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/x/checkfield"

	errorsx "github.com/instill-ai/x/errors"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

// CreateUserSecret creates a user secret.
func (h *PublicHandler) CreateUserSecret(ctx context.Context, req *pipelinepb.CreateUserSecretRequest) (resp *pipelinepb.CreateUserSecretResponse, err error) {
	r, err := h.CreateNamespaceSecret(ctx, &pipelinepb.CreateNamespaceSecretRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Secret:      req.Secret,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.CreateUserSecretResponse{Secret: r.Secret}, nil
}

// CreateOrganizationSecret creates an organization secret.
func (h *PublicHandler) CreateOrganizationSecret(ctx context.Context, req *pipelinepb.CreateOrganizationSecretRequest) (resp *pipelinepb.CreateOrganizationSecretResponse, err error) {
	r, err := h.CreateNamespaceSecret(ctx, &pipelinepb.CreateNamespaceSecretRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Secret:      req.Secret,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.CreateOrganizationSecretResponse{Secret: r.Secret}, nil
}

// CreateNamespaceSecret creates a namespace secret.
func (h *PublicHandler) CreateNamespaceSecret(ctx context.Context, req *pipelinepb.CreateNamespaceSecretRequest) (resp *pipelinepb.CreateNamespaceSecretResponse, err error) {

	// Return error if REQUIRED fields are not provided in the requested payload secret resource
	if err := checkfield.CheckRequiredFields(req.GetSecret(), append(createSecretRequiredFields, immutableSecretFields...)); err != nil {
		return nil, errorsx.ErrCheckRequiredFields
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload secret resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetSecret(), outputOnlySecretFields); err != nil {
		return nil, errorsx.ErrCheckOutputOnlyFields
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.GetSecret().GetId()); err != nil {
		return nil, fmt.Errorf("%w: invalid pipeline ID: %w", errorsx.ErrInvalidArgument, err)
	}

	ns, err := h.service.GetNamespaceByID(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	secretToCreate := req.GetSecret()
	secret, err := h.service.CreateNamespaceSecret(ctx, ns, secretToCreate)

	if err != nil {
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, err
	}

	return &pipelinepb.CreateNamespaceSecretResponse{Secret: secret}, nil
}

// ListUserSecrets lists user secrets.
func (h *PublicHandler) ListUserSecrets(ctx context.Context, req *pipelinepb.ListUserSecretsRequest) (resp *pipelinepb.ListUserSecretsResponse, err error) {
	r, err := h.ListNamespaceSecrets(ctx, &pipelinepb.ListNamespaceSecretsRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.ListUserSecretsResponse{Secrets: r.Secrets, NextPageToken: r.NextPageToken, TotalSize: r.TotalSize}, nil
}

// ListOrganizationSecrets lists organization secrets.
func (h *PublicHandler) ListOrganizationSecrets(ctx context.Context, req *pipelinepb.ListOrganizationSecretsRequest) (resp *pipelinepb.ListOrganizationSecretsResponse, err error) {
	r, err := h.ListNamespaceSecrets(ctx, &pipelinepb.ListNamespaceSecretsRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.ListOrganizationSecretsResponse{Secrets: r.Secrets, NextPageToken: r.NextPageToken, TotalSize: r.TotalSize}, nil
}

// ListNamespaceSecrets lists namespace secrets.
func (h *PublicHandler) ListNamespaceSecrets(ctx context.Context, req *pipelinepb.ListNamespaceSecretsRequest) (resp *pipelinepb.ListNamespaceSecretsResponse, err error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}

	if err := authenticateUser(ctx, true); err != nil {
		return nil, err
	}

	pbSecrets, totalSize, nextPageToken, err := h.service.ListNamespaceSecrets(ctx, ns, req.GetPageSize(), req.GetPageToken(), filtering.Filter{})
	if err != nil {
		return nil, err
	}

	return &pipelinepb.ListNamespaceSecretsResponse{
		Secrets:       pbSecrets,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}, nil
}

// GetUserSecret gets a user secret.
func (h *PublicHandler) GetUserSecret(ctx context.Context, req *pipelinepb.GetUserSecretRequest) (resp *pipelinepb.GetUserSecretResponse, err error) {
	splits := strings.Split(req.Name, "/")
	r, err := h.GetNamespaceSecret(ctx, &pipelinepb.GetNamespaceSecretRequest{NamespaceId: splits[1], SecretId: splits[3]})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.GetUserSecretResponse{Secret: r.Secret}, nil
}

// GetOrganizationSecret gets an organization secret.
func (h *PublicHandler) GetOrganizationSecret(ctx context.Context, req *pipelinepb.GetOrganizationSecretRequest) (resp *pipelinepb.GetOrganizationSecretResponse, err error) {
	splits := strings.Split(req.Name, "/")
	r, err := h.GetNamespaceSecret(ctx, &pipelinepb.GetNamespaceSecretRequest{NamespaceId: splits[1], SecretId: splits[3]})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.GetOrganizationSecretResponse{Secret: r.Secret}, nil
}

// GetNamespaceSecret gets a namespace secret.
func (h *PublicHandler) GetNamespaceSecret(ctx context.Context, req *pipelinepb.GetNamespaceSecretRequest) (*pipelinepb.GetNamespaceSecretResponse, error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		return nil, err
	}

	pbSecret, err := h.service.GetNamespaceSecretByID(ctx, ns, req.SecretId)

	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetNamespaceSecretResponse{Secret: pbSecret}, nil
}

// UpdateUserSecret updates a user secret.
func (h *PublicHandler) UpdateUserSecret(ctx context.Context, req *pipelinepb.UpdateUserSecretRequest) (resp *pipelinepb.UpdateUserSecretResponse, err error) {
	splits := strings.Split(req.Secret.Name, "/")
	r, err := h.UpdateNamespaceSecret(ctx, &pipelinepb.UpdateNamespaceSecretRequest{
		NamespaceId: splits[1],
		SecretId:    splits[3],
		Secret:      req.Secret,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.UpdateUserSecretResponse{Secret: r.Secret}, nil
}

// UpdateOrganizationSecret updates an organization secret.
func (h *PublicHandler) UpdateOrganizationSecret(ctx context.Context, req *pipelinepb.UpdateOrganizationSecretRequest) (resp *pipelinepb.UpdateOrganizationSecretResponse, err error) {
	splits := strings.Split(req.Secret.Name, "/")
	r, err := h.UpdateNamespaceSecret(ctx, &pipelinepb.UpdateNamespaceSecretRequest{
		NamespaceId: splits[1],
		SecretId:    splits[3],
		Secret:      req.Secret,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.UpdateOrganizationSecretResponse{Secret: r.Secret}, nil
}

// UpdateNamespaceSecret updates a namespace secret.
func (h *PublicHandler) UpdateNamespaceSecret(ctx context.Context, req *pipelinepb.UpdateNamespaceSecretRequest) (*pipelinepb.UpdateNamespaceSecretResponse, error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
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
		return nil, errorsx.ErrUpdateMask
	}

	getResp, err := h.GetNamespaceSecret(ctx, &pipelinepb.GetNamespaceSecretRequest{NamespaceId: req.NamespaceId, SecretId: req.SecretId})
	if err != nil {
		return nil, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlySecretFields)
	if err != nil {
		return nil, errorsx.ErrCheckOutputOnlyFields
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		return nil, errorsx.ErrFieldMask
	}

	if mask.IsEmpty() {
		return &pipelinepb.UpdateNamespaceSecretResponse{Secret: getResp.GetSecret()}, nil
	}

	pbSecretToUpdate := getResp.GetSecret()

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbSecretReq, pbSecretToUpdate, immutableSecretFields); err != nil {
		return nil, errorsx.ErrCheckUpdateImmutableFields
	}

	// Only the fields mentioned in the field mask will be copied to `pbSecretToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbSecretReq, pbSecretToUpdate)
	if err != nil {
		return nil, err
	}

	pbSecret, err := h.service.UpdateNamespaceSecretByID(ctx, ns, req.SecretId, pbSecretToUpdate)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.UpdateNamespaceSecretResponse{Secret: pbSecret}, nil
}

// DeleteUserSecret deletes a user secret.
func (h *PublicHandler) DeleteUserSecret(ctx context.Context, req *pipelinepb.DeleteUserSecretRequest) (resp *pipelinepb.DeleteUserSecretResponse, err error) {
	splits := strings.Split(req.Name, "/")
	_, err = h.DeleteNamespaceSecret(ctx, &pipelinepb.DeleteNamespaceSecretRequest{NamespaceId: splits[1], SecretId: splits[3]})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.DeleteUserSecretResponse{}, nil
}

// DeleteOrganizationSecret deletes an organization secret.
func (h *PublicHandler) DeleteOrganizationSecret(ctx context.Context, req *pipelinepb.DeleteOrganizationSecretRequest) (resp *pipelinepb.DeleteOrganizationSecretResponse, err error) {
	splits := strings.Split(req.Name, "/")
	_, err = h.DeleteNamespaceSecret(ctx, &pipelinepb.DeleteNamespaceSecretRequest{NamespaceId: splits[1], SecretId: splits[3]})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.DeleteOrganizationSecretResponse{}, nil
}

// DeleteNamespaceSecret deletes a namespace secret.
func (h *PublicHandler) DeleteNamespaceSecret(ctx context.Context, req *pipelinepb.DeleteNamespaceSecretRequest) (*pipelinepb.DeleteNamespaceSecretResponse, error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}
	_, err = h.GetNamespaceSecret(ctx, &pipelinepb.GetNamespaceSecretRequest{NamespaceId: req.NamespaceId, SecretId: req.SecretId})
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteNamespaceSecretByID(ctx, ns, req.SecretId); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return nil, err
	}

	return &pipelinepb.DeleteNamespaceSecretResponse{}, nil
}
