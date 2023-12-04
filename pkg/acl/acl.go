package acl

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	openfga "github.com/openfga/go-sdk"
	openfgaClient "github.com/openfga/go-sdk/client"
)

type ACLClient struct {
	client               *openfgaClient.OpenFgaClient
	authorizationModelId *string
}

type Relation struct {
	UID      uuid.UUID
	Relation string
}

func NewACLClient(c *openfgaClient.OpenFgaClient, a *string) ACLClient {
	return ACLClient{
		client:               c,
		authorizationModelId: a,
	}
}

func (c *ACLClient) SetOwner(objectType string, objectUID uuid.UUID, ownerType string, ownerUID uuid.UUID) error {
	var err error
	readOptions := openfgaClient.ClientReadOptions{}
	writeOptions := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelId,
	}

	readBody := openfgaClient.ClientReadRequest{
		User:     openfga.PtrString(fmt.Sprintf("%s:%s", ownerType, ownerUID.String())),
		Relation: openfga.PtrString("owner"),
		Object:   openfga.PtrString(fmt.Sprintf("%s:%s", objectType, objectUID.String())),
	}
	data, err := c.client.Read(context.Background()).Body(readBody).Options(readOptions).Execute()
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

	_, err = c.client.Write(context.Background()).Body(writeBody).Options(writeOptions).Execute()
	if err != nil {
		return err
	}
	return nil
}

func (c *ACLClient) SetPipelinePermissionMap(pipeline *datamodel.Pipeline) error {
	// TODO: use OpenFGA as single source of truth
	for user, perm := range pipeline.Permission.Users {
		userType := strings.Split(user, "/")[0]
		userType = userType[0 : len(userType)-1]
		userID := strings.Split(user, "/")[1]
		if userID != "*" {
			return fmt.Errorf("only support users/* and visitors/*")
		}

		if perm.Role == "ROLE_VIEWER" {
			err := c.SetPipelinePermission(pipeline.UID, fmt.Sprintf("%s:%s", userType, userID), "reader", perm.Enabled)
			if err != nil {
				return err
			}
		}
		if perm.Role == "ROLE_EXECUTOR" {
			err := c.SetPipelinePermission(pipeline.UID, fmt.Sprintf("%s:%s", userType, userID), "executor", perm.Enabled)
			if err != nil {
				return err
			}
		}
	}
	if pipeline.Permission.ShareCode != nil {
		if pipeline.Permission.ShareCode.Role == "ROLE_VIEWER" {
			err := c.SetPipelinePermission(pipeline.UID, fmt.Sprintf("code:%s", pipeline.Permission.ShareCode.Code), "reader", pipeline.Permission.ShareCode.Enabled)
			if err != nil {
				return err
			}
		}
		if pipeline.Permission.ShareCode.Role == "ROLE_EXECUTOR" {
			err := c.SetPipelinePermission(pipeline.UID, fmt.Sprintf("code:%s", pipeline.Permission.ShareCode.Code), "executor", pipeline.Permission.ShareCode.Enabled)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *ACLClient) SetPipelinePermission(pipelineUID uuid.UUID, user string, role string, enable bool) error {
	var err error
	options := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelId,
	}

	_ = c.DeletePipelinePermission(pipelineUID, user)

	if enable {
		body := openfgaClient.ClientWriteRequest{
			Writes: &[]openfgaClient.ClientTupleKey{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("pipeline:%s", pipelineUID.String()),
				}},
		}

		_, err = c.client.Write(context.Background()).Body(body).Options(options).Execute()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACLClient) DeletePipelinePermission(pipelineUID uuid.UUID, user string) error {
	// var err error
	options := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelId,
	}

	for _, role := range []string{"admin", "writer", "executor", "reader"} {
		body := openfgaClient.ClientWriteRequest{
			Deletes: &[]openfgaClient.ClientTupleKey{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("pipeline:%s", pipelineUID.String()),
				}}}
		_, _ = c.client.Write(context.Background()).Body(body).Options(options).Execute()

	}

	return nil
}

func (c *ACLClient) CheckPermission(objectType string, objectUID uuid.UUID, userType string, userUID uuid.UUID, code string, role string) (bool, error) {

	options := openfgaClient.ClientCheckOptions{
		AuthorizationModelId: c.authorizationModelId,
	}
	body := openfgaClient.ClientCheckRequest{
		User:     fmt.Sprintf("%s:%s", userType, userUID.String()),
		Relation: role,
		Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
	}
	data, err := c.client.Check(context.Background()).Body(body).Options(options).Execute()
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
	data, err = c.client.Check(context.Background()).Body(body).Options(options).Execute()

	if err != nil {
		return false, err
	}
	return *data.Allowed, nil
}
