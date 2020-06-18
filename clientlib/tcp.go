package shadowsocks2

import "net"

type TCPConnecter struct {
	ServerAddr   string
	localTCPAddr *net.TCPAddr
}

func (tc *TCPConnecter) Connect() (net.Conn, error) {
	if tc.localTCPAddr == nil {
		return net.Dial("tcp", tc.ServerAddr)
	} else {
		serverTCPAddr, err := net.ResolveTCPAddr("tcp4", tc.ServerAddr)
		if err != nil {
			return nil, err
		}
		return net.DialTCP("tcp4", tc.localTCPAddr, serverTCPAddr)
	}
}

func (tc *TCPConnecter) ServerHost() string {
	return tc.ServerAddr
}
