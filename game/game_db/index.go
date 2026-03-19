package game_db

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
	"server/pkg/db"
)

const RoleTable = "roles"

func CreateIndex() {
	idx := make(map[string]mongo.IndexModel)
	idx["role_id_1"] = mongo.IndexModel{
		Keys:    bson.D{{"id", 1}},
		Options: options.Index().SetUnique(true),
	}

	err := db.CreateIndexIfNotExist(db.MongoDB(), RoleTable, idx)
	if err != nil {
		zap.L().Error("create index failed", zap.Error(err))
	}
}
