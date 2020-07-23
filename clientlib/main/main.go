package main

import (
	"log"
	"time"

	shadowsocks2 "github.com/shadowsocks/go-shadowsocks2/clientlib"
)

func main() {
	shadowsocks2.StartTCPUDP("tsx-test", 8488, "AEAD_CHACHA20_POLY1305", "your-password", 1080, false)
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		log.Printf("RX: %d kb", shadowsocks2.GetRx()/1024)
		log.Printf("TX: %d kb", shadowsocks2.GetTx()/1024)
	}
}
