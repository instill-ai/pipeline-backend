package resource

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
)

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
