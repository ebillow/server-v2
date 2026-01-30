package model

import "strconv"

const RedisKeyRole = "role:"
const RedisKeyAccount = "acc:"
const RedisKeySaveWait = "server:role_save_wait"
const RedisKeyServerState = "server:state"
const RedisKeyIDs = "server:ids"

func KeyRole(roleID uint64) string {
	return RedisKeyRole + strconv.FormatUint(roleID, 10)
}

func KeyAccount(acc string) string {
	return RedisKeyAccount + acc
}
