package acl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	openfga "github.com/openfga/go-sdk"
	openfgaClient "github.com/openfga/go-sdk/client"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
)

type ACLClient struct {
	writeClient          *openfgaClient.OpenFgaClient
	readClient           *openfgaClient.OpenFgaClient
	redisClient          *redis.Client
	authorizationModelID *string
}

type Relation struct {
	UID      uuid.UUID
	Relation string
}

type Mode string

const (
	ReadMode  Mode = "read"
	WriteMode Mode = "write"
)

func NewACLClient(wc *openfgaClient.OpenFgaClient, rc *openfgaClient.OpenFgaClient, redisClient *redis.Client, a *string) ACLClient {
	if rc == nil {
		rc = wc
	}

	return ACLClient{
		writeClient:          wc,
		readClient:           rc,
		redisClient:          redisClient,
		authorizationModelID: a,
	}
}

func (c *ACLClient) getClient(ctx context.Context, mode Mode) *openfgaClient.OpenFgaClient {
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	if mode == WriteMode {
		// To solve the read-after-write inconsistency problem,
		// we will direct the user to read from the primary database for a certain time frame
		// to ensure that the data is synchronized from the primary DB to the replica DB.
		_ = c.redisClient.Set(ctx, fmt.Sprintf("db_pin_user:%s", userUID), time.Now(), time.Duration(config.Config.OpenFGA.Replica.ReplicationTimeFrame)*time.Second)
	}

	// If the user is pinned, we will use the primary database for querying.
	if !errors.Is(c.redisClient.Get(ctx, fmt.Sprintf("db_pin_user:%s", userUID)).Err(), redis.Nil) {
		return c.writeClient
	}
	if mode == ReadMode {
		return c.readClient
	}
	return c.writeClient
}

func (c *ACLClient) SetOwner(ctx context.Context, objectType string, objectUID uuid.UUID, ownerType string, ownerUID uuid.UUID) error {
	var err error
	readOptions := openfgaClient.ClientReadOptions{}
	writeOptions := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	readBody := openfgaClient.ClientReadRequest{
		User:     openfga.PtrString(fmt.Sprintf("%s:%s", ownerType, ownerUID.String())),
		Relation: openfga.PtrString("owner"),
		Object:   openfga.PtrString(fmt.Sprintf("%s:%s", objectType, objectUID.String())),
	}

	data, err := c.getClient(ctx, ReadMode).Read(ctx).Body(readBody).Options(readOptions).Execute()
	if err != nil {
		return err
	}
	if len(*data.Tuples) > 0 {
		return nil
	}

	writeBody := openfgaClient.ClientWriteRequest{
		Writes: &[]openfgaClient.ClientTupleKey{
			{
				User:     fmt.Sprintf("%s:%s", ownerType, ownerUID.String()),
				Relation: "owner",
				Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
			}},
	}

	_, err = c.getClient(ctx, WriteMode).Write(ctx).Body(writeBody).Options(writeOptions).Execute()
	if err != nil {
		return err
	}
	return nil
}

func (c *ACLClient) SetPipelinePermissionMap(ctx context.Context, pipeline *datamodel.Pipeline) error {
	// TODO: use OpenFGA as single source of truth
	// TODO: support fine-grained permission settings

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

func (c *ACLClient) SetPipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string, role string, enable bool) error {
	var err error
	options := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	_ = c.DeletePipelinePermission(ctx, pipelineUID, user)

	if enable {
		body := openfgaClient.ClientWriteRequest{
			Writes: &[]openfgaClient.ClientTupleKey{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("pipeline:%s", pipelineUID.String()),
				}},
		}

		_, err = c.getClient(ctx, WriteMode).Write(ctx).Body(body).Options(options).Execute()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACLClient) DeletePipelinePermission(ctx context.Context, pipelineUID uuid.UUID, user string) error {
	// var err error
	options := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	for _, role := range []string{"admin", "writer", "executor", "reader"} {
		body := openfgaClient.ClientWriteRequest{
			Deletes: &[]openfgaClient.ClientTupleKey{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("pipeline:%s", pipelineUID.String()),
				}}}
		_, _ = c.getClient(ctx, WriteMode).Write(ctx).Body(body).Options(options).Execute()

	}

	return nil
}

func (c *ACLClient) Purge(ctx context.Context, objectType string, objectUID uuid.UUID) error {
	readOptions := openfgaClient.ClientReadOptions{}
	writeOptions := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	readBody := openfgaClient.ClientReadRequest{
		Object: openfga.PtrString(fmt.Sprintf("%s:%s", objectType, objectUID)),
	}
	resp, err := c.getClient(ctx, ReadMode).Read(ctx).Body(readBody).Options(readOptions).Execute()
	if err != nil {
		return err
	}
	for _, data := range *resp.Tuples {
		body := openfgaClient.ClientWriteRequest{
			Deletes: &[]openfgaClient.ClientTupleKey{
				{
					User:     *data.Key.User,
					Relation: *data.Key.Relation,
					Object:   *data.Key.Object,
				}}}
		_, err := c.getClient(ctx, WriteMode).Write(ctx).Body(body).Options(writeOptions).Execute()

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACLClient) CheckPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error) {

	options := openfgaClient.ClientCheckOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	userType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	userUID := ""
	if userType == "user" {
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	} else {
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
	}
	code := resource.GetRequestSingleHeader(ctx, constant.HeaderInstillCodeKey)

	body := openfgaClient.ClientCheckRequest{
		User:     fmt.Sprintf("%s:%s", userType, userUID),
		Relation: role,
		Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
	}
	data, err := c.getClient(ctx, ReadMode).Check(ctx).Body(body).Options(options).Execute()
	if err != nil {
		return false, err
	}
	if *data.Allowed {
		return *data.Allowed, nil
	}

	if code == "" {
		return false, nil
	}
	body = openfgaClient.ClientCheckRequest{
		User:     fmt.Sprintf("code:%s", code),
		Relation: role,
		Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
	}
	data, err = c.getClient(ctx, ReadMode).Check(ctx).Body(body).Options(options).Execute()

	if err != nil {
		return false, err
	}
	return *data.Allowed, nil
}

// TODO refactor
func (c *ACLClient) CheckPublicExecutable(ctx context.Context, objectType string, objectUID uuid.UUID) (bool, error) {

	options := openfgaClient.ClientCheckOptions{
		AuthorizationModelId: c.authorizationModelID,
	}
	body := openfgaClient.ClientCheckRequest{
		User:     "user:*",
		Relation: "executor",
		Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
	}
	data, err := c.getClient(ctx, ReadMode).Check(ctx).Body(body).Options(options).Execute()
	if err != nil {
		return false, err
	}
	if *data.Allowed {
		return *data.Allowed, nil
	}

	return *data.Allowed, nil
}

func (c *ACLClient) ListPermissions(ctx context.Context, objectType string, role string, isPublic bool) ([]uuid.UUID, error) {

	userType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	userUIDStr := ""
	if userType == "user" {
		userUIDStr = resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	} else {
		userUIDStr = resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
	}

	options := openfgaClient.ClientListObjectsOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	if isPublic {
		userUIDStr = "*"
	}

	body := openfgaClient.ClientListObjectsRequest{
		User:     fmt.Sprintf("%s:%s", userType, userUIDStr),
		Relation: role,
		Type:     objectType,
	}
	listObjectsResult, err := c.getClient(ctx, ReadMode).ListObjects(ctx).Body(body).Options(options).Execute()
	if err != nil {
		return nil, err
	}
	objectUIDs := []uuid.UUID{}
	for _, object := range listObjectsResult.GetObjects() {
		objectUIDs = append(objectUIDs, uuid.FromStringOrNil(strings.Split(object, ":")[1]))
	}

	return objectUIDs, nil
}
