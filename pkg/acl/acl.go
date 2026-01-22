package acl

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	openfga "github.com/openfga/api/proto/openfga/v1"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/x/resource"

	aclx "github.com/instill-ai/x/acl"
)

// ACLClientInterface defines the interface for ACL operations.
type ACLClientInterface interface {
	CheckPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error)
	CheckPublicExecutable(ctx context.Context, objectType string, objectUID uuid.UUID) (bool, error)
	DeletePipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string) error
	ListPermissions(ctx context.Context, objectType string, role string, isPublic bool) ([]uuid.UUID, error)
	Purge(ctx context.Context, objectType string, objectUID uuid.UUID) error
	SetOwner(ctx context.Context, objectType string, objectUID uuid.UUID, ownerType string, ownerUID uuid.UUID) error
	SetPipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string, role string, enable bool) error
	SetPipelinePermissionMap(ctx context.Context, pipeline *datamodel.Pipeline) error
	CheckLinkPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error)
}

// ACLClient wraps the shared ACL client and adds pipeline-specific methods.
type ACLClient struct {
	*aclx.ACLClient
}

// Relation represents a permission relation.
type Relation struct {
	UID      uuid.UUID
	Relation string
}

type Mode string
type ObjectType string
type Role string

const (
	ReadMode  Mode = "read"
	WriteMode Mode = "write"

	Organization ObjectType = "organization"

	Member Role = "member"
	Admin  Role = "admin"
	Owner  Role = "owner"

	PipelineObject = "pipeline"
)

// NewACLClient creates a new ACL client using the shared library.
func NewACLClient(wc openfga.OpenFGAServiceClient, rc openfga.OpenFGAServiceClient, redisClient *redis.Client) ACLClient {
	cfg := aclx.Config{
		Host: config.Config.OpenFGA.Host,
		Port: config.Config.OpenFGA.Port,
		Replica: aclx.ReplicaConfig{
			Host:                 config.Config.OpenFGA.Replica.Host,
			Port:                 config.Config.OpenFGA.Replica.Port,
			ReplicationTimeFrame: config.Config.OpenFGA.Replica.ReplicationTimeFrame,
		},
		Cache: aclx.CacheConfig{
			Enabled: config.Config.OpenFGA.Cache.Enabled,
			TTL:     config.Config.OpenFGA.Cache.TTL,
		},
	}

	sharedClient := aclx.NewClient(wc, rc, redisClient, cfg)

	return ACLClient{
		ACLClient: sharedClient,
	}
}

// InitOpenFGAClient initializes gRPC connections to OpenFGA server.
func InitOpenFGAClient(ctx context.Context, host string, port int) (openfga.OpenFGAServiceClient, *grpc.ClientConn) {
	return aclx.InitOpenFGAClient(ctx, host, port, constant.MaxPayloadSize/(1024*1024))
}

// SetPipelinePermissionMap sets permissions on a pipeline based on its sharing settings.
func (c *ACLClient) SetPipelinePermissionMap(ctx context.Context, pipeline *datamodel.Pipeline) error {
	for user, perm := range pipeline.Sharing.Users {
		if user != "*/*" {
			return fmt.Errorf("only support users: `*/*`")
		}

		if perm.Role == "ROLE_VIEWER" || perm.Role == "ROLE_EXECUTOR" {
			for _, t := range []string{"user", "visitor"} {
				err := c.SetPipelinePermission(ctx, pipeline.UID, fmt.Sprintf("%s:*", t), "reader", perm.Enabled)
				if err != nil {
					return err
				}
			}
		}
		if perm.Role == "ROLE_EXECUTOR" {
			for _, t := range []string{"user"} {
				err := c.SetPipelinePermission(ctx, pipeline.UID, fmt.Sprintf("%s:*", t), "executor", perm.Enabled)
				if err != nil {
					return err
				}
			}
		}
	}

	if pipeline.Sharing.ShareCode != nil {
		if pipeline.Sharing.ShareCode.User != "*/*" {
			return fmt.Errorf("only support users: `*/*`")
		}
		if pipeline.Sharing.ShareCode.Role == "ROLE_VIEWER" {
			err := c.SetPipelinePermission(ctx, pipeline.UID, fmt.Sprintf("code:%s", pipeline.ShareCode), "reader", pipeline.Sharing.ShareCode.Enabled)
			if err != nil {
				return err
			}
		}
		if pipeline.Sharing.ShareCode.Role == "ROLE_EXECUTOR" {
			err := c.SetPipelinePermission(ctx, pipeline.UID, fmt.Sprintf("code:%s", pipeline.ShareCode), "executor", pipeline.Sharing.ShareCode.Enabled)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SetPipelinePermission sets a permission for a user on a pipeline.
func (c *ACLClient) SetPipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string, role string, enable bool) error {
	return c.SetResourcePermission(ctx, PipelineObject, pipelineUID, user, role, enable)
}

// DeletePipelinePermission deletes all permissions for a user on a pipeline.
func (c *ACLClient) DeletePipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string) error {
	return c.DeleteResourcePermission(ctx, PipelineObject, pipelineUID, user)
}

// CheckLinkPermission checks the access over a resource through a shareable link.
func (c *ACLClient) CheckLinkPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error) {
	return c.ACLClient.CheckLinkPermission(ctx, objectType, objectUID, role, constant.HeaderInstillCodeKey)
}

// CheckPermission returns the access of the context user over a resource.
// This overrides the shared client to add service-type check and link permission fallback.
func (c *ACLClient) CheckPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error) {
	// Check for internal service calls
	serviceType := resource.GetRequestSingleHeader(ctx, constant.HeaderServiceKey)
	if serviceType == "instill" {
		return true, nil
	}

	// Use shared client for standard permission check
	allowed, err := c.ACLClient.CheckPermission(ctx, objectType, objectUID, role)
	if err != nil {
		return false, err
	}

	if !allowed {
		// Fall back to link permission check
		return c.CheckLinkPermission(ctx, objectType, objectUID, role)
	}

	return true, nil
}

// ListPermissions lists all objects of a type that the current user has a role for.
func (c *ACLClient) ListPermissions(ctx context.Context, objectType string, role string, isPublic bool) ([]uuid.UUID, error) {
	return c.ACLClient.ListPermissions(ctx, objectType, role, isPublic)
}
