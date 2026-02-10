package robot

import (
	"encoding/json"
	"go.uber.org/zap"
	"io/ioutil"
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
		zap.S().Errorf("open setup error %v", err)
		return err
	}

	err = json.Unmarshal(b, cfg)
	if err != nil {
		zap.S().Errorf("unmarshal %s error:%v", fileName, err)
		return err
	}
	return nil
}
