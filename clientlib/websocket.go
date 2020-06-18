package shadowsocks2

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

type WSConnecter struct {
	ServerAddr string
	URL        string
	Username   string
	dailer     *websocket.Dialer
}

func (ws *WSConnecter) Connect() (net.Conn, error) {
	if ws.dailer == nil {
		ws.dailer = websocket.DefaultDialer
	}
	u := url.URL{Scheme: "ws", Host: ws.ServerAddr, Path: ws.URL}
	fmt.Printf("dial to %s\n", u.String())
	header := http.Header{
		"Shadowsocks-Username": []string{ws.Username},
	}
	wc, _, err := ws.dailer.Dial(u.String(), header)
	if err != nil {
		logf("websocket dail failed: %s", err)
		return nil, err
	}
	return wc.UnderlyingConn(), nil
}

func (ws *WSConnecter) ServerHost() string {
	return ws.ServerAddr
}