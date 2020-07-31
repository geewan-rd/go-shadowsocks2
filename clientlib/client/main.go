package main

import (
	"log"

	shadowsocks2 "github.com/shadowsocks/go-shadowsocks2/clientlib"
)

func main() {
	log.Printf("start quic to 8848")
	shadowsocks2.StartQuic("47.52.197.88", 8488, "AEAD_CHACHA20_POLY1305", "your-password", 1080, true)
	ch := make(chan bool)
	<-ch
}
