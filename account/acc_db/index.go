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
	idx["accid_1"] = mongo.IndexModel{
		Keys:    bson.D{{"accid", 1}},
		Options: options.Index().SetUnique(true),
	}

	idx["device_1"] = mongo.IndexModel{
		Keys:    bson.D{{"device", 1}},
		Options: options.Index().SetUnique(true),
	}

	idx["apple_1"] = mongo.IndexModel{
		Keys:    bson.D{{"appleid", 1}},
		Options: options.Index().SetUnique(false),
	}

	idx["google_1"] = mongo.IndexModel{
		Keys:    bson.D{{"googleid", 1}},
		Options: options.Index().SetUnique(false),
	}
	idx["fb_1"] = mongo.IndexModel{
		Keys:    bson.D{{"fbid", 1}},
		Options: options.Index().SetUnique(false),
	}

	err := db.CreateIndexIfNotExist(db.MongoDB, AccountTable, idx)
	if err != nil {
		zap.L().Error("create account index failed", zap.Error(err))
	}
}
