package shadowsocks2

import (
	"net"

	"github.com/fregie/mpx"
)

type mpxConnecter struct {
	*mpx.ConnPool
	dialer mpx.Dialer
}

func NewMpxConnecter(dialer mpx.Dialer, connNum int) (*mpxConnecter, error) {
	mc := &mpxConnecter{
		ConnPool: mpx.NewConnPool(),
		dialer:   dialer,
	}
	err := mc.StartWithDialer(mc.dialer, connNum)
	return mc, err
}

func (m *mpxConnecter) Connect() (net.Conn, error) { return m.ConnPool.Connect(nil) }
func (m *mpxConnecter) ServerHost() string {
	if m.Addr() == nil {
		return ""
	}
	return m.Addr().String()
}
