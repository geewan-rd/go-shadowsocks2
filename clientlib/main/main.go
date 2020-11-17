package main

import (
	"flag"
	"net"
	"strconv"

	shadowsocks2 "github.com/shadowsocks/go-shadowsocks2/clientlib"
)

var (
	addr = flag.String("addr", "0.0.0.0:10800", "address")
)

func main() {
	h, p, _ := net.SplitHostPort(*addr)
	port, _ := strconv.Atoi(p)
	shadowsocks2.StartWebsocketMpx(h, "/proxy", "fregie", port, "CHACHA20", "789632145", 1080, 5, true)
	ch := make(chan string)
	<-ch
}
