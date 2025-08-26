package acl

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	openfgaclient "github.com/openfga/go-sdk/client"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	openfgax "github.com/instill-ai/x/openfga"
)

// Pipeline-specific object types
const (
	ObjectTypePipeline     openfgax.ObjectType = "pipeline"
	ObjectTypeOrganization openfgax.ObjectType = "organization"
)

// ACLClient wraps the x/openfga Client with pipeline-backend specific operations
type aclClient struct {
	openfgax.Client
}

// ACLClientInterface defines the interface for pipeline ACL operations
type ACLClient interface {
	openfgax.Client

	CheckPublicExecutable(ctx context.Context, objectType openfgax.ObjectType, objectUID uuid.UUID) (bool, error)
	DeletePipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string) error
	SetPipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string, role string, enable bool) error
	SetPipelinePermissionMap(ctx context.Context, pipeline *datamodel.Pipeline) error
}

// NewFGAClient creates a new pipeline-backend specific FGA client
func NewFGAClient(client openfgax.Client) ACLClient {
	return &aclClient{Client: client}
}

// CheckPublicExecutable checks if public users can execute an object
func (c *aclClient) CheckPublicExecutable(ctx context.Context, objectType openfgax.ObjectType, objectUID uuid.UUID) (bool, error) {
	body := openfgaclient.ClientCheckRequest{
		User:     fmt.Sprintf("%s:*", openfgax.OwnerTypeUser),
		Relation: "executor",
		Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
	}
	data, err := c.SDKClient().Check(ctx).Body(body).Execute()
	if err != nil {
		return false, err
	}
	return *data.Allowed, nil
}

// DeletePipelinePermission deletes all permissions for a user on a pipeline
func (c *aclClient) DeletePipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string) error {
	// Delete all possible roles
	for _, role := range []string{"admin", "writer", "executor", "reader"} {
		body := openfgaclient.ClientWriteRequest{
			Deletes: []openfgaclient.ClientTupleKeyWithoutCondition{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("%s:%s", ObjectTypePipeline, pipelineUID.String()),
				},
			},
		}
		_, _ = c.SDKClient().Write(ctx).Body(body).Execute()
	}
	return nil
}

// SetPipelinePermission sets a specific permission for a user on a pipeline
func (c *aclClient) SetPipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string, role string, enable bool) error {
	// First delete existing permission for this user
	_ = c.DeletePipelinePermission(ctx, pipelineUID, user)

	if enable {
		// Write the new permission
		body := openfgaclient.ClientWriteRequest{
			Writes: []openfgaclient.ClientTupleKey{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("%s:%s", ObjectTypePipeline, pipelineUID.String()),
				},
			},
		}
		_, err := c.SDKClient().Write(ctx).Body(body).Execute()
		return err
	}

	return nil
}

// SetPipelinePermissionMap sets permissions based on pipeline configuration
func (c *aclClient) SetPipelinePermissionMap(ctx context.Context, pipeline *datamodel.Pipeline) error {
	if pipeline.Owner == "" {
		return nil // No owner to set
	}

	// Parse owner string to determine type and UUID
	// Owner format is typically "users/{uuid}" or "organizations/{uuid}"
	ownerParts := strings.Split(pipeline.Owner, "/")
	if len(ownerParts) != 2 {
		return fmt.Errorf("invalid owner format: %s", pipeline.Owner)
	}

	var ownerType openfgax.OwnerType

	switch ownerParts[0] {
	case "users":
		ownerType = openfgax.OwnerTypeUser
	case "organizations":
		ownerType = openfgax.OwnerTypeOrganization
	default:
		return fmt.Errorf("invalid owner type: %s", ownerParts[0])
	}

	ownerUID := uuid.FromStringOrNil(ownerParts[1])
	if ownerUID.IsNil() {
		return fmt.Errorf("invalid owner UUID: %s", ownerParts[1])
	}

	// Use embedded SetOwner method
	return c.SetOwner(ctx, ObjectTypePipeline, pipeline.UID, ownerType, ownerUID)
}
