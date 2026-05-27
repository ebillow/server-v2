package acc_db

import (
	"server/internal/account/auth"
	"server/pkg/db"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

func CreateIndex() {
	idx := make(map[string]mongo.IndexModel)
	acc := &auth.Account{}
	idx[acc.FieldAccID()+"_1"] = mongo.IndexModel{
		Keys:    bson.D{{acc.FieldAccID(), 1}},
		Options: options.Index().SetUnique(true),
	}

	idx[acc.FieldBinds()+"_1"] = mongo.IndexModel{
		Keys:    bson.D{{acc.FieldBinds(), 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}

	err := db.CreateIndexIfNotExist(db.MongoDB(), auth.AccountCollection, idx)
	if err != nil {
		zap.L().Error("create account index failed", zap.Error(err))
	}
}
