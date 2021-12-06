package shadowsocks2

import (
	"fmt"
	"log"
	"strings"
)

var (
	logger  = log.New(logWriter, "[shadowsocks]", log.LstdFlags)
	Verbose = true
)

func loge(s string, v ...interface{}) {
	if logger != nil {
		var newS = fmt.Sprintf(s, v...)
		if strings.Contains(newS, "closed") {
			return
		} else if strings.Contains(newS, "no such host") {
			newS = "Network is unreachable"
		}
		var f = "[ERROR] :" + newS
		logger.Print(f)
	}

}

func logf(f string, v ...interface{}) {
	if Verbose && logger != nil {
		logger.Printf(f, v...)
	}
}
