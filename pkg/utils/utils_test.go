package utils

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
)

func TestGetRequesterUIDAndUserUID(t *testing.T) {
	requesterUID := uuid.Must(uuid.NewV4()).String()
	userUID := uuid.Must(uuid.NewV4()).String()
	m := make(map[string]string)
	m[constant.HeaderRequesterUIDKey] = requesterUID
	m[constant.HeaderUserUIDKey] = userUID
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(m))

	c := qt.New(t)
	checkRequesterUID, checkUserUID := GetRequesterUIDAndUserUID(ctx)
	c.Check(checkRequesterUID, qt.Equals, requesterUID)
	c.Check(checkUserUID, qt.Equals, userUID)
}
