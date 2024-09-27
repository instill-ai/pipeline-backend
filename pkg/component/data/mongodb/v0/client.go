package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/structpb"
)

func newClient(ctx context.Context, setup *structpb.Struct) *mongo.Client {
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI(getURI(setup)))

	return client
}

func getURI(setup *structpb.Struct) string {
	return setup.GetFields()["uri"].GetStringValue()
}
