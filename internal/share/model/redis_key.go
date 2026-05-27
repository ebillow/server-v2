package model

import (
	"fmt"
	"server/api/pb"
)

const (
	// 全局
	RedisKeyIDs = "server:acc_id"
	// 账号
	RedisKeyAccountPre = "acc"
	RedisKeyAccBindPre = "acc_bind"
	RedisKeyLoginCD    = "acc_login_cd"

	RedisKeyRolePre = "role"
)

func KeyAccount(accID uint64) string {
	return fmt.Sprintf("%s:{%d}", RedisKeyAccountPre, accID)
}

func KeyAccBind(bindKey string) string {
	return fmt.Sprintf("%s:%s", RedisKeyAccBindPre, bindKey)
}

func KeyAccLoginCD(typ pb.SdkType, acc string) string {
	return fmt.Sprintf("%s:%d:%s", RedisKeyLoginCD, typ, acc)
}

func KeyRole(roleID uint64) string {
	return fmt.Sprintf("%s:{%d}", RedisKeyRolePre, roleID)
}
