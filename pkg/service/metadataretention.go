package service

import (
	"context"

	miniox "github.com/instill-ai/x/minio"
)

type MetadataRetentionHandler interface {
	GetExpiryTagBySubscriptionPlan(ctx context.Context, requesterUID string) (string, error)
}

type metadataRetentionHandler struct{}

func NewRetentionHandler() MetadataRetentionHandler {
	return &metadataRetentionHandler{}
}

func (r metadataRetentionHandler) GetExpiryTagBySubscriptionPlan(ctx context.Context, requesterUID string) (string, error) {
	return defaultExpiryTag, nil
}

const (
	defaultExpiryTag = "default-expiry"
)

var MetadataExpiryRules = []miniox.ExpiryRule{
	{
		Tag:            defaultExpiryTag,
		ExpirationDays: 3,
	},
}
