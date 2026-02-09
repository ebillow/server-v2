package model

import "strconv"

const RedisKeyRole = "role:"
const RedisKeyAccount = "acc:"
const RedisKeyAccBind = "acc_bind:"

// const RedisKeyIDs = "server:ids"

func KeyRole(roleID uint64) string {
	return RedisKeyRole + strconv.FormatUint(roleID, 10)
}

func KeyAccount(accID uint64) string {
	return RedisKeyAccount + strconv.FormatUint(accID, 10)
}

func KeyAccBind(acc string) string {
	return RedisKeyAccBind + acc
}
