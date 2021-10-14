package main

import (
	"flag"
	"log"
	"runtime"
	"time"

	"net/http"
	_ "net/http/pprof"

	ssStart "github.com/shadowsocks/go-shadowsocks2/clientlib/start"
)

var (
	addr    = flag.String("a", "0.0.0.0:10800", "address")
	verbose = flag.Bool("v", false, "verbose")
)

func init() {
	runtime.MemProfileRate = 1
}

func byteToMB(m uint64) float64 {
	return float64(m) / 1024 / 1024
}

var first uint64 = 0
var count uint64 = 0

func traceMemStats() {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	if first == 0 {
		first = ms.Alloc
	}
	value := ms.Alloc - first
	count += 1
	log.Printf("count(%d):当前占用内存:%f(mb) 已分配对象的字节数:%f(mb) HeapIdle:%f(mb) HeapReleased:%f(mb)", count, byteToMB(value), byteToMB(ms.Alloc), byteToMB(ms.HeapIdle), byteToMB(ms.HeapReleased))
}

// func main() {
// 	shadowsocks2.StartTCPUDP("45.153.219.200", 39227, "chacha20-ietf-poly1305", "RjOAwYH84B4uoH8q", 9999, false)

// 	for {
// 		time.Sleep(1 * time.Second)
// 	}
// }

func main() {
	traceMemStats()
	flag.Parse()

	var j = `
	{
		"Proto": 1,
		"Server": "cache-1558135236-proxy.tikvpn.in",
		"Url": "/proxy",
		"Username": "hEc88S9LHV1e0BUm",
		"Port": 80,
		"Method": "chacha20-ietf",
		"Password": "FW98t2ARSLb607e0",
		"Log": "",
		"Verbose": true,
		"MaxConnCount": 0,
		"tag": 0,
		"LocalPort": 7777,
		"Mpx": 0
	}
	`
	var j1 = `
	{
		"Proto": 1,
		"Server": "cache-755568305-proxy.tikvpn.in",
		"Url": "/proxy",
		"Username": "OTT1wQ6135zvBZ8z",
		"Port": 80,
		"Method": "chacha20-ietf",
		"Password": "m68jwjZHetuH1F6t",
		"Log": "",
		"Verbose": true,
		"MaxConnCount": 0,
		"tag": 0,
		"LocalPort": 7778,
		"Mpx": 0
	}
	`

	log.Printf("%v", "s")
	ssStart.Start(j)
	ssStart.Start(j1)

	// c.StartWebsocket("cache-1558135236-proxy.tikvpn.in", "/proxy", "1x5e14h0YxDARaX4", 80, "chacha20-ietf", "8CCrl6B6B76oHHCw", 7777, true)
	http.ListenAndServe("0.0.0.0:6060", nil)
	for {
		traceMemStats()
		runtime.GC()
		time.Sleep(1 * time.Second)
		// resp, err := http.Get("http://www.baidu.com")
		// if err != nil {
		// 	myLog.Println("请求百度错误：", err)
		// 	continue
		// }
		// if resp.StatusCode != 200 {
		// 	myLog.Println("百度出现故障，code:", resp.StatusCode)
		// 	continue
		// }
		// myLog.Println("百度运行正常")
	}

}
