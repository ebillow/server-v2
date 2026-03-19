package cfg

import "time"

func SetDefaultValue(conf *Config) {
	if conf.Time.AutoSave == 0 {
		conf.Time.AutoSave = int64(time.Minute * 5)
	}
}
