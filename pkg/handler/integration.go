package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

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

// CreateNamespaceConnection creates a connection under the ownership of
// a namespace.
func (h *PublicHandler) CreateNamespaceConnection(ctx context.Context, req *pipelinepb.CreateNamespaceConnectionRequest) (*pipelinepb.CreateNamespaceConnectionResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	conn, err := h.service.CreateNamespaceConnection(ctx, req)
	if err != nil {
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for
	// REST endpoint.
	err = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated)))
	if err != nil {
		return nil, err
	}

	return &pipelinepb.CreateNamespaceConnectionResponse{Connection: conn}, nil
}

// UpdateNamespaceConnection updates a connection with the supplied connection
// fields.
func (h *PublicHandler) UpdateNamespaceConnection(ctx context.Context, req *pipelinepb.UpdateNamespaceConnectionRequest) (*pipelinepb.UpdateNamespaceConnectionResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	conn, err := h.service.UpdateNamespaceConnection(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.UpdateNamespaceConnectionResponse{Connection: conn}, nil
}

// DeleteNamespaceConnection deletes a connection.
func (h *PublicHandler) DeleteNamespaceConnection(ctx context.Context, req *pipelinepb.DeleteNamespaceConnectionRequest) (*pipelinepb.DeleteNamespaceConnectionResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	err := h.service.DeleteNamespaceConnection(ctx, req.GetNamespaceId(), req.GetConnectionId())
	if err != nil {
		return nil, err
	}

	// Manually set the custom header to have a StatusNoContent http response for
	// REST endpoint.
	err = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent)))
	if err != nil {
		return nil, err
	}

	return &pipelinepb.DeleteNamespaceConnectionResponse{}, nil
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

// TestNamespaceConnection makes a request to the 3rd party app that the
// connection is configured to communicate with, and checks the result of the
// call.
func (h *PublicHandler) TestNamespaceConnection(ctx context.Context, req *pipelinepb.TestNamespaceConnectionRequest) (*pipelinepb.TestNamespaceConnectionResponse, error) {
	return nil, nil //status.Errorf(codes.Unimplemented, "not implemented")
}

// GetNamespaceConnection fetches the details of a namespace connection.
func (h *PublicHandler) GetNamespaceConnection(ctx context.Context, req *pipelinepb.GetNamespaceConnectionRequest) (*pipelinepb.GetNamespaceConnectionResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	conn, err := h.service.GetNamespaceConnection(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetNamespaceConnectionResponse{Connection: conn}, nil
}

// ListNamespaceConnections returns a paginated list of connections created by
// a namespace.
func (h *PublicHandler) ListNamespaceConnections(ctx context.Context, req *pipelinepb.ListNamespaceConnectionsRequest) (*pipelinepb.ListNamespaceConnectionsResponse, error) {

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	resp, err := h.service.ListNamespaceConnections(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// LookUpConnectionAdmin fetches a connection by UID.
func (h *PrivateHandler) LookUpConnectionAdmin(ctx context.Context, req *pipelinepb.LookUpConnectionAdminRequest) (*pipelinepb.LookUpConnectionAdminResponse, error) {
	view := pipelinepb.View_VIEW_BASIC
	if req.GetView() != pipelinepb.View_VIEW_UNSPECIFIED {
		view = req.GetView()
	}

	uid := uuid.FromStringOrNil(req.GetUid())
	conn, err := h.service.GetConnectionByUIDAdmin(ctx, uid, view)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.LookUpConnectionAdminResponse{Connection: conn}, nil
}
