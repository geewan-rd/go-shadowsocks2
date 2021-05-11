package main

import (
	"flag"
	"log"
	"runtime"
	"time"

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

func main() {
	traceMemStats()
	flag.Parse()
	// h, p, _ := net.SplitHostPort(*addr)
	// port, _ := strconv.Atoi(p)
	shadowsocks2.SetWSTimeout(5000)
	isSuccess := shadowsocks2.StartWebsocket("line-gzm02.transocks.com.cn", "/proxy", "ROl414ZzDG16UbSe", 2052, "chacha20-ietf-poly1305", "og86nkanSVbVz6GZ", 6666, *verbose)
	log.Printf("%v", isSuccess)
	// http.ListenAndServe("0.0.0.0:8080", nil)
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
