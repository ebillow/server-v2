package db

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.uber.org/zap"
	"time"
)

type Mongo struct {
	Client *mongo.Client
	DB     *mongo.Database
}

var (
	mongoCli *Mongo
)

func MongoDB() *mongo.Database {
	return mongoCli.DB
}

func MongoClient() *Mongo {
	return mongoCli
}

func InitMongo(uri string, dbName string, minPoolSize, maxPoolSize uint64) (err error) {
	mongoCli, err = NewMongo(uri, dbName, minPoolSize, maxPoolSize)
	if err != nil {
		return err
	}

	return err
}

func NewMongo(uri string, dbName string, minPoolSize, maxPoolSize uint64) (*Mongo, error) {
	cli, err := mongo.Connect(options.Client().ApplyURI(uri).
		SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)).
		SetMaxPoolSize(maxPoolSize).
		SetMinPoolSize(minPoolSize).
		SetConnectTimeout(3 * time.Second).
		// SetTimeout(10 * time.Second).  //由操作控制
		SetMaxConnIdleTime(5 * time.Minute))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = cli.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	zap.L().Info("connect to mongo", zap.String("uri", uri))
	return &Mongo{
		Client: cli,
		DB:     cli.Database(dbName),
	}, err
}

// MongoUse 只能在初始化时调用
func MongoUse(dbName string) {
	mongoCli.DB = mongoCli.Client.Database(dbName)
}

func CloseMongo() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if mongoCli != nil {
		return mongoCli.Client.Disconnect(ctx)
	}
	return nil
}

// -------------索引------------------
func getIndexNames(ctx context.Context, collection *mongo.Collection) (map[string]bool, error) {
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []struct {
		Name string `bson:"name"`
	}
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	names := make(map[string]bool)
	for _, item := range result {
		names[item.Name] = true
	}
	return names, nil
}

// CreateIndexIfNotExist 创建索引，可以新增，无法修改已存在的索引。大表要考虑手动创建
func CreateIndexIfNotExist(db *mongo.Database, table string, createIDXs map[string]mongo.IndexModel) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	names, err := getIndexNames(ctx, db.Collection(table))
	if err != nil {
		return err
	}

	for name, v := range createIDXs {
		if names[name] == true {
			continue
		}

		_, err = db.Collection(table).Indexes().CreateOne(ctx, v)
		if err != nil {
			return err
		}
		zap.S().Infof("table %s create index %v", table, name)
	}

	return err
}
