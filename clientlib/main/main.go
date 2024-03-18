package main

import (
	"flag"
	"runtime"

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

func main() {
	flag.Parse()
	// h, p, _ := net.SplitHostPort(*addr)
	// port, _ := strconv.Atoi(p)
	shadowsocks2.StartTCPUDP("tsp-relay", 41133, "aes-256-cfb", "1234", 1080, true, 1)
	http.ListenAndServe("0.0.0.0:8080", nil)
}
