package ssStart

import (
	"encoding/json"

	shadowsocks2 "github.com/shadowsocks/go-shadowsocks2/clientlib"
)

var j = `
{
    "Proto": 0,
    "Server": "",
    "Url": "",
    "Username": "",
    "Port": 0,
    "Method": "",
    "Password": "",
    "Log": "",
    "Verbose": true,
    "MaxConnCount": 0,
    "tag": 0,
    "LocalPort": 1,
    "Mpx": 0
}
`

type SSConfig struct {
	Proto        int    `json:"proto"`
	Server       string `json:"server"`
	Url          string `json:"url"`
	Username     string `json:"username"`
	Port         int    `json:"port"`
	Method       string `json:"method"`
	Password     string `json:"password"`
	Log          string `json:"log"`
	Verbose      bool   `json:"verbose"`
	MaxConnCount int    `json:"maxConnCount"`
	Tag          int    `json:"tag"`
	LocalHost    string `json:"localHost"`
	LocalPort    int    `json:"localPort"`
	Mpx          bool   `json:"mpx"`
	WSTimeout    int    `json:"wSTimeout"`
}

var clientMap = make(map[int]*shadowsocks2.SSClient, 0)
var clientProto = make(map[int]int, 0)

func Start(jsonS string) error {

	var cf SSConfig
	err := json.Unmarshal([]byte(jsonS), &cf)
	if err != nil {
		return err
	}
	var c = &shadowsocks2.SSClient{}
	c.SetMaxConnCount(cf.MaxConnCount)
	clientMap[cf.Tag] = c
	if cf.Log != "" {
		_ = shadowsocks2.SetlogOut(cf.Log)
	}
	if cf.LocalHost != "" {
		shadowsocks2.SetSSWLocalIP(cf.LocalHost)
		shadowsocks2.SetLocalIP(cf.LocalHost)
	}

	if cf.Proto == 0 {
		clientProto[cf.Tag] = 0
		return c.StartTCPUDP(cf.Server, cf.Port, cf.Method, cf.Password, cf.LocalPort, cf.Verbose)
	} else {
		if cf.WSTimeout > 0 {
			c.SetWSTimeout(cf.WSTimeout)
		}
		if !cf.Mpx {
			clientProto[cf.Tag] = 1
			return c.StartWebsocket(cf.Server, cf.Url, cf.Username, cf.Port, cf.Method, cf.Password, cf.LocalPort, cf.Verbose)
		} else {
			clientProto[cf.Tag] = 2
			return c.StartWebsocketMpx(cf.Server, cf.Url, cf.Username, cf.Port, cf.Method, cf.Password, cf.LocalPort, 0, cf.Verbose)
		}
	}
}
func Stop(tag int) {
	c := clientMap[tag]
	if c != nil {
		proto := clientProto[tag]
		if proto == 0 {
			c.StopTCPUDP()
		} else if proto == 1 {
			c.StopWebsocket()
		} else {
			c.StopWebsocketMpx()
		}
	}
}
