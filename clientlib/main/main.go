package main

import (
	"flag"
	"net"
	"runtime"
	"strconv"

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
	h, p, _ := net.SplitHostPort(*addr)
	port, _ := strconv.Atoi(p)
	shadowsocks2.SetWSTimeout(5000)
	shadowsocks2.StartWebsocketMpx(h, "/proxy", "fregie", port, "CHACHA20", "789632145", 1080, 2, *verbose)
	http.ListenAndServe("0.0.0.0:8080", nil)
}
