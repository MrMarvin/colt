package colt

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

var defaultTimeout = 5 * time.Second

type Database struct {
	db       *mongo.Database
	client   *mongo.Client
	traceCtx *context.Context // optional context for tracing
}

func NewDatabase() *Database {
	return &Database{}
}

func (db *Database) connect(options *options.ClientOptions, dbName string) error {
	ctx := db.getContextOrDefault()

	client, err := mongo.Connect(ctx, options)
	if err != nil {
		log.Fatal(err)
	}
	db.client = client
	err = db.client.Ping(ctx, readpref.Primary())
	if err == nil {
		log.Print("Connected to MongoDB!")
	} else {
		log.Panic("Could not connect to MongoDB! Please check if mongo is running.", err)
		return err
	}
	db.db = db.client.Database(dbName)
	return nil
}

func (db *Database) Connect(connectionString string, dbName string) error {
	options := options.Client().ApplyURI(connectionString)
	options.Monitor = otelmongo.NewMonitor()
	err := db.connect(options, dbName)
	return err
}

func (db *Database) WithContext(ctx context.Context) Database {
	contextualizedDatabase := Database{
		db:       db.db,
		client:   db.client,
		traceCtx: &ctx,
	}
	return contextualizedDatabase
}

func (db *Database) getContextOrDefault() context.Context {
	if ctx := db.traceCtx; ctx == nil {
		return DefaultContext()
	}
	if _, hasDeadline := (*db.traceCtx).Deadline(); !hasDeadline {
		ctxWithTimeout, _ := context.WithTimeout(*db.traceCtx, defaultTimeout)
		return ctxWithTimeout
	}
	return *db.traceCtx
}

func (db *Database) Ping() error {
	return db.client.Ping(db.getContextOrDefault(), nil)
}

func (db *Database) Disconnect() error {
	err := db.client.Disconnect(db.getContextOrDefault())
	db.db = nil
	return err
}

func DefaultContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), defaultTimeout)
	return ctx
}

func GetCollection[T Document](db *Database, collectionName string) *Collection[T] {
	return &Collection[T]{db.db.Collection(collectionName), nil}
}
