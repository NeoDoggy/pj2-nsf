package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoDb struct {
	client     *mongo.Client
	collection *mongo.Collection
}

type mongoRecord struct {
	Key   string `bson:"_id"`
	Value string `bson:"value"`
}

func newMongoDb() (*mongoDb, error) {
	// shortcut: local MongoDB settings for the assignment, configure them if deploying elsewhere.
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://127.0.0.1:27017").SetTimeout(5 * time.Second))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		closeErr := client.Disconnect(ctx)
		return nil, errors.Join(fmt.Errorf("connect to MongoDB: %w", err), closeErr)
	}
	return &mongoDb{client: client, collection: client.Database("nsf").Collection("services")}, nil
}

func (m *mongoDb) Release() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}

func (m *mongoDb) Save(key, value string) error {
	_, err := m.collection.ReplaceOne(context.Background(), bson.M{"_id": key},
		mongoRecord{Key: key, Value: value}, options.Replace().SetUpsert(true))
	return err
}

func (m *mongoDb) SaveAll(data map[string]string) error {
	// shortcut: writes are per key, use a transaction if all-or-nothing batches become required.
	for key, value := range data {
		if err := m.Save(key, value); err != nil {
			return err
		}
	}
	return nil
}

func (m *mongoDb) Exist(key string) (bool, error) {
	err := m.collection.FindOne(context.Background(), bson.M{"_id": key}).Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return err == nil, err
}

func (m *mongoDb) Load(key string) (string, error) {
	var record mongoRecord
	err := m.collection.FindOne(context.Background(), bson.M{"_id": key}).Decode(&record)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", nil
	}
	return record.Value, err
}

func (m *mongoDb) LoadAll() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := m.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var records []mongoRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	data := make(map[string]string, len(records))
	for _, record := range records {
		data[record.Key] = record.Value
	}
	return data, nil
}

func (m *mongoDb) Remove(key string) error {
	_, err := m.collection.DeleteOne(context.Background(), bson.M{"_id": key})
	return err
}

func (m *mongoDb) RemoveAll() error {
	_, err := m.collection.DeleteMany(context.Background(), bson.M{})
	return err
}
