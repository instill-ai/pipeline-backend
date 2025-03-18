package service

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/pipeline-backend/pkg/memory"

	miniox "github.com/instill-ai/x/minio"
)

// MetadataRetentionHandler allows clients to access the object expiration rule
// associated to a namespace. This is used to set the expiration of objects,
// e.g. when uploading the pipeline run data of a trigger. The preferred way to
// set the expiration of an object is by attaching a tag to the object. The
// MinIO client should set the tag-ased expiration rules for the bucket when it
// is initialized.
type MetadataRetentionHandler interface {
	ListExpiryRules() []miniox.ExpiryRule
	GetExpiryRuleByNamespace(_ context.Context, namespaceUID uuid.UUID) (miniox.ExpiryRule, error)
}

type metadataRetentionHandler struct{}

// NewRetentionHandler is the default implementation of
// MetadataRetentionHandler. It returns the same expiration rule for all
// namespaces.
func NewRetentionHandler() MetadataRetentionHandler {
	return &metadataRetentionHandler{}
}

var (
	// WorkflowMemoryExpiryRule defines the expiration time for workflow memory
	// blobs. This is the minimum expiration delay (1 day). Since processes
	// creating a workflow memory blob are responsible of purging it after
	// they're done with it, this is only a safeguard. By the time this
	// expiration rule is applied, the blob should not exist.
	WorkflowMemoryExpiryRule = miniox.ExpiryRule{
		Tag:            memory.WorkflowMemoryExpiryRuleTag,
		ExpirationDays: 1,
	}

	defaultExpiryRule = miniox.ExpiryRule{
		Tag:            "default-expiry",
		ExpirationDays: 3,
	}
)

func (h *metadataRetentionHandler) ListExpiryRules() []miniox.ExpiryRule {
	return []miniox.ExpiryRule{defaultExpiryRule, WorkflowMemoryExpiryRule}
}

func (h *metadataRetentionHandler) GetExpiryRuleByNamespace(_ context.Context, _ uuid.UUID) (miniox.ExpiryRule, error) {
	return defaultExpiryRule, nil
}
