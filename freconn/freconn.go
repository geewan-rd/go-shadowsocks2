package freconn

import (
	"net"

	"github.com/juju/ratelimit"
)

const (
	FlagRatelimit = 1 << iota
	FlagStat
)

type Conn struct {
	net.Conn
	Flag     int
	TxBucket *ratelimit.Bucket
	RxBucket *ratelimit.Bucket
	*Stat
}

func haveFlag(flag, have int) bool {
	if flag&have != 0 {
		return true
	}
	return false
}

func UpgradeConn(c net.Conn) *Conn {
	return &Conn{Conn: c, Flag: 0}
}

func (c *Conn) EnableStat(stat *Stat) {
	c.Stat = stat
	c.Flag = c.Flag | FlagStat
}

func (c *Conn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if err != nil {
		return n, err
	}
	if haveFlag(c.Flag, FlagRatelimit) && c.RxBucket != nil {
		c.RxBucket.Wait(int64(n * 8))
	}
	if haveFlag(c.Flag, FlagStat) && c.Stat != nil {
		c.Stat.AddRx(uint64(n) * 8)
	}
	return n, err
}

func (c *Conn) Write(b []byte) (int, error) {
	if haveFlag(c.Flag, FlagRatelimit) && c.TxBucket != nil {
		c.TxBucket.Wait(int64(len(b) * 8))
	}
	n, err := c.Conn.Write(b)
	if err != nil {
		return n, err
	}
	if haveFlag(c.Flag, FlagStat) && c.Stat != nil {
		c.Stat.AddTx(uint64(n) * 8)
	}
	return n, nil
}

func (c *Conn) Close() error {
	return c.Conn.Close()
}

type PacketConn struct {
	net.PacketConn
	Flag     int
	TxBucket *ratelimit.Bucket
	RxBucket *ratelimit.Bucket
	Stat     *Stat
}

func UpgradePacketConn(pc net.PacketConn) *PacketConn {
	return &PacketConn{PacketConn: pc, Flag: 0}
}

func (c *PacketConn) EnableStat(stat *Stat) {
	c.Stat = stat
	c.Flag = c.Flag | FlagStat
}

func (c *PacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, a, err := c.PacketConn.ReadFrom(b)
	if err != nil {
		return n, a, err
	}
	if haveFlag(c.Flag, FlagRatelimit) && c.RxBucket != nil {
		c.RxBucket.Wait(int64(n * 8))
	}
	if haveFlag(c.Flag, FlagStat) && c.Stat != nil {
		c.Stat.AddRx(uint64(n) * 8)
	}

	return n, a, err
}

func (c *PacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	if haveFlag(c.Flag, FlagRatelimit) && c.TxBucket != nil {
		c.TxBucket.Wait(int64(len(b) * 8))
	}
	n, err := c.PacketConn.WriteTo(b, addr)
	if err != nil {
		return n, err
	}
	if haveFlag(c.Flag, FlagStat) && c.Stat != nil {
		c.Stat.AddTx(uint64(n) * 8)
	}

	return n, nil
}
