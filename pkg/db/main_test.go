package db

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := InitRedis(RedisCfg{
		Addr:     []string{"127.0.0.1:6380", "127.0.0.1:6381", "127.0.0.1:6382"},
		Password: "",
		DB:       0,
		Name:     "",
	}, 0); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
