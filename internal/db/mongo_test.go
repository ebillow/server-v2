package db

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"testing"
)

func TestMongoConnect(t *testing.T) {
	err := InitMongo(&MongoCfg{
		URI:    "mongodb://localhost:27017",
		DbName: "test",
	}, 16, 200)
	require.NoError(t, err)
	for i := 0; i < 10; i++ {
		_, err = MongoDB.Collection("test").InsertOne(context.Background(), bson.M{"name": fmt.Sprintf("test%d", i)})
		require.NoError(t, err)
	}
}
