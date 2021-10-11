package shadowsocks2

import (
	"log"
)

var (
	logger = log.New(logWriter, "[shadowsocks]", log.LstdFlags)
)

func loge(s string, v ...interface{}) {
	if logger != nil {
		var f = "[ERROR] :" + s
		logger.Printf(f, v...)
	}

}

func logf(f string, v ...interface{}) {
	if config.Verbose && logger != nil {
		logger.Printf(f, v...)
	}
}
