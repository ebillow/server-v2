package acc_db

import (
	"server/account/logic/auth"
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

	idx[acc.FieldDevice()+"_1"] = mongo.IndexModel{
		Keys:    bson.D{{acc.FieldDevice(), 1}},
		Options: options.Index().SetUnique(false).SetSparse(true),
	}

	idx[acc.FieldAppleID()+"_1"] = mongo.IndexModel{
		Keys:    bson.D{{acc.FieldAppleID(), 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}

	idx[acc.FieldGoogleID()+"_1"] = mongo.IndexModel{
		Keys:    bson.D{{acc.FieldGoogleID(), 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}
	idx[acc.FieldFBID()+"_1"] = mongo.IndexModel{
		Keys:    bson.D{{acc.FieldFBID(), 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	}

	err := db.CreateIndexIfNotExist(db.MongoDB(), auth.AccountCollection, idx)
	if err != nil {
		zap.L().Error("create account index failed", zap.Error(err))
	}
}
