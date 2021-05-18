package main

import (
	"time"
	// "github.com/p4gefau1t/trojan-go/log"
	jsonProxy "github.com/p4gefau1t/trojan-go/clientlib"
)

func main() {

	go jsonProxy.StartProxy("127.0.0.1", 6666, "47.242.176.86", 443, "fobwifi")

	for {
		time.Sleep(1 * time.Second)

	}

}
