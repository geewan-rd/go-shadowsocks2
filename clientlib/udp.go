package shadowsocks2

import "net"

type UDPConnecter struct{}

func (c *UDPConnecter) DialPacketConn(localAddr net.Addr) (net.PacketConn, error) {
	pc, err := net.Dial("UDP", localAddr.String())
	return pc.(*net.UDPConn), err
}
