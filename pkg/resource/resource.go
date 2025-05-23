package resource

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetRscNameID returns the resource ID given a resource name
func GetRscNameID(path string) (string, error) {
	id := path[strings.LastIndex(path, "/")+1:]
	if id == "" {
		return "", fmt.Errorf("error when extract resource id from resource name '%s'", path)
	}
	return id, nil
}

// GetRscPermalinkUID returns the resource UID given a resource permalink
func GetRscPermalinkUID(path string) (uuid.UUID, error) {

	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return uuid.Nil, fmt.Errorf("error when extract resource id from resource permalink '%s'", path)
	}

	return uuid.FromStringOrNil(splits[1]), nil
}

type NamespaceType string

const (
	User         NamespaceType = "users"
	Organization NamespaceType = "organizations"
)

// TODO: We should neutralize the namespace type in the pipeline-backend, as it
// doesn't matter whether the namespace belongs to a user or organization. This
// refactor should be completed by August 2024.
type Namespace struct {
	NsType NamespaceType `json:"__NamespaceType"`
	NsID   string        `json:"__NamespaceID"`
	NsUID  uuid.UUID     `json:"__NamespaceUID"`
}

func (ns Namespace) Name() string {
	return fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)
}
func (ns Namespace) Permalink() string {
	return fmt.Sprintf("%s/%s", ns.NsType, ns.NsUID.String())
}

func GetOperationID(name string) (string, error) {
	id := strings.TrimPrefix(name, "operations/")
	if !strings.HasPrefix(name, "operations/") || id == "" {
		return "", status.Error(codes.InvalidArgument, "Error when extract operations resource id")
	}
	return id, nil
}
