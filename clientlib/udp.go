package shadowsocks2

import "net"

type UDPConnecter struct{}

func (c *UDPConnecter) DialPacketConn(localAddr net.Addr) (net.PacketConn, error) {
	pc, err := net.ListenUDP("udp", localAddr.(*net.UDPAddr))
	if err != nil {
		return nil, err
	}
	return pc, err
}
