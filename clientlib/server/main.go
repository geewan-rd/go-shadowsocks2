package main

import (
	shadowsocks2 "github.com/shadowsocks/go-shadowsocks2/clientlib"
)

func main() {
	shadowsocks2.RunServerQuic("0.0.0.0:8488", "AEAD_CHACHA20_POLY1305", "your-password")
}
