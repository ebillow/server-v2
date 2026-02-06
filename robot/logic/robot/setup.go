package robot

import (
	"encoding/json"
	"io/ioutil"
	"server/pkg/logger"
)

var Setup *ServerCfg

type ServerCfg struct {
	ServerAddr string
	Cnt        int
	BeginID    int
	WorldBegin uint32
	WorldEnd   uint32
	LoginOnly  bool
}

func ReadJson(cfg interface{}, fileName string) error {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		logger.Errorf("open setup error %v", err)
		return err
	}

	err = json.Unmarshal(b, cfg)
	if err != nil {
		logger.Errorf("unmarshal %s error:%v", fileName, err)
		return err
	}
	return nil
}
