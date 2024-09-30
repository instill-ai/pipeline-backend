package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type InsertInput struct {
	ID             string         `json:"id"`
	DatabaseName   string         `json:"database-name"`
	CollectionName string         `json:"collection-name"`
	Data           map[string]any `json:"data"`
}

type InsertOutput struct {
	Status string `json:"status"`
}

type InsertManyInput struct {
	ArrayID        []string         `json:"array-id"`
	DatabaseName   string           `json:"database-name"`
	CollectionName string           `json:"collection-name"`
	ArrayData      []map[string]any `json:"array-data"`
}

type InsertManyOutput struct {
	Status string `json:"status"`
}

type FindInput struct {
	ID             string         `json:"id"`
	DatabaseName   string         `json:"database-name"`
	CollectionName string         `json:"collection-name"`
	Filter         map[string]any `json:"filter"`
	Limit          int            `json:"limit"`
	Fields         []string       `json:"fields"`
}

type FindOutput struct {
	Status string     `json:"status"`
	Result FindResult `json:"result"`
}

type UpdateInput struct {
	ID             string         `json:"id"`
	DatabaseName   string         `json:"database-name"`
	CollectionName string         `json:"collection-name"`
	Filter         map[string]any `json:"filter"`
	UpdateData     map[string]any `json:"update-data"`
}

type UpdateOutput struct {
	Status string `json:"status"`
}

type DeleteInput struct {
	ID             string         `json:"id"`
	DatabaseName   string         `json:"database-name"`
	CollectionName string         `json:"collection-name"`
	Filter         map[string]any `json:"filter"`
}

type DeleteOutput struct {
	Status string `json:"status"`
}

type DropCollectionInput struct {
	DatabaseName   string `json:"database-name"`
	CollectionName string `json:"collection-name"`
}

type DropCollectionOutput struct {
	Status string `json:"status"`
}

type DropDatabaseInput struct {
	DatabaseName string `json:"database-name"`
}

type DropDatabaseOutput struct {
	Status string `json:"status"`
}

type CreateSearchIndexInput struct {
	DatabaseName   string         `json:"database-name"`
	CollectionName string         `json:"collection-name"`
	IndexName      string         `json:"index-name"`
	IndexType      string         `json:"index-type"`
	Syntax         map[string]any `json:"syntax"`
}

type CreateSearchIndexOutput struct {
	Status string `json:"status"`
}

type DropSearchIndexInput struct {
	DatabaseName   string `json:"database-name"`
	CollectionName string `json:"collection-name"`
	IndexName      string `json:"index-name"`
}

type DropSearchIndexOutput struct {
	Status string `json:"status"`
}

type VectorSearchInput struct {
	DatabaseName   string         `json:"database-name"`
	CollectionName string         `json:"collection-name"`
	Exact          bool           `json:"exact"`
	Filter         map[string]any `json:"filter"`
	IndexName      string         `json:"index-name"`
	Limit          int            `json:"limit"`
	NumCandidates  int            `json:"num-candidates"`
	Path           string         `json:"path"`
	QueryVector    []float64      `json:"query-vector"`
	Fields         []string       `json:"fields"`
}

type VectorResult struct {
	IDs       []string         `json:"ids"`
	Documents []map[string]any `json:"documents"`
	Vectors   [][]float64      `json:"vectors"`
	Metadata  []map[string]any `json:"metadata"`
}

type FindResult struct {
	IDs       []string         `json:"ids"`
	Documents []map[string]any `json:"documents"`
	Data      []map[string]any `json:"data"`
}

type VectorSearchOutput struct {
	Status string       `json:"status"`
	Result VectorResult `json:"result"`
}

func (e *execution) insert(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct InsertInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	if e.client.collectionClient == nil {
		collection := db.Collection(inputStruct.CollectionName)
		e.client.collectionClient = collection
		e.client.searchIndexClient = collection.SearchIndexes()
	}

	data := inputStruct.Data

	if inputStruct.ID != "" {
		id, err := primitive.ObjectIDFromHex(inputStruct.ID)
		if err != nil {
			data["_id"] = inputStruct.ID
		} else {
			data["_id"] = id
		}
	}

	_, err = e.client.collectionClient.InsertOne(ctx, data)
	if err != nil {
		return nil, err
	}

	outputStruct := InsertOutput{
		Status: "Successfully inserted 1 document",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) insertMany(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct InsertManyInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	if e.client.collectionClient == nil {
		collection := db.Collection(inputStruct.CollectionName)
		e.client.collectionClient = collection
		e.client.searchIndexClient = collection.SearchIndexes()
	}

	var anyArrayData []any
	idAllow := inputStruct.ArrayID != nil && len(inputStruct.ArrayID) == len(inputStruct.ArrayData)
	for i, data := range inputStruct.ArrayData {
		if idAllow {
			id, err := primitive.ObjectIDFromHex(inputStruct.ArrayID[i])
			if err != nil {
				data["_id"] = inputStruct.ArrayID[i]
			} else {
				data["_id"] = id
			}
		} else if inputStruct.ArrayID != nil && len(inputStruct.ArrayID) != len(inputStruct.ArrayData) {
			return nil, fmt.Errorf("arrayID and arrayData length mismatch")
		}
		anyArrayData = append(anyArrayData, data)
	}

	res, err := e.client.collectionClient.InsertMany(ctx, anyArrayData)
	if err != nil {
		return nil, err
	}

	outputStruct := InsertManyOutput{
		Status: fmt.Sprintf("Successfully inserted %v documents", len(res.InsertedIDs)),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// Limit is optional (default is 0)
func (e *execution) find(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct FindInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	if e.client.collectionClient == nil {
		collection := db.Collection(inputStruct.CollectionName)
		e.client.collectionClient = collection
		e.client.searchIndexClient = collection.SearchIndexes()
	}

	limit := inputStruct.Limit
	fields := inputStruct.Fields
	var filter map[string]any

	if inputStruct.ID != "" {
		id, err := primitive.ObjectIDFromHex(inputStruct.ID)
		if err != nil {
			filter = bson.M{"_id": inputStruct.ID}
		} else {
			filter = bson.M{"_id": id}
		}
	} else {
		filter = inputStruct.Filter
	}

	findOptions := options.Find()

	if limit > 0 {
		findOptions.SetLimit(int64(limit))
	}

	var cursor *mongo.Cursor
	if len(fields) > 0 {
		projection := bson.M{}
		projection["_id"] = 0
		for _, field := range fields {
			projection[field] = 1
		}
		findOptions.SetProjection(projection)
	}
	cursor, err = e.client.collectionClient.Find(ctx, filter, findOptions)

	if err != nil {
		return nil, err
	}

	var ids []string
	var documents []map[string]any
	var data []map[string]any
	for cursor.Next(ctx) {
		var document map[string]any
		err := cursor.Decode(&document)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)

		datum := make(map[string]any)
		for key, value := range document {
			if key != "_id" {
				datum[key] = value
			}
		}
		data = append(data, datum)

		id, ok := document["_id"].(primitive.ObjectID)
		if !ok {
			ids = append(ids, document["_id"].(string))
		} else {
			ids = append(ids, id.Hex())
		}
	}

	outputStruct := FindOutput{
		Status: fmt.Sprintf("Successfully found %v documents", len(documents)),
		Result: FindResult{
			IDs:       ids,
			Documents: documents,
			Data:      data,
		},
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) update(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct UpdateInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	if e.client.collectionClient == nil {
		collection := db.Collection(inputStruct.CollectionName)
		e.client.collectionClient = collection
		e.client.searchIndexClient = collection.SearchIndexes()
	}

	updateFields := inputStruct.UpdateData
	var filter map[string]any

	if inputStruct.ID == "" && inputStruct.Filter == nil {
		return nil, fmt.Errorf("either id or filter must be provided")
	}

	if inputStruct.ID != "" {
		id, err := primitive.ObjectIDFromHex(inputStruct.ID)
		if err != nil {
			filter = bson.M{"_id": inputStruct.ID}
		} else {
			filter = bson.M{"_id": id}
		}
	} else {
		filter = inputStruct.Filter
	}

	setFields := bson.M{}

	for key, value := range updateFields {
		setFields[key] = value
	}

	updateDoc := bson.M{}
	if len(setFields) > 0 {
		updateDoc["$set"] = setFields
	}

	if len(updateDoc) == 0 {
		return nil, fmt.Errorf("no valid update operations found")
	}

	res, err := e.client.collectionClient.UpdateMany(ctx, filter, updateDoc)
	if err != nil {
		return nil, err
	}

	outputStruct := UpdateOutput{
		Status: fmt.Sprintf("Successfully updated %v documents", res.ModifiedCount),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) delete(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DeleteInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	if e.client.collectionClient == nil {
		collection := db.Collection(inputStruct.CollectionName)
		e.client.collectionClient = collection
		e.client.searchIndexClient = collection.SearchIndexes()
	}

	var filter map[string]any

	if inputStruct.ID == "" && inputStruct.Filter == nil {
		return nil, fmt.Errorf("either id or filter must be provided")
	}

	if inputStruct.ID != "" {
		id, err := primitive.ObjectIDFromHex(inputStruct.ID)
		if err != nil {
			filter = bson.M{"_id": inputStruct.ID}
		} else {
			filter = bson.M{"_id": id}
		}
	} else {
		filter = inputStruct.Filter
	}

	res, err := e.client.collectionClient.DeleteMany(ctx, filter)
	if err != nil {
		return nil, err
	}

	outputStruct := DeleteOutput{
		Status: fmt.Sprintf("Successfully deleted %v documents", res.DeletedCount),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) dropCollection(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DropCollectionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	if e.client.collectionClient == nil {
		collection := db.Collection(inputStruct.CollectionName)
		e.client.collectionClient = collection
		e.client.searchIndexClient = collection.SearchIndexes()
	}

	err = e.client.collectionClient.Drop(ctx)
	if err != nil {
		return nil, err
	}

	outputStruct := DropCollectionOutput{
		Status: "Successfully dropped 1 collection",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) dropDatabase(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DropDatabaseInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	err = e.client.databaseClient.Drop(ctx)
	if err != nil {
		return nil, err
	}

	outputStruct := DropDatabaseOutput{
		Status: "Successfully dropped 1 database",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) createSearchIndex(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct CreateSearchIndexInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	if e.client.collectionClient == nil {
		collection := db.Collection(inputStruct.CollectionName)
		e.client.collectionClient = collection
		e.client.searchIndexClient = collection.SearchIndexes()
	}

	syntax := inputStruct.Syntax

	searchIndexModel := mongo.SearchIndexModel{
		Definition: syntax,
		Options: &options.SearchIndexesOptions{
			Name: &inputStruct.IndexName,
			Type: &inputStruct.IndexType,
		},
	}

	_, err = e.client.searchIndexClient.CreateOne(ctx, searchIndexModel)
	if err != nil {
		return nil, err
	}

	outputStruct := CreateSearchIndexOutput{
		Status: "Successfully created 1 search index",
	}

	// Convert the output structure to Structpb
	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) dropSearchIndex(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DropSearchIndexInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	if e.client.collectionClient == nil {
		collection := db.Collection(inputStruct.CollectionName)
		e.client.collectionClient = collection
		e.client.searchIndexClient = collection.SearchIndexes()
	}

	indexName := inputStruct.IndexName

	err = e.client.searchIndexClient.DropOne(ctx, indexName)
	if err != nil {
		return nil, err
	}

	outputStruct := DropSearchIndexOutput{
		Status: "Successfully dropped 1 search index",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// Exact is optional (default is false), false means ANN search, true means exact search
// numCandidates is optional (default is 3 * limit)
func (e *execution) vectorSearch(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct VectorSearchInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(ctx, e.Setup)

	var db *mongo.Database
	if e.client.databaseClient == nil {
		db = client.Database(inputStruct.DatabaseName)
		e.client.databaseClient = db
	}

	if e.client.collectionClient == nil {
		collection := db.Collection(inputStruct.CollectionName)
		e.client.collectionClient = collection
		e.client.searchIndexClient = collection.SearchIndexes()
	}

	exact := inputStruct.Exact
	filter := inputStruct.Filter
	indexName := inputStruct.IndexName
	limit := inputStruct.Limit
	numCandidates := inputStruct.NumCandidates
	path := inputStruct.Path
	queryVector := inputStruct.QueryVector
	fields := inputStruct.Fields

	vectorSearch := bson.M{
		"exact":       exact,
		"index":       indexName,
		"path":        path,
		"queryVector": queryVector,
		"limit":       limit,
	}
	if filter != nil {
		vectorSearch["filter"] = filter
	}

	if !exact {
		if numCandidates > 0 {
			vectorSearch["numCandidates"] = numCandidates
		} else {
			vectorSearch["numCandidates"] = 3 * limit
		}
	}

	project := bson.M{"_id": 0}
	for _, field := range fields {
		project[field] = 1
	}

	query := bson.A{
		bson.M{
			"$vectorSearch": vectorSearch,
		},
		bson.M{
			"$addFields": bson.M{
				"score": bson.M{
					"$meta": "vectorSearchScore",
				},
			},
		},
	}

	if len(fields) > 0 {
		query = append(query, bson.M{
			"$project": project,
		})
	}

	cursor, err := e.client.collectionClient.Aggregate(ctx, query)
	if err != nil {
		return nil, err
	}

	var ids []string
	var documents []map[string]any
	var vectors [][]float64
	var metadata []map[string]any
	for cursor.Next(ctx) {
		var document map[string]any
		err := cursor.Decode(&document)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
		vector, ok := document[path].(bson.A)
		if !ok {
			return nil, fmt.Errorf("unexpected type for vector")
		}
		var vectorData []float64
		for _, v := range vector {
			vectorData = append(vectorData, v.(float64))
		}
		vectors = append(vectors, vectorData)
		metadatum := make(map[string]any)
		for key, value := range document {
			if key != path && key != "score" && key != "_id" {
				metadatum[key] = value
			}
		}
		metadata = append(metadata, metadatum)

		id, ok := document["_id"].(primitive.ObjectID)
		if !ok {
			ids = append(ids, document["_id"].(string))
		} else {
			ids = append(ids, id.Hex())
		}
	}

	outputStruct := VectorSearchOutput{
		Status: fmt.Sprintf("Successfully found %v documents", len(documents)),
		Result: VectorResult{
			IDs:       ids,
			Documents: documents,
			Vectors:   vectors,
			Metadata:  metadata,
		},
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
