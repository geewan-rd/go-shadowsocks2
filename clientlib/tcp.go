package shadowsocks2

import (
	"net"

	"github.com/shadowsocks/go-shadowsocks2/freconn"
)

type TCPConnecter struct {
	ServerAddr   string
	Stat         *freconn.Stat
	localTCPAddr *net.TCPAddr
}

func (tc *TCPConnecter) Connect() (net.Conn, error) {
	var c net.Conn
	var err error
	if tc.localTCPAddr == nil {
		c, err = net.Dial("tcp", tc.ServerAddr)
	} else {
		serverTCPAddr, e := net.ResolveTCPAddr("tcp4", tc.ServerAddr)
		if e != nil {
			return nil, e
		}
		c, e = net.DialTCP("tcp4", tc.localTCPAddr, serverTCPAddr)
	}
	if err != nil {
		return c, err
	}
	newConn := freconn.UpgradeConn(c)
	newConn.EnableStat(tc.Stat)
	return newConn, nil
}

func (tc *TCPConnecter) ServerHost() string {
	return tc.ServerAddr
}
