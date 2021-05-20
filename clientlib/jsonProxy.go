package TrojanGO

import (
	"encoding/json"
	"fmt"

	"github.com/p4gefau1t/trojan-go/log"
	_ "github.com/p4gefau1t/trojan-go/log/golog"
	"github.com/p4gefau1t/trojan-go/proxy"
	_ "github.com/p4gefau1t/trojan-go/proxy/client"
)

func GoStartProxy(localAddr string, localPort int, remoteAddr string, remotePort int, password string) {
	go StartProxy(localAddr, localPort, remoteAddr, remotePort, password)

}
func StartProxy(localAddr string, localPort int, remoteAddr string, remotePort int, password string) error {
	jsonMap := map[string]interface{}{}
	jsonMap["run_type"] = "client"
	jsonMap["local_addr"] = localAddr
	jsonMap["local_port"] = localPort
	jsonMap["remote_addr"] = remoteAddr
	jsonMap["remote_port"] = remotePort
	jsonMap["password"] = []string{password}
	jsonMap["ssl"] = map[string]interface{}{"verify": false, "sni": ""}

	data, e := json.Marshal(jsonMap)
	if e != nil {
		log.Fatalf("Failed to read from stdin: %s", e.Error())
		return e
	}
	return startWithData(data)
}
func startWithData(data []byte) error {
	proxy, err := proxy.NewProxyFromConfigData(data, true)
	if err != nil {
		fmt.Print("error:%@", err.Error())
		log.Fatal(err)
		return err
	}
	currentProxy = proxy
	err = proxy.Run()
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

var currentProxy *proxy.Proxy

func GOStartWithData(data []byte) {
	go startWithData(data)
}
func StopProxy() {
	if currentProxy != nil {
		currentProxy
	}
}
