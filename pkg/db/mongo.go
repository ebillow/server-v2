package db

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
	"time"
)

type MongoCfg struct {
	URI    string
	DbName string
}

var (
	MongoCli *mongo.Client
	MongoDB  *mongo.Database
)

func InitMongo(cfg MongoCfg, minPoolSize, maxPoolSize uint64) (err error) {
	MongoCli, err = NewMongo(cfg, minPoolSize, maxPoolSize)
	if err != nil {
		return err
	}
	MongoUse(cfg.DbName)

	return err
}

func NewMongo(cfg MongoCfg, minPoolSize, maxPoolSize uint64) (*mongo.Client, error) {
	cli, err := mongo.Connect(options.Client().ApplyURI(cfg.URI).
		SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1)).
		SetMaxPoolSize(maxPoolSize).
		SetMinPoolSize(minPoolSize).
		SetConnectTimeout(3 * time.Second).
		SetTimeout(10 * time.Second).
		SetMaxConnIdleTime(5 * time.Minute))

	zap.L().Info("connect to mongo", zap.String("uri", cfg.URI))
	return cli, err
}

func MongoUse(dbName string) *mongo.Database {
	MongoDB = MongoCli.Database(dbName)
	return MongoDB
}

func CloseMongo() error {
	if MongoCli != nil {
		return MongoCli.Disconnect(context.Background())
	}
	return nil
}

// -------------索引------------------
func getIndexNames(collection *mongo.Collection) (names map[string]bool, err error) {
	ctx := context.Background()
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err = cursor.All(context.Background(), &result); err != nil {
		return nil, err
	}

	names = make(map[string]bool)
	for i := 0; i != len(result); i++ {
		for k, v := range result[i] {
			if k == "name" {
				names[v.(string)] = true
			}
		}
	}
	return
}

// CreateIndexIfNotExist 创建索引，可以新增，无法修改已存在的索引。
func CreateIndexIfNotExist(db *mongo.Database, table string, createIDXs map[string]mongo.IndexModel) error {
	names, err := getIndexNames(db.Collection(table))
	if err != nil {
		return err
	}

	for name, v := range createIDXs {
		if names[name] == true {
			continue
		}

		_, err = db.Collection(table).Indexes().CreateOne(context.Background(), v)
		if err != nil {
			return err
		}
		zap.S().Infof("table %s create index %v", table, name)
	}

	return err
}
