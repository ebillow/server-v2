package acc_db

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
	"server/pkg/db"
)

const AccountTable = "accounts"

func CreateIndex() {
	idx := make(map[string]mongo.IndexModel)
	idx["acc_id_1"] = mongo.IndexModel{
		Keys:    bson.D{{"acc_id", 1}},
		Options: options.Index().SetUnique(true),
	}

	idx["device_1"] = mongo.IndexModel{
		Keys:    bson.D{{"device", 1}},
		Options: options.Index().SetUnique(false).SetSparse(true),
	}

	idx["apple_id_1"] = mongo.IndexModel{
		Keys:    bson.D{{"apple_id", 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}

	idx["google_id_1"] = mongo.IndexModel{
		Keys:    bson.D{{"google_id", 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}
	idx["fb_id_1"] = mongo.IndexModel{
		Keys:    bson.D{{"fb_id", 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}

	err := db.CreateIndexIfNotExist(db.MongoDB(), AccountTable, idx)
	if err != nil {
		zap.L().Error("create account index failed", zap.Error(err))
	}
}
