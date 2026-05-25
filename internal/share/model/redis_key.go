package model

import (
	"fmt"
)

const RedisKeyRole = "role:"
const RedisKeyAccount = "acc:"
const RedisKeyAccBind = "acc_bind:"

// const RedisKeyIDs = "server:ids"

func KeyRole(roleID uint64) string {
	return fmt.Sprintf("%s{%d}", RedisKeyRole, roleID)
}

func KeyAccount(accID uint64) string {
	return fmt.Sprintf("%s{%d}", RedisKeyAccount, accID)
}

func KeyAccBind(acc string) string {
	return RedisKeyAccBind + acc
}
