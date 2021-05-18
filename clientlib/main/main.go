package main

import (
	// "log"
	"runtime"
	"runtime/debug"
	"time"

	TrojanGO "github.com/p4gefau1t/trojan-go/clientlib"
	"github.com/p4gefau1t/trojan-go/log"
	_ "github.com/p4gefau1t/trojan-go/log/golog"
)

func main() {

	go TrojanGO.StartProxy("127.0.0.1", 6666, "47.242.176.86", 443, "fobwifi")

	var first runtime.MemStats
	runtime.ReadMemStats(&first)
	// log.SetLogLevel(0)
	debug.SetGCPercent(1)

	for {
		time.Sleep(1 * time.Second)
		var current runtime.MemStats
		runtime.ReadMemStats(&current)
		firstO := current.HeapAlloc - first.HeapAlloc
		log.Info("firt0:%d", byteToMB(firstO))
		runtime.GC()
		debug.FreeOSMemory()

	}

}

func byteToMB(m uint64) float64 {
	return float64(m) / 1024 / 1024
}
