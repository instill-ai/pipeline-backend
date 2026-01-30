package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pipelinepb "github.com/instill-ai/protogen-go/pipeline/v1beta"
)

// parseConnectionFromName extracts namespace ID and connection ID from name string.
// Format: namespaces/{namespace}/connections/{connection}
func parseConnectionFromName(name string) (namespaceID, connectionID string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) < 4 || parts[0] != "namespaces" || parts[2] != "connections" {
		return "", "", fmt.Errorf("invalid connection name format: %s", name)
	}
	return parts[1], parts[3], nil
}

// GetIntegration returns the details of an integration.
func (h *PublicHandler) GetIntegration(ctx context.Context, req *pipelinepb.GetIntegrationRequest) (*pipelinepb.GetIntegrationResponse, error) {

	view := req.GetView()
	if view == pipelinepb.View_VIEW_UNSPECIFIED {
		view = pipelinepb.View_VIEW_BASIC
	}

	integration, err := h.service.GetIntegration(ctx, req.GetIntegrationId(), view)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetIntegrationResponse{Integration: integration}, nil
}

// ListIntegrations returns a paginated list of available integrations.
func (h *PublicHandler) ListIntegrations(ctx context.Context, req *pipelinepb.ListIntegrationsRequest) (*pipelinepb.ListIntegrationsResponse, error) {

	resp, err := h.service.ListIntegrations(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CreateConnection creates a connection under the ownership of a namespace.
func (h *PublicHandler) CreateConnection(ctx context.Context, req *pipelinepb.CreateConnectionRequest) (*pipelinepb.CreateConnectionResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	conn, err := h.service.CreateConnection(ctx, req)
	if err != nil {
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for
	// REST endpoint.
	err = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated)))
	if err != nil {
		return nil, err
	}

	return &pipelinepb.CreateConnectionResponse{Connection: conn}, nil
}

// UpdateConnection updates a connection with the supplied connection fields.
func (h *PublicHandler) UpdateConnection(ctx context.Context, req *pipelinepb.UpdateConnectionRequest) (*pipelinepb.UpdateConnectionResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	conn, err := h.service.UpdateConnection(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.UpdateConnectionResponse{Connection: conn}, nil
}

// DeleteConnection deletes a connection.
func (h *PublicHandler) DeleteConnection(ctx context.Context, req *pipelinepb.DeleteConnectionRequest) (*pipelinepb.DeleteConnectionResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	// Parse namespace ID and connection ID from name: namespaces/{namespace}/connections/{connection}
	namespaceID, connectionID, err := parseConnectionFromName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = h.service.DeleteConnection(ctx, namespaceID, connectionID)
	if err != nil {
		return nil, err
	}

	// Manually set the custom header to have a StatusNoContent http response for
	// REST endpoint.
	err = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent)))
	if err != nil {
		return nil, err
	}

	return &pipelinepb.DeleteConnectionResponse{}, nil
}

// ListPipelineIDsByConnectionID returns a paginated list with the IDs of the
// pipelines that reference a given connection. All the pipelines will belong
// to the same namespace as the connection.
func (h *PublicHandler) ListPipelineIDsByConnectionID(ctx context.Context, req *pipelinepb.ListPipelineIDsByConnectionIDRequest) (*pipelinepb.ListPipelineIDsByConnectionIDResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	resp, err := h.service.ListPipelineIDsByConnectionID(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// TestConnection makes a request to the 3rd party app that the connection is
// configured to communicate with, and checks the result of the call.
func (h *PublicHandler) TestConnection(ctx context.Context, req *pipelinepb.TestConnectionRequest) (*pipelinepb.TestConnectionResponse, error) {
	return nil, nil //status.Errorf(codes.Unimplemented, "not implemented")
}

// GetConnection fetches the details of a connection.
func (h *PublicHandler) GetConnection(ctx context.Context, req *pipelinepb.GetConnectionRequest) (*pipelinepb.GetConnectionResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	conn, err := h.service.GetConnection(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetConnectionResponse{Connection: conn}, nil
}

// ListConnections returns a paginated list of connections created by a namespace.
func (h *PublicHandler) ListConnections(ctx context.Context, req *pipelinepb.ListConnectionsRequest) (*pipelinepb.ListConnectionsResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	resp, err := h.service.ListConnections(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// LookUpConnectionAdmin fetches a connection by permalink.
// Permalink format: connections/{connection.uid}
func (h *PrivateHandler) LookUpConnectionAdmin(ctx context.Context, req *pipelinepb.LookUpConnectionAdminRequest) (*pipelinepb.LookUpConnectionAdminResponse, error) {
	view := pipelinepb.View_VIEW_BASIC
	if req.GetView() != pipelinepb.View_VIEW_UNSPECIFIED {
		view = req.GetView()
	}

	// Parse UID from permalink: connections/{connection.uid}
	permalink := req.GetPermalink()
	if !strings.HasPrefix(permalink, "connections/") {
		return nil, status.Errorf(codes.InvalidArgument, "invalid permalink format, expected connections/{uid}")
	}
	uidStr := strings.TrimPrefix(permalink, "connections/")
	uid := uuid.FromStringOrNil(uidStr)

	conn, err := h.service.GetConnectionByUIDAdmin(ctx, uid, view)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.LookUpConnectionAdminResponse{Connection: conn}, nil
}
