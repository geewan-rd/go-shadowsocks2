package shadowsocks2

import (
	"net"
	"net/http"
	"net/url"

	ssw "github.com/shadowsocks/go-shadowsocks2/websocket"

	"github.com/shadowsocks/go-shadowsocks2/freconn"

	"github.com/gorilla/websocket"
)

type WSConnecter struct {
	ServerAddr string
	URL        string
	Username   string
	Stat       *freconn.Stat
	dailer     *websocket.Dialer
}

func (ws *WSConnecter) Connect() (net.Conn, error) {
	if ws.dailer == nil {
		ws.dailer = websocket.DefaultDialer
	}
	u := url.URL{Scheme: "ws", Host: ws.ServerAddr, Path: ws.URL}
	logf("dial to %s\n", u.String())
	header := http.Header{
		"Shadowsocks-Username": []string{ws.Username},
		"Shadowsocks-Type":     []string{"connection"},
	}
	wc, _, err := ws.dailer.Dial(u.String(), header)
	if err != nil {
		logf("websocket dail failed: %s", err)
		return nil, err
	}
	newConn := freconn.UpgradeConn(wc.UnderlyingConn())
	newConn.EnableStat(ws.Stat)
	return newConn, nil
}

func (ws *WSConnecter) Dial() (net.Conn, error) {
	return ws.Connect()
}

func (ws *WSConnecter) DialPacketConn(localAddr net.Addr) (net.PacketConn, error) {
	return ssw.NewWSPacketConn(localAddr, ws.Username), nil
}

func (ws *WSConnecter) ServerHost() string {
	return ws.ServerAddr
}
