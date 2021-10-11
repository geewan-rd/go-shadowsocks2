package main

import (
	"flag"
	"log"
	"runtime"
	"time"

	"net/http"
	_ "net/http/pprof"

	shadowsocks2 "github.com/shadowsocks/go-shadowsocks2/clientlib"
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
	// h, p, _ := net.SplitHostPort(*addr)
	// port, _ := strconv.Atoi(p)
	// shadowsocks2.SetWSTimeout(5000)
	// shadowsocks2.SetMaxConnCount(20)
	// var jsons = map[string]interface{}{}
	// jsons["server"] = "120.232.193.224"
	// jsons["url"] = "/proxy"
	// jsons["username"] = "042AlO6I35A59hz3"
	// jsons["serverPort"] = 2052
	// jsons["method"] = "aes-256-cfb"
	// jsons["password"] = "Ml7704X8J2d5Ct22"
	// jsons["localPort"] = 7777
	// jsons["verbose"] = *verbose
	// jsons["pprofPort"] = 6060
	// dataType, _ := json.Marshal(jsons)
	// isSuccess := shadowsocks2.StartWebsocketWithjson(dataType)
	log.Printf("%v", "s")
	shadowsocks2.StartTCPUDP("92.223.65.196", 38967, "aes-256-cfb", "qSh1sXqVsY6n3I7Y", 7777, false)
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
