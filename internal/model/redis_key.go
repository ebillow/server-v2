package model

import "strconv"

const RedisKeyRole = "role:"
const RedisKeySaveWait = "server:role_save_wait"

func KeyRole(roleID uint64) string {
	return RedisKeyRole + strconv.FormatUint(roleID, 10)
}
