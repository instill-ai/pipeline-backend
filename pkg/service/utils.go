package service

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomStrWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func generateShareCode() string {
	return randomStrWithCharset(32, charset)
}

// Note: Currently, we don't allow changing the owner ID. We are safe to use a cache with a longer TTL for this function.
func (s *service) convertOwnerPermalinkToName(ctx context.Context, permalink string) (string, error) {

	splits := strings.Split(permalink, "/")
	nsType := splits[0]
	uid := splits[1]
	key := fmt.Sprintf("user:%s:uid_to_id", uid)
	if id, err := s.redisClient.Get(ctx, key).Result(); err != redis.Nil {
		return fmt.Sprintf("%s/%s", nsType, id), nil
	}

	if nsType == "users" {
		userResp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		s.redisClient.Set(ctx, key, userResp.User.Id, 24*time.Hour)
		return fmt.Sprintf("users/%s", userResp.User.Id), nil
	} else {
		orgResp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(ctx, &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		s.redisClient.Set(ctx, key, orgResp.Organization.Id, 24*time.Hour)
		return fmt.Sprintf("organizations/%s", orgResp.Organization.Id), nil
	}
}

func (s *service) fetchOwnerByPermalink(ctx context.Context, permalink string) (*mgmtPB.Owner, error) {

	key := fmt.Sprintf("owner_profile:%s", permalink)
	if b, err := s.redisClient.Get(ctx, key).Bytes(); err == nil {
		owner := &mgmtPB.Owner{}
		if protojson.Unmarshal(b, owner) == nil {
			return owner, nil
		}
	}

	if strings.HasPrefix(permalink, "users") {
		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtPB.Owner{Owner: &mgmtPB.Owner_User{User: resp.User}}
		if b, err := protojson.Marshal(owner); err == nil {
			s.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil
	} else {
		resp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(ctx, &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtPB.Owner{Owner: &mgmtPB.Owner_Organization{Organization: resp.Organization}}
		if b, err := protojson.Marshal(owner); err == nil {
			s.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil

	}
}

// Note: Currently, we don't allow changing the owner ID. We are safe to use a cache with a longer TTL for this function.
func (s *service) convertOwnerNameToPermalink(ctx context.Context, name string) (string, error) {

	splits := strings.Split(name, "/")
	nsType := splits[0]
	id := splits[1]
	key := fmt.Sprintf("user:%s:id_to_uid", id)
	if uid, err := s.redisClient.Get(ctx, key).Result(); err != redis.Nil {
		return fmt.Sprintf("%s/%s", nsType, uid), nil
	}

	if nsType == "users" {
		userResp, err := s.mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("convertOwnerNameToPermalink error %w", err)
		}
		s.redisClient.Set(ctx, key, *userResp.User.Uid, 24*time.Hour)
		return fmt.Sprintf("users/%s", *userResp.User.Uid), nil
	} else {
		orgResp, err := s.mgmtPrivateServiceClient.GetOrganizationAdmin(ctx, &mgmtPB.GetOrganizationAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("convertOwnerNameToPermalink error %w", err)
		}
		s.redisClient.Set(ctx, key, orgResp.Organization.Uid, 24*time.Hour)
		return fmt.Sprintf("organizations/%s", orgResp.Organization.Uid), nil
	}
}

func (s *service) checkNamespacePermission(ctx context.Context, ns resource.Namespace) error {
	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		granted, err := s.aclClient.CheckPermission(ctx, "organization", ns.NsUID, "member")
		if err != nil {
			return err
		}
		if !granted {
			return ErrNoPermission
		}
	} else {
		if ns.NsUID != uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)) {
			return ErrNoPermission
		}
	}
	return nil
}

func (s *service) GetCtxUserNamespace(ctx context.Context) (resource.Namespace, error) {

	uid := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	name, err := s.convertOwnerPermalinkToName(ctx, fmt.Sprintf("users/%s", uid))
	if err != nil {
		return resource.Namespace{}, fmt.Errorf("namespace error")
	}
	// TODO: optimize the flow to get namespace
	return resource.Namespace{
		NsType: resource.NamespaceType("users"),
		NsID:   strings.Split(name, "/")[1],
		NsUID:  uid,
	}, nil
}
func (s *service) GetRscNamespaceAndNameID(ctx context.Context, path string) (resource.Namespace, string, error) {

	if strings.HasPrefix(path, "user/") {

		uid := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
		splits := strings.Split(path, "/")

		name, err := s.convertOwnerPermalinkToName(ctx, fmt.Sprintf("users/%s", uid))
		if err != nil {
			return resource.Namespace{}, "", fmt.Errorf("namespace error")
		}

		return resource.Namespace{
			NsType: resource.NamespaceType("users"),
			NsID:   strings.Split(name, "/")[1],
			NsUID:  uid,
		}, splits[2], nil
	}

	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, "", fmt.Errorf("namespace error")
	}
	uidStr, err := s.convertOwnerNameToPermalink(ctx, fmt.Sprintf("%s/%s", splits[0], splits[1]))

	if err != nil {
		return resource.Namespace{}, "", fmt.Errorf("namespace error %w", err)
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsID:   splits[1],
			NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, "", nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsID:   splits[1],
		NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, splits[3], nil
}

func (s *service) GetRscNamespaceAndPermalinkUID(ctx context.Context, path string) (resource.Namespace, uuid.UUID, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, uuid.Nil, fmt.Errorf("namespace error")
	}
	uidStr, err := s.convertOwnerNameToPermalink(ctx, fmt.Sprintf("%s/%s", splits[0], splits[1]))
	if err != nil {
		return resource.Namespace{}, uuid.Nil, fmt.Errorf("namespace error")
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsID:   splits[1],
			NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, uuid.Nil, nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsID:   splits[1],
		NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, uuid.FromStringOrNil(splits[3]), nil
}

func (s *service) GetRscNamespaceAndNameIDAndReleaseID(ctx context.Context, path string) (resource.Namespace, string, string, error) {
	ns, pipelineID, err := s.GetRscNamespaceAndNameID(ctx, path)
	if err != nil {
		return ns, pipelineID, "", err
	}
	splits := strings.Split(path, "/")

	if len(splits) < 6 {
		return ns, pipelineID, "", fmt.Errorf("path error")
	}
	return ns, pipelineID, splits[5], err
}
