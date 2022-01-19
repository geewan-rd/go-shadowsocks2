package shadowsocks2

import (
	"context"
	"errors"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Jeffail/tunny"
	"github.com/shadowsocks/go-shadowsocks2/socks"
)

type Client struct {
	TCPSocksListener net.Listener
	UDPSocksPC       net.PacketConn
	MaxConnCount     int
	udpTimeout       time.Duration
	udpBufSize       int
	connCount        int
	mutex            sync.Mutex
	ConnMap          sync.Map
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewClient(maxConnCount, UDPBufSize int, UDPTimeout time.Duration) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		MaxConnCount: maxConnCount,
		udpTimeout:   UDPTimeout,
		udpBufSize:   UDPBufSize,
		ctx:          ctx,
		cancel:       cancel,
	}
	return c
}

type Connecter interface {
	Connect() (net.Conn, error)
	ServerHost() string
}

type shadowUpgradeConn func(net.Conn) net.Conn
type shadowUpgradePacketConn func(net.PacketConn) net.PacketConn

// Create a SOCKS server listening on addr and proxy to server.

func (c *Client) StartsocksConnLocal(addr string, connecter Connecter, shadow shadowUpgradeConn) error {
	logf("SOCKS proxy %s <-> %s", addr, connecter.ServerHost())
	var err error
	c.TCPSocksListener, err = net.Listen("tcp", addr)
	if err != nil {
		loge("failed to listen on %s: %v", addr, err)
		return err
	}
	go func() {
		var connArray = make([]net.Conn, c.MaxConnCount)
		addConn := func(conn net.Conn, index int) {
			if c.MaxConnCount > 0 {
				c.mutex.Lock()
				con := connArray[index]
				if con != nil {
					con.Close()
				}
				connArray[index] = conn
				c.mutex.Unlock()
			}
		}
		var pool *tunny.Pool
		if c.MaxConnCount > 0 {
			pool = tunny.NewFunc(c.MaxConnCount+2, func(p interface{}) interface{} {
				f := p.(func())
				f()
				return true
			})
		}

		defer func() {
			c.mutex.Lock()
			connArray = make([]net.Conn, 0)
			debug.FreeOSMemory()
			c.mutex.Unlock()
		}()
		var connCount = 0
		for {
			lc, err := c.TCPSocksListener.Accept()
			if err != nil {

				loge("failed to accept: %s", err)
				if strings.Contains(err.Error(), "closed") {
					break
				}
				continue
			}
			var currnetIndex = 0
			if c.MaxConnCount > 0 {
				currnetIndex = connCount % c.MaxConnCount
			}
			addConn(lc, currnetIndex)
			connCount += 1
			lc.(*net.TCPConn).SetKeepAlive(false)
			if pool == nil {
				go c.handleConn(lc, connecter, shadow)
			} else {
				go pool.Process(func() {
					c.handleConn(lc, connecter, shadow)
				})
			}

		}
	}()

	return nil
}

func (c *Client) handleConn(lc net.Conn, connecter Connecter, shadow shadowUpgradeConn) {
	defer lc.Close()
	tgt, err := socks.Handshake(lc)
	if err != nil {
		// UDP: keep the connection until disconnect then free the UDP socket
		if err == socks.InfoUDPAssociate {
			buf := []byte{}
			// block here
			for {
				_, err := lc.Read(buf)
				if err, ok := err.(net.Error); ok && err.Timeout() {
					continue
				}
				logf("UDP Associate End.")
				return
			}
		}
		loge("failed to get target address: %v", err)
		return
	}

	rc, err := connecter.Connect()
	// log.Printf("web addr:%s", rc.LocalAddr())
	// log.Printf("accept addr:%s", lc.RemoteAddr())
	if err != nil {
		loge("Connect to %s failed: %s", connecter.ServerHost(), err)
		return
	}
	defer rc.Close()

	remoteConn := shadow(rc)
	if _, err = remoteConn.Write(tgt); err != nil {
		loge("failed to send target address: %v", err)
		return
	}

	logf("proxy %s <-> %s <-> %s", lc.RemoteAddr(), connecter.ServerHost(), tgt)
	_, _, err = relay(remoteConn, lc)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return // ignore i/o timeout
		}
		loge("relay error: %v", err)
	}
}

type PcConnecter interface {
	DialPacketConn(localAddr net.Addr) (net.PacketConn, error)
}

// Listen on laddr for Socks5 UDP packets, encrypt and send to server to reach target.
func (c *Client) udpSocksLocal(laddr string, server net.Addr, connecter PcConnecter, shadow shadowUpgradePacketConn) error {
	var err error
	c.UDPSocksPC, err = net.ListenPacket("udp", laddr)
	if err != nil {
		loge("UDP local listen error: %v", err)
		return err
	}
	go func() {
		defer c.UDPSocksPC.Close()

		nm := newNATmap(c.udpTimeout)
		buf := make([]byte, udpBufSize)

		for {
			select {
			case <-c.ctx.Done():
				logf("exit udp\n")
				return
			default:
				n, raddr, err := c.UDPSocksPC.ReadFrom(buf)
				if err != nil {
					loge("UDP local read error: %v", err)
					continue
				}
				pc := nm.Get(raddr.String())
				if pc == nil {
					pc, err = connecter.DialPacketConn(&net.UDPAddr{})
					if err != nil {
						loge("UDP local listen error: %v", err)
						continue
					}
					logf("UDP socks tunnel %s <-> %s <-> %s", laddr, server, socks.Addr(buf[3:]))
					pc = shadow(pc)
					nm.Add(raddr, c.UDPSocksPC, pc, socksClient)
				}

				_, err = pc.WriteTo(buf[3:n], server)
				// _, err = pc.WriteTo(payload, tgtUDPAddr)
				if err != nil {
					loge("UDP local write error: %v", err)
					continue
				}
			}
		}
	}()
	return nil
}

func (c *Client) Stop() error {
	logf("stopping tcp ss")
	c.cancel()
	if c.TCPSocksListener != nil {
		err := c.TCPSocksListener.Close()
		if err != nil {
			logf("close tcp listener failed: %s", err)
			return err
		}
	}
	if c.UDPSocksPC != nil {
		err := c.UDPSocksPC.Close()
		if err != nil {
			logf("stop ss err: %s", err)
			return errors.New("stop ss err: " + err.Error())
		}
	}
	return nil
}
