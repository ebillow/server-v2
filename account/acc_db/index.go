package acc_db

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
	"server/pkg/db"
)

const AccountTable = "accounts"

func CreateAccIndex() {
	idx := make(map[string]mongo.IndexModel)
	idx["acc_1"] = mongo.IndexModel{
		Keys:    bson.D{{"account", 1}},
		Options: options.Index().SetUnique(true),
	}

	idx["accid_1"] = mongo.IndexModel{
		Keys:    bson.D{{"accid", 1}},
		Options: options.Index().SetUnique(false),
	}
	err := db.CreateIndexIfNotExist(db.MongoDB, AccountTable, idx)
	if err != nil {
		zap.L().Error("create account index failed", zap.Error(err))
	}
}
