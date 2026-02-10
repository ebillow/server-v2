package ip2region

import (
	"errors"
	"github.com/ip2location/ip2location-go/v9"
	"server/pkg/logger"
)

var ip2locationDB *ip2location.DB

func InitIpDB() {
	var err error
	ip2locationDB, err = ip2location.OpenDB("./IP2LOCATION-LITE-DB1.BIN")
	if err != nil {
		zap.S().Errorf("ip2location open acc_db err:%v", err)
		return
	}
}

func GetCountry(ip string) (string, error) {
	if ip2locationDB == nil {
		return "", errors.New("acc_db not open")
	}
	results, err := ip2locationDB.Get_country_short(ip)
	if err != nil {
		return "", err
	}

	return results.Country_short, nil
}
