package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/nosql"
)

type Driver struct {
	client   *mongo.Client
	database string
}

func NewMongoDriver(client *mongo.Client, database string) *Driver {
	return &Driver{client: client, database: database}
}

func (d *Driver) db() *mongo.Database {
	return d.client.Database(d.database)
}

func (d *Driver) Ping(ctx context.Context) error {
	return d.client.Ping(ctx, nil)
}

func (d *Driver) Close() error {
	return d.client.Disconnect(context.Background())
}

func (d *Driver) GetCollections(ctx context.Context) ([]nosql.Collection, error) {
	names, err := d.db().ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("mongo: GetCollections: %w", err)
	}

	var collections []nosql.Collection
	for _, name := range names {
		coll := nosql.Collection{Name: name}

		var result bson.M
		err := d.db().RunCommand(ctx, bson.D{{Key: "collStats", Value: name}}).Decode(&result)
		if err == nil {
			coll.DocumentCount = toInt64(result["count"])
			avgSize := toFloat64(result["avgObjSize"])
			totalSize := toFloat64(result["size"])
			coll.AvgDocSize = formatBytes(int64(avgSize))
			coll.TotalSize = formatBytes(int64(totalSize))

			if rawIndexes, ok := result["indexDetails"]; ok {
				if indexMap, ok := rawIndexes.(bson.M); ok {
					for idxName := range indexMap {
						coll.Indexes = append(coll.Indexes, nosql.Index{
							Name: idxName,
							Keys: map[string]int{},
						})
					}
				}
			}
		}

		idxView := d.db().Collection(name).Indexes()
		cursor, err := idxView.List(ctx)
		if err == nil {
			defer cursor.Close(ctx)
			var indexDocs []bson.M
			if err := cursor.All(ctx, &indexDocs); err == nil {
				coll.Indexes = nil
				for _, doc := range indexDocs {
					idx := nosql.Index{
						Keys: map[string]int{},
					}
					if n, ok := doc["name"].(string); ok {
						idx.Name = n
					}
					if unique, ok := doc["unique"].(bool); ok {
						idx.Unique = unique
					}
					if keyDoc, ok := doc["key"].(bson.M); ok {
						for k, v := range keyDoc {
							idx.Keys[k] = int(toFloat64(v))
						}
					}
					coll.Indexes = append(coll.Indexes, idx)
				}
			}
		}

		collections = append(collections, coll)
	}
	return collections, nil
}

func (d *Driver) GetDocuments(ctx context.Context, collection string, filter string, limit int) ([]nosql.Document, error) {
	filterDoc := bson.D{}
	if filter != "" {
		if err := bson.UnmarshalExtJSON([]byte(filter), true, &filterDoc); err != nil {
			return nil, fmt.Errorf("mongo: GetDocuments: parse filter: %w", err)
		}
	}

	opts := options.Find().SetLimit(int64(limit))
	cursor, err := d.db().Collection(collection).Find(ctx, filterDoc, opts)
	if err != nil {
		return nil, fmt.Errorf("mongo: GetDocuments: %w", err)
	}
	defer cursor.Close(ctx)

	var rawDocs []bson.M
	if err := cursor.All(ctx, &rawDocs); err != nil {
		return nil, fmt.Errorf("mongo: GetDocuments: decode: %w", err)
	}

	docs := make([]nosql.Document, 0, len(rawDocs))
	for _, raw := range rawDocs {
		docs = append(docs, convertBSONDoc(raw))
	}
	return docs, nil
}

func (d *Driver) InsertDocument(ctx context.Context, collection string, doc nosql.Document) (nosql.Document, error) {
	res, err := d.db().Collection(collection).InsertOne(ctx, doc)
	if err != nil {
		return nil, fmt.Errorf("mongo: InsertDocument: %w", err)
	}

	doc["_id"] = convertBSONValue(res.InsertedID)
	return doc, nil
}

func (d *Driver) UpdateDocument(ctx context.Context, collection string, id string, doc nosql.Document) (nosql.Document, error) {
	objID, err := bson.ObjectIDFromHex(id)
	var filter bson.D
	if err != nil {
		filter = bson.D{{Key: "_id", Value: id}}
	} else {
		filter = bson.D{{Key: "_id", Value: objID}}
	}

	delete(doc, "_id")
	_, err = d.db().Collection(collection).ReplaceOne(ctx, filter, doc)
	if err != nil {
		return nil, fmt.Errorf("mongo: UpdateDocument: %w", err)
	}

	doc["_id"] = id
	return doc, nil
}

func (d *Driver) CreateCollection(ctx context.Context, name string) error {
	if err := d.db().CreateCollection(ctx, name); err != nil {
		return fmt.Errorf("mongo: CreateCollection: %w", err)
	}
	return nil
}

func (d *Driver) DropCollection(ctx context.Context, name string) error {
	if err := d.db().Collection(name).Drop(ctx); err != nil {
		return fmt.Errorf("mongo: DropCollection: %w", err)
	}
	return nil
}

func (d *Driver) DeleteDocument(ctx context.Context, collection string, id string) error {
	objID, err := bson.ObjectIDFromHex(id)
	var filter bson.D
	if err != nil {
		filter = bson.D{{Key: "_id", Value: id}}
	} else {
		filter = bson.D{{Key: "_id", Value: objID}}
	}

	_, err = d.db().Collection(collection).DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("mongo: DeleteDocument: %w", err)
	}
	return nil
}

func convertBSONDoc(doc bson.M) nosql.Document {
	result := make(nosql.Document, len(doc))
	for k, v := range doc {
		result[k] = convertBSONValue(v)
	}
	return result
}

func convertBSONValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case bson.ObjectID:
		return val.Hex()
	case bson.M:
		return convertBSONDoc(val)
	case bson.A:
		arr := make([]any, len(val))
		for i, elem := range val {
			arr[i] = convertBSONValue(elem)
		}
		return arr
	case bson.D:
		m := make(bson.M, len(val))
		for _, e := range val {
			m[e.Key] = e.Value
		}
		return convertBSONDoc(m)
	default:
		return val
	}
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int32:
		return int64(val)
	case int64:
		return val
	case float64:
		return int64(val)
	}
	return 0
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	}
	return 0
}

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.2f GB", float64(b)/gb)
	case b >= mb:
		return fmt.Sprintf("%.2f MB", float64(b)/mb)
	case b >= kb:
		return fmt.Sprintf("%.2f KB", float64(b)/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
