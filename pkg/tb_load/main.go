package main

import (
	"server/pkg/logger"
	"server/pkg/tb_load/make_cfg"
)

//go:generate go run $GOFILE

func main() {
	logger.NewZapLog("./tool.log", logger.Config{
		Level:   -1,
		Console: true,
	})
	make_cfg.MakeAll("./data")
}
