package util

import (
	"github.com/google/uuid"
)

// UID  随机生成的uid
func NewUUID() string {
	//return uuid.NewV3(uuid.NewV1(), uuid.NewV4().String()).String()
	return uuid.New().String()
}
